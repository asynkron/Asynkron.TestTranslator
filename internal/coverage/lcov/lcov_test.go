package lcov

import (
	"strings"
	"testing"

	"github.com/asynkron/Asynkron.TestTranslator/internal/coverage"
	"github.com/asynkron/Asynkron.TestTranslator/internal/diagnostics"
	"github.com/asynkron/Asynkron.TestTranslator/internal/pathutil"
)

// validTracefile exercises line, branch and function directives across two
// source-file records, one terminated with end_of_record and one relying on the
// trailing-record flush.
const validTracefile = `TN:unit
SF:src/calc.js
FN:1,add
FN:5,sub
FNDA:3,add
FNDA:0,sub
FNF:2
FNH:1
DA:1,3
DA:2,3
DA:5,0
DA:6,0
BRDA:2,0,0,1
BRDA:2,0,1,-
LF:4
LH:2
BRF:2
BRH:1
end_of_record
SF:src/util.js
DA:10,7
DA:10,2
end_of_record
`

func parse(t *testing.T, input string) (*coverage.Report, error) {
	t.Helper()
	opts := coverage.Options{
		SourceName: "lcov.info",
		Paths:      pathutil.New("", ""),
		Diag:       diagnostics.NewCollector(),
	}
	return Adapter{}.Parse(strings.NewReader(input), opts)
}

func TestParseValid(t *testing.T) {
	report, err := parse(t, validTracefile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Producer != "lcov" || report.InterchangeFormat != "lcov-info" {
		t.Fatalf("unexpected report identity: %+v", report)
	}
	if len(report.Files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(report.Files))
	}

	// Normalize sorts files by path; calc.js sorts before util.js.
	calc := report.Files[0]
	if calc.Path != "src/calc.js" {
		t.Fatalf("expected src/calc.js first, got %q", calc.Path)
	}

	// Line metric uses the explicit LF/LH counters.
	lines := calc.Metrics[coverage.MetricLines]
	if lines.Covered != 2 || lines.Total != 4 {
		t.Fatalf("line metric = %+v, want covered=2 total=4", lines)
	}
	if lines.Derived {
		t.Fatalf("LCOV line metric must be native, not derived")
	}

	// Function metric uses explicit FNF/FNH, and functions are preserved.
	fns := calc.Metrics[coverage.MetricFunctions]
	if fns.Covered != 1 || fns.Total != 2 {
		t.Fatalf("function metric = %+v, want covered=1 total=2", fns)
	}
	if len(calc.Functions) != 2 {
		t.Fatalf("expected 2 functions, got %d", len(calc.Functions))
	}

	// Branch metric uses explicit BRF/BRH.
	br := calc.Metrics[coverage.MetricBranches]
	if br.Covered != 1 || br.Total != 2 {
		t.Fatalf("branch metric = %+v, want covered=1 total=2", br)
	}

	// The branch line carries aggregated per-line branch data.
	var found bool
	for _, l := range calc.Lines {
		if l.Number == 2 {
			found = true
			if !l.Branch || l.BranchesTotal != 2 || l.BranchesCovered != 1 {
				t.Fatalf("line 2 branch data = %+v, want total=2 covered=1", l)
			}
		}
	}
	if !found {
		t.Fatalf("line 2 missing from calc.js")
	}

	// util.js had duplicate DA on line 10; the higher hit count wins.
	util := report.Files[1]
	if util.Path != "src/util.js" {
		t.Fatalf("expected src/util.js, got %q", util.Path)
	}
	if len(util.Lines) != 1 || util.Lines[0].Number != 10 || util.Lines[0].Hits != 7 {
		t.Fatalf("util.js line aggregation = %+v, want line 10 hits 7", util.Lines)
	}
}

// lcov >= 2.0 / geninfo emits three-field FN records (start,end,name). The name
// must be the third field, and FNDA must still match it (regression: the name
// used to absorb the end line, "10,add", so FNDA created a spurious second
// function).
func TestParseFNThreeField(t *testing.T) {
	input := "SF:src/math.go\nFN:1,10,add\nFNDA:3,add\nDA:1,3\nDA:2,3\nLF:2\nLH:2\nend_of_record\n"
	report, err := parse(t, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	f := report.Files[0]
	if len(f.Functions) != 1 {
		t.Fatalf("expected exactly 1 function, got %d: %+v", len(f.Functions), f.Functions)
	}
	fn := f.Functions[0]
	if fn.Name != "add" {
		t.Errorf("function name = %q, want %q", fn.Name, "add")
	}
	if fn.Line != 1 {
		t.Errorf("function line = %d, want 1 (start line)", fn.Line)
	}
	if fn.Hits != 3 {
		t.Errorf("function hits = %d, want 3 (FNDA matched)", fn.Hits)
	}
}

// A legacy two-field function name that legitimately contains a comma (e.g. a
// C++ template) must be preserved intact.
func TestParseFNNameWithComma(t *testing.T) {
	input := "SF:src/a.cpp\nFN:5,20,std::pair<int, int>\nFNDA:1,std::pair<int, int>\nDA:5,1\nLF:1\nLH:1\nend_of_record\n"
	report, err := parse(t, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	f := report.Files[0]
	if len(f.Functions) != 1 {
		t.Fatalf("expected 1 function, got %d: %+v", len(f.Functions), f.Functions)
	}
	if f.Functions[0].Name != "std::pair<int, int>" {
		t.Errorf("function name = %q, want the full templated name", f.Functions[0].Name)
	}
}

func TestParseNoSFRecord(t *testing.T) {
	// Text with no SF record is not a valid tracefile.
	_, err := parse(t, "TN:only\nsome noise\nVER:1\n")
	if err == nil {
		t.Fatalf("expected error for input without SF records")
	}
	if !strings.Contains(err.Error(), "no SF") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseMalformedDA(t *testing.T) {
	input := "SF:src/a.js\nDA:notanumber,1\nend_of_record\n"
	_, err := parse(t, input)
	if err == nil {
		t.Fatalf("expected error for malformed DA directive")
	}
	if !strings.Contains(err.Error(), "DA line number") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseNilNormalizer(t *testing.T) {
	// When no normalizer is supplied the path is cleaned in place.
	opts := coverage.Options{Diag: diagnostics.NewCollector()}
	report, err := Adapter{}.Parse(strings.NewReader("SF:./src/a.js\nDA:1,1\nend_of_record\n"), opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Files[0].Path != "src/a.js" {
		t.Fatalf("path = %q, want src/a.js", report.Files[0].Path)
	}
}

func TestParseInputLimit(t *testing.T) {
	opts := coverage.Options{
		Diag:          diagnostics.NewCollector(),
		MaxInputBytes: 8,
	}
	_, err := Adapter{}.Parse(strings.NewReader(validTracefile), opts)
	if err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("expected input-limit error, got %v", err)
	}
}
