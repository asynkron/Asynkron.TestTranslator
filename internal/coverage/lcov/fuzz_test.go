package lcov

import (
	"strings"
	"testing"

	"github.com/asynkron/Asynkron.TestTranslator/internal/coverage"
	"github.com/asynkron/Asynkron.TestTranslator/internal/diagnostics"
	"github.com/asynkron/Asynkron.TestTranslator/internal/pathutil"
)

// FuzzParse ensures the LCOV tracefile parser never panics, including on the
// legacy and three-field FN/BRDA record variants.
func FuzzParse(f *testing.F) {
	f.Add(validTracefile)
	f.Add("SF:a.go\nFN:1,10,add\nFNDA:3,add\nend_of_record\n")
	f.Add("SF:a.go\nBRDA:1,0,0,-\nend_of_record\n")
	f.Add("end_of_record\n")
	f.Add("")
	f.Add("garbage")
	f.Fuzz(func(t *testing.T, in string) {
		_, _ = Adapter{}.Parse(strings.NewReader(in), coverage.Options{
			Paths: pathutil.New("", ""),
			Diag:  diagnostics.NewCollector(),
		})
	})
}
