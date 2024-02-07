package connect

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/loft-sh/loftctl/v3/pkg/client"
	"github.com/loft-sh/loftctl/v3/pkg/client/helper"
	"github.com/loft-sh/loftctl/v3/pkg/clihelper"
	"github.com/loft-sh/loftctl/v3/pkg/config"
	"github.com/loft-sh/loftctl/v3/pkg/kube"
	"github.com/loft-sh/log"
	"github.com/sirupsen/logrus"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"

	managementv1 "github.com/loft-sh/api/v3/pkg/apis/management/v1"
	storagev1 "github.com/loft-sh/api/v3/pkg/apis/storage/v1"
	"github.com/loft-sh/api/v3/pkg/product"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/cmd/generate"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/flags"
	"github.com/loft-sh/loftctl/v3/pkg/kubeconfig"
	"github.com/loft-sh/loftctl/v3/pkg/upgrade"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type ClusterCmd struct {
	*flags.GlobalFlags
	Log             log.Logger
	Context         string
	DisplayName     string
	HelmChartPath   string
	Namespace       string
	Project         string
	ServiceAccount  string
	Description     string
	HelmSet         []string
	HelmValues      []string
	Development     bool
	EgressOnlyAgent bool
	Insecure        bool
	Wait            bool
}

// NewClusterCmd creates a new command
func NewClusterCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &ClusterCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}

	description := product.ReplaceWithHeader("connect cluster", `
Connect a cluster to the Loft instance.

Example:
loft connect cluster my-cluster
########################################################
	`)
	if upgrade.IsPlugin == "true" {
		description = `
########################################################
################ devspace connect cluster ##############
########################################################
Connect a cluster to the Loft instance.

Example:
devspace connect cluster my-cluster
########################################################
	`
	}
	c := &cobra.Command{
		Use:   "cluster",
		Short: product.Replace("connect current cluster to Loft"),
		Long:  description,
		Args:  cobra.ExactArgs(1),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			loader := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{CurrentContext: cmd.Context})
			c, err := loader.ClientConfig()
			if err != nil {
				return fmt.Errorf("get kube config: %w", err)
			}

			// Check for newer version
			upgrade.PrintNewerVersionWarning()

			return cmd.Run(cobraCmd.Context(), c, args)
		},
	}

	c.Flags().StringVar(&cmd.Namespace, "namespace", "loft", "The namespace to generate the service account in. The namespace will be created if it does not exist")
	c.Flags().StringVar(&cmd.ServiceAccount, "service-account", "loft-admin", "The service account name to create")
	c.Flags().StringVar(&cmd.DisplayName, "display-name", "", "The display name to show in the UI for this cluster")
	c.Flags().BoolVar(&cmd.Wait, "wait", false, "If true, will wait until the cluster is initialized")
	c.Flags().BoolVar(&cmd.EgressOnlyAgent, "egress-only-agent", true, "If true, will use an egress-only cluster enrollment feature")
	c.Flags().StringVar(&cmd.Context, "context", "", "The kube context to use for installation")
	c.Flags().StringVar(&cmd.Project, "project", "", "The project name to use for the project cluster")
	c.Flags().StringVar(&cmd.Description, "description", "", "The project name to use for the project cluster")

	c.Flags().StringVar(&cmd.HelmChartPath, "helm-chart-path", "", "The agent chart to deploy")
	c.Flags().StringArrayVar(&cmd.HelmSet, "helm-set", []string{}, "Extra helm values for the agent chart")
	c.Flags().StringArrayVar(&cmd.HelmValues, "helm-values", []string{}, "Extra helm values for the agent chart")

	c.Flags().BoolVar(&cmd.Insecure, "insecure", false, "If true, deploys the agent in insecure mode")
	c.Flags().BoolVar(&cmd.Development, "development", os.Getenv("DEVELOPMENT") == "true", "If the development chart should be deployed")

	_ = c.Flags().MarkHidden("development")
	_ = c.Flags().MarkHidden("description")
	return c
}

func (cmd *ClusterCmd) Run(ctx context.Context, localConfig *rest.Config, args []string) error {
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

	// check if we should connect via the new way
	if cmd.Project != "" || cmd.EgressOnlyAgent {
		// create new kube client
		kubeClient, err := kubernetes.NewForConfig(localConfig)
		if err != nil {
			return nil
		}

		// connect cluster
		err = cmd.connectCluster(ctx, baseClient, managementClient, kubeClient, clusterName, user, team)
		if err != nil {
			return err
		}
	} else {
		// get cluster config
		clusterConfig, err := getClusterKubeConfig(ctx, localConfig, cmd.Namespace, cmd.ServiceAccount)
		if err != nil {
			return fmt.Errorf("get cluster kubeconfig: %w", err)
		}

		// connect cluster
		_, err = managementClient.Loft().ManagementV1().ClusterConnects().Create(ctx, &managementv1.ClusterConnect{
			ObjectMeta: metav1.ObjectMeta{
				Name: clusterName,
			},
			Spec: managementv1.ClusterConnectSpec{
				Config:    clusterConfig.String(),
				AdminUser: user,
				ClusterTemplate: managementv1.Cluster{
					Spec: managementv1.ClusterSpec{
						ClusterSpec: storagev1.ClusterSpec{
							DisplayName: cmd.DisplayName,
							Owner: &storagev1.UserOrTeam{
								User: user,
								Team: team,
							},
							Access: getAccess(user, team),
						},
					},
				},
			},
		}, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("create cluster connect: %w", err)
		}
	}

	if cmd.Wait {
		cmd.Log.Info("Waiting for the cluster to be initialized...")
		waitErr := wait.PollUntilContextTimeout(ctx, time.Second, config.Timeout(), false, func(ctx context.Context) (done bool, err error) {
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

func (cmd *ClusterCmd) connectCluster(ctx context.Context, baseClient client.Client, managementClient kube.Interface, kubeClient kubernetes.Interface, clusterName string, user, team string) error {
	loftVersion, err := baseClient.Version()
	if err != nil {
		return fmt.Errorf("get loft version: %w", err)
	}

	_, err = managementClient.Loft().ManagementV1().Clusters().Create(ctx, &managementv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterName,
		},
		Spec: managementv1.ClusterSpec{
			ClusterSpec: storagev1.ClusterSpec{
				DisplayName: cmd.DisplayName,
				Description: cmd.Description,
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

	return cmd.deployAgent(ctx, kubeClient, loftVersion.Version, accessKey.LoftHost, accessKey.AccessKey, accessKey.Insecure, accessKey.CaCert)
}

func (cmd *ClusterCmd) deployAgent(ctx context.Context, kubeClient kubernetes.Interface, loftVersion, loftHost, accessKey string, insecure bool, caCert string) error {
	args := []string{
		"upgrade", "loft",
	}

	// check what type of deployment is it
	if cmd.Development {
		image := "ghcr.io/loft-sh/enterprise:release-test"
		if overrideImage, ok := os.LookupEnv("DEVELOPMENT_IMAGE"); ok {
			image = overrideImage
		}
		args = append(args, "./chart", "--set", "image="+image)
	} else if cmd.HelmChartPath != "" {
		args = append(args, cmd.HelmChartPath)
	} else {
		args = append(args, "loft", "--repo", "https://charts.loft.sh")

		// set version
		if loftVersion != "" {
			args = append(args, "--version", loftVersion)
		}
	}

	// general args
	args = append(args, "--install", "--create-namespace", "--namespace", cmd.Namespace, "--set", "agentOnly=true")

	// set host etc.
	if loftHost != "" {
		args = append(args, "--set", "url="+loftHost)
	}
	if accessKey != "" {
		args = append(args, "--set", "token="+accessKey)
	}

	// check if insecure
	if cmd.Insecure || insecure {
		args = append(args, "--set", "insecureSkipVerify=true")
	} else if caCert != "" {
		args = append(args, "--set", "additionalCA="+caCert)
	}

	for _, set := range cmd.HelmSet {
		args = append(args, "--set", set)
	}
	for _, values := range cmd.HelmValues {
		args = append(args, "--values", values)
	}
	if cmd.Wait {
		args = append(args, "--wait")
	}
	if cmd.Context != "" {
		args = append(args, "--kube-context", cmd.Context)
	}

	errChanHelm := make(chan error)
	go func() {
		defer close(errChanHelm)

		helmCmd := exec.CommandContext(ctx, "helm", args...)
		helmCmd.Stdout = cmd.Log.Writer(logrus.DebugLevel, true)
		helmCmd.Stderr = cmd.Log.Writer(logrus.DebugLevel, true)
		helmCmd.Stdin = os.Stdin

		cmd.Log.Infof("Installing Loft agent to connect to %s...", loftHost)
		cmd.Log.Debugf("Running helm command: %v", helmCmd.Args)

		err := helmCmd.Run()
		if err != nil {
			errChanHelm <- fmt.Errorf("failed to install loft chart: %w", err)
		}
	}()

	errChannWait := make(chan error)
	go func() {
		defer close(errChannWait)

		_, err := clihelper.WaitForReadyLoftPod(ctx, kubeClient, cmd.Namespace, cmd.Log)
		if err != nil {
			errChannWait <- fmt.Errorf("wait ready: %w", err)
		}
	}()

	select {
	case err := <-errChanHelm:
		if err != nil {
			return err
		}
		return <-errChannWait
	case err := <-errChannWait:
		if err != nil {
			return err
		}
		return <-errChanHelm
	}
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

func getClusterKubeConfig(ctx context.Context, c *rest.Config, namespace, serviceAccount string) (bytes.Buffer, error) {
	var clusterConfig bytes.Buffer

	token, err := generate.GetAuthToken(ctx, c, namespace, serviceAccount)
	if err != nil {
		return bytes.Buffer{}, fmt.Errorf("get auth token: %w", err)
	}

	err = kubeconfig.WriteTokenKubeConfig(c, string(token), &clusterConfig)
	if err != nil {
		return bytes.Buffer{}, fmt.Errorf("write token kubeconfig: %w", err)
	}

	return clusterConfig, nil
}
