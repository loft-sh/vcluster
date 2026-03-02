package create

import (
	"fmt"
	"strings"

	"github.com/loft-sh/vcluster/pkg/cli"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/upgrade"
	"github.com/spf13/cobra"
)

const (
	FlagNameProject         = "project"
	FlagNameLabels          = "labels"
	FlagNameAnnotation      = "annotations"
	FlagNameCluster         = "cluster"
	FlagNameTemplate        = "template"
	FlagNameTemplateVersion = "template-version"
	FlagNameLinks           = "link"
	FlagNameParams          = "params"
	FlagNameParameters      = "parameters"
	FlagNameSetParams       = "set-params"
	FlagNameSetParameters   = "set-parameters"
	FlagNameDescription     = "description"
	FlagNameDisplayName     = "display-name"
	FlagNameTeam            = "team"
	FlagNameUser            = "user"
	FlagNameUseExisting     = "use"
	FlagNameRecreate        = "recreate"
	FlagNameSkipWait        = "skip-wait"
)

var platformFlags = []string{FlagNameProject, FlagNameLabels, FlagNameAnnotation, FlagNameCluster, FlagNameTemplate, FlagNameTemplateVersion, FlagNameLinks, FlagNameParams,
	FlagNameParameters, FlagNameSetParams, FlagNameSetParameters, FlagNameDescription, FlagNameDisplayName, FlagNameTeam, FlagNameUser, FlagNameUseExisting, FlagNameRecreate, FlagNameSkipWait,
}

func AddCommonFlags(cmd *cobra.Command, options *cli.CreateOptions) {
	cmd.Flags().StringVar(&options.KubeConfigContextName, "kube-config-context-name", "", "If set, will override the context name of the generated virtual cluster kube config with this name")
	cmd.Flags().StringVar(&options.ChartVersion, "chart-version", upgrade.GetVersion(), "The virtual cluster chart version to use (e.g. v0.9.1)")
	cmd.Flags().StringVar(&options.ChartName, "chart-name", "vcluster", "The virtual cluster chart name to use")
	cmd.Flags().StringVar(&options.ChartRepo, "chart-repo", constants.LoftChartRepo, "The virtual cluster chart repo to use")
	cmd.Flags().StringArrayVarP(&options.Values, "values", "f", []string{}, "Path where to load extra helm values from")
	cmd.Flags().StringArrayVar(&options.SetValues, "set", []string{}, "Set values for helm. E.g. --set 'persistence.enabled=true'")
	cmd.Flags().BoolVar(&options.Print, "print", false, "If enabled, prints the context to the console")
	cmd.Flags().BoolVar(&options.UpdateCurrent, "update-current", true, "If true updates the current kube config")
	cmd.Flags().BoolVar(&options.Expose, "expose", false, "If true will create a load balancer service to expose the vcluster endpoint")
	cmd.Flags().BoolVar(&options.Connect, "connect", true, "If true will run vcluster connect directly after the vcluster was created")
	cmd.Flags().BoolVar(&options.Upgrade, "upgrade", false, "If true will try to upgrade the vcluster instead of failing if it already exists")

	_ = cmd.Flags().MarkHidden("update-current")
	_ = cmd.Flags().MarkDeprecated("update-current", fmt.Sprintf("please use %q.", "--connect"))
}

func AddHelmFlags(cmd *cobra.Command, options *cli.CreateOptions) {
	cmd.Flags().BoolVar(&options.CreateNamespace, "create-namespace", true, "If true the namespace will be created if it does not exist")
	cmd.Flags().StringVar(&options.LocalChartDir, "local-chart-dir", "", "The virtual cluster local chart dir to use")
	cmd.Flags().StringVar(&options.Restore, "restore", "", "Restore the virtual cluster from a backup. E.g. --restore oci://ghcr.io/my-user/my-repo:my-tag")
	cmd.Flags().BoolVar(&options.ExposeLocal, "expose-local", true, "If true and a local Kubernetes distro is detected, will deploy vcluster with a NodePort service. Will be set to false and the passed value will be ignored if --expose is set to true.")
	cmd.Flags().BoolVar(&options.BackgroundProxy, "background-proxy", true, "Try to use a background-proxy to access the vCluster. Only works if docker is installed and reachable")
	cmd.Flags().StringVar(&options.BackgroundProxyImage, "background-proxy-image", constants.DefaultBackgroundProxyImage(upgrade.GetVersion()), "The image to use for the background proxy. Only used if --background-proxy is enabled.")
	cmd.Flags().BoolVar(&options.Add, "add", true, "Adds the virtual cluster automatically to the current vCluster platform when using helm driver")

	_ = cmd.Flags().MarkHidden("local-chart-dir")
	_ = cmd.Flags().MarkHidden("expose-local")
}

func AddPlatformFlags(cmd *cobra.Command, options *cli.CreateOptions, prefixes ...string) {
	prefix := strings.Join(prefixes, "")

	cmd.Flags().StringVar(&options.Project, FlagNameProject, "", fmt.Sprintf("%sThe vCluster platform project to use", prefix))
	cmd.Flags().StringSliceVarP(&options.Labels, FlagNameLabels, "l", []string{}, fmt.Sprintf("%sComma separated labels to apply to the virtualclusterinstance", prefix))
	cmd.Flags().StringSliceVar(&options.Annotations, FlagNameAnnotation, []string{}, fmt.Sprintf("%sComma separated annotations to apply to the virtualclusterinstance", prefix))
	cmd.Flags().StringVar(&options.Cluster, FlagNameCluster, "", fmt.Sprintf("%sThe vCluster platform connected cluster to use", prefix))
	cmd.Flags().StringVar(&options.Template, FlagNameTemplate, "", fmt.Sprintf("%sThe vCluster platform template to use", prefix))
	cmd.Flags().StringVar(&options.TemplateVersion, FlagNameTemplateVersion, "", fmt.Sprintf("%sThe vCluster platform template version to use", prefix))
	cmd.Flags().StringArrayVar(&options.Links, FlagNameLinks, []string{}, fmt.Sprintf("%sA link to add to the vCluster. E.g. --link 'prod=http://exampleprod.com'", prefix))
	cmd.Flags().StringVar(&options.Params, FlagNameParams, "", fmt.Sprintf("%sIf a template is used, this can be used to use a file for the parameters. E.g. --params path/to/my/file.yaml", prefix))
	cmd.Flags().StringVar(&options.Params, FlagNameParameters, "", fmt.Sprintf("%sIf a template is used, this can be used to use a file for the parameters. E.g. --parameters path/to/my/file.yaml", prefix))
	cmd.Flags().StringArrayVar(&options.SetParams, FlagNameSetParams, []string{}, fmt.Sprintf("%sIf a template is used, this can be used to set a specific parameter. E.g. --set-param 'my-param=my-value'", prefix))
	cmd.Flags().StringArrayVar(&options.SetParams, FlagNameSetParameters, []string{}, fmt.Sprintf("%sIf a template is used, this can be used to set a specific parameter. E.g. --set-parameter 'my-param=my-value'", prefix))
	cmd.Flags().StringVar(&options.Description, FlagNameDescription, "", fmt.Sprintf("%sThe description to show in the platform UI for this virtual cluster", prefix))
	cmd.Flags().StringVar(&options.DisplayName, FlagNameDisplayName, "", fmt.Sprintf("%sThe display name to show in the platform UI for this virtual cluster", prefix))
	cmd.Flags().StringVar(&options.Team, FlagNameTeam, "", fmt.Sprintf("%sThe team to create the space for", prefix))
	cmd.Flags().StringVar(&options.User, FlagNameUser, "", fmt.Sprintf("%sThe user to create the space for", prefix))
	cmd.Flags().BoolVar(&options.UseExisting, FlagNameUseExisting, false, fmt.Sprintf("%sIf the platform should use the virtual cluster if its already there", prefix))
	cmd.Flags().BoolVar(&options.Recreate, FlagNameRecreate, false, fmt.Sprintf("%sIf enabled and there already exists a virtual cluster with this name, it will be deleted first", prefix))
	cmd.Flags().BoolVar(&options.SkipWait, FlagNameSkipWait, false, fmt.Sprintf("%sIf true, will not wait until the virtual cluster is running", prefix))
}

func ChangedPlatformFlags(cmd *cobra.Command) map[string]bool {
	return flags.ChangedFlags(cmd, platformFlags)
}
