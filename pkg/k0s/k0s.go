package k0s

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/loft-sh/log/scanner"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"k8s.io/klog/v2"
)

const VClusterCommandEnv = "VCLUSTER_COMMAND"

type k0sCommand struct {
	Command []string `json:"command,omitempty"`
	Args    []string `json:"args,omitempty"`
}

const runDir = "/run/k0s"

func StartK0S(ctx context.Context) error {
	// make sure we delete the contents of /run/k0s
	dirEntries, _ := os.ReadDir(runDir)
	for _, entry := range dirEntries {
		_ = os.RemoveAll(filepath.Join(runDir, entry.Name()))
	}

	// start k0s binary
	reader, writer, err := os.Pipe()
	if err != nil {
		return err
	}
	defer writer.Close()

	command := &k0sCommand{}
	err = yaml.Unmarshal([]byte(os.Getenv(VClusterCommandEnv)), command)
	if err != nil {
		return fmt.Errorf("parsing k0s command %s: %w", os.Getenv(VClusterCommandEnv), err)
	}

	args := append(command.Command, command.Args...)

	// start func
	done := make(chan struct{})
	go func() {
		defer close(done)

		// make sure we scan the output correctly
		scan := scanner.NewScanner(reader)
		for scan.Scan() {
			line := scan.Text()
			if len(line) == 0 {
				continue
			}

			// print to our logs
			args := []interface{}{"component", "k0s"}
			loghelper.PrintKlogLine(line, args)
		}
	}()

	// start the command
	klog.InfoS("Starting k0s", "args", strings.Join(args, " "))
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Stdout = writer
	cmd.Stderr = writer
	err = cmd.Run()

	// make sure we wait for scanner to be done
	_ = writer.Close()
	<-done

	// regular stop case
	if err != nil && err.Error() != "signal: killed" {
		return err
	}
	return nil
}
