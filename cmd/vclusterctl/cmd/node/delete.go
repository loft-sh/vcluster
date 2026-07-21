package node

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
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
		DeleteEmptyDirData:  true,

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
		errc := make(chan error)

		o.Log.Infof("Draining node %s...", nodeName)

		// Poll for the node status, so we don't keep trying if it has been deleted or shutdown.
		go o.pollNodeStatus(ctx, kubeClient, nodeName, errc)
		go func() {
			errc <- drain.RunNodeDrain(drainHelper, nodeName)
		}()

		if err = <-errc; err != nil {
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

func (o *DeleteOptions) pollNodeStatus(ctx context.Context, kubeClient *kubernetes.Clientset, nodeName string, errc chan error) {
	for range time.Tick(time.Second * 5) {
		node, err := kubeClient.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
		// We don't care about an error here, as RunDrainNode should deal with errors in getting the Node resource.
		// We're looking for the case where the Node is present and has a status indicating it that it can't be
		// reached rather than the manifest/resource isn't there.
		if err != nil {
			continue
		}

		for _, s := range node.Status.Conditions {
			if s.Status == corev1.ConditionUnknown && s.Message == "Kubelet stopped posting node status." {
				o.Log.Warnf("The status of node %q is unknown. This may indicate the node was shutdown or lost connectivity.  If so, try rerunning with --drain=false", nodeName)
				errc <- fmt.Errorf("node %q status unknown", nodeName)
				return
			}
		}
	}
}
