//go:build unix

package logr

import (
	"fmt"
	"os"
	"syscall"
)

// openVerifiedRegularFile opens the log file for appending and verifies through
// the resulting descriptor that it is a regular file nothing else links to.
//
// O_NOFOLLOW refuses a symlink planted at the path, and O_NONBLOCK makes a
// planted FIFO fail with ENXIO instead of blocking forever on the write path,
// under the writer's lock, in a log directory shared with another container.
func openVerifiedRegularFile(path string) (*os.File, os.FileInfo, error) {
	fd, err := syscall.Open(path, syscall.O_APPEND|syscall.O_WRONLY|syscall.O_CLOEXEC|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, nil, &os.PathError{Op: "open", Path: path, Err: err}
	}
	file := os.NewFile(uintptr(fd), path)
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, nil, err
	}
	if err := verifyRegularFile(path, info); err != nil {
		_ = file.Close()
		return nil, nil, err
	}
	return file, info, nil
}

// verifyRegularFile rejects anything that is not an exclusively-linked regular
// file. A hard link planted in the shared log directory would redirect every
// mode change this package makes onto a file the other party controls.
func verifyRegularFile(path string, info os.FileInfo) error {
	if !info.Mode().IsRegular() {
		return fmt.Errorf("log file %q is not a regular file", path)
	}
	if stat, ok := info.Sys().(*syscall.Stat_t); ok && stat.Nlink > 1 {
		return fmt.Errorf("log file %q has %d hard links, expected exactly one", path, stat.Nlink)
	}
	return nil
}

// chmodOpenFile changes the mode through the already-verified descriptor, so
// the file whose type was checked is the file whose mode is changed.
func chmodOpenFile(file *os.File, mode os.FileMode) error {
	return file.Chmod(mode.Perm())
}
