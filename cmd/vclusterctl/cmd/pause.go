package cmd

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"

	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/find"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/flags"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/log"
	"github.com/loft-sh/vcluster/pkg/lifecycle"
)

// PauseCmd holds the cmd flags
type PauseCmd struct {
	*flags.GlobalFlags
	Log log.Logger

	kubeClient *kubernetes.Clientset
}

// NewPauseCmd creates a new command
func NewPauseCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &PauseCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}

	cobraCmd := &cobra.Command{
		Use:   "pause [flags] vcluster_name",
		Short: "Pauses a virtual cluster",
		Long: `
#######################################################
################### vcluster pause ####################
#######################################################
Pause will stop a virtual cluster and free all its used
computing resources.

Pause will scale down the virtual cluster and delete
all workloads created through the virtual cluster. Upon resume,
all workloads will be recreated. Other resources such 
as persistent volume claims, services etc. will not be affected.

Example:
vcluster pause test --namespace test
#######################################################
	`,
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: newValidVClusterNameFunc(globalFlags),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(args)
		},
	}
	return cobraCmd
}

// Run executes the functionality
func (cmd *PauseCmd) Run(args []string) error {
	err := cmd.prepare(args[0])
	if err != nil {
		return err
	}

	err = lifecycle.PauseVCluster(cmd.kubeClient, args[0], cmd.Namespace, cmd.Log)
	if err != nil {
		return err
	}

	err = lifecycle.DeleteVClusterWorkloads(cmd.kubeClient, "vcluster.loft.sh/managed-by="+args[0], cmd.Namespace, cmd.Log)
	if err != nil {
		return errors.Wrap(err, "delete vcluster workloads")
	}

	err = lifecycle.DeleteMultiNamespaceVclusterWorkloads(context.TODO(), cmd.kubeClient, args[0], cmd.Namespace, cmd.Log)
	if err != nil {
		return errors.Wrap(err, "delete vcluster multinamespace workloads")
	}

	cmd.Log.Donef("Successfully paused vcluster %s/%s", cmd.Namespace, args[0])
	return nil
}

func (cmd *PauseCmd) prepare(vClusterName string) error {
	vCluster, err := find.GetVCluster(cmd.Context, vClusterName, cmd.Namespace)
	if err != nil {
		return err
	}

	// load the rest config
	kubeConfig, err := vCluster.ClientFactory.ClientConfig()
	if err != nil {
		return fmt.Errorf("there is an error loading your current kube config (%v), please make sure you have access to a kubernetes cluster and the command `kubectl get namespaces` is working", err)
	}

	kubeClient, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return err
	}

	currentContext, currentRawConfig, err := find.CurrentContext()
	if err != nil {
		return err
	}

	vClusterName, vClusterNamespace, vClusterContext := find.VClusterFromContext(currentContext)
	if vClusterName == vCluster.Name && vClusterNamespace == vCluster.Namespace && vClusterContext == vCluster.Context {
		err = switchContext(currentRawConfig, vCluster.Context)
		if err != nil {
			return err
		}
	}

	cmd.Namespace = vCluster.Namespace
	cmd.kubeClient = kubeClient
	return nil
}
