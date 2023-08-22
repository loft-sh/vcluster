package pro

import (
	"context"
	"fmt"
	"os"
	"os/exec"
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

	cmd := exec.CommandContext(ctx, filePath, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.Dir = workingDir

	cmd.Env = append(cmd.Env, os.Environ()...)
	cmd.Env = append(cmd.Env, fmt.Sprintf("LOFT_CONFIG=%s", configFilePath))
	cmd.Env = append(cmd.Env, "LOFT_BRANDING=vcluster")

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to start vcluster pro server: %w", err)
	}

	return nil
}
