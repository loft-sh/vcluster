package standalone

import (
	"fmt"
	"os"
	"os/signal"
	"strings"

	"github.com/loft-sh/vcluster/pkg/cli/standalone"
	"github.com/loft-sh/vcluster/pkg/upgrade"
	"github.com/spf13/cobra"
)

func NewInstallCommand() *cobra.Command {
	options := &standalone.InstallOptions{}
	var extraEnv []string

	installCmd := &cobra.Command{
		Use:   "install",
		Short: "Install vCluster Standalone Node",
		Long: `#####################################################
############ vcluster standalone install ############
#####################################################
Install vCluster Standalone Node

Example:
# Install vCluster Standalone control-plane node
$ vcluster standalone install --version v0.33.0

# Install vCluster Standalone using already downloaded binaries
$ ls /path/to/downloaded/binaries/
vcluster  vcluster-cli
$ vcluster standalone install --binary /path/to/downloaded/binaries/

# Install vCluster Standalone using a custom download URL. The same structure as the GitHub release assets is expected.
$ vcluster standalone install --download-url https://github.com/loft-sh/vcluster/releases/latest/download/

# Install vCluster Standalone control-plane node and join it to a cluster.
# Command obtain from the "vcluster token create --control-plane" output, executed against the cluster.
$ vcluster standalone install --join "https:/host:port/node/join?token=xxxxxxx.yyyyyyyyyyyyyyy"

#####################################################
		`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			env, err := parseExtraEnv(extraEnv)
			if err != nil {
				return err
			}
			options.Env = env

			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, os.Kill)
			defer stop()

			return standalone.Install(ctx, options)
		},
	}

	installCmd.Flags().StringVar(&options.Version, "version", upgrade.GetVersion(), "Specific vCluster version to install")
	installCmd.Flags().BoolVar(&options.SkipDownload, "skip-download", false, "Do not download the Kubernetes bundle")
	installCmd.Flags().StringVar(&options.Name, "name", "standalone", "Name of the vCluster instance")
	installCmd.Flags().BoolVar(&options.SkipWait, "skip-wait", false, "Exit without waiting for vCluster to be ready")
	installCmd.Flags().StringArrayVar(&extraEnv, "extra-env", []string{}, "Additional environment variables for vCluster")
	installCmd.Flags().StringVar(&options.Config, "config", "", "Path to the vcluster.yaml configuration file")
	installCmd.Flags().StringVar(&options.Binary, "binary", "", "Path to the vcluster and vcluster-cli binaries")
	installCmd.Flags().StringVar(&options.JoinURL, "join", "", "join URL")
	installCmd.Flags().BoolVar(&options.InsecureSkipVerify, "insecure", true, "If TLS verify should be skipped for all initiated TLS connections")
	installCmd.Flags().StringVar(&options.KubernetesBundle, "kubernetes-bundle", "", "The Kubernetes bundle to use for installing vCluster")
	installCmd.Flags().BoolVar(&options.Fips, "fips", false, "Enable FIPS mode")
	installCmd.Flags().StringVar(&options.DownloadURL, "download-url", "", "todo: what download url?")

	return installCmd
}

func parseExtraEnv(extraEnv []string) (map[string]string, error) {
	env := make(map[string]string)
	for _, keyAndValue := range extraEnv {
		parts := strings.Split(keyAndValue, "=")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid extra-env format: %s", keyAndValue)
		}

		if parts[0] == "" {
			return nil, fmt.Errorf("env key must not be empty: %s", keyAndValue)
		}

		env[parts[0]] = parts[1]
	}

	return env, nil
}
