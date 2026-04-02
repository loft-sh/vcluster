package certs

import (
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"os"
	"strings"
	"time"

	loftlog "github.com/loft-sh/log"
	connectcmd "github.com/loft-sh/vcluster/cmd/vclusterctl/cmd"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/pkg/cli"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// reconnectVCluster establishes a fresh vcluster connection using a background proxy.
// Call this after destructive operations (cert rotation, restart) that kill the suite proxy.
// Returns a new *rest.Config and *kubernetes.Clientset connected to the vcluster.
func reconnectVCluster(ctx context.Context, vClusterName, vClusterNamespace string) (*rest.Config, *kubernetes.Clientset) {
	GinkgoHelper()

	tmpFile, err := os.CreateTemp("", "vcluster-certs-kubeconfig-*")
	Expect(err).To(Succeed(), "creating temp kubeconfig file")
	tmpFile.Close()
	DeferCleanup(func(_ context.Context) { _ = os.Remove(tmpFile.Name()) })

	connectCmd := connectcmd.ConnectCmd{
		CobraCmd: &cobra.Command{},
		Log:      loftlog.Discard,
		GlobalFlags: &flags.GlobalFlags{
			Namespace: vClusterNamespace,
		},
		ConnectOptions: cli.ConnectOptions{
			KubeConfig:           tmpFile.Name(),
			BackgroundProxy:      true,
			BackgroundProxyImage: constants.GetVClusterImage(),
		},
	}
	err = connectCmd.Run(ctx, []string{vClusterName})
	Expect(err).To(Succeed(), "vcluster connect failed after cert rotation")

	var restConfig *rest.Config
	var vClusterClient *kubernetes.Clientset

	Eventually(func(g Gomega) {
		data, err := os.ReadFile(tmpFile.Name())
		g.Expect(err).To(Succeed(), "reading temp kubeconfig")
		g.Expect(data).NotTo(BeEmpty(), "kubeconfig file is still empty after connect")

		restConfig, err = clientcmd.RESTConfigFromKubeConfig(data)
		g.Expect(err).To(Succeed(), "parsing kubeconfig")

		vClusterClient, err = kubernetes.NewForConfig(restConfig)
		g.Expect(err).To(Succeed(), "building kubernetes client from new kubeconfig")

		_, err = vClusterClient.CoreV1().ServiceAccounts("default").Get(ctx, "default", metav1.GetOptions{})
		g.Expect(err).To(Succeed(), "vcluster not yet reachable after cert rotation")
	}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())

	return restConfig, vClusterClient
}

// waitForPodsReady polls until all vcluster pods in the given namespace are running and ready.
func waitForPodsReady(ctx context.Context, hostClient kubernetes.Interface, vClusterNamespace, vClusterName string, timeout time.Duration) {
	GinkgoHelper()
	Eventually(func(g Gomega) {
		pods, err := hostClient.CoreV1().Pods(vClusterNamespace).List(ctx, metav1.ListOptions{
			LabelSelector: "app=vcluster,release=" + vClusterName,
		})
		g.Expect(err).To(Succeed(), "listing vcluster pods in %s", vClusterNamespace)
		g.Expect(pods.Items).NotTo(BeEmpty(), "no vcluster pods found in %s", vClusterNamespace)

		for _, pod := range pods.Items {
			g.Expect(pod.Status.ContainerStatuses).NotTo(BeEmpty(),
				"pod %s has no container statuses", pod.Name)
			for _, container := range pod.Status.ContainerStatuses {
				g.Expect(container.State.Running).NotTo(BeNil(),
					"container %s in pod %s is not running", container.Name, pod.Name)
				g.Expect(container.Ready).To(BeTrue(),
					"container %s in pod %s is not ready", container.Name, pod.Name)
			}
		}
	}).WithPolling(constants.PollingInterval).WithTimeout(timeout).Should(Succeed())
}

// parseCertFromPEM decodes the first PEM block and parses it as an x509 certificate.
func parseCertFromPEM(pemData []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, fmt.Errorf("decoding to PEM block")
	}
	if block.Type != "CERTIFICATE" {
		return nil, fmt.Errorf("not a certificate (type: %s)", block.Type)
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parsing certificate: %w", err)
	}
	return cert, nil
}

// certFingerprint returns a colon-separated uppercase SHA-256 hex fingerprint.
func certFingerprint(cert *x509.Certificate) string {
	hash := sha256.Sum256(cert.Raw)
	fingerprint := hex.EncodeToString(hash[:])

	var formatted strings.Builder
	for i := 0; i < len(fingerprint); i += 2 {
		if i > 0 {
			formatted.WriteString(":")
		}
		formatted.WriteString(strings.ToUpper(fingerprint[i : i+2]))
	}
	return formatted.String()
}
