// Package gotest converts `go test -json` (test2json) event streams into the
// internal result model. It preserves complete subtest paths, captures per-test
// output, represents package build failures as synthetic testcases, and treats
// tests that were started but never finished (crashes/timeouts) as errors.
//
// The event schema is documented at https://pkg.go.dev/cmd/test2json. This is a
// clean-room parser of that public, stable JSON contract rather than a copy of
// gotestsum; the naming and build-failure behavior follow gotestsum's
// conventions.
package gotest

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/asynkron/testtranslator/internal/results"
)

func init() {
	results.Register("go-test-json", "Go `go test -json` / test2json event stream", Adapter{}, "gotest", "go-json")
}

// defaultMaxInput bounds total input bytes to avoid unbounded memory use.
const defaultMaxInput = 512 << 20 // 512 MiB

// maxTokenBytes bounds a single JSON line so a pathological stream cannot
// exhaust memory.
const maxTokenBytes = 8 << 20 // 8 MiB per line

// syntheticPackage groups events that arrive without a Package field.
const syntheticPackage = "(unknown package)"

// event is one test2json record. Fields not needed are omitted.
type event struct {
	Action  string  `json:"Action"`
	Package string  `json:"Package"`
	Test    string  `json:"Test"`
	Elapsed float64 `json:"Elapsed"`
	Output  string  `json:"Output"`
}

// Adapter implements results.Adapter for go test -json.
type Adapter struct{}

// testState accumulates events for a single test within a package.
type testState struct {
	name     string
	status   results.Status
	finished bool
	elapsed  time.Duration
	output   strings.Builder
}

// pkgState accumulates events for a single package.
type pkgState struct {
	name        string
	tests       map[string]*testState
	testOrder   []string
	output      strings.Builder
	finished    bool
	failed      bool
	elapsed     time.Duration
	hadTestPass bool
}

// Parse reads the event stream and produces one suite per package.
func (Adapter) Parse(r io.Reader, opts results.Options) (*results.Report, error) {
	limit := opts.MaxInputBytes
	if limit <= 0 {
		limit = defaultMaxInput
	}
	lr := &io.LimitedReader{R: r, N: limit + 1}
	sc := bufio.NewScanner(lr)
	sc.Buffer(make([]byte, 0, 64*1024), maxTokenBytes)

	pkgs := map[string]*pkgState{}
	var pkgOrder []string
	getPkg := func(name string) *pkgState {
		p, ok := pkgs[name]
		if !ok {
			p = &pkgState{name: name, tests: map[string]*testState{}}
			pkgs[name] = p
			pkgOrder = append(pkgOrder, name)
		}
		return p
	}

	warnedNoPkg := false
	line := 0
	for sc.Scan() {
		line++
		raw := strings.TrimSpace(sc.Text())
		if raw == "" {
			continue
		}
		if lr.N <= 0 {
			return nil, fmt.Errorf("gotest: input exceeds %d byte limit", limit)
		}
		var ev event
		if err := json.Unmarshal([]byte(raw), &ev); err != nil {
			return nil, fmt.Errorf("gotest: line %d is not a valid test2json event: %w", line, err)
		}
		pkgName := ev.Package
		if pkgName == "" {
			// The go toolchain always emits a Package, but some producers emit
			// framework-level events without one. Group them under a synthetic
			// package rather than discarding the data.
			pkgName = syntheticPackage
			if !warnedNoPkg {
				opts.Diag.Warnf("gotest.nopackage", opts.SourceName, "one or more events had no Package field; grouped under %q", syntheticPackage)
				warnedNoPkg = true
			}
		}
		p := getPkg(pkgName)

		if ev.Test == "" {
			// Package-level event.
			switch ev.Action {
			case "output":
				p.output.WriteString(ev.Output)
			case "pass", "skip":
				p.finished = true
				p.elapsed = secs(ev.Elapsed)
			case "fail":
				p.finished = true
				p.failed = true
				p.elapsed = secs(ev.Elapsed)
			}
			continue
		}

		// Test-level event.
		ts, ok := p.tests[ev.Test]
		if !ok {
			ts = &testState{name: ev.Test, status: results.StatusPassed}
			p.tests[ev.Test] = ts
			p.testOrder = append(p.testOrder, ev.Test)
		}
		switch ev.Action {
		case "run", "cont", "pause", "start", "bench":
			// State transitions with no terminal effect.
		case "output":
			ts.output.WriteString(ev.Output)
		case "pass":
			ts.status = results.StatusPassed
			ts.finished = true
			ts.elapsed = secs(ev.Elapsed)
			p.hadTestPass = true
		case "fail":
			ts.status = results.StatusFailed
			ts.finished = true
			ts.elapsed = secs(ev.Elapsed)
		case "skip":
			ts.status = results.StatusSkipped
			ts.finished = true
			ts.elapsed = secs(ev.Elapsed)
		}
	}
	if err := sc.Err(); err != nil {
		if err == bufio.ErrTooLong {
			return nil, fmt.Errorf("gotest: a JSON line exceeds the %d byte limit", maxTokenBytes)
		}
		return nil, fmt.Errorf("gotest: read error: %w", err)
	}
	if lr.N <= 0 {
		return nil, fmt.Errorf("gotest: input exceeds %d byte limit", limit)
	}

	report := &results.Report{Name: "go test"}
	for _, pkgName := range pkgOrder {
		p := pkgs[pkgName]
		suite := results.TestSuite{Name: pkgName}

		if len(p.testOrder) == 0 {
			// No tests: either a build/discovery failure or a package with no
			// tests. Represent a failing empty package as a synthetic testcase
			// so the failure is never silently discarded.
			if p.failed {
				detail := strings.TrimSpace(p.output.String())
				msg := "package failed to build or run"
				if detail != "" {
					msg = firstLine(detail)
				}
				tc := results.TestCase{
					Name:      "build",
					Classname: pkgName,
					Status:    results.StatusError,
					Failure:   &results.Failure{Message: msg, Type: "build", Details: detail},
					SystemErr: detail,
				}
				if err := suite.AddCase(tc); err != nil {
					return nil, err
				}
				opts.Diag.Warnf("gotest.buildfailure", opts.SourceName, "package %q failed with no tests; emitted synthetic build testcase", pkgName)
			} else {
				opts.Diag.Notef("gotest.notests", opts.SourceName, "package %q reported no tests", pkgName)
			}
			report.Suites = append(report.Suites, suite)
			continue
		}

		for _, tn := range p.testOrder {
			ts := p.tests[tn]
			tc := results.TestCase{
				Name:      ts.name,
				Classname: pkgName,
				Duration:  ts.elapsed,
				SystemOut: strings.TrimRight(ts.output.String(), "\n"),
			}
			switch {
			case !ts.finished:
				// Started but no terminal event: a crash or timeout took the
				// process down. Report it as an error, not a silent pass.
				tc.Status = results.StatusError
				out := strings.TrimSpace(ts.output.String())
				tc.Failure = &results.Failure{
					Message: "test did not complete (crash or timeout)",
					Type:    "incomplete",
					Details: out,
				}
				opts.Diag.Warnf("gotest.incomplete", opts.SourceName, "test %q in %q never finished; treated as error", ts.name, pkgName)
			case ts.status == results.StatusFailed:
				tc.Status = results.StatusFailed
				out := strings.TrimSpace(ts.output.String())
				tc.Failure = &results.Failure{
					Message: firstFailureLine(out),
					Type:    "failure",
					Details: out,
				}
			case ts.status == results.StatusSkipped:
				tc.Status = results.StatusSkipped
				tc.SkipMessage = skipReason(ts.output.String())
			default:
				tc.Status = results.StatusPassed
			}
			if err := suite.AddCase(tc); err != nil {
				return nil, err
			}
		}
		report.Suites = append(report.Suites, suite)
	}
	return report, nil
}

// secs converts a test2json Elapsed (seconds) to a Duration, clamping negatives
// to zero.
func secs(f float64) time.Duration {
	if f < 0 {
		return 0
	}
	return time.Duration(f * float64(time.Second))
}

// firstLine returns the first non-empty line of s, or a placeholder.
func firstLine(s string) string {
	for _, l := range strings.Split(s, "\n") {
		l = strings.TrimSpace(l)
		if l != "" {
			return l
		}
	}
	return ""
}

// skipReason extracts the human-readable skip reason from a skipped test's
// captured output, ignoring test2json framing lines such as "=== RUN" and
// "--- SKIP" so the returned message is the actual reason (typically a
// "foo_test.go:NN: ..." line) rather than the run banner.
func skipReason(s string) string {
	for _, l := range strings.Split(s, "\n") {
		t := strings.TrimSpace(l)
		if t == "" || strings.HasPrefix(t, "===") || strings.HasPrefix(t, "---") {
			continue
		}
		return t
	}
	return ""
}

// firstFailureLine finds the most informative line from failing test output,
// preferring lines that look like assertions or errors.
func firstFailureLine(s string) string {
	lines := strings.Split(s, "\n")
	for _, l := range lines {
		t := strings.TrimSpace(l)
		if strings.HasPrefix(t, "panic:") || strings.Contains(t, "Error:") || strings.Contains(t, "_test.go:") {
			return t
		}
	}
	if fl := firstLine(s); fl != "" {
		return fl
	}
	return "test failed"
}
