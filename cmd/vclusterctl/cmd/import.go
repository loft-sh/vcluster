package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	managementv1 "github.com/loft-sh/api/v3/pkg/apis/management/v1"
	"github.com/loft-sh/loftctl/v3/pkg/client"
	loftctlUtil "github.com/loft-sh/loftctl/v3/pkg/util"
	"github.com/loft-sh/log"
	"github.com/loft-sh/log/survey"
	"github.com/loft-sh/log/terminal"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/find"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/flags"
	"github.com/loft-sh/vcluster/pkg/procli"
	"github.com/loft-sh/vcluster/pkg/util/compress"
	"github.com/mgutz/ansi"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type ImportCmd struct {
	*flags.GlobalFlags
	Log            log.Logger
	ClusterName    string
	Project        string
	ImportName     string
	DisableUpgrade bool
}

func NewImportCmd(globalFlags *flags.GlobalFlags) (*cobra.Command, error) {
	cmd := &ImportCmd{
		GlobalFlags: globalFlags,
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
			proClient, err := procli.CreateProClient()
			if err != nil {
				return err
			}

			if cmd.Project == "" {
				cmd.Project, err = getProjectName(cobraCmd.Context(), proClient)
				if err != nil {
					return err
				}
			}

			if cmd.ClusterName == "" {
				cmd.ClusterName, err = getClusterName(cobraCmd.Context(), globalFlags.Context)
				if err != nil {
					return err
				}
			}

			return cmd.Run(cobraCmd.Context(), args, proClient)
		},
	}

	importCmd.Flags().StringVar(&cmd.ClusterName, "cluster", "", "Cluster name of the cluster the virtual cluster is running on")
	importCmd.Flags().StringVar(&cmd.Project, "project", "", "The project to import the vcluster into")
	importCmd.Flags().StringVar(&cmd.ImportName, "importname", "", "The name of the vcluster under projects. If unspecified, will use the vcluster name")
	importCmd.Flags().BoolVar(&cmd.DisableUpgrade, "disable-upgrade", false, "If true, will disable auto-upgrade of the imported vcluster to vCluster.Pro")

	return importCmd, nil
}

// Run executes the functionality
func (cmd *ImportCmd) Run(ctx context.Context, args []string, proClient client.Client) error {
	// Get vClusterName from command argument
	vClusterName := args[0]

	managementClient, err := proClient.Management()
	if err != nil {
		return err
	}

	if cmd.Namespace == "" {
		cmd.Namespace, err = GetVClusterNamespace(ctx, cmd.Context, vClusterName, cmd.Namespace, cmd.Log)
		if err != nil {
			cmd.Log.Warnf("Error retrieving vcluster namespace: %v", err)
		}
	}

	if _, err = managementClient.Loft().ManagementV1().Projects().ImportVirtualCluster(ctx, cmd.Project, &managementv1.ProjectImportVirtualCluster{
		SourceVirtualCluster: managementv1.ProjectImportVirtualClusterSource{
			Name:       vClusterName,
			Namespace:  cmd.Namespace,
			Cluster:    cmd.ClusterName,
			ImportName: cmd.ImportName,
		},
		UpgradeToPro: !cmd.DisableUpgrade,
	}, metav1.CreateOptions{}); err != nil {
		return err
	}

	cmd.Log.Donef("Successfully imported vcluster %s into project %s", ansi.Color(vClusterName, "white+b"), ansi.Color(cmd.Project, "white+b"))

	return nil
}

func GetVClusterNamespace(ctx context.Context, context, name, namespace string, log log.Logger) (string, error) {
	if name == "" {
		return "", fmt.Errorf("please specify a name")
	}

	// list vclusters
	ossVClusters, err := find.ListOSSVClusters(ctx, context, name, namespace)
	if err != nil {
		log.Warnf("Error retrieving vclusters: %v", err)
		return "", err
	}

	// figure out what we want to return
	if len(ossVClusters) == 0 {
		return "", fmt.Errorf("couldn't find vcluster %s", name)
	} else if len(ossVClusters) == 1 {
		return ossVClusters[0].Namespace, nil
	}

	// check if terminal
	if !terminal.IsTerminalIn {
		return "", fmt.Errorf("multiple vclusters with name %s found, please specify a namespace via --namespace to select the correct one", name)
	}

	// ask a question
	questionOptionsUnformatted := [][]string{}
	for _, vCluster := range ossVClusters {
		questionOptionsUnformatted = append(questionOptionsUnformatted, []string{name, vCluster.Namespace})
	}

	questionOptions := find.FormatOptions("Name: %s | Namespace: %s ", questionOptionsUnformatted)
	selectedVCluster, err := log.Question(&survey.QuestionOptions{
		Question:     "Please choose a virtual cluster to use",
		DefaultValue: questionOptions[0],
		Options:      questionOptions,
	})
	if err != nil {
		return "", err
	}

	// match answer
	for idx, s := range questionOptions {
		if s == selectedVCluster {
			return ossVClusters[idx].Namespace, nil
		}
	}

	return "", fmt.Errorf("unexpected error searching for selected vcluster")
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
	} else if agentConfigSecret == nil {
		return nil, fmt.Errorf("could not determine current vCluster.Pro cluster, please make sure the current context is connected to vCluster.Pro. If this is not an error, you can also specify the cluster manually via '--cluster'")
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

func getProjectName(ctx context.Context, proClient client.Client) (string, error) {
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
