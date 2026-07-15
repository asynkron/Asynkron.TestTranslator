package coberturain

import (
	"strings"
	"testing"

	"github.com/asynkron/testtranslator/internal/coverage"
	"github.com/asynkron/testtranslator/internal/diagnostics"
)

const validCobertura = `<?xml version="1.0"?>
<!DOCTYPE coverage SYSTEM "http://cobertura.sourceforge.net/xml/coverage-04.dtd">
<coverage line-rate="0.75" branch-rate="0.5" lines-covered="3" lines-valid="4" version="1.9">
  <sources>
    <source>/repo/root</source>
  </sources>
  <packages>
    <package name="pkg">
      <classes>
        <class name="foo" filename="pkg/foo.py" line-rate="0.75" branch-rate="0.5">
          <methods>
            <method name="do_it" signature="()" line-rate="1.0">
              <lines>
                <line number="10" hits="3" branch="false"/>
              </lines>
            </method>
          </methods>
          <lines>
            <line number="10" hits="3" branch="false"/>
            <line number="12" hits="1" branch="true" condition-coverage="50% (1/2)">
              <conditions>
                <condition number="0" type="jump" coverage="50%"/>
              </conditions>
            </line>
            <line number="14" hits="0" branch="false"/>
          </lines>
        </class>
      </classes>
    </package>
  </packages>
</coverage>`

func TestParseValid(t *testing.T) {
	opts := coverage.Options{
		SourceName: "coverage.xml",
		Diag:       diagnostics.NewCollector(),
	}
	rep, err := Adapter{}.Parse(strings.NewReader(validCobertura), opts)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	if rep.Producer != "cobertura" {
		t.Errorf("Producer = %q, want cobertura", rep.Producer)
	}
	if rep.InterchangeFormat != "cobertura-xml" {
		t.Errorf("InterchangeFormat = %q, want cobertura-xml", rep.InterchangeFormat)
	}
	if len(rep.SourceRoots) != 1 || rep.SourceRoots[0] != "/repo/root" {
		t.Errorf("SourceRoots = %v, want [/repo/root]", rep.SourceRoots)
	}
	if len(rep.Files) != 1 {
		t.Fatalf("Files len = %d, want 1", len(rep.Files))
	}

	f := rep.Files[0]
	if f.Path != "pkg/foo.py" {
		t.Errorf("Path = %q, want pkg/foo.py", f.Path)
	}
	if len(f.Lines) != 3 {
		t.Fatalf("Lines len = %d, want 3", len(f.Lines))
	}

	// Locate the branch line (12) and assert branch totals parsed from
	// condition-coverage "50% (1/2)".
	var branchLine *coverage.LineHit
	for i := range f.Lines {
		if f.Lines[i].Number == 12 {
			branchLine = &f.Lines[i]
		}
	}
	if branchLine == nil {
		t.Fatalf("line 12 not found in %+v", f.Lines)
	}
	if !branchLine.Branch {
		t.Errorf("line 12 Branch = false, want true")
	}
	if branchLine.BranchesCovered != 1 || branchLine.BranchesTotal != 2 {
		t.Errorf("line 12 branches = %d/%d, want 1/2", branchLine.BranchesCovered, branchLine.BranchesTotal)
	}

	if len(f.Functions) != 1 {
		t.Fatalf("Functions len = %d, want 1", len(f.Functions))
	}
	fn := f.Functions[0]
	if fn.Name != "do_it()" {
		t.Errorf("Function.Name = %q, want do_it()", fn.Name)
	}
	if fn.Line != 10 {
		t.Errorf("Function.Line = %d, want 10", fn.Line)
	}
	if fn.Hits <= 0 {
		t.Errorf("Function.Hits = %d, want > 0", fn.Hits)
	}
}

func TestParseWrongRoot(t *testing.T) {
	const notCobertura = `<?xml version="1.0"?><report><foo/></report>`
	opts := coverage.Options{SourceName: "x", Diag: diagnostics.NewCollector()}
	_, err := Adapter{}.Parse(strings.NewReader(notCobertura), opts)
	if err == nil {
		t.Fatal("expected error for non-<coverage> root, got nil")
	}
	if !strings.Contains(err.Error(), "root element") {
		t.Errorf("error = %v, want mention of root element", err)
	}
}

func TestParseMalformedXML(t *testing.T) {
	const broken = `<coverage><packages><package></coverage>`
	opts := coverage.Options{SourceName: "x", Diag: diagnostics.NewCollector()}
	_, err := Adapter{}.Parse(strings.NewReader(broken), opts)
	if err == nil {
		t.Fatal("expected error for malformed XML, got nil")
	}
}

func TestParseCustomEntityRejected(t *testing.T) {
	const withEntity = `<?xml version="1.0"?>
<!DOCTYPE coverage [<!ENTITY x "boom">]>
<coverage><packages><package><classes>
<class filename="f.py"><lines><line number="1" hits="&x;"/></lines></class>
</classes></package></packages></coverage>`
	opts := coverage.Options{SourceName: "x", Diag: diagnostics.NewCollector()}
	_, err := Adapter{}.Parse(strings.NewReader(withEntity), opts)
	if err == nil {
		t.Fatal("expected error for custom entity, got nil")
	}
}

func TestParseEmptyClasses(t *testing.T) {
	const empty = `<coverage><packages></packages></coverage>`
	opts := coverage.Options{SourceName: "x", Diag: diagnostics.NewCollector()}
	_, err := Adapter{}.Parse(strings.NewReader(empty), opts)
	if err == nil {
		t.Fatal("expected error for report with no classes, got nil")
	}
}

func TestParseConditionCoverage(t *testing.T) {
	cases := []struct {
		in           string
		covered, tot int
		ok           bool
	}{
		{"50% (1/2)", 1, 2, true},
		{"100% (3/3)", 3, 3, true},
		{"0% (0/4)", 0, 4, true},
		{"nonsense", 0, 0, false},
		{"50%", 0, 0, false},
		{"(2/1)", 0, 0, false}, // covered > total
	}
	for _, c := range cases {
		got, tot, ok := parseConditionCoverage(c.in)
		if ok != c.ok || (ok && (got != c.covered || tot != c.tot)) {
			t.Errorf("parseConditionCoverage(%q) = (%d,%d,%v), want (%d,%d,%v)",
				c.in, got, tot, ok, c.covered, c.tot, c.ok)
		}
	}
}
