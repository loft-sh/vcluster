package command

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/util/commandwriter"
	"k8s.io/klog/v2"
)

const VClusterManaged = "_VCLUSTER_MANAGED=yes"

func MergeArgs(baseArgs []string, extraArgs []string) []string {
	newArgs := []string{}
	for _, arg := range baseArgs {
		if containsFlag(extraArgs, arg) {
			continue
		}

		newArgs = append(newArgs, arg)
	}
	newArgs = append(newArgs, extraArgs...)
	return newArgs
}

func containsFlag(args []string, flag string) bool {
	for _, arg := range args {
		if !strings.HasPrefix(arg, "--") || !strings.HasPrefix(flag, "--") {
			continue
		}

		trimmedArg, _, _ := strings.Cut(arg, "=")
		trimmedFlag, _, _ := strings.Cut(flag, "=")
		if trimmedArg == trimmedFlag {
			return true
		}
	}

	return false
}

func RunCommand(ctx context.Context, command []string, component string) error {
	writer, err := commandwriter.NewCommandWriter(component, false)
	if err != nil {
		return err
	}
	defer writer.Writer()

	// maybe kill pid file
	pidFile, err := MaybeKillPidFile(command[0], component)
	if err != nil {
		return fmt.Errorf("failed to kill still running process %s: %w", pidFile, err)
	}

	// start the command
	klog.InfoS("Starting "+component, "args", strings.Join(command, " "))
	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
	cmd.Stdout = writer.Writer()
	cmd.Stderr = writer.Writer()
	cmd.Dir = constants.DataDir

	// modify environment variables
	cmd.Env = []string{}
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "PATH=") {
			env = env + string(os.PathListSeparator) + constants.BinariesDir + string(os.PathListSeparator) + "/usr/local/bin"
		}

		cmd.Env = append(cmd.Env, env)
	}
	cmd.Env = append([]string{VClusterManaged}, cmd.Env...)
	cmd.Cancel = func() error {
		err := cmd.Process.Signal(os.Interrupt)
		if err != nil {
			return fmt.Errorf("signal %s: %w", command[0], err)
		}

		state, err := cmd.Process.Wait()
		if err == nil && state.Pid() > 0 {
			time.Sleep(2 * time.Second)
		}

		err = cmd.Process.Kill()
		if err != nil {
			return fmt.Errorf("kill %s: %w", command[0], err)
		}

		return nil
	}
	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("error starting command %s: %w", command[0], err)
	}

	// write pid file
	err = os.WriteFile(pidFile, []byte(strconv.Itoa(cmd.Process.Pid)+"\n"), 0644)
	if err != nil {
		klog.ErrorS(err, "Failed to write pid file", "file", pidFile)
	}

	// wait for command to finish
	err = cmd.Wait()

	// make sure we wait for scanner to be done
	writer.CloseAndWait(ctx, err)
	if err != nil {
		return fmt.Errorf("error running command %s: %w", command[0], err)
	}

	return nil
}

// maybeKillPidFile checks kills the process in the pidFile if it's has
// the same binary as the supervisor's and also checks that the env
// `_KOS_MANAGED=yes`. This function does not delete the old pidFile as
// this is done by the caller.
func MaybeKillPidFile(binPath, component string) (string, error) {
	// pid file
	_ = os.MkdirAll(path.Join(constants.DataDir, "pids"), 0755)
	pidFile := path.Join(constants.DataDir, "pids", component) + ".pid"

	pid, err := os.ReadFile(pidFile)
	if os.IsNotExist(err) {
		return pidFile, nil
	} else if err != nil {
		return pidFile, fmt.Errorf("failed to read PID file %s: %w", pidFile, err)
	}

	p, err := strconv.Atoi(strings.TrimSuffix(string(pid), "\n"))
	if err != nil {
		return pidFile, fmt.Errorf("failed to parse PID file %s: %w", pidFile, err)
	}

	ph, err := newProcHandle(p)
	if err != nil {
		return pidFile, fmt.Errorf("cannot interact with PID %d from PID file %s: %w", p, pidFile, err)
	}

	if err := killProcess(ph, binPath); err != nil {
		return pidFile, fmt.Errorf("failed to kill PID %d from PID file %s: %w", p, pidFile, err)
	}

	return pidFile, nil
}

// Tries to terminate a process gracefully. If it's still running after
// s.TimeoutStop, the process is forcibly terminated.
func killProcess(ph procHandle, binPath string) error {
	// Kill the process pid
	deadlineTicker := time.NewTicker(5 * time.Second)
	defer deadlineTicker.Stop()
	checkTicker := time.NewTicker(200 * time.Millisecond)
	defer checkTicker.Stop()

Loop:
	for {
		select {
		case <-checkTicker.C:
			shouldKill, err := shouldKillProcess(ph, binPath)
			if err != nil {
				return err
			}
			if !shouldKill {
				return nil
			}

			err = ph.terminateGracefully()
			if errors.Is(err, syscall.ESRCH) {
				return nil
			} else if err != nil {
				return fmt.Errorf("failed to terminate gracefully: %w", err)
			}
		case <-deadlineTicker.C:
			break Loop
		}
	}

	shouldKill, err := shouldKillProcess(ph, binPath)
	if err != nil {
		return err
	}
	if !shouldKill {
		return nil
	}

	err = ph.terminateForcibly()
	if errors.Is(err, syscall.ESRCH) {
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to terminate forcibly: %w", err)
	}
	return nil
}

func shouldKillProcess(ph procHandle, binPath string) (bool, error) {
	// only kill process if it has the expected cmd
	if cmd, err := ph.cmdline(); err != nil {
		if errors.Is(err, syscall.ESRCH) {
			return false, nil
		}
		return false, err
	} else if len(cmd) > 0 && cmd[0] != binPath {
		return false, nil
	}

	// only kill process if it has the _VCLUSTER_MANAGED env set
	if env, err := ph.environ(); err != nil {
		if errors.Is(err, syscall.ESRCH) {
			return false, nil
		}
		return false, err
	} else if !slices.Contains(env, VClusterManaged) {
		return false, nil
	}

	return true, nil
}
