package gocover

import (
	"strings"
	"testing"

	"github.com/asynkron/testtranslator/internal/coverage"
	"github.com/asynkron/testtranslator/internal/diagnostics"
	"github.com/asynkron/testtranslator/internal/pathutil"
)

// FuzzParse ensures the coverprofile parser never panics.
func FuzzParse(f *testing.F) {
	f.Add(profile)
	f.Add("mode: set\n")
	f.Add("mode: count\nx/y.go:1.1,2.2 1 5\n")
	f.Add("")
	f.Add("garbage")
	f.Fuzz(func(t *testing.T, in string) {
		_, _ = Adapter{}.Parse(strings.NewReader(in), coverage.Options{
			Paths: pathutil.New("", "x"),
			Diag:  diagnostics.NewCollector(),
		})
	})
}
