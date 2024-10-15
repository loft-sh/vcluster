package add

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"

	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/cli/util"
	"github.com/loft-sh/vcluster/pkg/platform"
	"github.com/loft-sh/vcluster/pkg/platform/clihelper"
	"github.com/loft-sh/vcluster/pkg/platform/kube"
	"github.com/loft-sh/vcluster/pkg/upgrade"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type ClusterCmd struct {
	Log log.Logger
	*flags.GlobalFlags
	Namespace        string
	ServiceAccount   string
	DisplayName      string
	Context          string
	Insecure         bool
	Wait             bool
	HelmChartPath    string
	HelmChartVersion string
	HelmSet          []string
	HelmValues       []string
}

// NewClusterCmd creates a new command
func NewClusterCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &ClusterCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}

	c := &cobra.Command{
		Use:   "cluster",
		Short: "add current cluster to vCluster platform",
		Long: `#######################################################
############ vcluster platform add cluster ############
#######################################################
Adds a cluster to the vCluster platform instance.

Example:
vcluster platform add cluster my-cluster
########################################################
		`,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			newArgs, err := util.PromptForArgs(cmd.Log, args, "cluster name")
			if err != nil {
				switch {
				case errors.Is(err, util.ErrNonInteractive):
					if err := cobra.ExactArgs(1)(cobraCmd, args); err != nil {
						return err
					}
				default:
					return err
				}
			}

			// Check for newer version
			upgrade.PrintNewerVersionWarning()

			return cmd.Run(cobraCmd.Context(), newArgs)
		},
	}

	c.Flags().StringVar(&cmd.Namespace, "namespace", clihelper.DefaultPlatformNamespace, "The namespace to generate the service account in. The namespace will be created if it does not exist")
	c.Flags().StringVar(&cmd.ServiceAccount, "service-account", "loft-admin", "The service account name to create")
	c.Flags().StringVar(&cmd.DisplayName, "display-name", "", "The display name to show in the UI for this cluster")
	c.Flags().BoolVar(&cmd.Wait, "wait", false, "If true, will wait until the cluster is initialized")
	c.Flags().BoolVar(&cmd.Insecure, "insecure", false, "If true, deploys the agent in insecure mode")
	c.Flags().StringVar(&cmd.HelmChartVersion, "helm-chart-version", "", "The agent chart version to deploy")
	c.Flags().StringVar(&cmd.HelmChartPath, "helm-chart-path", "", "The agent chart to deploy")
	c.Flags().StringArrayVar(&cmd.HelmSet, "helm-set", []string{}, "Extra helm values for the agent chart")
	c.Flags().StringArrayVar(&cmd.HelmValues, "helm-values", []string{}, "Extra helm values for the agent chart")
	c.Flags().StringVar(&cmd.Context, "context", "", "The kube context to use for installation")

	return c
}

func (cmd *ClusterCmd) Run(ctx context.Context, args []string) error {
	// Get clusterName from command argument
	clusterName := args[0]
	platformClient, err := platform.InitClientFromConfig(ctx, cmd.LoadedConfig(cmd.Log))
	if err != nil {
		return fmt.Errorf("new client from path: %w", err)
	}

	managementClient, err := platformClient.Management()
	if err != nil {
		return fmt.Errorf("create management client: %w", err)
	}

	// get user details
	user, team, err := getUserOrTeam(ctx, managementClient)
	if err != nil {
		return fmt.Errorf("get user or team: %w", err)
	}

	loftVersion, err := platformClient.Version()
	if err != nil {
		return fmt.Errorf("get platform version: %w", err)
	}

	// TODO(ThomasK33): Eventually change this into an Apply instead of a Create call
	_, err = managementClient.Loft().ManagementV1().Clusters().Create(ctx, &managementv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterName,
		},
		Spec: managementv1.ClusterSpec{
			ClusterSpec: storagev1.ClusterSpec{
				DisplayName: cmd.DisplayName,
				Owner: &storagev1.UserOrTeam{
					User: user,
					Team: team,
				},
				NetworkPeer:         true,
				ManagementNamespace: cmd.Namespace,
				Access:              getAccess(user, team),
			},
		},
	}, metav1.CreateOptions{})
	if err != nil && !kerrors.IsAlreadyExists(err) {
		return fmt.Errorf("create cluster: %w", err)
	}

	// get namespace to install if cluster already exists
	if kerrors.IsAlreadyExists(err) {
		cluster, err := managementClient.Loft().ManagementV1().Clusters().Get(ctx, clusterName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("get cluster: %w", err)
		}

		cmd.Namespace = cluster.Spec.ManagementNamespace
		if cmd.Namespace == "" {
			cmd.Namespace = "loft" // since this is hardcoded in the platform at https://github.com/loft-sh/loft-enterprise/blob/b716f86a83d5f037ad993a0c3467b54393ef3b1f/pkg/util/agenthelper/helper.go#L9
		}

		cmd.Log.Infof("Using namespace %s because cluster already exists", cmd.Namespace)
	}

	accessKey, err := managementClient.Loft().ManagementV1().Clusters().GetAccessKey(ctx, clusterName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("get cluster access key: %w", err)
	}

	namespace := cmd.Namespace

	helmArgs := []string{
		"upgrade", "loft",
	}

	if os.Getenv("DEVELOPMENT") == "true" {
		helmArgs = []string{
			"upgrade", "--install", "loft", cmp.Or(os.Getenv("DEVELOPMENT_CHART_DIR"), "./chart"),
			"--create-namespace",
			"--namespace", namespace,
			"--set", "agentOnly=true",
			"--set", "image=" + cmp.Or(os.Getenv("DEVELOPMENT_IMAGE"), "ghcr.io/loft-sh/enterprise:release-test"),
		}
	} else {
		if cmd.HelmChartPath != "" {
			helmArgs = append(helmArgs, cmd.HelmChartPath)
		} else {
			helmArgs = append(helmArgs, "loft", "--repo", "https://charts.loft.sh")
		}

		if loftVersion.Version != "" {
			helmArgs = append(helmArgs, "--version", loftVersion.Version)
		}

		if cmd.HelmChartVersion != "" {
			helmArgs = append(helmArgs, "--version", cmd.HelmChartVersion)
		}

		// general arguments
		helmArgs = append(helmArgs, "--install", "--create-namespace", "--namespace", cmd.Namespace, "--set", "agentOnly=true")
	}

	for _, set := range cmd.HelmSet {
		helmArgs = append(helmArgs, "--set", set)
	}
	for _, values := range cmd.HelmValues {
		helmArgs = append(helmArgs, "--values", values)
	}

	if accessKey.LoftHost != "" {
		helmArgs = append(helmArgs, "--set", "url="+accessKey.LoftHost)
	}

	if accessKey.AccessKey != "" {
		helmArgs = append(helmArgs, "--set", "token="+accessKey.AccessKey)
	}

	if cmd.Insecure || accessKey.Insecure {
		helmArgs = append(helmArgs, "--set", "insecureSkipVerify=true")
	}

	if accessKey.CaCert != "" {
		helmArgs = append(helmArgs, "--set", "additionalCA="+accessKey.CaCert)
	}

	if cmd.Wait {
		helmArgs = append(helmArgs, "--wait")
	}

	if cmd.Context != "" {
		helmArgs = append(helmArgs, "--kube-context", cmd.Context)
	}

	kubeClientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{})

	if cmd.Context != "" {
		kubeConfig, err := kubeClientConfig.RawConfig()
		if err != nil {
			return fmt.Errorf("there is an error loading your current kube config (%w), please make sure you have access to a kubernetes cluster and the command `kubectl get namespaces` is working", err)
		}

		kubeClientConfig = clientcmd.NewNonInteractiveClientConfig(kubeConfig, cmd.Context, &clientcmd.ConfigOverrides{}, clientcmd.NewDefaultClientConfigLoadingRules())
	}

	config, err := kubeClientConfig.ClientConfig()
	if err != nil {
		return fmt.Errorf("there is an error loading your current kube config (%w), please make sure you have access to a kubernetes cluster and the command `kubectl get namespaces` is working", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("create kube client: %w", err)
	}

	agentAlreadyInstalled := true
	_, err = clientset.AppsV1().Deployments(cmd.Namespace).Get(ctx, "loft", metav1.GetOptions{})
	if err != nil {
		cmd.Log.Debugf("Error retrieving deployment: %v", err)
		agentAlreadyInstalled = false
	}

	helmCmd := exec.CommandContext(ctx, "helm", helmArgs...)

	helmCmd.Stdout = cmd.Log.Writer(logrus.DebugLevel, true)
	helmCmd.Stderr = cmd.Log.Writer(logrus.DebugLevel, true)
	helmCmd.Stdin = os.Stdin

	if agentAlreadyInstalled {
		cmd.Log.Info("Existing vCluster Platform agent found")
		cmd.Log.Info("Upgrading vCluster Platform agent...")
	} else {
		cmd.Log.Info("Installing vCluster Platform agent...")
	}
	cmd.Log.Debugf("Running helm command: %v", helmCmd.Args)

	err = helmCmd.Run()
	if err != nil {
		return fmt.Errorf("failed to install loft chart: %w", err)
	}

	_, err = clihelper.WaitForReadyLoftPod(ctx, clientset, namespace, cmd.Log)
	if err != nil {
		return fmt.Errorf("wait for loft pod: %w", err)
	}

	if cmd.Wait {
		cmd.Log.Info("Waiting for the cluster to be initialized...")
		waitErr := wait.PollUntilContextTimeout(ctx, time.Second, 5*time.Minute, false, func(ctx context.Context) (done bool, err error) {
			clusterInstance, err := managementClient.Loft().ManagementV1().Clusters().Get(ctx, clusterName, metav1.GetOptions{})
			if err != nil && !kerrors.IsNotFound(err) {
				return false, err
			}

			return clusterInstance != nil && clusterInstance.Status.Phase == storagev1.ClusterStatusPhaseInitialized, nil
		})
		if waitErr != nil {
			return fmt.Errorf("get cluster: %w", waitErr)
		}
	}

	if !agentAlreadyInstalled {
		cmd.Log.Donef("Successfully added cluster %s to the platform", clusterName)
	} else {
		cmd.Log.Donef("Successfully upgraded platform agent")
	}

	return nil
}

func getUserOrTeam(ctx context.Context, managementClient kube.Interface) (string, string, error) {
	var user, team string

	userName, teamName, err := platform.GetCurrentUser(ctx, managementClient)
	if err != nil {
		return "", "", fmt.Errorf("get current user: %w", err)
	}

	if userName != nil {
		user = userName.Name
	} else {
		team = teamName.Name
	}

	return user, team, nil
}

func getAccess(user, team string) []storagev1.Access {
	access := []storagev1.Access{
		{
			Verbs:        []string{"*"},
			Subresources: []string{"*"},
		},
	}

	if team != "" {
		access[0].Teams = []string{team}
	} else {
		access[0].Users = []string{user}
	}

	return access
}
