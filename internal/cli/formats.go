package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// formatRow is a display row for a bundled adapter.
type formatRow struct {
	ID          string   `json:"id"`
	Aliases     []string `json:"aliases,omitempty"`
	Description string   `json:"description"`
}

// runFormats implements `testtranslator formats`, listing every bundled
// adapter across both pipelines.
func runFormats(args []string, s Streams) int {
	f, err := parseFlags(args)
	if err != nil {
		return fail(s.Err, "%v", err)
	}
	if f.diagnostics == "json" {
		payload := struct {
			Results  []formatRow `json:"results"`
			Coverage []formatRow `json:"coverage"`
		}{resultFormatInfos(), coverageFormatInfos()}
		enc := json.NewEncoder(s.Out)
		enc.SetIndent("", "  ")
		if err := enc.Encode(payload); err != nil {
			return fail(s.Err, "%v", err)
		}
		return 0
	}
	fmt.Fprintln(s.Out, "Test-result adapters:")
	writeFormatRows(s.Out, resultFormatInfos())
	fmt.Fprintln(s.Out)
	fmt.Fprintln(s.Out, "Coverage adapters:")
	writeFormatRows(s.Out, coverageFormatInfos())
	return 0
}

// printFormats renders a single pipeline's adapters in text or JSON.
func printFormats(kind string, rows []formatRow, mode string, s Streams) int {
	if mode == "json" {
		enc := json.NewEncoder(s.Out)
		enc.SetIndent("", "  ")
		if err := enc.Encode(rows); err != nil {
			return fail(s.Err, "%v", err)
		}
		return 0
	}
	fmt.Fprintf(s.Out, "%s adapters:\n", kind)
	writeFormatRows(s.Out, rows)
	return 0
}

func writeFormatRows(w io.Writer, rows []formatRow) {
	for _, r := range rows {
		alias := ""
		if len(r.Aliases) > 0 {
			alias = " (aliases: " + strings.Join(r.Aliases, ", ") + ")"
		}
		fmt.Fprintf(w, "  %-20s %s%s\n", r.ID, r.Description, alias)
	}
}
