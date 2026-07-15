// Package istanbul converts Istanbul `coverage-final.json` reports into the
// internal coverage model. This is the JSON shape emitted by nyc/istanbul and,
// in Istanbul-compatible form, by Vitest's V8 coverage provider.
//
// Istanbul tracks coverage at statement, branch, and function granularity keyed
// by opaque ids rather than by line. Line coverage is therefore derived: a line
// is considered covered when any statement whose start position falls on that
// line executed at least once. Statement, branch, and function counts are
// preserved as native metrics.
package istanbul

import (
	"encoding/json"
	"fmt"
	"io"
	"path"
	"sort"
	"strings"

	"github.com/asynkron/testtranslator/internal/coverage"
)

func init() {
	coverage.Register(
		"istanbul-json",
		"Istanbul/nyc coverage-final.json (also emitted by Vitest's V8 provider)",
		Adapter{},
		"istanbul", "nyc", "vitest-json",
	)
}

// defaultMaxInput bounds the amount of JSON read when opts.MaxInputBytes is not
// set, guarding against unbounded or hostile input.
const defaultMaxInput = 256 << 20

// Adapter implements coverage.Adapter for Istanbul coverage-final.json files.
type Adapter struct{}

// position is a source position within a file coverage entry. Only the line is
// consumed; the column is retained for completeness but ignored.
type position struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

// rangeLoc is a start/end source range as used by statement and branch maps.
type rangeLoc struct {
	Start position `json:"start"`
	End   position `json:"end"`
}

// branchMeta describes a single branch point: its type and the source
// locations of each of its arms.
type branchMeta struct {
	Type      string     `json:"type"`
	Loc       rangeLoc   `json:"loc"`
	Locations []rangeLoc `json:"locations"`
}

// fnMeta describes a single function definition.
type fnMeta struct {
	Name string   `json:"name"`
	Decl rangeLoc `json:"decl"`
	Loc  rangeLoc `json:"loc"`
}

// fileCoverage is one file's entry in a coverage-final.json object.
type fileCoverage struct {
	Path         string                `json:"path"`
	StatementMap map[string]rangeLoc   `json:"statementMap"`
	S            map[string]int64      `json:"s"`
	BranchMap    map[string]branchMeta `json:"branchMap"`
	B            map[string][]int64    `json:"b"`
	FnMap        map[string]fnMeta     `json:"fnMap"`
	F            map[string]int64      `json:"f"`
}

// Parse reads an Istanbul coverage-final.json document and produces a
// normalized coverage report. It fails clearly when the input is not a JSON
// object of file-coverage entries.
func (Adapter) Parse(r io.Reader, opts coverage.Options) (*coverage.Report, error) {
	limit := opts.MaxInputBytes
	if limit <= 0 {
		limit = defaultMaxInput
	}
	lr := &io.LimitedReader{R: r, N: limit + 1}
	raw, err := io.ReadAll(lr)
	if err != nil {
		return nil, fmt.Errorf("istanbul: read error: %w", err)
	}
	if lr.N <= 0 {
		return nil, fmt.Errorf("istanbul: input exceeds %d byte limit", limit)
	}

	// Reject anything that is not a JSON object at the root. A top-level array
	// (or scalar) is a common mistake and must not be silently accepted.
	trimmed := strings.TrimSpace(string(raw))
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("istanbul: empty input")
	}
	if trimmed[0] != '{' {
		return nil, fmt.Errorf("istanbul: expected a JSON object of file-coverage entries, got root %q", string(trimmed[0]))
	}

	var doc map[string]fileCoverage
	dec := json.NewDecoder(strings.NewReader(trimmed))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&doc); err != nil {
		// Retry without DisallowUnknownFields so that forward-compatible extra
		// keys (e.g. inputSourceMap, _coverageSchema) do not fail the parse,
		// while still surfacing genuinely malformed documents.
		var relaxed map[string]fileCoverage
		if err2 := json.Unmarshal([]byte(trimmed), &relaxed); err2 != nil {
			return nil, fmt.Errorf("istanbul: malformed coverage-final.json: %w", err2)
		}
		doc = relaxed
	}
	if len(doc) == 0 {
		return nil, fmt.Errorf("istanbul: no file-coverage entries found")
	}

	// Every entry must carry a statementMap; its absence means the input is not
	// an Istanbul coverage document.
	for key, fc := range doc {
		if fc.StatementMap == nil {
			return nil, fmt.Errorf("istanbul: entry %q is missing statementMap; input is not an Istanbul coverage-final.json", key)
		}
	}

	report := &coverage.Report{
		Producer:          "istanbul",
		InterchangeFormat: "istanbul-json",
	}

	// Iterate file keys in deterministic order; Normalize re-sorts the report,
	// but stable iteration keeps diagnostics deterministic too.
	keys := make([]string, 0, len(doc))
	for k := range doc {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		fc := doc[key]
		file, err := convertFile(key, fc, opts)
		if err != nil {
			return nil, err
		}
		report.Files = append(report.Files, file)
	}

	opts.Diag.Notef("istanbul.shape", opts.SourceName,
		"parsed as Istanbul coverage-final.json; Vitest's V8 provider can also emit this shape")

	if err := report.Normalize(); err != nil {
		return nil, err
	}
	return report, nil
}

// convertFile turns one file-coverage entry into a coverage.File.
func convertFile(key string, fc fileCoverage, opts coverage.Options) (coverage.File, error) {
	src := fc.Path
	if src == "" {
		src = key
	}
	rel, err := resolvePath(opts, src)
	if err != nil {
		return coverage.File{}, fmt.Errorf("istanbul: %w", err)
	}
	file := coverage.File{Path: rel}

	// lineHits accumulates the MAX statement hit count per source line, since a
	// line is covered when any statement on it ran. AddLine sums duplicates, so
	// we aggregate locally and add each line exactly once.
	lineHits := map[int]int64{}
	stmtCovered := int64(0)
	for id, loc := range fc.StatementMap {
		ln := loc.Start.Line
		if ln <= 0 {
			continue
		}
		hits := fc.S[id]
		if hits > 0 {
			stmtCovered++
		}
		if cur, ok := lineHits[ln]; !ok || hits > cur {
			lineHits[ln] = hits
		}
	}

	// branchAgg accumulates branch coverage per line so it can be folded into
	// the same LineHit as statement data.
	type branchAgg struct {
		covered int
		total   int
	}
	branchByLine := map[int]*branchAgg{}
	branchCovered := 0
	branchTotal := 0
	for id, meta := range fc.BranchMap {
		hits := fc.B[id]
		total := len(meta.Locations)
		if total == 0 {
			total = len(hits)
		}
		if total == 0 {
			continue
		}
		covered := 0
		for _, h := range hits {
			if h > 0 {
				covered++
			}
		}
		branchTotal += total
		branchCovered += covered

		ln := meta.Loc.Start.Line
		if ln <= 0 {
			// Fall back to the first arm's start line when the branch has no
			// own location.
			if len(meta.Locations) > 0 {
				ln = meta.Locations[0].Start.Line
			}
		}
		if ln <= 0 {
			opts.Diag.Warnf("istanbul.branch.noline", opts.SourceName,
				"file %s: branch %q has no source line; branch counts kept in metrics only", rel, id)
			continue
		}
		agg, ok := branchByLine[ln]
		if !ok {
			agg = &branchAgg{}
			branchByLine[ln] = agg
		}
		agg.covered += covered
		agg.total += total
	}

	// Emit line hits, folding branch data into the matching line where present.
	lineNumbers := make([]int, 0, len(lineHits))
	for ln := range lineHits {
		lineNumbers = append(lineNumbers, ln)
	}
	// Include branch-only lines that have no statement on them.
	for ln := range branchByLine {
		if _, ok := lineHits[ln]; !ok {
			lineNumbers = append(lineNumbers, ln)
		}
	}
	sort.Ints(lineNumbers)

	linesCovered := int64(0)
	for _, ln := range lineNumbers {
		hit := coverage.LineHit{Number: ln, Hits: lineHits[ln]}
		if agg, ok := branchByLine[ln]; ok {
			hit.Branch = true
			hit.BranchesCovered = agg.covered
			hit.BranchesTotal = agg.total
		}
		if hit.Hits > 0 {
			linesCovered++
		}
		if err := file.AddLine(hit); err != nil {
			return coverage.File{}, fmt.Errorf("istanbul: file %s: %w", rel, err)
		}
	}

	// Statement metric is native to Istanbul.
	file.SetMetric(coverage.MetricStatements, coverage.Count{
		Covered: stmtCovered,
		Total:   int64(len(fc.StatementMap)),
	})

	// Branch metric only when branch data is present.
	if len(fc.BranchMap) > 0 && branchTotal > 0 {
		file.SetMetric(coverage.MetricBranches, coverage.Count{
			Covered: int64(branchCovered),
			Total:   int64(branchTotal),
		})
	}

	// Function metric and function entries when a function map is present.
	if len(fc.FnMap) > 0 {
		fnCovered := int64(0)
		fnIDs := make([]string, 0, len(fc.FnMap))
		for id := range fc.FnMap {
			fnIDs = append(fnIDs, id)
		}
		sort.Strings(fnIDs)
		for _, id := range fnIDs {
			meta := fc.FnMap[id]
			hits := fc.F[id]
			if hits > 0 {
				fnCovered++
			}
			ln := meta.Decl.Start.Line
			if ln <= 0 {
				ln = meta.Loc.Start.Line
			}
			if ln < 0 {
				ln = 0
			}
			file.Functions = append(file.Functions, coverage.Function{
				Name: meta.Name,
				Line: ln,
				Hits: hits,
			})
		}
		file.SetMetric(coverage.MetricFunctions, coverage.Count{
			Covered: fnCovered,
			Total:   int64(len(fc.FnMap)),
		})
	}

	// Line coverage is derived from statement start lines.
	file.SetMetric(coverage.MetricLines, coverage.Count{
		Covered:        linesCovered,
		Total:          int64(len(lineNumbers)),
		Derived:        true,
		DerivationRule: "a line is covered when any statement starting on it executed (max over per-statement hit counts)",
	})

	return file, nil
}

// resolvePath normalizes an Istanbul file path into repo-relative POSIX form.
// When a normalizer is configured it reroots (and rejects escaping) paths;
// otherwise the path is cleaned to POSIX form as-is.
func resolvePath(opts coverage.Options, src string) (string, error) {
	if opts.Paths != nil {
		rel, err := opts.Paths.Rel(src)
		if err != nil {
			return "", fmt.Errorf("cannot reroot %q: %w", src, err)
		}
		return rel, nil
	}
	cleaned := path.Clean(strings.ReplaceAll(src, "\\", "/"))
	cleaned = strings.TrimPrefix(cleaned, "/")
	if cleaned == "" || cleaned == "." {
		return "", fmt.Errorf("empty file path in coverage entry")
	}
	return cleaned, nil
}
