package secret

import (
	"github.com/loft-sh/api/v3/pkg/product"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/flags"
	"github.com/loft-sh/loftctl/v3/pkg/defaults"
	"github.com/spf13/cobra"
)

func NewRootCmd(globalFlags *flags.GlobalFlags, defaults *defaults.Defaults) *cobra.Command {
	short := "Management operations on secret resources"

	secretCmd := &cobra.Command{
		Use:   "secret",
		Short: short,
		Long:  product.ReplaceWithHeader("secret", short),

		Aliases: []string{"secrets"},

		Example: product.Replace(`  # Get the value for a given key of a secret
  vcluster pro secret get test-secret.key

  # List all secrets
  vcluster pro secret list

  # Set the value for a given key of a secret
  vcluster pro secret set test-secret.key value
`),
		Args: cobra.NoArgs,
	}
	secretCmd.AddCommand(NewSecretListCmd(globalFlags, defaults))
	secretCmd.AddCommand(NewSecretGetCmd(globalFlags, defaults))
	secretCmd.AddCommand(NewSecretSetCmd(globalFlags, defaults))

	return secretCmd
}
