package connect

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/loft-sh/loftctl/v3/pkg/client"
	"github.com/loft-sh/loftctl/v3/pkg/client/helper"
	"github.com/loft-sh/loftctl/v3/pkg/clihelper"
	"github.com/loft-sh/loftctl/v3/pkg/kube"
	"github.com/loft-sh/log"
	"github.com/sirupsen/logrus"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"

	managementv1 "github.com/loft-sh/api/v3/pkg/apis/management/v1"
	storagev1 "github.com/loft-sh/api/v3/pkg/apis/storage/v1"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/flags"
	"github.com/loft-sh/loftctl/v3/pkg/upgrade"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type ClusterCmd struct {
	Log log.Logger
	*flags.GlobalFlags
	Namespace      string
	ServiceAccount string
	DisplayName    string
	Context        string
	Wait           bool
}

// NewClusterCmd creates a new command
func NewClusterCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &ClusterCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}

	c := &cobra.Command{
		Use:   "cluster",
		Short: "connect current cluster to vCluster.Pro",
		Long: `#######################################################
############ vcluster pro connect cluster #############
#######################################################
Connect a cluster to the vCluster.Pro instance.

Example:
vcluster pro connect cluster my-cluster
########################################################
		`,
		Args: cobra.ExactArgs(1),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			// Check for newer version
			upgrade.PrintNewerVersionWarning()

			return cmd.Run(cobraCmd.Context(), args)
		},
	}

	c.Flags().StringVar(&cmd.Namespace, "namespace", "loft", "The namespace to generate the service account in. The namespace will be created if it does not exist")
	c.Flags().StringVar(&cmd.ServiceAccount, "service-account", "loft-admin", "The service account name to create")
	c.Flags().StringVar(&cmd.DisplayName, "display-name", "", "The display name to show in the UI for this cluster")
	c.Flags().BoolVar(&cmd.Wait, "wait", false, "If true, will wait until the cluster is initialized")
	c.Flags().StringVar(&cmd.Context, "context", "", "The kube context to use for installation")

	return c
}

func (cmd *ClusterCmd) Run(ctx context.Context, args []string) error {
	// Get clusterName from command argument
	clusterName := args[0]

	baseClient, err := client.NewClientFromPath(cmd.Config)
	if err != nil {
		return fmt.Errorf("new client from path: %w", err)
	}

	err = client.VerifyVersion(baseClient)
	if err != nil {
		return fmt.Errorf("verify loft version: %w", err)
	}

	managementClient, err := baseClient.Management()
	if err != nil {
		return fmt.Errorf("create management client: %w", err)
	}

	// get user details
	user, team, err := getUserOrTeam(ctx, managementClient)
	if err != nil {
		return fmt.Errorf("get user or team: %w", err)
	}

	loftVersion, err := baseClient.Version()
	if err != nil {
		return fmt.Errorf("get loft version: %w", err)
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
				NetworkPeer: true,
				Access:      getAccess(user, team),
			},
		},
	}, metav1.CreateOptions{})
	if err != nil && !kerrors.IsAlreadyExists(err) {
		return fmt.Errorf("create cluster: %w", err)
	}

	accessKey, err := managementClient.Loft().ManagementV1().Clusters().GetAccessKey(ctx, clusterName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("get cluster access key: %w", err)
	}

	namespace := cmd.Namespace

	helmArgs := []string{
		"upgrade", "--install", "loft", "loft",
		"--repo", "https://charts.loft.sh",
		"--create-namespace",
		"--namespace", namespace,
		"--set", "agentOnly=true",
	}

	if os.Getenv("DEVELOPMENT") == "true" {
		helmArgs = []string{
			"upgrade", "--install", "loft", "./chart",
			"--create-namespace",
			"--namespace", namespace,
			"--set", "agentOnly=true",
			"--set", "image=ghcr.io/loft-sh/enterprise:release-test",
		}
	} else if loftVersion.Version != "" {
		helmArgs = append(helmArgs, "--version", loftVersion.Version)
	}

	if accessKey.LoftHost != "" {
		helmArgs = append(helmArgs, "--set", "url="+accessKey.LoftHost)
	}

	if accessKey.AccessKey != "" {
		helmArgs = append(helmArgs, "--set", "token="+accessKey.AccessKey)
	}

	if accessKey.Insecure {
		helmArgs = append(helmArgs, "--set", "insecureSkipVerify=true")
	}

	if accessKey.CaCert != "" {
		helmArgs = append(helmArgs, "--set", "additionalCA="+accessKey.CaCert)
	}

	if cmd.Wait {
		helmArgs = append(helmArgs, "--wait")
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

	errChan := make(chan error)

	go func() {
		helmCmd := exec.CommandContext(ctx, "helm", helmArgs...)

		helmCmd.Stdout = cmd.Log.Writer(logrus.DebugLevel, true)
		helmCmd.Stderr = cmd.Log.Writer(logrus.DebugLevel, true)
		helmCmd.Stdin = os.Stdin

		cmd.Log.Info("Installing Loft agent...")
		cmd.Log.Debugf("Running helm command: %v", helmCmd.Args)

		err = helmCmd.Run()
		if err != nil {
			errChan <- fmt.Errorf("failed to install loft chart: %w", err)
		}

		close(errChan)
	}()

	_, err = clihelper.WaitForReadyLoftPod(ctx, clientset, namespace, cmd.Log)
	if err = errors.Join(err, <-errChan); err != nil {
		return fmt.Errorf("wait for loft pod: %w", err)
	}

	if cmd.Wait {
		cmd.Log.Info("Waiting for the cluster to be initialized...")
		waitErr := wait.PollUntilContextTimeout(ctx, time.Second, 5*time.Minute, false, func(ctx context.Context) (done bool, err error) {
			clusterInstance, err := managementClient.Loft().ManagementV1().Clusters().Get(ctx, clusterName, metav1.GetOptions{})
			if err != nil && !kerrors.IsNotFound(err) {
				return false, err
			}

			return clusterInstance.Status.Phase == storagev1.ClusterStatusPhaseInitialized, nil
		})
		if waitErr != nil {
			return fmt.Errorf("get cluster: %w", waitErr)
		}
	}

	cmd.Log.Donef("Successfully connected cluster %s to Loft", clusterName)

	return nil
}

func getUserOrTeam(ctx context.Context, managementClient kube.Interface) (string, string, error) {
	var user, team string

	userName, teamName, err := helper.GetCurrentUser(ctx, managementClient)
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
