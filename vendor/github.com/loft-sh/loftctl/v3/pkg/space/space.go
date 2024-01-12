package space

import (
	"context"
	"fmt"
	"strconv"
	"time"

	clusterv1 "github.com/loft-sh/agentapi/v3/pkg/apis/loft/cluster/v1"
	managementv1 "github.com/loft-sh/api/v3/pkg/apis/management/v1"
	storagev1 "github.com/loft-sh/api/v3/pkg/apis/storage/v1"
	"github.com/loft-sh/loftctl/v3/pkg/config"
	"github.com/loft-sh/loftctl/v3/pkg/kube"
	"github.com/loft-sh/loftctl/v3/pkg/util"
	"github.com/loft-sh/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var waitDuration = 20 * time.Second

func WaitForSpaceInstance(ctx context.Context, managementClient kube.Interface, namespace, name string, waitUntilReady bool, log log.Logger) (*managementv1.SpaceInstance, error) {
	now := time.Now()
	nextMessage := now.Add(waitDuration)
	spaceInstance, err := managementClient.Loft().ManagementV1().SpaceInstances(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	if spaceInstance.Status.Phase == storagev1.InstanceSleeping {
		log.Info("Wait until space wakes up")
		defer log.Donef("Successfully woken up space %s", name)
		err := wakeup(ctx, managementClient, spaceInstance)
		if err != nil {
			return nil, fmt.Errorf("error waking up space %s: %s", name, util.GetCause(err))
		}
	}

	if !waitUntilReady {
		return spaceInstance, nil
	}

	warnCounter := 0
	return spaceInstance, wait.PollUntilContextTimeout(ctx, time.Second, config.Timeout(), true, func(ctx context.Context) (bool, error) {
		spaceInstance, err = managementClient.Loft().ManagementV1().SpaceInstances(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		if spaceInstance.Status.Phase != storagev1.InstanceReady && spaceInstance.Status.Phase != storagev1.InstanceSleeping {
			if time.Now().After(nextMessage) {
				if warnCounter > 1 {
					log.Warnf("Cannot reach space because: %s (%s). Loft will continue waiting, but this operation may timeout", spaceInstance.Status.Message, spaceInstance.Status.Reason)
				} else {
					log.Info("Waiting for space to be available...")
				}
				nextMessage = time.Now().Add(waitDuration)
				warnCounter++
			}
			return false, nil
		}

		return true, nil
	})
}

func wakeup(ctx context.Context, managementClient kube.Interface, spaceInstance *managementv1.SpaceInstance) error {
	// Update instance to wake up
	oldSpaceInstance := spaceInstance.DeepCopy()
	if spaceInstance.Annotations == nil {
		spaceInstance.Annotations = map[string]string{}
	}
	delete(spaceInstance.Annotations, clusterv1.SleepModeForceAnnotation)
	delete(spaceInstance.Annotations, clusterv1.SleepModeForceDurationAnnotation)
	spaceInstance.Annotations[clusterv1.SleepModeLastActivityAnnotation] = strconv.FormatInt(time.Now().Unix(), 10)

	// Calculate patch
	patch := client.MergeFrom(oldSpaceInstance)
	patchData, err := patch.Data(spaceInstance)
	if err != nil {
		return err
	}

	// Patch the instance
	_, err = managementClient.Loft().ManagementV1().SpaceInstances(spaceInstance.Namespace).Patch(ctx, spaceInstance.Name, patch.Type(), patchData, metav1.PatchOptions{})
	if err != nil {
		return err
	}

	return nil
}
