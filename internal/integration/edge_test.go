package integration

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/asynkron/Asynkron.TestTranslator/internal/coverage"
	"github.com/asynkron/Asynkron.TestTranslator/internal/diagnostics"
	"github.com/asynkron/Asynkron.TestTranslator/internal/pathutil"
	"github.com/asynkron/Asynkron.TestTranslator/internal/results"
	"github.com/asynkron/Asynkron.TestTranslator/internal/results/junit"
)

// parseEdgeResults / parseEdgeCoverage parse an edge fixture with the adapter
// named by format. They never call t.Fatal on a parse error so callers can
// assert on it; a panic (not an error) fails the test by crashing it.
func parseEdgeResults(t *testing.T, format string, data []byte) error {
	t.Helper()
	adapter, err := results.Lookup(format)
	if err != nil {
		t.Fatalf("lookup %s: %v", format, err)
	}
	_, perr := adapter.Parse(bytes.NewReader(data), results.Options{SourceName: "edge", Diag: diagnostics.NewCollector()})
	return perr
}

func parseEdgeCoverage(t *testing.T, format string, data []byte) error {
	t.Helper()
	adapter, err := coverage.Lookup(format)
	if err != nil {
		t.Fatalf("lookup %s: %v", format, err)
	}
	_, perr := adapter.Parse(bytes.NewReader(data), coverage.Options{
		SourceName: "edge",
		Paths:      pathutil.New("/", ""),
		Diag:       diagnostics.NewCollector(),
	})
	return perr
}

// Every malformed fixture must be rejected with a clean, described error — never
// a panic and never a silent success.
func TestMalformedRejectedCleanly(t *testing.T) {
	cases := []struct {
		file     string
		format   string
		coverage bool
	}{
		{"trx.trx", "vstest-trx", false},
		{"junit-xml.xml", "junit-xml", false},
		{"xunit.xml", "xunit", false},
		{"nunit.xml", "nunit", false},
		{"tap.tap", "tap", false},
		{"go-test-json.out", "go-test-json", false},
		{"lcov.info", "lcov", true},
		{"istanbul-json.json", "istanbul-json", true},
		{"jacoco.xml", "jacoco", true},
		{"cobertura.xml", "cobertura", true},
		{"go-coverprofile.out", "go-coverprofile", true},
	}
	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "edge", "malformed", tc.file))
			if err != nil {
				t.Fatalf("read fixture: %v", err)
			}
			var perr error
			if tc.coverage {
				perr = parseEdgeCoverage(t, tc.format, data)
			} else {
				perr = parseEdgeResults(t, tc.format, data)
			}
			if perr == nil {
				t.Fatalf("%s: malformed input was accepted; expected a clean error", tc.format)
			}
			if strings.TrimSpace(perr.Error()) == "" {
				t.Errorf("%s: error message is empty", tc.format)
			}
		})
	}
}

// Empty test runs must convert to a valid (empty) JUnit document, not error or
// crash.
func TestEmptyRunsConvert(t *testing.T) {
	for _, tc := range []struct{ file, format string }{
		{"junit-xml.xml", "junit-xml"},
		{"tap.tap", "tap"},
		{"go-test-json.out", "go-test-json"},
	} {
		t.Run(tc.file, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "edge", "empty", tc.file))
			if err != nil {
				t.Fatalf("read fixture: %v", err)
			}
			adapter, err := results.Lookup(tc.format)
			if err != nil {
				t.Fatalf("lookup: %v", err)
			}
			rep, err := adapter.Parse(bytes.NewReader(data), results.Options{SourceName: "edge", Diag: diagnostics.NewCollector()})
			if err != nil {
				t.Fatalf("empty run should parse cleanly: %v", err)
			}
			rep.Sort()
			out, err := junit.Marshal(rep)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			if err := junit.Validate(bytes.NewReader(out)); err != nil {
				t.Errorf("empty JUnit failed validation: %v", err)
			}
			if tot := rep.Totals(); tot.Tests != 0 {
				t.Errorf("expected 0 tests for an empty run, got %d", tot.Tests)
			}
		})
	}
}

// Unicode in test names and messages must survive parse -> JUnit marshal intact.
func TestUnicodePreserved(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "edge", "unicode", "junit-xml.xml"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	adapter, err := results.Lookup("junit-xml")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	rep, err := adapter.Parse(bytes.NewReader(data), results.Options{SourceName: "edge", Diag: diagnostics.NewCollector()})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	rep.Sort()
	out, err := junit.Marshal(rep)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := junit.Validate(bytes.NewReader(out)); err != nil {
		t.Fatalf("unicode JUnit failed validation: %v", err)
	}
	for _, want := range []string{"日本語のテスト_🎉", "Ω≈ç√∫˜µ", "café ≠ résumé", "Straße", "кириллица_и_ελληνικά", "παραλείπεται"} {
		if !bytes.Contains(out, []byte(want)) {
			t.Errorf("unicode content %q was lost in conversion", want)
		}
	}
}
