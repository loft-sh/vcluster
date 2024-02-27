package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"

	proclient "github.com/loft-sh/loftctl/v3/pkg/client"
	loftctlUtil "github.com/loft-sh/loftctl/v3/pkg/util"
	"github.com/loft-sh/loftctl/v3/pkg/vcluster"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/find"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/flags"
	"github.com/loft-sh/vcluster/pkg/lifecycle"
	"github.com/loft-sh/vcluster/pkg/procli"
)

// ResumeCmd holds the cmd flags
type ResumeCmd struct {
	*flags.GlobalFlags
	Log        log.Logger
	kubeClient *kubernetes.Clientset
	Project    string
}

// NewResumeCmd creates a new command
func NewResumeCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &ResumeCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}

	cobraCmd := &cobra.Command{
		Use:     "resume" + loftctlUtil.VClusterNameOnlyUseLine,
		Aliases: []string{"wakeup"},
		Short:   "Resumes a virtual cluster",
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
		Args:              loftctlUtil.VClusterNameOnlyValidator,
		ValidArgsFunction: newValidVClusterNameFunc(globalFlags),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context(), args)
		},
	}

	cobraCmd.Flags().StringVar(&cmd.Project, "project", "", "[PRO] The pro project the vcluster is in")
	return cobraCmd
}

// Run executes the functionality
func (cmd *ResumeCmd) Run(ctx context.Context, args []string) error {
	// get pro client
	proClient, err := procli.CreateProClient()
	if err != nil {
		cmd.Log.Debugf("Error creating pro client: %v", err)
	}

	// find vcluster
	vClusterName := args[0]
	vCluster, proVCluster, err := find.GetVCluster(ctx, proClient, cmd.Context, vClusterName, cmd.Namespace, cmd.Project, cmd.Log)
	if err != nil {
		return err
	} else if proVCluster != nil {
		return cmd.resumeProVCluster(ctx, proClient, proVCluster)
	}

	err = cmd.prepare(vCluster)
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

func (cmd *ResumeCmd) resumeProVCluster(ctx context.Context, proClient proclient.Client, vCluster *procli.VirtualClusterInstanceProject) error {
	managementClient, err := proClient.Management()
	if err != nil {
		return err
	}

	cmd.Log.Infof("Waking up virtual cluster %s in project %s", vCluster.VirtualCluster.Name, vCluster.Project.Name)

	_, err = vcluster.WaitForVirtualClusterInstance(ctx, managementClient, vCluster.VirtualCluster.Namespace, vCluster.VirtualCluster.Name, true, cmd.Log)
	if err != nil {
		return err
	}

	cmd.Log.Donef("Successfully woke up vcluster %s", vCluster.VirtualCluster.Name)
	return nil
}

func (cmd *ResumeCmd) prepare(vCluster *find.VCluster) error {
	// load the rest config
	kubeConfig, err := vCluster.ClientFactory.ClientConfig()
	if err != nil {
		return fmt.Errorf("there is an error loading your current kube config (%w), please make sure you have access to a kubernetes cluster and the command `kubectl get namespaces` is working", err)
	}

	kubeClient, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return err
	}

	cmd.Namespace = vCluster.Namespace
	cmd.kubeClient = kubeClient
	return nil
}
