package cp

import "os"

// this is a function that allows us to copy from
// distroless containers as they don't have the
// necessary util binaries to do it
func Cp(src, dest string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	bytes, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dest, bytes, info.Mode())
}
