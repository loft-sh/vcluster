package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli"
	"github.com/loft-sh/vcluster/pkg/cli/config"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/cli/util"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/platform"
	"github.com/loft-sh/vcluster/pkg/upgrade"
	"github.com/spf13/cobra"
)

// CreateCmd holds the login cmd flags
type CreateCmd struct {
	*flags.GlobalFlags
	cli.CreateOptions

	log log.Logger
}

// NewCreateCmd creates a new command
func NewCreateCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &CreateCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}

	cobraCmd := &cobra.Command{
		Use:   "create" + util.VClusterNameOnlyUseLine,
		Short: "Create a new virtual cluster",
		Long: `#######################################################
################### vcluster create ###################
#######################################################
Creates a new virtual cluster

Example:
vcluster create test --namespace test
#######################################################
	`,
		Args: util.VClusterNameOnlyValidator,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			// Check for newer version
			upgrade.PrintNewerVersionWarning()

			return cmd.Run(cobraCmd.Context(), args)
		},
	}

	// generic flags
	cobraCmd.Flags().StringVar(&cmd.Manager, "manager", "", "The manager to use for managing the virtual cluster, can be either helm or platform.")
	cobraCmd.Flags().StringVar(&cmd.KubeConfigContextName, "kube-config-context-name", "", "If set, will override the context name of the generated virtual cluster kube config with this name")
	cobraCmd.Flags().StringVar(&cmd.ChartVersion, "chart-version", upgrade.GetVersion(), "The virtual cluster chart version to use (e.g. v0.9.1)")
	cobraCmd.Flags().StringVar(&cmd.ChartName, "chart-name", "vcluster", "The virtual cluster chart name to use")
	cobraCmd.Flags().StringVar(&cmd.ChartRepo, "chart-repo", constants.LoftChartRepo, "The virtual cluster chart repo to use")
	cobraCmd.Flags().StringVar(&cmd.KubernetesVersion, "kubernetes-version", "", "The kubernetes version to use (e.g. v1.20). Patch versions are not supported")
	cobraCmd.Flags().StringArrayVarP(&cmd.Values, "values", "f", []string{}, "Path where to load extra helm values from")
	cobraCmd.Flags().StringArrayVar(&cmd.SetValues, "set", []string{}, "Set values for helm. E.g. --set 'persistence.enabled=true'")
	cobraCmd.Flags().BoolVar(&cmd.Print, "print", false, "If enabled, prints the context to the console")

	cobraCmd.Flags().BoolVar(&cmd.CreateNamespace, "create-namespace", true, "If true the namespace will be created if it does not exist")
	cobraCmd.Flags().BoolVar(&cmd.UpdateCurrent, "update-current", true, "If true updates the current kube config")
	cobraCmd.Flags().BoolVar(&cmd.CreateContext, "create-context", true, "If the CLI should create a kube context for the space")
	cobraCmd.Flags().BoolVar(&cmd.SwitchContext, "switch-context", true, "If the CLI should switch the current context to the new context")
	cobraCmd.Flags().BoolVar(&cmd.Expose, "expose", false, "If true will create a load balancer service to expose the vcluster endpoint")
	cobraCmd.Flags().BoolVar(&cmd.Connect, "connect", true, "If true will run vcluster connect directly after the vcluster was created")
	cobraCmd.Flags().BoolVar(&cmd.Upgrade, "upgrade", false, "If true will try to upgrade the vcluster instead of failing if it already exists")
	cobraCmd.Flags().BoolVar(&cmd.Upgrade, "update", false, "If true will try to upgrade the vcluster instead of failing if it already exists")

	// Platform flags
	cobraCmd.Flags().BoolVar(&cmd.Activate, "activate", true, "[PLATFORM] Activate the vCluster automatically when using helm manager")
	cobraCmd.Flags().StringVar(&cmd.Project, "project", "", "[PLATFORM] The vCluster platform project to use")
	cobraCmd.Flags().StringSliceVarP(&cmd.Labels, "labels", "l", []string{}, "[PLATFORM] Comma separated labels to apply to the virtualclusterinstance")
	cobraCmd.Flags().StringSliceVar(&cmd.Annotations, "annotations", []string{}, "[PLATFORM] Comma separated annotations to apply to the virtualclusterinstance")
	cobraCmd.Flags().StringVar(&cmd.Cluster, "cluster", "", "[PLATFORM] The vCluster platform connected cluster to use")
	cobraCmd.Flags().StringVar(&cmd.Template, "template", "", "[PLATFORM] The vCluster platform template to use")
	cobraCmd.Flags().StringVar(&cmd.TemplateVersion, "template-version", "", "[PLATFORM] The vCluster platform template version to use")
	cobraCmd.Flags().StringArrayVar(&cmd.Links, "link", []string{}, "[PLATFORM] A link to add to the vCluster. E.g. --link 'prod=http://exampleprod.com'")
	cobraCmd.Flags().StringVar(&cmd.Params, "params", "", "[PLATFORM] If a template is used, this can be used to use a file for the parameters. E.g. --params path/to/my/file.yaml")
	cobraCmd.Flags().StringVar(&cmd.Params, "parameters", "", "[PLATFORM] If a template is used, this can be used to use a file for the parameters. E.g. --parameters path/to/my/file.yaml")
	cobraCmd.Flags().StringArrayVar(&cmd.SetParams, "set-param", []string{}, "[PLATFORM] If a template is used, this can be used to set a specific parameter. E.g. --set-param 'my-param=my-value'")
	cobraCmd.Flags().StringArrayVar(&cmd.SetParams, "set-parameter", []string{}, "[PLATFORM] If a template is used, this can be used to set a specific parameter. E.g. --set-parameter 'my-param=my-value'")
	cobraCmd.Flags().StringVar(&cmd.Description, "description", "", "[PLATFORM] The description to show in the platform UI for this virtual cluster")
	cobraCmd.Flags().StringVar(&cmd.DisplayName, "display-name", "", "[PLATFORM] The display name to show in the platform UI for this virtual cluster")
	cobraCmd.Flags().StringVar(&cmd.Team, "team", "", "[PLATFORM] The team to create the space for")
	cobraCmd.Flags().StringVar(&cmd.User, "user", "", "[PLATFORM] The user to create the space for")
	cobraCmd.Flags().BoolVar(&cmd.UseExisting, "use", false, "[PLATFORM] If the platform should use the virtual cluster if its already there")
	cobraCmd.Flags().BoolVar(&cmd.Recreate, "recreate", false, "[PLATFORM] If enabled and there already exists a virtual cluster with this name, it will be deleted first")
	cobraCmd.Flags().BoolVar(&cmd.SkipWait, "skip-wait", false, "[PLATFORM] If true, will not wait until the virtual cluster is running")

	// hidden / deprecated
	cobraCmd.Flags().StringVar(&cmd.LocalChartDir, "local-chart-dir", "", "The virtual cluster local chart dir to use")
	cobraCmd.Flags().BoolVar(&cmd.ExposeLocal, "expose-local", true, "If true and a local Kubernetes distro is detected, will deploy vcluster with a NodePort service. Will be set to false and the passed value will be ignored if --expose is set to true.")
	cobraCmd.Flags().StringVar(&cmd.Distro, "distro", "k8s", fmt.Sprintf("Kubernetes distro to use for the virtual cluster. Allowed distros: %s", strings.Join(cli.AllowedDistros, ", ")))

	_ = cobraCmd.Flags().MarkHidden("local-chart-dir")
	_ = cobraCmd.Flags().MarkHidden("expose-local")
	_ = cobraCmd.Flags().MarkHidden("distro")
	_ = cobraCmd.Flags().MarkDeprecated("distro", fmt.Sprintf("please specify the distro by setting %q accordingly via values.yaml file.", "controlPlane.distro"))
	return cobraCmd
}

// Run executes the functionality
func (cmd *CreateCmd) Run(ctx context.Context, args []string) error {
	cfg := cmd.LoadedConfig(cmd.log)

	// check if there is a platform client or we skip the info message
	_, err := platform.InitClientFromConfig(ctx, cfg)
	if err == nil {
		config.PrintManagerInfo("create", cfg.Manager.Type, cmd.log)
	}
	// check if we should create a platform vCluster
	if cfg.Manager.Type == config.ManagerPlatform {
		return cli.CreatePlatform(ctx, &cmd.CreateOptions, cmd.GlobalFlags, args[0], cmd.log)
	}

	return cli.CreateHelm(ctx, &cmd.CreateOptions, cmd.GlobalFlags, args[0], cmd.log)
}
