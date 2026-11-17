package commands

import "context"

type Command interface {
	// Name is the name of the command, e.g. helm or kubectl
	Name() string

	// InstallPath is the path where this command should get installed, if
	// it is not found in the PATH environment variable
	InstallPath(toolHomeFolder string) (string, error)

	// DownloadURL is the url where the command binary or archive can be downloaded
	// from.
	DownloadURL() string

	// IsValid checks if the command at the given path exists.
	IsValid(ctx context.Context, path string) (bool, error)

	// Install installs the command after it was downloaded from the DownloadURL() in toolHomeFolder
	Install(toolHomeFolder string, archiveFile string) error
}
