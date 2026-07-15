package testtranslator

import (
	"io"

	"github.com/asynkron/testtranslator/internal/coverage"
	"github.com/asynkron/testtranslator/internal/diagnostics"
	"github.com/asynkron/testtranslator/internal/pathutil"
	"github.com/asynkron/testtranslator/internal/results"
)

// ParseResults parses test-result data of the named format into the structured
// [TestReport] model, without rendering JUnit XML. Use this when you want the
// parsed suites, cases, and totals programmatically (e.g. to persist or join
// them) rather than a JUnit document. format is a canonical id or alias (see
// [ResultFormats]). Suites and cases are deterministically ordered. It returns
// diagnostics for lossy or unsupported mappings and an error for malformed or
// mismatched input. The input is read through a bounded reader.
func ParseResults(format string, r io.Reader) (*TestReport, []Diagnostic, error) {
	adapter, err := results.Lookup(format)
	if err != nil {
		return nil, nil, err
	}
	diag := diagnostics.NewCollector()
	rep, err := adapter.Parse(r, results.Options{SourceName: "input", Diag: diag})
	if err != nil {
		return nil, collectDiagnostics(diag), err
	}
	rep.Sort()
	return toTestReport(rep), collectDiagnostics(diag), nil
}

// ParseCoverage parses coverage data of the named format into the structured
// [CoverageReport] model, without rendering Cobertura XML. Use this when you
// want per-file coverage metrics programmatically. format is a canonical id or
// alias (see [CoverageFormats]); opt controls source-path normalization. Files
// are deterministically ordered by path. It returns diagnostics and an error for
// malformed input or paths that cannot be rerooted.
func ParseCoverage(format string, r io.Reader, opt CoverageOptions) (*CoverageReport, []Diagnostic, error) {
	adapter, err := coverage.Lookup(format)
	if err != nil {
		return nil, nil, err
	}
	diag := diagnostics.NewCollector()
	rep, err := adapter.Parse(r, coverage.Options{
		SourceName: "input",
		Paths:      pathutil.New(opt.RepoRoot, opt.GoModule),
		Diag:       diag,
	})
	if err != nil {
		return nil, collectDiagnostics(diag), err
	}
	if err := rep.Normalize(); err != nil {
		return nil, collectDiagnostics(diag), err
	}
	return toCoverageReport(rep), collectDiagnostics(diag), nil
}

// toTestReport converts the internal results model into the public model.
func toTestReport(rep *results.Report) *TestReport {
	out := &TestReport{Name: rep.Name}
	tot := rep.Totals()
	out.Totals = Totals{
		Tests:         tot.Tests,
		Failures:      tot.Failures,
		Errors:        tot.Errors,
		Skipped:       tot.Skipped,
		DurationNanos: int64(tot.Duration),
	}
	out.Suites = make([]TestSuite, 0, len(rep.Suites))
	for i := range rep.Suites {
		s := &rep.Suites[i]
		ps := TestSuite{
			Name:      s.Name,
			Timestamp: s.Timestamp,
			SystemOut: s.SystemOut,
			SystemErr: s.SystemErr,
		}
		for _, p := range s.Properties {
			ps.Properties = append(ps.Properties, Property{Name: p.Name, Value: p.Value})
		}
		ps.Cases = make([]TestCase, 0, len(s.Cases))
		for _, c := range s.Cases {
			pc := TestCase{
				Name:          c.Name,
				Classname:     c.Classname,
				Status:        Status(c.Status),
				DurationNanos: int64(c.Duration),
				SkipMessage:   c.SkipMessage,
				SystemOut:     c.SystemOut,
				SystemErr:     c.SystemErr,
				File:          c.File,
				Line:          c.Line,
			}
			if c.Failure != nil {
				pc.Failure = &TestFailure{
					Message: c.Failure.Message,
					Type:    c.Failure.Type,
					Details: c.Failure.Details,
				}
			}
			ps.Cases = append(ps.Cases, pc)
		}
		out.Suites = append(out.Suites, ps)
	}
	return out
}

// toCoverageReport converts the internal coverage model into the public model.
func toCoverageReport(rep *coverage.Report) *CoverageReport {
	out := &CoverageReport{
		Producer:          rep.Producer,
		InterchangeFormat: rep.InterchangeFormat,
		SourceRoots:       append([]string(nil), rep.SourceRoots...),
	}
	out.Files = make([]CoverageFile, 0, len(rep.Files))
	for i := range rep.Files {
		f := &rep.Files[i]
		pf := CoverageFile{Path: f.Path}
		if len(f.Metrics) > 0 {
			pf.Metrics = make(map[Metric]Count, len(f.Metrics))
			for m, c := range f.Metrics {
				pf.Metrics[Metric(m)] = Count{
					Covered:        c.Covered,
					Total:          c.Total,
					Derived:        c.Derived,
					DerivationRule: c.DerivationRule,
				}
			}
		}
		for _, fn := range f.Functions {
			pf.Functions = append(pf.Functions, CoverageFunction{Name: fn.Name, Line: fn.Line, Hits: fn.Hits})
		}
		for _, l := range f.Lines {
			pf.Lines = append(pf.Lines, LineHit{
				Number:          l.Number,
				Hits:            l.Hits,
				Branch:          l.Branch,
				BranchesCovered: l.BranchesCovered,
				BranchesTotal:   l.BranchesTotal,
			})
		}
		out.Files = append(out.Files, pf)
	}
	return out
}
