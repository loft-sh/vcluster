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

// DescribeFromHostConfigMaps registers configmap sync from host tests against the given vCluster.
func DescribeFromHostConfigMaps(vcluster suite.Dependency) bool {
	return Describe("ConfigMaps sync from host",
		labels.Core,
		labels.PR,
		labels.Sync,
		labels.ConfigMaps,
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
			// DeferCleanup for the namespace itself – callers that own the namespace must
			// do so themselves. For shared/fixed namespaces leave deletion to vcluster teardown.
			ensureNamespace := func(ctx context.Context, name string) {
				GinkgoHelper()
				_, err := hostClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: name},
				}, metav1.CreateOptions{})
				if !kerrors.IsAlreadyExists(err) {
					Expect(err).NotTo(HaveOccurred())
				}
			}

			It("syncs configmaps from wildcard namespace and propagates updates", func(ctx context.Context) {
				// from-host-sync-test/* maps to barfoo/* in the vcluster config.
				hostNS := "from-host-sync-test"
				virtualNS := "barfoo"
				cmName := "dummy"

				// Ensure the host namespace exists. It is a fixed name in the vcluster config
				// so multiple specs may reuse it; we create it idempotently and leave deletion
				// to vcluster teardown.
				ensureNamespace(ctx, hostNS)

				By("creating CM (dummy) in from-host-sync-test namespace on host", func() {
					_, err := hostClient.CoreV1().ConfigMaps(hostNS).Create(ctx, &corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      cmName,
							Namespace: hostNS,
						},
						Data: map[string]string{
							"BOO_BAR":     "hello-world",
							"ANOTHER_ENV": "another-hello-world",
						},
					}, metav1.CreateOptions{})
					Expect(err).NotTo(HaveOccurred())
					DeferCleanup(func(ctx context.Context) {
						err := hostClient.CoreV1().ConfigMaps(hostNS).Delete(ctx, cmName, metav1.DeleteOptions{})
						if !kerrors.IsNotFound(err) {
							Expect(err).NotTo(HaveOccurred())
						}
					})
				})

				By("waiting for CM to be synced to barfoo namespace in vcluster", func() {
					Eventually(func(g Gomega) {
						cm, err := vClusterClient.CoreV1().ConfigMaps(virtualNS).Get(ctx, cmName, metav1.GetOptions{})
						g.Expect(err).NotTo(HaveOccurred())
						g.Expect(cm.Data).To(Equal(map[string]string{
							"BOO_BAR":     "hello-world",
							"ANOTHER_ENV": "another-hello-world",
						}))
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
				})

				By("updating the host CM with new data, labels, and annotations", func() {
					freshHostCM, err := hostClient.CoreV1().ConfigMaps(hostNS).Get(ctx, cmName, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())

					freshHostCM.Data["UPDATED_ENV"] = "one"
					if freshHostCM.Labels == nil {
						freshHostCM.Labels = make(map[string]string, 1)
					}
					freshHostCM.Labels["updated-label"] = "updated-value"
					if freshHostCM.Annotations == nil {
						freshHostCM.Annotations = make(map[string]string, 1)
					}
					freshHostCM.Annotations["updated-annotation"] = "updated-value"
					_, err = hostClient.CoreV1().ConfigMaps(hostNS).Update(ctx, freshHostCM, metav1.UpdateOptions{})
					Expect(err).NotTo(HaveOccurred())
				})

				By("waiting for the update to propagate to vcluster", func() {
					Eventually(func(g Gomega) {
						updatedCM, err := vClusterClient.CoreV1().ConfigMaps(virtualNS).Get(ctx, cmName, metav1.GetOptions{})
						g.Expect(err).NotTo(HaveOccurred())
						g.Expect(updatedCM.Data["UPDATED_ENV"]).To(Equal("one"))
						g.Expect(updatedCM.Labels["updated-label"]).To(Equal("updated-value"))
						g.Expect(updatedCM.Annotations["updated-annotation"]).To(Equal("updated-value"))
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
				})
			})

			It("syncs exact-name configmap and restores vcluster mutations from host", func(ctx context.Context) {
				// default/my.cm -> barfoo/cm-my is a fixed exact-name mapping in the vcluster config.
				hostNS := "default"
				hostCMName := "my.cm"
				virtualNS := "barfoo"
				virtualCMName := "cm-my"

				By("creating my.cm in default namespace on host", func() {
					_, err := hostClient.CoreV1().ConfigMaps(hostNS).Create(ctx, &corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      hostCMName,
							Namespace: hostNS,
						},
						Data: map[string]string{
							"ENV_FROM_DEFAULT_NS":         "one",
							"ANOTHER_ENV_FROM_DEFAULT_NS": "two",
						},
					}, metav1.CreateOptions{})
					Expect(err).NotTo(HaveOccurred())
					DeferCleanup(func(ctx context.Context) {
						err := hostClient.CoreV1().ConfigMaps(hostNS).Delete(ctx, hostCMName, metav1.DeleteOptions{})
						if !kerrors.IsNotFound(err) {
							Expect(err).NotTo(HaveOccurred())
						}
					})
				})

				By("waiting for CM to be synced as cm-my to barfoo namespace in vcluster", func() {
					Eventually(func(g Gomega) {
						cm, err := vClusterClient.CoreV1().ConfigMaps(virtualNS).Get(ctx, virtualCMName, metav1.GetOptions{})
						g.Expect(err).NotTo(HaveOccurred())
						g.Expect(cm.Data).To(Equal(map[string]string{
							"ENV_FROM_DEFAULT_NS":         "one",
							"ANOTHER_ENV_FROM_DEFAULT_NS": "two",
						}))
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
				})

				var uidBeforeDeletion string
				By("recording the UID of cm-my in vcluster before deletion", func() {
					oldCM, err := vClusterClient.CoreV1().ConfigMaps(virtualNS).Get(ctx, virtualCMName, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					uidBeforeDeletion = string(oldCM.UID)
				})

				By("deleting cm-my from vcluster; it should be re-synced from host", func() {
					Expect(vClusterClient.CoreV1().ConfigMaps(virtualNS).Delete(ctx, virtualCMName, metav1.DeleteOptions{})).To(Succeed())

					Eventually(func(g Gomega) {
						newCM, err := vClusterClient.CoreV1().ConfigMaps(virtualNS).Get(ctx, virtualCMName, metav1.GetOptions{})
						g.Expect(err).NotTo(HaveOccurred())
						g.Expect(newCM.Data["ENV_FROM_DEFAULT_NS"]).To(Equal("one"))
						g.Expect(newCM.Data["ANOTHER_ENV_FROM_DEFAULT_NS"]).To(Equal("two"))
						g.Expect(string(newCM.UID)).NotTo(Equal(uidBeforeDeletion))
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
				})

				By("overwriting cm-my data in vcluster; host should restore it", func() {
					vClusterMap, err := vClusterClient.CoreV1().ConfigMaps(virtualNS).Get(ctx, virtualCMName, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					vClusterMap.Data = map[string]string{"new-value": "will-be-overwritten"}
					_, err = vClusterClient.CoreV1().ConfigMaps(virtualNS).Update(ctx, vClusterMap, metav1.UpdateOptions{})
					Expect(err).NotTo(HaveOccurred())

					Eventually(func(g Gomega) {
						cm, err := vClusterClient.CoreV1().ConfigMaps(virtualNS).Get(ctx, virtualCMName, metav1.GetOptions{})
						g.Expect(err).NotTo(HaveOccurred())
						g.Expect(cm.Data["ENV_FROM_DEFAULT_NS"]).To(Equal("one"))
						g.Expect(cm.Data["ANOTHER_ENV_FROM_DEFAULT_NS"]).To(Equal("two"))
						_, updatedKeyExists := cm.Data["new-value"]
						g.Expect(updatedKeyExists).To(BeFalse())
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
				})
			})

			It("synced configmaps can be used as env source for a pod in vcluster", func(ctx context.Context) {
				hostNS := "from-host-sync-test"
				virtualNS := "barfoo"
				cmName := "pod-env-test-cm"
				podName := "my-pod"

				ensureNamespace(ctx, hostNS)

				By("creating pod-env-test-cm with all env data on host", func() {
					_, err := hostClient.CoreV1().ConfigMaps(hostNS).Create(ctx, &corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      cmName,
							Namespace: hostNS,
						},
						Data: map[string]string{
							"BOO_BAR":     "hello-world",
							"ANOTHER_ENV": "another-hello-world",
							"UPDATED_ENV": "one",
						},
					}, metav1.CreateOptions{})
					Expect(err).NotTo(HaveOccurred())
					DeferCleanup(func(ctx context.Context) {
						err := hostClient.CoreV1().ConfigMaps(hostNS).Delete(ctx, cmName, metav1.DeleteOptions{})
						if !kerrors.IsNotFound(err) {
							Expect(err).NotTo(HaveOccurred())
						}
					})
				})

				By("waiting for CM to be synced to barfoo namespace in vcluster", func() {
					Eventually(func(g Gomega) {
						_, err := vClusterClient.CoreV1().ConfigMaps(virtualNS).Get(ctx, cmName, metav1.GetOptions{})
						g.Expect(err).NotTo(HaveOccurred())
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
				})

				By("waiting for the default service account to be ready in barfoo namespace", func() {
					Eventually(func(g Gomega) {
						_, err := vClusterClient.CoreV1().ServiceAccounts(virtualNS).Get(ctx, "default", metav1.GetOptions{})
						g.Expect(err).NotTo(HaveOccurred())
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
										ConfigMapRef: &corev1.ConfigMapEnvSource{
											LocalObjectReference: corev1.LocalObjectReference{Name: cmName},
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
					Expect(err).NotTo(HaveOccurred())
					DeferCleanup(func(ctx context.Context) {
						err := vClusterClient.CoreV1().Pods(virtualNS).Delete(ctx, podName, metav1.DeleteOptions{})
						if !kerrors.IsNotFound(err) {
							Expect(err).NotTo(HaveOccurred())
						}
					})
				})

				By("waiting for the pod to reach Running phase in vcluster", func() {
					Eventually(func(g Gomega) {
						vpod, err := vClusterClient.CoreV1().Pods(virtualNS).Get(ctx, podName, metav1.GetOptions{})
						g.Expect(err).NotTo(HaveOccurred())
						g.Expect(vpod.Status.Phase).To(Equal(corev1.PodRunning))
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
				})

				By("exec-ing printenv in the pod to verify env vars from synced configmap", func() {
					currentClusterName := cluster.CurrentClusterNameFrom(ctx)
					vClusterRestConfig := cluster.From(ctx, currentClusterName).KubernetesRestConfig()
					stdout, _, err := podhelper.ExecBuffered(ctx, vClusterRestConfig, virtualNS, podName, "default", []string{"sh", "-c", "printenv"}, nil)
					Expect(err).NotTo(HaveOccurred())

					output := string(bytes.TrimSpace(stdout))
					envVars := strings.Split(strings.TrimSpace(output), "\n")
					envs := make(map[string]string, len(envVars))
					for _, envVar := range envVars {
						parts := strings.SplitN(envVar, "=", 2)
						if len(parts) == 2 {
							envs[parts[0]] = strings.ReplaceAll(parts[1], "\r", "")
						}
					}

					Expect(envs).To(HaveKeyWithValue("UPDATED_ENV", "one"))
					Expect(envs).To(HaveKeyWithValue("ANOTHER_ENV", "another-hello-world"))
					Expect(envs).To(HaveKeyWithValue("BOO_BAR", "hello-world"))
				})
			})

			It("syncs configmaps from vcluster host namespace to configured virtual namespaces", func(ctx context.Context) {
				// Tests three mappings that all source from the vcluster host namespace:
				//   ""          -> "my-new-ns"                          (all CMs from host ns)
				//   "/my-cm-4"  -> "barfoo/my-cm-4"                    (exact name, to barfoo)
				//   "/specific-cm" -> "my-virtual-namespace/specific-cm" (exact name, to different ns)
				type cmFixture struct {
					name      string
					virtualNS string
					data      map[string]string
				}
				fixtures := []cmFixture{
					{
						name:      "from-vcluster-ns",
						virtualNS: "my-new-ns",
						data:      map[string]string{"VAL1": "abcdef", "VAL2": "defgh"},
					},
					{
						name:      "my-cm-4",
						virtualNS: "barfoo",
						data:      map[string]string{"VAL3": "ghijkl", "VAL4": "defg"},
					},
					{
						name:      "specific-cm",
						virtualNS: "my-virtual-namespace",
						data:      map[string]string{"key5": "value5", "key6": "value6"},
					},
				}

				for _, f := range fixtures {
					By("creating "+f.name+" in vcluster host namespace on host", func() {
						_, err := hostClient.CoreV1().ConfigMaps(vClusterHostNS).Create(ctx, &corev1.ConfigMap{
							ObjectMeta: metav1.ObjectMeta{
								Name:      f.name,
								Namespace: vClusterHostNS,
							},
							Data: f.data,
						}, metav1.CreateOptions{})
						Expect(err).NotTo(HaveOccurred())
						name := f.name // capture for closure
						DeferCleanup(func(ctx context.Context) {
							err := hostClient.CoreV1().ConfigMaps(vClusterHostNS).Delete(ctx, name, metav1.DeleteOptions{})
							if !kerrors.IsNotFound(err) {
								Expect(err).NotTo(HaveOccurred())
							}
						})
					})
				}

				for _, f := range fixtures {
					By("waiting for "+f.name+" to appear in "+f.virtualNS+" in vcluster", func() {
						expectedData := f.data
						virtualNS := f.virtualNS
						cmName := f.name
						Eventually(func(g Gomega) {
							cm, err := vClusterClient.CoreV1().ConfigMaps(virtualNS).Get(ctx, cmName, metav1.GetOptions{})
							g.Expect(err).NotTo(HaveOccurred())
							g.Expect(cm.Data).To(Equal(expectedData))
						}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
					})
				}
			})

			It("deleting configmap on host also deletes it from vcluster", func(ctx context.Context) {
				// my-ns/to-be-deleted -> my-ns/to-be-deleted is a fixed mapping in the vcluster config.
				hostNS := "my-ns"
				cmName := "to-be-deleted"

				By("creating my-ns namespace on host", func() {
					_, err := hostClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{Name: hostNS},
					}, metav1.CreateOptions{})
					Expect(err).NotTo(HaveOccurred())
					DeferCleanup(func(ctx context.Context) {
						err := hostClient.CoreV1().Namespaces().Delete(ctx, hostNS, metav1.DeleteOptions{})
						if !kerrors.IsNotFound(err) {
							Expect(err).NotTo(HaveOccurred())
						}
						Eventually(func(g Gomega) {
							_, err := hostClient.CoreV1().Namespaces().Get(ctx, hostNS, metav1.GetOptions{})
							g.Expect(kerrors.IsNotFound(err)).To(BeTrue())
						}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
					})
				})

				By("creating the to-be-deleted CM in my-ns on host", func() {
					_, err := hostClient.CoreV1().ConfigMaps(hostNS).Create(ctx, &corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      cmName,
							Namespace: hostNS,
						},
						Data: map[string]string{"key6": "value6"},
					}, metav1.CreateOptions{})
					Expect(err).NotTo(HaveOccurred())
				})

				By("waiting for the CM to appear in vcluster", func() {
					Eventually(func(g Gomega) {
						cm, err := vClusterClient.CoreV1().ConfigMaps(hostNS).Get(ctx, cmName, metav1.GetOptions{})
						g.Expect(err).NotTo(HaveOccurred())
						g.Expect(cm.Data["key6"]).To(Equal("value6"))
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
				})

				By("deleting the CM from host", func() {
					Expect(hostClient.CoreV1().ConfigMaps(hostNS).Delete(ctx, cmName, metav1.DeleteOptions{})).To(Succeed())
				})

				By("waiting for the CM to be deleted from vcluster", func() {
					Eventually(func(g Gomega) {
						_, err := vClusterClient.CoreV1().ConfigMaps(hostNS).Get(ctx, cmName, metav1.GetOptions{})
						g.Expect(kerrors.IsNotFound(err)).To(BeTrue())
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
				})
			})

			It("syncs all configmaps from host namespace to the same namespace in vcluster", func(ctx context.Context) {
				// same-ns/* -> same-ns/* is a wildcard mapping in the vcluster config.
				ns := "same-ns"
				cmName := "cm-same-ns"

				By("creating same-ns namespace on host", func() {
					_, err := hostClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{Name: ns},
					}, metav1.CreateOptions{})
					Expect(err).NotTo(HaveOccurred())
					DeferCleanup(func(ctx context.Context) {
						err := hostClient.CoreV1().Namespaces().Delete(ctx, ns, metav1.DeleteOptions{})
						if !kerrors.IsNotFound(err) {
							Expect(err).NotTo(HaveOccurred())
						}
						Eventually(func(g Gomega) {
							_, err := hostClient.CoreV1().Namespaces().Get(ctx, ns, metav1.GetOptions{})
							g.Expect(kerrors.IsNotFound(err)).To(BeTrue())
						}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
					})
				})

				By("creating cm-same-ns CM in same-ns namespace on host", func() {
					_, err := hostClient.CoreV1().ConfigMaps(ns).Create(ctx, &corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      cmName,
							Namespace: ns,
						},
						Data: map[string]string{"key7": "value7"},
					}, metav1.CreateOptions{})
					Expect(err).NotTo(HaveOccurred())
					DeferCleanup(func(ctx context.Context) {
						err := hostClient.CoreV1().ConfigMaps(ns).Delete(ctx, cmName, metav1.DeleteOptions{})
						if !kerrors.IsNotFound(err) {
							Expect(err).NotTo(HaveOccurred())
						}
					})
				})

				By("waiting for cm-same-ns to appear in same-ns namespace in vcluster", func() {
					Eventually(func(g Gomega) {
						cm, err := vClusterClient.CoreV1().ConfigMaps(ns).Get(ctx, cmName, metav1.GetOptions{})
						g.Expect(err).NotTo(HaveOccurred())
						g.Expect(cm.Data["key7"]).To(Equal("value7"))
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
				})
			})
		})
}
