//go:build unix

package command

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

type unixPID int

func newProcHandle(pid int) (procHandle, error) {
	return unixPID(pid), nil
}

func (pid unixPID) cmdline() ([]string, error) {
	cmdline, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(int(pid)), "cmdline"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("%w: %w", syscall.ESRCH, err)
		}
		return nil, fmt.Errorf("failed to read process cmdline: %w", err)
	}

	return strings.Split(string(cmdline), "\x00"), nil
}

func (pid unixPID) environ() ([]string, error) {
	env, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(int(pid)), "environ"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("%w: %w", syscall.ESRCH, err)
		}
		return nil, fmt.Errorf("failed to read process environ: %w", err)
	}

	return strings.Split(string(env), "\x00"), nil
}

func (pid unixPID) terminateGracefully() error {
	return syscall.Kill(int(pid), syscall.SIGTERM)
}

func (pid unixPID) terminateForcibly() error {
	return syscall.Kill(int(pid), syscall.SIGKILL)
}
