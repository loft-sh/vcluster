package cli

import (
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
)

func TestWriteOwnerOnlyFile(t *testing.T) {
	dir := t.TempDir()

	t.Run("new file", func(t *testing.T) {
		path := filepath.Join(dir, "new.yaml")
		assert.NilError(t, writeOwnerOnlyFile(path, []byte("secret")))

		info, err := os.Stat(path)
		assert.NilError(t, err)
		assert.Equal(t, info.Mode().Perm(), os.FileMode(0o600))
	})

	t.Run("existing world readable file", func(t *testing.T) {
		path := filepath.Join(dir, "existing.yaml")
		assert.NilError(t, os.WriteFile(path, []byte("old"), 0o644))
		assert.NilError(t, writeOwnerOnlyFile(path, []byte("secret")))

		info, err := os.Stat(path)
		assert.NilError(t, err)
		assert.Equal(t, info.Mode().Perm(), os.FileMode(0o600))
		content, err := os.ReadFile(path)
		assert.NilError(t, err)
		assert.Equal(t, string(content), "secret")
	})
}
