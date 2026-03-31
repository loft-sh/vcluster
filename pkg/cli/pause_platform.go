package cli

import (
	"context"
	"fmt"
	"strconv"
	"time"

	clusterv1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/cluster/v1"
	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	"github.com/loft-sh/log"
	cliconfig "github.com/loft-sh/vcluster/pkg/cli/config"
	"github.com/loft-sh/vcluster/pkg/cli/find"
	"github.com/loft-sh/vcluster/pkg/cli/sleepmode"
	"github.com/loft-sh/vcluster/pkg/platform"
	"github.com/loft-sh/vcluster/pkg/platform/clihelper"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

func PausePlatform(ctx context.Context, options *PauseOptions, cfg *cliconfig.CLI, vClusterName string, log log.Logger) error {
	platformClient, err := platform.InitClientFromConfig(ctx, cfg)
	if err != nil {
		return err
	}

	vCluster, err := find.GetPlatformVCluster(ctx, platformClient, vClusterName, options.Project, log)
	if err != nil {
		return err
	}

	projectName := vCluster.Project.Name
	log.Infof("Putting virtual cluster %s in project %s to sleep", vCluster.VirtualCluster.Name, vCluster.Project.Name)
	virtualClusterInstance := vCluster.VirtualCluster

	// Check if the vCluster config enables workload sleep mode natively (no agent).
	used, err := tryWorkloadSleepPlatform(ctx, platformClient, projectName, options.ForceDuration, log, vClusterName, virtualClusterInstance)
	if used {
		return err
	}

	if vCluster.IsInstanceSleeping() {
		log.Infof("vcluster %s/%s is already paused", vCluster.VirtualCluster.Namespace, vClusterName)
		return nil
	}

	managementClient, err := platformClient.Management()
	if err != nil {
		return err
	}

	if virtualClusterInstance.Annotations == nil {
		virtualClusterInstance.Annotations = map[string]string{}
	}
	virtualClusterInstance.Annotations[clusterv1.SleepModeForceAnnotation] = "true"
	if options.ForceDuration >= 0 {
		virtualClusterInstance.Annotations[clusterv1.SleepModeForceDurationAnnotation] = strconv.FormatInt(options.ForceDuration, 10)
	}

	_, err = managementClient.Loft().ManagementV1().VirtualClusterInstances(vCluster.VirtualCluster.Namespace).Update(ctx, virtualClusterInstance, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	// wait for sleeping
	log.Info("Wait until virtual cluster is sleeping...")
	err = wait.PollUntilContextTimeout(ctx, time.Second, clihelper.Timeout(), false, func(ctx context.Context) (done bool, err error) {
		virtualClusterInstance, err := managementClient.Loft().ManagementV1().VirtualClusterInstances(vCluster.VirtualCluster.Namespace).Get(ctx, vCluster.VirtualCluster.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		return virtualClusterInstance.Status.Phase == storagev1.InstanceSleeping, nil
	})
	if err != nil {
		return fmt.Errorf("error waiting for vcluster to start sleeping: %w", err)
	}

	log.Donef("Successfully put vcluster %s to sleep", vCluster.VirtualCluster.Name)
	return nil
}

// tryWorkloadSleepPlatform will determine if workload sleep is applicable and apply it if so, returning true if applied
func tryWorkloadSleepPlatform(ctx context.Context, platformClient platform.Client, projectName string, forceDuration int64, log log.Logger, vClusterName string, virtualClusterInstance *managementv1.VirtualClusterInstance) (applied bool, retErr error) {
	sleepMgr, used, err := sleepmode.NewManager(ctx,
		sleepmode.WithPlatformClient(platformClient),
		sleepmode.WithProjectName(projectName),
		sleepmode.WithVirtualClusterInstance(virtualClusterInstance),
		sleepmode.WithVClusterName(vClusterName),
		sleepmode.WithNamespace(virtualClusterInstance.Spec.ClusterRef.Namespace),
		sleepmode.WithLogger(log))

	if err != nil {
		return false, err
	}

	if !used {
		return false, nil
	}

	if virtualClusterInstance.Spec.Standalone {
		return sleepMgr.SleepStandalone(ctx, forceDuration)
	}

	defer func() {
		if retErr != nil {
			log.Error(retErr, "Please try again.  If the problem persists, please contact support.")
		}
	}()

	sleepingSince := strconv.FormatInt(time.Now().Unix(), 10)
	log.Infof("vCluster %s/%s is configured for workload sleep mode, sleeping workloads only (control plane stays running)", virtualClusterInstance.Spec.ClusterRef.Namespace, vClusterName)
	return true, sleepMgr.Sleep(ctx, sleepingSince, sleepmode.SleepDuration(forceDuration))
}
