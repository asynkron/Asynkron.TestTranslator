package surefire

import (
	"strings"
	"testing"

	"github.com/asynkron/Asynkron.TestTranslator/internal/diagnostics"
	"github.com/asynkron/Asynkron.TestTranslator/internal/results"
)

const suites = `<?xml version="1.0" encoding="UTF-8"?>
<testsuites>
  <testsuite name="com.example.FooTest" tests="3" failures="1" skipped="1" timestamp="2023-01-02T03:04:05">
    <properties><property name="java.version" value="17"/></properties>
    <testcase name="testPass" classname="com.example.FooTest" time="0.010"/>
    <testcase name="testFail" classname="com.example.FooTest" time="0.020">
      <failure message="expected 1 got 2" type="AssertionError">stack trace here</failure>
      <system-out>hello</system-out>
    </testcase>
    <testcase name="testSkip" classname="com.example.FooTest" time="0">
      <skipped message="not ready"/>
    </testcase>
  </testsuite>
</testsuites>`

func TestParseSuites(t *testing.T) {
	diag := diagnostics.NewCollector()
	rep, err := Adapter{}.Parse(strings.NewReader(suites), results.Options{Diag: diag})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	tot := rep.Totals()
	if tot.Tests != 3 || tot.Failures != 1 || tot.Skipped != 1 {
		t.Fatalf("unexpected totals: %+v", tot)
	}
	s := rep.Suites[0]
	if len(s.Properties) != 1 || s.Properties[0].Value != "17" {
		t.Errorf("properties not preserved: %+v", s.Properties)
	}
	if s.Timestamp.IsZero() {
		t.Error("timestamp not parsed")
	}
	var fail *results.TestCase
	for i := range s.Cases {
		if s.Cases[i].Name == "testFail" {
			fail = &s.Cases[i]
		}
	}
	if fail == nil || fail.Failure == nil || fail.Failure.Type != "AssertionError" {
		t.Fatalf("failure detail not preserved: %+v", fail)
	}
	if fail.SystemOut != "hello" {
		t.Errorf("system-out not captured: %q", fail.SystemOut)
	}
}

const single = `<testsuite name="s" tests="1"><testcase name="a" time="0"/></testsuite>`

func TestParseSingleSuiteRoot(t *testing.T) {
	rep, err := Adapter{}.Parse(strings.NewReader(single), results.Options{Diag: diagnostics.NewCollector()})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(rep.Suites) != 1 || len(rep.Suites[0].Cases) != 1 {
		t.Fatalf("unexpected parse: %+v", rep.Suites)
	}
}

func TestParseWrongRootFails(t *testing.T) {
	if _, err := (Adapter{}).Parse(strings.NewReader(`<coverage/>`), results.Options{Diag: diagnostics.NewCollector()}); err == nil {
		t.Fatal("expected error for wrong root element")
	}
}
