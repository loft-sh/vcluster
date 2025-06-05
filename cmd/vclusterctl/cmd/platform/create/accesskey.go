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
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type AccessKeyCmd struct {
	*flags.GlobalFlags

	ExpireAfter  string
	VClusterRole bool

	DisplayName string

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

	return c
}

func (cmd *AccessKeyCmd) Run(ctx context.Context, args []string) error {
	accessKeyName := args[0]
	cfg := cmd.LoadedConfig(cmd.Log)
	platformClient, err := platform.InitClientFromConfig(ctx, cfg)
	if err != nil {
		return err
	}

	managementClient, err := platformClient.Management()
	if err != nil {
		return err
	}

	self := platformClient.Self()
	if self.Status.User == nil {
		return fmt.Errorf("current user not found")
	}

	accessKey := &managementv1.OwnedAccessKey{
		ObjectMeta: metav1.ObjectMeta{
			Name: accessKeyName,
		},
		Spec: managementv1.OwnedAccessKeySpec{
			AccessKeySpec: storagev1.AccessKeySpec{
				Type: "User",
				User: self.Status.User.Name,
			},
		},
	}
	if cmd.DisplayName != "" {
		accessKey.Spec.DisplayName = cmd.DisplayName
	} else {
		accessKey.Spec.DisplayName = accessKeyName
	}
	if cmd.ExpireAfter != "" {
		duration, err := time.ParseDuration(cmd.ExpireAfter)
		if err != nil {
			return err
		}

		accessKey.Spec.TTL = int64(duration.Seconds())
	}
	if cmd.VClusterRole {
		accessKey.Spec.Scope = &storagev1.AccessKeyScope{}
		accessKey.Spec.Scope.Roles = []storagev1.AccessKeyScopeRole{
			{
				Role: storagev1.AccessKeyScopeRoleVCluster,
			},
		}
	}

	accessKey, err = managementClient.Loft().ManagementV1().OwnedAccessKeys().Create(ctx, accessKey, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	fmt.Println(accessKey.Spec.Key)
	return nil
}
