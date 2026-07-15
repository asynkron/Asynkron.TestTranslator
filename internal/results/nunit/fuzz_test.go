package nunit

import (
	"strings"
	"testing"

	"github.com/asynkron/testtranslator/internal/diagnostics"
	"github.com/asynkron/testtranslator/internal/results"
)

// FuzzParse ensures the NUnit 2/3 parser never panics on arbitrary input.
func FuzzParse(f *testing.F) {
	f.Add(nunit2)
	f.Add(nunit3)
	f.Add("<test-run>")
	f.Add("<test-results/>")
	f.Add("")
	f.Add("<!DOCTYPE x><test-run/>")
	f.Fuzz(func(t *testing.T, in string) {
		_, _ = Adapter{}.Parse(strings.NewReader(in), results.Options{Diag: diagnostics.NewCollector()})
	})
}
