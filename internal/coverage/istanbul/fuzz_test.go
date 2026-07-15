package istanbul

import (
	"strings"
	"testing"

	"github.com/asynkron/Asynkron.TestTranslator/internal/coverage"
	"github.com/asynkron/Asynkron.TestTranslator/internal/diagnostics"
	"github.com/asynkron/Asynkron.TestTranslator/internal/pathutil"
)

// FuzzParse ensures the Istanbul coverage-final.json parser never panics.
func FuzzParse(f *testing.F) {
	f.Add(validFixture)
	f.Add("{}")
	f.Add(`{"a.js":{}}`)
	f.Add("")
	f.Add("{")
	f.Fuzz(func(t *testing.T, in string) {
		_, _ = Adapter{}.Parse(strings.NewReader(in), coverage.Options{
			Paths: pathutil.New("", ""),
			Diag:  diagnostics.NewCollector(),
		})
	})
}
