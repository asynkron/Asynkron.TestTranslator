// Package results defines the internal test-result model and the adapter
// contract that source parsers implement. JUnit XML is the public output
// contract; this model is an implementation detail shared by every adapter and
// the writers.
package results

import (
	"fmt"
	"sort"
	"time"
)

// Status is the outcome of a single test case. The set is closed so writers can
// map every value deterministically.
type Status string

const (
	// StatusPassed marks a test that ran and succeeded.
	StatusPassed Status = "passed"
	// StatusFailed marks a test that ran and failed an assertion.
	StatusFailed Status = "failed"
	// StatusError marks a test that errored (panic, unexpected runtime error).
	StatusError Status = "error"
	// StatusSkipped marks a test that was skipped.
	StatusSkipped Status = "skipped"
)

// valid reports whether s is a known status.
func (s Status) valid() bool {
	switch s {
	case StatusPassed, StatusFailed, StatusError, StatusSkipped:
		return true
	default:
		return false
	}
}

// Failure describes why a test failed or errored.
type Failure struct {
	// Message is the short failure summary (JUnit message attribute).
	Message string
	// Type is an optional failure classification (JUnit type attribute).
	Type string
	// Details is the full failure body (element text), e.g. a stack trace.
	Details string
}

// TestCase is a single executed or discovered test.
type TestCase struct {
	// Name is the test case name, unique within its suite after ordering.
	Name string
	// Classname is the JUnit classname (typically package or class).
	Classname string
	// Status is the outcome.
	Status Status
	// Duration is the wall-clock execution time. Negative durations are
	// rejected at construction.
	Duration time.Duration
	// Failure is set when Status is failed or error.
	Failure *Failure
	// SkipMessage is the reason recorded when Status is skipped.
	SkipMessage string
	// SystemOut is captured stdout for this case.
	SystemOut string
	// SystemErr is captured stderr for this case.
	SystemErr string
	// File is the repo-relative source file, when known.
	File string
	// Line is the source line, when known (0 means unknown).
	Line int
}

// TestSuite groups related test cases.
type TestSuite struct {
	// Name identifies the suite (typically the package or assembly).
	Name string
	// Cases are the test cases in the suite.
	Cases []TestCase
	// Timestamp is the suite start time, when known.
	Timestamp time.Time
	// SystemOut is suite-level captured stdout.
	SystemOut string
	// SystemErr is suite-level captured stderr.
	SystemErr string
	// Properties are ordered key/value metadata pairs.
	Properties []Property
}

// Property is a suite-level metadata key/value pair.
type Property struct {
	Name  string
	Value string
}

// Report is the complete converted test-result document.
type Report struct {
	// Name is an optional top-level testsuites name.
	Name string
	// Suites are the ordered test suites.
	Suites []TestSuite
}

// AddCase appends a validated test case to the suite, rejecting contradictory
// states so a Report can never hold an impossible case.
func (ts *TestSuite) AddCase(tc TestCase) error {
	if tc.Name == "" {
		return fmt.Errorf("test case in suite %q has an empty name", ts.Name)
	}
	if !tc.Status.valid() {
		return fmt.Errorf("test case %q has invalid status %q", tc.Name, tc.Status)
	}
	if tc.Duration < 0 {
		return fmt.Errorf("test case %q has negative duration", tc.Name)
	}
	switch tc.Status {
	case StatusFailed, StatusError:
		if tc.Failure == nil {
			return fmt.Errorf("test case %q has status %q but no failure detail", tc.Name, tc.Status)
		}
	case StatusPassed, StatusSkipped:
		if tc.Failure != nil {
			return fmt.Errorf("test case %q has status %q but carries failure detail", tc.Name, tc.Status)
		}
	}
	if tc.Line < 0 {
		return fmt.Errorf("test case %q has negative line number", tc.Name)
	}
	ts.Cases = append(ts.Cases, tc)
	return nil
}

// Totals summarizes a report.
type Totals struct {
	Tests    int
	Failures int
	Errors   int
	Skipped  int
	Duration time.Duration
}

// Totals computes aggregate counts across all suites.
func (r *Report) Totals() Totals {
	var t Totals
	for _, s := range r.Suites {
		for _, c := range s.Cases {
			t.Tests++
			t.Duration += c.Duration
			switch c.Status {
			case StatusFailed:
				t.Failures++
			case StatusError:
				t.Errors++
			case StatusSkipped:
				t.Skipped++
			}
		}
	}
	return t
}

// SuiteTotals summarizes a single suite.
func (ts *TestSuite) SuiteTotals() Totals {
	var t Totals
	for _, c := range ts.Cases {
		t.Tests++
		t.Duration += c.Duration
		switch c.Status {
		case StatusFailed:
			t.Failures++
		case StatusError:
			t.Errors++
		case StatusSkipped:
			t.Skipped++
		}
	}
	return t
}

// Sort applies deterministic ordering: suites by name, then cases by
// (classname, name). Ordering is stable so equal keys preserve input order,
// which matters for interleaved event streams already ordered by the adapter.
func (r *Report) Sort() {
	sort.SliceStable(r.Suites, func(i, j int) bool {
		return r.Suites[i].Name < r.Suites[j].Name
	})
	for si := range r.Suites {
		cases := r.Suites[si].Cases
		sort.SliceStable(cases, func(i, j int) bool {
			if cases[i].Classname != cases[j].Classname {
				return cases[i].Classname < cases[j].Classname
			}
			return cases[i].Name < cases[j].Name
		})
	}
}
