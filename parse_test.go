package testtranslator_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/asynkron/testtranslator"
)

func TestParseResultsModel(t *testing.T) {
	in := readFixture(t, "go-testjson/go-test-json.out")
	rep, _, err := testtranslator.ParseResults("go-test-json", bytes.NewReader(in))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if rep.Totals.Tests == 0 || len(rep.Suites) == 0 {
		t.Fatalf("empty report: %+v", rep.Totals)
	}
	// The model must expose failures with a message and a closed status set.
	var sawFailure bool
	for _, s := range rep.Suites {
		for _, c := range s.Cases {
			switch c.Status {
			case testtranslator.StatusPassed, testtranslator.StatusFailed,
				testtranslator.StatusError, testtranslator.StatusSkipped:
			default:
				t.Errorf("unexpected status %q for %s", c.Status, c.Name)
			}
			if c.Status == testtranslator.StatusFailed || c.Status == testtranslator.StatusError {
				if c.Failure == nil {
					t.Errorf("failed/errored case %q has no Failure", c.Name)
				} else {
					sawFailure = true
				}
			}
		}
	}
	if !sawFailure {
		t.Errorf("expected at least one failure/error in this fixture")
	}
	// The model must be JSON-serializable (a consumer persists it).
	if _, err := json.Marshal(rep); err != nil {
		t.Errorf("model is not JSON-serializable: %v", err)
	}
}

// The public coverage model must keep Go statement counts distinct from line
// counts (statements native, lines derived) — the whole reason coverage callers
// need the model rather than a rendered document.
func TestParseCoverageGoStatementsVsLines(t *testing.T) {
	in := readFixture(t, "go-coverprofile/pathutil-set.out")
	rep, _, err := testtranslator.ParseCoverage("go-coverprofile", bytes.NewReader(in), testtranslator.CoverageOptions{
		GoModule: "github.com/asynkron/testtranslator",
	})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(rep.Files) == 0 {
		t.Fatal("no files parsed")
	}
	f := rep.Files[0]
	if f.Path == "" {
		t.Error("file path (the join key) is empty")
	}
	stmts, ok := f.Metrics[testtranslator.MetricStatements]
	if !ok {
		t.Fatalf("expected a statements metric, got metrics: %v", f.Metrics)
	}
	if stmts.Derived {
		t.Error("Go statement counts must be native, not derived")
	}
	if lines, ok := f.Metrics[testtranslator.MetricLines]; ok && !lines.Derived {
		t.Error("Go line counts must be marked derived (they are folded from statements)")
	}
}

func TestParseCoverageLCOVBranchesFunctions(t *testing.T) {
	in := readFixture(t, "lcov/lcov-merger-basic-a.info")
	rep, _, err := testtranslator.ParseCoverage("lcov", bytes.NewReader(in), testtranslator.CoverageOptions{RepoRoot: "/"})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(rep.Files) == 0 {
		t.Fatal("no files parsed")
	}
	// LCOV supplies branch and function data; it must survive into the model.
	var sawBranches, sawFunctions bool
	for _, f := range rep.Files {
		if _, ok := f.Metrics[testtranslator.MetricBranches]; ok {
			sawBranches = true
		}
		if _, ok := f.Metrics[testtranslator.MetricFunctions]; ok {
			sawFunctions = true
		}
	}
	if !sawBranches {
		t.Error("expected branch metrics from LCOV")
	}
	if !sawFunctions {
		t.Error("expected function metrics from LCOV")
	}
	if _, err := json.Marshal(rep); err != nil {
		t.Errorf("coverage model is not JSON-serializable: %v", err)
	}
}

func TestParseResultsMismatchFails(t *testing.T) {
	tap := readFixture(t, "tap/tapjs_basic.tap")
	if _, _, err := testtranslator.ParseResults("go-test-json", bytes.NewReader(tap)); err == nil {
		t.Fatalf("expected an error for mismatched input")
	}
}

func TestParseDeterministic(t *testing.T) {
	in := readFixture(t, "lcov/lcov-merger-basic-a.info")
	a, _, err := testtranslator.ParseCoverage("lcov", bytes.NewReader(in), testtranslator.CoverageOptions{RepoRoot: "/"})
	if err != nil {
		t.Fatal(err)
	}
	b, _, err := testtranslator.ParseCoverage("lcov", bytes.NewReader(in), testtranslator.CoverageOptions{RepoRoot: "/"})
	if err != nil {
		t.Fatal(err)
	}
	ja, _ := json.Marshal(a)
	jb, _ := json.Marshal(b)
	if !bytes.Equal(ja, jb) {
		t.Errorf("parsed coverage model is not deterministic")
	}
}
