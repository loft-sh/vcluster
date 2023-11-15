package use

import (
	"context"
	"fmt"
	"os"

	managementv1 "github.com/loft-sh/api/v3/pkg/apis/management/v1"
	"github.com/loft-sh/api/v3/pkg/product"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/flags"
	"github.com/loft-sh/loftctl/v3/pkg/client"
	"github.com/loft-sh/loftctl/v3/pkg/client/helper"
	"github.com/loft-sh/loftctl/v3/pkg/client/naming"
	pdefaults "github.com/loft-sh/loftctl/v3/pkg/defaults"
	"github.com/loft-sh/loftctl/v3/pkg/kubeconfig"
	"github.com/loft-sh/loftctl/v3/pkg/space"
	"github.com/loft-sh/loftctl/v3/pkg/upgrade"
	"github.com/loft-sh/loftctl/v3/pkg/util"
	"github.com/loft-sh/log"
	"github.com/mgutz/ansi"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SpaceCmd holds the cmd flags
type SpaceCmd struct {
	*flags.GlobalFlags

	Cluster                      string
	Project                      string
	Print                        bool
	SkipWait                     bool
	DisableDirectClusterEndpoint bool

	log log.Logger
}

// NewSpaceCmd creates a new command
func NewSpaceCmd(globalFlags *flags.GlobalFlags, defaults *pdefaults.Defaults) *cobra.Command {
	cmd := &SpaceCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}

	description := product.ReplaceWithHeader("use space", `
Creates a new kube context for the given space.

Example:
loft use space
loft use space myspace
loft use space myspace --project myproject
########################################################
	`)
	if upgrade.IsPlugin == "true" {
		description = `
########################################################
################# devspace use space ###################
########################################################
Creates a new kube context for the given space.

Example:
devspace use space
devspace use space myspace
devspace use space myspace --project myproject
########################################################
	`
	}
	useLine, validator := util.NamedPositionalArgsValidator(false, false, "SPACE_NAME")
	c := &cobra.Command{
		Use:   "space" + useLine,
		Short: "Creates a kube context for the given space",
		Long:  description,
		Args:  validator,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			// Check for newer version
			if !cmd.Print {
				upgrade.PrintNewerVersionWarning()
			}

			return cmd.Run(cobraCmd.Context(), args)
		},
	}

	p, _ := defaults.Get(pdefaults.KeyProject, "")
	c.Flags().StringVar(&cmd.Cluster, "cluster", "", "The cluster to use")
	c.Flags().StringVarP(&cmd.Project, "project", "p", p, "The project to use")
	c.Flags().BoolVar(&cmd.Print, "print", false, "When enabled prints the context to stdout")
	c.Flags().BoolVar(&cmd.SkipWait, "skip-wait", false, "If true, will not wait until the space is running")
	c.Flags().BoolVar(&cmd.DisableDirectClusterEndpoint, "disable-direct-cluster-endpoint", false, "When enabled does not use an available direct cluster endpoint to connect to the cluster")
	return c
}

// Run executes the command
func (cmd *SpaceCmd) Run(ctx context.Context, args []string) error {
	baseClient, err := client.NewClientFromPath(cmd.Config)
	if err != nil {
		return err
	}

	err = client.VerifyVersion(baseClient)
	if err != nil {
		return err
	}

	spaceName := ""
	if len(args) > 0 {
		spaceName = args[0]
	}

	cmd.Cluster, cmd.Project, spaceName, err = helper.SelectSpaceInstanceOrSpace(baseClient, spaceName, cmd.Project, cmd.Cluster, cmd.log)
	if err != nil {
		return err
	}

	if cmd.Project == "" {
		return cmd.legacyUseSpace(ctx, baseClient, spaceName)
	}

	return cmd.useSpace(ctx, baseClient, spaceName)
}

func (cmd *SpaceCmd) useSpace(ctx context.Context, baseClient client.Client, spaceName string) error {
	managementClient, err := baseClient.Management()
	if err != nil {
		return err
	}

	// wait until space is ready
	spaceInstance, err := space.WaitForSpaceInstance(ctx, managementClient, naming.ProjectNamespace(cmd.Project), spaceName, !cmd.SkipWait, cmd.log)
	if err != nil {
		return err
	}

	// create kube context options
	contextOptions, err := CreateSpaceInstanceOptions(ctx, baseClient, cmd.Config, cmd.Project, spaceInstance, cmd.DisableDirectClusterEndpoint, true, cmd.log)
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

		cmd.log.Donef("Successfully updated kube context to use space %s in project %s", ansi.Color(spaceName, "white+b"), ansi.Color(cmd.Project, "white+b"))
	}

	return nil
}

func (cmd *SpaceCmd) legacyUseSpace(ctx context.Context, baseClient client.Client, spaceName string) error {
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

	// create kube context options
	contextOptions, err := CreateClusterContextOptions(baseClient, cmd.Config, cluster, spaceName, cmd.DisableDirectClusterEndpoint, true, cmd.log)
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

		cmd.log.Donef("Successfully updated kube context to use space %s in cluster %s", ansi.Color(spaceName, "white+b"), ansi.Color(cmd.Cluster, "white+b"))
	}

	return nil
}

func CreateSpaceInstanceOptions(ctx context.Context, baseClient client.Client, config string, projectName string, spaceInstance *managementv1.SpaceInstance, disableClusterGateway, setActive bool, log log.Logger) (kubeconfig.ContextOptions, error) {
	cluster, err := findProjectCluster(ctx, baseClient, projectName, spaceInstance.Spec.ClusterRef.Cluster)
	if err != nil {
		return kubeconfig.ContextOptions{}, errors.Wrap(err, "find space instance cluster")
	}

	contextOptions := kubeconfig.ContextOptions{
		Name:             kubeconfig.SpaceInstanceContextName(projectName, spaceInstance.Name),
		ConfigPath:       config,
		CurrentNamespace: spaceInstance.Spec.ClusterRef.Namespace,
		SetActive:        setActive,
	}
	if !disableClusterGateway && cluster.Annotations != nil && cluster.Annotations[LoftDirectClusterEndpoint] != "" {
		contextOptions = ApplyDirectClusterEndpointOptions(contextOptions, cluster, "/kubernetes/project/"+projectName+"/space/"+spaceInstance.Name, log)
		_, err := baseClient.DirectClusterEndpointToken(true)
		if err != nil {
			return kubeconfig.ContextOptions{}, fmt.Errorf("retrieving direct cluster endpoint token: %w. Use --disable-direct-cluster-endpoint to create a context without using direct cluster endpoints", err)
		}
	} else {
		contextOptions.Server = baseClient.Config().Host + "/kubernetes/project/" + projectName + "/space/" + spaceInstance.Name
		contextOptions.InsecureSkipTLSVerify = baseClient.Config().Insecure
	}

	data, err := retrieveCaData(cluster)
	if err != nil {
		return kubeconfig.ContextOptions{}, err
	}
	contextOptions.CaData = data
	return contextOptions, nil
}
