package logr

import (
	"os"
)

// statRegularFile reads the active log file's metadata without opening it, so
// neither a FIFO nor a symlink planted at the path can block this call or
// redirect it to another file.
func statRegularFile(path string) (os.FileInfo, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if err := verifyRegularFile(path, info); err != nil {
		return nil, err
	}
	return info, nil
}

// chmodRegularFile restores the configured mode on the active log file,
// rejecting anything that is not a regular file the process alone links to.
func chmodRegularFile(path string, mode os.FileMode) error {
	file, _, err := openVerifiedRegularFile(path)
	if err != nil {
		return err
	}
	if err := chmodOpenFile(file, mode); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}
