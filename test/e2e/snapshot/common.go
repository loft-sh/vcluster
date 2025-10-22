package snapshot

import (
	"context"
	"encoding/json"
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
)

func createSnapshot(f *framework.Framework, useNewCommand bool, snapshotPath string, includeVolumes bool) {
	By("Snapshot vcluster")
	var cmd *exec.Cmd
	if useNewCommand {
		// snapshots created asynchronously by the controller
		args := []string{
			"snapshot",
			"create",
			f.VClusterName,
			snapshotPath,
			"-n", f.VClusterNamespace,
		}
		if includeVolumes {
			args = append(args, "--include-volumes")
		}
		cmd = exec.Command(
			"vcluster",
			args...,
		)
	} else {
		// snapshots created synchronously by the CLI
		cmd = exec.Command(
			"vcluster",
			"snapshot",
			f.VClusterName,
			snapshotPath,
			"-n", f.VClusterNamespace,
			"--pod-mount", "pvc:snapshot-pvc:/snapshot-pvc",
		)
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	framework.ExpectNoError(err)
}

func restoreVCluster(ctx context.Context, f *framework.Framework, snapshotPath string, controllerBasedSnapshot, restoreVolumes bool) {
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

	waitUntilVClusterIsRunning(ctx, f)

	if restoreVolumes {
		waitForRestoreRequestToFinish(ctx, f)
	}
}

func waitUntilVClusterIsRunning(ctx context.Context, f *framework.Framework) {
	// wait until vCluster is running
	Eventually(func(g Gomega, ctx context.Context) {
		newPods, err := f.HostClient.CoreV1().Pods(f.VClusterNamespace).List(ctx, metav1.ListOptions{
			LabelSelector: "app=vcluster",
		})
		g.Expect(err).NotTo(HaveOccurred())

		g.Expect(newPods.Items).NotTo(BeEmpty())
		for _, pod := range newPods.Items {
			g.Expect(pod.Status.Phase).To(Equal(corev1.PodRunning))
		}
	}).WithContext(ctx).
		WithPolling(time.Second).
		WithTimeout(2 * time.Minute).
		Should(Succeed())

	// wait until all vCluster containers are running
	Eventually(func(g Gomega, ctx context.Context) {
		pods, err := f.HostClient.CoreV1().Pods(f.VClusterNamespace).List(ctx, metav1.ListOptions{
			LabelSelector: "app=vcluster,release=" + f.VClusterName,
		})
		g.Expect(err).NotTo(HaveOccurred())

		// check all containers
		for _, pod := range pods.Items {
			g.Expect(pod.Status.ContainerStatuses).NotTo(BeEmpty())
			for _, container := range pod.Status.ContainerStatuses {
				g.Expect(container.State.Running).NotTo(BeNil())
				g.Expect(container.Ready).To(BeTrue())
			}
		}
	}).WithContext(ctx).
		WithPolling(time.Second).
		WithTimeout(framework.PollTimeout).
		Should(Succeed())

	// refresh the connection
	err := f.RefreshVirtualClient()
	Expect(err).NotTo(HaveOccurred())
}

func waitForSnapshotToBeCreated(ctx context.Context, f *framework.Framework) {
	waitForRequestToFinish(ctx, f, constants.SnapshotRequestLabel, snapshot.UnmarshalSnapshotRequest, 5*time.Minute)
}

func waitForRestoreRequestToFinish(ctx context.Context, f *framework.Framework) {
	waitForRequestToFinish(ctx, f, constants.RestoreRequestLabel, snapshot.UnmarshalRestoreRequest, 5*time.Minute)
}

type unmarshalRequestFunc[T snapshot.LongRunningRequest] func(request *corev1.ConfigMap) (T, error)

func waitForRequestToFinish[T snapshot.LongRunningRequest](ctx context.Context, f *framework.Framework, requestLabel string, unmarshal unmarshalRequestFunc[T], timeout time.Duration) {
	Eventually(func(g Gomega, ctx context.Context) {
		listOptions := metav1.ListOptions{
			LabelSelector: requestLabel,
		}
		requestConfigMaps, err := f.HostClient.CoreV1().ConfigMaps(f.VClusterNamespace).List(ctx, listOptions)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(requestConfigMaps.Items).To(HaveLen(1))

		// extract snapshot/restore request
		requestConfigMap := requestConfigMaps.Items[0]
		request, err := unmarshal(&requestConfigMap)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(request).NotTo(BeNil())

		// check if the snapshot/restore request has been completed
		g.Expect(request.GetPhase()).To(
			Equal(snapshot.RequestPhaseCompleted),
			fmt.Sprintf("request is not completed, current phase is %s, request details: %s", request.GetPhase(), toJSON(request)))
	}).WithContext(ctx).
		WithPolling(framework.PollInterval).
		WithTimeout(timeout).
		Should(Succeed())
}

func toJSON[T interface{}](obj T) string {
	objJSON, _ := json.Marshal(obj)
	return string(objJSON)
}

func getTwoSnapshotRequests(g Gomega, ctx context.Context, f *framework.Framework) (*snapshot.Request, *snapshot.Request) {
	listOptions := metav1.ListOptions{
		LabelSelector: constants.SnapshotRequestLabel,
	}
	configMaps, err := f.HostClient.CoreV1().ConfigMaps(f.VClusterNamespace).List(ctx, listOptions)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(configMaps.Items).To(HaveLen(2))

	var previousConfigMap, newerConfigMap corev1.ConfigMap
	if configMaps.Items[0].CreationTimestamp.Time.Before(configMaps.Items[1].CreationTimestamp.Time) {
		previousConfigMap = configMaps.Items[0]
		newerConfigMap = configMaps.Items[1]
	} else {
		previousConfigMap = configMaps.Items[1]
		newerConfigMap = configMaps.Items[0]
	}
	previousSnapshotRequest, err := snapshot.UnmarshalSnapshotRequest(&previousConfigMap)
	g.Expect(err).NotTo(HaveOccurred())
	newerSnapshotRequest, err := snapshot.UnmarshalSnapshotRequest(&newerConfigMap)
	g.Expect(err).NotTo(HaveOccurred())

	return previousSnapshotRequest, newerSnapshotRequest
}

func deleteSnapshotRequestConfigMaps(ctx context.Context, f *framework.Framework) {
	// delete snapshot request config maps
	deleteOptions := metav1.DeleteOptions{}
	listOptions := metav1.ListOptions{
		LabelSelector: constants.SnapshotRequestLabel,
	}
	err := f.HostClient.CoreV1().ConfigMaps(f.VClusterNamespace).DeleteCollection(f.Context, deleteOptions, listOptions)
	framework.ExpectNoError(err)
	Eventually(func(g Gomega, ctx context.Context) []corev1.ConfigMap {
		configMaps, err := f.HostClient.CoreV1().ConfigMaps(f.VClusterNamespace).List(ctx, listOptions)
		g.Expect(err).NotTo(HaveOccurred())
		return configMaps.Items
	}).WithContext(ctx).
		WithPolling(framework.PollInterval).
		WithTimeout(framework.PollTimeout).
		Should(BeEmpty())
}
