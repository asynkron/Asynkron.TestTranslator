// Package atomicio writes files atomically so a failed conversion cannot leave
// a partial or corrupt destination file. Content is written to a temporary file
// in the destination directory and renamed into place only after a successful
// validation callback.
package atomicio

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Validator inspects the fully-written bytes before they replace the
// destination. Returning an error aborts the write and removes the temp file.
type Validator func(data []byte) error

// WriteFile writes data to path atomically. If validate is non-nil it must
// succeed before the temp file is renamed over path. The temp file is always
// cleaned up on failure.
func WriteFile(path string, data []byte, validate Validator) error {
	if validate != nil {
		if err := validate(data); err != nil {
			return fmt.Errorf("atomicio: validation failed, destination %q left unchanged: %w", path, err)
		}
	}
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".testtranslator-*.tmp")
	if err != nil {
		return fmt.Errorf("atomicio: create temp file: %w", err)
	}
	tmpName := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpName) }

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("atomicio: write temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("atomicio: sync temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("atomicio: close temp file: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		cleanup()
		return fmt.Errorf("atomicio: rename into place: %w", err)
	}
	return nil
}

// WriteTo writes data to w, validating first when validate is non-nil. It is
// used for stdout output, which cannot be written atomically but must still be
// validated before emission.
func WriteTo(w io.Writer, data []byte, validate Validator) error {
	if validate != nil {
		if err := validate(data); err != nil {
			return fmt.Errorf("atomicio: validation failed, nothing written: %w", err)
		}
	}
	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("atomicio: write: %w", err)
	}
	return nil
}
