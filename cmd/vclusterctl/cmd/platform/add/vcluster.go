package add

import (
	"context"
	"errors"
	"fmt"

	"github.com/loft-sh/log"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/loft-sh/vcluster/pkg/cli"
	"github.com/loft-sh/vcluster/pkg/cli/find"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/platform"
	"github.com/loft-sh/vcluster/pkg/projectutil"
)

type VClusterCmd struct {
	*flags.GlobalFlags
	cli.AddVClusterOptions

	Log log.Logger
}

func NewVClusterCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &VClusterCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}

	description := `###############################################
############# vcluster platform add vcluster ##############
###############################################
Adds a vCluster to the vCluster platform.

Example:
vcluster platform add vcluster my-vcluster --namespace vcluster-my-vcluster --project my-project --import-name my-vcluster

Add all vCluster instances in the host cluster:
vcluster platform add vcluster --project my-project --all

###############################################
	`

	addCmd := &cobra.Command{
		Use:   "vcluster",
		Short: "Adds an existing vCluster to the vCluster platform",
		Long:  description,
		Args:  cobra.RangeArgs(0, 1),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context(), args)
		},
	}

	addCmd.Flags().StringVar(&cmd.Project, "project", "", "The project to import the vCluster into")
	addCmd.Flags().StringVar(&cmd.ImportName, "import-name", "", "The name of the vCluster under projects. If unspecified, will use the vcluster name")
	addCmd.Flags().BoolVar(&cmd.Restart, "restart", true, "Restart the vCluster control-plane after creating the platform secret")
	addCmd.Flags().StringVar(&cmd.AccessKey, "access-key", "", "The access key for the vCluster to connect to the platform. If empty, the CLI will generate one")
	addCmd.Flags().StringVar(&cmd.Host, "host", "", "The host where to reach the platform")
	addCmd.Flags().BoolVar(&cmd.Insecure, "insecure", false, "If the platform host is insecure")
	addCmd.Flags().BytesBase64Var(&cmd.CertificateAuthorityData, "ca-data", []byte{}, "additional, base64 encoded certificate authority data that will be passed to the platform secret")
	addCmd.Flags().BoolVar(&cmd.All, "all", false, "all will try to add Virtual Cluster found in all namespaces in the host cluster. If this flag is set, any provided vCluster name argument is ignored")

	return addCmd
}

// Run executes the functionality
func (cmd *VClusterCmd) Run(ctx context.Context, args []string) error {
	localVClusters, err := cmd.localVClustersFromOptions(ctx, cmd.Log, cmd.GlobalFlags, args, cmd.AddVClusterOptions)
	if err != nil {
		return err
	}

	if err := cmd.validateForPreExistence(ctx, localVClusters); err != nil {
		return err
	}

	return cli.AddVClusterHelm(ctx, cmd.Log, &cmd.AddVClusterOptions, cmd.GlobalFlags, localVClusters)
}

func (cmd *VClusterCmd) validateForPreExistence(ctx context.Context, localVClusters []find.VCluster) error {
	targetProject := cmd.Project
	if targetProject == "" {
		targetProject = "default"
	}

	byNameMap := make(map[string]bool, len(localVClusters))
	for _, v := range localVClusters {
		byNameMap[v.Name] = true
	}

	platformClient, err := platform.InitClientFromConfig(ctx, cmd.LoadedConfig(cmd.Log))
	if err != nil {
		return fmt.Errorf("new client from path: %w", err)
	}

	managementClient, err := platformClient.Management()
	if err != nil {
		return fmt.Errorf("create management client: %w", err)
	}

	vcInstanceList, err := managementClient.Loft().ManagementV1().VirtualClusterInstances("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, vClusterInstance := range vcInstanceList.Items {
		if !byNameMap[vClusterInstance.Name] {
			continue
		}
		projectFromVirtualCluster := projectutil.ProjectFromNamespace(vClusterInstance.Namespace)
		if projectFromVirtualCluster != targetProject {
			return fmt.Errorf("vClusterInstance %q already exists in project %q, moving vCluster from projects is not allowed", vClusterInstance.Name, projectFromVirtualCluster)
		}
	}

	return nil
}

func (cmd *VClusterCmd) localVClustersFromOptions(ctx context.Context, log log.Logger, globalFlags *flags.GlobalFlags, args []string, options cli.AddVClusterOptions) ([]find.VCluster, error) {
	var vClusters []find.VCluster

	if len(args) == 0 && !options.All {
		return nil, errors.New("empty vCluster name but no --all flag set, please either set vCluster name to add one cluster or set --all flag to add all of them")
	}
	if options.All {
		log.Info("looking for vCluster instances in all namespaces")
		vClustersInNamespace, err := find.ListVClusters(ctx, globalFlags.Context, "", "", log)
		if err != nil {
			return nil, err
		}
		if len(vClustersInNamespace) == 0 {
			log.Infof("no vCluster instances found in context %s", globalFlags.Context)
		} else {
			vClusters = append(vClusters, vClustersInNamespace...)
		}
	} else {
		// check if vCluster exists
		vClusterName := args[0]
		vCluster, err := find.GetVCluster(ctx, globalFlags.Context, vClusterName, globalFlags.Namespace, log)
		if err != nil {
			return nil, err
		}
		vClusters = append(vClusters, *vCluster)
	}

	return vClusters, nil
}
