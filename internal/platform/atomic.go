package platform

import (
	"fmt"
	"log/slog"
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

	// Always close and remove temp file.
	defer func() {
		if _, err := os.Stat(f.Name()); err == nil {
			if err := f.Close(); err != nil {
				slog.Warn("failed to close temp file", "err", err)
			}
			if err := os.Remove(f.Name()); err != nil {
				slog.Warn("failed to cleanup temp file", "err", err)
			}
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

	if err := os.Rename(f.Name(), filename); err != nil {
		return fmt.Errorf("rename temp file: %w", err)
	}

	return nil
}
