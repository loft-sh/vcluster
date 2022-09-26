package downloader

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/loft-sh/utils/pkg/downloader/commands"
	"github.com/loft-sh/utils/pkg/log"

	"github.com/pkg/errors"
)

type Downloader interface {
	EnsureCommand(ctx context.Context) (string, error)
}

type downloader struct {
	httpGet        getRequest
	command        commands.Command
	log            log.Logger
	toolHomeFolder string
}

func NewDownloader(command commands.Command, log log.Logger, toolHomeFolder string) Downloader {
	return &downloader{
		httpGet:        http.Get,
		command:        command,
		log:            log,
		toolHomeFolder: toolHomeFolder,
	}
}

func (d *downloader) EnsureCommand(ctx context.Context) (string, error) {
	command := d.command.Name()
	valid, err := d.command.IsValid(ctx, command)
	if err != nil {
		return "", err
	} else if valid {
		return command, nil
	}

	installPath, err := d.command.InstallPath(d.toolHomeFolder)
	if err != nil {
		return "", err
	}

	valid, err = d.command.IsValid(ctx, installPath)
	if err != nil {
		return "", err
	} else if valid {
		return installPath, nil
	}

	return installPath, d.downloadExecutable(command, installPath, d.command.DownloadURL())
}

func (d *downloader) downloadExecutable(command, installPath, installFromURL string) error {
	err := os.MkdirAll(filepath.Dir(installPath), 0755)
	if err != nil {
		return err
	}

	err = d.downloadFile(command, installPath, installFromURL)
	if err != nil {
		return errors.Wrap(err, "download file")
	}

	err = os.Chmod(installPath, 0755)
	if err != nil {
		return errors.Wrap(err, "cannot make file executable")
	}

	return nil
}

type getRequest func(url string) (*http.Response, error)

func (d *downloader) downloadFile(command, installPath, installFromURL string) error {
	d.log.Info("Downloading " + command + "...")

	t, err := os.MkdirTemp("", "")
	if err != nil {
		return err
	}
	defer func(path string) {
		_ = os.RemoveAll(path)
	}(t)

	archiveFile := filepath.Join(t, "download")
	f, err := os.Create(archiveFile)
	if err != nil {
		return err
	}
	defer func(f *os.File) {
		_ = f.Close()
	}(f)

	resp, err := d.httpGet(installFromURL)
	if err != nil {
		return errors.Wrap(err, "get url")
	}

	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	_, err = io.Copy(f, resp.Body)
	if err != nil {
		return errors.Wrap(err, "download file")
	}

	err = f.Close()
	if err != nil {
		return err
	}

	// install the file in toolHomeFolder
	return d.command.Install(d.toolHomeFolder, archiveFile)
}
