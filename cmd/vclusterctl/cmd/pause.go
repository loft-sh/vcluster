package cmd

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"

	clusterv1 "github.com/loft-sh/agentapi/v3/pkg/apis/loft/cluster/v1"
	storagev1 "github.com/loft-sh/api/v3/pkg/apis/storage/v1"
	"github.com/loft-sh/api/v3/pkg/product"
	proclient "github.com/loft-sh/loftctl/v3/pkg/client"
	"github.com/loft-sh/loftctl/v3/pkg/config"
	loftctlUtil "github.com/loft-sh/loftctl/v3/pkg/util"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/find"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/flags"
	"github.com/loft-sh/vcluster/pkg/lifecycle"
	"github.com/loft-sh/vcluster/pkg/procli"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PauseCmd holds the cmd flags
type PauseCmd struct {
	*flags.GlobalFlags
	Log           log.Logger
	kubeClient    *kubernetes.Clientset
	Project       string
	ForceDuration int64
}

// NewPauseCmd creates a new command
func NewPauseCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &PauseCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}

	cobraCmd := &cobra.Command{
		Use:     "pause" + loftctlUtil.VClusterNameOnlyUseLine,
		Aliases: []string{"sleep"},
		Short:   "Pauses a virtual cluster",
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
		Args:              loftctlUtil.VClusterNameOnlyValidator,
		ValidArgsFunction: newValidVClusterNameFunc(globalFlags),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context(), args)
		},
	}

	cobraCmd.Flags().StringVar(&cmd.Project, "project", "", "[PRO] The pro project the vcluster is in")
	cobraCmd.Flags().Int64Var(&cmd.ForceDuration, "prevent-wakeup", -1, product.Replace("[PRO] The amount of seconds this vcluster should sleep until it can be woken up again (use 0 for infinite sleeping). During this time the space can only be woken up by `vcluster resume vcluster`, manually deleting the annotation on the namespace or through the loft UI"))
	return cobraCmd
}

// Run executes the functionality
func (cmd *PauseCmd) Run(ctx context.Context, args []string) error {
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
		return cmd.pauseProVCluster(ctx, proClient, proVCluster)
	}

	err = cmd.prepare(vCluster)
	if err != nil {
		return err
	}

	err = lifecycle.PauseVCluster(ctx, cmd.kubeClient, args[0], cmd.Namespace, cmd.Log)
	if err != nil {
		return err
	}

	err = lifecycle.DeleteVClusterWorkloads(ctx, cmd.kubeClient, "vcluster.loft.sh/managed-by="+args[0], cmd.Namespace, cmd.Log)
	if err != nil {
		return errors.Wrap(err, "delete vcluster workloads")
	}

	err = lifecycle.DeleteMultiNamespaceVclusterWorkloads(ctx, cmd.kubeClient, args[0], cmd.Namespace, cmd.Log)
	if err != nil {
		return errors.Wrap(err, "delete vcluster multinamespace workloads")
	}

	cmd.Log.Donef("Successfully paused vcluster %s/%s", cmd.Namespace, args[0])
	return nil
}

func (cmd *PauseCmd) pauseProVCluster(ctx context.Context, proClient proclient.Client, vCluster *procli.VirtualClusterInstanceProject) error {
	managementClient, err := proClient.Management()
	if err != nil {
		return err
	}

	cmd.Log.Infof("Putting virtual cluster %s in project %s to sleep", vCluster.VirtualCluster.Name, vCluster.Project.Name)

	virtualClusterInstance, err := managementClient.Loft().ManagementV1().VirtualClusterInstances(vCluster.VirtualCluster.Namespace).Get(ctx, vCluster.VirtualCluster.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if virtualClusterInstance.Annotations == nil {
		virtualClusterInstance.Annotations = map[string]string{}
	}
	virtualClusterInstance.Annotations[clusterv1.SleepModeForceAnnotation] = "true"
	if cmd.ForceDuration >= 0 {
		virtualClusterInstance.Annotations[clusterv1.SleepModeForceDurationAnnotation] = strconv.FormatInt(cmd.ForceDuration, 10)
	}

	_, err = managementClient.Loft().ManagementV1().VirtualClusterInstances(vCluster.VirtualCluster.Namespace).Update(ctx, virtualClusterInstance, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	// wait for sleeping
	cmd.Log.Info("Wait until virtual cluster is sleeping...")
	err = wait.PollUntilContextTimeout(ctx, time.Second, config.Timeout(), false, func(ctx context.Context) (done bool, err error) {
		virtualClusterInstance, err := managementClient.Loft().ManagementV1().VirtualClusterInstances(vCluster.VirtualCluster.Namespace).Get(ctx, vCluster.VirtualCluster.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		return virtualClusterInstance.Status.Phase == storagev1.InstanceSleeping, nil
	})
	if err != nil {
		return fmt.Errorf("error waiting for vcluster to start sleeping: %w", err)
	}

	cmd.Log.Donef("Successfully put vcluster %s to sleep", vCluster.VirtualCluster.Name)
	return nil
}

func (cmd *PauseCmd) prepare(vCluster *find.VCluster) error {
	// load the rest config
	kubeConfig, err := vCluster.ClientFactory.ClientConfig()
	if err != nil {
		return fmt.Errorf("there is an error loading your current kube config (%w), please make sure you have access to a kubernetes cluster and the command `kubectl get namespaces` is working", err)
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
		err = find.SwitchContext(currentRawConfig, vCluster.Context)
		if err != nil {
			return err
		}
	}

	cmd.Namespace = vCluster.Namespace
	cmd.kubeClient = kubeClient
	return nil
}
