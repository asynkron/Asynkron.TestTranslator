package coverage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testModule = "github.com/asynkron/faktorial-go"

func openFixture(t *testing.T, name string) *os.File {
	t.Helper()
	f, err := os.Open(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("open fixture %s: %v", name, err)
	}
	t.Cleanup(func() { f.Close() })
	return f
}

func fileByPath(files []FileCoverage, path string) (FileCoverage, bool) {
	for _, fc := range files {
		if fc.Path == path {
			return fc, true
		}
	}
	return FileCoverage{}, false
}

// AC-2: Go coverprofile parses into the shared model with repo-relative paths
// and statement (line) totals/covered counts.
func TestParseGoCoverprofile(t *testing.T) {
	report, err := ParseGoCoverprofile(openFixture(t, "go-coverprofile.txt"), "/repo", testModule)
	if err != nil {
		t.Fatalf("ParseGoCoverprofile: %v", err)
	}
	if report.SchemaVersion != SchemaVersion {
		t.Fatalf("schema version = %q, want %q", report.SchemaVersion, SchemaVersion)
	}
	if report.SourceFormat != goSourceFormat {
		t.Fatalf("source format = %q, want %q", report.SourceFormat, goSourceFormat)
	}

	sample, ok := fileByPath(report.Files, "verticals/experiments/coverage/sample.go")
	if !ok {
		t.Fatalf("missing sample.go in %+v", report.Files)
	}
	// blocks: 2@1, 2@0, 1@1 -> 5 statements, 3 covered (the native Go metric).
	if sample.StatementsTotal == nil || *sample.StatementsTotal != 5 ||
		sample.StatementsCovered == nil || *sample.StatementsCovered != 3 {
		t.Fatalf("sample.go statements = %v/%v, want 3/5", sample.StatementsCovered, sample.StatementsTotal)
	}
	if sample.StatementCoverage == nil || *sample.StatementCoverage != 0.6 {
		t.Fatalf("sample.go statement coverage = %v, want 0.6", sample.StatementCoverage)
	}
	// The line metric is now distinct (derived from statement block ranges): the
	// blocks span lines 10-12, 14-16, 16-18 -> 8 distinct lines, 6 covered.
	if sample.LinesTotal != 8 || sample.LinesCovered != 6 {
		t.Fatalf("sample.go lines = %d/%d, want 6/8", sample.LinesCovered, sample.LinesTotal)
	}
	if sample.LineCoverage != 0.75 {
		t.Fatalf("sample.go line coverage = %v, want 0.75", sample.LineCoverage)
	}
	if sample.Language != "go" {
		t.Fatalf("sample.go language = %q, want go", sample.Language)
	}
	// Go has no native branch/function coverage.
	if sample.BranchesTotal != nil || sample.FunctionsTotal != nil {
		t.Fatalf("Go record must not synthesize branch/function fields: %+v", sample)
	}

	quickdup, ok := fileByPath(report.Files, "verticals/web/quickdup.go")
	if !ok {
		t.Fatalf("missing quickdup.go in %+v", report.Files)
	}
	// blocks: 3@1, 2@0 -> 5 statements, 3 covered; lines span 5-7 and 9-11 ->
	// 6 distinct lines, 3 covered.
	if quickdup.StatementsTotal == nil || *quickdup.StatementsTotal != 5 ||
		quickdup.StatementsCovered == nil || *quickdup.StatementsCovered != 3 {
		t.Fatalf("quickdup.go statements = %v/%v, want 3/5", quickdup.StatementsCovered, quickdup.StatementsTotal)
	}
	if quickdup.LinesTotal != 6 || quickdup.LinesCovered != 3 {
		t.Fatalf("quickdup.go lines = %d/%d, want 3/6", quickdup.LinesCovered, quickdup.LinesTotal)
	}

	// AC-4 (unsafe case): the foreign dependency import path is reported, not
	// silently translated into a bogus repo-relative key.
	if _, ok := fileByPath(report.Files, "golang.org/x/tools/go/packages/packages.go"); ok {
		t.Fatalf("foreign import path must not appear as a repo-relative key")
	}
	if len(report.UnavailableReasons) == 0 {
		t.Fatalf("expected an unavailable reason for the out-of-module import path")
	}
	if !strings.Contains(strings.Join(report.UnavailableReasons, "\n"), "golang.org/x/tools") {
		t.Fatalf("unavailable reasons = %v, want the foreign import path", report.UnavailableReasons)
	}
}

func TestParseGoCoverprofileSortsOutput(t *testing.T) {
	body := `mode: set
github.com/asynkron/faktorial-go/verticals/web/quickdup.go:5.10,7.2 3 1
github.com/asynkron/faktorial-go/verticals/experiments/coverage/sample.go:10.20,12.2 2 0
z.example/dep/pkg.go:1.1,2.2 1 1
a.example/dep/pkg.go:1.1,2.2 1 1
`
	report, err := ParseGoCoverprofile(strings.NewReader(body), "/repo", testModule)
	if err != nil {
		t.Fatalf("ParseGoCoverprofile: %v", err)
	}
	gotPaths := make([]string, 0, len(report.Files))
	for _, fc := range report.Files {
		gotPaths = append(gotPaths, fc.Path)
	}
	wantPaths := []string{
		"verticals/experiments/coverage/sample.go",
		"verticals/web/quickdup.go",
	}
	if strings.Join(gotPaths, "\n") != strings.Join(wantPaths, "\n") {
		t.Fatalf("file order = %v, want %v", gotPaths, wantPaths)
	}
	if len(report.UnavailableReasons) != 2 {
		t.Fatalf("unavailable reasons = %v, want 2 entries", report.UnavailableReasons)
	}
	if !strings.Contains(report.UnavailableReasons[0], "a.example/dep/pkg.go") ||
		!strings.Contains(report.UnavailableReasons[1], "z.example/dep/pkg.go") {
		t.Fatalf("unavailable reasons not sorted: %v", report.UnavailableReasons)
	}
}

func TestParseGoCoverprofileDerivesNamedFunctionsAndMethodsWithoutPositions(t *testing.T) {
	root := t.TempDir()
	source := `package sample

func Covered() int {
	return 1
}

type Thing struct{}

func (Thing) Missed() int {
	return 2
}
`
	full := filepath.Join(root, "sample", "sample.go")
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	profile := `mode: set
example.com/repo/sample/sample.go:3.1,5.2 1 1
example.com/repo/sample/sample.go:9.1,11.2 1 0
`
	report, err := ParseGoCoverprofile(strings.NewReader(profile), root, "example.com/repo")
	if err != nil {
		t.Fatalf("ParseGoCoverprofile: %v", err)
	}
	file, ok := fileByPath(report.Files, "sample/sample.go")
	if !ok {
		t.Fatalf("missing sample file: %+v", report.Files)
	}
	if file.FunctionsTotal == nil || *file.FunctionsTotal != 2 ||
		file.FunctionsCovered == nil || *file.FunctionsCovered != 1 {
		t.Fatalf("function totals = %v/%v, want 1/2", file.FunctionsCovered, file.FunctionsTotal)
	}
	if len(file.Symbols) != 2 {
		t.Fatalf("symbols = %+v, want two", file.Symbols)
	}
	if file.Symbols[0].Name != "(Thing).Missed" || file.Symbols[0].Kind != "method" || file.Symbols[0].Covered {
		t.Fatalf("method symbol = %+v", file.Symbols[0])
	}
	if file.Symbols[1].Name != "Covered" || file.Symbols[1].Kind != "function" || !file.Symbols[1].Covered {
		t.Fatalf("function symbol = %+v", file.Symbols[1])
	}
	raw, err := json.Marshal(file.Symbols)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), `"line"`) || strings.Contains(string(raw), `"column"`) {
		t.Fatalf("durable symbols contain source positions: %s", raw)
	}
}

// AC-3: Vitest Istanbul coverage-final.json parses into the same shared model
// with repo-relative paths and line totals/covered counts, plus optional
// branch/function fields where Istanbul provides them.
func TestParseVitestIstanbul(t *testing.T) {
	report, err := ParseVitestIstanbul(openFixture(t, "vitest-coverage-final.json"), "/repo")
	if err != nil {
		t.Fatalf("ParseVitestIstanbul: %v", err)
	}
	if report.SchemaVersion != SchemaVersion {
		t.Fatalf("schema version = %q, want %q", report.SchemaVersion, SchemaVersion)
	}
	if report.SourceFormat != vitestSourceFormat {
		t.Fatalf("source format = %q, want %q", report.SourceFormat, vitestSourceFormat)
	}

	sum, ok := fileByPath(report.Files, "packages/frontend/src/sum.ts")
	if !ok {
		t.Fatalf("missing sum.ts in %+v", report.Files)
	}
	// statements on lines 1,2,3 with hits 1,0,3 -> 2 of 3 lines covered.
	if sum.LinesTotal != 3 || sum.LinesCovered != 2 {
		t.Fatalf("sum.ts totals = %d/%d, want 2/3", sum.LinesCovered, sum.LinesTotal)
	}
	// The native statement metric is surfaced distinctly: 3 statements, 2 with hits.
	if sum.StatementsTotal == nil || *sum.StatementsTotal != 3 ||
		sum.StatementsCovered == nil || *sum.StatementsCovered != 2 {
		t.Fatalf("sum.ts statements = %v/%v, want 2/3", sum.StatementsCovered, sum.StatementsTotal)
	}
	if sum.Language != "typescript" {
		t.Fatalf("sum.ts language = %q, want typescript", sum.Language)
	}
	if sum.BranchesTotal == nil || *sum.BranchesTotal != 2 || sum.BranchesCovered == nil || *sum.BranchesCovered != 1 {
		t.Fatalf("sum.ts branch coverage not populated correctly: %+v", sum)
	}
	if sum.FunctionsTotal == nil || *sum.FunctionsTotal != 1 || sum.FunctionsCovered == nil || *sum.FunctionsCovered != 1 {
		t.Fatalf("sum.ts function coverage not populated correctly: %+v", sum)
	}
	if len(sum.Symbols) != 1 || sum.Symbols[0].Name != "sum" || !sum.Symbols[0].Covered ||
		sum.Symbols[0].Hits == nil || *sum.Symbols[0].Hits != 1 {
		t.Fatalf("sum.ts symbols not preserved correctly: %+v", sum.Symbols)
	}

	mul, ok := fileByPath(report.Files, "packages/frontend/src/mul.ts")
	if !ok {
		t.Fatalf("missing mul.ts in %+v", report.Files)
	}
	if mul.LinesTotal != 2 || mul.LinesCovered != 0 {
		t.Fatalf("mul.ts totals = %d/%d, want 0/2", mul.LinesCovered, mul.LinesTotal)
	}
	// No branch/function data in the fixture -> fields stay nil (absence, not 0).
	if mul.BranchesTotal != nil || mul.FunctionsTotal != nil {
		t.Fatalf("mul.ts must not synthesize absent branch/function fields: %+v", mul)
	}
}

// AC-3 realism: real Vitest emits absolute build-machine paths; those must be
// re-rooted to repo-relative keys when they fall under the repo root.
func TestParseVitestIstanbulAbsoluteUnderRoot(t *testing.T) {
	root := t.TempDir()
	abs := filepath.ToSlash(filepath.Join(root, "packages/frontend/src/app.ts"))
	body := `{"` + abs + `":{"path":"` + abs + `","statementMap":{"0":{"start":{"line":1}}},"s":{"0":1}}}`
	report, err := ParseVitestIstanbul(strings.NewReader(body), root)
	if err != nil {
		t.Fatalf("ParseVitestIstanbul: %v", err)
	}
	if _, ok := fileByPath(report.Files, "packages/frontend/src/app.ts"); !ok {
		t.Fatalf("absolute-under-root path not re-rooted: %+v", report.Files)
	}
}

// AC-4 (unsafe case): an Istanbul report whose file lives outside the repo root
// is flagged in UnavailableReasons, never emitted as a coverage record.
func TestParseVitestIstanbulUnsafePath(t *testing.T) {
	report, err := ParseVitestIstanbul(openFixture(t, "vitest-unsafe.json"), "/repo")
	if err != nil {
		t.Fatalf("ParseVitestIstanbul: %v", err)
	}
	if len(report.Files) != 0 {
		t.Fatalf("escaping path must not produce a record, got %+v", report.Files)
	}
	if len(report.UnavailableReasons) == 0 {
		t.Fatalf("expected an unavailable reason for the escaping path")
	}
}

func TestNormalizeKeyUnsafe(t *testing.T) {
	cases := []struct {
		name   string
		raw    string
		root   string
		module string
	}{
		{"escaping relative", "../../etc/passwd", "/repo", ""},
		{"absolute outside root", "/home/runner/leak.ts", "/repo", ""},
		{"foreign module import", "golang.org/x/tools/x.go", "/repo", testModule},
		{"empty", "  ", "/repo", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := normalizeKey(tc.raw, tc.root, tc.module); err == nil {
				t.Fatalf("normalizeKey(%q) = nil error, want ErrUnsafePath", tc.raw)
			}
		})
	}
}

// AC-1/AC-3: both formats produce the same shared shape — same schema version
// and the same repo-relative POSIX key style — so a future ingester can join
// them on one column.
func TestBothFormatsShareSchema(t *testing.T) {
	goReport, err := ParseGoCoverprofile(openFixture(t, "go-coverprofile.txt"), "/repo", testModule)
	if err != nil {
		t.Fatalf("go parse: %v", err)
	}
	vitestReport, err := ParseVitestIstanbul(openFixture(t, "vitest-coverage-final.json"), "/repo")
	if err != nil {
		t.Fatalf("vitest parse: %v", err)
	}
	if goReport.SchemaVersion != vitestReport.SchemaVersion {
		t.Fatalf("schema mismatch: go=%q vitest=%q", goReport.SchemaVersion, vitestReport.SchemaVersion)
	}
	for _, report := range []*Report{goReport, vitestReport} {
		if len(report.Files) == 0 {
			t.Fatalf("%s produced no files", report.SourceFormat)
		}
		for _, fc := range report.Files {
			if filepath.IsAbs(fc.Path) || strings.Contains(fc.Path, "\\") {
				t.Fatalf("%s key %q is not a repo-relative POSIX path", report.SourceFormat, fc.Path)
			}
			if strings.HasPrefix(fc.Path, "../") {
				t.Fatalf("%s key %q escapes the repo root", report.SourceFormat, fc.Path)
			}
		}
	}
}
