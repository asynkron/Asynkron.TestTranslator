package cli

import (
	"bytes"
	"fmt"
	"os"

	"github.com/asynkron/testtranslator/internal/atomicio"
	"github.com/asynkron/testtranslator/internal/diagnostics"
	"github.com/asynkron/testtranslator/internal/results"
	"github.com/asynkron/testtranslator/internal/results/junit"
)

// runResults implements `testtranslator results`.
func runResults(args []string, s Streams) int {
	f, err := parseFlags(args)
	if err != nil {
		return fail(s.Err, "%v", err)
	}
	if len(f.positional) == 1 && f.positional[0] == "formats" {
		return listResultFormats(f.diagnostics, s)
	}
	if len(f.positional) > 0 {
		return fail(s.Err, "unexpected argument %q", f.positional[0])
	}
	if len(f.inputs) == 0 {
		return fail(s.Err, "results requires at least one --input")
	}

	diag := diagnostics.NewCollector()
	merged := &results.Report{Name: "testtranslator"}
	for _, in := range f.inputs {
		if in.format == "" {
			return fail(s.Err, "input %q has no format; pass --format or use format=path", in.path)
		}
		adapter, err := results.Lookup(in.format)
		if err != nil {
			return fail(s.Err, "%v", err)
		}
		data, name, err := readInput(in.path, s)
		if err != nil {
			return fail(s.Err, "%v", err)
		}
		rep, err := adapter.Parse(bytes.NewReader(data), results.Options{SourceName: name, Diag: diag})
		if err != nil {
			return fail(s.Err, "%v", err)
		}
		merged.Suites = append(merged.Suites, rep.Suites...)
	}
	merged.Sort()

	out, err := junit.Marshal(merged)
	if err != nil {
		return fail(s.Err, "%v", err)
	}
	if err := writeOutput(f.output, out, junitValidator, s); err != nil {
		return fail(s.Err, "%v", err)
	}
	if err := emitDiagnostics(diag, f.diagnostics, s); err != nil {
		return fail(s.Err, "%v", err)
	}
	return 0
}

// junitValidator validates generated JUnit bytes before they replace the
// destination.
func junitValidator(data []byte) error {
	return junit.Validate(bytes.NewReader(data))
}

func listResultFormats(mode string, s Streams) int {
	return printFormats("test-result", resultFormatInfos(), mode, s)
}

func resultFormatInfos() []formatRow {
	var rows []formatRow
	for _, fi := range results.Formats() {
		rows = append(rows, formatRow{ID: fi.ID, Aliases: fi.Aliases, Description: fi.Description})
	}
	return rows
}

// readInput reads a file or stdin ("-"), returning the bytes and a label.
func readInput(path string, s Streams) ([]byte, string, error) {
	if path == "-" || path == "" {
		data, err := readAll(s.In)
		if err != nil {
			return nil, "", fmt.Errorf("read stdin: %w", err)
		}
		return data, "<stdin>", nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, "", fmt.Errorf("read %q: %w", path, err)
	}
	defer f.Close()
	// Bound file input the same way stdin is bounded, so a very large file cannot
	// be slurped into memory before an adapter's own limit applies.
	data, err := readAll(f)
	if err != nil {
		return nil, "", fmt.Errorf("read %q: %w", path, err)
	}
	return data, path, nil
}

// writeOutput writes to stdout ("-"/"") or atomically to a file, validating
// first.
func writeOutput(path string, data []byte, validate atomicio.Validator, s Streams) error {
	if path == "-" || path == "" {
		return atomicio.WriteTo(s.Out, data, validate)
	}
	return atomicio.WriteFile(path, data, validate)
}
