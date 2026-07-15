package junit

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/asynkron/testtranslator/internal/results"
)

func sampleReport(t *testing.T) *results.Report {
	t.Helper()
	suite := results.TestSuite{Name: "pkg/foo"}
	must := func(err error) {
		if err != nil {
			t.Fatalf("AddCase: %v", err)
		}
	}
	must(suite.AddCase(results.TestCase{Name: "TestA", Classname: "pkg/foo", Status: results.StatusPassed, Duration: 10 * time.Millisecond}))
	must(suite.AddCase(results.TestCase{Name: "TestB", Classname: "pkg/foo", Status: results.StatusFailed,
		Failure: &results.Failure{Message: "boom", Type: "failure", Details: "stack"}}))
	must(suite.AddCase(results.TestCase{Name: "TestC", Classname: "pkg/foo", Status: results.StatusSkipped, SkipMessage: "later"}))
	must(suite.AddCase(results.TestCase{Name: "TestD", Classname: "pkg/foo", Status: results.StatusError,
		Failure: &results.Failure{Message: "panic", Type: "error", Details: "goroutine"}}))
	return &results.Report{Name: "sample", Suites: []results.TestSuite{suite}}
}

func TestMarshalCountsAndValidate(t *testing.T) {
	r := sampleReport(t)
	r.Sort()
	out, err := Marshal(r)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	s := string(out)
	if !strings.HasPrefix(s, `<?xml version="1.0" encoding="UTF-8"?>`) {
		t.Fatalf("missing XML declaration:\n%s", s)
	}
	for _, want := range []string{
		`tests="4"`, `failures="1"`, `errors="1"`, `skipped="1"`,
		`<failure message="boom"`, `<error message="panic"`, `<skipped message="later">`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("output missing %q\n%s", want, s)
		}
	}
	if err := Validate(bytes.NewReader(out)); err != nil {
		t.Fatalf("Validate rejected our own output: %v", err)
	}
}

func TestMarshalDeterministic(t *testing.T) {
	r1 := sampleReport(t)
	r1.Sort()
	a, _ := Marshal(r1)
	r2 := sampleReport(t)
	r2.Sort()
	b, _ := Marshal(r2)
	if !bytes.Equal(a, b) {
		t.Fatal("Marshal is not deterministic across identical reports")
	}
}

func TestSanitizeInvalidXMLChars(t *testing.T) {
	suite := results.TestSuite{Name: "s"}
	if err := suite.AddCase(results.TestCase{Name: "T\x00\x01ext", Classname: "c", Status: results.StatusPassed}); err != nil {
		t.Fatal(err)
	}
	r := &results.Report{Suites: []results.TestSuite{suite}}
	out, err := Marshal(r)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if bytes.ContainsAny(out, "\x00\x01") {
		t.Fatal("output contains illegal control characters")
	}
	if err := Validate(bytes.NewReader(out)); err != nil {
		t.Fatalf("sanitized output failed validation: %v", err)
	}
}

func TestValidateRejectsCountMismatch(t *testing.T) {
	bad := `<?xml version="1.0"?><testsuites tests="5"><testsuite name="s" tests="1"><testcase name="a" time="0"/></testsuite></testsuites>`
	if err := Validate(strings.NewReader(bad)); err == nil {
		t.Fatal("expected validation error for count mismatch")
	}
}

func TestValidateRejectsWrongRoot(t *testing.T) {
	if err := Validate(strings.NewReader(`<coverage/>`)); err == nil {
		t.Fatal("expected validation error for wrong root element")
	}
}
