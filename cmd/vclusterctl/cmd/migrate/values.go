package migrate

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/flags"
	"github.com/loft-sh/vcluster/pkg/config/legacyconfig"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/yaml"
)

type valuesCmd struct {
	*flags.GlobalFlags
	log      log.Logger
	distro   string
	filePath string
	format   string
}

func migrateValues(globalFlags *flags.GlobalFlags) *cobra.Command {
	c := &valuesCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}

	cobraCmd := &cobra.Command{
		Use:   "values",
		Short: "Migrates current cluster values",
		Long: `
#######################################################
############### vcluster migrate values ###############
#######################################################
Migrates values for a vcluster to the 0.20 format

Examples:
vcluster migrate values --distro k8s -f /my/k8s/values.yaml
vcluster migrate values --distro k3s < /my/k3s/values.yaml
cat /my/k0s/values.yaml | vcluster migrate values --distro k0s
#######################################################
	`,
		RunE: func(_ *cobra.Command, _ []string) error {
			return c.Run()
		}}

	cobraCmd.Flags().StringVarP(&c.filePath, "file", "f", "", "Path to the input file")
	cobraCmd.Flags().StringVar(&c.distro, "distro", "", fmt.Sprintf("Kubernetes distro of the values. Allowed distros: %s", strings.Join([]string{"k8s", "k3s", "k0s", "eks"}, ", ")))
	cobraCmd.Flags().StringVarP(&c.format, "output", "o", "yaml", "Prints the output in the specified format. Allowed values: yaml, json")

	return cobraCmd
}

func (cmd *valuesCmd) Run() error {
	var (
		migratedConfig string
		err            error
	)

	if cmd.distro == "" {
		return fmt.Errorf("no distro given: please set \"--distro\" (IMPORTANT: distro must match the given values)")
	}

	if cmd.filePath != "" {
		file, err := os.Open(cmd.filePath)
		if err != nil {
			return err
		}
		migratedConfig, err = migrate(cmd.distro, file)
		if err != nil {
			return fmt.Errorf("unable to migrate config values: %w", err)
		}
		defer file.Close()
	} else {
		// If no files provided, read from stdin
		migratedConfig, err = migrate(cmd.distro, os.Stdin)
		if err != nil {
			return fmt.Errorf("unable to migrate config values: %w", err)
		}
	}

	var out string
	switch cmd.format {
	case "json":
		j, err := yaml.ToJSON([]byte(migratedConfig))
		if err != nil {
			return err
		}
		out = string(j)
	case "yaml":
		out = migratedConfig
	default:
		fmt.Fprintf(os.Stderr, "unsupported output format: %s. falling back to yaml\n", cmd.format)
		out = migratedConfig
	}

	cmd.log.WriteString(logrus.InfoLevel, out)

	return nil
}

func migrate(distro string, r io.Reader) (string, error) {
	content, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}

	return legacyconfig.MigrateLegacyConfig(distro, string(content))
}
