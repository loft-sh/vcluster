package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/completion"
	"github.com/loft-sh/vcluster/pkg/cli/find"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/cli/util"
	"github.com/loft-sh/vcluster/pkg/lifecycle"
	"github.com/loft-sh/vcluster/pkg/snapshot"
	"github.com/loft-sh/vcluster/pkg/snapshot/pod"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type RestoreCmd struct {
	*flags.GlobalFlags

	RestartWorkloads bool

	Snapshot snapshot.Options
	Pod      pod.Options

	Log log.Logger
}

// NewRestore creates a new command
func NewRestore(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &RestoreCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}

	useLine, nameValidator := util.NamedPositionalArgsValidator(true, false, "VCLUSTER_NAME")
	cobraCmd := &cobra.Command{
		Use:   "restore" + useLine,
		Short: "Restores a virtual cluster from snapshot",
		Long: `#######################################################
################# vcluster restore ####################
#######################################################
Restore a virtual cluster.

Example:
vcluster restore test --namespace test
#######################################################
	`,
		Args:              nameValidator,
		ValidArgsFunction: completion.NewValidVClusterNameFunc(globalFlags),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context(), args)
		},
	}

	cobraCmd.Flags().BoolVar(&cmd.RestartWorkloads, "restart-workloads", true, "If true restarts the vCluster workloads")

	// add storage flags
	snapshot.AddFlags(cobraCmd.Flags(), &cmd.Snapshot)
	pod.AddFlags(cobraCmd.Flags(), &cmd.Pod)
	return cobraCmd
}

func (cmd *RestoreCmd) Run(ctx context.Context, args []string) error {
	// init kube client and vCluster
	vCluster, kubeClient, err := initSnapshotCommand(ctx, args, &cmd.Snapshot, &cmd.Pod, cmd.Log)
	if err != nil {
		return err
	}

	// pause vCluster
	cmd.Log.Infof("Pausing vCluster %s", vCluster.Name)
	err = cmd.pauseVCluster(ctx, kubeClient, vCluster)
	if err != nil {
		return fmt.Errorf("pause vCluster %s: %w", vCluster.Name, err)
	}

	// try to scale up the vCluster again
	defer func() {
		cmd.Log.Infof("Resuming vCluster %s after it was paused", vCluster.Name)
		err = lifecycle.ResumeVCluster(ctx, kubeClient, vCluster.Name, vCluster.Namespace, cmd.Log)
		if err != nil {
			cmd.Log.Warnf("Error resuming vCluster %s: %v", vCluster.Name, err)
		}
	}()

	// set missing pod options and run snapshot restore pod
	cmd.Pod.Command = []string{"/vcluster", "restore"}
	return pod.RunSnapshotPod(ctx, kubeClient, &cmd.Pod, &cmd.Snapshot, cmd.Log)
}

func (cmd *RestoreCmd) pauseVCluster(ctx context.Context, kubeClient *kubernetes.Clientset, vCluster *find.VCluster) error {
	err := lifecycle.PauseVCluster(ctx, kubeClient, vCluster.Name, vCluster.Namespace, false, cmd.Log)
	if err != nil {
		return err
	}

	if cmd.RestartWorkloads {
		err = lifecycle.DeletePods(ctx, kubeClient, "vcluster.loft.sh/managed-by="+vCluster.Name, vCluster.Namespace)
		if err != nil {
			return fmt.Errorf("delete vcluster workloads: %w", err)
		}

		err = lifecycle.DeleteMultiNamespaceVClusterWorkloads(ctx, kubeClient, vCluster.Name, vCluster.Namespace, cmd.Log)
		if err != nil {
			return fmt.Errorf("delete vcluster multinamespace workloads: %w", err)
		}
	}

	// ensure there is only one pvc
	err = cmd.ensurePVCs(ctx, kubeClient, vCluster)
	if err != nil {
		return err
	}

	return nil
}

func (cmd *RestoreCmd) ensurePVCs(ctx context.Context, kubeClient *kubernetes.Clientset, vCluster *find.VCluster) error {
	// if not a statefulset we don't care
	if vCluster.StatefulSet == nil || len(vCluster.StatefulSet.Spec.VolumeClaimTemplates) == 0 {
		return nil
	}

	// two things we need to check for:
	// 1. if there is more than 1 pvc (this is the case for embedded etcd) then we delete all pvc's except the first one
	// 2. if there is no pvc and the statefulset has a persistent volume claim template we create the pvc
	pvcList, err := kubeClient.CoreV1().PersistentVolumeClaims(vCluster.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app=vcluster,release=%s", vCluster.Name),
	})
	if err != nil {
		return fmt.Errorf("list vcluster pvcs: %w", err)
	}

	// handle the two cases now
	if len(pvcList.Items) == 0 {
		// create the pvc
		cmd.Log.Infof("No vCluster pvcs found in namespace %s, creating a new one...", vCluster.Namespace)
		dataVolume := vCluster.StatefulSet.Spec.VolumeClaimTemplates[0]
		_, err := kubeClient.CoreV1().PersistentVolumeClaims(vCluster.Namespace).Create(ctx, &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name: dataVolume.Name + "-" + vCluster.StatefulSet.Name + "-0",
			},
			Spec: dataVolume.Spec,
		}, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("create vcluster pvc: %w", err)
		}
	} else if len(pvcList.Items) > 1 {
		// delete the non -0 ones
		for _, pvc := range pvcList.Items {
			if strings.HasSuffix(pvc.Name, "-0") {
				continue
			}

			cmd.Log.Infof("Deleting vCluster pvc %s/%s", pvc.Namespace, pvc.Name)
			err = kubeClient.CoreV1().PersistentVolumeClaims(vCluster.Namespace).Delete(ctx, pvc.Name, metav1.DeleteOptions{})
			if err != nil {
				return fmt.Errorf("delete vcluster pvc: %w", err)
			}
		}
	}

	return nil
}
