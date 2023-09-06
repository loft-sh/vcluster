package pro

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

// RunLoftCli executes a loft CLI command for the given version with the
// provided arguments.
//
// It determines the loft binary path and sets the working directory, config
// file, and env vars needed to run the command.
//
// The CLI arguments are appended to "pro" as the first arg. The command is
// executed and any error returned.
func RunLoftCli(ctx context.Context, version string, args []string) error {
	var (
		filePath string
		err      error
	)

	if devVersion, ok := os.LookupEnv("PRO_FORCE_VERSION"); ok {
		version = devVersion
	}

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

	workingDir, err := LoftAbsoluteWorkingDirectory(version)
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
	cmd.Env = append(cmd.Env, fmt.Sprintf("LOFT_CACHE_FOLDER=%s", LoftWorkingDirectory(version)))
	cmd.Env = append(cmd.Env, "PRODUCT=vcluster-pro")

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to run vcluster pro command: %w", err)
	}

	return nil
}
