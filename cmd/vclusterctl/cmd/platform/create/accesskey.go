package create

import (
	"context"
	"fmt"
	"time"

	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	"github.com/loft-sh/api/v4/pkg/product"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/cli/util"
	"github.com/loft-sh/vcluster/pkg/platform"
	"github.com/loft-sh/vcluster/pkg/platform/kube"
	"github.com/loft-sh/vcluster/pkg/platform/random"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
)

type AccessKeyCmd struct {
	*flags.GlobalFlags

	ExpireAfter  string
	VClusterRole bool

	InCluster bool

	DisplayName string
	User        string

	Log log.Logger
}

func newAccessKeyCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &AccessKeyCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}
	description := product.ReplaceWithHeader("create access key", `
Creates a new access key for the current user.
Example:
vcluster platform create accesskey test
# To connect vClusters to the platform
vcluster platform create accesskey test --vcluster-role
vcluster platform create accesskey test --in-cluster --user admin
########################################################
	`)

	c := &cobra.Command{
		Use:   "accesskey" + util.NamespaceNameOnlyUseLine,
		Short: "Creates a new access key for the current user",
		Long:  description,
		Args:  util.NamespaceNameOnlyValidator,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context(), args)
		},
	}

	c.Flags().StringVar(&cmd.ExpireAfter, "expire-after", "", "The duration after which the access key will expire, e.g. 1h, 1d, 1w")
	c.Flags().BoolVar(&cmd.VClusterRole, "vcluster-role", false, "If true, the access key can be used to connect vClusters to the platform")
	c.Flags().StringVar(&cmd.DisplayName, "display-name", "", "The display name of the access key as shown in the UI")
	c.Flags().BoolVar(&cmd.InCluster, "in-cluster", false, "If true, the access key will be created in the current Kubernetes context instead of using the platform api. This allows access key creation without the need to be already logged in.")
	c.Flags().StringVar(&cmd.User, "user", "", "The user to create the access key for")

	return c
}

func (cmd *AccessKeyCmd) Run(ctx context.Context, args []string) error {
	accessKeyName := args[0]

	var accessKey string
	var err error
	if cmd.InCluster {
		accessKey, err = cmd.createAccessKeyInCluster(ctx, accessKeyName)
		if err != nil {
			return err
		}
	} else {
		accessKey, err = cmd.createAccessKeyPlatform(ctx, accessKeyName)
		if err != nil {
			return err
		}
	}

	fmt.Println(accessKey)
	return nil
}

func getClient(flags *flags.GlobalFlags) (kube.Interface, error) {
	// first load the kube config
	kubeClientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{
		CurrentContext: flags.Context,
	})

	// get the client config
	restConfig, err := kubeClientConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get client config: %w", err)
	}

	return kube.NewForConfig(restConfig)
}

func (cmd *AccessKeyCmd) createAccessKeyInCluster(ctx context.Context, accessKeyName string) (string, error) {
	client, err := getClient(cmd.GlobalFlags)
	if err != nil {
		return "", err
	}

	if cmd.User == "" {
		return "", fmt.Errorf("user is required when creating access key in cluster")
	}

	accessKey := &storagev1.AccessKey{
		ObjectMeta: metav1.ObjectMeta{
			Name: accessKeyName,
		},
		Spec: storagev1.AccessKeySpec{},
	}
	accessKey.Spec.Key = random.String(64)

	if err := cmd.fillAccessKey(&accessKey.Spec, accessKeyName); err != nil {
		return "", err
	}

	accessKey, err = client.Loft().StorageV1().AccessKeys().Create(ctx, accessKey, metav1.CreateOptions{})
	if err != nil {
		return "", err
	}

	return accessKey.Spec.Key, nil
}

func (cmd *AccessKeyCmd) createAccessKeyPlatform(ctx context.Context, accessKeyName string) (string, error) {
	cfg := cmd.LoadedConfig(cmd.Log)
	platformClient, err := platform.InitClientFromConfig(ctx, cfg)
	if err != nil {
		return "", err
	}

	managementClient, err := platformClient.Management()
	if err != nil {
		return "", err
	}

	if cmd.User == "" {
		self := platformClient.Self()
		if self.Status.User == nil {
			return "", fmt.Errorf("current user not found")
		}
		cmd.User = self.Status.User.Name
	}

	accessKey := &managementv1.OwnedAccessKey{
		ObjectMeta: metav1.ObjectMeta{
			Name: accessKeyName,
		},
		Spec: managementv1.OwnedAccessKeySpec{
			AccessKeySpec: storagev1.AccessKeySpec{},
		},
	}

	if err := cmd.fillAccessKey(&accessKey.Spec.AccessKeySpec, accessKeyName); err != nil {
		return "", err
	}

	accessKey, err = managementClient.Loft().ManagementV1().OwnedAccessKeys().Create(ctx, accessKey, metav1.CreateOptions{})
	if err != nil {
		return "", err
	}

	return accessKey.Spec.Key, nil
}

func (cmd *AccessKeyCmd) fillAccessKey(spec *storagev1.AccessKeySpec, accessKeyName string) error {
	spec.Type = "User"
	spec.User = cmd.User

	if cmd.DisplayName != "" {
		spec.DisplayName = cmd.DisplayName
	} else {
		spec.DisplayName = accessKeyName
	}
	if cmd.ExpireAfter != "" {
		duration, err := time.ParseDuration(cmd.ExpireAfter)
		if err != nil {
			return err
		}

		spec.TTL = int64(duration.Seconds())
	}
	if cmd.VClusterRole {
		spec.Scope = &storagev1.AccessKeyScope{}
		spec.Scope.Roles = []storagev1.AccessKeyScopeRole{
			{
				Role: storagev1.AccessKeyScopeRoleVCluster,
			},
		}
	}
	return nil
}
