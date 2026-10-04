// Package results projects native test reports into runner-neutral counts,
// failure diagnostics, and case observations.
package results

import (
	"fmt"
	"io"

	testtranslator "github.com/asynkron/Asynkron.TestTranslator"
)

// Native format identifiers for Go event streams and JUnit XML reports.
const (
	FormatGoTestJSON = "go-test-json"
	FormatJUnitXML   = "junit-xml"
)

// Counts mirrors the "counts" object of the test-results.v1 payload. Fields are
// always emitted (no omitempty) so the shape matches the JS producer exactly.
type Counts struct {
	Total   int `json:"total"`
	Passed  int `json:"passed"`
	Failed  int `json:"failed"`
	Skipped int `json:"skipped"`
}

// Failure is one entry of the test-results.v1 diagnostics array. Test is
// always present; suite, file, and message are optional.
type Failure struct {
	Suite   string `json:"suite,omitempty"`
	File    string `json:"file,omitempty"`
	Test    string `json:"test"`
	Message string `json:"message,omitempty"`
}

// Payload is the runner-neutral test-results.v1 envelope. Failures and
// observations are always emitted as arrays, including when empty.
type Payload struct {
	Counts       Counts        `json:"counts"`
	Failures     []Failure     `json:"failures"`
	Observations []Observation `json:"observations"`
}

// Observation preserves a native test case without host-specific identity,
// redaction, or storage policy. A nil duration means no positive measurement
// was established by the source model.
type Observation struct {
	Suite         string `json:"suite,omitempty"`
	Class         string `json:"class,omitempty"`
	File          string `json:"file,omitempty"`
	Test          string `json:"test"`
	Status        string `json:"status"`
	DurationNanos *int64 `json:"duration_nanos"`
	Message       string `json:"message,omitempty"`
}

// Parse reads a report of the given format (e.g. "go-test-json") from r via
// testtranslator and maps it into test-results.v1. Parser diagnostics remain
// separate from test failures. Go lifecycle events exclude tests still paused at
// the end of the stream and preserve bounded compiler output for build failures.
func Parse(format string, r io.Reader) (*Payload, error) {
	if format == FormatGoTestJSON {
		return parseGoTestJSON(r)
	}
	return parseTranslatedResults(format, r)
}

func parseTranslatedResults(format string, r io.Reader) (*Payload, error) {
	report, _, err := testtranslator.ParseResults(format, r)
	if err != nil {
		return nil, err
	}
	if report == nil {
		return nil, fmt.Errorf("testtranslator returned a nil report for format %q", format)
	}
	return Map(report), nil
}

// Map converts a *testtranslator.TestReport into the test-results.v1 payload.
//
// Counts and failures are both derived from Suites/Cases rather than trusting
// the separately stored Totals roll-up. This keeps the payload invariants intact
// even when a caller supplies a manually constructed or mutated public report.
//
// Failures holds one record for every case counted as failed, including an
// invalid public status, so Counts.Failed stays equal to len(Failures).
func Map(report *testtranslator.TestReport) *Payload {
	if report == nil {
		return &Payload{Failures: []Failure{}, Observations: []Observation{}}
	}

	counts := Counts{}
	failures := make([]Failure, 0)
	observations := make([]Observation, 0)
	for _, suite := range report.Suites {
		for _, c := range suite.Cases {
			observation := Observation{Suite: suite.Name, Class: c.Classname, File: c.File, Test: c.Name, Status: string(c.Status)}
			if c.DurationNanos > 0 {
				duration := c.DurationNanos
				observation.DurationNanos = &duration
			}
			if c.Failure != nil {
				observation.Message = failureMessage(c.Failure)
			} else if c.Status == testtranslator.StatusSkipped {
				observation.Message = c.SkipMessage
			}
			observations = append(observations, observation)
			counts.Total++
			switch c.Status {
			case testtranslator.StatusPassed:
				counts.Passed++
			case testtranslator.StatusSkipped:
				counts.Skipped++
			case testtranslator.StatusFailed, testtranslator.StatusError:
				counts.Failed++
				failures = append(failures, Failure{
					Suite:   suite.Name,
					File:    c.File,
					Test:    c.Name,
					Message: failureMessage(c.Failure),
				})
			default:
				// Public TestReport values can be constructed without going
				// through the validated parser. Preserve both payload invariants
				// and surface the invalid outcome as a failure.
				counts.Failed++
				failures = append(failures, Failure{
					Suite:   suite.Name,
					File:    c.File,
					Test:    c.Name,
					Message: fmt.Sprintf("unknown test status %q", c.Status),
				})
			}
		}
	}

	return &Payload{Counts: counts, Failures: failures, Observations: observations}
}

// failureMessage prefers the richer Details blob and falls back to Message,
// mirroring the mapping contract. A nil Failure yields an empty string (a
// failed/error case still produces a record; only its message is empty).
func failureMessage(f *testtranslator.TestFailure) string {
	if f == nil {
		return ""
	}
	if f.Details != "" {
		return f.Details
	}
	return f.Message
}
