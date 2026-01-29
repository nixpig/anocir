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

	defer func() {
		_ = f.Close()
		if _, statErr := os.Stat(tempFileName); statErr == nil {
			os.Remove(tempFileName)
		}
	}()

	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("write data to temp file: %w", err)
	}

	if err := f.Sync(); err != nil {
		return fmt.Errorf("flush temp file data: %w", err)
	}

	if err := f.Chmod(perm); err != nil {
		return fmt.Errorf("chmod temp file: %w", err)
	}

	if err := os.Rename(tempFileName, filename); err != nil {
		return fmt.Errorf("rename temp file: %w", err)
	}

	return nil
}
