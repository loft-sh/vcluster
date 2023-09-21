package vclusterpro

import (
	"strings"

	"github.com/loft-sh/api/v3/pkg/product"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/cmd/create"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/cmd/delete"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/cmd/importcmd"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/cmd/list"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/cmd/use"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/flags"
	"github.com/loft-sh/loftctl/v3/pkg/defaults"
	"github.com/spf13/cobra"
)

func NewConnectCmd(globalFlags *flags.GlobalFlags, defaults *defaults.Defaults) *cobra.Command {
	connectCmd := use.NewVirtualClusterCmd(globalFlags, defaults)
	connectCmd.Use = strings.Replace(connectCmd.Use, "vcluster", "connect", 1)
	connectCmd.Aliases = []string{"use"}

	connectCmd.Long = product.ReplaceWithHeader("connect", `
Creates a new kube context for the given virtual cluster.`)

	connectCmd.Example = product.Replace(`  vcluster pro connect
  vcluster pro connect myvcluster
  vcluster pro connect myvcluster --cluster mycluster
  vcluster pro connect myvcluster --cluster mycluster --space myspace`)

	return connectCmd
}

func NewCreateCmd(globalFlags *flags.GlobalFlags, defaults *defaults.Defaults) *cobra.Command {
	createCmd := create.NewVirtualClusterCmd(globalFlags, defaults)
	createCmd.Use = strings.Replace(createCmd.Use, "vcluster", "create", 1)

	createCmd.Long = product.ReplaceWithHeader("create", `
Creates a new virtual cluster in a given space and
cluster. If no space or cluster is specified the user
will be asked.`)

	createCmd.Example = product.Replace(`  vcluster pro create test
  vcluster pro create test --project myproject`)

	return createCmd
}

func NewDeleteCmd(globalFlags *flags.GlobalFlags, defaults *defaults.Defaults) *cobra.Command {
	deleteCmd := delete.NewVirtualClusterCmd(globalFlags, defaults)
	deleteCmd.Use = strings.Replace(deleteCmd.Use, "vcluster", "delete", 1)

	deleteCmd.Long = product.ReplaceWithHeader("delete", `
Deletes a virtual cluster from a cluster.`)

	deleteCmd.Example = product.Replace(`  vcluster pro delete myvirtualcluster
  vcluster pro delete myvirtualcluster --project myproject`)

	return deleteCmd
}

func NewImportCmd(globalFlags *flags.GlobalFlags, defaults *defaults.Defaults) *cobra.Command {
	importCmd := importcmd.NewVClusterCmd(globalFlags)
	importCmd.Use = strings.Replace(importCmd.Use, "vcluster", "import", 1)

	importCmd.Long = product.ReplaceWithHeader("import", `
Imports a vcluster into a vCluster.Pro project.`)

	importCmd.Example = product.Replace(`  vcluster pro import my-vcluster --cluster connected-cluster my-vcluster \
    --namespace vcluster-my-vcluster --project my-project --importname my-vcluster`)

	return importCmd
}

func NewListCmd(globalFlags *flags.GlobalFlags, defaults *defaults.Defaults) *cobra.Command {
	listCmd := list.NewVirtualClustersCmd(globalFlags)
	listCmd.Use = "list"
	listCmd.Aliases = []string{"ls"}

	listCmd.Long = product.ReplaceWithHeader("list", `
List the virtual clusters you have access to.`)

	listCmd.Example = "  vcluster pro list"

	return listCmd
}
