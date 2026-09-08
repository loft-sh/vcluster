package credits

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"strings"

	"github.com/loft-sh/log"
	"github.com/spf13/cobra"
)

//go:embed licenses
var licenses embed.FS

func NewCreditsCmd() *cobra.Command {
	creditsCmd := &cobra.Command{
		Use:   "credits [OUTDIR]",
		Args:  cobra.ExactArgs(1),
		Short: "Saves the OSS credits",
		Long:  "Saves licenses, copyright notices and source code, as required by a Go package's dependencies, to a directory.",
		RunE: func(_ *cobra.Command, args []string) error {
			outDir := args[0]
			if outDir == "" {
				return errors.New("please specify the out directory")
			}

			logger := log.GetInstance()

			if err := fs.WalkDir(licenses, "licenses", func(filePath string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}

				if !d.IsDir() {
					contents, err := licenses.ReadFile(filePath)
					if err != nil {
						return fmt.Errorf("failed to read file %s: %w", filePath, err)
					}

					outPath := path.Join(outDir, strings.TrimPrefix(filePath, "licenses/"))

					dir := path.Dir(outPath)

					if _, err := os.Stat(dir); os.IsNotExist(err) {
						if err := os.MkdirAll(dir, 0700); err != nil {
							return fmt.Errorf("failed to create directory %s: %w", dir, err)
						}
					}

					if err := os.WriteFile(outPath, contents, 0644); err != nil {
						return fmt.Errorf("failed to write file %s: %w", outPath, err)
					}
				}

				return nil
			}); err != nil {
				return fmt.Errorf("failed to walk through licenses: %w", err)
			}

			outPath := path.Clean(outDir)
			logger.Info("Wrote credits to " + outPath)

			return nil
		},
	}

	return creditsCmd
}
