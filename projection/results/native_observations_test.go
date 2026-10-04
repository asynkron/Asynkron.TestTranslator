package results

import (
	"encoding/json"
	"io"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestParseExcludesPausedTestsAndPreservesCompletedResults(t *testing.T) {
	input := strings.NewReader(strings.Join([]string{
		`{"Action":"run","Package":"example/pkg","Test":"TestPaused"}`,
		`{"Action":"pause","Package":"example/pkg","Test":"TestPaused"}`,
		`{"Action":"run","Package":"example/pkg","Test":"TestResumed"}`,
		`{"Action":"pause","Package":"example/pkg","Test":"TestResumed"}`,
		`{"Action":"cont","Package":"example/pkg","Test":"TestResumed"}`,
		`{"Action":"run","Package":"example/pkg","Test":"TestPass"}`,
		`{"Action":"pass","Package":"example/pkg","Test":"TestPass"}`,
		`{"Action":"run","Package":"example/pkg","Test":"TestFail"}`,
		`{"Action":"fail","Package":"example/pkg","Test":"TestFail"}`,
		`{"Action":"run","Package":"example/pkg","Test":"TestFailedThenPaused"}`,
		`{"Action":"fail","Package":"example/pkg","Test":"TestFailedThenPaused"}`,
		`{"Action":"pause","Package":"example/pkg","Test":"TestFailedThenPaused"}`,
		`{"Action":"run","Package":"example/pkg","Test":"TestSkip"}`,
		`{"Action":"skip","Package":"example/pkg","Test":"TestSkip"}`,
		`{"Action":"run","Package":"example/pkg","Test":"TestParent"}`,
		`{"Action":"run","Package":"example/pkg","Test":"TestParent/child"}`,
		`{"Action":"pass","Package":"example/pkg","Test":"TestParent/child"}`,
		`{"Action":"pass","Package":"example/pkg","Test":"TestParent"}`,
		`{"Action":"fail","Package":"example/pkg"}`,
		`{"Action":"output","Package":"example/build","Output":"compile failed\n"}`,
		`{"Action":"fail","Package":"example/build"}`,
	}, "\n"))

	payload := parsePayload(t, "go-test-json", input)
	if payload.Counts != (Counts{Total: 8, Passed: 3, Failed: 4, Skipped: 1}) {
		t.Fatalf("counts = %+v", payload.Counts)
	}
	if len(payload.Observations) != 8 {
		t.Fatalf("observation population = %d, want 8", len(payload.Observations))
	}
	for _, o := range payload.Observations {
		if o.Test == "TestPaused" {
			t.Fatal("paused-only test persisted as executed")
		}
	}
	wantFailures := []Failure{
		{Suite: "example/build", Test: "build", Message: "compile failed"},
		{Suite: "example/pkg", Test: "TestFail", Message: "test failed"},
		{Suite: "example/pkg", Test: "TestFailedThenPaused", Message: "test failed"},
		{Suite: "example/pkg", Test: "TestResumed", Message: "test did not complete (crash or timeout)"},
	}
	if !slices.Equal(payload.Failures, wantFailures) {
		t.Fatalf("failures = %+v, want %+v", payload.Failures, wantFailures)
	}
}

func TestParsePreservesCompilerDiagnosticForBuildFailure(t *testing.T) {
	input := strings.NewReader(strings.Join([]string{
		`{"ImportPath":"example/pkg","Action":"build-output","Output":"example.go:7:2: undefined: MissingSymbol\n"}`,
		`{"ImportPath":"example/pkg","Action":"build-fail"}`,
		`{"Action":"fail","Package":"example/pkg","FailedBuild":"example/pkg"}`,
	}, "\n"))

	payload := parsePayload(t, "go-test-json", input)
	if payload.Counts != (Counts{Total: 1, Failed: 1}) {
		t.Fatalf("counts = %+v", payload.Counts)
	}
	wantFailures := []Failure{{
		Suite:   "example/pkg",
		Test:    "build",
		Message: "example.go:7:2: undefined: MissingSymbol",
	}}
	if !slices.Equal(payload.Failures, wantFailures) {
		t.Fatalf("failures = %+v, want %+v", payload.Failures, wantFailures)
	}
}

func TestParseCapsCompilerDiagnosticAtUTF8Boundary(t *testing.T) {
	tests := []struct {
		name    string
		outputs []string
		want    string
	}{
		{
			name:    "single event ends with split rune",
			outputs: []string{strings.Repeat("x", maxGoBuildOutputBytes-1) + "å"},
			want:    strings.Repeat("x", maxGoBuildOutputBytes-1),
		},
		{
			name: "later event crosses boundary",
			outputs: []string{
				strings.Repeat("x", maxGoBuildOutputBytes-2),
				"aå",
			},
			want: strings.Repeat("x", maxGoBuildOutputBytes-2) + "a",
		},
		{
			name:    "complete rune exactly fills boundary",
			outputs: []string{strings.Repeat("x", maxGoBuildOutputBytes-2) + "å"},
			want:    strings.Repeat("x", maxGoBuildOutputBytes-2) + "å",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var input strings.Builder
			for _, output := range test.outputs {
				writeGoTestEvent(t, &input, goTestEvent{
					ImportPath: "example/pkg",
					Action:     "build-output",
					Output:     output,
				})
			}
			writeGoTestEvent(t, &input, goTestEvent{ImportPath: "example/pkg", Action: "build-fail"})
			writeGoTestEvent(t, &input, goTestEvent{
				Package:     "example/pkg",
				Action:      "fail",
				FailedBuild: "example/pkg",
			})

			payload := parsePayload(t, "go-test-json", strings.NewReader(input.String()))
			if len(payload.Failures) != 1 {
				t.Fatalf("failures = %+v, want one build failure", payload.Failures)
			}
			message := payload.Failures[0].Message
			if message != test.want {
				t.Fatalf("message = %q, want %q", message, test.want)
			}
			if !utf8.ValidString(message) {
				t.Fatalf("message is not valid UTF-8: %q", message)
			}
			if len(message) > maxGoBuildOutputBytes {
				t.Fatalf("message bytes = %d, want at most %d", len(message), maxGoBuildOutputBytes)
			}
		})
	}
}

func writeGoTestEvent(t *testing.T, output io.Writer, event goTestEvent) {
	t.Helper()
	if err := json.NewEncoder(output).Encode(event); err != nil {
		t.Fatalf("encode Go test event: %v", err)
	}
}

func TestParsePreservesJUnitTestDurationsAndClassIdentity(t *testing.T) {
	input := strings.NewReader(`<testsuite name="frontend" tests="4" failures="1" skipped="1">
 <testcase name="shared" classname="ClassA" time="0.25"/>
 <testcase name="shared" classname="ClassB" time="1.5"><failure message="expected true">actual false</failure></testcase>
 <testcase name="zero" time="0"/>
 <testcase name="skipped"><skipped message="unsupported platform"/></testcase>
 </testsuite>`)
	payload := parsePayload(t, "junit-xml", input)
	if payload.Counts != (Counts{Total: 4, Passed: 2, Failed: 1, Skipped: 1}) {
		t.Fatalf("counts = %+v", payload.Counts)
	}
	if len(payload.Observations) != 4 {
		t.Fatalf("lost cases: %+v", payload.Observations)
	}
	var a, b, zero, skipped Observation
	for _, o := range payload.Observations {
		switch {
		case o.Class == "ClassA":
			a = o
		case o.Class == "ClassB":
			b = o
		case o.Test == "zero":
			zero = o
		case o.Test == "skipped":
			skipped = o
		}
	}
	if a.Class != "ClassA" || a.Status != "passed" || a.DurationNanos == nil || *a.DurationNanos != 250_000_000 ||
		b.Class != "ClassB" || b.Status != "failed" || b.DurationNanos == nil || *b.DurationNanos != 1_500_000_000 || b.Message != "actual false" {
		t.Fatalf("durations/identity lost: %+v", payload.Observations)
	}
	if zero.DurationNanos != nil || skipped.DurationNanos != nil || skipped.Status != "skipped" || skipped.Message != "unsupported platform" {
		t.Fatalf("unavailable timing/skip lost: %+v", payload.Observations)
	}
}

func TestParsePreservesGoSubtestDuration(t *testing.T) {
	input := strings.NewReader(strings.Join([]string{
		`{"Action":"run","Package":"example/pkg","Test":"TestParent/child"}`,
		`{"Action":"pass","Package":"example/pkg","Test":"TestParent/child","Elapsed":0.125}`,
		`{"Action":"pass","Package":"example/pkg"}`,
	}, "\n"))
	payload := parsePayload(t, "go-test-json", input)
	if len(payload.Observations) != 1 {
		t.Fatalf("observations = %+v", payload.Observations)
	}
	o := payload.Observations[0]
	if o.Test != "TestParent/child" || o.Suite != "example/pkg" || o.Status != "passed" || o.DurationNanos == nil || *o.DurationNanos != 125_000_000 {
		t.Fatalf("observation = %+v", o)
	}
}
func parsePayload(t *testing.T, format string, input io.Reader) *Payload {
	t.Helper()
	payload, err := Parse(format, input)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	return payload
}
