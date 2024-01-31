package k0s

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/loft-sh/vcluster/pkg/util/commandwriter"
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
	// create command
	command := &k0sCommand{}
	err := yaml.Unmarshal([]byte(os.Getenv(VClusterCommandEnv)), command)
	if err != nil {
		return fmt.Errorf("parsing k0s command %s: %w", os.Getenv(VClusterCommandEnv), err)
	}

	args := append(command.Command, command.Args...)

	// check what writer we should use
	writer, err := commandwriter.NewCommandWriter("k0s")
	if err != nil {
		return err
	}
	defer writer.Close()

	// start the command
	klog.InfoS("Starting k0s", "args", strings.Join(args, " "))
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Stdout = writer.Writer()
	cmd.Stderr = writer.Writer()
	err = cmd.Run()

	// make sure we wait for scanner to be done
	writer.CloseAndWait(ctx, err)

	// regular stop case
	if err != nil && err.Error() != "signal: killed" {
		return err
	}
	return nil
}
