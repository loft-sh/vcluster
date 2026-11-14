package cli

import (
	"fmt"
	"os"
)

// writeOwnerOnlyFile writes data to path so that only the owner can read it. An existing
// file keeps its mode on write, so the mode is applied afterwards as well.
func writeOwnerOnlyFile(path string, data []byte) error {
	err := os.WriteFile(path, data, 0o600)
	if err != nil {
		return err
	}

	err = os.Chmod(path, 0o600)
	if err != nil {
		return fmt.Errorf("set permissions of %s: %w", path, err)
	}

	return nil
}
