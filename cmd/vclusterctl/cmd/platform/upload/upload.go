package upload

import (
	"github.com/loft-sh/api/v4/pkg/product"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/spf13/cobra"
)

// NewUploadCmd creates a new cobra command
func NewUploadCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	description := product.ReplaceWithHeader("upload", "")
	c := &cobra.Command{
		Use:   "upload",
		Short: product.Replace("Uploads a file to a vCluster platform resource"),
		Long:  description,
		Args:  cobra.NoArgs,
	}
	c.AddCommand(newOSImageCmd(globalFlags))
	return c
}
