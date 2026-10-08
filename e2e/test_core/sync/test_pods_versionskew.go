package test_core

import (
	"context"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e/constants"
	"github.com/loft-sh/vcluster/e2e/labels"
	"github.com/loft-sh/vcluster/pkg/util/random"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilversion "k8s.io/apimachinery/pkg/util/version"
	"k8s.io/client-go/kubernetes"
)

// PodVersionSkewSpec registers pod sync specs that require a virtual cluster older than
// the host (virtual < 1.34 <= host), the setup from
// https://github.com/loft-sh/vcluster/issues/3578.
func PodVersionSkewSpec() {
	Describe("Pod sync with a virtual cluster older than the host",
		labels.Core, labels.Pods,
		func() {
			var (
				hostClient     kubernetes.Interface
				vClusterClient kubernetes.Interface
				vClusterName   string
			)

			k8s134 := utilversion.MustParseSemantic("1.34.0")

			BeforeEach(func(ctx context.Context) {
				hostClient = cluster.KubeClientFrom(ctx, constants.GetHostClusterName())
				Expect(hostClient).NotTo(BeNil())
				vClusterClient = cluster.CurrentKubeClientFrom(ctx)
				Expect(vClusterClient).NotTo(BeNil())
				vClusterName = cluster.CurrentClusterNameFrom(ctx)
			})

			createTestNamespace := func(ctx context.Context, nsName string) {
				GinkgoHelper()
				_, err := vClusterClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: nsName},
				}, metav1.CreateOptions{})
				Expect(err).To(Succeed())
				DeferCleanup(func(ctx context.Context) {
					err := vClusterClient.CoreV1().Namespaces().Delete(ctx, nsName, metav1.DeleteOptions{})
					if !kerrors.IsNotFound(err) {
						Expect(err).To(Succeed())
					}
				})
			}

			testContainer := func() corev1.Container {
				return corev1.Container{
					Name:            testingContainerName,
					Image:           testingContainerImage,
					ImagePullPolicy: corev1.PullIfNotPresent,
					SecurityContext: &corev1.SecurityContext{
						Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
						RunAsNonRoot:             boolPtr(true),
						RunAsUser:                int64Ptr(12345),
						AllowPrivilegeEscalation: boolPtr(false),
						SeccompProfile:           &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault},
					},
				}
			}

			readyCondition := And(
				HaveField("Type", corev1.PodReady),
				HaveField("Status", corev1.ConditionTrue),
			)

			It("should run a virtual cluster older than 1.34 on a host at least 1.34", func(ctx context.Context) {
				// the other specs only prove anything with this skew; adjust
				// vcluster-versionskew.yaml when this fails
				hostInfo, err := hostClient.Discovery().ServerVersion()
				Expect(err).To(Succeed())
				hostVersion, err := utilversion.ParseGeneric(hostInfo.String())
				Expect(err).To(Succeed())

				virtualInfo, err := vClusterClient.Discovery().ServerVersion()
				Expect(err).To(Succeed())
				virtualVersion, err := utilversion.ParseGeneric(virtualInfo.String())
				Expect(err).To(Succeed())

				Expect(virtualVersion.LessThan(k8s134)).To(BeTrue(),
					"virtual cluster version %s must be below 1.34 for the version skew tests, adjust vcluster-versionskew.yaml", virtualInfo)
				Expect(hostVersion.AtLeast(k8s134)).To(BeTrue(),
					"host cluster version %s must be at least 1.34 for the version skew tests", hostInfo)
			})

			It("should make a deployment available and keep it available", func(ctx context.Context) {
				suffix := random.String(6)
				ns := "skew-deploy-test-" + suffix
				createTestNamespace(ctx, ns)

				deployName := "skew-deploy-" + suffix
				podLabels := map[string]string{"app": deployName}
				By("Creating a deployment", func() {
					replicas := int32(1)
					_, err := vClusterClient.AppsV1().Deployments(ns).Create(ctx, &appsv1.Deployment{
						ObjectMeta: metav1.ObjectMeta{Name: deployName},
						Spec: appsv1.DeploymentSpec{
							Replicas: &replicas,
							Selector: &metav1.LabelSelector{MatchLabels: podLabels},
							Template: corev1.PodTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{Labels: podLabels},
								Spec:       corev1.PodSpec{Containers: []corev1.Container{testContainer()}},
							},
						},
					}, metav1.CreateOptions{})
					Expect(err).To(Succeed())
				})

				availableCondition := And(
					HaveField("Type", appsv1.DeploymentAvailable),
					HaveField("Status", corev1.ConditionTrue),
				)

				By("Waiting for the deployment to become available", func() {
					Eventually(func(g Gomega) {
						deploy, err := vClusterClient.AppsV1().Deployments(ns).Get(ctx, deployName, metav1.GetOptions{})
						g.Expect(err).To(Succeed(), "failed to get deployment %s/%s", ns, deployName)
						g.Expect(deploy.Status.AvailableReplicas).To(BeNumerically("==", 1),
							"deployment has %d available replicas, expected 1", deploy.Status.AvailableReplicas)
						g.Expect(deploy.Status.Conditions).To(ContainElement(availableCondition),
							"deployment Available condition is not True: %v", deploy.Status.Conditions)
					}).WithContext(ctx).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
				})

				By("Verifying the deployment stays available", func() {
					Consistently(func(g Gomega) {
						deploy, err := vClusterClient.AppsV1().Deployments(ns).Get(ctx, deployName, metav1.GetOptions{})
						g.Expect(err).To(Succeed(), "failed to get deployment %s/%s", ns, deployName)
						g.Expect(deploy.Status.AvailableReplicas).To(BeNumerically("==", 1),
							"deployment lost its available replica")
						g.Expect(deploy.Status.Conditions).To(ContainElement(availableCondition),
							"deployment Available condition flapped away from True: %v", deploy.Status.Conditions)
					}).WithContext(ctx).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutShort).Should(Succeed())
				})
			})

			It("should keep the pod Ready and not sync unpersisted status fields", func(ctx context.Context) {
				suffix := random.String(6)
				ns := "skew-pod-test-" + suffix
				createTestNamespace(ctx, ns)

				podName := "skew-pod-" + suffix
				By("Creating a pod", func() {
					_, err := vClusterClient.CoreV1().Pods(ns).Create(ctx, &corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{Name: podName},
						Spec:       corev1.PodSpec{Containers: []corev1.Container{testContainer()}},
					}, metav1.CreateOptions{})
					Expect(err).To(Succeed())
				})

				By("Waiting for the pod to be Running with Ready=True", func() {
					Eventually(func(g Gomega) {
						pod, err := vClusterClient.CoreV1().Pods(ns).Get(ctx, podName, metav1.GetOptions{})
						g.Expect(err).To(Succeed(), "failed to get pod %s/%s", ns, podName)
						g.Expect(pod.Status.Phase).To(Equal(corev1.PodRunning),
							"pod phase is %s, not yet Running", pod.Status.Phase)
						g.Expect(pod.Status.Conditions).To(ContainElement(readyCondition),
							"pod Ready condition is not yet True: %v", pod.Status.Conditions)
					}).WithContext(ctx).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
				})

				By("Verifying the skew is exercised: the host stores observedGeneration, the virtual cluster does not", func() {
					pPodName := translate.SingleNamespaceHostName(podName, ns, vClusterName)
					hostNS := vClusterHostNamespace(vClusterName)
					Eventually(func(g Gomega) {
						pPod, err := hostClient.CoreV1().Pods(hostNS).Get(ctx, pPodName, metav1.GetOptions{})
						g.Expect(err).To(Succeed(), "failed to get host pod %s/%s", hostNS, pPodName)
						g.Expect(pPod.Status.ObservedGeneration).To(BeNumerically(">=", 1),
							"host pod status.observedGeneration is not set, the host kubelet should set it on K8s >= 1.34")
					}).WithContext(ctx).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())

					vPod, err := vClusterClient.CoreV1().Pods(ns).Get(ctx, podName, metav1.GetOptions{})
					Expect(err).To(Succeed())
					Expect(vPod.Status.ObservedGeneration).To(BeNumerically("==", 0),
						"virtual pod status.observedGeneration should stay 0, a virtual apiserver below 1.34 strips it on write")
				})

				By("Verifying the Ready condition stays True and does not flap", func() {
					Consistently(func(g Gomega) {
						pod, err := vClusterClient.CoreV1().Pods(ns).Get(ctx, podName, metav1.GetOptions{})
						g.Expect(err).To(Succeed(), "failed to get pod %s/%s", ns, podName)
						g.Expect(pod.Status.Conditions).To(ContainElement(readyCondition),
							"pod Ready condition flapped away from True: %v", pod.Status.Conditions)
					}).WithContext(ctx).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutShort).Should(Succeed())
				})
			})
		},
	)
}
