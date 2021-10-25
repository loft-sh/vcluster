package cmd

import (
	"context"
	"encoding/json"
	"time"

	"github.com/loft-sh/vcluster/cmd/vclusterctl/flags"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// VCluster holds information about a cluster
type VCluster struct {
	Name       string
	Namespace  string
	Created    time.Time
	AgeSeconds int
}

// ListCmd holds the login cmd flags
type ListCmd struct {
	*flags.GlobalFlags

	log    log.Logger
	output string
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
vcluster list --output json
vcluster list --namespace test
#######################################################
	`,
		Args: cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd, args)
		},
	}

	cobraCmd.Flags().StringVar(&cmd.output, "output", "table", "Choose the format of the output. [table|json]")

	return cobraCmd
}

// Run executes the functionality
func (cmd *ListCmd) Run(cobraCmd *cobra.Command, args []string) error {
	// first load the kube config
	kubeClientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{
		CurrentContext: cmd.Context,
	})
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

	vclusters := []VCluster{}
	for _, s := range statefulSets.Items {
		vclusters = append(vclusters, VCluster{
			Name:       s.Name,
			Namespace:  s.Namespace,
			Created:    s.CreationTimestamp.Time,
			AgeSeconds: int(time.Since(s.CreationTimestamp.Time).Seconds()),
		})
	}

	if cmd.output == "json" {
		bytes, err := json.MarshalIndent(&vclusters, "", "    ")
		if err != nil {
			return errors.Wrap(err, "json marshal vclusters")
		}
		cmd.log.WriteString(string(bytes) + "\n")
	} else {
		header := []string{"NAME", "NAMESPACE", "CREATED", "AGE"}
		values := [][]string{}
		for _, vcluster := range vclusters {
			values = append(values, []string{
				vcluster.Name,
				vcluster.Namespace,
				vcluster.Created.String(),
				time.Since(vcluster.Created).Round(1 * time.Second).String(),
			})
		}

		log.PrintTable(cmd.log, header, values)
	}

	return nil
}
