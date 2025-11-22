package platform

import (
	"fmt"
	"os"
	"path/filepath"
)

// AtomicWriteFile writes data to a file atomically by operating on a temp file
// then renaming it.
func AtomicWriteFile(filename string, data []byte, perm os.FileMode) error {
	f, err := os.CreateTemp(filepath.Dir(filename), ".tmp-")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}

	tempFileName := f.Name()

	defer os.Remove(tempFileName)
	defer f.Close()

	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("write data to temp file: %w", err)
	}

	if err := f.Sync(); err != nil {
		return fmt.Errorf("flush temp file data: %w", err)
	}

	if err := f.Chmod(perm); err != nil {
		return fmt.Errorf("chmod temp file: %w", err)
	}

	if err := f.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	if err := os.Rename(tempFileName, filename); err != nil {
		return fmt.Errorf("rename temp file: %w", err)
	}

	return nil
}
