package convert

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/config/legacyconfig"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/yaml"
)

type configCmd struct {
	*flags.GlobalFlags
	log      log.Logger
	distro   string
	filePath string
	format   string
}

func convertValues(globalFlags *flags.GlobalFlags) *cobra.Command {
	c := &configCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}

	cobraCmd := &cobra.Command{
		Use:   "config",
		Short: "Converts virtual cluster config values to the v0.20 format",
		Long: `##############################################################
################## vcluster convert config ###################
##############################################################
Converts the given virtual cluster config to the v0.20 format.
Reads from stdin if no file is given via "-f".

Examples:
vcluster convert config --distro k8s -f /my/k8s/values.yaml
vcluster convert config --distro k3s < /my/k3s/values.yaml
cat /my/k0s/values.yaml | vcluster convert config --distro k0s
##############################################################
	`,
		RunE: func(_ *cobra.Command, _ []string) error {
			return c.Run()
		}}

	cobraCmd.Flags().StringVarP(&c.filePath, "file", "f", "", "Path to the input file")
	cobraCmd.Flags().StringVar(&c.distro, "distro", "", fmt.Sprintf("Kubernetes distro of the config. Allowed distros: %s", strings.Join([]string{"k8s", "k3s", "k0s"}, ", ")))
	cobraCmd.Flags().StringVarP(&c.format, "output", "o", "yaml", "Prints the output in the specified format. Allowed values: yaml, json")

	return cobraCmd
}

func (cmd *configCmd) Run() error {
	var (
		convertedConfig string
		err             error
	)

	if cmd.distro == "" {
		return fmt.Errorf("no distro given: please set \"--distro\" (IMPORTANT: distro must match the given config values, or be \"k8s\" if you are migrating from eks distro)")
	}

	if cmd.filePath != "" {
		file, err := os.Open(cmd.filePath)
		if err != nil {
			return err
		}
		convertedConfig, err = convert(cmd.distro, file)
		if err != nil {
			return fmt.Errorf("unable to convert config values: %w", err)
		}
		defer file.Close()
	} else {
		// If no files provided, read from stdin
		convertedConfig, err = convert(cmd.distro, os.Stdin)
		if err != nil {
			return fmt.Errorf("unable to convert config values: %w", err)
		}
	}

	var out string
	switch cmd.format {
	case "json":
		j, err := yaml.ToJSON([]byte(convertedConfig))
		if err != nil {
			return err
		}
		out = string(j)
	case "yaml":
		out = convertedConfig
	default:
		fmt.Fprintf(os.Stderr, "unsupported output format: %s. falling back to yaml\n", cmd.format)
		out = convertedConfig
	}

	cmd.log.WriteString(logrus.InfoLevel, out)

	return nil
}

func convert(distro string, r io.Reader) (string, error) {
	content, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}

	return legacyconfig.MigrateLegacyConfig(distro, string(content))
}
