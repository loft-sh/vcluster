//go:build !unix

package command

import "fmt"

// newProcHandle is not implemented on Windows.
func newProcHandle(int) (procHandle, error) {
	return nil, fmt.Errorf("not implemented")
}
