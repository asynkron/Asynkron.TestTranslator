package xunit

import (
	"strings"
	"testing"

	"github.com/asynkron/testtranslator/internal/diagnostics"
	"github.com/asynkron/testtranslator/internal/results"
)

// FuzzParse ensures the xUnit.net v2/v3 parser never panics on arbitrary input.
func FuzzParse(f *testing.F) {
	f.Add(validV2)
	f.Add(validV3BareAssembly)
	f.Add("<assemblies>")
	f.Add("<assembly name=\"x\">")
	f.Add("")
	f.Add("<!DOCTYPE x><assemblies/>")
	f.Fuzz(func(t *testing.T, in string) {
		_, _ = Adapter{}.Parse(strings.NewReader(in), results.Options{Diag: diagnostics.NewCollector()})
	})
}
