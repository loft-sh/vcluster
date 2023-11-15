package use

import (
	"context"
	"fmt"
	"io"
	"os"

	managementv1 "github.com/loft-sh/api/v3/pkg/apis/management/v1"
	"github.com/loft-sh/api/v3/pkg/product"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/flags"
	"github.com/loft-sh/loftctl/v3/pkg/client"
	"github.com/loft-sh/loftctl/v3/pkg/client/helper"
	"github.com/loft-sh/loftctl/v3/pkg/client/naming"
	pdefaults "github.com/loft-sh/loftctl/v3/pkg/defaults"
	"github.com/loft-sh/loftctl/v3/pkg/kubeconfig"
	"github.com/loft-sh/loftctl/v3/pkg/upgrade"
	"github.com/loft-sh/loftctl/v3/pkg/util"
	"github.com/loft-sh/loftctl/v3/pkg/vcluster"
	"github.com/loft-sh/log"
	"github.com/mgutz/ansi"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

// VirtualClusterCmd holds the cmd flags
type VirtualClusterCmd struct {
	*flags.GlobalFlags

	Space                        string
	Cluster                      string
	Project                      string
	SkipWait                     bool
	Print                        bool
	PrintToken                   bool
	DisableDirectClusterEndpoint bool

	Out io.Writer
	Log log.Logger
}

// NewVirtualClusterCmd creates a new command
func NewVirtualClusterCmd(globalFlags *flags.GlobalFlags, defaults *pdefaults.Defaults) *cobra.Command {
	cmd := &VirtualClusterCmd{
		GlobalFlags: globalFlags,
		Out:         os.Stdout,
		Log:         log.GetInstance(),
	}

	description := product.ReplaceWithHeader("use vcluster", `
Creates a new kube context for the given virtual cluster.

Example:
loft use vcluster
loft use vcluster myvcluster
loft use vcluster myvcluster --cluster mycluster
loft use vcluster myvcluster --cluster mycluster --space myspace
########################################################
	`)
	if upgrade.IsPlugin == "true" {
		description = `
########################################################
################ devspace use vcluster #################
########################################################
Creates a new kube context for the given virtual cluster.

Example:
devspace use vcluster
devspace use vcluster myvcluster
devspace use vcluster myvcluster --cluster mycluster
devspace use vcluster myvcluster --cluster mycluster --space myspace
########################################################
	`
	}
	useLine, validator := util.NamedPositionalArgsValidator(false, false, "VCLUSTER_NAME")
	c := &cobra.Command{
		Use:   "vcluster" + useLine,
		Short: "Creates a kube context for the given virtual cluster",
		Long:  description,
		Args:  validator,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			// Check for newer version
			if !cmd.Print && !cmd.PrintToken {
				upgrade.PrintNewerVersionWarning()
			}

			return cmd.Run(cobraCmd.Context(), args)
		},
	}

	p, _ := defaults.Get(pdefaults.KeyProject, "")
	c.Flags().StringVar(&cmd.Space, "space", "", "The space to use")
	c.Flags().StringVar(&cmd.Cluster, "cluster", "", "The cluster to use")
	c.Flags().StringVarP(&cmd.Project, "project", "p", p, "The project to use")
	c.Flags().BoolVar(&cmd.SkipWait, "skip-wait", false, "If true, will not wait until the virtual cluster is running")
	c.Flags().BoolVar(&cmd.Print, "print", false, "When enabled prints the context to stdout")
	c.Flags().BoolVar(&cmd.DisableDirectClusterEndpoint, "disable-direct-cluster-endpoint", false, "When enabled does not use an available direct cluster endpoint to connect to the vcluster")
	return c
}

// Run executes the command
func (cmd *VirtualClusterCmd) Run(ctx context.Context, args []string) error {
	baseClient, err := client.NewClientFromPath(cmd.Config)
	if err != nil {
		return err
	}

	err = client.VerifyVersion(baseClient)
	if err != nil {
		return err
	}

	virtualClusterName := ""
	if len(args) > 0 {
		virtualClusterName = args[0]
	}

	cmd.Cluster, cmd.Project, cmd.Space, virtualClusterName, err = helper.SelectVirtualClusterInstanceOrVirtualCluster(baseClient, virtualClusterName, cmd.Space, cmd.Project, cmd.Cluster, cmd.Log)
	if err != nil {
		return err
	}

	if cmd.Project == "" {
		return cmd.legacyUseVirtualCluster(ctx, baseClient, virtualClusterName)
	}

	return cmd.useVirtualCluster(ctx, baseClient, virtualClusterName)
}

func (cmd *VirtualClusterCmd) useVirtualCluster(ctx context.Context, baseClient client.Client, virtualClusterName string) error {
	managementClient, err := baseClient.Management()
	if err != nil {
		return err
	}

	virtualClusterInstance, err := vcluster.WaitForVirtualClusterInstance(ctx, managementClient, naming.ProjectNamespace(cmd.Project), virtualClusterName, !cmd.SkipWait, cmd.Log)
	if err != nil {
		return err
	}

	// create kube context options
	contextOptions, err := CreateVirtualClusterInstanceOptions(ctx, baseClient, cmd.Config, cmd.Project, virtualClusterInstance, cmd.DisableDirectClusterEndpoint, true, cmd.Log)
	if err != nil {
		return err
	}

	// check if we should print or update the config
	if cmd.Print {
		err = kubeconfig.PrintKubeConfigTo(contextOptions, os.Stdout)
		if err != nil {
			return err
		}
	} else {
		// update kube config
		err = kubeconfig.UpdateKubeConfig(contextOptions)
		if err != nil {
			return err
		}

		cmd.Log.Donef("Successfully updated kube context to use virtual cluster %s in project %s", ansi.Color(virtualClusterName, "white+b"), ansi.Color(cmd.Project, "white+b"))
	}

	return nil
}

func CreateVirtualClusterInstanceOptions(ctx context.Context, baseClient client.Client, config string, projectName string, virtualClusterInstance *managementv1.VirtualClusterInstance, disableClusterGateway, setActive bool, log log.Logger) (kubeconfig.ContextOptions, error) {
	cluster, err := findProjectCluster(ctx, baseClient, projectName, virtualClusterInstance.Spec.ClusterRef.Cluster)
	if err != nil {
		return kubeconfig.ContextOptions{}, errors.Wrap(err, "find space instance cluster")
	}

	contextOptions := kubeconfig.ContextOptions{
		Name:       kubeconfig.VirtualClusterInstanceContextName(projectName, virtualClusterInstance.Name),
		ConfigPath: config,
		SetActive:  setActive,
	}
	if virtualClusterInstance.Status.VirtualCluster != nil && virtualClusterInstance.Status.VirtualCluster.AccessPoint.Ingress.Enabled {
		kubeConfig, err := getVirtualClusterInstanceAccessConfig(ctx, baseClient, virtualClusterInstance)
		if err != nil {
			return kubeconfig.ContextOptions{}, errors.Wrap(err, "retrieve kube config")
		}

		// get server
		for _, val := range kubeConfig.Clusters {
			contextOptions.Server = val.Server
		}

		contextOptions.InsecureSkipTLSVerify = true
		contextOptions.VirtualClusterAccessPointEnabled = true
	} else {
		if !disableClusterGateway && cluster.Annotations != nil && cluster.Annotations[LoftDirectClusterEndpoint] != "" {
			contextOptions = ApplyDirectClusterEndpointOptions(contextOptions, cluster, "/kubernetes/project/"+projectName+"/virtualcluster/"+virtualClusterInstance.Name, log)
			_, err := baseClient.DirectClusterEndpointToken(true)
			if err != nil {
				return kubeconfig.ContextOptions{}, fmt.Errorf("retrieving direct cluster endpoint token: %w. Use --disable-direct-cluster-endpoint to create a context without using direct cluster endpoints", err)
			}
		} else {
			contextOptions.Server = baseClient.Config().Host + "/kubernetes/project/" + projectName + "/virtualcluster/" + virtualClusterInstance.Name
			contextOptions.InsecureSkipTLSVerify = baseClient.Config().Insecure
		}

		data, err := retrieveCaData(cluster)
		if err != nil {
			return kubeconfig.ContextOptions{}, err
		}
		contextOptions.CaData = data
	}
	return contextOptions, nil
}

func (cmd *VirtualClusterCmd) legacyUseVirtualCluster(ctx context.Context, baseClient client.Client, virtualClusterName string) error {
	managementClient, err := baseClient.Management()
	if err != nil {
		return err
	}

	// check if the cluster exists
	cluster, err := managementClient.Loft().ManagementV1().Clusters().Get(ctx, cmd.Cluster, metav1.GetOptions{})
	if err != nil {
		if kerrors.IsForbidden(err) {
			return fmt.Errorf("cluster '%s' does not exist, or you don't have permission to use it", cmd.Cluster)
		}

		return err
	}

	// get token for virtual cluster
	if !cmd.Print && !cmd.PrintToken {
		cmd.Log.Info("Waiting for virtual cluster to become ready...")
	}
	err = vcluster.WaitForVCluster(ctx, baseClient, cmd.Cluster, cmd.Space, virtualClusterName, cmd.Log)
	if err != nil {
		return err
	}

	// create kube context options
	contextOptions, err := CreateVClusterContextOptions(baseClient, cmd.Config, cluster, cmd.Space, virtualClusterName, cmd.DisableDirectClusterEndpoint, true, cmd.Log)
	if err != nil {
		return err
	}

	// check if we should print or update the config
	if cmd.Print {
		err = kubeconfig.PrintKubeConfigTo(contextOptions, cmd.Out)
		if err != nil {
			return err
		}
	} else {
		// update kube config
		err = kubeconfig.UpdateKubeConfig(contextOptions)
		if err != nil {
			return err
		}

		cmd.Log.Donef("Successfully updated kube context to use space %s in cluster %s", ansi.Color(cmd.Space, "white+b"), ansi.Color(cmd.Cluster, "white+b"))
	}

	return nil
}

func CreateVClusterContextOptions(baseClient client.Client, config string, cluster *managementv1.Cluster, spaceName, virtualClusterName string, disableClusterGateway, setActive bool, log log.Logger) (kubeconfig.ContextOptions, error) {
	contextOptions := kubeconfig.ContextOptions{
		Name:       kubeconfig.VirtualClusterContextName(cluster.Name, spaceName, virtualClusterName),
		ConfigPath: config,
		SetActive:  setActive,
	}
	if !disableClusterGateway && cluster.Annotations != nil && cluster.Annotations[LoftDirectClusterEndpoint] != "" {
		contextOptions = ApplyDirectClusterEndpointOptions(contextOptions, cluster, "/kubernetes/virtualcluster/"+spaceName+"/"+virtualClusterName, log)
		_, err := baseClient.DirectClusterEndpointToken(true)
		if err != nil {
			return kubeconfig.ContextOptions{}, fmt.Errorf("retrieving direct cluster endpoint token: %w. Use --disable-direct-cluster-endpoint to create a context without using direct cluster endpoints", err)
		}
	} else {
		contextOptions.Server = baseClient.Config().Host + "/kubernetes/virtualcluster/" + cluster.Name + "/" + spaceName + "/" + virtualClusterName
		contextOptions.InsecureSkipTLSVerify = baseClient.Config().Insecure
	}

	data, err := retrieveCaData(cluster)
	if err != nil {
		return kubeconfig.ContextOptions{}, err
	}
	contextOptions.CaData = data
	return contextOptions, nil
}

func getVirtualClusterInstanceAccessConfig(ctx context.Context, baseClient client.Client, virtualClusterInstance *managementv1.VirtualClusterInstance) (*api.Config, error) {
	managementClient, err := baseClient.Management()
	if err != nil {
		return nil, err
	}

	kubeConfig, err := managementClient.Loft().ManagementV1().VirtualClusterInstances(virtualClusterInstance.Namespace).GetKubeConfig(
		ctx,
		virtualClusterInstance.Name,
		&managementv1.VirtualClusterInstanceKubeConfig{
			Spec: managementv1.VirtualClusterInstanceKubeConfigSpec{},
		},
		metav1.CreateOptions{},
	)
	if err != nil {
		return nil, err
	}

	// parse kube config string
	clientCfg, err := clientcmd.NewClientConfigFromBytes([]byte(kubeConfig.Status.KubeConfig))
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	apiCfg, err := clientCfg.RawConfig()
	if err != nil {
		return nil, err
	}

	return &apiCfg, nil
}
