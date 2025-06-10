package node

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/spf13/cobra"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubectl/pkg/drain"
)

type DeleteOptions struct {
	*flags.GlobalFlags

	Drain bool

	Log log.Logger
}

func NewDeleteCommand(globalFlags *flags.GlobalFlags) *cobra.Command {
	options := &DeleteOptions{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a node from the vcluster",
		Long: `#######################################################
################### vcluster delete ####################
#######################################################
		`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return options.Run(cmd.Context(), args)
		},
	}

	cmd.Flags().BoolVar(&options.Drain, "drain", true, "Drain the node before deleting it")

	return cmd
}

func (o *DeleteOptions) Run(ctx context.Context, args []string) error {
	// get the node name
	nodeName := args[0]
	kubeClient, err := getClient(o.GlobalFlags)
	if err != nil {
		return fmt.Errorf("failed to get vcluster client: %w", err)
	}

	// get the node
	node, err := kubeClient.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		if kerrors.IsNotFound(err) {
			o.Log.Infof("Node %s not found", nodeName)
			return nil
		}

		return fmt.Errorf("failed to get node: %w", err)
	}

	// evict the pods
	drainHelper := &drain.Helper{
		Ctx:    ctx,
		Client: kubeClient,

		Force:               true,
		IgnoreAllDaemonSets: true,

		Timeout: 30 * time.Minute,

		GracePeriodSeconds: 30,

		Out:    os.Stdout,
		ErrOut: os.Stderr,
	}

	// cordon node first
	o.Log.Infof("Cordoning node %s...", nodeName)
	err = drain.RunCordonOrUncordon(drainHelper, node, true)
	if err != nil {
		return fmt.Errorf("failed to cordon node: %w", err)
	}

	// drain node
	if o.Drain {
		o.Log.Infof("Draining node %s...", nodeName)
		err = drain.RunNodeDrain(drainHelper, nodeName)
		if err != nil {
			return fmt.Errorf("failed to drain node: %w", err)
		}
	}

	// delete node
	o.Log.Infof("Deleting node %s...", nodeName)
	err = kubeClient.CoreV1().Nodes().Delete(ctx, nodeName, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete node: %w", err)
	}

	o.Log.Infof("Successfully deleted node %s", nodeName)
	return nil
}
