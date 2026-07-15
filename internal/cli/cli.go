// Package cli implements the testtranslator command-line interface. It keeps
// CLI parsing separate from conversion logic: commands resolve adapters, read
// inputs, invoke writers/validators, and write output atomically. Diagnostics
// go to stderr (or a JSON stream); generated artifacts go to stdout or the
// requested file.
package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/asynkron/testtranslator/internal/diagnostics"
)

// Streams bundles the process IO so commands are testable without globals.
type Streams struct {
	In  io.Reader
	Out io.Writer
	Err io.Writer
}

// Run dispatches a testtranslator invocation and returns a process exit code.
// A non-zero code is preserved for malformed input, unsupported data, or
// conversion failure.
func Run(args []string, s Streams) int {
	if len(args) == 0 {
		usage(s.Err)
		return 2
	}
	cmd, rest := args[0], args[1:]
	switch cmd {
	case "results":
		return runResults(rest, s)
	case "coverage":
		return runCoverage(rest, s)
	case "validate":
		return runValidate(rest, s)
	case "manifest":
		return runManifest(rest, s)
	case "formats":
		return runFormats(rest, s)
	case "help", "-h", "--help":
		usage(s.Out)
		return 0
	case "version", "--version":
		fmt.Fprintln(s.Out, "testtranslator 1.0.0")
		return 0
	default:
		fmt.Fprintf(s.Err, "error: unknown command %q\n\n", cmd)
		usage(s.Err)
		return 2
	}
}

// fail prints an error to stderr and returns exit code 1.
func fail(err io.Writer, format string, args ...any) int {
	fmt.Fprintf(err, "error: "+format+"\n", args...)
	return 1
}

// emitDiagnostics writes collected diagnostics in the requested mode. JSON goes
// to stderr as a machine-readable stream; text is the human default. Diagnostics
// never contaminate stdout artifacts.
func emitDiagnostics(diag *diagnostics.Collector, mode string, s Streams) error {
	if diag.Len() == 0 {
		return nil
	}
	switch mode {
	case "json":
		return diag.WriteJSON(s.Err)
	default:
		return diag.WriteText(s.Err)
	}
}

func usage(w io.Writer) {
	fmt.Fprint(w, strings.TrimSpace(`
testtranslator - convert test and coverage artifacts to JUnit and Cobertura XML

Usage:
  testtranslator results  [flags]    Convert test results to JUnit XML
  testtranslator coverage [flags]    Convert coverage to Cobertura XML
  testtranslator validate junit|cobertura --input FILE
  testtranslator results formats     List test-result adapters
  testtranslator coverage formats    List coverage adapters
  testtranslator formats             List every bundled adapter
  testtranslator version

Common flags:
  --format ID              Format for inputs lacking an explicit format= prefix
  --input [ID=]PATH        Input file ("-" for stdin); repeatable
  --output PATH            Output file ("-" or omitted for stdout)
  --diagnostics text|json  Diagnostic rendering (default text)

Coverage flags:
  --repo-root DIR          Repository root for rerooting absolute paths
  --go-module PATH         Go module import path for import-path profiles
  --merge none|union       Merge mode for overlapping coverage files (default none)

Inputs are selected explicitly; there is no filename-based auto-detection.
`)+"\n")
}
