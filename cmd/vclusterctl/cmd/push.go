package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/loft-sh/vcluster/pkg/oci"
	"github.com/loft-sh/vcluster/pkg/registry"
	"github.com/loft-sh/vcluster/pkg/util/kubeexec"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	loftctlUtil "github.com/loft-sh/loftctl/v3/pkg/util"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/find"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/flags"
	"github.com/loft-sh/vcluster/pkg/pro"
)

// PushCmd holds the cmd flags
type PushCmd struct {
	*flags.GlobalFlags

	kubeClient *kubernetes.Clientset
	kubeConfig *rest.Config
	Log        log.Logger
}

// NewPushCmd creates a new command
func NewPushCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &PushCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}

	cobraCmd := &cobra.Command{
		Use:   "push" + loftctlUtil.VClusterNameOnlyUseLine,
		Short: "Push a virtual cluster to OCI registry",
		Long: `
#######################################################
#################### vcluster push ####################
#######################################################
Push will package a vCluster (config, etcd & pvcs) and
push it to an OCI compliant registry.

Example:
vcluster push my-vcluster ghcr.io/my-user/my-repo:v1
#######################################################
	`,
		Args: cobra.ExactArgs(2),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context(), args)
		},
	}

	return cobraCmd
}

// Run executes the functionality
func (cmd *PushCmd) Run(ctx context.Context, args []string) error {
	// get pro client
	proClient, err := pro.CreateProClient()
	if err != nil {
		cmd.Log.Debugf("Error creating pro client: %v", err)
	}

	// find vcluster
	vClusterName := args[0]
	vCluster, proVCluster, err := find.GetVCluster(ctx, proClient, cmd.Context, vClusterName, cmd.Namespace, "", cmd.Log)
	if err != nil {
		return err
	} else if proVCluster != nil {
		return fmt.Errorf("currently not supported for pro clusters")
	}

	err = cmd.prepare(vCluster)
	if err != nil {
		return err
	}

	// select vCluster pod
	podList, err := cmd.kubeClient.CoreV1().Pods(vCluster.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "app=vcluster,release=" + vClusterName,
	})
	if err != nil {
		return err
	} else if len(podList.Items) == 0 {
		return fmt.Errorf("couldn't find a vCluster pod")
	}

	destination := args[1]
	registryName, _, _, err := oci.ParseReference(destination)
	if err != nil {
		return fmt.Errorf("error parsing destination: %w", err)
	}

	authConfig, err := registry.GetAuthConfig(registryName)
	if err == nil {
		cmd.Log.Debugf("Error finding auth config for %s: %w", registryName, err)
	}

	pushCommand := []string{"/vcluster", "push", "--destination", destination}
	if authConfig != nil && authConfig.Username != "" {
		pushCommand = append(pushCommand, "--username", authConfig.Username)
		pushCommand = append(pushCommand, "--password", authConfig.Secret)
	}

	// use this pod
	pod := podList.Items[0]
	cmd.Log.Infof("Use pod %s to push vCluster", pod.Name)
	err = kubeexec.Exec(ctx, cmd.kubeConfig, &kubeexec.ExecStreamOptions{
		Pod:       pod.Name,
		Namespace: pod.Namespace,
		Container: "syncer",
		Command:   pushCommand,
		Stdout:    os.Stdout,
		Stderr:    os.Stderr,
	})
	if err != nil {
		return fmt.Errorf("error pushing vCluster: %w", err)
	}

	cmd.Log.Donef("Successfully pushed vcluster %s/%s to %s", cmd.Namespace, args[0], args[1])
	return nil
}

func (cmd *PushCmd) prepare(vCluster *find.VCluster) error {
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
	cmd.kubeConfig = kubeConfig
	return nil
}
