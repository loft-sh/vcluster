package cli

import (
	"context"
	"fmt"

	"github.com/loft-sh/loftctl/v4/pkg/vcluster"
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
	} else if vCluster.VirtualCluster != nil && vCluster.VirtualCluster.Spec.NetworkPeer {
		return fmt.Errorf("cannot resume a virtual cluster that was created via helm, please run 'vcluster use manager helm' or use the '--manager helm' flag")
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
