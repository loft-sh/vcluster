package space

import (
	"strings"

	"github.com/loft-sh/api/v3/pkg/product"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/cmd/create"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/cmd/delete"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/cmd/list"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/cmd/use"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/flags"
	"github.com/loft-sh/loftctl/v3/pkg/defaults"
	"github.com/spf13/cobra"
)

func NewSpaceListCmd(globalFlags *flags.GlobalFlags, defaults *defaults.Defaults) *cobra.Command {
	listCmd := list.NewSpacesCmd(globalFlags)
	listCmd.Use = "list"

	listCmd.Long = product.ReplaceWithHeader("space list", `
List the vCluster.Pro spaces you have access to.`)

	listCmd.Example = "  vcluster pro space list"

	return listCmd
}

func NewSpaceCreateCmd(globalFlags *flags.GlobalFlags, defaults *defaults.Defaults) *cobra.Command {
	createCmd := create.NewSpaceCmd(globalFlags, defaults)
	createCmd.Use = strings.Replace(createCmd.Use, "space", "create", 1)

	createCmd.Long = product.ReplaceWithHeader("space create", `
Creates a new space for the given project, if
it does not yet exist.`)

	createCmd.Example = product.Replace(`  vcluster pro space create myspace
  vcluster pro space create myspace --project myproject
  vcluster pro space create myspace --project myproject --team myteam`)

	return createCmd
}

func NewSpaceDeleteCmd(globalFlags *flags.GlobalFlags, defaults *defaults.Defaults) *cobra.Command {
	deleteCmd := delete.NewSpaceCmd(globalFlags, defaults)
	deleteCmd.Use = strings.Replace(deleteCmd.Use, "space", "delete", 1)

	deleteCmd.Long = product.ReplaceWithHeader("space delete", `
Deletes a space from a cluster.`)

	deleteCmd.Example = product.Replace(`  vcluster pro space delete myspace
  vcluster pro space delete myspace --project myproject`)

	return deleteCmd
}

func NewSpaceUseCmd(globalFlags *flags.GlobalFlags, defaults *defaults.Defaults) *cobra.Command {
	useCmd := use.NewSpaceCmd(globalFlags, defaults)
	useCmd.Use = strings.Replace(useCmd.Use, "space", "use", 1)
	useCmd.Aliases = []string{"connect"}

	useCmd.Long = product.ReplaceWithHeader("space use", `
Creates a new kube context for the given space.`)

	useCmd.Example = product.Replace(`  vcluster pro space use
  vcluster pro space use myspace
  vcluster pro space use myspace --project myproject`)

	return useCmd
}
