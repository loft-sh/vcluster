package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"

	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/find"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/flags"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/log"
	"github.com/loft-sh/vcluster/pkg/lifecycle"
)

// ResumeCmd holds the cmd flags
type ResumeCmd struct {
	*flags.GlobalFlags
	Log log.Logger

	kubeClient *kubernetes.Clientset
}

// NewResumeCmd creates a new command
func NewResumeCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &ResumeCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}

	cobraCmd := &cobra.Command{
		Use:   "resume [flags] vcluster_name",
		Short: "Resumes a virtual cluster",
		Long: `
#######################################################
################### vcluster resume ###################
#######################################################
Resume will start a vcluster after it was paused. 
vcluster will recreate all the workloads after it has 
started automatically.

Example:
vcluster resume test --namespace test
#######################################################
	`,
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: newValidVClusterNameFunc(globalFlags),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context(), args)
		},
	}
	return cobraCmd
}

// Run executes the functionality
func (cmd *ResumeCmd) Run(ctx context.Context, args []string) error {
	err := cmd.prepare(ctx, args[0])
	if err != nil {
		return err
	}

	err = lifecycle.ResumeVCluster(ctx, cmd.kubeClient, args[0], cmd.Namespace, cmd.Log)
	if err != nil {
		return err
	}

	cmd.Log.Donef("Successfully resumed vcluster %s in namespace %s", args[0], cmd.Namespace)
	return nil
}

func (cmd *ResumeCmd) prepare(ctx context.Context, vClusterName string) error {
	vCluster, err := find.GetVCluster(ctx, cmd.Context, vClusterName, cmd.Namespace)
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

	cmd.Namespace = vCluster.Namespace
	cmd.kubeClient = kubeClient
	return nil
}
