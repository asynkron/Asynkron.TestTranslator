package trx

import (
	"strings"
	"testing"

	"github.com/asynkron/Asynkron.TestTranslator/internal/diagnostics"
	"github.com/asynkron/Asynkron.TestTranslator/internal/results"
)

// FuzzParse ensures the TRX parser never panics on arbitrary input.
func FuzzParse(f *testing.F) {
	f.Add(doc)
	f.Add("<TestRun>")
	f.Add("<TestRun></TestRun>")
	f.Add("")
	f.Add("<!DOCTYPE x><TestRun/>")
	f.Fuzz(func(t *testing.T, in string) {
		_, _ = Adapter{}.Parse(strings.NewReader(in), results.Options{Diag: diagnostics.NewCollector()})
	})
}
