package pro

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/google/go-github/v53/github"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/flags"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
)

func NewProCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	proCmd := &cobra.Command{
		Use:   "pro",
		Short: "vcluster.pro subcommands",
		Long: `
#######################################################
#################### vcluster get #####################
#######################################################
		`,
		Args:              cobra.NoArgs,
		PersistentPreRunE: preRun,
	}

	proCmd.AddCommand(NewStartCmd(globalFlags))
	proCmd.AddCommand(NewStopCmd(globalFlags))
	proCmd.AddCommand(NewLoginCmd(globalFlags))
	proCmd.AddCommand(NewCreateCmd(globalFlags))
	proCmd.AddCommand(NewImportCmd(globalFlags))
	proCmd.AddCommand(NewDeleteCmd(globalFlags))
	proCmd.AddCommand(NewListCmd(globalFlags))

	return proCmd
}

func preRun(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	proConfig, err := pro.GetConfig()
	if err != nil {
		return fmt.Errorf("failed to get pro config: %v", err)
	}

	if time.Since(proConfig.LatestVersionCheckedAt).Hours() > 24 || os.Getenv("PRO_FORCE_UPDATE") == "true" {
		client := github.NewClient(nil)

		release, _, err := client.Repositories.GetLatestRelease(ctx, "loft-sh", "loft")
		if err != nil {
			return fmt.Errorf("failed to get latest release: %w", err)
		}

		tagName := release.GetTagName()

		if tagName != "" {
			filePath, err := pro.LoftBinaryFilePath(tagName)
			if err != nil {
				return fmt.Errorf("failed to get loft binary file path: %v", err)
			}

			_, err = os.Stat(filePath)
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("failed to stat loft binary: %w", err)
			}

			asset, found := lo.Find(release.Assets, func(asset *github.ReleaseAsset) bool {
				return fmt.Sprintf("loft-%s-%s", runtime.GOOS, runtime.GOARCH) == *asset.Name
			})

			if !found {
				return fmt.Errorf("failed to find loft binary for tag %s", tagName)
			}

			// download binary
			err = pro.DownloadLoftBinary(ctx, filePath, asset.GetBrowserDownloadURL())
			if err != nil {
				return fmt.Errorf("failed to download loft binary: %w", err)
			}
		}

		proConfig.LatestVersionCheckedAt = time.Now()
		err = pro.WriteConfig(proConfig)
		if err != nil {
			return fmt.Errorf("failed to write pro config: %w", err)
		}
	}

	return nil
}
