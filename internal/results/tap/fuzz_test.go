package tap

import (
	"strings"
	"testing"

	"github.com/asynkron/testtranslator/internal/diagnostics"
	"github.com/asynkron/testtranslator/internal/results"
)

// FuzzParse ensures the TAP parser never panics on arbitrary input, including
// pathological indentation and brace nesting.
func FuzzParse(f *testing.F) {
	f.Add("TAP version 13\n1..1\nok 1 - a\n")
	f.Add("1..0 # SKIP all\n")
	f.Add("# Subtest: p\n    ok 1\n    1..1\nok 1 - p\n")
	f.Add("not ok 1 - c\n{\n    ok 1\n    1..1\n}\n1..1\n")
	f.Add("Bail out! nope\n")
	f.Add("")
	f.Fuzz(func(t *testing.T, in string) {
		_, _ = Adapter{}.Parse(strings.NewReader(in), results.Options{Diag: diagnostics.NewCollector()})
	})
}
