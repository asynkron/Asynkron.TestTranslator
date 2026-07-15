// Package coverage defines the internal coverage model and the adapter contract
// for coverage source parsers. Cobertura XML is the public output contract; the
// internal model preserves richer native metrics so conversion losses can be
// detected and reported.
package coverage

import (
	"fmt"
	"sort"
)

// LineHit records execution count for a single source line.
type LineHit struct {
	// Number is the 1-based source line number.
	Number int
	// Hits is the execution count. For "set"-mode profiles it is 0 or 1.
	Hits int64
	// Branch marks the line as a branch point when the source supplies branch
	// data.
	Branch bool
	// BranchesCovered and BranchesTotal describe branch coverage on this line
	// when Branch is true.
	BranchesCovered int
	BranchesTotal   int
}

// Metric names a distinct coverage dimension. Keeping them distinct prevents
// mislabeling (for example, Go statement counts must not be reported as literal
// line counts).
type Metric string

const (
	MetricStatements   Metric = "statements"
	MetricLines        Metric = "lines"
	MetricBranches     Metric = "branches"
	MetricFunctions    Metric = "functions"
	MetricMethods      Metric = "methods"
	MetricInstructions Metric = "instructions"
	MetricRegions      Metric = "regions"
)

// Count is a covered/total pair for one metric, optionally marked as derived.
type Count struct {
	Covered int64
	Total   int64
	// Derived is true when this metric was computed rather than supplied
	// natively by the source.
	Derived bool
	// DerivationRule documents how a derived metric was computed.
	DerivationRule string
}

// File is coverage for a single repository-relative source file.
type File struct {
	// Path is the repo-relative POSIX path; it is the file identity.
	Path string
	// Lines are per-line hits, ordered by line number after Normalize.
	Lines []LineHit
	// Metrics holds native and derived counts keyed by metric.
	Metrics map[Metric]Count
	// Functions optionally records per-function coverage.
	Functions []Function
}

// Function is optional per-function coverage.
type Function struct {
	Name string
	Line int
	Hits int64
}

// Report is the complete converted coverage document.
type Report struct {
	// SourceRoots are absolute or repo-relative roots preserved for Cobertura
	// consumers.
	SourceRoots []string
	// Files are coverage entries keyed by path, ordered deterministically after
	// Normalize.
	Files []File
	// Producer names the coverage producer (e.g. "go", "vitest-v8").
	Producer string
	// InterchangeFormat names the on-disk format (e.g. "istanbul-json").
	InterchangeFormat string
}

// AddLine records or merges a line hit into a file, rejecting contradictory
// input. Repeated lines accumulate hits deterministically.
func (f *File) AddLine(l LineHit) error {
	if l.Number <= 0 {
		return fmt.Errorf("file %q has non-positive line number %d", f.Path, l.Number)
	}
	if l.Hits < 0 {
		return fmt.Errorf("file %q line %d has negative hit count", f.Path, l.Number)
	}
	if l.Branch && l.BranchesTotal < l.BranchesCovered {
		return fmt.Errorf("file %q line %d has more covered branches than total", f.Path, l.Number)
	}
	for i := range f.Lines {
		if f.Lines[i].Number == l.Number {
			f.Lines[i].Hits += l.Hits
			if l.Branch {
				f.Lines[i].Branch = true
				f.Lines[i].BranchesCovered += l.BranchesCovered
				f.Lines[i].BranchesTotal += l.BranchesTotal
			}
			return nil
		}
	}
	f.Lines = append(f.Lines, l)
	return nil
}

// SetMetric records a native metric count.
func (f *File) SetMetric(m Metric, c Count) {
	if f.Metrics == nil {
		f.Metrics = map[Metric]Count{}
	}
	f.Metrics[m] = c
}

// Normalize sorts files by path and lines by number, and validates counts. It
// must be called before writing so output is deterministic.
func (r *Report) Normalize() error {
	sort.Slice(r.Files, func(i, j int) bool { return r.Files[i].Path < r.Files[j].Path })
	seen := map[string]bool{}
	for i := range r.Files {
		f := &r.Files[i]
		if f.Path == "" {
			return fmt.Errorf("coverage report contains a file with an empty path")
		}
		if seen[f.Path] {
			return fmt.Errorf("coverage report contains duplicate file %q; use an explicit merge mode", f.Path)
		}
		seen[f.Path] = true
		sort.Slice(f.Lines, func(a, b int) bool { return f.Lines[a].Number < f.Lines[b].Number })
		for m, c := range f.Metrics {
			if c.Total < c.Covered {
				return fmt.Errorf("file %q metric %q has covered %d greater than total %d", f.Path, m, c.Covered, c.Total)
			}
			if c.Covered < 0 || c.Total < 0 {
				return fmt.Errorf("file %q metric %q has negative count", f.Path, m)
			}
		}
	}
	sort.Strings(r.SourceRoots)
	return nil
}

// LineRate returns the covered/total line ratio across all files, or 0 when no
// line data exists.
func (r *Report) LineRate() float64 {
	var covered, total int64
	for i := range r.Files {
		for _, l := range r.Files[i].Lines {
			total++
			if l.Hits > 0 {
				covered++
			}
		}
	}
	if total == 0 {
		return 0
	}
	return float64(covered) / float64(total)
}

// BranchRate returns the covered/total branch ratio across all files, or 0 when
// no branch data exists. The second return reports whether any branch data was
// present.
func (r *Report) BranchRate() (float64, bool) {
	var covered, total int64
	for i := range r.Files {
		for _, l := range r.Files[i].Lines {
			if l.Branch {
				covered += int64(l.BranchesCovered)
				total += int64(l.BranchesTotal)
			}
		}
	}
	if total == 0 {
		return 0, false
	}
	return float64(covered) / float64(total), true
}
