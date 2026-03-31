package cli

import (
	"context"
	"fmt"

	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/config"
	"github.com/loft-sh/vcluster/pkg/cli/find"
	"github.com/loft-sh/vcluster/pkg/cli/sleepmode"
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
	}

	projectName := vCluster.Project.Name
	virtualClusterInstance := vCluster.VirtualCluster
	// Check if the vCluster is in workload sleep mode and wake it.
	used, err := tryWorkloadWakePlatform(ctx, platformClient, projectName, log, vClusterName, virtualClusterInstance)
	if used {
		return err
	}

	if !vCluster.IsInstanceSleeping() {
		return fmt.Errorf(
			"couldn't find a paused vcluster %s in namespace %s. Make sure the vcluster exists and was paused previously",
			vCluster.VirtualCluster.Spec.ClusterRef.VirtualCluster,
			vCluster.VirtualCluster.Spec.ClusterRef.Namespace,
		)
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

// tryWorkloadWakePlatform checks if whether workload sleep mode is configured and clears annotations for the instance to wake itself
// sleep mode and clears the sleep annotations to wake it. Mirrors tryWorkloadSleepPlatform.
func tryWorkloadWakePlatform(ctx context.Context, platformClient platform.Client, projectName string, log log.Logger, vClusterName string, virtualClusterInstance *managementv1.VirtualClusterInstance) (applied bool, retErr error) {
	vcNamespace := virtualClusterInstance.Spec.ClusterRef.Namespace
	sleepMgr, used, err := sleepmode.NewManager(ctx,
		sleepmode.WithPlatformClient(platformClient),
		sleepmode.WithProjectName(projectName),
		sleepmode.WithVClusterName(vClusterName),
		sleepmode.WithNamespace(vcNamespace),
		sleepmode.WithVirtualClusterInstance(virtualClusterInstance),
		sleepmode.WithLogger(log))
	if err != nil {
		return false, err
	}

	if !used {
		return false, nil
	}

	// Standalone vClusters wake via the virtual cluster proxy.
	if virtualClusterInstance.Spec.Standalone {
		return sleepMgr.WakeStandalone(ctx)
	}

	defer func() {
		if retErr != nil {
			log.Error(retErr, "Please try again.  If the problem persists, please contact support.")
		}
	}()

	if !sleepMgr.IsSleeping() {
		log.Infof("vCluster %s/%s workloads are already running", vcNamespace, vClusterName)
		return true, nil
	}

	log.Infof("Waking vCluster %s/%s workloads", vcNamespace, vClusterName)
	return true, sleepMgr.Wake(ctx)
}
