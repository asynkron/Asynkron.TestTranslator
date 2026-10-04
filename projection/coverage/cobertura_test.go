package coverage

import (
	"os"
	"reflect"
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

func TestParseCoberturaMethodIdentity(t *testing.T) {
	sharedFile, err := os.ReadFile("testdata/cobertura-shared-file.xml")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name    string
		xml     string
		covered int
		symbols []SymbolCoverage
	}{
		{
			name: "same method in different classes",
			xml:  string(sharedFile), covered: 1,
			symbols: []SymbolCoverage{
				{Name: "A.Run()", Kind: "function", Covered: true},
				{Name: "B.Run()", Kind: "function", Covered: false},
			},
		},
		{
			name: "repeated same method retains covered observation",
			xml:  strings.ReplaceAll(string(sharedFile), `name="B"`, `name="A"`), covered: 1,
			symbols: []SymbolCoverage{{Name: "A.Run()", Kind: "function", Covered: true}},
		},
		{
			name: "repeated same method becomes covered",
			xml: strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(string(sharedFile),
				`name="B"`, `name="A"`), `hits="1"`, `hits="0"`),
				`number="20" hits="0"`, `number="20" hits="1"`), covered: 1,
			symbols: []SymbolCoverage{{Name: "A.Run()", Kind: "function", Covered: true}},
		},
		{
			name: "overloads in same class",
			xml: strings.Replace(strings.ReplaceAll(string(sharedFile), `name="B"`, `name="A"`),
				`signature="()"`, `signature="(int)"`, 1), covered: 1,
			symbols: []SymbolCoverage{
				{Name: "A.Run()", Kind: "function", Covered: false},
				{Name: "A.Run(int)", Kind: "function", Covered: true},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report, err := ParseCobertura(strings.NewReader(tt.xml), t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			if len(report.Files) != 1 || len(report.UnavailableReasons) != 0 {
				t.Fatalf("report = %+v", report)
			}
			file := report.Files[0]
			if file.FunctionsTotal == nil || *file.FunctionsTotal != len(tt.symbols) ||
				file.FunctionsCovered == nil || *file.FunctionsCovered != tt.covered {
				t.Fatalf("functions = %+v", file)
			}
			if !reflect.DeepEqual(file.Symbols, tt.symbols) {
				t.Fatalf("symbols = %+v, want %+v", file.Symbols, tt.symbols)
			}
			if file.Path != "Services.cs" || file.Language != "csharp" ||
				file.LinesTotal != 2 || file.LinesCovered != 1 || file.LineCoverage != 0.5 {
				t.Fatalf("file = %+v", file)
			}
		})
	}
}
