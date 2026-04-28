package symlink

import (
	"fmt"
	"os"
	"path/filepath"
)

func writeCurrentVersion(base, version string) error {
	if err := os.WriteFile(filepath.Join(base, "current-version"), []byte(version), 0644); err != nil {
		return fmt.Errorf("write current-version: %w", err)
	}
	return nil
}
