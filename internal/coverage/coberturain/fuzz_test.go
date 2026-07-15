package coberturain

import (
	"strings"
	"testing"

	"github.com/asynkron/testtranslator/internal/coverage"
	"github.com/asynkron/testtranslator/internal/diagnostics"
	"github.com/asynkron/testtranslator/internal/pathutil"
)

// FuzzParse ensures the Cobertura-input parser never panics, including on custom
// entities and malformed nesting.
func FuzzParse(f *testing.F) {
	f.Add(validCobertura)
	f.Add("<coverage><packages><package></coverage>")
	f.Add("<coverage/>")
	f.Add("")
	f.Add("<!DOCTYPE x><coverage/>")
	f.Fuzz(func(t *testing.T, in string) {
		_, _ = Adapter{}.Parse(strings.NewReader(in), coverage.Options{
			Paths: pathutil.New("", ""),
			Diag:  diagnostics.NewCollector(),
		})
	})
}
