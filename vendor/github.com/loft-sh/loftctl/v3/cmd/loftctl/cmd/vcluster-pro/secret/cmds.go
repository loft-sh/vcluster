package secret

import (
	"strings"

	"github.com/loft-sh/api/v3/pkg/product"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/cmd/get"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/cmd/list"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/cmd/set"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/flags"
	"github.com/loft-sh/loftctl/v3/pkg/defaults"
	"github.com/spf13/cobra"
)

func NewSecretGetCmd(globalFlags *flags.GlobalFlags, defaults *defaults.Defaults) *cobra.Command {
	getCmd := get.NewSecretCmd(globalFlags, defaults)
	getCmd.Use = strings.Replace(getCmd.Use, "secret", "get", 1)

	getCmd.Long = product.ReplaceWithHeader("secret get", `
Returns the key value of a project / shared secret.
	`)

	getCmd.Example = product.Replace(`  vcluster pro secret get test-secret.key
  vcluster pro secret get test-secret.key --project myproject`)

	return getCmd
}

func NewSecretSetCmd(globalFlags *flags.GlobalFlags, defaults *defaults.Defaults) *cobra.Command {
	setCmd := set.NewSecretCmd(globalFlags, defaults)
	setCmd.Use = strings.Replace(setCmd.Use, "secret", "set", 1)

	setCmd.Long = product.ReplaceWithHeader("secret set", `
Sets the key value of a project / shared secret.`)

	setCmd.Example = product.Replace(`  vcluster pro secret set test-secret.key value
  vcluster pro secret set test-secret.key value --project myproject`)

	return setCmd
}

func NewSecretListCmd(globalFlags *flags.GlobalFlags, defaults *defaults.Defaults) *cobra.Command {
	listCmd := list.NewSharedSecretsCmd(globalFlags)
	listCmd.Use = "list"

	listCmd.Long = product.ReplaceWithHeader("secret list", `
List the shared secrets you have access to.`)

	listCmd.Example = product.Replace(`  vcluster pro secret list`)

	return listCmd
}
