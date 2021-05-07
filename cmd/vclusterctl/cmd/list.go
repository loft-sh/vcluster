package cmd

import (
	"context"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/flags"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// ListCmd holds the login cmd flags
type ListCmd struct {
	*flags.GlobalFlags

	Namespace string
	log       log.Logger
}

// NewListCmd creates a new command
func NewListCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &ListCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}

	cobraCmd := &cobra.Command{
		Use:   "list",
		Short: "Lists all virtual clusters",
		Long: `
#######################################################
#################### vcluster list ####################
#######################################################
Lists all virtual clusters

Example:
vcluster list
vcluster list --namespace test
#######################################################
	`,
		Args: cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd, args)
		},
	}

	cobraCmd.Flags().StringVarP(&cmd.Namespace, "namespace", "n", "", "The namespace the vcluster was created in")
	return cobraCmd
}

// Run executes the functionality
func (cmd *ListCmd) Run(cobraCmd *cobra.Command, args []string) error {
	// first load the kube config
	kubeClientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{})
	namespace := metav1.NamespaceAll
	if cmd.Namespace != "" {
		namespace = cmd.Namespace
	}

	// get all statefulsets with the label app=vcluster
	restConfig, err := kubeClientConfig.ClientConfig()
	if err != nil {
		return err
	}
	client, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	statefulSets, err := client.AppsV1().StatefulSets(namespace).List(context.Background(), metav1.ListOptions{LabelSelector: "app=vcluster"})
	if err != nil {
		if kerrors.IsForbidden(err) {
			// try the current namespace instead
			namespace, _, err = kubeClientConfig.Namespace()
			if err != nil {
				return err
			} else if namespace == "" {
				namespace = "default"
			}

			statefulSets, err = client.AppsV1().StatefulSets(namespace).List(context.Background(), metav1.ListOptions{LabelSelector: "app=vcluster"})
			if err != nil {
				return err
			}
		} else {
			return errors.Wrap(err, "list stateful sets")
		}
	}

	header := []string{"NAME", "NAMESPACE", "CREATED"}
	values := [][]string{}
	for _, s := range statefulSets.Items {
		values = append(values, []string{
			s.Name,
			s.Namespace,
			s.CreationTimestamp.String(),
		})
	}

	log.PrintTable(cmd.log, header, values)
	return nil
}
