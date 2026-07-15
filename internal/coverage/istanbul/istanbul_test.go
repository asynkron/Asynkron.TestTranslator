package istanbul

import (
	"strings"
	"testing"

	"github.com/asynkron/Asynkron.TestTranslator/internal/coverage"
	"github.com/asynkron/Asynkron.TestTranslator/internal/diagnostics"
	"github.com/asynkron/Asynkron.TestTranslator/internal/pathutil"
)

// validFixture is a minimal but representative coverage-final.json with two
// statements on the same line (to exercise the max-over-statements rule), a
// branch, and a function.
const validFixture = `{
  "src/foo.ts": {
    "path": "src/foo.ts",
    "statementMap": {
      "0": {"start": {"line": 1, "column": 0}, "end": {"line": 1, "column": 10}},
      "1": {"start": {"line": 1, "column": 11}, "end": {"line": 1, "column": 20}},
      "2": {"start": {"line": 3, "column": 0}, "end": {"line": 3, "column": 5}}
    },
    "s": {"0": 0, "1": 3, "2": 0},
    "branchMap": {
      "0": {"type": "if", "loc": {"start": {"line": 3, "column": 0}, "end": {"line": 3, "column": 5}}, "locations": [{"start": {"line": 3}}, {"start": {"line": 4}}]}
    },
    "b": {"0": [1, 0]},
    "fnMap": {
      "0": {"name": "foo", "decl": {"start": {"line": 1}}, "loc": {"start": {"line": 1}}}
    },
    "f": {"0": 2}
  }
}`

func parse(t *testing.T, input string) (*coverage.Report, *diagnostics.Collector) {
	t.Helper()
	diag := diagnostics.NewCollector()
	opts := coverage.Options{
		SourceName: "coverage-final.json",
		Paths:      pathutil.New("", ""),
		Diag:       diag,
	}
	report, err := Adapter{}.Parse(strings.NewReader(input), opts)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	return report, diag
}

func TestParseValid(t *testing.T) {
	report, _ := parse(t, validFixture)

	if report.Producer != "istanbul" {
		t.Errorf("Producer = %q, want istanbul", report.Producer)
	}
	if report.InterchangeFormat != "istanbul-json" {
		t.Errorf("InterchangeFormat = %q, want istanbul-json", report.InterchangeFormat)
	}
	if len(report.Files) != 1 {
		t.Fatalf("len(Files) = %d, want 1", len(report.Files))
	}
	f := report.Files[0]
	if f.Path != "src/foo.ts" {
		t.Errorf("Path = %q, want src/foo.ts", f.Path)
	}

	// Line 1 has two statements (hits 0 and 3); max => 3.
	var line1, line3 *coverage.LineHit
	for i := range f.Lines {
		switch f.Lines[i].Number {
		case 1:
			line1 = &f.Lines[i]
		case 3:
			line3 = &f.Lines[i]
		}
	}
	if line1 == nil {
		t.Fatal("no LineHit for line 1")
	}
	if line1.Hits != 3 {
		t.Errorf("line 1 Hits = %d, want 3 (max over statements)", line1.Hits)
	}
	if line3 == nil {
		t.Fatal("no LineHit for line 3")
	}
	if !line3.Branch || line3.BranchesTotal != 2 || line3.BranchesCovered != 1 {
		t.Errorf("line 3 branch data = {%v cov=%d tot=%d}, want {true cov=1 tot=2}",
			line3.Branch, line3.BranchesCovered, line3.BranchesTotal)
	}

	// Statement metric: 1 of 3 covered.
	if c := f.Metrics[coverage.MetricStatements]; c.Covered != 1 || c.Total != 3 {
		t.Errorf("statements = %+v, want covered=1 total=3", c)
	}
	// Branch metric: 1 of 2 covered.
	if c := f.Metrics[coverage.MetricBranches]; c.Covered != 1 || c.Total != 2 {
		t.Errorf("branches = %+v, want covered=1 total=2", c)
	}
	// Function metric: 1 of 1 covered.
	if c := f.Metrics[coverage.MetricFunctions]; c.Covered != 1 || c.Total != 1 {
		t.Errorf("functions = %+v, want covered=1 total=1", c)
	}
	// Line metric must be marked derived.
	lc := f.Metrics[coverage.MetricLines]
	if !lc.Derived {
		t.Errorf("lines metric should be Derived")
	}

	if len(f.Functions) != 1 || f.Functions[0].Name != "foo" || f.Functions[0].Hits != 2 {
		t.Errorf("Functions = %+v, want one foo with hits=2", f.Functions)
	}
}

func TestParseNilNormalizer(t *testing.T) {
	diag := diagnostics.NewCollector()
	opts := coverage.Options{SourceName: "x", Diag: diag} // Paths nil
	report, err := Adapter{}.Parse(strings.NewReader(validFixture), opts)
	if err != nil {
		t.Fatalf("Parse with nil normalizer: %v", err)
	}
	if report.Files[0].Path != "src/foo.ts" {
		t.Errorf("Path = %q, want src/foo.ts", report.Files[0].Path)
	}
}

func TestParseMalformed(t *testing.T) {
	cases := map[string]string{
		"top-level array":      `[{"path":"a.ts"}]`,
		"scalar":               `42`,
		"empty":                ``,
		"missing statementMap": `{"src/foo.ts": {"path": "src/foo.ts", "s": {}}}`,
		"invalid json":         `{"src/foo.ts": {`,
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			diag := diagnostics.NewCollector()
			opts := coverage.Options{SourceName: "x", Paths: pathutil.New("", ""), Diag: diag}
			a := Adapter{}
			if _, err := a.Parse(strings.NewReader(input), opts); err == nil {
				t.Errorf("expected error for %s, got nil", name)
			}
		})
	}
}
