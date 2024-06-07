package delete

import (
	"fmt"
	"strings"

	"github.com/loft-sh/vcluster/pkg/cli"
	"github.com/spf13/cobra"
)

func AddCommonFlags(cmd *cobra.Command, options *cli.DeleteOptions) {
	cmd.Flags().BoolVar(&options.Wait, "wait", true, "If enabled, vcluster will wait until the vcluster is deleted")
	cmd.Flags().BoolVar(&options.DeleteContext, "delete-context", true, "If the corresponding kube context should be deleted if there is any")
}

func AddHelmFlags(cmd *cobra.Command, options *cli.DeleteOptions) {
	cmd.Flags().BoolVar(&options.DeleteConfigMap, "delete-configmap", false, "If enabled, vCluster will delete the ConfigMap of the vCluster")
	cmd.Flags().BoolVar(&options.KeepPVC, "keep-pvc", false, "If enabled, vcluster will not delete the persistent volume claim of the vcluster")
	cmd.Flags().BoolVar(&options.DeleteNamespace, "delete-namespace", false, "If enabled, vcluster will delete the namespace of the vcluster. In the case of multi-namespace mode, will also delete all other namespaces created by vcluster")
	cmd.Flags().BoolVar(&options.AutoDeleteNamespace, "auto-delete-namespace", true, "If enabled, vcluster will delete the namespace of the vcluster if it was created by vclusterctl. In the case of multi-namespace mode, will also delete all other namespaces created by vcluster")
	cmd.Flags().BoolVar(&options.IgnoreNotFound, "ignore-not-found", false, "If enabled, vcluster will not error out in case the target vcluster does not exist")
}

func AddPlatformFlags(cmd *cobra.Command, options *cli.DeleteOptions, prefixes ...string) {
	prefix := strings.Join(prefixes, "")

	cmd.Flags().StringVar(&options.Project, "project", "", fmt.Sprintf("%sThe vCluster platform project to use", prefix))
}
