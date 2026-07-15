package xunit

import (
	"strings"
	"testing"
	"time"

	"github.com/asynkron/testtranslator/internal/diagnostics"
	"github.com/asynkron/testtranslator/internal/results"
)

// validV2 is a representative xUnit.net v2 document with the <assemblies>
// wrapper and one pass, one fail, and one skip.
const validV2 = `<?xml version="1.0" encoding="utf-8"?>
<assemblies timestamp="2024-01-02 03:04:05">
  <assembly name="MyTests.dll" total="3" passed="1" failed="1" skipped="1" time="0.123">
    <collection name="Test collection for MyTests.Math" total="3" passed="1" failed="1" skipped="1" time="0.123">
      <test name="MyTests.MathTests.Adds" type="MyTests.MathTests" method="Adds" time="0.010" result="Pass">
        <output><![CDATA[hello from adds]]></output>
      </test>
      <test name="MyTests.MathTests.Divides" type="MyTests.MathTests" method="Divides" time="0.020" result="Fail">
        <failure exception-type="Xunit.Sdk.EqualException">
          <message><![CDATA[Assert.Equal() Failure
Expected: 2
Actual:   3]]></message>
          <stack-trace><![CDATA[   at MyTests.MathTests.Divides() in Math.cs:line 42]]></stack-trace>
        </failure>
      </test>
      <test name="MyTests.MathTests.Pending" type="MyTests.MathTests" method="Pending" time="0" result="Skip">
        <reason><![CDATA[not implemented yet]]></reason>
      </test>
    </collection>
  </assembly>
</assemblies>`

// validV3BareAssembly uses a single <assembly> root without the wrapper.
const validV3BareAssembly = `<assembly name="V3.dll" total="1" passed="1" time="0.5">
  <collection name="col" total="1" passed="1" time="0.5">
    <test name="V3.Foo.Bar" type="V3.Foo" method="Bar" time="0.5" result="Pass"/>
  </collection>
</assembly>`

func TestParseValidV2(t *testing.T) {
	rep, err := Adapter{}.Parse(strings.NewReader(validV2), results.Options{
		SourceName: "test.xml",
		Diag:       diagnostics.NewCollector(),
	})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if len(rep.Suites) != 1 {
		t.Fatalf("want 1 suite, got %d", len(rep.Suites))
	}
	suite := rep.Suites[0]
	if want := "MyTests.dll / Test collection for MyTests.Math"; suite.Name != want {
		t.Errorf("suite name = %q, want %q", suite.Name, want)
	}
	if len(suite.Cases) != 3 {
		t.Fatalf("want 3 cases, got %d", len(suite.Cases))
	}

	pass := suite.Cases[0]
	if pass.Name != "MyTests.MathTests.Adds" || pass.Status != results.StatusPassed {
		t.Errorf("pass case unexpected: %+v", pass)
	}
	if pass.Classname != "MyTests.MathTests" {
		t.Errorf("pass classname = %q", pass.Classname)
	}
	if pass.Duration != 10*time.Millisecond {
		t.Errorf("pass duration = %v, want 10ms", pass.Duration)
	}
	if pass.SystemOut != "hello from adds" {
		t.Errorf("pass system out = %q", pass.SystemOut)
	}

	fail := suite.Cases[1]
	if fail.Status != results.StatusFailed {
		t.Errorf("fail status = %v", fail.Status)
	}
	if fail.Failure == nil {
		t.Fatalf("fail case has nil Failure")
	}
	if fail.Failure.Type != "Xunit.Sdk.EqualException" {
		t.Errorf("failure type = %q", fail.Failure.Type)
	}
	if fail.Failure.Message != "Assert.Equal() Failure" {
		t.Errorf("failure message = %q", fail.Failure.Message)
	}
	if !strings.Contains(fail.Failure.Details, "Math.cs:line 42") {
		t.Errorf("failure details missing stack: %q", fail.Failure.Details)
	}

	skip := suite.Cases[2]
	if skip.Status != results.StatusSkipped {
		t.Errorf("skip status = %v", skip.Status)
	}
	if skip.Failure != nil {
		t.Errorf("skip case must have nil Failure")
	}
	if skip.SkipMessage != "not implemented yet" {
		t.Errorf("skip message = %q", skip.SkipMessage)
	}
}

func TestParseBareAssemblyRoot(t *testing.T) {
	rep, err := Adapter{}.Parse(strings.NewReader(validV3BareAssembly), results.Options{
		Diag: diagnostics.NewCollector(),
	})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if len(rep.Suites) != 1 || len(rep.Suites[0].Cases) != 1 {
		t.Fatalf("unexpected report shape: %+v", rep)
	}
	if rep.Suites[0].Name != "V3.dll / col" {
		t.Errorf("suite name = %q", rep.Suites[0].Name)
	}
	if rep.Suites[0].Cases[0].Status != results.StatusPassed {
		t.Errorf("status = %v", rep.Suites[0].Cases[0].Status)
	}
}

func TestParseWrongRoot(t *testing.T) {
	_, err := Adapter{}.Parse(strings.NewReader(`<testsuite name="x"></testsuite>`), results.Options{
		Diag: diagnostics.NewCollector(),
	})
	if err == nil {
		t.Fatal("expected error for wrong root element, got nil")
	}
	if !strings.Contains(err.Error(), "root element") {
		t.Errorf("error = %v, want mention of root element", err)
	}
}

func TestParseMalformedXML(t *testing.T) {
	_, err := Adapter{}.Parse(strings.NewReader(`<assemblies><assembly name="x">`), results.Options{
		Diag: diagnostics.NewCollector(),
	})
	if err == nil {
		t.Fatal("expected error for malformed XML, got nil")
	}
}

func TestParseUnknownResult(t *testing.T) {
	doc := `<assembly name="a.dll"><collection name="c">` +
		`<test name="T.M" type="T" method="M" time="0" result="Bogus"/></collection></assembly>`
	_, err := Adapter{}.Parse(strings.NewReader(doc), results.Options{
		Diag: diagnostics.NewCollector(),
	})
	if err == nil {
		t.Fatal("expected error for unknown result value, got nil")
	}
	if !strings.Contains(err.Error(), "unknown result") {
		t.Errorf("error = %v, want mention of unknown result", err)
	}
}

func TestParseRejectsCustomEntity(t *testing.T) {
	doc := `<!DOCTYPE assemblies [<!ENTITY x "boom">]>` +
		`<assembly name="&x;"><collection name="c">` +
		`<test name="T.M" type="T" method="M" time="0" result="Pass"/></collection></assembly>`
	_, err := Adapter{}.Parse(strings.NewReader(doc), results.Options{
		Diag: diagnostics.NewCollector(),
	})
	if err == nil {
		t.Fatal("expected error for custom entity, got nil")
	}
}
