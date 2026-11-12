package snapshot

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
)

func TestBackupFolderRecreatesPrivateDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "etcd")
	assert.NilError(t, os.MkdirAll(filepath.Join(dir, "member"), 0700))
	assert.NilError(t, os.WriteFile(filepath.Join(dir, "member", "db"), []byte("old"), 0600))

	assert.NilError(t, backupFolder(context.Background(), dir))

	info, err := os.Stat(dir)
	assert.NilError(t, err)
	assert.Equal(t, info.Mode().Perm(), os.FileMode(0700))

	_, err = os.Stat(filepath.Join(dir+".backup", "member", "db"))
	assert.NilError(t, err, "the previous data directory is kept as a backup")
}
