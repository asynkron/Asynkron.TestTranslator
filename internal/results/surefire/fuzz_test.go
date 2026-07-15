package surefire

import (
	"strings"
	"testing"

	"github.com/asynkron/testtranslator/internal/diagnostics"
	"github.com/asynkron/testtranslator/internal/results"
)

// FuzzParse ensures the JUnit-family XML parser never panics.
func FuzzParse(f *testing.F) {
	f.Add(suites)
	f.Add(single)
	f.Add("<testsuites>")
	f.Add("")
	f.Add("<!DOCTYPE x><testsuite/>")
	f.Fuzz(func(t *testing.T, in string) {
		_, _ = Adapter{}.Parse(strings.NewReader(in), results.Options{Diag: diagnostics.NewCollector()})
	})
}
