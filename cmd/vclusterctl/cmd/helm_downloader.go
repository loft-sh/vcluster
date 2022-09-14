package cmd

import (
	"context"
	"fmt"
	"github.com/loft-sh/loft-util/pkg/downloader"
	"github.com/loft-sh/loft-util/pkg/downloader/commands"
	log "github.com/loft-sh/loft-util/pkg/logger"
	"os/exec"
)

const DefaultHomeVClusterFolder = ".vcluster"

// GetHelmBinaryPath checks for helm binary and downloads if it's not present.
func GetHelmBinaryPath(log log.Logger) (string, error) {
	// test for helm
	helmExecutablePath, err := exec.LookPath("helm")
	if err != nil {
		_ = fmt.Errorf("seems like helm is not installed. Helm is required for the creation of a virtual cluster")
		helmExecutablePath, err = downloader.NewDownloader(commands.NewHelmV3Command(), log, DefaultHomeVClusterFolder).EnsureCommand(context.Background())
		if err != nil {
			_ = fmt.Errorf("error while installing helm")
			return "", err
		}
	}
	return helmExecutablePath, nil
}
