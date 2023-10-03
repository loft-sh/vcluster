package vcluster

import (
	"context"
	"fmt"
	"strconv"
	"time"

	clusterv1 "github.com/loft-sh/agentapi/v3/pkg/apis/loft/cluster/v1"
	managementv1 "github.com/loft-sh/api/v3/pkg/apis/management/v1"
	storagev1 "github.com/loft-sh/api/v3/pkg/apis/storage/v1"
	"github.com/loft-sh/loftctl/v3/pkg/client"
	"github.com/loft-sh/loftctl/v3/pkg/config"
	"github.com/loft-sh/loftctl/v3/pkg/kube"
	"github.com/loft-sh/loftctl/v3/pkg/util"
	"github.com/loft-sh/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	client2 "sigs.k8s.io/controller-runtime/pkg/client"
)

var waitDuration = 20 * time.Second

func WaitForVCluster(ctx context.Context, baseClient client.Client, clusterName, spaceName, virtualClusterName string, log log.Logger) error {
	vClusterClient, err := baseClient.VirtualCluster(clusterName, spaceName, virtualClusterName)
	if err != nil {
		return err
	}

	now := time.Now()
	nextMessage := now.Add(waitDuration)

	warnCounter := 0

	return wait.PollUntilContextTimeout(ctx, time.Second, config.Timeout(), true, func(ctx context.Context) (bool, error) {
		_, err = vClusterClient.CoreV1().ServiceAccounts("default").Get(ctx, "default", metav1.GetOptions{})
		if err != nil && time.Now().After(nextMessage) {
			if warnCounter > 1 {
				log.Warnf("Cannot reach virtual cluster because: %v. Loft will continue waiting, but this operation may timeout", err)
			} else {
				log.Info("Waiting for virtual cluster to be available...")
			}

			nextMessage = time.Now().Add(waitDuration)
			warnCounter++
			return false, nil
		}

		return err == nil, nil
	})
}

func WaitForVirtualClusterInstance(ctx context.Context, managementClient kube.Interface, namespace, name string, waitUntilReady bool, log log.Logger) (*managementv1.VirtualClusterInstance, error) {
	now := time.Now()
	nextMessage := now.Add(waitDuration)
	virtualClusterInstance, err := managementClient.Loft().ManagementV1().VirtualClusterInstances(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	if virtualClusterInstance.Status.Phase == storagev1.InstanceSleeping {
		log.Info("Wait until vcluster wakes up")
		defer log.Donef("Successfully woken up vcluster %s", name)
		err := wakeup(ctx, managementClient, virtualClusterInstance)
		if err != nil {
			return nil, fmt.Errorf("error waking up vcluster %s: %s", name, util.GetCause(err))
		}
	}

	if !waitUntilReady {
		return virtualClusterInstance, nil
	}

	warnCounter := 0
	return virtualClusterInstance, wait.PollUntilContextTimeout(ctx, time.Second, config.Timeout(), true, func(ctx context.Context) (bool, error) {
		virtualClusterInstance, err = managementClient.Loft().ManagementV1().VirtualClusterInstances(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		if virtualClusterInstance.Status.Phase != storagev1.InstanceReady && virtualClusterInstance.Status.Phase != storagev1.InstanceSleeping {
			if time.Now().After(nextMessage) {
				if warnCounter > 1 {
					log.Warnf("Cannot reach virtual cluster because: %s (%s). Loft will continue waiting, but this operation may timeout", virtualClusterInstance.Status.Message, virtualClusterInstance.Status.Reason)
				} else {
					log.Info("Waiting for virtual cluster to be available...")
				}
				nextMessage = time.Now().Add(waitDuration)
				warnCounter++
			}
			return false, nil
		}

		return true, nil
	})
}

func wakeup(ctx context.Context, managementClient kube.Interface, virtualClusterInstance *managementv1.VirtualClusterInstance) error {
	// Update instance to wake up
	oldVirtualClusterInstance := virtualClusterInstance.DeepCopy()
	if virtualClusterInstance.Annotations == nil {
		virtualClusterInstance.Annotations = map[string]string{}
	}
	delete(virtualClusterInstance.Annotations, clusterv1.SleepModeForceAnnotation)
	delete(virtualClusterInstance.Annotations, clusterv1.SleepModeForceDurationAnnotation)
	virtualClusterInstance.Annotations[clusterv1.SleepModeLastActivityAnnotation] = strconv.FormatInt(time.Now().Unix(), 10)

	// Calculate patch
	patch := client2.MergeFrom(oldVirtualClusterInstance)
	patchData, err := patch.Data(virtualClusterInstance)
	if err != nil {
		return err
	}

	_, err = managementClient.Loft().ManagementV1().VirtualClusterInstances(virtualClusterInstance.Namespace).Patch(ctx, virtualClusterInstance.Name, patch.Type(), patchData, metav1.PatchOptions{})
	if err != nil {
		return err
	}

	return nil
}
