// Package lcov converts LCOV tracefiles (conventionally named `lcov.info`) into
// the internal coverage model. LCOV is a line-oriented text format emitted by
// gcov/lcov itself and, in a compatible dialect, by JavaScript tooling such as
// Istanbul/nyc, Jest and Vitest (the `lcovonly` reporter).
//
// A tracefile is a sequence of per-file records. Each record begins with an
// `SF:` (source file) directive, carries any number of line (`DA:`), branch
// (`BRDA:`) and function (`FN:`/`FNDA:`) directives plus optional summary
// counters, and is terminated by `end_of_record`. LCOV is fundamentally a
// line-based format, so line coverage is preserved as the native metric rather
// than derived. Branch and function counts are preserved as native metrics
// when present.
package lcov

import (
	"bufio"
	"fmt"
	"io"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/asynkron/testtranslator/internal/coverage"
)

func init() {
	coverage.Register(
		"lcov",
		"LCOV tracefile (lcov.info), including Istanbul/nyc lcovonly output",
		Adapter{},
		"lcov-info",
	)
}

// defaultMaxInput bounds the amount of text read when opts.MaxInputBytes is not
// set, guarding against unbounded or hostile input.
const defaultMaxInput = 256 << 20

// Adapter implements coverage.Adapter for LCOV tracefiles.
type Adapter struct{}

// lineAgg aggregates the line- and branch-level data seen for a single source
// line before it is committed to the coverage model. LCOV may report several
// `DA:`/`BRDA:` directives touching the same line, so they are merged locally
// and emitted with a single AddLine call.
type lineAgg struct {
	hits          int64
	hasDA         bool
	branchesTotal int
	branchCovered int
	hasBranch     bool
	order         int
}

// fileAgg accumulates all directives belonging to one `SF:` record.
type fileAgg struct {
	path      string
	lines     map[int]*lineAgg
	lineOrder int

	functions []coverage.Function
	fnIndex   map[string]int // function name -> index into functions

	// Explicit summary counters, when the tracefile supplies them.
	lf, lh   int64
	fnf, fnh int64
	brf, brh int64
	haveLF   bool
	haveLH   bool
	haveFNF  bool
	haveFNH  bool
	haveBRF  bool
	haveBRH  bool

	sawBRDA bool
}

// Parse reads an LCOV tracefile and produces a normalized coverage report. It
// fails clearly when the input contains no `SF:` records or when a coverage
// directive is malformed. It performs no fuzzy detection: input that does not
// look like an LCOV tracefile is rejected.
func (Adapter) Parse(r io.Reader, opts coverage.Options) (*coverage.Report, error) {
	limit := opts.MaxInputBytes
	if limit <= 0 {
		limit = defaultMaxInput
	}
	lr := &io.LimitedReader{R: r, N: limit + 1}
	sc := bufio.NewScanner(lr)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)

	var files []*fileAgg
	var cur *fileAgg
	droppedChecksums := false
	droppedTestNames := false

	lineNo := 0
	for sc.Scan() {
		lineNo++
		text := strings.TrimSpace(sc.Text())
		if text == "" {
			continue
		}

		prefix, value := splitDirective(text)
		switch prefix {
		case "TN":
			// Test-name records have no representation in the coverage model.
			if strings.TrimSpace(value) != "" && !droppedTestNames {
				opts.Diag.Warnf("lcov.tn-dropped", opts.SourceName,
					"LCOV test names (TN:) are not representable in the coverage model and were dropped")
				droppedTestNames = true
			}
		case "SF":
			// A new source-file record begins. Flush any record that lacked an
			// explicit end_of_record so its data is never silently dropped.
			if cur != nil {
				files = append(files, cur)
			}
			rel, err := resolvePath(opts, value)
			if err != nil {
				return nil, fmt.Errorf("lcov: line %d: %w", lineNo, err)
			}
			cur = newFileAgg(rel)
		case "DA":
			if cur == nil {
				return nil, fmt.Errorf("lcov: line %d: DA directive outside of an SF record", lineNo)
			}
			ln, hits, hadChecksum, err := parseDA(value)
			if err != nil {
				return nil, fmt.Errorf("lcov: line %d: %w", lineNo, err)
			}
			if hadChecksum && !droppedChecksums {
				opts.Diag.Warnf("lcov.checksum-dropped", opts.SourceName,
					"LCOV line checksums (third DA: field) are not representable and were dropped")
				droppedChecksums = true
			}
			la := cur.line(ln)
			if la.hits < hits {
				la.hits = hits
			}
			la.hasDA = true
		case "BRDA":
			if cur == nil {
				return nil, fmt.Errorf("lcov: line %d: BRDA directive outside of an SF record", lineNo)
			}
			ln, taken, err := parseBRDA(value)
			if err != nil {
				return nil, fmt.Errorf("lcov: line %d: %w", lineNo, err)
			}
			cur.sawBRDA = true
			la := cur.line(ln)
			la.hasBranch = true
			la.branchesTotal++
			if taken > 0 {
				la.branchCovered++
			}
		case "FN":
			if cur == nil {
				return nil, fmt.Errorf("lcov: line %d: FN directive outside of an SF record", lineNo)
			}
			ln, name, err := parseFN(value)
			if err != nil {
				return nil, fmt.Errorf("lcov: line %d: %w", lineNo, err)
			}
			cur.setFunctionLine(name, ln)
		case "FNDA":
			if cur == nil {
				return nil, fmt.Errorf("lcov: line %d: FNDA directive outside of an SF record", lineNo)
			}
			hits, name, err := parseFNDA(value)
			if err != nil {
				return nil, fmt.Errorf("lcov: line %d: %w", lineNo, err)
			}
			cur.setFunctionHits(name, hits)
		case "LF", "LH", "FNF", "FNH", "BRF", "BRH":
			if cur == nil {
				return nil, fmt.Errorf("lcov: line %d: %s directive outside of an SF record", lineNo, prefix)
			}
			n, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
			if err != nil || n < 0 {
				return nil, fmt.Errorf("lcov: line %d: invalid %s counter %q", lineNo, prefix, value)
			}
			cur.setSummary(prefix, n)
		case "end_of_record":
			if cur == nil {
				return nil, fmt.Errorf("lcov: line %d: end_of_record without a preceding SF record", lineNo)
			}
			files = append(files, cur)
			cur = nil
		default:
			// Unknown record prefixes (e.g. VER:, VERSION:, comments) are
			// ignored so newer LCOV variants still parse, but the enclosing
			// file record is preserved.
		}
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("lcov: read error: %w", err)
	}
	if lr.N <= 0 {
		return nil, fmt.Errorf("lcov: input exceeds %d byte limit", limit)
	}
	// Flush a trailing record that was missing its end_of_record marker.
	if cur != nil {
		files = append(files, cur)
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("lcov: no SF (source file) records found; not an LCOV tracefile")
	}

	report := &coverage.Report{
		Producer:          "lcov",
		InterchangeFormat: "lcov-info",
	}
	for _, fa := range files {
		f, err := fa.build()
		if err != nil {
			return nil, fmt.Errorf("lcov: %w", err)
		}
		report.Files = append(report.Files, f)
	}
	if err := report.Normalize(); err != nil {
		return nil, err
	}
	return report, nil
}

// newFileAgg constructs an empty aggregator for the given repo-relative path.
func newFileAgg(rel string) *fileAgg {
	return &fileAgg{
		path:    rel,
		lines:   map[int]*lineAgg{},
		fnIndex: map[string]int{},
	}
}

// line returns the aggregator for a source line, creating it on first use and
// recording first-seen order for stable behaviour before normalization.
func (fa *fileAgg) line(n int) *lineAgg {
	la, ok := fa.lines[n]
	if !ok {
		la = &lineAgg{order: fa.lineOrder}
		fa.lineOrder++
		fa.lines[n] = la
	}
	return la
}

// setFunctionLine records or updates a function's declaration line, preserving
// first-seen order.
func (fa *fileAgg) setFunctionLine(name string, ln int) {
	if idx, ok := fa.fnIndex[name]; ok {
		fa.functions[idx].Line = ln
		return
	}
	fa.fnIndex[name] = len(fa.functions)
	fa.functions = append(fa.functions, coverage.Function{Name: name, Line: ln})
}

// setFunctionHits records a function's execution count, creating the entry if
// only an FNDA directive (without a matching FN) was seen.
func (fa *fileAgg) setFunctionHits(name string, hits int64) {
	if idx, ok := fa.fnIndex[name]; ok {
		fa.functions[idx].Hits = hits
		return
	}
	fa.fnIndex[name] = len(fa.functions)
	fa.functions = append(fa.functions, coverage.Function{Name: name, Hits: hits})
}

// setSummary stores an explicit summary counter.
func (fa *fileAgg) setSummary(prefix string, n int64) {
	switch prefix {
	case "LF":
		fa.lf, fa.haveLF = n, true
	case "LH":
		fa.lh, fa.haveLH = n, true
	case "FNF":
		fa.fnf, fa.haveFNF = n, true
	case "FNH":
		fa.fnh, fa.haveFNH = n, true
	case "BRF":
		fa.brf, fa.haveBRF = n, true
	case "BRH":
		fa.brh, fa.haveBRH = n, true
	}
}

// build converts an aggregated file record into a coverage.File.
func (fa *fileAgg) build() (coverage.File, error) {
	f := coverage.File{Path: fa.path}

	// Emit lines in ascending line-number order for deterministic output.
	nums := make([]int, 0, len(fa.lines))
	for n := range fa.lines {
		nums = append(nums, n)
	}
	sort.Ints(nums)

	var daTotal, daCovered int64
	for _, n := range nums {
		la := fa.lines[n]
		lh := coverage.LineHit{Number: n, Hits: la.hits}
		if la.hasBranch {
			lh.Branch = true
			lh.BranchesTotal = la.branchesTotal
			lh.BranchesCovered = la.branchCovered
		}
		if err := f.AddLine(lh); err != nil {
			return coverage.File{}, err
		}
		if la.hasDA {
			daTotal++
			if la.hits > 0 {
				daCovered++
			}
		}
	}

	// Line coverage is LCOV's native metric.
	lineCount := coverage.Count{Covered: daCovered, Total: daTotal}
	if fa.haveLF {
		lineCount.Total = fa.lf
	}
	if fa.haveLH {
		lineCount.Covered = fa.lh
	}
	f.SetMetric(coverage.MetricLines, lineCount)

	// Function coverage, when the tracefile carries any function data.
	if len(fa.functions) > 0 || fa.haveFNF || fa.haveFNH {
		var fnTotal, fnCovered int64
		for _, fn := range fa.functions {
			fnTotal++
			if fn.Hits > 0 {
				fnCovered++
			}
		}
		fnCount := coverage.Count{Covered: fnCovered, Total: fnTotal}
		if fa.haveFNF {
			fnCount.Total = fa.fnf
		}
		if fa.haveFNH {
			fnCount.Covered = fa.fnh
		}
		f.SetMetric(coverage.MetricFunctions, fnCount)
		f.Functions = fa.functions
	}

	// Branch coverage, only when BRDA directives were present.
	if fa.sawBRDA || fa.haveBRF || fa.haveBRH {
		var brTotal, brCovered int64
		for _, n := range nums {
			la := fa.lines[n]
			if la.hasBranch {
				brTotal += int64(la.branchesTotal)
				brCovered += int64(la.branchCovered)
			}
		}
		brCount := coverage.Count{Covered: brCovered, Total: brTotal}
		if fa.haveBRF {
			brCount.Total = fa.brf
		}
		if fa.haveBRH {
			brCount.Covered = fa.brh
		}
		f.SetMetric(coverage.MetricBranches, brCount)
	}

	return f, nil
}

// splitDirective splits a tracefile line into its uppercase-ish prefix and the
// value following the first colon. Lines without a colon (such as
// `end_of_record`) return the whole line as the prefix and an empty value.
func splitDirective(text string) (prefix, value string) {
	if i := strings.IndexByte(text, ':'); i >= 0 {
		return text[:i], text[i+1:]
	}
	return text, ""
}

// parseDA parses a `DA:` value of the form `line,hits[,checksum]`.
func parseDA(value string) (line int, hits int64, hadChecksum bool, err error) {
	fields := strings.Split(value, ",")
	if len(fields) < 2 {
		return 0, 0, false, fmt.Errorf("malformed DA directive %q: expected line,hits", value)
	}
	line, err = strconv.Atoi(strings.TrimSpace(fields[0]))
	if err != nil || line <= 0 {
		return 0, 0, false, fmt.Errorf("invalid DA line number %q", fields[0])
	}
	hits, err = strconv.ParseInt(strings.TrimSpace(fields[1]), 10, 64)
	if err != nil || hits < 0 {
		return 0, 0, false, fmt.Errorf("invalid DA hit count %q", fields[1])
	}
	return line, hits, len(fields) > 2 && strings.TrimSpace(fields[2]) != "", nil
}

// parseBRDA parses a `BRDA:` value of the form `line,block,branch,taken` where
// taken is `-` when the branch was never reached, otherwise an execution count.
func parseBRDA(value string) (line int, taken int64, err error) {
	fields := strings.Split(value, ",")
	if len(fields) < 4 {
		return 0, 0, fmt.Errorf("malformed BRDA directive %q: expected line,block,branch,taken", value)
	}
	line, err = strconv.Atoi(strings.TrimSpace(fields[0]))
	if err != nil || line <= 0 {
		return 0, 0, fmt.Errorf("invalid BRDA line number %q", fields[0])
	}
	t := strings.TrimSpace(fields[3])
	if t == "-" {
		return line, 0, nil
	}
	taken, err = strconv.ParseInt(t, 10, 64)
	if err != nil || taken < 0 {
		return 0, 0, fmt.Errorf("invalid BRDA taken count %q", fields[3])
	}
	return line, taken, nil
}

// parseFN parses an `FN:` value of the form `line,name`.
func parseFN(value string) (line int, name string, err error) {
	parts := strings.SplitN(value, ",", 2)
	if len(parts) < 2 {
		return 0, "", fmt.Errorf("malformed FN directive %q: expected line,name", value)
	}
	line, err = strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil || line <= 0 {
		return 0, "", fmt.Errorf("invalid FN line number %q", parts[0])
	}
	name = strings.TrimSpace(parts[1])
	if name == "" {
		return 0, "", fmt.Errorf("empty FN function name in %q", value)
	}
	return line, name, nil
}

// parseFNDA parses an `FNDA:` value of the form `hits,name`.
func parseFNDA(value string) (hits int64, name string, err error) {
	parts := strings.SplitN(value, ",", 2)
	if len(parts) < 2 {
		return 0, "", fmt.Errorf("malformed FNDA directive %q: expected hits,name", value)
	}
	hits, err = strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
	if err != nil || hits < 0 {
		return 0, "", fmt.Errorf("invalid FNDA hit count %q", parts[0])
	}
	name = strings.TrimSpace(parts[1])
	if name == "" {
		return 0, "", fmt.Errorf("empty FNDA function name in %q", value)
	}
	return hits, name, nil
}

// resolvePath normalizes a `SF:` source path into repo-relative POSIX form. When
// a normalizer is configured it is used (and rejects escaping paths); otherwise
// the path is cleaned to POSIX form and stripped of a leading slash so it is
// non-empty and repo-relative.
func resolvePath(opts coverage.Options, src string) (string, error) {
	src = strings.TrimSpace(src)
	if src == "" {
		return "", fmt.Errorf("empty SF source path")
	}
	if opts.Paths != nil {
		rel, err := opts.Paths.Rel(src)
		if err != nil {
			return "", fmt.Errorf("cannot normalize %q: %w", src, err)
		}
		return rel, nil
	}
	cleaned := path.Clean(strings.ReplaceAll(src, "\\", "/"))
	cleaned = strings.TrimPrefix(cleaned, "/")
	if cleaned == "" || cleaned == "." {
		return "", fmt.Errorf("SF source path %q resolves to an empty repo-relative path", src)
	}
	return cleaned, nil
}
