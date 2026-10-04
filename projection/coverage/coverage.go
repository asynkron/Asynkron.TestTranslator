// Package coverage projects Go coverprofiles, Istanbul coverage-final.json and
// Cobertura reports into repository-relative, filepath-keyed coverage.v1 records.
// Format parsing and metric counting use TestTranslator's canonical adapters.
// The projection reports unsafe or unrooted paths as unavailable and preserves
// statements separately from line, branch and function metrics.
//
// Consumers supply repository context and own persistence, transport and UI.
// Parsing is independent of those integrations; source-map resolution remains
// the caller's responsibility.
package coverage

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io"
	"math"
	"path/filepath"
	"sort"
	"strings"

	testtranslator "github.com/asynkron/Asynkron.TestTranslator"
	"github.com/asynkron/Asynkron.TestTranslator/internal/pathutil"
	"golang.org/x/tools/cover"
)

// SchemaVersion identifies the shared coverage record schema.
const SchemaVersion = "coverage.v1"

// ErrUnsafePath flags a coverage entry whose file identifier cannot be
// normalized to a repository-relative path under the supplied repo root (for
// example an absolute build-machine path that escapes the root, or an import
// path outside the module). It mirrors sourcepath.ErrUnsafePath semantics, but
// normalization here is purely lexical: a coverage report is a snapshot and may
// reference files that have since moved, so we never require the path to exist
// on disk.
var ErrUnsafePath = errors.New("unsafe coverage path")

// Report is the shared coverage.v1 record produced from any supported format.
type Report struct {
	SchemaVersion string         `json:"schema_version"`
	RepoRoot      string         `json:"repo_root"`
	GeneratedAt   string         `json:"generated_at,omitempty"`
	SourceTool    string         `json:"source_tool"`
	SourceFormat  string         `json:"source_format"`
	Files         []FileCoverage `json:"files"`
	// UnavailableReasons records entries that could not be normalized (unsafe
	// or unrooted paths), so callers see them rather than silently dropping
	// them. This mirrors how complexity/quickdup disclose what they skipped.
	UnavailableReasons []string `json:"unavailable_reasons,omitempty"`
}

// FileCoverage is one repository-relative file's coverage record.
//
// The statements metric is the native unit both Go (`go test`) and Istanbul
// report; the line metric is derived from it (a line is covered when a covering
// statement executed). Both are surfaced distinctly here so a consumer never
// mistakes one for the other. Branch and function coverage are optional pointer
// fields populated only when the source format provides them (never
// synthesized — absence is reported, not zero).
type FileCoverage struct {
	Path         string  `json:"path"` // repo-relative POSIX path — THE stable key
	Language     string  `json:"language,omitempty"`
	LinesTotal   int     `json:"lines_total"`
	LinesCovered int     `json:"lines_covered"`
	LineCoverage float64 `json:"line_coverage"`

	// Statements is the native coverage unit (distinct from the derived line
	// metric above). Populated whenever the source supplies it; absent formats
	// leave these nil rather than reporting a false zero.
	StatementsTotal   *int     `json:"statements_total,omitempty"`
	StatementsCovered *int     `json:"statements_covered,omitempty"`
	StatementCoverage *float64 `json:"statement_coverage,omitempty"`

	BranchesTotal   *int     `json:"branches_total,omitempty"`
	BranchesCovered *int     `json:"branches_covered,omitempty"`
	BranchCoverage  *float64 `json:"branch_coverage,omitempty"`

	FunctionsTotal   *int             `json:"functions_total,omitempty"`
	FunctionsCovered *int             `json:"functions_covered,omitempty"`
	Symbols          []SymbolCoverage `json:"symbols,omitempty"`

	SourceTool   string `json:"source_tool"`
	SourceFormat string `json:"source_format"`
}

// SymbolCoverage is one executable symbol within a file. Deliberately no source
// position is persisted: the file path plus symbol name is the durable identity
// needed by Insights, while native report line/column details remain ephemeral.
type SymbolCoverage struct {
	Name              string `json:"name"`
	Kind              string `json:"kind"`
	Covered           bool   `json:"covered"`
	Hits              *int64 `json:"hits,omitempty"`
	StatementsTotal   *int   `json:"statements_total,omitempty"`
	StatementsCovered *int   `json:"statements_covered,omitempty"`
}

// normalizeKey converts a raw file identifier from a coverage report into a
// repository-relative POSIX path, or returns ErrUnsafePath if it cannot be
// rooted safely. modulePrefix, when non-empty, is stripped from import-style
// keys (Go coverprofiles, JaCoCo) before re-rooting.
//
// The normalization is purely lexical so a stale snapshot referencing a moved
// file still normalizes; it never touches the filesystem.
//
// Format parsing and metric counting are delegated to
// testtranslator.ParseCoverage; this classifier survives because testtranslator
// intentionally treats an out-of-module Go import path as a plain relative key
// rather than rejecting it, whereas Faktorial's contract is to disclose such a
// path as unavailable and keep the remaining files (collect + continue). Running
// this pre-filter first preserves that contract without hard-failing the whole
// report.
func normalizeKey(raw, repoRoot, modulePrefix string) (string, error) {
	key := strings.TrimSpace(raw)
	if key == "" {
		return "", fmt.Errorf("%w: empty path", ErrUnsafePath)
	}
	if strings.ContainsRune(key, '\x00') {
		return "", fmt.Errorf("%w: NUL byte in %q", ErrUnsafePath, raw)
	}

	normalizer := pathutil.New(repoRoot, modulePrefix)

	// Import-path style keys (e.g. Go coverprofile) carry the module prefix.
	if modulePrefix != "" {
		normalizedKey := strings.ReplaceAll(key, "\\", "/")
		prefix := strings.Trim(strings.ReplaceAll(modulePrefix, "\\", "/"), "/") + "/"
		if strings.HasPrefix(normalizedKey, prefix) {
			rel, err := normalizer.RelFromImportPath(normalizedKey)
			if err != nil {
				return "", fmt.Errorf("%w: %v", ErrUnsafePath, err)
			}
			return rel, nil
		}
		// Not under the module: it is a dependency or otherwise foreign to the
		// repository, so it cannot be a repo-relative key.
		if !isAbsoluteCoveragePath(key) {
			return "", fmt.Errorf("%w: %q is outside module %q", ErrUnsafePath, raw, modulePrefix)
		}
	}

	rel, err := normalizer.Rel(key)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrUnsafePath, err)
	}
	return rel, nil
}

// isAbsoluteCoveragePath recognizes both POSIX and Windows absolute paths
// independent of the host OS that happens to run the translator.
func isAbsoluteCoveragePath(value string) bool {
	slashed := strings.ReplaceAll(value, "\\", "/")
	if strings.HasPrefix(slashed, "/") {
		return true
	}
	return len(slashed) >= 3 &&
		((slashed[0] >= 'A' && slashed[0] <= 'Z') || (slashed[0] >= 'a' && slashed[0] <= 'z')) &&
		slashed[1] == ':' && slashed[2] == '/'
}

func ratio(covered, total int) float64 {
	if total <= 0 {
		return 0
	}
	r := float64(covered) / float64(total)
	// Round to 6 decimals so output is stable across platforms.
	return math.Round(r*1e6) / 1e6
}

func intPtr(v int) *int           { return &v }
func floatPtr(v float64) *float64 { return &v }

// sortFiles orders records by path for deterministic output.
func sortFiles(files []FileCoverage) {
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
}

// unsafeCollector deduplicates and accumulates the disclosed reasons for entries
// whose path could not be safely rerooted, matching the prior collect+continue
// behavior.
type unsafeCollector struct {
	seen    map[string]bool
	reasons []string
}

func newUnsafeCollector() *unsafeCollector {
	return &unsafeCollector{seen: map[string]bool{}}
}

func (u *unsafeCollector) add(raw string, err error) {
	if u.seen[raw] {
		return
	}
	u.seen[raw] = true
	u.reasons = append(u.reasons, err.Error())
}

// metricCount extracts a metric's covered/total pair from a testtranslator file
// record, reporting whether the metric was present.
func metricCount(f testtranslator.CoverageFile, m testtranslator.Metric) (covered, total int, ok bool) {
	c, present := f.Metrics[m]
	if !present {
		return 0, 0, false
	}
	return int(c.Covered), int(c.Total), true
}

// toFileCoverage folds one testtranslator file record into the coverage.v1
// record, surfacing the distinct statements metric alongside the derived line
// metric and preserving branch/function coverage as optional fields.
func toFileCoverage(f testtranslator.CoverageFile, language, sourceTool, sourceFormat string) FileCoverage {
	fc := FileCoverage{
		Path:         f.Path,
		Language:     language,
		SourceTool:   sourceTool,
		SourceFormat: sourceFormat,
	}

	// Line coverage: the derived metric (a line is covered when any covering
	// statement executed). It remains the always-present headline number.
	if covered, total, ok := metricCount(f, testtranslator.MetricLines); ok {
		fc.LinesTotal = total
		fc.LinesCovered = covered
		fc.LineCoverage = ratio(covered, total)
	}

	// Statements: the native unit, now surfaced distinctly rather than folded
	// into the line metric.
	if covered, total, ok := metricCount(f, testtranslator.MetricStatements); ok {
		fc.StatementsTotal = intPtr(total)
		fc.StatementsCovered = intPtr(covered)
		fc.StatementCoverage = floatPtr(ratio(covered, total))
	}

	// Branch coverage (optional): only when the format provides it.
	if covered, total, ok := metricCount(f, testtranslator.MetricBranches); ok {
		fc.BranchesTotal = intPtr(total)
		fc.BranchesCovered = intPtr(covered)
		fc.BranchCoverage = floatPtr(ratio(covered, total))
	}

	// Function coverage (optional): only when the format provides it.
	if covered, total, ok := metricCount(f, testtranslator.MetricFunctions); ok {
		fc.FunctionsTotal = intPtr(total)
		fc.FunctionsCovered = intPtr(covered)
	}
	for _, function := range f.Functions {
		hits := function.Hits
		fc.Symbols = append(fc.Symbols, SymbolCoverage{
			Name: function.Name, Kind: "function", Covered: hits > 0, Hits: &hits,
		})
	}
	sort.Slice(fc.Symbols, func(i, j int) bool {
		if fc.Symbols[i].Covered != fc.Symbols[j].Covered {
			return !fc.Symbols[i].Covered
		}
		return fc.Symbols[i].Name < fc.Symbols[j].Name
	})

	return fc
}

// --- Go coverprofile ---------------------------------------------------------

const goSourceTool = "go test -coverprofile"
const goSourceFormat = "coverprofile"
const goLanguage = "go"
const maxProjectionInputBytes int64 = 256 << 20

func readAllBounded(r io.Reader, context string, limit int64) ([]byte, error) {
	if r == nil {
		return nil, fmt.Errorf("%s: nil input", context)
	}
	lr := &io.LimitedReader{R: r, N: limit + 1}
	raw, err := io.ReadAll(lr)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", context, err)
	}
	if int64(len(raw)) > limit {
		return nil, fmt.Errorf("%s: input exceeds %d byte limit", context, limit)
	}
	return raw, nil
}

// ParseGoCoverprofile reads a Go coverprofile (the text emitted by
// `go test -coverprofile`) and translates it into a coverage.v1 Report.
//
// modulePath is the Go module path (e.g. github.com/asynkron/faktorial-go) used
// to strip the import-path prefix and recover repo-relative keys. Parsing and
// per-file statement/line counting are delegated to testtranslator; Go's native
// statement metric is surfaced distinctly and the derived line metric alongside
// it (branch/function coverage stay unset, as Go reports neither).
//
// Entries whose import path is outside the module are disclosed in
// UnavailableReasons and dropped, rather than hard-failing the whole report.
func ParseGoCoverprofile(r io.Reader, repoRoot, modulePath string) (*Report, error) {
	raw, err := readAllBounded(r, "coverprofile", maxProjectionInputBytes)
	if err != nil {
		return nil, err
	}

	unsafe := newUnsafeCollector()
	safe, err := filterGoCoverprofile(raw, repoRoot, modulePath, unsafe)
	if err != nil {
		return nil, fmt.Errorf("coverprofile: %w", err)
	}

	rep, _, perr := testtranslator.ParseCoverage("go-coverprofile", bytes.NewReader(safe),
		testtranslator.CoverageOptions{RepoRoot: repoRoot, GoModule: modulePath})
	if perr != nil {
		return nil, fmt.Errorf("coverprofile: %w", perr)
	}

	report := &Report{
		SchemaVersion: SchemaVersion,
		RepoRoot:      repoRoot,
		SourceTool:    goSourceTool,
		SourceFormat:  goSourceFormat,
	}
	for _, f := range rep.Files {
		report.Files = append(report.Files, toFileCoverage(f, goLanguage, goSourceTool, goSourceFormat))
	}
	enrichGoSymbols(report, safe, repoRoot, modulePath)
	finalize(report, unsafe)
	return report, nil
}

// enrichGoSymbols derives named function and method coverage while the source
// checkout is available. Go's profile has statement ranges but no symbol
// identities, so the ranges are intersected with declarations and only compact
// names/counts are retained in the durable coverage record.
func enrichGoSymbols(report *Report, profileBytes []byte, repoRoot, modulePath string) {
	profiles, err := cover.ParseProfilesFromReader(bytes.NewReader(profileBytes))
	if err != nil {
		return
	}
	byPath := make(map[string]*cover.Profile, len(profiles))
	for _, profile := range profiles {
		rel, err := normalizeKey(profile.FileName, repoRoot, modulePath)
		if err == nil {
			byPath[rel] = profile
		}
	}
	for i := range report.Files {
		profile := byPath[report.Files[i].Path]
		if profile == nil {
			continue
		}
		symbols, err := goSymbolsForFile(filepath.Join(repoRoot, filepath.FromSlash(report.Files[i].Path)), profile)
		if err != nil {
			continue
		}
		report.Files[i].Symbols = symbols
		if len(symbols) > 0 {
			total, covered := len(symbols), 0
			for _, symbol := range symbols {
				if symbol.Covered {
					covered++
				}
			}
			report.Files[i].FunctionsTotal = intPtr(total)
			report.Files[i].FunctionsCovered = intPtr(covered)
		}
	}
}

type goSymbolExtent struct {
	name                   string
	kind                   string
	startLine, startColumn int
	endLine, endColumn     int
}

func goSymbolsForFile(filename string, profile *cover.Profile) ([]SymbolCoverage, error) {
	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, filename, nil, 0)
	if err != nil {
		return nil, err
	}
	extents := make([]goSymbolExtent, 0)
	for _, decl := range parsed.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		start, end := fset.Position(fn.Pos()), fset.Position(fn.End())
		name, kind := fn.Name.Name, "function"
		if fn.Recv != nil && len(fn.Recv.List) > 0 {
			var receiver bytes.Buffer
			if err := format.Node(&receiver, fset, fn.Recv.List[0].Type); err == nil {
				name = "(" + receiver.String() + ")." + name
			}
			kind = "method"
		}
		extents = append(extents, goSymbolExtent{
			name: name, kind: kind,
			startLine: start.Line, startColumn: start.Column,
			endLine: end.Line, endColumn: end.Column,
		})
	}
	symbols := make([]SymbolCoverage, 0, len(extents))
	for _, extent := range extents {
		covered, total := extent.coverage(profile)
		if total == 0 {
			continue
		}
		symbols = append(symbols, SymbolCoverage{
			Name: extent.name, Kind: extent.kind, Covered: covered > 0,
			StatementsTotal: intPtr(total), StatementsCovered: intPtr(covered),
		})
	}
	sort.Slice(symbols, func(i, j int) bool {
		if symbols[i].Covered != symbols[j].Covered {
			return !symbols[i].Covered
		}
		return symbols[i].Name < symbols[j].Name
	})
	return symbols, nil
}

func (f goSymbolExtent) coverage(profile *cover.Profile) (covered, total int) {
	for _, block := range profile.Blocks {
		if block.StartLine > f.endLine || block.StartLine == f.endLine && block.StartCol >= f.endColumn {
			break
		}
		if block.EndLine < f.startLine || block.EndLine == f.startLine && block.EndCol <= f.startColumn {
			continue
		}
		total += block.NumStmt
		if block.Count > 0 {
			covered += block.NumStmt
		}
	}
	return covered, total
}

// filterGoCoverprofile splits a coverprofile into the subset of lines whose file
// identifier normalizes to a safe repo-relative key, disclosing the rest through
// the collector. The mode directive and any structurally-unexpected lines are
// preserved so testtranslator still sees a well-formed (or honestly malformed)
// profile.
func filterGoCoverprofile(raw []byte, repoRoot, modulePath string, unsafe *unsafeCollector) ([]byte, error) {
	var out bytes.Buffer
	sc := bufio.NewScanner(bytes.NewReader(raw))
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for sc.Scan() {
		line := sc.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "mode:") {
			out.WriteString(line)
			out.WriteByte('\n')
			continue
		}
		// A coverage line is `file:sLine.sCol,eLine.eCol numStmt count`; the file
		// identifier is everything before the last colon (matching the shared
		// parser's own split).
		colon := strings.LastIndex(trimmed, ":")
		if colon < 0 {
			// Structurally unexpected — hand it to testtranslator to judge.
			out.WriteString(line)
			out.WriteByte('\n')
			continue
		}
		file := trimmed[:colon]
		if _, err := normalizeKey(file, repoRoot, modulePath); err != nil {
			unsafe.add(file, err)
			continue
		}
		out.WriteString(line)
		out.WriteByte('\n')
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("read error: %w", err)
	}
	return out.Bytes(), nil
}

// --- Vitest / Istanbul coverage-final.json -----------------------------------

const vitestSourceTool = "vitest (@vitest/coverage-istanbul)"
const vitestSourceFormat = "istanbul-json"
const vitestLanguage = "typescript"

// ParseVitestIstanbul reads a Vitest-produced Istanbul coverage-final.json and
// translates it into a coverage.v1 Report. Istanbul file keys are typically
// absolute build-machine paths, so they are re-rooted against repoRoot;
// relative keys are accepted too. Parsing and per-file statement/line/branch/
// function counting are delegated to testtranslator, and the native statement
// metric is surfaced distinctly.
//
// Paths that escape the repo root are disclosed in UnavailableReasons and
// dropped rather than hard-failing the whole report.
func ParseVitestIstanbul(r io.Reader, repoRoot string) (*Report, error) {
	raw, err := readAllBounded(r, "istanbul json", maxProjectionInputBytes)
	if err != nil {
		return nil, err
	}

	unsafe := newUnsafeCollector()
	safe, kept, ferr := filterIstanbul(raw, repoRoot, unsafe)
	if ferr != nil {
		return nil, fmt.Errorf("istanbul json: %w", ferr)
	}

	report := &Report{
		SchemaVersion: SchemaVersion,
		RepoRoot:      repoRoot,
		SourceTool:    vitestSourceTool,
		SourceFormat:  vitestSourceFormat,
	}
	// An empty safe subset is a valid, fully-unavailable report: every entry
	// escaped the repo root. testtranslator rejects an empty document, so skip
	// the call and disclose the reasons rather than hard-failing.
	if kept > 0 {
		rep, _, perr := testtranslator.ParseCoverage("istanbul-json", bytes.NewReader(safe),
			testtranslator.CoverageOptions{RepoRoot: repoRoot})
		if perr != nil {
			return nil, fmt.Errorf("istanbul json: %w", perr)
		}
		for _, f := range rep.Files {
			report.Files = append(report.Files, toFileCoverage(f, vitestLanguage, vitestSourceTool, vitestSourceFormat))
		}
	}
	finalize(report, unsafe)
	return report, nil
}

// filterIstanbul decodes a coverage-final.json object and returns a re-encoded
// object containing only the entries whose path normalizes safely, along with
// the number of entries kept. The rest are disclosed through the collector. A
// malformed document (not a JSON object of entries) returns an error so the
// caller can surface it, matching the prior hard-fail on unparseable input.
func filterIstanbul(raw []byte, repoRoot string, unsafe *unsafeCollector) ([]byte, int, error) {
	var entries map[string]json.RawMessage
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil, 0, err
	}
	kept := make(map[string]json.RawMessage, len(entries))
	// Deterministic classification order keeps disclosed reasons stable.
	keys := make([]string, 0, len(entries))
	for k := range entries {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		body := entries[k]
		var meta struct {
			Path string `json:"path"`
		}
		_ = json.Unmarshal(body, &meta)
		rawPath := meta.Path
		if strings.TrimSpace(rawPath) == "" {
			rawPath = k
		}
		if _, err := normalizeKey(rawPath, repoRoot, ""); err != nil {
			unsafe.add(rawPath, err)
			continue
		}
		kept[k] = body
	}
	encoded, err := json.Marshal(kept)
	if err != nil {
		return nil, 0, err
	}
	return encoded, len(kept), nil
}

// finalize sorts the report's files and merges the disclosed unsafe reasons.
func finalize(report *Report, unsafe *unsafeCollector) {
	sortFiles(report.Files)
	report.UnavailableReasons = append(report.UnavailableReasons, unsafe.reasons...)
	sort.Strings(report.UnavailableReasons)
}
