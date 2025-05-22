package command

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/loft-sh/vcluster/pkg/util/commandwriter"
	"k8s.io/klog/v2"
)

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

	// start the command
	klog.InfoS("Starting "+component, "args", strings.Join(command, " "))
	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
	cmd.Stdout = writer.Writer()
	cmd.Stderr = writer.Writer()
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
	err = cmd.Run()

	// make sure we wait for scanner to be done
	writer.CloseAndWait(ctx, err)
	return fmt.Errorf("error running command %s: %w", command[0], err)
}
