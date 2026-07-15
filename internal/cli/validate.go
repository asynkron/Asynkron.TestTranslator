package cli

import (
	"bytes"
	"fmt"

	"github.com/asynkron/Asynkron.TestTranslator/internal/coverage/cobertura"
	"github.com/asynkron/Asynkron.TestTranslator/internal/results/junit"
)

// runValidate implements `testtranslator validate junit|cobertura --input FILE`.
func runValidate(args []string, s Streams) int {
	if len(args) == 0 {
		return fail(s.Err, "validate requires a kind: junit or cobertura")
	}
	kind := args[0]
	f, err := parseFlags(args[1:])
	if err != nil {
		return fail(s.Err, "%v", err)
	}
	// Allow the file as either --input or a positional argument.
	var path string
	switch {
	case len(f.inputs) == 1:
		path = f.inputs[0].path
	case len(f.positional) == 1:
		path = f.positional[0]
	default:
		return fail(s.Err, "validate %s requires exactly one --input FILE", kind)
	}
	data, name, err := readInput(path, s)
	if err != nil {
		return fail(s.Err, "%v", err)
	}
	switch kind {
	case "junit":
		if err := junit.Validate(bytes.NewReader(data)); err != nil {
			return fail(s.Err, "%v", err)
		}
	case "cobertura":
		if err := cobertura.Validate(bytes.NewReader(data)); err != nil {
			return fail(s.Err, "%v", err)
		}
	default:
		return fail(s.Err, "unknown validate kind %q (want junit or cobertura)", kind)
	}
	fmt.Fprintf(s.Out, "%s: %s is valid\n", kind, name)
	return 0
}
