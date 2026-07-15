package manifest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildAndValidate(t *testing.T) {
	dir := t.TempDir()
	junitPath := filepath.Join(dir, "results.junit.xml")
	covPath := filepath.Join(dir, "coverage.cobertura.xml")
	if err := os.WriteFile(junitPath, []byte("<testsuites/>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(covPath, []byte("<coverage/>"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := New("1.0.0")
	if err := m.AddFile(KindCoverage, "cobertura", covPath, []string{"go-coverprofile"}, nil); err != nil {
		t.Fatal(err)
	}
	if err := m.AddFile(KindTestResults, "junit", junitPath, []string{"go-test-json"}, []string{"gotest.incomplete"}); err != nil {
		t.Fatal(err)
	}
	data, err := m.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	// Sorted by kind: coverage before test-results.
	if strings.Index(s, "coverage") > strings.Index(s, "test-results") {
		t.Error("artifacts not sorted by kind")
	}
	if !strings.Contains(s, `"schema_version": "test-artifacts.v1"`) {
		t.Error("missing schema version")
	}
	if err := Validate(data); err != nil {
		t.Fatalf("Validate rejected our own manifest: %v", err)
	}
}

func TestValidateRejectsBadSchema(t *testing.T) {
	if err := Validate([]byte(`{"schema_version":"wrong","artifacts":[]}`)); err == nil {
		t.Fatal("expected error for wrong schema version")
	}
}
