// Package certs contains certificate rotation and expiration tests.
package certs

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	loftlog "github.com/loft-sh/log"
	connectcmd "github.com/loft-sh/vcluster/cmd/vclusterctl/cmd"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/pkg/certs"
	"github.com/loft-sh/vcluster/pkg/cli"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	utilportforward "github.com/loft-sh/vcluster/pkg/util/portforward"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/client-go/transport/spdy"
	"k8s.io/kubectl/pkg/scheme"
)

const certRotationAnnotation = "vcluster.loft.sh/cert-rotation-at"

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

func listPodsBySelector(ctx context.Context, hostClient kubernetes.Interface, namespace, selector string) []corev1.Pod {
	GinkgoHelper()
	pods, err := hostClient.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: selector})
	Expect(err).To(Succeed(), "listing pods in %s with selector %q", namespace, selector)
	return pods.Items
}

func podUIDs(pods []corev1.Pod) map[string]struct{} {
	result := make(map[string]struct{}, len(pods))
	for _, pod := range pods {
		result[string(pod.UID)] = struct{}{}
	}
	return result
}

func waitForPodsRolled(ctx context.Context, hostClient kubernetes.Interface, namespace, selector string, previousUIDs map[string]struct{}, expectedMinPods int, timeout time.Duration) {
	GinkgoHelper()
	Eventually(func(g Gomega) {
		pods, err := hostClient.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: selector})
		g.Expect(err).To(Succeed(), "listing pods in %s with selector %q", namespace, selector)
		g.Expect(pods.Items).To(HaveLen(expectedMinPods), "unexpected number of pods after rollout")

		for _, pod := range pods.Items {
			_, exists := previousUIDs[string(pod.UID)]
			g.Expect(exists).To(BeFalse(), "pod %s still has a pre-rollout UID", pod.Name)
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

func getControlPlaneRolloutAnnotation(ctx context.Context, hostClient kubernetes.Interface, namespace, vClusterName string) (string, error) {
	sts, err := hostClient.AppsV1().StatefulSets(namespace).Get(ctx, vClusterName, metav1.GetOptions{})
	if err == nil {
		return sts.Spec.Template.Annotations[certRotationAnnotation], nil
	}
	if !kerrors.IsNotFound(err) {
		return "", err
	}

	deploy, err := hostClient.AppsV1().Deployments(namespace).Get(ctx, vClusterName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	return deploy.Spec.Template.Annotations[certRotationAnnotation], nil
}

func getStatefulSetRolloutAnnotation(ctx context.Context, hostClient kubernetes.Interface, namespace, name string) (string, error) {
	sts, err := hostClient.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	return sts.Spec.Template.Annotations[certRotationAnnotation], nil
}

// expectCertRenewed polls the cert secret until apiserver.crt has a NotAfter
// more than 90 days from now, indicating it was renewed.
func expectCertRenewed(ctx context.Context, hostClient kubernetes.Interface, namespace, vClusterName string, timeout time.Duration) {
	GinkgoHelper()
	Eventually(func(g Gomega) {
		secret, err := hostClient.CoreV1().Secrets(namespace).Get(ctx,
			certs.CertSecretName(vClusterName), metav1.GetOptions{})
		g.Expect(err).To(Succeed())

		block, _ := pem.Decode(secret.Data["apiserver.crt"])
		g.Expect(block).NotTo(BeNil(), "failed to decode apiserver cert PEM")

		cert, err := x509.ParseCertificate(block.Bytes)
		g.Expect(err).To(Succeed())

		g.Expect(cert.NotAfter.After(time.Now().Add(90*24*time.Hour))).To(BeTrue(),
			"apiserver cert should have been renewed, NotAfter=%s", cert.NotAfter.Format(time.RFC3339))
	}).WithPolling(constants.PollingInterval).WithTimeout(timeout).Should(Succeed())
}

// execWriteFile writes data to a file inside a container using kubectl exec.
func execWriteFile(ctx context.Context, restConfig *rest.Config, client kubernetes.Interface, namespace, podName, container, filePath string, data []byte) error {
	cmd := []string{"sh", "-c", fmt.Sprintf("cat > %s", filePath)}

	req := client.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: container,
			Command:   cmd,
			Stdin:     true,
			Stdout:    false,
			Stderr:    true,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(restConfig, "POST", req.URL())
	if err != nil {
		return fmt.Errorf("creating executor: %w", err)
	}

	reader := bytes.NewReader(data)
	var stderr bytes.Buffer
	err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdin:  reader,
		Stderr: &stderr,
	})
	if err != nil {
		return fmt.Errorf("exec failed: %w (stderr: %s)", err, stderr.String())
	}
	return nil
}

// containerRestartCounts returns a map of container name to restart count for
// all containers in pods matching the given selector.
func containerRestartCounts(ctx context.Context, hostClient kubernetes.Interface, namespace, selector string) map[string]int32 {
	GinkgoHelper()
	pods := listPodsBySelector(ctx, hostClient, namespace, selector)
	counts := make(map[string]int32)
	for _, pod := range pods {
		for _, cs := range pod.Status.ContainerStatuses {
			key := pod.Name + "/" + cs.Name
			counts[key] = cs.RestartCount
		}
	}
	return counts
}

// getServingCertSerial port-forwards to a vcluster pod and returns the serial
// number of the TLS serving certificate from the handshake. This is the cert
// generated by the syncer's cert syncer, not the kubeadm apiserver.crt.
func getServingCertSerial(ctx context.Context, restConfig *rest.Config, client kubernetes.Interface, namespace, podName string) (string, error) {
	t, upgrader, err := spdy.RoundTripperFor(restConfig)
	if err != nil {
		return "", fmt.Errorf("creating round tripper: %w", err)
	}

	req := client.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("portforward")

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: t}, "POST", req.URL())
	stopChan := make(chan struct{})
	readyChan := make(chan struct{})
	errChan := make(chan error, 1)

	fw, err := utilportforward.NewOnAddresses(dialer, []string{"127.0.0.1"}, []string{"0:8443"}, stopChan, readyChan, errChan, io.Discard, io.Discard)
	if err != nil {
		return "", fmt.Errorf("creating port forwarder: %w", err)
	}

	go func() {
		if err := fw.ForwardPorts(ctx); err != nil {
			errChan <- err
		}
	}()

	select {
	case err := <-errChan:
		close(stopChan)
		return "", fmt.Errorf("port forward failed: %w", err)
	case <-readyChan:
	}
	defer close(stopChan)

	ports, err := fw.GetPorts()
	if err != nil {
		return "", fmt.Errorf("getting forwarded ports: %w", err)
	}

	conn, err := tls.DialWithDialer(
		&net.Dialer{Timeout: 5 * time.Second},
		"tcp",
		fmt.Sprintf("127.0.0.1:%d", ports[0].Local),
		&tls.Config{InsecureSkipVerify: true},
	)
	if err != nil {
		return "", fmt.Errorf("TLS dial: %w", err)
	}
	defer conn.Close()

	peerCerts := conn.ConnectionState().PeerCertificates
	if len(peerCerts) == 0 {
		return "", fmt.Errorf("no peer certificates")
	}
	return peerCerts[0].SerialNumber.String(), nil
}
