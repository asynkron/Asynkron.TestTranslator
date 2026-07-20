// Package results is the repository-owned native-test translator used by
// scripts/quality-evidence.mjs. It maps native Go and JUnit reports into the
// runner-neutral test-results.v1 payload before Faktorial reads the evidence.
//
// The emitted payload is validated downstream by
// qualityevidence.validateTestPayload, whose invariants are:
//
//	total == passed + failed + skipped
//	failed == len(failures) + omitted_failures
//
// so every failed/error case MUST yield exactly one failure record.
package results

import (
	"fmt"
	testtranslator "github.com/asynkron/Asynkron.TestTranslator"
	"io"
)

// testtranslator format ids understood by Parse for the two native producers
// Faktorial emits: the backend go-test-json stream and per-workspace Vitest
// JUnit XML reports.
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

// Failure mirrors one entry of the test-results.v1 "failures" array. The field
// names and omitempty rules match qualityevidence's payload decoder
// (testresults.Failure): test is always present, the rest are optional.
type Failure struct {
	Suite   string `json:"suite,omitempty"`
	File    string `json:"file,omitempty"`
	Test    string `json:"test"`
	Message string `json:"message,omitempty"`
}

// Payload is the test-results.v1 {counts,failures} envelope body. "failures" is
// always rendered (as [] when empty) to match the JS producer.
type Payload struct {
	Counts   Counts    `json:"counts"`
	Failures []Failure `json:"failures"`
}

// Parse reads a report of the given format (e.g. "go-test-json") from r via
// testtranslator and maps it into the test-results.v1 payload. Diagnostics from
// the translator are intentionally NOT leaked into the payload — parser status
// and source stay 'quality-evidence'/'ok'/'unparsed' downstream.
func Parse(format string, r io.Reader) (*Payload, error) {
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
// counts.total   = Totals.Tests
// counts.skipped = Totals.Skipped
// counts.failed  = Totals.Failures + Totals.Errors (errors folded into failed)
// counts.passed  = total - failed - skipped
//
// failures[] holds one record for every case whose status is failed or error,
// so counts.failed stays equal to len(failures) as validateTestPayload demands.
func Map(report *testtranslator.TestReport) *Payload {
	totals := report.Totals
	counts := Counts{
		Total:   totals.Tests,
		Skipped: totals.Skipped,
		Failed:  totals.Failures + totals.Errors,
	}
	counts.Passed = counts.Total - counts.Failed - counts.Skipped

	failures := make([]Failure, 0, counts.Failed)
	for _, suite := range report.Suites {
		for _, c := range suite.Cases {
			if c.Status != testtranslator.StatusFailed && c.Status != testtranslator.StatusError {
				continue
			}
			failures = append(failures, Failure{
				Suite:   suite.Name,
				File:    c.File,
				Test:    c.Name,
				Message: failureMessage(c.Failure),
			})
		}
	}

	return &Payload{Counts: counts, Failures: failures}
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
