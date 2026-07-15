// Package manifest implements the optional bundle manifest that references
// JUnit and Cobertura artifacts without replacing either format. It records
// source formats, the conversion tool version, warnings, and content checksums
// so a consumer can locate and verify every artifact.
package manifest

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

// SchemaVersion is the stable identifier for this manifest layout.
const SchemaVersion = "test-artifacts.v1"

// Kind classifies an artifact.
type Kind string

const (
	KindTestResults Kind = "test-results"
	KindCoverage    Kind = "coverage"
)

// Artifact references one generated file.
type Artifact struct {
	Kind Kind `json:"kind"`
	// Format is the output format ("junit" or "cobertura").
	Format string `json:"format"`
	// Path is the artifact location, relative to the manifest.
	Path string `json:"path"`
	// SourceFormats lists the adapter ids that produced this artifact.
	SourceFormats []string `json:"source_formats,omitempty"`
	// SHA256 is the hex content checksum.
	SHA256 string `json:"sha256"`
	// Bytes is the artifact size.
	Bytes int64 `json:"bytes"`
	// Warnings are diagnostics codes recorded during conversion.
	Warnings []string `json:"warnings,omitempty"`
}

// Manifest is the bundle document.
type Manifest struct {
	SchemaVersion string     `json:"schema_version"`
	ToolVersion   string     `json:"tool_version"`
	Artifacts     []Artifact `json:"artifacts"`
}

// New builds an empty manifest for the given tool version.
func New(toolVersion string) *Manifest {
	return &Manifest{SchemaVersion: SchemaVersion, ToolVersion: toolVersion}
}

// AddFile hashes path, records its size, and appends an artifact entry.
func (m *Manifest) AddFile(kind Kind, format, path string, sourceFormats, warnings []string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("manifest: read artifact %q: %w", path, err)
	}
	sum := sha256.Sum256(data)
	sf := append([]string(nil), sourceFormats...)
	sort.Strings(sf)
	w := append([]string(nil), warnings...)
	sort.Strings(w)
	m.Artifacts = append(m.Artifacts, Artifact{
		Kind:          kind,
		Format:        format,
		Path:          path,
		SourceFormats: sf,
		SHA256:        hex.EncodeToString(sum[:]),
		Bytes:         int64(len(data)),
		Warnings:      w,
	})
	return nil
}

// Marshal renders the manifest as deterministic, indented JSON. Artifacts are
// sorted by (kind, path) for stable output.
func (m *Manifest) Marshal() ([]byte, error) {
	sort.SliceStable(m.Artifacts, func(i, j int) bool {
		if m.Artifacts[i].Kind != m.Artifacts[j].Kind {
			return m.Artifacts[i].Kind < m.Artifacts[j].Kind
		}
		return m.Artifacts[i].Path < m.Artifacts[j].Path
	})
	out, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("manifest: marshal: %w", err)
	}
	return append(out, '\n'), nil
}

// Validate checks a manifest document for structural soundness.
func Validate(data []byte) error {
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return fmt.Errorf("manifest: invalid JSON: %w", err)
	}
	if m.SchemaVersion != SchemaVersion {
		return fmt.Errorf("manifest: unsupported schema_version %q", m.SchemaVersion)
	}
	for i, a := range m.Artifacts {
		if a.Kind != KindTestResults && a.Kind != KindCoverage {
			return fmt.Errorf("manifest: artifact %d has invalid kind %q", i, a.Kind)
		}
		if a.Path == "" {
			return fmt.Errorf("manifest: artifact %d has empty path", i)
		}
		if len(a.SHA256) != 64 {
			return fmt.Errorf("manifest: artifact %d has invalid sha256", i)
		}
	}
	return nil
}
