package cmd

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"os/exec"

	"github.com/loft-sh/vcluster/cmd/vclusterctl/flags"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/log"
	"github.com/loft-sh/vcluster/pkg/helm"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
)

// DeleteCmd holds the login cmd flags
type DeleteCmd struct {
	*flags.GlobalFlags

	KeepPVC         bool
	DeleteNamespace bool

	log log.Logger
}

// NewDeleteCmd creates a new command
func NewDeleteCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &DeleteCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}

	cobraCmd := &cobra.Command{
		Use:   "delete",
		Short: "Deletes a virtual cluster",
		Long: `
#######################################################
################### vcluster delete ###################
#######################################################
Deletes a virtual cluster

Example:
vcluster delete test --namespace test
#######################################################
	`,
		Args: cobra.ExactArgs(1),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd, args)
		},
	}

	cobraCmd.Flags().BoolVar(&cmd.KeepPVC, "keep-pvc", false, "If enabled, vcluster will not delete the persistent volume claim of the vcluster")
	cobraCmd.Flags().BoolVar(&cmd.DeleteNamespace, "delete-namespace", false, "If enabled, vcluster will delete the namespace of the vcluster")
	return cobraCmd
}

// Run executes the functionality
func (cmd *DeleteCmd) Run(cobraCmd *cobra.Command, args []string) error {
	// test for helm
	_, err := exec.LookPath("helm")
	if err != nil {
		return fmt.Errorf("seems like helm is not installed. Helm is required for the deletion of a virtual cluster. Please visit https://helm.sh/docs/intro/install/ for install instructions")
	}

	output, err := exec.Command("helm", "version").CombinedOutput()
	if err != nil {
		return fmt.Errorf("seems like there are issues with your helm client: \n\n%s", output)
	}

	// first load the kube config
	kubeClientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{
		CurrentContext: cmd.Context,
	})

	// load the raw config
	rawConfig, err := kubeClientConfig.RawConfig()
	if err != nil {
		return fmt.Errorf("there is an error loading your current kube config (%v), please make sure you have access to a kubernetes cluster and the command `kubectl get namespaces` is working", err)
	}
	if cmd.Context != "" {
		rawConfig.CurrentContext = cmd.Context
	}

	namespace, _, err := kubeClientConfig.Namespace()
	if err != nil {
		return err
	} else if namespace == "" {
		namespace = "default"
	}
	if cmd.Namespace != "" {
		namespace = cmd.Namespace
	}

	// we have to delete the chart
	err = helm.NewClient(&rawConfig, cmd.log).Delete(args[0], namespace)
	if err != nil {
		return err
	}
	cmd.log.Donef("Successfully deleted virtual cluster %s in namespace %s", args[0], namespace)

	// try to delete the pvc
	if cmd.KeepPVC == false {
		pvcName := fmt.Sprintf("data-%s-0", args[0])
		restConfig, err := kubeClientConfig.ClientConfig()
		if err != nil {
			return err
		}

		client, err := kubernetes.NewForConfig(restConfig)
		if err != nil {
			return err
		}

		err = client.CoreV1().PersistentVolumeClaims(namespace).Delete(context.Background(), pvcName, metav1.DeleteOptions{})
		if err != nil {
			if kerrors.IsNotFound(err) == false {
				return errors.Wrap(err, "delete pvc")
			}
		} else {
			cmd.log.Donef("Successfully deleted virtual cluster pvc %s in namespace %s", pvcName, namespace)
		}
	}

	// try to delete the namespace
	if cmd.DeleteNamespace {
		restConfig, err := kubeClientConfig.ClientConfig()
		if err != nil {
			return err
		}

		client, err := kubernetes.NewForConfig(restConfig)
		if err != nil {
			return err
		}

		err = client.CoreV1().Namespaces().Delete(context.Background(), namespace, metav1.DeleteOptions{})
		if err != nil {
			if kerrors.IsNotFound(err) == false {
				return errors.Wrap(err, "delete namespace")
			}
		} else {
			cmd.log.Donef("Successfully deleted virtual cluster namespace %s", namespace)
		}
	}

	return nil
}
