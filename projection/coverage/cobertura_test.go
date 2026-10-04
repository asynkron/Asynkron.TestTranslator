package coverage

import (
	"strings"
	"testing"
)

const testCobertura = `<?xml version="1.0" encoding="utf-8"?>
<coverage>
  <sources><source>.</source></sources>
  <packages><package name="Project">
    <classes>
      <class name="Service" filename="GVCore\Service.cs">
        <methods>
          <method name="Covered"><lines><line number="10" hits="2" /></lines></method>
          <method name="Missed"><lines><line number="20" hits="0" /></lines></method>
        </methods>
        <lines>
          <line number="10" hits="2" branch="true" condition-coverage="50% (1/2)" />
          <line number="20" hits="0" />
        </lines>
      </class>
    </classes>
  </package></packages>
</coverage>`

func TestParseCoberturaProducesCoverageV1(t *testing.T) {
	report, err := ParseCobertura(strings.NewReader(testCobertura), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if report.SchemaVersion != "coverage.v1" || report.SourceFormat != FormatCobertura || len(report.Files) != 1 {
		t.Fatalf("report = %+v", report)
	}
	file := report.Files[0]
	if file.Path != "GVCore/Service.cs" || file.Language != "csharp" ||
		file.LinesTotal != 2 || file.LinesCovered != 1 || file.LineCoverage != 0.5 {
		t.Fatalf("file = %+v", file)
	}
	if file.BranchesTotal == nil || *file.BranchesTotal != 2 || *file.BranchesCovered != 1 || *file.BranchCoverage != 0.5 {
		t.Fatalf("branches = %+v", file)
	}
	if file.FunctionsTotal == nil || *file.FunctionsTotal != 2 || *file.FunctionsCovered != 1 || len(file.Symbols) != 2 {
		t.Fatalf("functions = %+v", file)
	}
}

func TestParseCoberturaDisclosesOutsideRepositoryPath(t *testing.T) {
	xml := strings.Replace(testCobertura, `GVCore\Service.cs`, `..\outside\Service.cs`, 1)
	report, err := ParseCobertura(strings.NewReader(xml), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Files) != 0 || len(report.UnavailableReasons) != 1 {
		t.Fatalf("report = %+v", report)
	}
}

func TestParseCoberturaMergesClassesWithoutInflatingFileTotals(t *testing.T) {
	input := `<coverage><packages><package><classes>
<class filename="Service.cs"><methods><method name="Shared"><lines><line number="10" hits="0"/></lines></method></methods>
<lines><line number="10" hits="0" branch="true" condition-coverage="0% (0/2)"/></lines></class>
<class filename="Service.cs"><methods><method name="Shared"><lines><line number="10" hits="1"/></lines></method>
<method name="Missed"><lines><line number="20" hits="0"/></lines></method></methods>
<lines><line number="10" hits="1" branch="true" condition-coverage="50% (1/2)"/><line number="20" hits="0"/></lines></class>
<class filename="../outside.cs"><lines><line number="1" hits="1"/></lines></class>
</classes></package></packages></coverage>`
	report, err := ParseCobertura(strings.NewReader(input), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Files) != 1 || len(report.UnavailableReasons) != 1 {
		t.Fatalf("safe file or rejected-path evidence lost: %+v", report)
	}
	file := report.Files[0]
	if file.LinesTotal != 2 || file.LinesCovered != 1 ||
		file.BranchesTotal == nil || *file.BranchesTotal != 2 || *file.BranchesCovered != 1 ||
		file.FunctionsTotal == nil || *file.FunctionsTotal != 2 || *file.FunctionsCovered != 1 {
		t.Fatalf("merged file totals = %+v", file)
	}
	if len(file.Symbols) != 2 || file.Symbols[0].Name != "Missed" || file.Symbols[0].Covered ||
		file.Symbols[1].Name != "Shared" || !file.Symbols[1].Covered {
		t.Fatalf("merged methods = %+v", file.Symbols)
	}
}
