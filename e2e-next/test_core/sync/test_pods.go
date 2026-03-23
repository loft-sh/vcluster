package test_core

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/e2e-framework/pkg/setup/suite"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	podtranslate "github.com/loft-sh/vcluster/pkg/controllers/resources/pods/token"
	"github.com/loft-sh/vcluster/pkg/util/podhelper"
	"github.com/loft-sh/vcluster/pkg/util/random"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"
)

const (
	testingContainerName  = "nginx"
	testingContainerImage = "nginxinc/nginx-unprivileged:stable-alpine3.20-slim"
	ipRegExp              = "(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5]).){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])"
	initialNsLabelKey     = "testing-ns-label"
	initialNsLabelValue   = "testing-ns-label-value"
)

// DescribePodSync registers pod sync tests against the given vCluster.
func DescribePodSync(vcluster suite.Dependency) bool {
	return Describe("Pod sync from vCluster to host",
		labels.Core,
		labels.Sync,
		labels.Pods,
		cluster.Use(vcluster),
		cluster.Use(clusters.HostCluster),
		func() {
			var (
				hostClient     kubernetes.Interface
				vClusterClient kubernetes.Interface
				hostConfig     *rest.Config
				vClusterName   string
			)

			BeforeEach(func(ctx context.Context) {
				hostClient = cluster.KubeClientFrom(ctx, constants.GetHostClusterName())
				Expect(hostClient).NotTo(BeNil())
				vClusterClient = cluster.CurrentKubeClientFrom(ctx)
				Expect(vClusterClient).NotTo(BeNil())
				hostConfig = cluster.From(ctx, constants.GetHostClusterName()).KubernetesRestConfig()
				Expect(hostConfig).NotTo(BeNil())
				vClusterName = cluster.CurrentClusterNameFrom(ctx)
			})

			// createTestNamespace creates a namespace with the standard test label and registers cleanup.
			createTestNamespace := func(ctx context.Context, nsName string) {
				GinkgoHelper()
				_, err := vClusterClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name:   nsName,
						Labels: map[string]string{initialNsLabelKey: initialNsLabelValue},
					},
				}, metav1.CreateOptions{})
				Expect(err).To(Succeed())
				DeferCleanup(func(ctx context.Context) {
					err := vClusterClient.CoreV1().Namespaces().Delete(ctx, nsName, metav1.DeleteOptions{})
					if !kerrors.IsNotFound(err) {
						Expect(err).To(Succeed())
					}
				})
			}

			// waitPodRunning waits for a pod to reach Running phase.
			waitPodRunning := func(ctx context.Context, podName, ns string) {
				GinkgoHelper()
				Eventually(func(g Gomega) {
					pod, err := vClusterClient.CoreV1().Pods(ns).Get(ctx, podName, metav1.GetOptions{})
					g.Expect(err).NotTo(HaveOccurred(), "failed to get pod %s/%s", ns, podName)
					g.Expect(pod.Status.Phase).To(Equal(corev1.PodRunning),
						"pod %s/%s phase is %s, not yet Running", ns, podName, pod.Status.Phase)
				}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
			}

			defaultSecurityContext := func() *corev1.SecurityContext {
				return &corev1.SecurityContext{
					Capabilities: &corev1.Capabilities{
						Drop: []corev1.Capability{"ALL"},
					},
					RunAsNonRoot:             boolPtr(true),
					RunAsUser:                int64Ptr(12345),
					AllowPrivilegeEscalation: boolPtr(false),
					SeccompProfile:           &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault},
				}
			}

			It("should start a pod and sync status back to vCluster", func(ctx context.Context) {
				suffix := random.String(6)
				ns := "pod-status-test-" + suffix
				createTestNamespace(ctx, ns)

				podName := "test-" + suffix
				By("Creating a pod in the vCluster", func() {
					_, err := vClusterClient.CoreV1().Pods(ns).Create(ctx, &corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{Name: podName},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:            testingContainerName,
									Image:           testingContainerImage,
									ImagePullPolicy: corev1.PullIfNotPresent,
									SecurityContext: defaultSecurityContext(),
								},
							},
						},
					}, metav1.CreateOptions{})
					Expect(err).To(Succeed())
				})

				By("Waiting for the pod to be running", func() {
					waitPodRunning(ctx, podName, ns)
				})

				By("Verifying pod status matches between vCluster and host", func() {
					vpod, err := vClusterClient.CoreV1().Pods(ns).Get(ctx, podName, metav1.GetOptions{})
					Expect(err).To(Succeed())
					pPodName := translate.SingleNamespaceHostName(podName, ns, vClusterName)
					hostNS := "vcluster-" + vClusterName
					pod, err := hostClient.CoreV1().Pods(hostNS).Get(ctx, pPodName, metav1.GetOptions{})
					Expect(err).To(Succeed())

					// Since k8s 1.32, status.QOSClass field has become immutable,
					// hence we have stopped syncing it.
					pod.Status.QOSClass = vpod.Status.QOSClass
					Expect(vpod.Status).To(Equal(pod.Status))
				})

				By("Verifying ephemeral containers work (k8s > 1.22)", func() {
					version, err := vClusterClient.Discovery().ServerVersion()
					Expect(err).To(Succeed())
					if version != nil {
						minor, err := strconv.Atoi(strings.ReplaceAll(version.Minor, "+", ""))
						Expect(err).To(Succeed())
						if minor > 22 {
							vpod, err := vClusterClient.CoreV1().Pods(ns).Get(ctx, podName, metav1.GetOptions{})
							Expect(err).To(Succeed())
							vpod.Spec.EphemeralContainers = []corev1.EphemeralContainer{{
								EphemeralContainerCommon: corev1.EphemeralContainerCommon{
									Name:  "busybox",
									Image: "busybox:1.28",
								},
							}}
							_, err = vClusterClient.CoreV1().Pods(ns).UpdateEphemeralContainers(ctx, vpod.Name, vpod, metav1.UpdateOptions{})
							Expect(err).To(Succeed())
							waitPodRunning(ctx, vpod.Name, vpod.Namespace)

							Eventually(func(g Gomega) {
								p, err := vClusterClient.CoreV1().Pods(ns).Get(ctx, vpod.Name, metav1.GetOptions{})
								g.Expect(err).To(Succeed())
								g.Expect(p.Status.EphemeralContainerStatuses).NotTo(BeEmpty(),
									"expected ephemeral container statuses to be present")
							}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
						}
					}
				})
			})

			It("should schedule a pod without explicit scheduler using default-scheduler", func(ctx context.Context) {
				suffix := random.String(6)
				ns := "pod-sched-impl-test-" + suffix
				createTestNamespace(ctx, ns)

				podName := "implicit-sched-" + suffix
				By("Creating a pod without explicit schedulerName", func() {
					_, err := vClusterClient.CoreV1().Pods(ns).Create(ctx, &corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{Name: podName},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:            testingContainerName,
									Image:           testingContainerImage,
									ImagePullPolicy: corev1.PullIfNotPresent,
									SecurityContext: defaultSecurityContext(),
								},
							},
						},
					}, metav1.CreateOptions{})
					Expect(err).To(Succeed())
				})

				waitPodRunning(ctx, podName, ns)

				By("Checking the scheduled event reports default-scheduler", func() {
					events, err := vClusterClient.CoreV1().Events(ns).List(ctx, metav1.ListOptions{
						FieldSelector: "reason==Scheduled,involvedObject.name==" + podName,
					})
					Expect(err).To(Succeed())
					Expect(events.Items).To(HaveLen(1))
					Expect(events.Items[0].ReportingController).To(Equal("default-scheduler"))
				})
			})

			It("should schedule a pod with explicit default-scheduler using default-scheduler", func(ctx context.Context) {
				suffix := random.String(6)
				ns := "pod-sched-expl-test-" + suffix
				createTestNamespace(ctx, ns)

				podName := "explicit-sched-" + suffix
				By("Creating a pod with schedulerName=default-scheduler", func() {
					_, err := vClusterClient.CoreV1().Pods(ns).Create(ctx, &corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{Name: podName},
						Spec: corev1.PodSpec{
							SchedulerName: "default-scheduler",
							Containers: []corev1.Container{
								{
									Name:            testingContainerName,
									Image:           testingContainerImage,
									ImagePullPolicy: corev1.PullIfNotPresent,
									SecurityContext: defaultSecurityContext(),
								},
							},
						},
					}, metav1.CreateOptions{})
					Expect(err).To(Succeed())
				})

				waitPodRunning(ctx, podName, ns)

				By("Checking the scheduled event reports default-scheduler", func() {
					events, err := vClusterClient.CoreV1().Events(ns).List(ctx, metav1.ListOptions{
						FieldSelector: "reason==Scheduled,involvedObject.name==" + podName,
					})
					Expect(err).To(Succeed())
					Expect(events.Items).To(HaveLen(1))
					Expect(events.Items[0].ReportingController).To(Equal("default-scheduler"))
				})
			})

			It("should sync readiness conditions back to the vCluster pod", func(ctx context.Context) {
				suffix := random.String(6)
				ns := "pod-readiness-test-" + suffix
				createTestNamespace(ctx, ns)

				podName := "readiness-" + suffix
				By("Creating a pod with a readiness gate", func() {
					_, err := vClusterClient.CoreV1().Pods(ns).Create(ctx, &corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{Name: podName},
						Spec: corev1.PodSpec{
							ReadinessGates: []corev1.PodReadinessGate{
								{ConditionType: "www.example.com/gate-1"},
							},
							Containers: []corev1.Container{
								{
									Name:            testingContainerName,
									Image:           testingContainerImage,
									ImagePullPolicy: corev1.PullIfNotPresent,
									SecurityContext: defaultSecurityContext(),
								},
							},
						},
					}, metav1.CreateOptions{})
					Expect(err).To(Succeed())
				})

				waitPodRunning(ctx, podName, ns)

				By("Verifying status matches between vCluster and host", func() {
					vpod, err := vClusterClient.CoreV1().Pods(ns).Get(ctx, podName, metav1.GetOptions{})
					Expect(err).To(Succeed())
					pPodName := translate.SingleNamespaceHostName(podName, ns, vClusterName)
					hostNS := "vcluster-" + vClusterName
					pod, err := hostClient.CoreV1().Pods(hostNS).Get(ctx, pPodName, metav1.GetOptions{})
					Expect(err).To(Succeed())
					pod.Status.QOSClass = vpod.Status.QOSClass
					Expect(vpod.Status).To(Equal(pod.Status))
				})

				By("Updating readiness conditions and verifying they sync to host", func() {
					vpod, err := vClusterClient.CoreV1().Pods(ns).Get(ctx, podName, metav1.GetOptions{})
					Expect(err).To(Succeed())
					vpod.Status.Conditions = append(vpod.Status.Conditions, corev1.PodCondition{
						Status: corev1.ConditionFalse,
						Type:   "www.example.com/gate-1",
					})
					_, err = vClusterClient.CoreV1().Pods(ns).UpdateStatus(ctx, vpod, metav1.UpdateOptions{})
					Expect(err).To(Succeed())

					waitPodRunning(ctx, podName, ns)

					pPodName := translate.SingleNamespaceHostName(podName, ns, vClusterName)
					hostNS := "vcluster-" + vClusterName
					Eventually(func(g Gomega) {
						pPod, err := hostClient.CoreV1().Pods(hostNS).Get(ctx, pPodName, metav1.GetOptions{})
						g.Expect(err).To(Succeed())
						hasCondition := false
						for _, c := range pPod.Status.Conditions {
							if c.Type == "www.example.com/gate-1" {
								hasCondition = true
								break
							}
						}
						g.Expect(hasCondition).To(BeTrue(), "readiness condition not synced to host pod")
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
				})
			})

			It("should start a pod with a non-default service account", func(ctx context.Context) {
				suffix := random.String(6)
				ns := "pod-sa-test-" + suffix
				createTestNamespace(ctx, ns)

				saName := "test-account-" + suffix
				podName := "pod-sa-" + suffix

				By("Creating a service account", func() {
					_, err := vClusterClient.CoreV1().ServiceAccounts(ns).Create(ctx, &corev1.ServiceAccount{
						ObjectMeta: metav1.ObjectMeta{Name: saName},
					}, metav1.CreateOptions{})
					Expect(err).To(Succeed())
				})

				By("Waiting for the service account to be available", func() {
					Eventually(func(g Gomega) {
						_, err := vClusterClient.CoreV1().ServiceAccounts(ns).Get(ctx, saName, metav1.GetOptions{})
						g.Expect(err).To(Succeed())
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
				})

				By("Creating a pod using the non-default service account", func() {
					_, err := vClusterClient.CoreV1().Pods(ns).Create(ctx, &corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{Name: podName},
						Spec: corev1.PodSpec{
							ServiceAccountName: saName,
							Containers: []corev1.Container{
								{
									Name:            testingContainerName,
									Image:           testingContainerImage,
									ImagePullPolicy: corev1.PullIfNotPresent,
									SecurityContext: defaultSecurityContext(),
								},
							},
						},
					}, metav1.CreateOptions{})
					Expect(err).To(Succeed())
				})

				waitPodRunning(ctx, podName, ns)

				By("Verifying the service account name is preserved", func() {
					vpod, err := vClusterClient.CoreV1().Pods(ns).Get(ctx, podName, metav1.GetOptions{})
					Expect(err).To(Succeed())
					Expect(vpod.Spec.ServiceAccountName).To(Equal(saName))
				})
			})

			It("should expose ConfigMap data as env vars and volume files in a pod", func(ctx context.Context) {
				suffix := random.String(6)
				ns := "pod-cm-test-" + suffix
				createTestNamespace(ctx, ns)

				podName := "pod-cm-" + suffix
				cmName := "test-configmap-" + suffix
				cmKey := "test-key"
				cmKeyValue := "test-value"
				envVarName := "TEST_ENVVAR"
				fileName := "test.file"
				filePath := "/test-path"

				By("Creating a ConfigMap", func() {
					_, err := vClusterClient.CoreV1().ConfigMaps(ns).Create(ctx, &corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{Name: cmName},
						Data:       map[string]string{cmKey: cmKeyValue},
					}, metav1.CreateOptions{})
					Expect(err).To(Succeed())
				})

				By("Creating a pod that references the ConfigMap", func() {
					_, err := vClusterClient.CoreV1().Pods(ns).Create(ctx, &corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{Name: podName},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:            testingContainerName,
									Image:           testingContainerImage,
									ImagePullPolicy: corev1.PullIfNotPresent,
									SecurityContext: defaultSecurityContext(),
									Env: []corev1.EnvVar{
										{
											Name: envVarName,
											ValueFrom: &corev1.EnvVarSource{
												ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
													LocalObjectReference: corev1.LocalObjectReference{Name: cmName},
													Key:                  cmKey,
												},
											},
										},
									},
									VolumeMounts: []corev1.VolumeMount{
										{Name: "volume-name", MountPath: filePath, ReadOnly: true},
									},
								},
							},
							Volumes: []corev1.Volume{
								{
									Name: "volume-name",
									VolumeSource: corev1.VolumeSource{
										ConfigMap: &corev1.ConfigMapVolumeSource{
											LocalObjectReference: corev1.LocalObjectReference{Name: cmName},
											Items:                []corev1.KeyToPath{{Key: cmKey, Path: fileName}},
										},
									},
								},
							},
						},
					}, metav1.CreateOptions{})
					Expect(err).To(Succeed())
				})

				waitPodRunning(ctx, podName, ns)

				pPodName := translate.SingleNamespaceHostName(podName, ns, vClusterName)
				hostNS := "vcluster-" + vClusterName

				By("Checking the env var value", func() {
					stdout, stderr, err := podhelper.ExecBuffered(ctx, hostConfig, hostNS, pPodName, testingContainerName, []string{"sh", "-c", "echo $" + envVarName}, nil)
					Expect(err).To(Succeed())
					Expect(string(stdout)).To(Equal(cmKeyValue + "\n"))
					Expect(string(stderr)).To(BeEmpty())
				})

				By("Checking the mounted file content", func() {
					stdout, stderr, err := podhelper.ExecBuffered(ctx, hostConfig, hostNS, pPodName, testingContainerName, []string{"cat", filePath + "/" + fileName}, nil)
					Expect(err).To(Succeed())
					Expect(string(stdout)).To(Equal(cmKeyValue))
					Expect(string(stderr)).To(BeEmpty())
				})
			})

			It("should expose Secret data as env vars and volume files in a pod", func(ctx context.Context) {
				suffix := random.String(6)
				ns := "pod-secret-test-" + suffix
				createTestNamespace(ctx, ns)

				podName := "pod-secret-" + suffix
				secretName := "test-secret-" + suffix
				secretKey := "test-key"
				secretKeyValue := "test-value"
				envVarName := "TEST_ENVVAR"
				fileName := "test.file"
				filePath := "/test-path"

				By("Creating a Secret", func() {
					_, err := vClusterClient.CoreV1().Secrets(ns).Create(ctx, &corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{Name: secretName},
						Data:       map[string][]byte{secretKey: []byte(secretKeyValue)},
					}, metav1.CreateOptions{})
					Expect(err).To(Succeed())
				})

				By("Creating a pod that references the Secret", func() {
					_, err := vClusterClient.CoreV1().Pods(ns).Create(ctx, &corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{Name: podName},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:            testingContainerName,
									Image:           testingContainerImage,
									ImagePullPolicy: corev1.PullIfNotPresent,
									SecurityContext: defaultSecurityContext(),
									Env: []corev1.EnvVar{
										{
											Name: envVarName,
											ValueFrom: &corev1.EnvVarSource{
												SecretKeyRef: &corev1.SecretKeySelector{
													LocalObjectReference: corev1.LocalObjectReference{Name: secretName},
													Key:                  secretKey,
												},
											},
										},
									},
									VolumeMounts: []corev1.VolumeMount{
										{Name: "volume-name", MountPath: filePath, ReadOnly: true},
									},
								},
							},
							Volumes: []corev1.Volume{
								{
									Name: "volume-name",
									VolumeSource: corev1.VolumeSource{
										Secret: &corev1.SecretVolumeSource{
											SecretName: secretName,
											Items:      []corev1.KeyToPath{{Key: secretKey, Path: fileName}},
										},
									},
								},
							},
						},
					}, metav1.CreateOptions{})
					Expect(err).To(Succeed())
				})

				waitPodRunning(ctx, podName, ns)

				pPodName := translate.SingleNamespaceHostName(podName, ns, vClusterName)
				hostNS := "vcluster-" + vClusterName

				By("Checking the env var value", func() {
					stdout, stderr, err := podhelper.ExecBuffered(ctx, hostConfig, hostNS, pPodName, testingContainerName, []string{"sh", "-c", "echo $" + envVarName}, nil)
					Expect(err).To(Succeed())
					Expect(string(stdout)).To(Equal(secretKeyValue + "\n"))
					Expect(string(stderr)).To(BeEmpty())
				})

				By("Checking the mounted file content", func() {
					stdout, stderr, err := podhelper.ExecBuffered(ctx, hostConfig, hostNS, pPodName, testingContainerName, []string{"cat", filePath + "/" + fileName}, nil)
					Expect(err).To(Succeed())
					Expect(string(stdout)).To(Equal(secretKeyValue))
					Expect(string(stderr)).To(BeEmpty())
				})
			})

			It("should resolve dependent environment variables correctly", func(ctx context.Context) {
				suffix := random.String(6)
				ns := "pod-depenv-test-" + suffix
				createTestNamespace(ctx, ns)

				svcName := "myservice"
				svcPort := 80
				myProtocol := "https"
				podName := "pod-depenv-" + suffix

				By("Creating a service for environment variable injection", func() {
					_, err := vClusterClient.CoreV1().Services(ns).Create(ctx, &corev1.Service{
						ObjectMeta: metav1.ObjectMeta{Name: svcName},
						Spec: corev1.ServiceSpec{
							Selector: map[string]string{"doesnt": "matter"},
							Ports:    []corev1.ServicePort{{Port: int32(svcPort)}},
						},
					}, metav1.CreateOptions{})
					Expect(err).To(Succeed())
				})

				// Wait for the service to be synced to the host with a ClusterIP assigned
				hostNSForSvc := "vcluster-" + vClusterName
				Eventually(func(g Gomega) {
					pSvcName := translate.SingleNamespaceHostName(svcName, ns, vClusterName)
					pSvc, err := hostClient.CoreV1().Services(hostNSForSvc).Get(ctx, pSvcName, metav1.GetOptions{})
					g.Expect(err).NotTo(HaveOccurred(), "host service not yet synced")
					g.Expect(pSvc.Spec.ClusterIP).NotTo(BeEmpty(), "host service ClusterIP not yet assigned")
				}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())

				By("Creating a pod with dependent env vars", func() {
					_, err := vClusterClient.CoreV1().Pods(ns).Create(ctx, &corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{Name: podName},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:            testingContainerName,
									Image:           testingContainerImage,
									ImagePullPolicy: corev1.PullIfNotPresent,
									SecurityContext: defaultSecurityContext(),
									Env: []corev1.EnvVar{
										{Name: "FIRST", Value: "Hello"},
										{Name: "SECOND", Value: "World"},
										{Name: "HELLO_WORLD", Value: "$(FIRST) $(SECOND)"},
										{Name: "ESCAPED_VAR", Value: "$$(FIRST)"},
										{Name: "MY_PROTOCOL", Value: myProtocol},
										{Name: "MY_SERVICE", Value: "$(MY_PROTOCOL)://$(" + strings.ToUpper(svcName) + "_SERVICE_HOST):$(" + strings.ToUpper(svcName) + "_SERVICE_PORT)"},
									},
								},
							},
						},
					}, metav1.CreateOptions{})
					Expect(err).To(Succeed())
				})

				waitPodRunning(ctx, podName, ns)

				pPodName := translate.SingleNamespaceHostName(podName, ns, vClusterName)
				hostNS := "vcluster-" + vClusterName

				By("Checking dependent env var resolution", func() {
					stdout, stderr, err := podhelper.ExecBuffered(ctx, hostConfig, hostNS, pPodName, testingContainerName, []string{"sh", "-c", "echo $HELLO_WORLD"}, nil)
					Expect(err).To(Succeed())
					Expect(string(stdout)).To(Equal("Hello World\n"))
					Expect(string(stderr)).To(BeEmpty())
				})

				By("Checking escaped var resolution", func() {
					stdout, stderr, err := podhelper.ExecBuffered(ctx, hostConfig, hostNS, pPodName, testingContainerName, []string{"sh", "-c", "echo $ESCAPED_VAR"}, nil)
					Expect(err).To(Succeed())
					Expect(string(stdout)).To(Equal("$(FIRST)\n"))
					Expect(string(stderr)).To(BeEmpty())
				})

				By("Checking service env var resolution", func() {
					stdout, stderr, err := podhelper.ExecBuffered(ctx, hostConfig, hostNS, pPodName, testingContainerName, []string{"sh", "-c", "echo $MY_SERVICE"}, nil)
					Expect(err).To(Succeed())
					Expect(string(stdout)).To(MatchRegexp(fmt.Sprintf("^%s://%s:%d\n$", myProtocol, ipRegExp, svcPort)))
					Expect(string(stderr)).To(BeEmpty())
				})
			})

			It("should propagate namespace labels to host pod labels", func(ctx context.Context) {
				suffix := random.String(6)
				ns := "pod-nslabel-test-" + suffix
				createTestNamespace(ctx, ns)

				podName := "pod-nslabel-" + suffix
				By("Creating a pod", func() {
					_, err := vClusterClient.CoreV1().Pods(ns).Create(ctx, &corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{Name: podName},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:            testingContainerName,
									Image:           testingContainerImage,
									ImagePullPolicy: corev1.PullIfNotPresent,
									SecurityContext: defaultSecurityContext(),
								},
							},
						},
					}, metav1.CreateOptions{})
					Expect(err).To(Succeed())
				})

				waitPodRunning(ctx, podName, ns)

				pPodName := translate.SingleNamespaceHostName(podName, ns, vClusterName)
				hostNS := "vcluster-" + vClusterName

				// findNsLabel scans pod labels for a namespace-prefixed label matching the
				// expected value. translate.HostLabelNamespace uses a process-global VClusterName
				// which is not set in the test process, so we search by value+prefix instead.
				nsLabelPrefix := "vcluster.loft.sh/ns-label-"
				findNsLabel := func(podLabels map[string]string, expectedValue string) string {
					for k, v := range podLabels {
						if strings.HasPrefix(k, nsLabelPrefix) && v == expectedValue {
							return k
						}
					}
					return ""
				}

				By("Checking the initial namespace label is present on the host pod", func() {
					pPod, err := hostClient.CoreV1().Pods(hostNS).Get(ctx, pPodName, metav1.GetOptions{})
					Expect(err).To(Succeed())
					pKey := findNsLabel(pPod.GetLabels(), initialNsLabelValue)
					Expect(pKey).NotTo(BeEmpty(),
						"no namespace label with value %q found on host pod, labels: %v",
						initialNsLabelValue, pPod.GetLabels())
				})

				additionalLabelKey := "another-one"
				additionalLabelValue := "good-syncer"

				By("Adding a label to the namespace and verifying it propagates", func() {
					err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
						namespace, err := vClusterClient.CoreV1().Namespaces().Get(ctx, ns, metav1.GetOptions{})
						if err != nil {
							return err
						}
						labels := namespace.GetLabels()
						labels[additionalLabelKey] = additionalLabelValue
						namespace.SetLabels(labels)
						_, err = vClusterClient.CoreV1().Namespaces().Update(ctx, namespace, metav1.UpdateOptions{})
						return err
					})
					Expect(err).To(Succeed())

					Eventually(func(g Gomega) {
						pPod, err := hostClient.CoreV1().Pods(hostNS).Get(ctx, pPodName, metav1.GetOptions{})
						g.Expect(err).To(Succeed())
						pKey := findNsLabel(pPod.GetLabels(), additionalLabelValue)
						g.Expect(pKey).NotTo(BeEmpty(),
							"namespace label with value %q not yet propagated to host pod, labels: %v",
							additionalLabelValue, pPod.GetLabels())
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
				})
			})

			It("should sync service account tokens through secrets", func(ctx context.Context) {
				suffix := random.String(6)
				ns := "pod-satoken-test-" + suffix
				createTestNamespace(ctx, ns)

				podName := "pod-satoken-" + suffix
				By("Creating a pod", func() {
					_, err := vClusterClient.CoreV1().Pods(ns).Create(ctx, &corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{Name: podName, Namespace: ns},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{Name: podName, Image: "nginx"},
							},
						},
					}, metav1.CreateOptions{})
					Expect(err).To(Succeed())
				})

				waitPodRunning(ctx, podName, ns)

				pPodName := translate.SingleNamespaceHostName(podName, ns, vClusterName)
				hostNS := "vcluster-" + vClusterName

				By("Verifying the service account token annotation is not present on host pod", func() {
					pPod, err := hostClient.CoreV1().Pods(hostNS).Get(ctx, pPodName, metav1.GetOptions{})
					Expect(err).To(Succeed())
					_, ok := pPod.GetAnnotations()[podtranslate.PodServiceAccountTokenSecretName]
					Expect(ok).To(BeFalse(), "service account token annotation should not be present")
				})

				By("Verifying the SA token secret exists in host cluster", func() {
					secretName := translate.SingleNamespaceHostName(fmt.Sprintf("%s-sa-token", podName), ns, vClusterName)
					_, err := hostClient.CoreV1().Secrets(hostNS).Get(ctx, secretName, metav1.GetOptions{})
					Expect(err).To(Succeed())
				})

				By("Verifying projected volume uses a secret instead of direct SA token", func() {
					pPod, err := hostClient.CoreV1().Pods(hostNS).Get(ctx, pPodName, metav1.GetOptions{})
					Expect(err).To(Succeed())
					secretName := translate.SingleNamespaceHostName(fmt.Sprintf("%s-sa-token", podName), ns, vClusterName)
					for _, volume := range pPod.Spec.Volumes {
						if volume.Projected != nil {
							for _, source := range volume.Projected.Sources {
								if source.Secret != nil {
									Expect(source.Secret.Name).To(Equal(secretName))
								}
							}
						}
					}
				})
			})

			It("should perform bidirectional sync on labels and annotations", func(ctx context.Context) {
				suffix := random.String(6)
				ns := "pod-bidir-test-" + suffix
				createTestNamespace(ctx, ns)

				podName := "pod-bidir-" + suffix
				hostNS := "vcluster-" + vClusterName

				By("Creating a pod with initial annotations and labels", func() {
					_, err := vClusterClient.CoreV1().Pods(ns).Create(ctx, &corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Name: podName,
							Annotations: map[string]string{
								"vcluster-annotation": "from vCluster with love",
							},
							Labels: map[string]string{
								"vcluster-specific-label": "with_its_value",
							},
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:            testingContainerName,
									Image:           testingContainerImage,
									ImagePullPolicy: corev1.PullIfNotPresent,
									SecurityContext: defaultSecurityContext(),
								},
							},
						},
					}, metav1.CreateOptions{})
					Expect(err).To(Succeed())
				})

				waitPodRunning(ctx, podName, ns)

				pPodName := translate.SingleNamespaceHostName(podName, ns, vClusterName)
				additionalLabelKey := "another-one"
				additionalLabelValue := "good-syncer"
				additionalAnnotationKey := "annotation-key"
				additionalAnnotationValue := "annotation-value"

				By("Adding labels and annotations on the host pod", func() {
					err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
						pPod, err := hostClient.CoreV1().Pods(hostNS).Get(ctx, pPodName, metav1.GetOptions{})
						if err != nil {
							return err
						}
						if pPod.Labels == nil {
							pPod.Labels = map[string]string{}
						}
						pPod.Labels[additionalLabelKey] = additionalLabelValue
						if pPod.Annotations == nil {
							pPod.Annotations = map[string]string{}
						}
						pPod.Annotations[additionalAnnotationKey] = additionalAnnotationValue
						_, err = hostClient.CoreV1().Pods(hostNS).Update(ctx, pPod, metav1.UpdateOptions{})
						return err
					})
					Expect(err).To(Succeed())
				})

				By("Verifying host-added labels and annotations sync to vCluster", func() {
					Eventually(func(g Gomega) {
						vPod, err := vClusterClient.CoreV1().Pods(ns).Get(ctx, podName, metav1.GetOptions{})
						g.Expect(err).To(Succeed())
						g.Expect(vPod.Annotations).To(HaveKeyWithValue(additionalAnnotationKey, additionalAnnotationValue),
							"annotation not synced from host to vCluster")
						g.Expect(vPod.Labels).To(HaveKeyWithValue(additionalLabelKey, additionalLabelValue),
							"label not synced from host to vCluster")
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
				})

				additionalLabelValueFromVCluster := "good-syncer-from-vcluster"
				additionalAnnotationValueFromVCluster := "annotation-value-from-vcluster"

				By("Updating labels and annotations from vCluster side", func() {
					err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
						vPod, err := vClusterClient.CoreV1().Pods(ns).Get(ctx, podName, metav1.GetOptions{})
						if err != nil {
							return err
						}
						if vPod.Labels == nil {
							vPod.Labels = map[string]string{}
						}
						vPod.Labels[additionalLabelKey] = additionalLabelValueFromVCluster
						if vPod.Annotations == nil {
							vPod.Annotations = map[string]string{}
						}
						vPod.Annotations[additionalAnnotationKey] = additionalAnnotationValueFromVCluster
						_, err = vClusterClient.CoreV1().Pods(ns).Update(ctx, vPod, metav1.UpdateOptions{})
						return err
					})
					Expect(err).To(Succeed())
				})

				By("Verifying vCluster-updated values sync to host", func() {
					Eventually(func(g Gomega) {
						pPod, err := hostClient.CoreV1().Pods(hostNS).Get(ctx, pPodName, metav1.GetOptions{})
						g.Expect(err).To(Succeed())
						g.Expect(pPod.Annotations).To(HaveKeyWithValue(additionalAnnotationKey, additionalAnnotationValueFromVCluster),
							"annotation not synced from vCluster to host")
						g.Expect(pPod.Labels).To(HaveKeyWithValue(additionalLabelKey, additionalLabelValueFromVCluster),
							"label not synced from vCluster to host")
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
				})
			})

			It("should increment pod generation when tolerations are updated", func(ctx context.Context) {
				suffix := random.String(6)
				ns := "pod-gen-test-" + suffix
				createTestNamespace(ctx, ns)

				podName := "pod-gen-" + suffix
				By("Creating a pod", func() {
					_, err := vClusterClient.CoreV1().Pods(ns).Create(ctx, &corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{Name: podName},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:            testingContainerName,
									Image:           testingContainerImage,
									ImagePullPolicy: corev1.PullIfNotPresent,
									SecurityContext: defaultSecurityContext(),
								},
							},
						},
					}, metav1.CreateOptions{})
					Expect(err).To(Succeed())
				})

				waitPodRunning(ctx, podName, ns)

				By("Waiting for status.observedGeneration to reach 1", func() {
					Eventually(func(g Gomega) {
						vpod, err := vClusterClient.CoreV1().Pods(ns).Get(ctx, podName, metav1.GetOptions{})
						g.Expect(err).To(Succeed())
						g.Expect(vpod.Status.ObservedGeneration).To(BeNumerically("==", 1),
							"status.observedGeneration is %d, expected 1", vpod.Status.ObservedGeneration)
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
				})

				By("Updating pod tolerations to trigger a generation bump", func() {
					err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
						vpod, err := vClusterClient.CoreV1().Pods(ns).Get(ctx, podName, metav1.GetOptions{})
						if err != nil {
							return err
						}
						vpod.Spec.Tolerations = append(vpod.Spec.Tolerations, corev1.Toleration{
							Key:      "e2e-generation-test",
							Operator: corev1.TolerationOpExists,
							Effect:   corev1.TaintEffectNoExecute,
						})
						_, err = vClusterClient.CoreV1().Pods(ns).Update(ctx, vpod, metav1.UpdateOptions{})
						return err
					})
					Expect(err).To(Succeed())
				})

				By("Waiting for status.observedGeneration to reach 2", func() {
					Eventually(func(g Gomega) {
						vpod, err := vClusterClient.CoreV1().Pods(ns).Get(ctx, podName, metav1.GetOptions{})
						g.Expect(err).To(Succeed())
						g.Expect(vpod.Status.ObservedGeneration).To(BeNumerically(">=", 2),
							"status.observedGeneration is %d, expected >= 2", vpod.Status.ObservedGeneration)
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
				})
			})
		},
	)
}

func boolPtr(b bool) *bool    { return &b }
func int64Ptr(i int64) *int64 { return &i }
