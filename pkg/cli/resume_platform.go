package cli

import (
	"context"

	"github.com/loft-sh/loftctl/v3/pkg/vcluster"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/find"
	"github.com/loft-sh/vcluster/pkg/platform"
)

func ResumePlatform(ctx context.Context, options *ResumeOptions, vClusterName string, log log.Logger) error {
	platformClient, err := platform.CreatePlatformClient()
	if err != nil {
		return err
	}

	vCluster, err := find.GetPlatformVCluster(ctx, platformClient, vClusterName, options.Project, log)
	if err != nil {
		return err
	}

	managementClient, err := platformClient.Management()
	if err != nil {
		return err
	}

	log.Infof("Waking up virtual cluster %s in project %s", vCluster.VirtualCluster.Name, vCluster.Project.Name)
	_, err = vcluster.WaitForVirtualClusterInstance(ctx, managementClient, vCluster.VirtualCluster.Namespace, vCluster.VirtualCluster.Name, true, log)
	if err != nil {
		return err
	}

	log.Donef("Successfully woke up vCluster %s", vCluster.VirtualCluster.Name)
	return nil
}
