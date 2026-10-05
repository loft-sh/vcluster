package token

import (
	"context"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	bootstraputil "k8s.io/cluster-bootstrap/token/util"
)

type DeleteCmd struct {
	*flags.GlobalFlags

	Log log.Logger
}

func NewDeleteCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &DeleteCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}

	description := `########################################################
################# vcluster token delete #################
########################################################
Delete a node bootstrap token for a vCluster with private nodes enabled.
#######################################################
	`

	deleteCmd := &cobra.Command{
		Use:   "delete <token-id>",
		Short: "Delete a node bootstrap token for a vCluster with private nodes enabled.",
		Long:  description,
		Args:  cobra.ExactArgs(1),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context(), args)
		},
	}

	return deleteCmd
}

func (cmd *DeleteCmd) Run(ctx context.Context, args []string) error {
	// get the client
	vClient, err := getClient(cmd.GlobalFlags)
	if err != nil {
		return err
	}

	// delete the token
	secretName := bootstraputil.BootstrapTokenSecretName(args[0])
	err = vClient.CoreV1().Secrets(metav1.NamespaceSystem).Delete(ctx, secretName, metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	cmd.Log.WriteString(logrus.InfoLevel, "Token deleted successfully")
	return nil
}
