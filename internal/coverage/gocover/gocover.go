// Package gocover converts Go coverprofiles (`go test -coverprofile`) in set,
// count, and atomic modes into the internal coverage model. Go's native
// coverage unit is the statement, so statement counts are preserved as the
// native metric and per-line hits are marked as derived from statement blocks.
package gocover

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/asynkron/Asynkron.TestTranslator/internal/coverage"
)

func init() {
	coverage.Register("go-coverprofile", "Go coverprofile (set, count, atomic)", Adapter{}, "gocover", "gocoverage")
}

const defaultMaxInput = 256 << 20

// Adapter implements coverage.Adapter for Go coverprofiles.
type Adapter struct{}

// block is one parsed coverprofile record.
type block struct {
	startLine int
	endLine   int
	numStmt   int
	count     int64
}

// fileAgg aggregates line hits and statement totals for one source file.
type fileAgg struct {
	// lineHits maps a line number to its maximum observed hit count so
	// overlapping blocks do not double-count.
	lineHits   map[int]int64
	stmtTotal  int64
	stmtCover  int64
	firstOrder int
}

// Parse reads a coverprofile and produces a normalized coverage report.
func (Adapter) Parse(r io.Reader, opts coverage.Options) (*coverage.Report, error) {
	if opts.Paths == nil {
		return nil, fmt.Errorf("gocover: path normalizer is required")
	}
	limit := opts.MaxInputBytes
	if limit <= 0 {
		limit = defaultMaxInput
	}
	lr := &io.LimitedReader{R: r, N: limit + 1}
	sc := bufio.NewScanner(lr)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)

	files := map[string]*fileAgg{}
	order := 0
	mode := ""
	lineNo := 0
	for sc.Scan() {
		lineNo++
		text := strings.TrimSpace(sc.Text())
		if text == "" {
			continue
		}
		if strings.HasPrefix(text, "mode:") {
			mode = strings.TrimSpace(strings.TrimPrefix(text, "mode:"))
			switch mode {
			case "set", "count", "atomic":
			default:
				return nil, fmt.Errorf("gocover: unknown coverage mode %q", mode)
			}
			continue
		}
		if mode == "" {
			return nil, fmt.Errorf("gocover: line %d: missing leading `mode:` directive", lineNo)
		}
		srcFile, blk, err := parseLine(text)
		if err != nil {
			return nil, fmt.Errorf("gocover: line %d: %w", lineNo, err)
		}
		rel, err := resolvePath(opts, srcFile)
		if err != nil {
			return nil, fmt.Errorf("gocover: %w", err)
		}
		agg, ok := files[rel]
		if !ok {
			agg = &fileAgg{lineHits: map[int]int64{}, firstOrder: order}
			files[rel] = agg
			order++
		}
		agg.stmtTotal += int64(blk.numStmt)
		if blk.count > 0 {
			agg.stmtCover += int64(blk.numStmt)
		}
		for ln := blk.startLine; ln <= blk.endLine; ln++ {
			if blk.count > agg.lineHits[ln] {
				agg.lineHits[ln] = blk.count
			} else if _, seen := agg.lineHits[ln]; !seen {
				agg.lineHits[ln] = blk.count
			}
		}
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("gocover: read error: %w", err)
	}
	if lr.N <= 0 {
		return nil, fmt.Errorf("gocover: input exceeds %d byte limit", limit)
	}
	if mode == "" {
		return nil, fmt.Errorf("gocover: empty or missing coverprofile")
	}

	report := &coverage.Report{
		Producer:          "go",
		InterchangeFormat: "go-coverprofile",
		SourceRoots:       []string{"."},
	}
	for path, agg := range files {
		f := coverage.File{Path: path}
		for ln, hits := range agg.lineHits {
			if err := f.AddLine(coverage.LineHit{Number: ln, Hits: hits}); err != nil {
				return nil, fmt.Errorf("gocover: %w", err)
			}
		}
		// Statements are the native Go metric.
		f.SetMetric(coverage.MetricStatements, coverage.Count{
			Covered: agg.stmtCover,
			Total:   agg.stmtTotal,
		})
		// Line coverage is derived from statement blocks; mark it as such.
		var lc int64
		for _, h := range agg.lineHits {
			if h > 0 {
				lc++
			}
		}
		f.SetMetric(coverage.MetricLines, coverage.Count{
			Covered:        lc,
			Total:          int64(len(agg.lineHits)),
			Derived:        true,
			DerivationRule: "a line is covered when any covering statement block executed at least once",
		})
		report.Files = append(report.Files, f)
	}
	opts.Diag.Notef("gocover.derived", opts.SourceName, "line coverage derived from statement blocks; statement counts preserved as the native metric")

	if err := report.Normalize(); err != nil {
		return nil, err
	}
	return report, nil
}

// parseLine parses `file:sLine.sCol,eLine.eCol numStmt count`.
func parseLine(text string) (string, block, error) {
	colon := strings.LastIndex(text, ":")
	if colon < 0 {
		return "", block{}, fmt.Errorf("missing file:range separator in %q", text)
	}
	file := text[:colon]
	rest := text[colon+1:]
	fields := strings.Fields(rest)
	if len(fields) != 3 {
		return "", block{}, fmt.Errorf("expected `range numStmt count`, got %q", rest)
	}
	rng := fields[0]
	comma := strings.IndexByte(rng, ',')
	if comma < 0 {
		return "", block{}, fmt.Errorf("malformed range %q", rng)
	}
	start := rng[:comma]
	end := rng[comma+1:]
	sLine, err := lineOf(start)
	if err != nil {
		return "", block{}, err
	}
	eLine, err := lineOf(end)
	if err != nil {
		return "", block{}, err
	}
	if eLine < sLine {
		return "", block{}, fmt.Errorf("range end line %d before start line %d", eLine, sLine)
	}
	numStmt, err := strconv.Atoi(fields[1])
	if err != nil || numStmt < 0 {
		return "", block{}, fmt.Errorf("invalid statement count %q", fields[1])
	}
	count, err := strconv.ParseInt(fields[2], 10, 64)
	if err != nil || count < 0 {
		return "", block{}, fmt.Errorf("invalid execution count %q", fields[2])
	}
	return file, block{startLine: sLine, endLine: eLine, numStmt: numStmt, count: count}, nil
}

// lineOf parses the "line.col" half of a range and returns the line.
func lineOf(s string) (int, error) {
	dot := strings.IndexByte(s, '.')
	if dot < 0 {
		return 0, fmt.Errorf("malformed position %q", s)
	}
	n, err := strconv.Atoi(s[:dot])
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("invalid line number %q", s[:dot])
	}
	return n, nil
}

// resolvePath reroots a coverprofile file into repo-relative POSIX form. Go
// coverprofiles use module-qualified import paths; fall back to path-based
// rerooting when a repo root is supplied.
func resolvePath(opts coverage.Options, srcFile string) (string, error) {
	mod := opts.Paths.ModulePath()
	if mod != "" && (srcFile == mod || strings.HasPrefix(srcFile, mod+"/")) {
		return opts.Paths.RelFromImportPath(srcFile)
	}
	rel, err := opts.Paths.Rel(srcFile)
	if err == nil {
		return rel, nil
	}
	return "", fmt.Errorf("cannot reroot %q: supply --go-module for import-path profiles or --repo-root for path-based profiles (%v)", srcFile, err)
}
