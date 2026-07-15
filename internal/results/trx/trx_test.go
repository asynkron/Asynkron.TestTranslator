package trx

import (
	"strings"
	"testing"

	"github.com/asynkron/Asynkron.TestTranslator/internal/diagnostics"
	"github.com/asynkron/Asynkron.TestTranslator/internal/results"
)

const doc = `<?xml version="1.0" encoding="UTF-8"?>
<TestRun name="run 1" xmlns="http://microsoft.com/schemas/VisualStudio/TeamTest/2010">
  <Results>
    <UnitTestResult testName="Ns.FooTests.Passes" testId="a" outcome="Passed" duration="00:00:00.0120000"/>
    <UnitTestResult testName="Ns.FooTests.Fails" testId="b" outcome="Failed" duration="00:00:00.0050000">
      <Output><ErrorInfo><Message>Assert.AreEqual failed</Message><StackTrace>at Foo</StackTrace></ErrorInfo></Output>
    </UnitTestResult>
    <UnitTestResult testName="Ns.FooTests.Skips" testId="c" outcome="NotExecuted"/>
    <UnitTestResult testName="Ns.FooTests.Times" testId="d" outcome="Timeout"/>
  </Results>
  <TestDefinitions>
    <UnitTest name="Passes" id="a"><TestMethod className="Ns.FooTests" name="Passes"/></UnitTest>
    <UnitTest name="Fails" id="b"><TestMethod className="Ns.FooTests" name="Fails"/></UnitTest>
    <UnitTest name="Skips" id="c"><TestMethod className="Ns.FooTests" name="Skips"/></UnitTest>
    <UnitTest name="Times" id="d"><TestMethod className="Ns.FooTests" name="Times"/></UnitTest>
  </TestDefinitions>
</TestRun>`

func TestParseTRX(t *testing.T) {
	rep, err := Adapter{}.Parse(strings.NewReader(doc), results.Options{Diag: diagnostics.NewCollector()})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	tot := rep.Totals()
	if tot.Tests != 4 || tot.Failures != 1 || tot.Errors != 1 || tot.Skipped != 1 {
		t.Fatalf("unexpected totals: %+v", tot)
	}
	s := rep.Suites[0]
	var fail *results.TestCase
	for i := range s.Cases {
		if s.Cases[i].Name == "Ns.FooTests.Fails" {
			fail = &s.Cases[i]
		}
	}
	if fail == nil || fail.Classname != "Ns.FooTests" {
		t.Fatalf("classname not resolved from TestDefinitions: %+v", fail)
	}
	if fail.Duration <= 0 {
		t.Error("duration not parsed from HH:MM:SS.fff")
	}
	if !strings.Contains(fail.Failure.Details, "Assert.AreEqual") {
		t.Errorf("error info discarded: %+v", fail.Failure)
	}
}

func TestWrongRootFails(t *testing.T) {
	if _, err := (Adapter{}).Parse(strings.NewReader(`<foo/>`), results.Options{Diag: diagnostics.NewCollector()}); err == nil {
		t.Fatal("expected error for wrong root")
	}
}
