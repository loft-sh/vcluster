package cli

import (
	"context"
	"fmt"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/config"
	"github.com/loft-sh/vcluster/pkg/cli/find"
	"github.com/loft-sh/vcluster/pkg/platform"
)

func ResumePlatform(ctx context.Context, options *ResumeOptions, config *config.CLI, vClusterName string, log log.Logger) error {
	platformClient, err := platform.InitClientFromConfig(ctx, config)
	if err != nil {
		return err
	}
	vCluster, err := find.GetPlatformVCluster(ctx, platformClient, vClusterName, options.Project, log)
	if err != nil {
		return err
	} else if vCluster.VirtualCluster != nil && vCluster.VirtualCluster.Spec.External {
		return fmt.Errorf("cannot resume a virtual cluster that was created via helm, please run 'vcluster use driver helm' or use the '--driver helm' flag")
	}

	managementClient, err := platformClient.Management()
	if err != nil {
		return err
	}

	log.Infof("Waking up virtual cluster %s in project %s", vCluster.VirtualCluster.Name, vCluster.Project.Name)
	_, err = platform.WaitForVirtualClusterInstance(ctx, managementClient, vCluster.VirtualCluster.Namespace, vCluster.VirtualCluster.Name, true, log)
	if err != nil {
		return err
	}

	log.Donef("Successfully woke up vCluster %s", vCluster.VirtualCluster.Name)
	return nil
}
