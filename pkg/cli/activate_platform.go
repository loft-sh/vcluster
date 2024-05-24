package cli

import (
	"context"
	"encoding/json"
	"fmt"

	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	"github.com/loft-sh/log"
	"github.com/loft-sh/log/survey"
	"github.com/loft-sh/log/terminal"
	"github.com/loft-sh/vcluster/pkg/cli/find"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/platform"
	"github.com/loft-sh/vcluster/pkg/util/compress"
	"github.com/mgutz/ansi"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func ActivatePlatform(ctx context.Context, options *ActivateOptions, globalFlags *flags.GlobalFlags, vClusterName string, log log.Logger) error {
	platformClient, err := platform.CreatePlatformClient()
	if err != nil {
		return err
	}

	if options.Project == "" {
		options.Project, err = getProjectName(ctx, platformClient)
		if err != nil {
			return err
		}
	}

	if options.ClusterName == "" {
		options.ClusterName, err = getClusterName(ctx, globalFlags.Context)
		if err != nil {
			return err
		}
	}

	managementClient, err := platformClient.Management()
	if err != nil {
		return err
	}

	if globalFlags.Namespace == "" {
		globalFlags.Namespace, err = GetVClusterNamespace(ctx, globalFlags.Context, vClusterName, globalFlags.Namespace, log)
		if err != nil {
			log.Warnf("Error retrieving vCluster namespace: %v", err)
		}
	}

	if _, err = managementClient.Loft().ManagementV1().Projects().ImportVirtualCluster(ctx, options.Project, &managementv1.ProjectImportVirtualCluster{
		SourceVirtualCluster: managementv1.ProjectImportVirtualClusterSource{
			Name:       vClusterName,
			Namespace:  globalFlags.Namespace,
			Cluster:    options.ClusterName,
			ImportName: options.ImportName,
		},
	}, metav1.CreateOptions{}); err != nil {
		return err
	}

	log.Donef("Successfully imported vCluster %s into project %s", ansi.Color(vClusterName, "white+b"), ansi.Color(options.Project, "white+b"))
	return nil
}

func GetVClusterNamespace(ctx context.Context, context, name, namespace string, log log.Logger) (string, error) {
	if name == "" {
		return "", fmt.Errorf("please specify a name")
	}

	// list virtual clusters
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

	return "", fmt.Errorf("unexpected error searching for selected virtual cluster")
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
		return nil, fmt.Errorf("could not determine connected vCluster platform cluster, please make sure the current context is connected to vCluster platform. If this is not an error, you can also specify the cluster manually via '--cluster'")
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

func getProjectName(ctx context.Context, platformClient platform.Client) (string, error) {
	managementClient, err := platformClient.Management()
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
