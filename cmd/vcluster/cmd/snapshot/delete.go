package snapshot

import (
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
)

func NewDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "delete vCluster snapshot",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			options := &Options{}
			envOptions, err := parseOptionsFromEnv()
			if err != nil {
				klog.Warningf("Error parsing environment variables: %v", err)
			} else {
				options.Snapshot = *envOptions
			}
			return options.Delete(cmd.Context())
		},
	}

	return cmd
}
