package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	// Register all adapters for end-to-end CLI tests.
	_ "github.com/asynkron/Asynkron.TestTranslator/internal/adapters"
)

// run invokes the CLI with the given stdin and returns exit code, stdout, stderr.
func run(args []string, stdin string) (int, string, string) {
	var out, errb bytes.Buffer
	code := Run(args, Streams{In: strings.NewReader(stdin), Out: &out, Err: &errb})
	return code, out.String(), errb.String()
}

const goJSON = `{"Action":"run","Package":"p","Test":"TestA"}
{"Action":"pass","Package":"p","Test":"TestA","Elapsed":0.01}
{"Action":"run","Package":"p","Test":"TestB"}
{"Action":"output","Package":"p","Test":"TestB","Output":"boom\n"}
{"Action":"fail","Package":"p","Test":"TestB","Elapsed":0.02}
{"Action":"fail","Package":"p","Elapsed":0.03}
`

func TestResultsStdinToStdout(t *testing.T) {
	code, out, _ := run([]string{"results", "--format", "go-test-json", "--input", "-", "--output", "-"}, goJSON)
	if code != 0 {
		t.Fatalf("exit=%d, want 0", code)
	}
	if !strings.Contains(out, `<testsuites`) || !strings.Contains(out, `failures="1"`) {
		t.Fatalf("unexpected stdout:\n%s", out)
	}
}

func TestResultsToFileIsValid(t *testing.T) {
	dir := t.TempDir()
	inPath := filepath.Join(dir, "go.jsonl")
	outPath := filepath.Join(dir, "junit.xml")
	if err := os.WriteFile(inPath, []byte(goJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	code, _, errOut := run([]string{"results", "--format", "go-test-json", "--input", inPath, "--output", outPath}, "")
	if code != 0 {
		t.Fatalf("exit=%d stderr=%s", code, errOut)
	}
	vcode, _, _ := run([]string{"validate", "junit", "--input", outPath}, "")
	if vcode != 0 {
		t.Fatal("generated JUnit did not validate")
	}
}

func TestMultipleTypedInputsMerge(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.jsonl")
	b := filepath.Join(dir, "b.xml")
	os.WriteFile(a, []byte(goJSON), 0o644)
	os.WriteFile(b, []byte(`<testsuite name="x" tests="1"><testcase name="t" classname="x" time="0"/></testsuite>`), 0o644)
	code, out, errOut := run([]string{
		"results",
		"--input", "go-test-json=" + a,
		"--input", "junit-xml=" + b,
		"--output", "-",
	}, "")
	if code != 0 {
		t.Fatalf("exit=%d stderr=%s", code, errOut)
	}
	// 2 from go (TestA, TestB) + 1 from junit-xml = 3 tests.
	if !strings.Contains(out, `tests="3"`) {
		t.Fatalf("inputs not merged:\n%s", out)
	}
}

func TestCoverageGoProfile(t *testing.T) {
	prof := "mode: set\ngithub.com/example/proj/pkg/foo.go:1.1,2.10 2 1\ngithub.com/example/proj/pkg/foo.go:3.1,3.5 1 0\n"
	code, out, errOut := run([]string{
		"coverage", "--format", "go-coverprofile",
		"--go-module", "github.com/example/proj",
		"--input", "-", "--output", "-",
	}, prof)
	if code != 0 {
		t.Fatalf("exit=%d stderr=%s", code, errOut)
	}
	if !strings.Contains(out, "<!DOCTYPE coverage") || !strings.Contains(out, `filename="pkg/foo.go"`) {
		t.Fatalf("unexpected cobertura:\n%s", out)
	}
}

func TestUnknownFormatFails(t *testing.T) {
	code, _, errOut := run([]string{"results", "--format", "nope", "--input", "-"}, goJSON)
	if code == 0 {
		t.Fatal("expected non-zero exit for unknown format")
	}
	if !strings.Contains(errOut, "unknown test-result format") {
		t.Fatalf("unexpected error: %s", errOut)
	}
}

func TestMissingInputFails(t *testing.T) {
	code, _, _ := run([]string{"results", "--format", "go-test-json"}, "")
	if code == 0 {
		t.Fatal("expected non-zero exit when no input given")
	}
}

func TestValidateRejectsBadJUnit(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "bad.xml")
	os.WriteFile(p, []byte(`<testsuites tests="9"><testsuite name="s" tests="0"></testsuite></testsuites>`), 0o644)
	code, _, _ := run([]string{"validate", "junit", "--input", p}, "")
	if code == 0 {
		t.Fatal("expected validation failure for count mismatch")
	}
}

func TestDiagnosticsJSON(t *testing.T) {
	_, _, errOut := run([]string{
		"results", "--format", "go-test-json", "--input", "-", "--output", "-",
		"--diagnostics", "json",
	}, "{\"Action\":\"fail\",\"Package\":\"broken\",\"Elapsed\":0}\n")
	if !strings.Contains(errOut, `"severity"`) || !strings.Contains(errOut, `"code"`) {
		t.Fatalf("expected JSON diagnostics on stderr:\n%s", errOut)
	}
}

func TestFormatsJSON(t *testing.T) {
	code, out, _ := run([]string{"formats", "--diagnostics", "json"}, "")
	if code != 0 {
		t.Fatal("formats failed")
	}
	if !strings.Contains(out, `"results"`) || !strings.Contains(out, `"coverage"`) {
		t.Fatalf("unexpected formats json:\n%s", out)
	}
}

func TestManifestRoundTrip(t *testing.T) {
	dir := t.TempDir()
	junitPath := filepath.Join(dir, "j.xml")
	covPath := filepath.Join(dir, "c.xml")
	os.WriteFile(junitPath, []byte("<testsuites/>"), 0o644)
	os.WriteFile(covPath, []byte("<coverage/>"), 0o644)
	mPath := filepath.Join(dir, "bundle.json")
	code, _, errOut := run([]string{
		"manifest",
		"--add", "test-results:junit:" + junitPath,
		"--add", "coverage:cobertura:" + covPath,
		"--output", mPath,
	}, "")
	if code != 0 {
		t.Fatalf("manifest exit=%d stderr=%s", code, errOut)
	}
	data, _ := os.ReadFile(mPath)
	if !strings.Contains(string(data), `"sha256"`) {
		t.Fatalf("manifest missing checksums:\n%s", data)
	}
}
