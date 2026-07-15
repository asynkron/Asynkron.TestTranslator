package nunit

import (
	"strings"
	"testing"

	"github.com/asynkron/Asynkron.TestTranslator/internal/diagnostics"
	"github.com/asynkron/Asynkron.TestTranslator/internal/results"
)

const nunit3 = `<?xml version="1.0" encoding="utf-8"?>
<test-run id="1" testcasecount="3" result="Failed">
  <test-suite type="Assembly" name="MyTests.dll" fullname="MyTests.dll">
    <test-suite type="TestFixture" name="FooFixture" fullname="Ns.FooFixture">
      <test-case name="Passes" fullname="Ns.FooFixture.Passes" classname="Ns.FooFixture" result="Passed" duration="0.012"/>
      <test-case name="Fails" fullname="Ns.FooFixture.Fails" classname="Ns.FooFixture" result="Failed" duration="0.004">
        <failure><message>Expected 1 but was 2</message><stack-trace>at Ns.FooFixture.Fails()</stack-trace></failure>
      </test-case>
      <test-case name="Ignored" fullname="Ns.FooFixture.Ignored" classname="Ns.FooFixture" result="Skipped">
        <reason><message>not ready</message></reason>
      </test-case>
    </test-suite>
  </test-suite>
</test-run>`

func TestParseNUnit3(t *testing.T) {
	rep, err := Adapter{}.Parse(strings.NewReader(nunit3), results.Options{Diag: diagnostics.NewCollector()})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	tot := rep.Totals()
	if tot.Tests != 3 || tot.Failures != 1 || tot.Skipped != 1 {
		t.Fatalf("unexpected totals: %+v", tot)
	}
	var fail *results.TestCase
	for si := range rep.Suites {
		for ci := range rep.Suites[si].Cases {
			if rep.Suites[si].Cases[ci].Name == "Fails" {
				fail = &rep.Suites[si].Cases[ci]
			}
		}
	}
	if fail == nil || fail.Failure == nil {
		t.Fatal("failing case not found")
	}
	if !strings.Contains(fail.Failure.Details, "Expected 1") {
		t.Errorf("failure details lost: %+v", fail.Failure)
	}
	if fail.Classname != "Ns.FooFixture" {
		t.Errorf("classname wrong: %q", fail.Classname)
	}
}

const nunit2 = `<?xml version="1.0" encoding="utf-8"?>
<test-results name="MyTests" total="2" failures="1">
  <test-suite name="FooFixture">
    <results>
      <test-case name="Ns.FooFixture.Passes" executed="True" success="True" time="0.010" result="Success"/>
      <test-case name="Ns.FooFixture.Fails" executed="True" success="False" time="0.020" result="Failure">
        <failure><message>bad</message><stack-trace>trace</stack-trace></failure>
      </test-case>
    </results>
  </test-suite>
</test-results>`

func TestParseNUnit2(t *testing.T) {
	rep, err := Adapter{}.Parse(strings.NewReader(nunit2), results.Options{Diag: diagnostics.NewCollector()})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	tot := rep.Totals()
	if tot.Tests != 2 || tot.Failures != 1 {
		t.Fatalf("unexpected totals: %+v", tot)
	}
}

func TestWrongRootFails(t *testing.T) {
	if _, err := (Adapter{}).Parse(strings.NewReader(`<foo/>`), results.Options{Diag: diagnostics.NewCollector()}); err == nil {
		t.Fatal("expected error for wrong root")
	}
}
