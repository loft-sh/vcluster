package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	managementv1 "github.com/loft-sh/api/v3/pkg/apis/management/v1"
	loftctlImport "github.com/loft-sh/loftctl/v3/cmd/loftctl/cmd/importcmd"
	loftctlUtil "github.com/loft-sh/loftctl/v3/pkg/util"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/flags"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/util/compress"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func NewImportCmd(globalFlags *flags.GlobalFlags) (*cobra.Command, error) {
	loftctlGlobalFlags, err := pro.GlobalFlags(globalFlags)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pro flags: %w", err)
	}

	cmd := &loftctlImport.VClusterCmd{
		GlobalFlags: loftctlGlobalFlags,
		Log:         log.GetInstance(),
	}

	description := `########################################################
################### vcluster import ####################
########################################################
Imports a vcluster into a vCluster.Pro project.

Example:
vcluster import my-vcluster --cluster connected-cluster \
--namespace vcluster-my-vcluster --project my-project --importname my-vcluster
#######################################################
	`

	importCmd := &cobra.Command{
		Use:   "import" + loftctlUtil.VClusterNameOnlyUseLine,
		Short: "Imports a vcluster into a vCluster.Pro project",
		Long:  description,
		Args:  loftctlUtil.VClusterNameOnlyValidator,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			if cmd.Project == "" {
				cmd.Project, err = getProjectName(cobraCmd.Context())
				if err != nil {
					return err
				}
			}

			if cmd.VClusterClusterName == "" {
				cmd.VClusterClusterName, err = getClusterName(cobraCmd.Context(), globalFlags.Context)
				if err != nil {
					return err
				}
			}

			return cmd.Run(cobraCmd.Context(), args)
		},
	}

	importCmd.Flags().StringVar(&cmd.VClusterClusterName, "cluster", "", "Cluster name of the cluster the virtual cluster is running on")
	importCmd.Flags().StringVar(&cmd.VClusterNamespace, "namespace", "", "The namespace of the vcluster")
	importCmd.Flags().StringVar(&cmd.Project, "project", "", "The project to import the vcluster into")
	importCmd.Flags().StringVar(&cmd.ImportName, "importname", "", "The name of the vcluster under projects. If unspecified, will use the vcluster name")

	_ = importCmd.MarkFlagRequired("namespace")

	return importCmd, nil
}

func getClusterName(ctx context.Context, kubeContext string) (string, error) {
	config, err := getAgentConfig(ctx, kubeContext)
	if err != nil {
		return "", err
	}

	if config == nil {
		return "", fmt.Errorf("could not find an agent config")
	}

	return config.Cluster, nil
}

func getAgentConfig(ctx context.Context, kubeContext string) (*managementv1.AgentLoftAccess, error) {
	agentConfigSecret, err := findSecret(ctx, kubeContext, "loft-agent-config")
	if err != nil {
		return nil, err
	}

	if agentConfigSecret == nil {
		return nil, fmt.Errorf("could not find agent config secret")
	}

	// get data
	data := []byte{}
	for _, d := range agentConfigSecret.Data {
		data = d
	}

	configString, err := compress.UncompressBytes(data)
	if err != nil {
		return nil, err
	}

	var agentConfig managementv1.AgentLoftAccess
	err = json.Unmarshal([]byte(configString), &agentConfig)
	if err != nil {
		return nil, err
	}

	return &agentConfig, nil
}

func findSecret(ctx context.Context, kubeContext, secretName string) (*corev1.Secret, error) {
	// first load the kube config
	kubeClientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{
		CurrentContext: kubeContext,
	})

	// load the rest config
	kubeConfig, err := kubeClientConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("there is an error loading your current kube config (%w), please make sure you have access to a kubernetes cluster and the command `kubectl get namespaces` is working", err)
	}

	client, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return nil, err
	}

	namespaces, err := client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, namespace := range namespaces.Items {
		secret, err := client.CoreV1().Secrets(namespace.Name).Get(ctx, secretName, metav1.GetOptions{})
		if err == nil {
			return secret, nil
		} else if !kerrors.IsNotFound(err) {
			return nil, err
		}
	}

	return nil, nil
}

func getProjectName(ctx context.Context) (string, error) {
	proClient, err := pro.CreateProClient()
	if err != nil {
		return "", err
	}

	managementClient, err := proClient.Management()
	if err != nil {
		return "", err
	}

	projectList, err := managementClient.Loft().ManagementV1().Projects().List(ctx, metav1.ListOptions{})
	if err != nil {
		return "", err
	}

	if len(projectList.Items) > 1 {
		return "", fmt.Errorf("please specify a project via --project ")
	} else if len(projectList.Items) == 0 {
		return "", fmt.Errorf("no projects found. Please create a project and specify via --project ")
	}

	return projectList.Items[0].Name, nil
}
