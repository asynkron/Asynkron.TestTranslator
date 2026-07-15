package integration

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/asynkron/testtranslator/internal/coverage"
	"github.com/asynkron/testtranslator/internal/coverage/cobertura"
	"github.com/asynkron/testtranslator/internal/diagnostics"
	"github.com/asynkron/testtranslator/internal/pathutil"
	"github.com/asynkron/testtranslator/internal/results"
	"github.com/asynkron/testtranslator/internal/results/junit"
)

// updateGolden regenerates the checked-in golden files instead of comparing.
// Run: go test ./internal/integration/ -run TestGolden -update
var updateGolden = flag.Bool("update", false, "update golden output files")

// Golden tests pin the exact JUnit/Cobertura bytes produced for representative
// real fixtures. The golden files are committed and human-reviewed, so any
// future change in writer output shows up as a reviewable diff. Each golden is
// also re-validated by the independent validator.
func TestGolden(t *testing.T) {
	cases := []struct {
		name     string
		coverage bool
		format   string
		input    string // relative to testdata/real
		golden   string // filename under testdata/golden
	}{
		{"gotest", false, "go-test-json", "go-testjson/go-test-json.out", "gotest.junit.xml"},
		{"surefire", false, "junit-xml", "junit-xml/surefire-TEST-surefire.MyTest.xml", "surefire.junit.xml"},
		{"tap", false, "tap", "tap/tapjs_simple_yaml.tap", "tap.junit.xml"},
		{"lcov", true, "lcov", "lcov/lcov-merger-basic-a.info", "lcov.cobertura.xml"},
		{"istanbul", true, "istanbul-json", "istanbul-json/macaroons_js.json", "istanbul.cobertura.xml"},
		{"coberturain", true, "cobertura", "cobertura/coveragepy-python-branch.xml", "coveragepy.cobertura.xml"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in, err := os.ReadFile(filepath.Join("..", "..", "testdata", "real", tc.input))
			if err != nil {
				t.Skipf("fixture unavailable: %v", err)
			}

			var got []byte
			if tc.coverage {
				adapter, lerr := coverage.Lookup(tc.format)
				if lerr != nil {
					t.Fatalf("lookup: %v", lerr)
				}
				rep, perr := adapter.Parse(bytes.NewReader(in), coverage.Options{
					SourceName: "golden",
					Paths:      pathutil.New("/", ""),
					Diag:       diagnostics.NewCollector(),
				})
				if perr != nil {
					t.Fatalf("parse: %v", perr)
				}
				got, err = cobertura.Marshal(rep)
				if err != nil {
					t.Fatalf("marshal: %v", err)
				}
				if verr := cobertura.Validate(bytes.NewReader(got)); verr != nil {
					t.Fatalf("produced cobertura invalid: %v", verr)
				}
			} else {
				adapter, lerr := results.Lookup(tc.format)
				if lerr != nil {
					t.Fatalf("lookup: %v", lerr)
				}
				rep, perr := adapter.Parse(bytes.NewReader(in), results.Options{
					SourceName: "golden",
					Diag:       diagnostics.NewCollector(),
				})
				if perr != nil {
					t.Fatalf("parse: %v", perr)
				}
				rep.Sort()
				got, err = junit.Marshal(rep)
				if err != nil {
					t.Fatalf("marshal: %v", err)
				}
				if verr := junit.Validate(bytes.NewReader(got)); verr != nil {
					t.Fatalf("produced junit invalid: %v", verr)
				}
			}

			goldenPath := filepath.Join("..", "..", "testdata", "golden", tc.golden)
			if *updateGolden {
				if err := os.WriteFile(goldenPath, got, 0o644); err != nil {
					t.Fatalf("write golden: %v", err)
				}
				t.Logf("updated %s", tc.golden)
				return
			}
			want, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatalf("read golden (run with -update to create): %v", err)
			}
			if !bytes.Equal(got, want) {
				t.Errorf("output does not match golden %s (run -update to refresh if the change is intended)\n--- got ---\n%s", tc.golden, got)
			}
		})
	}
}
