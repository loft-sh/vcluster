package test_cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/loft-sh/vcluster/e2e-next/constants"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func vclusterBinPath() string {
	return filepath.Join(os.Getenv("GOBIN"), "vcluster")
}

func localChartDir() string {
	return filepath.Join("..", "chart")
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
	zero := int32(0)
	sts, err := hostClient.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
	Expect(err).To(Succeed(), "get StatefulSet %s/%s", namespace, name)

	sts.Spec.Replicas = &zero
	_, err = hostClient.AppsV1().StatefulSets(namespace).Update(ctx, sts, metav1.UpdateOptions{})
	Expect(err).To(Succeed(), "scale down StatefulSet %s/%s", namespace, name)

	Eventually(func(g Gomega) {
		pods, err := hostClient.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
			LabelSelector: "app=vcluster,release=" + name,
		})
		g.Expect(err).To(Succeed())
		g.Expect(pods.Items).To(BeEmpty(), "tenant cluster pods should be gone after scale down")
	}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
}

func waitForVClusterReady(ctx context.Context, hostClient kubernetes.Interface, name, namespace string) {
	Eventually(func(g Gomega) {
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
	}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutVeryLong).Should(Succeed())
}

func hostKubeClient() kubernetes.Interface {
	kubeContext := "kind-" + constants.GetHostClusterName()
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{CurrentContext: kubeContext}
	cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides).ClientConfig()
	Expect(err).To(Succeed(), "build host cluster rest config for context %s", kubeContext)

	client, err := kubernetes.NewForConfig(cfg)
	Expect(err).To(Succeed(), "create host cluster kube client")
	return client
}
