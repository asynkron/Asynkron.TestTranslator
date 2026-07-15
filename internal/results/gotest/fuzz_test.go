package gotest

import (
	"strings"
	"testing"

	"github.com/asynkron/Asynkron.TestTranslator/internal/diagnostics"
	"github.com/asynkron/Asynkron.TestTranslator/internal/results"
)

// FuzzParse ensures the test2json parser never panics on arbitrary input; it
// may return an error, but must not crash.
func FuzzParse(f *testing.F) {
	f.Add(passFailSkip)
	f.Add(buildFailure)
	f.Add(`{"Action":"pass","Package":"p","Test":"T","Elapsed":0.1}`)
	f.Add("")
	f.Add("{")
	f.Fuzz(func(t *testing.T, in string) {
		_, _ = Adapter{}.Parse(strings.NewReader(in), results.Options{Diag: diagnostics.NewCollector()})
	})
}
