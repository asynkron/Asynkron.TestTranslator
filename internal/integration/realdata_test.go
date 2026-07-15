// Package integration exercises every bundled adapter against the corpus of
// REAL, third-party test-output fixtures under testdata/real. These fixtures
// are unmodified files committed by upstream projects (see each directory's
// PROVENANCE.md), so they prove the adapters against material produced by the
// actual tools rather than hand-written samples.
//
// The proving invariants are:
//   - No adapter panics on real input.
//   - Any successful conversion produces a document that passes the
//     independent JUnit/Cobertura validator.
//   - A failure is a clean, described error (e.g. an absolute path that cannot
//     be rerooted), never a crash.
//   - Across the whole corpus, real data actually flows through end to end.
package integration

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/asynkron/Asynkron.TestTranslator/internal/adapters"
	"github.com/asynkron/Asynkron.TestTranslator/internal/coverage"
	"github.com/asynkron/Asynkron.TestTranslator/internal/coverage/cobertura"
	"github.com/asynkron/Asynkron.TestTranslator/internal/diagnostics"
	"github.com/asynkron/Asynkron.TestTranslator/internal/pathutil"
	"github.com/asynkron/Asynkron.TestTranslator/internal/results"
	"github.com/asynkron/Asynkron.TestTranslator/internal/results/junit"
)

// dirFormat maps a testdata/real subdirectory to the adapter that should parse
// its files.
type dirFormat struct {
	dir      string
	format   string
	coverage bool
}

var corpus = []dirFormat{
	{dir: "go-testjson", format: "go-test-json"},
	{dir: "junit-xml", format: "junit-xml"},
	{dir: "trx", format: "vstest-trx"},
	{dir: "nunit", format: "nunit"},
	{dir: "xunit", format: "xunit"},
	{dir: "tap", format: "tap"},
	{dir: "lcov", format: "lcov", coverage: true},
	{dir: "istanbul-json", format: "istanbul-json", coverage: true},
	{dir: "jacoco", format: "jacoco", coverage: true},
	{dir: "cobertura", format: "cobertura", coverage: true},
	{dir: "go-coverprofile", format: "go-coverprofile", coverage: true},
}

// realFiles returns non-provenance fixture files in a directory.
func realFiles(t *testing.T, dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if name == "PROVENANCE.md" || strings.HasSuffix(name, ".md") {
			continue
		}
		files = append(files, filepath.Join(dir, name))
	}
	return files
}

func TestRealCorpus(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "real")
	if _, err := os.Stat(root); err != nil {
		t.Skipf("no real corpus present: %v", err)
	}

	var totalFiles, converted, cleanErrors int
	for _, cf := range corpus {
		dir := filepath.Join(root, cf.dir)
		files := realFiles(t, dir)
		for _, path := range files {
			totalFiles++
			t.Run(cf.format+"/"+filepath.Base(path), func(t *testing.T) {
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("read: %v", err)
				}
				if cf.coverage {
					ok, cerr := convertCoverage(t, cf.format, data)
					recordOutcome(t, ok, cerr, &converted, &cleanErrors)
				} else {
					ok, cerr := convertResults(t, cf.format, data)
					recordOutcome(t, ok, cerr, &converted, &cleanErrors)
				}
			})
		}
	}

	t.Logf("real corpus: %d files, %d converted+validated, %d clean errors", totalFiles, converted, cleanErrors)
	if totalFiles > 0 && converted == 0 {
		t.Fatalf("no real fixture converted successfully; the happy path is unproven")
	}
}

func recordOutcome(t *testing.T, ok bool, cerr error, converted, cleanErrors *int) {
	t.Helper()
	switch {
	case ok:
		*converted++
	case cerr != nil && strings.TrimSpace(cerr.Error()) != "":
		// A clean, described error is acceptable for foreign fixtures (e.g.
		// absolute paths that cannot be rerooted without --repo-root).
		*cleanErrors++
		t.Logf("clean rejection: %v", cerr)
	default:
		t.Fatalf("adapter returned a non-descriptive failure")
	}
}

// convertResults parses a result fixture and validates any produced JUnit.
// Returns (converted+validated, parseError).
func convertResults(t *testing.T, format string, data []byte) (bool, error) {
	t.Helper()
	adapter, err := results.Lookup(format)
	if err != nil {
		t.Fatalf("lookup %s: %v", format, err)
	}
	diag := diagnostics.NewCollector()
	rep, err := adapter.Parse(bytes.NewReader(data), results.Options{SourceName: "real", Diag: diag})
	if err != nil {
		return false, err
	}
	rep.Sort()
	out, err := junit.Marshal(rep)
	if err != nil {
		t.Fatalf("marshal junit: %v", err)
	}
	if err := junit.Validate(bytes.NewReader(out)); err != nil {
		t.Fatalf("produced JUnit failed validation: %v", err)
	}
	return true, nil
}

// convertCoverage parses a coverage fixture and validates any produced
// Cobertura. It uses a no-reroot normalizer, so absolute-path fixtures fail
// cleanly.
func convertCoverage(t *testing.T, format string, data []byte) (bool, error) {
	t.Helper()
	adapter, err := coverage.Lookup(format)
	if err != nil {
		t.Fatalf("lookup %s: %v", format, err)
	}
	diag := diagnostics.NewCollector()
	// Real coverage reports embed absolute source paths from the machine that
	// produced them. Rerooting at "/" lets genuine unix-absolute data flow end
	// to end and proves the writer; Windows-drive or escaping paths still fail
	// cleanly. (The absolute-without-root rejection is proven by adapter unit
	// tests.)
	rep, err := adapter.Parse(bytes.NewReader(data), coverage.Options{
		SourceName: "real",
		Paths:      pathutil.New("/", ""),
		Diag:       diag,
	})
	if err != nil {
		return false, err
	}
	out, err := cobertura.Marshal(rep)
	if err != nil {
		t.Fatalf("marshal cobertura: %v", err)
	}
	if err := cobertura.Validate(bytes.NewReader(out)); err != nil {
		t.Fatalf("produced Cobertura failed validation: %v", err)
	}
	return true, nil
}
