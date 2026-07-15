// Package testtranslator is the public Go API for converting test-result and
// coverage artifacts into the two standard interchange formats: JUnit XML for
// test results and Cobertura XML for coverage.
//
// It is a thin, stable facade over the tool's internal conversion pipeline — the
// same pipeline the testtranslator CLI uses — so a program can convert in-process
// without shelling out. The conversion is explicit (you name the source format;
// there is no auto-detection), deterministic, and validated: the returned
// document has already passed the corresponding validator.
//
//	junitXML, diags, err := testtranslator.ConvertResults("go-test-json", r)
//	if err != nil {
//	        // malformed or mismatched input
//	}
//	for _, d := range diags {
//	        // lossy or unsupported mappings, reported never silently dropped
//	}
//
// Use [ResultFormats] and [CoverageFormats] to discover supported format ids and
// their aliases.
package testtranslator

import (
	"bytes"
	"io"

	// Blank import registers every bundled adapter with the internal registries.
	_ "github.com/asynkron/Asynkron.TestTranslator/internal/adapters"
	"github.com/asynkron/Asynkron.TestTranslator/internal/coverage"
	"github.com/asynkron/Asynkron.TestTranslator/internal/coverage/cobertura"
	"github.com/asynkron/Asynkron.TestTranslator/internal/diagnostics"
	"github.com/asynkron/Asynkron.TestTranslator/internal/pathutil"
	"github.com/asynkron/Asynkron.TestTranslator/internal/results"
	"github.com/asynkron/Asynkron.TestTranslator/internal/results/junit"
)

// Severity classifies a [Diagnostic].
type Severity string

const (
	// SeverityWarning marks a lossy or unsupported mapping.
	SeverityWarning Severity = "warning"
	// SeverityNote marks informational output (e.g. a synthetic testcase).
	SeverityNote Severity = "note"
)

// Diagnostic is a structured warning or note produced during conversion, such as
// a lossy mapping the output format cannot represent. Diagnostics do not fail the
// conversion; they exist so nothing is silently discarded.
type Diagnostic struct {
	Severity Severity
	// Code is a stable, machine-readable identifier (e.g. "lossy.branches").
	Code string
	// Message is the human-readable explanation.
	Message string
	// Source optionally identifies the input or adapter.
	Source string
}

// FormatInfo describes one supported format: its canonical id, its aliases, and a
// short human-readable description.
type FormatInfo struct {
	ID          string
	Aliases     []string
	Description string
}

// CoverageOptions configures coverage-source path normalization. Coverage reports
// carry source paths that must be normalized to repository-relative POSIX form.
type CoverageOptions struct {
	// RepoRoot reroots absolute source paths to repository-relative form. Leave
	// empty when the report already uses relative paths. Paths that escape the
	// root are rejected.
	RepoRoot string
	// GoModule is the Go module path, required to unambiguously reroot the
	// import-path-based file names in a Go coverprofile.
	GoModule string
}

// ConvertResults converts test-result data of the named format into JUnit XML.
// format is a canonical id or alias (see [ResultFormats]). It returns the JUnit
// document, any diagnostics gathered during conversion, and an error for
// malformed or mismatched input. The returned document has already been
// validated. The input is read with a bounded reader, so an oversized stream is
// rejected rather than exhausting memory.
func ConvertResults(format string, r io.Reader) ([]byte, []Diagnostic, error) {
	adapter, err := results.Lookup(format)
	if err != nil {
		return nil, nil, err
	}
	diag := diagnostics.NewCollector()
	rep, err := adapter.Parse(r, results.Options{SourceName: "input", Diag: diag})
	if err != nil {
		return nil, collectDiagnostics(diag), err
	}
	rep.Sort()
	out, err := junit.Marshal(rep)
	if err != nil {
		return nil, collectDiagnostics(diag), err
	}
	if err := junit.Validate(bytes.NewReader(out)); err != nil {
		return nil, collectDiagnostics(diag), err
	}
	return out, collectDiagnostics(diag), nil
}

// ConvertCoverage converts coverage data of the named format into Cobertura XML.
// format is a canonical id or alias (see [CoverageFormats]). opt controls source
// path normalization. It returns the Cobertura document, any diagnostics, and an
// error for malformed input or paths that cannot be rerooted. The returned
// document has already been validated.
func ConvertCoverage(format string, r io.Reader, opt CoverageOptions) ([]byte, []Diagnostic, error) {
	adapter, err := coverage.Lookup(format)
	if err != nil {
		return nil, nil, err
	}
	diag := diagnostics.NewCollector()
	rep, err := adapter.Parse(r, coverage.Options{
		SourceName: "input",
		Paths:      pathutil.New(opt.RepoRoot, opt.GoModule),
		Diag:       diag,
	})
	if err != nil {
		return nil, collectDiagnostics(diag), err
	}
	// Normalize gives deterministic file/line ordering, matching the CLI (whose
	// single-input path normalizes via the merge step).
	if err := rep.Normalize(); err != nil {
		return nil, collectDiagnostics(diag), err
	}
	out, err := cobertura.Marshal(rep)
	if err != nil {
		return nil, collectDiagnostics(diag), err
	}
	if err := cobertura.Validate(bytes.NewReader(out)); err != nil {
		return nil, collectDiagnostics(diag), err
	}
	return out, collectDiagnostics(diag), nil
}

// ResultFormats returns every supported test-result format, in a stable order.
func ResultFormats() []FormatInfo {
	src := results.Formats()
	out := make([]FormatInfo, len(src))
	for i, f := range src {
		out[i] = FormatInfo{ID: f.ID, Aliases: f.Aliases, Description: f.Description}
	}
	return out
}

// CoverageFormats returns every supported coverage format, in a stable order.
func CoverageFormats() []FormatInfo {
	src := coverage.Formats()
	out := make([]FormatInfo, len(src))
	for i, f := range src {
		out[i] = FormatInfo{ID: f.ID, Aliases: f.Aliases, Description: f.Description}
	}
	return out
}

// collectDiagnostics converts the internal collector into the public slice.
func collectDiagnostics(diag *diagnostics.Collector) []Diagnostic {
	items := diag.Items()
	if len(items) == 0 {
		return nil
	}
	out := make([]Diagnostic, len(items))
	for i, d := range items {
		out[i] = Diagnostic{
			Severity: Severity(d.Severity),
			Code:     d.Code,
			Message:  d.Message,
			Source:   d.Source,
		}
	}
	return out
}
