// Package pathutil provides deterministic, safe normalization of source paths
// into repository-relative POSIX form. It rejects paths that escape the
// repository root and handles Windows drive letters and separators
// deterministically.
package pathutil

import (
	"fmt"
	"path"
	"regexp"
	"strings"
)

// Normalizer converts arbitrary source paths into repository-relative POSIX
// paths. A zero Normalizer (empty RepoRoot) still normalizes separators and
// rejects escaping paths, but cannot reroot absolute paths.
type Normalizer struct {
	// repoRoot is the cleaned, POSIX-form absolute repository root. It may be
	// empty when no --repo-root was supplied.
	repoRoot string
	// modulePath is an optional Go module import path used to reroot
	// import-path-style coverage entries.
	modulePath string
}

var driveLetter = regexp.MustCompile(`^[A-Za-z]:`)

// New builds a Normalizer. repoRoot and modulePath may be empty. repoRoot is
// normalized to POSIX form; a Windows drive letter is preserved as an absolute
// prefix.
func New(repoRoot, modulePath string) *Normalizer {
	return &Normalizer{
		repoRoot:   cleanAbs(repoRoot),
		modulePath: strings.Trim(strings.ReplaceAll(modulePath, "\\", "/"), "/"),
	}
}

// ModulePath returns the configured Go module import path, if any.
func (n *Normalizer) ModulePath() string { return n.modulePath }

// toSlash converts backslashes to forward slashes without collapsing them so
// callers see the intended separators.
func toSlash(p string) string { return strings.ReplaceAll(p, "\\", "/") }

// cleanAbs converts a path to POSIX form and cleans it, preserving a leading
// slash or Windows drive designator so absoluteness is retained.
func cleanAbs(p string) string {
	if p == "" {
		return ""
	}
	s := toSlash(p)
	drive := ""
	if driveLetter.MatchString(s) {
		drive = strings.ToUpper(s[:2])
		s = s[2:]
	}
	abs := strings.HasPrefix(s, "/")
	cleaned := path.Clean(s)
	if abs && !strings.HasPrefix(cleaned, "/") {
		cleaned = "/" + cleaned
	}
	return drive + cleaned
}

// isAbs reports whether a POSIX-or-Windows path is absolute.
func isAbs(s string) bool {
	return strings.HasPrefix(s, "/") || driveLetter.MatchString(s)
}

// Rel normalizes source into a repository-relative POSIX path. It returns an
// error when the resulting path escapes the repository root, when an absolute
// path is supplied without a repo root to reroot it against, or when the path
// is empty.
func (n *Normalizer) Rel(source string) (string, error) {
	if strings.TrimSpace(source) == "" {
		return "", fmt.Errorf("empty source path")
	}
	s := cleanAbs(source)

	if isAbs(s) {
		if n.repoRoot == "" {
			return "", fmt.Errorf("absolute path %q requires --repo-root to reroot", source)
		}
		rootPrefix := n.repoRoot
		if !strings.HasSuffix(rootPrefix, "/") {
			rootPrefix += "/"
		}
		switch {
		case s == n.repoRoot:
			return "", fmt.Errorf("path %q resolves to the repository root itself", source)
		case strings.HasPrefix(s, rootPrefix):
			rel := strings.TrimPrefix(s, rootPrefix)
			return validateRel(rel, source)
		default:
			return "", fmt.Errorf("path %q escapes repository root %q", source, n.repoRoot)
		}
	}

	// Relative path: reject traversal that escapes the root.
	return validateRel(strings.TrimPrefix(s, "./"), source)
}

// RelFromImportPath reroots a Go import-path-qualified file (e.g.
// github.com/example/project/pkg/file.go) into repo-relative form using the
// configured module path.
func (n *Normalizer) RelFromImportPath(importFile string) (string, error) {
	s := toSlash(strings.TrimSpace(importFile))
	if s == "" {
		return "", fmt.Errorf("empty import path")
	}
	if n.modulePath == "" {
		return "", fmt.Errorf("import-path file %q requires --go-module to reroot", importFile)
	}
	modPrefix := n.modulePath + "/"
	switch {
	case s == n.modulePath:
		return "", fmt.Errorf("import path %q has no file component", importFile)
	case strings.HasPrefix(s, modPrefix):
		return validateRel(strings.TrimPrefix(s, modPrefix), importFile)
	default:
		return "", fmt.Errorf("import path %q is outside module %q", importFile, n.modulePath)
	}
}

// validateRel cleans a relative POSIX path and rejects escaping or absolute
// results.
func validateRel(rel, original string) (string, error) {
	rel = toSlash(rel)
	if rel == "" || rel == "." {
		return "", fmt.Errorf("path %q resolves to an empty relative path", original)
	}
	cleaned := path.Clean(rel)
	if cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", fmt.Errorf("path %q escapes repository root", original)
	}
	if strings.HasPrefix(cleaned, "/") {
		return "", fmt.Errorf("path %q resolves to an absolute path", original)
	}
	return cleaned, nil
}
