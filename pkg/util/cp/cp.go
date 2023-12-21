package cp

import (
	"io"
	"os"
)

// this is a function that allows us to copy from
// distroless containers as they don't have the
// necessary util binaries to do it
func Cp(src, dest string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()
	destFile, err := os.Create(dest)
	if err != nil {
		return err
	}
	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return err
	}
	err = destFile.Close()
	if err != nil {
		return err
	}

	return os.Chmod(dest, info.Mode())
}
