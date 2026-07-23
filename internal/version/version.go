// Package version resolves the version reported by the CLI and generated
// manifests from linker or Go module build metadata.
package version

import (
	"runtime/debug"
	"strings"
)

// buildVersion can be set by release builds with:
//
//	-ldflags "-X github.com/asynkron/Asynkron.TestTranslator/internal/version.buildVersion=v1.2.3"
var buildVersion string

// Current returns a release version without a leading "v". Local and other
// unversioned builds report "devel" instead of claiming a release they are not.
func Current() string {
	if buildVersion != "" {
		return normalize(buildVersion)
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		if value := info.Main.Version; value != "" && value != "(devel)" {
			return normalize(value)
		}
	}
	return "devel"
}

func normalize(value string) string {
	return strings.TrimPrefix(strings.TrimSpace(value), "v")
}
