package integration

import (
	"bytes"
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

// Round-trip tests prove that the writers' output is stable: feeding a produced
// document back through its own input adapter and re-serializing yields an
// identical document. This catches normalization/ordering drift independently
// of the source fixtures.

func TestCoberturaRoundTripStable(t *testing.T) {
	adapter, err := coverage.Lookup("cobertura")
	if err != nil {
		t.Fatalf("lookup cobertura: %v", err)
	}
	parse := func(data []byte) []byte {
		t.Helper()
		rep, perr := adapter.Parse(bytes.NewReader(data), coverage.Options{
			SourceName: "roundtrip",
			Paths:      pathutil.New("/", ""),
			Diag:       diagnostics.NewCollector(),
		})
		if perr != nil {
			t.Fatalf("parse: %v", perr)
		}
		out, merr := cobertura.Marshal(rep)
		if merr != nil {
			t.Fatalf("marshal: %v", merr)
		}
		return out
	}

	in, err := os.ReadFile(filepath.Join("..", "..", "testdata", "real", "cobertura", "coveragepy-python-branch.xml"))
	if err != nil {
		t.Skipf("fixture unavailable: %v", err)
	}
	first := parse(in)
	second := parse(first)
	if !bytes.Equal(first, second) {
		t.Errorf("cobertura output is not stable under round-trip:\n--- first ---\n%s\n--- second ---\n%s", first, second)
	}
	if err := cobertura.Validate(bytes.NewReader(first)); err != nil {
		t.Errorf("round-tripped cobertura fails validation: %v", err)
	}
}

func TestJUnitRoundTripStable(t *testing.T) {
	adapter, err := results.Lookup("junit-xml")
	if err != nil {
		t.Fatalf("lookup junit-xml: %v", err)
	}
	parse := func(data []byte) []byte {
		t.Helper()
		rep, perr := adapter.Parse(bytes.NewReader(data), results.Options{
			SourceName: "roundtrip",
			Diag:       diagnostics.NewCollector(),
		})
		if perr != nil {
			t.Fatalf("parse: %v", perr)
		}
		rep.Sort()
		out, merr := junit.Marshal(rep)
		if merr != nil {
			t.Fatalf("marshal: %v", merr)
		}
		return out
	}

	in, err := os.ReadFile(filepath.Join("..", "..", "testdata", "real", "junit-xml", "surefire-TEST-surefire.MyTest.xml"))
	if err != nil {
		t.Skipf("fixture unavailable: %v", err)
	}
	first := parse(in)
	second := parse(first)
	if !bytes.Equal(first, second) {
		t.Errorf("junit output is not stable under round-trip:\n--- first ---\n%s\n--- second ---\n%s", first, second)
	}
	if err := junit.Validate(bytes.NewReader(first)); err != nil {
		t.Errorf("round-tripped junit fails validation: %v", err)
	}
}
