package jacoco

import (
	"strings"
	"testing"

	"github.com/asynkron/Asynkron.TestTranslator/internal/coverage"
	"github.com/asynkron/Asynkron.TestTranslator/internal/diagnostics"
	"github.com/asynkron/Asynkron.TestTranslator/internal/pathutil"
)

const validJacoco = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE report PUBLIC "-//JACOCO//DTD Report 1.1//EN" "report.dtd">
<report name="example">
  <sessioninfo id="host-1" start="1700000000000" dump="1700000001000"/>
  <package name="com/example">
    <class name="com/example/Foo" sourcefilename="Foo.java">
      <method name="doIt" desc="()V" line="10">
        <counter type="INSTRUCTION" missed="0" covered="4"/>
        <counter type="LINE" missed="0" covered="2"/>
        <counter type="METHOD" missed="0" covered="1"/>
      </method>
    </class>
    <sourcefile name="Foo.java">
      <line nr="10" mi="0" ci="4" mb="0" cb="0"/>
      <line nr="12" mi="0" ci="6" mb="1" cb="3"/>
      <line nr="14" mi="3" ci="0" mb="2" cb="0"/>
      <counter type="INSTRUCTION" missed="3" covered="10"/>
      <counter type="BRANCH" missed="3" covered="3"/>
      <counter type="LINE" missed="1" covered="2"/>
      <counter type="METHOD" missed="0" covered="1"/>
      <counter type="CLASS" missed="0" covered="1"/>
    </sourcefile>
  </package>
</report>`

func TestParseValid(t *testing.T) {
	opts := coverage.Options{
		SourceName: "jacoco.xml",
		Diag:       diagnostics.NewCollector(),
		Paths:      pathutil.New("", ""),
	}
	rep, err := Adapter{}.Parse(strings.NewReader(validJacoco), opts)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	if rep.Producer != "jacoco" {
		t.Errorf("Producer = %q, want jacoco", rep.Producer)
	}
	if rep.InterchangeFormat != "jacoco-xml" {
		t.Errorf("InterchangeFormat = %q, want jacoco-xml", rep.InterchangeFormat)
	}
	if len(rep.Files) != 1 {
		t.Fatalf("Files len = %d, want 1", len(rep.Files))
	}

	f := rep.Files[0]
	if f.Path != "com/example/Foo.java" {
		t.Errorf("Path = %q, want com/example/Foo.java", f.Path)
	}
	if len(f.Lines) != 3 {
		t.Fatalf("Lines len = %d, want 3", len(f.Lines))
	}

	byNr := map[int]coverage.LineHit{}
	for _, l := range f.Lines {
		byNr[l.Number] = l
	}

	// Line 10: covered instructions, no branches -> Hits=1, not a branch line.
	if l := byNr[10]; l.Hits != 1 || l.Branch {
		t.Errorf("line 10 = %+v, want Hits=1 Branch=false", l)
	}
	// Line 12: covered instructions and branches -> Hits=1, 3/4 branches.
	if l := byNr[12]; l.Hits != 1 || !l.Branch || l.BranchesCovered != 3 || l.BranchesTotal != 4 {
		t.Errorf("line 12 = %+v, want Hits=1 Branch=true 3/4", l)
	}
	// Line 14: no covered instructions -> Hits=0; branches present, 0/2.
	if l := byNr[14]; l.Hits != 0 || !l.Branch || l.BranchesCovered != 0 || l.BranchesTotal != 2 {
		t.Errorf("line 14 = %+v, want Hits=0 Branch=true 0/2", l)
	}

	// Native metrics from the sourcefile-level counters.
	inst := f.Metrics[coverage.MetricInstructions]
	if inst.Covered != 10 || inst.Total != 13 || inst.Derived {
		t.Errorf("instructions metric = %+v, want covered=10 total=13 derived=false", inst)
	}
	br := f.Metrics[coverage.MetricBranches]
	if br.Covered != 3 || br.Total != 6 {
		t.Errorf("branches metric = %+v, want covered=3 total=6", br)
	}
	if _, ok := f.Metrics[coverage.MetricLines]; !ok {
		t.Errorf("expected a lines metric to be set")
	}
	// CLASS counters have no representable metric and must be dropped.
	if _, ok := f.Metrics["class"]; ok {
		t.Errorf("class counter should not produce a metric")
	}
}

func TestParseNilPaths(t *testing.T) {
	opts := coverage.Options{SourceName: "x", Diag: diagnostics.NewCollector()}
	rep, err := Adapter{}.Parse(strings.NewReader(validJacoco), opts)
	if err != nil {
		t.Fatalf("Parse with nil Paths returned error: %v", err)
	}
	if rep.Files[0].Path != "com/example/Foo.java" {
		t.Errorf("Path = %q, want com/example/Foo.java", rep.Files[0].Path)
	}
}

func TestParseWrongRoot(t *testing.T) {
	const notJacoco = `<?xml version="1.0"?><coverage><foo/></coverage>`
	opts := coverage.Options{SourceName: "x", Diag: diagnostics.NewCollector()}
	_, err := Adapter{}.Parse(strings.NewReader(notJacoco), opts)
	if err == nil {
		t.Fatal("expected error for non-<report> root, got nil")
	}
	if !strings.Contains(err.Error(), "root element") {
		t.Errorf("error = %v, want mention of root element", err)
	}
}

func TestParseMalformedXML(t *testing.T) {
	const broken = `<report><package><sourcefile></report>`
	opts := coverage.Options{SourceName: "x", Diag: diagnostics.NewCollector()}
	_, err := Adapter{}.Parse(strings.NewReader(broken), opts)
	if err == nil {
		t.Fatal("expected error for malformed XML, got nil")
	}
}

func TestParseCustomEntityRejected(t *testing.T) {
	const withEntity = `<?xml version="1.0"?>
<!DOCTYPE report [<!ENTITY x "boom">]>
<report name="x"><package name="p"><sourcefile name="F.java">
<line nr="1" ci="&x;" mi="0" cb="0" mb="0"/>
</sourcefile></package></report>`
	opts := coverage.Options{SourceName: "x", Diag: diagnostics.NewCollector()}
	_, err := Adapter{}.Parse(strings.NewReader(withEntity), opts)
	if err == nil {
		t.Fatal("expected error for custom entity, got nil")
	}
}

func TestParseNoSourcefiles(t *testing.T) {
	const empty = `<report name="x"><package name="p"></package></report>`
	opts := coverage.Options{SourceName: "x", Diag: diagnostics.NewCollector()}
	_, err := Adapter{}.Parse(strings.NewReader(empty), opts)
	if err == nil {
		t.Fatal("expected error for report with no sourcefiles, got nil")
	}
}

func TestStripDoctype(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "public doctype",
			in:   `<?xml version="1.0"?>` + "\n" + `<!DOCTYPE report PUBLIC "a" "b">` + "\n" + `<report/>`,
			want: `<?xml version="1.0"?>` + "\n" + "\n" + `<report/>`,
		},
		{
			name: "internal subset with > inside",
			in:   `<!DOCTYPE report [<!ENTITY x "a>b">]><report/>`,
			want: `<report/>`,
		},
		{
			name: "no doctype",
			in:   `<report/>`,
			want: `<report/>`,
		},
	}
	for _, c := range cases {
		if got := string(stripDoctype([]byte(c.in))); got != c.want {
			t.Errorf("%s: stripDoctype = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestMetricFor(t *testing.T) {
	if m, ok := metricFor("instruction"); !ok || m != coverage.MetricInstructions {
		t.Errorf("metricFor(instruction) = %q,%v", m, ok)
	}
	if _, ok := metricFor("COMPLEXITY"); ok {
		t.Errorf("metricFor(COMPLEXITY) should be unmapped")
	}
}
