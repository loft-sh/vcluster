package snapshot

import (
	"context"
	"encoding/json"
	"os/exec"
	"time"

	"github.com/loft-sh/vcluster/e2e-next/constants"
	pkgconstants "github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/snapshot"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func runVClusterCmd(args ...string) {
	GinkgoHelper()
	cmd := exec.Command("vcluster", args...)
	cmd.Stdout = GinkgoWriter
	cmd.Stderr = GinkgoWriter
	err := cmd.Run()
	Expect(err).NotTo(HaveOccurred(), "vcluster command failed: vcluster %v", args)
}

func createSnapshot(vClusterName, vClusterNamespace string, useNewCommand bool, snapshotPath string, includeVolumes bool) {
	GinkgoHelper()
	By("Creating a snapshot", func() {
		if useNewCommand {
			args := []string{"snapshot", "create", vClusterName, snapshotPath, "-n", vClusterNamespace}
			if includeVolumes {
				args = append(args, "--include-volumes")
			}
			runVClusterCmd(args...)
		} else {
			runVClusterCmd("snapshot", vClusterName, snapshotPath, "-n", vClusterNamespace,
				"--pod-mount", "pvc:snapshot-pvc:/snapshot-pvc")
		}
	})
}

func restoreVCluster(ctx context.Context, hostClient kubernetes.Interface, vClusterName, vClusterNamespace, snapshotPath string, controllerBased, restoreVolumes bool) {
	GinkgoHelper()
	By("Restoring the vCluster", func() {
		args := []string{"restore", vClusterName, snapshotPath, "-n", vClusterNamespace}
		if !controllerBased {
			args = append(args, "--pod-mount", "pvc:snapshot-pvc:/snapshot-pvc")
		}
		if restoreVolumes {
			args = append(args, "--restore-volumes")
		}
		runVClusterCmd(args...)
	})

	By("Waiting for vCluster pods to be running after restore", func() {
		Eventually(func(g Gomega) {
			pods, err := hostClient.CoreV1().Pods(vClusterNamespace).List(ctx, metav1.ListOptions{
				LabelSelector: "app=vcluster",
			})
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(pods.Items).NotTo(BeEmpty(), "no vcluster pods found")
			for _, pod := range pods.Items {
				g.Expect(pod.Status.Phase).To(Equal(corev1.PodRunning),
					"pod %s phase is %s, expected Running", pod.Name, pod.Status.Phase)
			}
		}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutVeryLong).Should(Succeed())
	})

	By("Waiting for all vCluster containers to be ready", func() {
		Eventually(func(g Gomega) {
			pods, err := hostClient.CoreV1().Pods(vClusterNamespace).List(ctx, metav1.ListOptions{
				LabelSelector: "app=vcluster,release=" + vClusterName,
			})
			g.Expect(err).NotTo(HaveOccurred())
			for _, pod := range pods.Items {
				g.Expect(pod.Status.ContainerStatuses).NotTo(BeEmpty())
				for _, container := range pod.Status.ContainerStatuses {
					g.Expect(container.State.Running).NotTo(BeNil(),
						"container %s in pod %s is not running", container.Name, pod.Name)
					g.Expect(container.Ready).To(BeTrue(),
						"container %s in pod %s is not ready", container.Name, pod.Name)
				}
			}
		}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
	})

	if restoreVolumes {
		waitForRequestToFinish(ctx, hostClient, vClusterNamespace, pkgconstants.RestoreRequestLabel, snapshot.UnmarshalRestoreRequest, 5*time.Minute)
	}
}

func waitForSnapshotToBeCreated(ctx context.Context, hostClient kubernetes.Interface, vClusterNamespace string) {
	waitForRequestToFinish(ctx, hostClient, vClusterNamespace, pkgconstants.SnapshotRequestLabel, snapshot.UnmarshalSnapshotRequest, 5*time.Minute)
}

type unmarshalRequestFunc[T snapshot.LongRunningRequest] func(request *corev1.ConfigMap) (T, error)

func waitForRequestToFinish[T snapshot.LongRunningRequest](ctx context.Context, hostClient kubernetes.Interface, vClusterNamespace, requestLabel string, unmarshal unmarshalRequestFunc[T], timeout time.Duration) {
	GinkgoHelper()
	Eventually(func(g Gomega) {
		requestConfigMaps, err := hostClient.CoreV1().ConfigMaps(vClusterNamespace).List(ctx, metav1.ListOptions{
			LabelSelector: requestLabel,
		})
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(requestConfigMaps.Items).To(HaveLen(1),
			"expected 1 request configmap with label %s, got %d", requestLabel, len(requestConfigMaps.Items))

		request, err := unmarshal(&requestConfigMaps.Items[0])
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(request).NotTo(BeNil())
		g.Expect(request.GetPhase()).To(
			Equal(snapshot.RequestPhaseCompleted),
			"request not completed, phase: %s, details: %s", request.GetPhase(), toJSON(request))
	}).WithPolling(constants.PollingInterval).WithTimeout(timeout).Should(Succeed())
}

func deleteSnapshotRequestConfigMaps(ctx context.Context, hostClient kubernetes.Interface, vClusterNamespace string) {
	GinkgoHelper()
	listOptions := metav1.ListOptions{
		LabelSelector: pkgconstants.SnapshotRequestLabel,
	}
	err := hostClient.CoreV1().ConfigMaps(vClusterNamespace).DeleteCollection(ctx, metav1.DeleteOptions{}, listOptions)
	Expect(err).NotTo(HaveOccurred())
	Eventually(func(g Gomega) {
		configMaps, err := hostClient.CoreV1().ConfigMaps(vClusterNamespace).List(ctx, listOptions)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(configMaps.Items).To(BeEmpty(), "snapshot request configmaps not yet deleted")
	}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
}

func toJSON(obj any) string {
	b, _ := json.Marshal(obj)
	return string(b)
}
