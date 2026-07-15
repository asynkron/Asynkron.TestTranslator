package jacoco

import (
	"strings"
	"testing"

	"github.com/asynkron/Asynkron.TestTranslator/internal/coverage"
	"github.com/asynkron/Asynkron.TestTranslator/internal/diagnostics"
	"github.com/asynkron/Asynkron.TestTranslator/internal/pathutil"
)

// FuzzParse ensures the JaCoCo XML parser never panics, including on DOCTYPE and
// malformed nesting.
func FuzzParse(f *testing.F) {
	f.Add(validJacoco)
	f.Add("<report><package><sourcefile></report>")
	f.Add("<!DOCTYPE report><report/>")
	f.Add("")
	f.Add("<report/>")
	f.Fuzz(func(t *testing.T, in string) {
		_, _ = Adapter{}.Parse(strings.NewReader(in), coverage.Options{
			Paths: pathutil.New("", ""),
			Diag:  diagnostics.NewCollector(),
		})
	})
}
