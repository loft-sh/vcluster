// Package misconfig holds specs for a control plane whose running state
// doesn't match its vcluster.yaml.
package misconfig

import (
	"context"
	"fmt"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e/constants"
	"github.com/loft-sh/vcluster/e2e/labels"
	"github.com/loft-sh/vcluster/pkg/leaderelection"
	"github.com/loft-sh/vcluster/pkg/util/random"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/utils/ptr"
)

const (
	// The customer lost all 9 mounted ConfigMaps after one app scale cycle.
	mountedObjects = 9
	appScaleCycles = 3

	writtenAfterScale = "written-after-scale"
)

// ReplicaMismatchSpec scales the control plane past highAvailability.replicas
// and checks that no data is lost and only one pod serves (ENGCP-1879).
func ReplicaMismatchSpec(kind ControlPlaneKind) {
	// Ordered because the later specs rely on the namespace, app and mounted objects created
	// earlier, and on the control plane already being scaled to 2.
	Describe("Control plane scaled outside vcluster.yaml", Ordered,
		labels.Core, labels.ConfigMaps, labels.Secrets,
		func() {
			var (
				hostClient      kubernetes.Interface
				hostConfig      *rest.Config
				vClusterClient  kubernetes.Interface
				vClusterConfig  *rest.Config
				vClusterName    string
				hostNamespace   string
				namespace       string
				expectedCMs     []string
				expectedSecrets []string
			)

			BeforeAll(func(ctx context.Context) {
				hostClient = cluster.KubeClientFrom(ctx, constants.GetHostClusterName())
				Expect(hostClient).NotTo(BeNil())
				vClusterClient = cluster.CurrentKubeClientFrom(ctx)
				Expect(vClusterClient).NotTo(BeNil())
				vClusterName = cluster.CurrentClusterNameFrom(ctx)
				hostConfig = cluster.From(ctx, constants.GetHostClusterName()).KubernetesRestConfig()
				vClusterConfig = cluster.From(ctx, vClusterName).KubernetesRestConfig()
				hostNamespace = "vcluster-" + vClusterName
				namespace = "replica-mismatch-" + random.String(6)

				_, err := vClusterClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: namespace,
					},
				}, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())

				DeferCleanup(func(ctx context.Context) {
					err := scaleControlPlane(ctx, hostClient, kind, hostNamespace, vClusterName, 1)
					Expect(err).NotTo(HaveOccurred())

					// The restart spec kills the background proxy, so talk to the serving pod directly.
					var serving corev1.Pod
					Eventually(func(g Gomega) {
						pod, err := servingPod(ctx, hostClient, hostNamespace, vClusterName)
						g.Expect(err).NotTo(HaveOccurred())
						serving = pod
					}).WithContext(ctx).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
					podClient, stop, err := podVClusterClient(ctx, hostConfig, hostClient, vClusterConfig, hostNamespace, serving.Name)
					Expect(err).NotTo(HaveOccurred())
					defer stop()

					err = podClient.CoreV1().Namespaces().Delete(ctx, namespace, metav1.DeleteOptions{})
					// emptyDir: the Deployment's data went with the pod the restart spec deleted.
					if kind == Deployment && kerrors.IsNotFound(err) {
						return
					}
					Expect(err).NotTo(HaveOccurred(), "namespace %s vanished before cleanup", namespace)
				})
			})

			It("starts with the single replica vcluster.yaml asks for", func(ctx context.Context) {
				replicas, err := controlPlaneReplicas(ctx, hostClient, kind, hostNamespace, vClusterName)
				Expect(err).NotTo(HaveOccurred())
				Expect(replicas).To(Equal(int32(1)), "%s %s/%s must start with 1 replica", kind, hostNamespace, vClusterName)

				Eventually(func(g Gomega) {
					pods, err := controlPlanePods(ctx, hostClient, hostNamespace, vClusterName)
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(pods).To(HaveLen(1))
					g.Expect(isPodReady(pods[0])).To(BeTrue(), "control plane pod %s is not Ready", pods[0].Name)
				}).WithContext(ctx).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
			})

			It("runs an app that mounts ConfigMaps and Secrets", func(ctx context.Context) {
				for i := range mountedObjects {
					cmName := fmt.Sprintf("mounted-cm-%d", i)
					_, err := vClusterClient.CoreV1().ConfigMaps(namespace).Create(ctx, &corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name: cmName,
						},
						Data: map[string]string{
							"key": cmName,
						},
					}, metav1.CreateOptions{})
					Expect(err).NotTo(HaveOccurred())
					expectedCMs = append(expectedCMs, cmName)

					secretName := fmt.Sprintf("mounted-secret-%d", i)
					_, err = vClusterClient.CoreV1().Secrets(namespace).Create(ctx, &corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name: secretName,
						},
						StringData: map[string]string{
							"key": secretName,
						},
					}, metav1.CreateOptions{})
					Expect(err).NotTo(HaveOccurred())
					expectedSecrets = append(expectedSecrets, secretName)
				}

				_, err := vClusterClient.AppsV1().Deployments(namespace).Create(ctx,
					mountingDeployment(namespace, expectedCMs, expectedSecrets), metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())
				waitForAppReplicas(ctx, vClusterClient, namespace, 1)
			})

			It("accepts a direct scale of the control plane to 2 replicas", func(ctx context.Context) {
				err := scaleControlPlane(ctx, hostClient, kind, hostNamespace, vClusterName, 2)
				Expect(err).NotTo(HaveOccurred())

				Eventually(func(g Gomega) {
					pods, err := controlPlanePods(ctx, hostClient, hostNamespace, vClusterName)
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(pods).To(HaveLen(2))
					for _, pod := range pods {
						g.Expect(pod.Status.Phase).To(Equal(corev1.PodRunning), "control plane pod %s is not Running", pod.Name)
					}
				}).WithContext(ctx).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutVeryLong).Should(Succeed())
			})

			It("shows a write made after the scale on every ready control plane pod", func(ctx context.Context) {
				_, err := vClusterClient.CoreV1().ConfigMaps(namespace).Create(ctx, &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name: writtenAfterScale,
					},
					Data: map[string]string{
						"key": writtenAfterScale,
					},
				}, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())
				expectedCMs = append(expectedCMs, writtenAfterScale)

				// The Service routes to every ready pod, so each one must see the write.
				Consistently(func(g Gomega) {
					pods, err := controlPlanePods(ctx, hostClient, hostNamespace, vClusterName)
					g.Expect(err).NotTo(HaveOccurred())
					ready := 0
					var missing []string
					for _, pod := range pods {
						if !isPodReady(pod) {
							continue
						}
						ready++
						if err := configMapOnPod(ctx, hostConfig, hostClient, vClusterConfig, hostNamespace, pod.Name, namespace, writtenAfterScale); err != nil {
							missing = append(missing, err.Error())
						}
					}
					g.Expect(ready).To(BeNumerically(">=", 1), "no control plane pod is Ready")
					g.Expect(missing).To(BeEmpty(), "Ready control plane pods without ConfigMap %s/%s", namespace, writtenAfterScale)
				}).WithContext(ctx).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
			})

			It("keeps every mounted ConfigMap and Secret across app restarts", func(ctx context.Context) {
				for cycle := range appScaleCycles {
					By(fmt.Sprintf("scaling the app to 0 and back to 1 (cycle %d)", cycle+1), func() {
						scaleApp(ctx, vClusterClient, namespace, 0)
						waitForAppReplicas(ctx, vClusterClient, namespace, 0)
						scaleApp(ctx, vClusterClient, namespace, 1)
						waitForAppReplicas(ctx, vClusterClient, namespace, 1)
						expectMountedObjects(ctx, vClusterClient, namespace, expectedCMs, expectedSecrets)
					})
				}

				By("leaving the cluster alone and checking nothing disappears", func() {
					Expect(expectedCMs).NotTo(BeEmpty(), "no ConfigMaps recorded; did an earlier spec skip?")
					Expect(expectedSecrets).NotTo(BeEmpty(), "no Secrets recorded; did an earlier spec skip?")
					Consistently(func(g Gomega) {
						cms, err := configMapNames(ctx, vClusterClient, namespace)
						g.Expect(err).NotTo(HaveOccurred())
						g.Expect(cms).To(Equal(expectedCMs), "ConfigMaps were deleted")
						secrets, err := secretNames(ctx, vClusterClient, namespace)
						g.Expect(err).NotTo(HaveOccurred())
						g.Expect(secrets).To(Equal(expectedSecrets), "mounted Secrets were deleted")
					}).WithContext(ctx).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
				})
			})

			It("serves the vCluster API from exactly one control plane pod", func(ctx context.Context) {
				Eventually(func(g Gomega) {
					ready, err := readyServiceEndpoints(ctx, hostClient, hostNamespace, vClusterName)
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(ready).To(Equal(1), "Service %s/%s must route to exactly one control plane pod", hostNamespace, vClusterName)
				}).WithContext(ctx).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
			})

			It("reports the control plane pod that is not serving", func(ctx context.Context) {
				Eventually(func(g Gomega) {
					events, err := hostClient.CoreV1().Events(hostNamespace).List(ctx, metav1.ListOptions{
						FieldSelector: fields.Set{
							"involvedObject.name": vClusterName,
							"involvedObject.kind": string(kind),
							"reason":              leaderelection.EventReasonReplicaNotServing,
							"type":                corev1.EventTypeWarning,
						}.String(),
					})
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(events.Items).NotTo(BeEmpty(), "expected a %s Warning event on %s %s/%s",
						leaderelection.EventReasonReplicaNotServing, kind, hostNamespace, vClusterName)
				}).WithContext(ctx).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
			})

			It("keeps one serving pod after the serving pod restarts", func(ctx context.Context) {
				before, err := servingPod(ctx, hostClient, hostNamespace, vClusterName)
				Expect(err).NotTo(HaveOccurred())
				err = hostClient.CoreV1().Pods(hostNamespace).Delete(ctx, before.Name, metav1.DeleteOptions{})
				Expect(err).NotTo(HaveOccurred())

				var after corev1.Pod
				Eventually(func(g Gomega) {
					pod, err := servingPod(ctx, hostClient, hostNamespace, vClusterName)
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(pod.UID).NotTo(Equal(before.UID), "deleted pod %s still serves", before.Name)
					ready, err := readyServiceEndpoints(ctx, hostClient, hostNamespace, vClusterName)
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(ready).To(Equal(1), "Service %s/%s must route to exactly one control plane pod", hostNamespace, vClusterName)
					after = pod
				}).WithContext(ctx).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutVeryLong).Should(Succeed())

				switch kind {
				case StatefulSet:
					// pod-0 owns the data volume, so it must serve again with the same data.
					Expect(before.Name).To(Equal(vClusterName+"-0"), "only pod-0 may serve embedded SQLite")
					Expect(after.Name).To(Equal(before.Name), "pod %s took over from %s", after.Name, before.Name)
					podClient, stop, err := podVClusterClient(ctx, hostConfig, hostClient, vClusterConfig, hostNamespace, after.Name)
					Expect(err).NotTo(HaveOccurred())
					defer stop()
					expectMountedObjects(ctx, podClient, namespace, expectedCMs, expectedSecrets)
				case Deployment:
					// emptyDir: the data went with the deleted pod, so only the single serving pod is checked.
				}
			})
		},
	)
}

func scaleApp(ctx context.Context, c kubernetes.Interface, namespace string, replicas int32) {
	GinkgoHelper()
	scale, err := c.AppsV1().Deployments(namespace).GetScale(ctx, appName, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred())
	scale.Spec.Replicas = replicas
	_, err = c.AppsV1().Deployments(namespace).UpdateScale(ctx, appName, scale, metav1.UpdateOptions{})
	Expect(err).NotTo(HaveOccurred())
}

// waitForAppReplicas waits until the app has exactly `replicas` pods, all Ready.
// A pod stuck on a missing ConfigMap or Secret mount never gets Ready.
func waitForAppReplicas(ctx context.Context, c kubernetes.Interface, namespace string, replicas int32) {
	GinkgoHelper()
	Eventually(func(g Gomega) {
		deploy, err := c.AppsV1().Deployments(namespace).Get(ctx, appName, metav1.GetOptions{})
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(deploy.Status.ObservedGeneration).To(Equal(deploy.Generation), "app %s/%s not reconciled yet", deploy.Namespace, deploy.Name)
		g.Expect(deploy.Status.Replicas).To(Equal(replicas), appNotReadyMessage(deploy))
		g.Expect(deploy.Status.ReadyReplicas).To(Equal(replicas), appNotReadyMessage(deploy))
	}).WithContext(ctx).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
}

func appNotReadyMessage(deploy *appsv1.Deployment) string {
	return fmt.Sprintf("app %s/%s has %d/%d ready pods; a missing mounted ConfigMap or Secret keeps pods in ContainerCreating",
		deploy.Namespace, deploy.Name, deploy.Status.ReadyReplicas, ptr.Deref(deploy.Spec.Replicas, 0))
}

func expectMountedObjects(ctx context.Context, c kubernetes.Interface, namespace string, expectedCMs, expectedSecrets []string) {
	GinkgoHelper()
	Expect(expectedCMs).NotTo(BeEmpty(), "no ConfigMaps recorded; did an earlier spec skip?")
	Expect(expectedSecrets).NotTo(BeEmpty(), "no Secrets recorded; did an earlier spec skip?")
	cms, err := configMapNames(ctx, c, namespace)
	Expect(err).NotTo(HaveOccurred())
	Expect(cms).To(Equal(expectedCMs), "ConfigMaps were deleted")
	secrets, err := secretNames(ctx, c, namespace)
	Expect(err).NotTo(HaveOccurred())
	Expect(secrets).To(Equal(expectedSecrets), "mounted Secrets were deleted")
}

// configMapOnPod reads the ConfigMap straight from one control plane pod, bypassing the Service.
func configMapOnPod(ctx context.Context, hostConfig *rest.Config, hostClient kubernetes.Interface, vClusterConfig *rest.Config, hostNamespace, pod, namespace, name string) error {
	podClient, stop, err := podVClusterClient(ctx, hostConfig, hostClient, vClusterConfig, hostNamespace, pod)
	if err != nil {
		return fmt.Errorf("connect to control plane pod %s: %w", pod, err)
	}
	defer stop()

	_, err = podClient.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("control plane pod %s: %w", pod, err)
	}
	return nil
}
