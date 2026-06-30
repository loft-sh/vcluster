// Package testmigration contains vCluster upgrade and migration tests.
package testmigration

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e/constants"
	"github.com/loft-sh/vcluster/e2e/labels"
	"github.com/loft-sh/vcluster/pkg/certs"
	"github.com/loft-sh/vcluster/pkg/util/random"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/retry"
)

const (
	k3sBaseChartVersion       = "0.32.2"
	migratedFromK3sAnnotation = "vcluster.loft.sh/migrated-from-k3s"
	k3sNodeFinalizer          = "wrangler.cattle.io/node"
)

// K3SToK8SMigrationSpec verifies that k3s-era CA material is migrated into a
// k8s distro vCluster in a way that survives later kubeadm leaf regeneration.
func K3SToK8SMigrationSpec() {
	Describe("k3s to k8s migration",
		labels.Migration,
		Ordered,
		func() {
			var (
				clusterName       string
				namespace         string
				hostClient        kubernetes.Interface
				initialRootCA     []byte
				finalizerNodeName string
			)

			BeforeAll(func(ctx context.Context) {
				clusterName = "migration-k3s-" + random.String(6)
				namespace = clusterName
				hostClient = cluster.KubeClientFrom(ctx, constants.GetHostClusterName())
				Expect(hostClient).NotTo(BeNil())
			})

			AfterAll(func(ctx context.Context) {
				_, err := runVClusterCmd(ctx, "delete", clusterName, "-n", namespace, "--delete-namespace", "--ignore-not-found")
				Expect(err).To(Succeed())
			})

			It("creates a k3s vCluster from the last k3s-capable chart", func(ctx context.Context) {
				_, err := runHelmCmd(ctx, createK3SHelmArgs(clusterName, namespace)...)
				Expect(err).To(Succeed())

				waitForVClusterPodsReady(ctx, hostClient, namespace, clusterName)

				secret := getCertSecret(ctx, hostClient, namespace, clusterName)
				initialRootCA = append([]byte(nil), secret.Data[certs.CACertName]...)
				Expect(initialRootCA).NotTo(BeEmpty())
				Expect(secret.Data[certs.ClientCACertName]).To(Equal(initialRootCA))
				Expect(secret.Data[certs.ServerCACertName]).To(Equal(initialRootCA))
			})

			// Under the k3s distro, k3s's wrangler node controller stamps this finalizer onto the
			// synced node objects on its own. We capture an affected node here so a later spec can
			// assert the migration strips it; otherwise a terminated host node would ghost.
			It("has k3s stamp the node finalizer onto synced nodes", func(ctx context.Context) {
				vClusterClient := connectVClusterClient(ctx, namespace, clusterName)
				finalizerNodeName = awaitNodeWithK3sFinalizer(ctx, vClusterClient)
			})

			It("migrates k3s CA aliases when upgraded to the local k8s chart", func(ctx context.Context) {
				_, err := runHelmCmd(ctx, upgradeToLocalK8SHelmArgs(clusterName, namespace)...)
				Expect(err).To(Succeed())

				waitForVClusterPodsReady(ctx, hostClient, namespace, clusterName)

				Eventually(func(g Gomega, ctx context.Context) {
					secret := getCertSecret(ctx, hostClient, namespace, clusterName)
					g.Expect(secret.Annotations).To(HaveKeyWithValue(migratedFromK3sAnnotation, "true"))

					ca := secret.Data[certs.CACertName]
					clientCA := secret.Data[certs.ClientCACertName]
					serverCA := secret.Data[certs.ServerCACertName]

					g.Expect(ca).NotTo(BeEmpty())
					g.Expect(clientCA).NotTo(BeEmpty())
					g.Expect(serverCA).NotTo(BeEmpty())
					g.Expect(ca).To(Equal(clientCA), "ca.crt must follow the migrated k3s client CA")
					g.Expect(secret.Data[certs.CAKeyName]).To(Equal(secret.Data[certs.ClientCAKeyName]))
					g.Expect(ca).NotTo(Equal(initialRootCA), "ca.crt should not remain the pre-migration kubeadm CA")
					g.Expect(serverCA).NotTo(Equal(clientCA), "k3s server CA should stay split until kubeadm regenerates leaves")
				}).WithContext(ctx).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutVeryLong).Should(Succeed())
			})

			// The migration step in StartControllers should have stripped the orphaned k3s
			// finalizer, so a terminated host node no longer gets stuck as a ghost node.
			It("removes the orphaned k3s node finalizer after migrating to the k8s distro", func(ctx context.Context) {
				vClusterClient := connectVClusterClient(ctx, namespace, clusterName)

				Eventually(func(g Gomega, ctx context.Context) {
					node, err := vClusterClient.CoreV1().Nodes().Get(ctx, finalizerNodeName, metav1.GetOptions{})
					g.Expect(err).To(Succeed())
					g.Expect(node.Finalizers).NotTo(ContainElement(k3sNodeFinalizer),
						"node %s still carries the orphaned k3s finalizer after migration", finalizerNodeName)
				}).WithContext(ctx).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutVeryLong).Should(Succeed())
			})

			It("regenerates leaf certs without breaking client authentication", func(ctx context.Context) {
				deleteCertSecretKey(ctx, hostClient, namespace, clusterName, certs.APIServerCertName)
				restartVClusterPods(ctx, hostClient, namespace, clusterName)
				waitForVClusterPodsReady(ctx, hostClient, namespace, clusterName)

				Eventually(func(g Gomega, ctx context.Context) {
					secret := getCertSecret(ctx, hostClient, namespace, clusterName)
					ca := secret.Data[certs.CACertName]
					clientCA := secret.Data[certs.ClientCACertName]
					serverCA := secret.Data[certs.ServerCACertName]

					g.Expect(ca).To(Equal(clientCA))
					g.Expect(serverCA).NotTo(Equal(clientCA), "leaf renewal should preserve the migrated k3s server CA")
					g.Expect(secret.Data[certs.CAKeyName]).To(Equal(secret.Data[certs.ClientCAKeyName]))
					g.Expect(secret.Data[certs.ServerCAKeyName]).NotTo(Equal(secret.Data[certs.ClientCAKeyName]))
					expectCertSignedBy(g, secret.Data[certs.APIServerKubeletClientCertName], ca)
					expectCertSignedBy(g, secret.Data[certs.APIServerCertName], serverCA)
					expectKubeConfigTrustsCA(g, secret.Data[certs.AdminKubeConfigFileName], serverCA)
					expectKubeConfigTrustsCA(g, secret.Data[certs.ControllerManagerKubeConfigFileName], serverCA)
					expectKubeConfigTrustsCA(g, secret.Data[certs.SchedulerKubeConfigFileName], serverCA)
				}).WithContext(ctx).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutVeryLong).Should(Succeed())

				expectVClusterAPIReachable(ctx, namespace, clusterName)
			})
		},
	)
}

func createK3SHelmArgs(name, namespace string) []string {
	return []string{
		"upgrade", "--install", name,
		oldChartURL(),
		"-n", namespace,
		"--create-namespace",
		"--set", "controlPlane.distro.k3s.enabled=true",
		"--set", "controlPlane.distro.k8s.enabled=false",
		"--set", "sync.fromHost.nodes.enabled=true",
		"--set", "sync.fromHost.nodes.selector.all=true",
		"--set", "controlPlane.statefulSet.image.registry=ghcr.io",
		"--set", "controlPlane.statefulSet.image.repository=loft-sh/vcluster-oss",
		"--set", "controlPlane.statefulSet.image.tag=" + k3sBaseChartVersion,
	}
}

func upgradeToLocalK8SHelmArgs(name, namespace string) []string {
	return []string{
		"upgrade", "--install", name,
		filepath.Join("..", "chart"),
		"-n", namespace,
		"--reset-values",
		"--set", "controlPlane.distro.k8s.enabled=true",
		"--set", "controlPlane.distro.k8s.image.tag=v1.35.0",
		"--set", "sync.fromHost.nodes.enabled=true",
		"--set", "sync.fromHost.nodes.selector.all=true",
		"--set", "controlPlane.statefulSet.image.registry=",
		"--set", "controlPlane.statefulSet.image.repository=" + constants.GetRepository(),
		"--set", "controlPlane.statefulSet.image.tag=" + constants.GetTag(),
	}
}

func oldChartURL() string {
	return "https://charts.loft.sh/charts/vcluster-" + k3sBaseChartVersion + ".tgz"
}

func runHelmCmd(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "helm", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("helm %s failed: %w\noutput: %s", strings.Join(args, " "), err, string(out))
	}
	return string(out), nil
}

func runVClusterCmd(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, filepath.Join(os.Getenv("GOBIN"), "vcluster"), args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("vcluster %s failed: %w\noutput: %s", strings.Join(args, " "), err, string(out))
	}
	return string(out), nil
}

func getCertSecret(ctx context.Context, hostClient kubernetes.Interface, namespace, vClusterName string) *corev1.Secret {
	GinkgoHelper()
	secret, err := hostClient.CoreV1().Secrets(namespace).Get(ctx, certs.CertSecretName(vClusterName), metav1.GetOptions{})
	Expect(err).To(Succeed())
	return secret
}

func waitForVClusterPodsReady(ctx context.Context, hostClient kubernetes.Interface, namespace, vClusterName string) {
	GinkgoHelper()
	Eventually(func(g Gomega, ctx context.Context) {
		pods, err := hostClient.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
			LabelSelector: "app=vcluster,release=" + vClusterName,
		})
		g.Expect(err).To(Succeed())
		g.Expect(pods.Items).NotTo(BeEmpty(), "no vCluster pods found")
		for _, pod := range pods.Items {
			g.Expect(pod.Status.Phase).To(Equal(corev1.PodRunning),
				"pod %s phase is %s, expected Running", pod.Name, pod.Status.Phase)
			g.Expect(pod.Status.ContainerStatuses).NotTo(BeEmpty(),
				"pod %s has no container statuses", pod.Name)
			for _, container := range pod.Status.ContainerStatuses {
				g.Expect(container.Ready).To(BeTrue(),
					"container %s in pod %s is not ready", container.Name, pod.Name)
			}
		}
	}).WithContext(ctx).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutVeryLong).Should(Succeed())
}

func deleteCertSecretKey(ctx context.Context, hostClient kubernetes.Interface, namespace, vClusterName, key string) {
	GinkgoHelper()
	err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		secret, err := hostClient.CoreV1().Secrets(namespace).Get(ctx, certs.CertSecretName(vClusterName), metav1.GetOptions{})
		if err != nil {
			return err
		}
		delete(secret.Data, key)
		_, err = hostClient.CoreV1().Secrets(namespace).Update(ctx, secret, metav1.UpdateOptions{})
		return err
	})
	Expect(err).To(Succeed())
}

func restartVClusterPods(ctx context.Context, hostClient kubernetes.Interface, namespace, vClusterName string) {
	GinkgoHelper()
	pods, err := hostClient.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "app=vcluster,release=" + vClusterName,
	})
	Expect(err).To(Succeed())
	Expect(pods.Items).NotTo(BeEmpty())

	oldUIDs := map[string]struct{}{}
	for _, pod := range pods.Items {
		oldUIDs[string(pod.UID)] = struct{}{}
		err := hostClient.CoreV1().Pods(namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{})
		Expect(err).To(Succeed())
	}

	Eventually(func(g Gomega, ctx context.Context) {
		pods, err := hostClient.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
			LabelSelector: "app=vcluster,release=" + vClusterName,
		})
		g.Expect(err).To(Succeed())
		g.Expect(pods.Items).NotTo(BeEmpty())
		for _, pod := range pods.Items {
			_, old := oldUIDs[string(pod.UID)]
			g.Expect(old).To(BeFalse(), "pod %s still has pre-restart UID", pod.Name)
		}
	}).WithContext(ctx).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutVeryLong).Should(Succeed())
}

func expectCertSignedBy(g Gomega, certPEM, caPEM []byte) {
	GinkgoHelper()
	cert := parseCertificate(g, certPEM)
	ca := parseCertificate(g, caPEM)
	g.Expect(cert.CheckSignatureFrom(ca)).To(Succeed(),
		"certificate issuer %q should be signed by CA subject %q", cert.Issuer.String(), ca.Subject.String())
}

func expectKubeConfigTrustsCA(g Gomega, kubeConfigBytes, caPEM []byte) {
	GinkgoHelper()
	kubeConfig, err := clientcmd.Load(kubeConfigBytes)
	g.Expect(err).To(Succeed())
	for _, cluster := range kubeConfig.Clusters {
		g.Expect(cluster.CertificateAuthority).To(BeEmpty())
		g.Expect(cluster.CertificateAuthorityData).To(Equal(caPEM))
	}
}

func parseCertificate(g Gomega, certPEM []byte) *x509.Certificate {
	GinkgoHelper()
	block, _ := pem.Decode(certPEM)
	g.Expect(block).NotTo(BeNil(), "failed to decode certificate PEM")
	g.Expect(block.Type).To(Equal("CERTIFICATE"))
	cert, err := x509.ParseCertificate(block.Bytes)
	g.Expect(err).To(Succeed())
	return cert
}

func expectVClusterAPIReachable(ctx context.Context, namespace, vClusterName string) {
	GinkgoHelper()
	connectVClusterClient(ctx, namespace, vClusterName)
}

// connectVClusterClient connects to the tenant cluster via a background proxy and returns a
// client for it. The connection is one-shot, so callers must reconnect after destructive
// operations such as a helm upgrade that restarts the vCluster pod.
func connectVClusterClient(ctx context.Context, namespace, vClusterName string) kubernetes.Interface {
	GinkgoHelper()
	tmpFile, err := os.CreateTemp("", "vcluster-migration-kubeconfig-*")
	Expect(err).To(Succeed())
	Expect(tmpFile.Close()).To(Succeed())
	DeferCleanup(func() { _ = os.Remove(tmpFile.Name()) })

	_, err = runVClusterCmd(ctx,
		"connect", vClusterName,
		"-n", namespace,
		"--driver", "helm",
		"--kube-config", tmpFile.Name(),
		"--update-current=false",
		"--background-proxy=true",
		"--background-proxy-image", constants.GetVClusterImage(),
	)
	Expect(err).To(Succeed())

	var vClusterClient kubernetes.Interface
	Eventually(func(g Gomega, ctx context.Context) {
		data, err := os.ReadFile(tmpFile.Name())
		g.Expect(err).To(Succeed())
		g.Expect(bytes.TrimSpace(data)).NotTo(BeEmpty(), "kubeconfig file is still empty after connect")

		restConfig, err := clientcmd.RESTConfigFromKubeConfig(data)
		g.Expect(err).To(Succeed())

		vClusterClient, err = kubernetes.NewForConfig(restConfig)
		g.Expect(err).To(Succeed())

		_, err = vClusterClient.CoreV1().ServiceAccounts("default").Get(ctx, "default", metav1.GetOptions{})
		g.Expect(err).To(Succeed())
	}).WithContext(ctx).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())

	return vClusterClient
}

// awaitNodeWithK3sFinalizer waits for the k3s distro to stamp the node finalizer (k3sNodeFinalizer)
// onto a synced node on its own and returns that node's name, so a later spec can assert the
// migration cleaned it up. If k3s never adds it, this fails — the correct signal that the bug's
// premise no longer holds.
func awaitNodeWithK3sFinalizer(ctx context.Context, vClusterClient kubernetes.Interface) string {
	GinkgoHelper()
	var nodeName string
	Eventually(func(g Gomega, ctx context.Context) {
		nodes, err := vClusterClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
		g.Expect(err).To(Succeed())
		g.Expect(nodes.Items).NotTo(BeEmpty(), "no nodes were synced into the tenant cluster")

		nodeName = ""
		for i := range nodes.Items {
			if slices.Contains(nodes.Items[i].Finalizers, k3sNodeFinalizer) {
				nodeName = nodes.Items[i].Name
				break
			}
		}
		g.Expect(nodeName).NotTo(BeEmpty(), "no synced node has acquired the %q finalizer from k3s", k3sNodeFinalizer)
	}).WithContext(ctx).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutVeryLong).Should(Succeed())

	return nodeName
}
