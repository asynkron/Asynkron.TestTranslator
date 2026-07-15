package cli

import (
	"strings"

	"github.com/asynkron/testtranslator/internal/manifest"
)

// toolVersion is reported in generated manifests.
const toolVersion = "1.0.0"

// runManifest implements `testtranslator manifest --add KIND:FORMAT:PATH ...
// --output bundle.json`, building an optional bundle that references existing
// JUnit and Cobertura artifacts with checksums.
func runManifest(args []string, s Streams) int {
	var adds []string
	output := ""
	for i := 0; i < len(args); i++ {
		a := args[i]
		name := a
		inlineVal := ""
		hasInline := false
		if eq := strings.IndexByte(a, '='); eq >= 0 {
			name, inlineVal, hasInline = a[:eq], a[eq+1:], true
		}
		takeValue := func() (string, bool) {
			if hasInline {
				return inlineVal, true
			}
			if i+1 >= len(args) {
				return "", false
			}
			i++
			return args[i], true
		}
		switch name {
		case "--add":
			v, ok := takeValue()
			if !ok {
				return fail(s.Err, "--add requires KIND:FORMAT:PATH")
			}
			adds = append(adds, v)
		case "--output":
			v, ok := takeValue()
			if !ok {
				return fail(s.Err, "--output requires a path")
			}
			output = v
		default:
			return fail(s.Err, "unknown manifest flag %q", a)
		}
	}
	if len(adds) == 0 {
		return fail(s.Err, "manifest requires at least one --add KIND:FORMAT:PATH")
	}

	m := manifest.New(toolVersion)
	for _, spec := range adds {
		parts := strings.SplitN(spec, ":", 3)
		if len(parts) != 3 {
			return fail(s.Err, "invalid --add %q, want KIND:FORMAT:PATH", spec)
		}
		kind := manifest.Kind(parts[0])
		if kind != manifest.KindTestResults && kind != manifest.KindCoverage {
			return fail(s.Err, "invalid kind %q in --add (want test-results or coverage)", parts[0])
		}
		if err := m.AddFile(kind, parts[1], parts[2], nil, nil); err != nil {
			return fail(s.Err, "%v", err)
		}
	}
	out, err := m.Marshal()
	if err != nil {
		return fail(s.Err, "%v", err)
	}
	if err := writeOutput(output, out, manifestValidator, s); err != nil {
		return fail(s.Err, "%v", err)
	}
	return 0
}

func manifestValidator(data []byte) error {
	return manifest.Validate(data)
}
