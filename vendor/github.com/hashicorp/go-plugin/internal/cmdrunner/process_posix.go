// Copyright IBM Corp. 2016, 2025
// SPDX-License-Identifier: MPL-2.0

//go:build !windows

package cmdrunner

import (
	"os"
	"syscall"
)

// _pidAlive tests whether a process is alive or not by sending it Signal 0,
// since Go otherwise has no way to test this.
func _pidAlive(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err == nil {
		// On Linux with Go 1.23+, FindProcess opens a pidfd which must be
		// released or it leaks an FD on every call. Release errors are
		// intentionally ignored; the handle is short-lived and there's
		// nothing actionable to recover from a release failure.
		defer func() { _ = proc.Release() }()
		err = proc.Signal(syscall.Signal(0))
	}

	return err == nil
}
