package pro

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/loft-sh/vcluster/pkg/util/cliconfig"
)

// RunLoftCli runs a loft cli command
func RunLoftCli(ctx context.Context, version string, args []string) error {
	var (
		filePath string
		err      error
	)

	if version == "" || version == "latest" {
		filePath, version, err = LatestLoftBinary(ctx)
	} else {
		filePath, err = LoftBinary(ctx, version)
	}

	if err != nil {
		return fmt.Errorf("failed to get latest loft binary: %w", err)
	}

	configFilePath, err := LoftConfigFilePath(version)
	if err != nil {
		return fmt.Errorf("failed to get loft config file path: %w", err)
	}

	workingDir, err := LoftWorkingDirectory(version)
	if err != nil {
		return fmt.Errorf("failed to get loft working directory: %w", err)
	}

	args = append([]string{"pro"}, args...)

	cmd := exec.CommandContext(ctx, filePath, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.Dir = workingDir

	cmd.Env = append(cmd.Env, os.Environ()...)
	cmd.Env = append(cmd.Env, fmt.Sprintf("LOFT_CONFIG=%s", configFilePath))
	cmd.Env = append(cmd.Env, fmt.Sprintf("LOFT_CACHE_FOLDER=%s", filepath.Join(cliconfig.VclusterFolder, VclusterProFolder)))
	cmd.Env = append(cmd.Env, "PRODUCT=vcluster-pro")

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to start vcluster pro server: %w", err)
	}

	return nil
}
