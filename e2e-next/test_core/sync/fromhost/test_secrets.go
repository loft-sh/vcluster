package fromhost

import (
	"bytes"
	"context"
	"strings"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/e2e-framework/pkg/setup/suite"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	"github.com/loft-sh/vcluster/pkg/util/podhelper"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/ptr"
)

// DescribeFromHostSecrets registers secret sync from host tests against the given vCluster.
func DescribeFromHostSecrets(vcluster suite.Dependency) bool {
	return Describe("Secrets sync from host",
		labels.Core,
		labels.PR,
		labels.Sync,
		labels.Secrets,
		cluster.Use(vcluster),
		func() {
			var (
				hostClient     kubernetes.Interface
				vClusterClient kubernetes.Interface
				vClusterHostNS string
				vClusterName   string
			)

			BeforeEach(func(ctx context.Context) {
				hostClient = cluster.KubeClientFrom(ctx, constants.GetHostClusterName())
				Expect(hostClient).NotTo(BeNil())
				vClusterClient = cluster.CurrentKubeClientFrom(ctx)
				Expect(vClusterClient).NotTo(BeNil())
				vClusterName = cluster.CurrentClusterNameFrom(ctx)
				vClusterHostNS = "vcluster-" + vClusterName
			})

			// ensureNamespace creates a namespace idempotently. It does NOT register a
			// DeferCleanup for the namespace itself - callers that own the namespace must
			// do so themselves. For shared/fixed namespaces leave deletion to vcluster teardown.
			ensureNamespace := func(ctx context.Context, name string) {
				GinkgoHelper()
				_, err := hostClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: name},
				}, metav1.CreateOptions{})
				if !kerrors.IsAlreadyExists(err) {
					Expect(err).To(Succeed())
				}
			}

			It("syncs secrets from wildcard namespace and propagates updates", func(ctx context.Context) {
				// from-host-sync-test-2/* maps to barfoo2/* in the vcluster config.
				hostNS := "from-host-sync-test-2"
				virtualNS := "barfoo2"
				secretName := "dummy"

				By("creating from-host-sync-test-2 namespace on host", func() {
					ensureNamespace(ctx, hostNS)
				})

				By("creating secret (dummy) in from-host-sync-test-2 namespace on host", func() {
					_, err := hostClient.CoreV1().Secrets(hostNS).Create(ctx, &corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      secretName,
							Namespace: hostNS,
						},
						Data: map[string][]byte{
							"BOO_BAR":     []byte("hello-world"),
							"ANOTHER_ENV": []byte("another-hello-world"),
						},
					}, metav1.CreateOptions{})
					Expect(err).To(Succeed())
					DeferCleanup(func(ctx context.Context) {
						err := hostClient.CoreV1().Secrets(hostNS).Delete(ctx, secretName, metav1.DeleteOptions{})
						if !kerrors.IsNotFound(err) {
							Expect(err).To(Succeed())
						}
					})
				})

				By("waiting for secret to be synced to barfoo2 namespace in vcluster", func() {
					Eventually(func(g Gomega) {
						secret, err := vClusterClient.CoreV1().Secrets(virtualNS).Get(ctx, secretName, metav1.GetOptions{})
						g.Expect(err).To(Succeed())
						g.Expect(secret.Data).To(Equal(map[string][]byte{
							"BOO_BAR":     []byte("hello-world"),
							"ANOTHER_ENV": []byte("another-hello-world"),
						}))
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
				})

				By("updating the host secret with new data, labels, and annotations", func() {
					freshHostSecret, err := hostClient.CoreV1().Secrets(hostNS).Get(ctx, secretName, metav1.GetOptions{})
					Expect(err).To(Succeed())

					freshHostSecret.Data["UPDATED_ENV"] = []byte("one")
					if freshHostSecret.Labels == nil {
						freshHostSecret.Labels = make(map[string]string, 1)
					}
					freshHostSecret.Labels["updated-label"] = "updated-value"
					if freshHostSecret.Annotations == nil {
						freshHostSecret.Annotations = make(map[string]string, 1)
					}
					freshHostSecret.Annotations["updated-annotation"] = "updated-value"
					_, err = hostClient.CoreV1().Secrets(hostNS).Update(ctx, freshHostSecret, metav1.UpdateOptions{})
					Expect(err).To(Succeed())
				})

				By("waiting for the update to propagate to vcluster", func() {
					Eventually(func(g Gomega) {
						updatedSecret, err := vClusterClient.CoreV1().Secrets(virtualNS).Get(ctx, secretName, metav1.GetOptions{})
						g.Expect(err).To(Succeed())
						g.Expect(string(updatedSecret.Data["UPDATED_ENV"])).To(Equal("one"))
						g.Expect(updatedSecret.Labels["updated-label"]).To(Equal("updated-value"))
						g.Expect(updatedSecret.Annotations["updated-annotation"]).To(Equal("updated-value"))
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
				})
			})

			It("syncs exact-name secret from default namespace to vcluster", func(ctx context.Context) {
				// default/my-secret -> barfoo2/secret-my is a fixed exact-name mapping in the vcluster config.
				hostNS := "default"
				hostSecretName := "my-secret"
				virtualNS := "barfoo2"
				virtualSecretName := "secret-my"

				By("creating my-secret in default namespace on host", func() {
					_, err := hostClient.CoreV1().Secrets(hostNS).Create(ctx, &corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      hostSecretName,
							Namespace: hostNS,
						},
						Data: map[string][]byte{
							"ENV_FROM_DEFAULT_NS":         []byte("one"),
							"ANOTHER_ENV_FROM_DEFAULT_NS": []byte("two"),
						},
					}, metav1.CreateOptions{})
					Expect(err).To(Succeed())
					DeferCleanup(func(ctx context.Context) {
						err := hostClient.CoreV1().Secrets(hostNS).Delete(ctx, hostSecretName, metav1.DeleteOptions{})
						if !kerrors.IsNotFound(err) {
							Expect(err).To(Succeed())
						}
					})
				})

				By("waiting for secret to be synced as secret-my to barfoo2 namespace in vcluster", func() {
					Eventually(func(g Gomega) {
						secret, err := vClusterClient.CoreV1().Secrets(virtualNS).Get(ctx, virtualSecretName, metav1.GetOptions{})
						g.Expect(err).To(Succeed())
						g.Expect(secret.Data).To(Equal(map[string][]byte{
							"ENV_FROM_DEFAULT_NS":         []byte("one"),
							"ANOTHER_ENV_FROM_DEFAULT_NS": []byte("two"),
						}))
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
				})
			})

			It("syncs secret from vcluster host namespace to configured virtual namespace", func(ctx context.Context) {
				// /my-secret -> barfoo2/secret-from-default-ns is a fixed mapping
				// sourced from the vcluster host namespace in the vcluster config.
				secretName := "my-secret-in-default-ns"
				virtualNS := "barfoo2"
				virtualSecretName := "secret-from-default-ns"

				By("creating my-secret-in-default-ns in vcluster host namespace on host", func() {
					_, err := hostClient.CoreV1().Secrets(vClusterHostNS).Create(ctx, &corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      secretName,
							Namespace: vClusterHostNS,
						},
						Data: map[string][]byte{
							"dummy":   []byte("one"),
							"dummy-2": []byte("two"),
						},
					}, metav1.CreateOptions{})
					Expect(err).To(Succeed())
					DeferCleanup(func(ctx context.Context) {
						err := hostClient.CoreV1().Secrets(vClusterHostNS).Delete(ctx, secretName, metav1.DeleteOptions{})
						if !kerrors.IsNotFound(err) {
							Expect(err).To(Succeed())
						}
					})
				})

				By("waiting for secret to be synced as secret-from-default-ns to barfoo2 namespace in vcluster", func() {
					Eventually(func(g Gomega) {
						secret, err := vClusterClient.CoreV1().Secrets(virtualNS).Get(ctx, virtualSecretName, metav1.GetOptions{})
						g.Expect(err).To(Succeed())
						g.Expect(secret.Data).To(Equal(map[string][]byte{
							"dummy":   []byte("one"),
							"dummy-2": []byte("two"),
						}))
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
				})
			})

			It("deleting secret on host also deletes it from vcluster", func(ctx context.Context) {
				// from-host-sync-test-2/* maps to barfoo2/* in the vcluster config.
				hostNS := "from-host-sync-test-2"
				virtualNS := "barfoo2"
				secretName := "deletion-test-secret"

				By("creating from-host-sync-test-2 namespace on host", func() {
					ensureNamespace(ctx, hostNS)
				})

				By("creating the deletion-test-secret in from-host-sync-test-2 on host", func() {
					_, err := hostClient.CoreV1().Secrets(hostNS).Create(ctx, &corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      secretName,
							Namespace: hostNS,
						},
						Data: map[string][]byte{"key": []byte("value")},
					}, metav1.CreateOptions{})
					Expect(err).To(Succeed())
				})

				By("waiting for secret to be synced to barfoo2 namespace in vcluster", func() {
					Eventually(func(g Gomega) {
						_, err := vClusterClient.CoreV1().Secrets(virtualNS).Get(ctx, secretName, metav1.GetOptions{})
						g.Expect(err).To(Succeed())
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
				})

				By("deleting the secret from host", func() {
					Expect(hostClient.CoreV1().Secrets(hostNS).Delete(ctx, secretName, metav1.DeleteOptions{})).To(Succeed())
				})

				By("waiting for the secret to be deleted from vcluster", func() {
					Eventually(func(g Gomega) {
						_, err := vClusterClient.CoreV1().Secrets(virtualNS).Get(ctx, secretName, metav1.GetOptions{})
						g.Expect(kerrors.IsNotFound(err)).To(BeTrue())
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
				})
			})

			It("synced secrets from multiple sources can be used as env source for a pod in vcluster", func(ctx context.Context) {
				// Tests that secrets synced from different host namespaces via fromHost.secrets
				// can all be used simultaneously as envFrom sources in a pod running in the vcluster.
				// Uses from-host-sync-test-3 (separate from the wildcard test) to avoid
				// namespace collision when specs run in parallel.
				hostNS := "from-host-sync-test-3"
				virtualNS := "barfoo2"
				secret1Name := "pod-env-test-secret-1"
				secret2Name := "pod-env-test-secret-2"
				podName := "my-pod"

				By("creating from-host-sync-test-3 namespace on host", func() {
					ensureNamespace(ctx, hostNS)
				})

				By("creating first secret with env data on host", func() {
					_, err := hostClient.CoreV1().Secrets(hostNS).Create(ctx, &corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      secret1Name,
							Namespace: hostNS,
						},
						Data: map[string][]byte{
							"BOO_BAR":     []byte("hello-world"),
							"ANOTHER_ENV": []byte("another-hello-world"),
							"UPDATED_ENV": []byte("one"),
						},
					}, metav1.CreateOptions{})
					Expect(err).To(Succeed())
					DeferCleanup(func(ctx context.Context) {
						err := hostClient.CoreV1().Secrets(hostNS).Delete(ctx, secret1Name, metav1.DeleteOptions{})
						if !kerrors.IsNotFound(err) {
							Expect(err).To(Succeed())
						}
					})
				})

				By("creating second secret with additional env data on host", func() {
					_, err := hostClient.CoreV1().Secrets(hostNS).Create(ctx, &corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      secret2Name,
							Namespace: hostNS,
						},
						Data: map[string][]byte{
							"ENV_FROM_DEFAULT_NS":         []byte("one"),
							"ANOTHER_ENV_FROM_DEFAULT_NS": []byte("two"),
						},
					}, metav1.CreateOptions{})
					Expect(err).To(Succeed())
					DeferCleanup(func(ctx context.Context) {
						err := hostClient.CoreV1().Secrets(hostNS).Delete(ctx, secret2Name, metav1.DeleteOptions{})
						if !kerrors.IsNotFound(err) {
							Expect(err).To(Succeed())
						}
					})
				})

				By("waiting for both secrets to be synced to barfoo2 namespace in vcluster", func() {
					Eventually(func(g Gomega) {
						_, err := vClusterClient.CoreV1().Secrets(virtualNS).Get(ctx, secret1Name, metav1.GetOptions{})
						g.Expect(err).To(Succeed())
						_, err = vClusterClient.CoreV1().Secrets(virtualNS).Get(ctx, secret2Name, metav1.GetOptions{})
						g.Expect(err).To(Succeed())
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
				})

				By("waiting for the default service account to be ready in barfoo2 namespace", func() {
					Eventually(func(g Gomega) {
						_, err := vClusterClient.CoreV1().ServiceAccounts(virtualNS).Get(ctx, "default", metav1.GetOptions{})
						g.Expect(err).To(Succeed())
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
				})

				optional := false
				pod := &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      podName,
						Namespace: virtualNS,
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:            "default",
								Image:           "nginxinc/nginx-unprivileged:stable-alpine3.20-slim",
								ImagePullPolicy: corev1.PullIfNotPresent,
								SecurityContext: &corev1.SecurityContext{
									RunAsUser: ptr.To(int64(12345)),
								},
								EnvFrom: []corev1.EnvFromSource{
									{
										SecretRef: &corev1.SecretEnvSource{
											LocalObjectReference: corev1.LocalObjectReference{Name: secret1Name},
											Optional:             &optional,
										},
									},
									{
										SecretRef: &corev1.SecretEnvSource{
											LocalObjectReference: corev1.LocalObjectReference{Name: secret2Name},
											Optional:             &optional,
										},
									},
								},
							},
						},
					},
				}

				By("creating the pod in vcluster", func() {
					_, err := vClusterClient.CoreV1().Pods(virtualNS).Create(ctx, pod, metav1.CreateOptions{})
					Expect(err).To(Succeed())
					DeferCleanup(func(ctx context.Context) {
						err := vClusterClient.CoreV1().Pods(virtualNS).Delete(ctx, podName, metav1.DeleteOptions{})
						if !kerrors.IsNotFound(err) {
							Expect(err).To(Succeed())
						}
					})
				})

				By("waiting for the pod to reach Running phase in vcluster", func() {
					Eventually(func(g Gomega) {
						vpod, err := vClusterClient.CoreV1().Pods(virtualNS).Get(ctx, podName, metav1.GetOptions{})
						g.Expect(err).To(Succeed())
						g.Expect(vpod.Status.Phase).To(Equal(corev1.PodRunning))
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
				})

				By("exec-ing printenv in the pod to verify env vars from both synced secrets", func() {
					currentClusterName := cluster.CurrentClusterNameFrom(ctx)
					vClusterRestConfig := cluster.From(ctx, currentClusterName).KubernetesRestConfig()
					stdout, _, err := podhelper.ExecBuffered(ctx, vClusterRestConfig, virtualNS, podName, "default", []string{"sh", "-c", "printenv"}, nil)
					Expect(err).To(Succeed())

					output := string(bytes.TrimSpace(stdout))
					envVars := strings.Split(strings.TrimSpace(output), "\n")
					envs := make(map[string]string, len(envVars))
					for _, envVar := range envVars {
						parts := strings.SplitN(envVar, "=", 2)
						if len(parts) == 2 {
							envs[parts[0]] = strings.ReplaceAll(parts[1], "\r", "")
						}
					}

					Expect(envs).To(HaveKeyWithValue("BOO_BAR", "hello-world"))
					Expect(envs).To(HaveKeyWithValue("ANOTHER_ENV", "another-hello-world"))
					Expect(envs).To(HaveKeyWithValue("UPDATED_ENV", "one"))
					Expect(envs).To(HaveKeyWithValue("ENV_FROM_DEFAULT_NS", "one"))
					Expect(envs).To(HaveKeyWithValue("ANOTHER_ENV_FROM_DEFAULT_NS", "two"))
				})
			})
		})
}
