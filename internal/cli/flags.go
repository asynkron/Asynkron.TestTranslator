package cli

import (
	"fmt"
	"regexp"
	"strings"
)

// inputSpec is one explicitly typed input.
type inputSpec struct {
	format string
	path   string // "-" means stdin
}

// commonFlags holds parsed flags shared by results and coverage commands.
type commonFlags struct {
	format      string
	inputs      []inputSpec
	output      string // "" or "-" means stdout
	diagnostics string
	// coverage-only
	repoRoot string
	goModule string
	merge    string
	// positional leftovers (e.g. "formats")
	positional []string
}

var formatID = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

// parseFlags parses the shared flag grammar. Unknown flags are an error so
// mistakes fail loudly rather than being silently ignored.
func parseFlags(args []string) (*commonFlags, error) {
	f := &commonFlags{diagnostics: "text", merge: "none"}
	for i := 0; i < len(args); i++ {
		a := args[i]
		takeValue := func() (string, error) {
			if eq := strings.IndexByte(a, '='); eq >= 0 {
				return a[eq+1:], nil
			}
			if i+1 >= len(args) {
				return "", fmt.Errorf("flag %q requires a value", a)
			}
			i++
			return args[i], nil
		}
		name := a
		if eq := strings.IndexByte(a, '='); eq >= 0 {
			name = a[:eq]
		}
		switch name {
		case "--format":
			v, err := takeValue()
			if err != nil {
				return nil, err
			}
			f.format = v
		case "--input":
			v, err := takeValue()
			if err != nil {
				return nil, err
			}
			spec, err := parseInputSpec(v, f.format)
			if err != nil {
				return nil, err
			}
			f.inputs = append(f.inputs, spec)
		case "--output":
			v, err := takeValue()
			if err != nil {
				return nil, err
			}
			f.output = v
		case "--diagnostics":
			v, err := takeValue()
			if err != nil {
				return nil, err
			}
			if v != "text" && v != "json" {
				return nil, fmt.Errorf("--diagnostics must be text or json, got %q", v)
			}
			f.diagnostics = v
		case "--repo-root":
			v, err := takeValue()
			if err != nil {
				return nil, err
			}
			f.repoRoot = v
		case "--go-module":
			v, err := takeValue()
			if err != nil {
				return nil, err
			}
			f.goModule = v
		case "--merge":
			v, err := takeValue()
			if err != nil {
				return nil, err
			}
			if v != "none" && v != "union" {
				return nil, fmt.Errorf("--merge must be none or union, got %q", v)
			}
			f.merge = v
		default:
			if strings.HasPrefix(a, "-") {
				return nil, fmt.Errorf("unknown flag %q", a)
			}
			f.positional = append(f.positional, a)
		}
	}
	// Backfill format for inputs that had none and were parsed before --format
	// appeared later on the command line.
	for idx := range f.inputs {
		if f.inputs[idx].format == "" {
			f.inputs[idx].format = f.format
		}
	}
	return f, nil
}

// parseInputSpec parses "[format=]path". The prefix is treated as a format only
// when it is a valid format identifier, so paths containing "=" are preserved.
func parseInputSpec(v, defaultFormat string) (inputSpec, error) {
	if v == "" {
		return inputSpec{}, fmt.Errorf("empty --input value")
	}
	if eq := strings.IndexByte(v, '='); eq > 0 {
		left := v[:eq]
		right := v[eq+1:]
		if formatID.MatchString(left) && !strings.ContainsAny(left, "/\\") {
			if right == "" {
				return inputSpec{}, fmt.Errorf("input %q has empty path", v)
			}
			return inputSpec{format: left, path: right}, nil
		}
	}
	return inputSpec{format: defaultFormat, path: v}, nil
}
