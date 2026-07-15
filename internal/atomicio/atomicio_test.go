package atomicio

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteFileAtomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.xml")
	if err := WriteFile(path, []byte("hello"), nil); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello" {
		t.Fatalf("content = %q", data)
	}
}

func TestWriteFileValidationFailureLeavesNoFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.xml")
	err := WriteFile(path, []byte("bad"), func(_ []byte) error { return errors.New("nope") })
	if err == nil {
		t.Fatal("expected validation error")
	}
	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		t.Fatal("destination should not exist after failed validation")
	}
	// No leftover temp files.
	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Fatalf("temp files left behind: %v", entries)
	}
}

func TestWriteFileValidationFailurePreservesExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.xml")
	if err := os.WriteFile(path, []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := WriteFile(path, []byte("new"), func(_ []byte) error { return errors.New("nope") })
	if err == nil {
		t.Fatal("expected validation error")
	}
	data, _ := os.ReadFile(path)
	if string(data) != "original" {
		t.Fatalf("existing file was modified: %q", data)
	}
}
