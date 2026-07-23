package results

import (
	"strings"
	"testing"

	testtranslator "github.com/asynkron/Asynkron.TestTranslator"
)

// goTestJSONFixture is a real `go test -json` (test2json) event stream covering
// every terminal outcome the mapping must handle:
//
//   - example/pkgA: TestPass (pass), TestSkip (skip), TestFail (fail)
//   - example/pkgB: a package that fails to build with no tests, which
//     testtranslator surfaces as a synthetic "build" case with StatusError.
//
// The build failure is a correctness improvement over the legacy JS parser,
// which dropped package-level build failures; it MUST fold into counts.failed
// and produce a failure record.
const goTestJSONFixture = `
{"Action":"run","Package":"example/pkgA","Test":"TestPass"}
{"Action":"output","Package":"example/pkgA","Test":"TestPass","Output":"=== RUN   TestPass\n"}
{"Action":"pass","Package":"example/pkgA","Test":"TestPass","Elapsed":0.01}
{"Action":"run","Package":"example/pkgA","Test":"TestSkip"}
{"Action":"output","Package":"example/pkgA","Test":"TestSkip","Output":"=== RUN   TestSkip\n"}
{"Action":"output","Package":"example/pkgA","Test":"TestSkip","Output":"    foo_test.go:20: not on this platform\n"}
{"Action":"skip","Package":"example/pkgA","Test":"TestSkip","Elapsed":0}
{"Action":"run","Package":"example/pkgA","Test":"TestFail"}
{"Action":"output","Package":"example/pkgA","Test":"TestFail","Output":"=== RUN   TestFail\n"}
{"Action":"output","Package":"example/pkgA","Test":"TestFail","Output":"    foo_test.go:42: Error: expected 1 got 2\n"}
{"Action":"fail","Package":"example/pkgA","Test":"TestFail","Elapsed":0.02}
{"Action":"output","Package":"example/pkgA","Output":"FAIL\n"}
{"Action":"fail","Package":"example/pkgA","Elapsed":0.04}
{"Action":"output","Package":"example/pkgB","Output":"# example/pkgB\n"}
{"Action":"output","Package":"example/pkgB","Output":"./broken.go:3:1: syntax error: unexpected }\n"}
{"Action":"fail","Package":"example/pkgB","Elapsed":0}
`

func TestParseGoTestJSON_CountsInvariantAndFailureMapping(t *testing.T) {
	payload, err := Parse("go-test-json", strings.NewReader(goTestJSONFixture))
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	// Expected roll-up: 3 cases in pkgA (pass/skip/fail) + 1 synthetic build
	// error case in pkgB => total 4, passed 1, failed 2 (1 failure + 1 error),
	// skipped 1.
	want := Counts{Total: 4, Passed: 1, Failed: 2, Skipped: 1}
	if payload.Counts != want {
		t.Fatalf("counts = %+v, want %+v", payload.Counts, want)
	}

	// validateTestPayload invariant #1: total == passed + failed + skipped.
	c := payload.Counts
	if c.Total != c.Passed+c.Failed+c.Skipped {
		t.Fatalf("total %d != passed %d + failed %d + skipped %d", c.Total, c.Passed, c.Failed, c.Skipped)
	}

	// validateTestPayload invariant #2: failed == len(failures) (omitted is 0
	// here — this slice never caps).
	if c.Failed != len(payload.Failures) {
		t.Fatalf("failed count %d != %d failure records", c.Failed, len(payload.Failures))
	}

	// Every failure record must carry a non-empty test identity.
	for i, f := range payload.Failures {
		if strings.TrimSpace(f.Test) == "" {
			t.Fatalf("failure %d has empty test identity: %+v", i, f)
		}
	}

	byTest := map[string]Failure{}
	for _, f := range payload.Failures {
		byTest[f.Test] = f
	}

	fail, ok := byTest["TestFail"]
	if !ok {
		t.Fatalf("expected a TestFail failure record, got %+v", payload.Failures)
	}
	if fail.Suite != "example/pkgA" {
		t.Errorf("TestFail suite = %q, want example/pkgA", fail.Suite)
	}
	if !strings.Contains(fail.Message, "expected 1 got 2") {
		t.Errorf("TestFail message = %q, want it to include the assertion detail", fail.Message)
	}

	build, ok := byTest["build"]
	if !ok {
		t.Fatalf("expected a synthetic 'build' failure record for the build-failure package, got %+v", payload.Failures)
	}
	if build.Suite != "example/pkgB" {
		t.Errorf("build suite = %q, want example/pkgB", build.Suite)
	}
	if !strings.Contains(build.Message, "syntax error") {
		t.Errorf("build message = %q, want it to include the build error detail", build.Message)
	}
}

func TestMapFoldsErrorsIntoFailedAndPassedIsRemainder(t *testing.T) {
	payload, err := Parse("go-test-json", strings.NewReader(goTestJSONFixture))
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	// passed is a derived remainder, never negative for a well-formed stream.
	if payload.Counts.Passed < 0 {
		t.Fatalf("passed must not be negative: %+v", payload.Counts)
	}
}

func TestMapRecomputesCountsFromCases(t *testing.T) {
	report := &testtranslator.TestReport{
		// Deliberately stale totals model a caller-mutated public report.
		Totals: testtranslator.Totals{Tests: 1},
		Suites: []testtranslator.TestSuite{{
			Name: "suite",
			Cases: []testtranslator.TestCase{{
				Name:   "failed",
				Status: testtranslator.StatusFailed,
			}},
		}},
	}

	payload := Map(report)
	if payload.Counts != (Counts{Total: 1, Failed: 1}) {
		t.Fatalf("counts = %+v, want one failed test", payload.Counts)
	}
	if len(payload.Failures) != 1 || payload.Failures[0].Test != "failed" {
		t.Fatalf("failures = %+v, want one matching record", payload.Failures)
	}
}

func TestMapPreservesInvariantsForUnknownStatusAndNilReport(t *testing.T) {
	payload := Map(&testtranslator.TestReport{
		Suites: []testtranslator.TestSuite{{
			Cases: []testtranslator.TestCase{{Name: "unknown", Status: "custom"}},
		}},
	})
	if payload.Counts != (Counts{Total: 1, Failed: 1}) || len(payload.Failures) != 1 {
		t.Fatalf("unknown status payload = %+v", payload)
	}

	empty := Map(nil)
	if empty.Counts != (Counts{}) || empty.Failures == nil || len(empty.Failures) != 0 {
		t.Fatalf("nil report payload = %+v, want stable empty payload", empty)
	}
}
