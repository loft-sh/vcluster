package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/loft-sh/utils/pkg/command"
	"github.com/loft-sh/utils/pkg/extract"
	"github.com/mitchellh/go-homedir"
	"github.com/otiai10/copy"
	"mvdan.cc/sh/v3/expand"
)

// helmCommand provides a shared implementation for helm v3 and v4.
type helmCommand struct {
	version       string
	versionPrefix string
}

func (h *helmCommand) Name() string {
	return "helm"
}

func (h *helmCommand) InstallPath(toolHomeFolder string) (string, error) {
	home, err := homedir.Dir()
	if err != nil {
		return "", err
	}

	installPath := filepath.Join(home, toolHomeFolder, "bin", h.Name())
	if runtime.GOOS == "windows" {
		installPath += ".exe"
	}

	return installPath, nil
}

func (h *helmCommand) DownloadURL() string {
	base := "https://get.helm.sh/helm-" + h.version + "-" + runtime.GOOS + "-" + runtime.GOARCH
	if runtime.GOOS == "windows" {
		return base + ".zip"
	}
	return base + ".tar.gz"
}

func (h *helmCommand) IsValid(ctx context.Context, path string) (bool, error) {
	out, err := command.Output(ctx, "", expand.ListEnviron(os.Environ()...), path, "version")
	if err != nil {
		return false, nil
	}

	return strings.Contains(string(out), h.versionPrefix), nil
}

func (h *helmCommand) Install(toolHomeFolder, archiveFile string) error {
	installPath, err := h.InstallPath(toolHomeFolder)
	if err != nil {
		return err
	}

	return installHelmBinary(extract.NewExtractor(), archiveFile, installPath, h.DownloadURL())
}

func installHelmBinary(extractor extract.Extract, archiveFile, installPath, installFromURL string) error {
	t := filepath.Dir(archiveFile)

	// Extract the binary
	if strings.HasSuffix(installFromURL, ".tar.gz") {
		err := extractor.UntarGz(archiveFile, t)
		if err != nil {
			return fmt.Errorf("extract tar.gz: %w", err)
		}
	} else if strings.HasSuffix(installFromURL, ".zip") {
		err := extractor.Unzip(archiveFile, t)
		if err != nil {
			return fmt.Errorf("extract zip: %w", err)
		}
	}

	// Copy file to target location
	binaryName := "helm"
	if runtime.GOOS == "windows" {
		binaryName = "helm.exe"
	}

	return copy.Copy(filepath.Join(t, runtime.GOOS+"-"+runtime.GOARCH, binaryName), installPath)
}
