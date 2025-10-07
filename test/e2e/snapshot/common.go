package snapshot

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/snapshot"
	"github.com/loft-sh/vcluster/test/framework"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

func restoreVCluster(f *framework.Framework, snapshotPath string, controllerBasedSnapshot, restoreVolumes bool) {
	By("Restore vCluster")
	restoreArgs := []string{
		"restore",
		f.VClusterName,
		snapshotPath,
		"-n", f.VClusterNamespace,
	}
	if !controllerBasedSnapshot {
		restoreArgs = append(
			restoreArgs,
			"--pod-mount", "pvc:snapshot-pvc:/snapshot-pvc")
	}
	if restoreVolumes {
		restoreArgs = append(
			restoreArgs,
			"--restore-volumes")
	}
	cmd := exec.Command(
		"vcluster",
		restoreArgs...,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	framework.ExpectNoError(err)

	waitUntilVClusterIsRunning(f)

	if restoreVolumes {
		waitForRestoreRequestToFinish(f)
	}
}

func waitUntilVClusterIsRunning(f *framework.Framework) {
	// wait until vCluster is running
	err := wait.PollUntilContextTimeout(f.Context, time.Second, time.Minute*2, false, func(ctx context.Context) (done bool, err error) {
		newPods, _ := f.HostClient.CoreV1().Pods(f.VClusterNamespace).List(ctx, metav1.ListOptions{
			LabelSelector: "app=vcluster",
		})
		p := len(newPods.Items)
		if p > 0 {
			// rp, running pod counter
			rp := 0
			for _, pod := range newPods.Items {
				if pod.Status.Phase == corev1.PodRunning {
					rp = rp + 1
				}
			}
			if rp == p {
				return true, nil
			}
		}
		return false, nil
	})
	framework.ExpectNoError(err)

	// wait until all vCluster replicas are running
	Eventually(func() error {
		pods, err := f.HostClient.CoreV1().Pods(f.VClusterNamespace).List(f.Context, metav1.ListOptions{
			LabelSelector: "app=vcluster,release=" + f.VClusterName,
		})
		framework.ExpectNoError(err)

		for _, pod := range pods.Items {
			if len(pod.Status.ContainerStatuses) == 0 {
				return fmt.Errorf("pod %s has no container status", pod.Name)
			}

			for _, container := range pod.Status.ContainerStatuses {
				if container.State.Running == nil || !container.Ready {
					return fmt.Errorf("pod %s container %s is not running", pod.Name, container.Name)
				}
			}
		}
		return nil
	}).WithPolling(time.Second).
		WithTimeout(framework.PollTimeout).
		Should(Succeed())

	// refresh the connection
	err = f.RefreshVirtualClient()
	framework.ExpectNoError(err)
}

func waitForSnapshotRequestToFinish(f *framework.Framework) {
	waitForRequestToFinish(f, constants.SnapshotRequestLabel, snapshot.UnmarshalSnapshotRequest)
}

func waitForRestoreRequestToFinish(f *framework.Framework) {
	waitForRequestToFinish(f, constants.RestoreRequestLabel, snapshot.UnmarshalRestoreRequest)
}

type unmarshalRequestFunc[T snapshot.LongRunningRequest] func(request *corev1.ConfigMap) (T, error)

func waitForRequestToFinish[T snapshot.LongRunningRequest](f *framework.Framework, requestLabel string, unmarshal unmarshalRequestFunc[T]) {
	Eventually(func() error {
		listOptions := metav1.ListOptions{
			LabelSelector: requestLabel,
		}
		requestConfigMaps, err := f.HostClient.CoreV1().ConfigMaps(f.VClusterNamespace).List(f.Context, listOptions)
		framework.ExpectNoError(err)
		Expect(requestConfigMaps.Items).To(HaveLen(1))

		// extract snapshot/restore request
		requestConfigMap := requestConfigMaps.Items[0]
		request, err := unmarshal(&requestConfigMap)
		framework.ExpectNoError(err)

		// check if the snapshot/restore request has been completed
		if request.GetPhase() != snapshot.RequestPhaseCompleted {
			return fmt.Errorf("request is not completed, current phase is %s", request.GetPhase())
		}
		return nil
	}).
		WithPolling(framework.PollInterval).
		WithTimeout(framework.PollTimeout).
		Should(Succeed())
}

const RequestPhaseCompleted snapshot.RequestPhase = "Completed"
