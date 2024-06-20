package create

import (
	"fmt"
	"strings"

	"github.com/loft-sh/vcluster/pkg/cli"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/upgrade"
	"github.com/spf13/cobra"
)

func AddCommonFlags(cmd *cobra.Command, options *cli.CreateOptions) {
	cmd.Flags().StringVar(&options.KubeConfigContextName, "kube-config-context-name", "", "If set, will override the context name of the generated virtual cluster kube config with this name")
	cmd.Flags().StringVar(&options.ChartVersion, "chart-version", upgrade.GetVersion(), "The virtual cluster chart version to use (e.g. v0.9.1)")
	cmd.Flags().StringVar(&options.ChartName, "chart-name", "vcluster", "The virtual cluster chart name to use")
	cmd.Flags().StringVar(&options.ChartRepo, "chart-repo", constants.LoftChartRepo, "The virtual cluster chart repo to use")
	cmd.Flags().StringVar(&options.KubernetesVersion, "kubernetes-version", "", "The kubernetes version to use (e.g. v1.20). Patch versions are not supported")
	cmd.Flags().StringArrayVarP(&options.Values, "values", "f", []string{}, "Path where to load extra helm values from")
	cmd.Flags().StringArrayVar(&options.SetValues, "set", []string{}, "Set values for helm. E.g. --set 'persistence.enabled=true'")
	cmd.Flags().BoolVar(&options.Print, "print", false, "If enabled, prints the context to the console")
	cmd.Flags().BoolVar(&options.UpdateCurrent, "update-current", true, "If true updates the current kube config")
	cmd.Flags().BoolVar(&options.CreateContext, "create-context", true, "If the CLI should create a kube context for the space")
	cmd.Flags().BoolVar(&options.SwitchContext, "switch-context", true, "If the CLI should switch the current context to the new context")
	cmd.Flags().BoolVar(&options.Expose, "expose", false, "If true will create a load balancer service to expose the vcluster endpoint")
	cmd.Flags().BoolVar(&options.Connect, "connect", true, "If true will run vcluster connect directly after the vcluster was created")
	cmd.Flags().BoolVar(&options.Upgrade, "upgrade", false, "If true will try to upgrade the vcluster instead of failing if it already exists")
	cmd.Flags().StringVar(&options.Distro, "distro", "k8s", fmt.Sprintf("Kubernetes distro to use for the virtual cluster. Allowed distros: %s", strings.Join(cli.AllowedDistros, ", ")))

	_ = cmd.Flags().MarkHidden("distro")
	_ = cmd.Flags().MarkDeprecated("distro", fmt.Sprintf("please specify the distro by setting %q accordingly via values.yaml file.", "controlPlane.distro"))
}

func AddHelmFlags(cmd *cobra.Command, options *cli.CreateOptions) {
	cmd.Flags().BoolVar(&options.CreateNamespace, "create-namespace", true, "If true the namespace will be created if it does not exist")
	cmd.Flags().StringVar(&options.LocalChartDir, "local-chart-dir", "", "The virtual cluster local chart dir to use")
	cmd.Flags().BoolVar(&options.ExposeLocal, "expose-local", true, "If true and a local Kubernetes distro is detected, will deploy vcluster with a NodePort service. Will be set to false and the passed value will be ignored if --expose is set to true.")
	cmd.Flags().BoolVar(&options.BackgroundProxy, "background-proxy", true, "Try to use a background-proxy to access the vCluster. Only works if docker is installed and reachable")
	cmd.Flags().BoolVar(&options.Add, "add", true, "Adds the virtual cluster automatically to the current vCluster platform when using helm driver")

	_ = cmd.Flags().MarkHidden("local-chart-dir")
	_ = cmd.Flags().MarkHidden("expose-local")
}

func AddPlatformFlags(cmd *cobra.Command, options *cli.CreateOptions, prefixes ...string) {
	prefix := strings.Join(prefixes, "")

	cmd.Flags().StringVar(&options.Project, "project", "", fmt.Sprintf("%sThe vCluster platform project to use", prefix))
	cmd.Flags().StringSliceVarP(&options.Labels, "labels", "l", []string{}, fmt.Sprintf("%sComma separated labels to apply to the virtualclusterinstance", prefix))
	cmd.Flags().StringSliceVar(&options.Annotations, "annotations", []string{}, fmt.Sprintf("%sComma separated annotations to apply to the virtualclusterinstance", prefix))
	cmd.Flags().StringVar(&options.Cluster, "cluster", "", fmt.Sprintf("%sThe vCluster platform connected cluster to use", prefix))
	cmd.Flags().StringVar(&options.Template, "template", "", fmt.Sprintf("%sThe vCluster platform template to use", prefix))
	cmd.Flags().StringVar(&options.TemplateVersion, "template-version", "", fmt.Sprintf("%sThe vCluster platform template version to use", prefix))
	cmd.Flags().StringArrayVar(&options.Links, "link", []string{}, fmt.Sprintf("%sA link to add to the vCluster. E.g. --link 'prod=http://exampleprod.com'", prefix))
	cmd.Flags().StringVar(&options.Params, "params", "", fmt.Sprintf("%sIf a template is used, this can be used to use a file for the parameters. E.g. --params path/to/my/file.yaml", prefix))
	cmd.Flags().StringVar(&options.Params, "parameters", "", fmt.Sprintf("%sIf a template is used, this can be used to use a file for the parameters. E.g. --parameters path/to/my/file.yaml", prefix))
	cmd.Flags().StringArrayVar(&options.SetParams, "set-param", []string{}, fmt.Sprintf("%sIf a template is used, this can be used to set a specific parameter. E.g. --set-param 'my-param=my-value'", prefix))
	cmd.Flags().StringArrayVar(&options.SetParams, "set-parameter", []string{}, fmt.Sprintf("%sIf a template is used, this can be used to set a specific parameter. E.g. --set-parameter 'my-param=my-value'", prefix))
	cmd.Flags().StringVar(&options.Description, "description", "", fmt.Sprintf("%sThe description to show in the platform UI for this virtual cluster", prefix))
	cmd.Flags().StringVar(&options.DisplayName, "display-name", "", fmt.Sprintf("%sThe display name to show in the platform UI for this virtual cluster", prefix))
	cmd.Flags().StringVar(&options.Team, "team", "", fmt.Sprintf("%sThe team to create the space for", prefix))
	cmd.Flags().StringVar(&options.User, "user", "", fmt.Sprintf("%sThe user to create the space for", prefix))
	cmd.Flags().BoolVar(&options.UseExisting, "use", false, fmt.Sprintf("%sIf the platform should use the virtual cluster if its already there", prefix))
	cmd.Flags().BoolVar(&options.Recreate, "recreate", false, fmt.Sprintf("%sIf enabled and there already exists a virtual cluster with this name, it will be deleted first", prefix))
	cmd.Flags().BoolVar(&options.SkipWait, "skip-wait", false, fmt.Sprintf("%sIf true, will not wait until the virtual cluster is running", prefix))
}
