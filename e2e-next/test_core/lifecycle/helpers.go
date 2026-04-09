package lifecycle

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/loft-sh/vcluster/e2e-next/constants"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func vclusterBinPath() string {
	return filepath.Join(os.Getenv("GOBIN"), "vcluster")
}

func localChartDir() string {
	return filepath.Join("..", "chart")
}

// createArgs returns the common arguments for creating a tenant cluster
// using the local chart and the dev image built from the current branch.
func createArgs(name, namespace string) []string {
	return []string{
		"create", name,
		"-n", namespace,
		"--connect=false",
		"--local-chart-dir", localChartDir(),
		// Set registry to empty because GetRepository() already includes it (e.g. "ghcr.io/loft-sh/vcluster"),
		// and the chart would otherwise prepend the default registry, producing "ghcr.io/ghcr.io/...".
		"--set", "controlPlane.statefulSet.image.registry=",
		"--set", "controlPlane.statefulSet.image.repository=" + constants.GetRepository(),
		"--set", "controlPlane.statefulSet.image.tag=" + constants.GetTag(),
	}
}

func runVClusterCmd(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, vclusterBinPath(), args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("vcluster %s failed: %w\noutput: %s", strings.Join(args, " "), err, string(out))
	}
	return string(out), nil
}

type listEntry struct {
	Name      string `json:"Name"`
	Namespace string `json:"Namespace"`
	Status    string `json:"Status"`
	Version   string `json:"Version"`
	Connected bool   `json:"Connected"`
}

func listVClusters(ctx context.Context, namespace string) ([]listEntry, error) {
	args := []string{"list", "--output", "json"}
	if namespace != "" {
		args = append(args, "-n", namespace)
	}
	out, err := runVClusterCmd(ctx, args...)
	if err != nil {
		return nil, err
	}
	var entries []listEntry
	if err := json.Unmarshal([]byte(out), &entries); err != nil {
		return nil, fmt.Errorf("parse vcluster list JSON: %w\nraw output: %s", err, out)
	}
	return entries, nil
}

func findByName(entries []listEntry, name string) *listEntry {
	for i := range entries {
		if entries[i].Name == name {
			return &entries[i]
		}
	}
	return nil
}

func scaleDownVCluster(ctx context.Context, hostClient kubernetes.Interface, name, namespace string) {
	GinkgoHelper()
	patch := []byte(`{"spec":{"replicas":0}}`)
	_, err := hostClient.AppsV1().StatefulSets(namespace).Patch(ctx, name, types.MergePatchType, patch, metav1.PatchOptions{})
	Expect(err).To(Succeed(), "scale down StatefulSet %s/%s", namespace, name)

	Eventually(func(g Gomega, ctx context.Context) {
		pods, err := hostClient.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
			LabelSelector: "app=vcluster,release=" + name,
		})
		g.Expect(err).To(Succeed())
		g.Expect(pods.Items).To(BeEmpty(), "tenant cluster pods should be gone after scale down")
	}).WithContext(ctx).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
}

func waitForVClusterReady(ctx context.Context, hostClient kubernetes.Interface, name, namespace string) {
	GinkgoHelper()
	Eventually(func(g Gomega, ctx context.Context) {
		pods, err := hostClient.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
			LabelSelector: "app=vcluster,release=" + name,
		})
		g.Expect(err).To(Succeed())
		g.Expect(pods.Items).NotTo(BeEmpty(), "no tenant cluster pods found")
		for _, pod := range pods.Items {
			g.Expect(pod.Status.Phase).To(Equal(corev1.PodRunning),
				"pod %s phase is %s, expected Running", pod.Name, pod.Status.Phase)
			for _, container := range pod.Status.ContainerStatuses {
				g.Expect(container.Ready).To(BeTrue(),
					"container %s in pod %s is not ready", container.Name, pod.Name)
			}
		}
	}).WithContext(ctx).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutVeryLong).Should(Succeed())
}

func createAndWaitForReady(ctx context.Context, hostClient kubernetes.Interface, clusterName, namespace string) {
	GinkgoHelper()
	By("Creating a tenant cluster", func() {
		_, err := runVClusterCmd(ctx, createArgs(clusterName, namespace)...)
		Expect(err).To(Succeed())
	})

	By("Waiting for tenant cluster to be ready", func() {
		waitForVClusterReady(ctx, hostClient, clusterName, namespace)
	})
}

func hostKubeClient() kubernetes.Interface {
	GinkgoHelper()
	kubeContext := "kind-" + constants.GetHostClusterName()
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{CurrentContext: kubeContext}
	cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides).ClientConfig()
	Expect(err).To(Succeed(), "build host cluster rest config for context %s", kubeContext)

	client, err := kubernetes.NewForConfig(cfg)
	Expect(err).To(Succeed(), "create host cluster kube client")
	return client
}
