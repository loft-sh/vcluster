package fs

import (
	"io/fs"
	"os"
	"time"

	"github.com/vladimirvivien/gexe/vars"
)

type FSInfo struct {
	err  error
	path string
	mode os.FileMode
	info os.FileInfo
	vars *vars.Variables
}

// Err returns last opertion error on the directory
func (i *FSInfo) Err() error {
	return i.err
}

// Path is the original path for the directory
func (i *FSInfo) Path() string {
	return i.path
}

func (i *FSInfo) Name() string {
	return i.info.Name()
}

// Mode returns the fs.FileMode for the directory
func (i *FSInfo) Mode() fs.FileMode {
	return i.mode
}

// Size returns the directory size or -1 if not known or error
func (i *FSInfo) Size() int64 {
	return i.info.Size()
}

// IsDir returns true if path points to a directory
func (i *FSInfo) IsDir() bool {
	return i.info.IsDir()
}

// ModTime returns the last know modification time.
func (i *FSInfo) ModTime() time.Time {
	return i.info.ModTime()
}
