package testtranslator_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/asynkron/testtranslator"
)

func readFixture(t *testing.T, rel string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "real", rel))
	if err != nil {
		t.Fatalf("read fixture %s: %v", rel, err)
	}
	return data
}

func TestConvertResults(t *testing.T) {
	in := readFixture(t, "go-testjson/go-test-json.out")
	out, diags, err := testtranslator.ConvertResults("go-test-json", bytes.NewReader(in))
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	if !bytes.Contains(out, []byte("<testsuites")) {
		t.Errorf("output is not JUnit XML: %s", out[:min(120, len(out))])
	}
	// This fixture contains a build failure and an empty package, so the
	// conversion must surface diagnostics rather than dropping that information.
	if len(diags) == 0 {
		t.Errorf("expected lossy/informational diagnostics for this fixture")
	}
}

func TestConvertResultsDeterministic(t *testing.T) {
	in := readFixture(t, "go-testjson/go-test-json.out")
	a, _, err := testtranslator.ConvertResults("go-test-json", bytes.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}
	b, _, err := testtranslator.ConvertResults("go-test-json", bytes.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a, b) {
		t.Errorf("conversion is not deterministic")
	}
}

func TestConvertResultsUnknownFormat(t *testing.T) {
	_, _, err := testtranslator.ConvertResults("does-not-exist", strings.NewReader(""))
	if err == nil {
		t.Fatalf("expected an error for an unknown format")
	}
}

func TestConvertResultsMismatchedInput(t *testing.T) {
	// Feeding TAP to the go-test-json adapter must fail clearly, never guess.
	tap := readFixture(t, "tap/tapjs_basic.tap")
	_, _, err := testtranslator.ConvertResults("go-test-json", bytes.NewReader(tap))
	if err == nil {
		t.Fatalf("expected an error for mismatched input")
	}
}

func TestConvertResultsAlias(t *testing.T) {
	// "gotest" is an alias for "go-test-json".
	in := readFixture(t, "go-testjson/go-test-json.out")
	if _, _, err := testtranslator.ConvertResults("gotest", bytes.NewReader(in)); err != nil {
		t.Fatalf("alias lookup failed: %v", err)
	}
}

func TestConvertCoverageLCOV(t *testing.T) {
	in := readFixture(t, "lcov/lcov-merger-basic-a.info")
	out, _, err := testtranslator.ConvertCoverage("lcov", bytes.NewReader(in), testtranslator.CoverageOptions{RepoRoot: "/"})
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	if !bytes.Contains(out, []byte("<coverage")) {
		t.Errorf("output is not Cobertura XML")
	}
}

func TestConvertCoverageGoProfileWithModule(t *testing.T) {
	in := readFixture(t, "go-coverprofile/pathutil-set.out")
	out, _, err := testtranslator.ConvertCoverage("go-coverprofile", bytes.NewReader(in), testtranslator.CoverageOptions{
		GoModule: "github.com/asynkron/testtranslator",
	})
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	// The module path is stripped, leaving repository-relative file names.
	if !bytes.Contains(out, []byte("internal/pathutil/pathutil.go")) {
		t.Errorf("go module path was not rerooted: %s", out[:min(200, len(out))])
	}
}

func TestConvertCoverageDeterministic(t *testing.T) {
	in := readFixture(t, "lcov/lcov-merger-basic-a.info")
	a, _, err := testtranslator.ConvertCoverage("lcov", bytes.NewReader(in), testtranslator.CoverageOptions{RepoRoot: "/"})
	if err != nil {
		t.Fatal(err)
	}
	b, _, err := testtranslator.ConvertCoverage("lcov", bytes.NewReader(in), testtranslator.CoverageOptions{RepoRoot: "/"})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a, b) {
		t.Errorf("coverage conversion is not deterministic")
	}
}

func TestFormatsListed(t *testing.T) {
	rf := testtranslator.ResultFormats()
	cf := testtranslator.CoverageFormats()
	if len(rf) == 0 || len(cf) == 0 {
		t.Fatalf("expected non-empty format lists, got results=%d coverage=%d", len(rf), len(cf))
	}
	has := func(list []testtranslator.FormatInfo, id string) bool {
		for _, f := range list {
			if f.ID == id {
				return true
			}
		}
		return false
	}
	if !has(rf, "go-test-json") {
		t.Errorf("result formats missing go-test-json")
	}
	if !has(cf, "lcov") {
		t.Errorf("coverage formats missing lcov")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
