//go:build !unix

package logr

import (
	"fmt"
	"os"
)

// This file exists so the package compiles for non-unix consumers -- the CLI
// that imports it ships Windows binaries -- not because the security model
// ports.
// The unix build's O_NOFOLLOW and O_NONBLOCK have no portable
// equivalent, so this checks with Lstat and acts by name: a deliberate
// check-then-use gap, guarding a sidecar guarantee that is unix-only anyway.
// Rejecting a non-regular file still holds, which keeps the logger out of a
// pipe or device.

// openVerifiedRegularFile opens the log file for appending after rejecting a
// symlink or any other non-regular file at the path.
func openVerifiedRegularFile(path string) (*os.File, os.FileInfo, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, nil, err
	}
	if err := verifyRegularFile(path, info); err != nil {
		return nil, nil, err
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		return nil, nil, err
	}
	return file, info, nil
}

// verifyRegularFile rejects anything that is not a regular file. Link counts
// are not portably available outside unix, so they are not checked here.
func verifyRegularFile(path string, info os.FileInfo) error {
	if !info.Mode().IsRegular() {
		return fmt.Errorf("log file %q is not a regular file", path)
	}
	return nil
}

// chmodOpenFile changes the mode by name, because (*os.File).Chmod is not
// available on every non-unix target.
func chmodOpenFile(file *os.File, mode os.FileMode) error {
	return os.Chmod(file.Name(), mode.Perm())
}
