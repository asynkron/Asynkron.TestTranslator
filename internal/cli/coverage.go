package cli

import (
	"bytes"

	"github.com/asynkron/Asynkron.TestTranslator/internal/coverage"
	"github.com/asynkron/Asynkron.TestTranslator/internal/coverage/cobertura"
	"github.com/asynkron/Asynkron.TestTranslator/internal/diagnostics"
	"github.com/asynkron/Asynkron.TestTranslator/internal/pathutil"
)

// runCoverage implements `testtranslator coverage`.
func runCoverage(args []string, s Streams) int {
	f, err := parseFlags(args)
	if err != nil {
		return fail(s.Err, "%v", err)
	}
	if len(f.positional) == 1 && f.positional[0] == "formats" {
		return listCoverageFormats(f.diagnostics, s)
	}
	if len(f.positional) > 0 {
		return fail(s.Err, "unexpected argument %q", f.positional[0])
	}
	if len(f.inputs) == 0 {
		return fail(s.Err, "coverage requires at least one --input")
	}

	diag := diagnostics.NewCollector()
	norm := pathutil.New(f.repoRoot, f.goModule)

	var reports []*coverage.Report
	for _, in := range f.inputs {
		if in.format == "" {
			return fail(s.Err, "input %q has no format; pass --format or use format=path", in.path)
		}
		adapter, err := coverage.Lookup(in.format)
		if err != nil {
			return fail(s.Err, "%v", err)
		}
		data, name, err := readInput(in.path, s)
		if err != nil {
			return fail(s.Err, "%v", err)
		}
		rep, err := adapter.Parse(bytes.NewReader(data), coverage.Options{
			SourceName: name,
			Paths:      norm,
			Diag:       diag,
		})
		if err != nil {
			return fail(s.Err, "%v", err)
		}
		reports = append(reports, rep)
	}

	merged, err := coverage.Merge(reports, f.merge, diag)
	if err != nil {
		return fail(s.Err, "%v", err)
	}

	out, err := cobertura.Marshal(merged)
	if err != nil {
		return fail(s.Err, "%v", err)
	}
	if err := writeOutput(f.output, out, coberturaValidator, s); err != nil {
		return fail(s.Err, "%v", err)
	}
	if err := emitDiagnostics(diag, f.diagnostics, s); err != nil {
		return fail(s.Err, "%v", err)
	}
	return 0
}

func coberturaValidator(data []byte) error {
	return cobertura.Validate(bytes.NewReader(data))
}

func listCoverageFormats(mode string, s Streams) int {
	return printFormats("coverage", coverageFormatInfos(), mode, s)
}

func coverageFormatInfos() []formatRow {
	var rows []formatRow
	for _, fi := range coverage.Formats() {
		rows = append(rows, formatRow{ID: fi.ID, Aliases: fi.Aliases, Description: fi.Description})
	}
	return rows
}
