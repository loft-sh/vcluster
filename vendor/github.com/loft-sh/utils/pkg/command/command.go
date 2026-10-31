package command

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"mvdan.cc/sh/v3/expand"
)

// streamCommand is the command whose output is streamed to a log
type streamCommand struct {
	cmd         *exec.Cmd
	killTimeout time.Duration
}

// newStreamCommand creates a new stream command
func newStreamCommand(command string, args []string) *streamCommand {
	return &streamCommand{
		cmd:         exec.Command(command, args...),
		killTimeout: time.Second * 2,
	}
}

func ListVars(environ expand.Environ) map[string]string {
	variables := map[string]string{}
	environ.Each(func(name string, vr expand.Variable) bool {
		if vr.Kind == expand.String && vr.Str != "" {
			variables[name] = vr.Str
		}
		return true
	})
	return variables
}

// RunWithEnv runs a stream command
func (s *streamCommand) RunWithEnv(ctx context.Context, dir string, environ expand.Environ, stdout io.Writer, stderr io.Writer, stdin io.Reader) error {
	s.cmd.Dir = dir
	env := []string{}
	for k, v := range ListVars(environ) {
		env = append(env, k+"="+v)
	}

	s.cmd.Env = env
	if stdout != nil {
		s.cmd.Stdout = stdout
	}

	var defaultStderr *prefixSuffixSaver
	if stderr != nil {
		s.cmd.Stderr = stderr
	} else {
		defaultStderr = &prefixSuffixSaver{N: 32 << 10}
		s.cmd.Stderr = defaultStderr
	}

	if stdin != nil {
		s.cmd.Stdin = stdin
	}

	var err error
	err = s.cmd.Start()
	if err == nil {
		if done := ctx.Done(); done != nil {
			go func() {
				<-done

				if s.killTimeout <= 0 || runtime.GOOS == "windows" {
					_ = s.cmd.Process.Signal(os.Kill)
					return
				}

				// TODO: don't temporarily leak this goroutine
				// if the program stops itself with the
				// interrupt.
				go func() {
					time.Sleep(s.killTimeout)
					_ = s.cmd.Process.Signal(os.Kill)
				}()
				_ = s.cmd.Process.Signal(os.Interrupt)
			}()
		}

		err = s.cmd.Wait()
	}
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && defaultStderr != nil {
			exitErr.Stderr = defaultStderr.Bytes()
		}

		return err
	}

	return nil
}

// Run runs a stream command
func (s *streamCommand) Run(ctx context.Context, dir string, stdout io.Writer, stderr io.Writer, stdin io.Reader) error {
	return s.RunWithEnv(ctx, dir, expand.ListEnviron(os.Environ()...), stdout, stderr, stdin)
}

func ShouldExecuteOnOS(os string) bool {
	// if the operating system is set and the current is not specified
	// we skip the hook
	if os != "" {
		found := false
		oss := strings.Split(os, ",")
		for _, os := range oss {
			if strings.TrimSpace(os) == runtime.GOOS {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

func Command(ctx context.Context, dir string, environ expand.Environ, stdout io.Writer, stderr io.Writer, stdin io.Reader, cmd string, args ...string) error {
	err := newStreamCommand(cmd, args).RunWithEnv(ctx, dir, environ, stdout, stderr, stdin)
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return fmt.Errorf("error executing '%s %s': %s", cmd, strings.Join(args, " "), string(exitErr.Stderr))
		}

		return err
	}

	return nil
}

func CombinedOutput(ctx context.Context, dir string, environ expand.Environ, cmd string, args ...string) ([]byte, error) {
	stdout := &bytes.Buffer{}
	err := Command(ctx, dir, environ, stdout, stdout, nil, cmd, args...)
	return stdout.Bytes(), err
}

func Output(ctx context.Context, dir string, environ expand.Environ, cmd string, args ...string) ([]byte, error) {
	stdout := &bytes.Buffer{}
	err := Command(ctx, dir, environ, stdout, nil, nil, cmd, args...)
	return stdout.Bytes(), err
}

func FormatCommandName(cmd string, args []string) string {
	commandString := strings.TrimSpace(cmd + " " + strings.Join(args, " "))
	splitted := strings.Split(commandString, "\n")
	if len(splitted) > 1 {
		return splitted[0] + "..."
	}

	return commandString
}
