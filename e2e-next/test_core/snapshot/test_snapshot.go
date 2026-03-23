package snapshot

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ghodss/yaml"
	snapshotsv1 "github.com/kubernetes-csi/external-snapshotter/client/v8/clientset/versioned"
	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/e2e-framework/pkg/setup/suite"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	vclusterconfig "github.com/loft-sh/vcluster/pkg/config"
	pkgconstants "github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/helm"
	"github.com/loft-sh/vcluster/pkg/snapshot"
	"github.com/loft-sh/vcluster/pkg/snapshot/volumes"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/utils/ptr"
)

// DescribeSnapshotAndRestore registers snapshot and restore tests against the given vCluster.
func DescribeSnapshotAndRestore(vcluster suite.Dependency) bool {
	return Describe("Snapshot and restore",
		Ordered,
		labels.Core,
		labels.Snapshots,
		cluster.Use(vcluster),
		cluster.Use(clusters.HostCluster),
		func() {
			var (
				hostClient        kubernetes.Interface
				vClusterClient    kubernetes.Interface
				vClusterName      string
				vClusterNamespace string
			)

			// Shared test resources
			var (
				configMapToRestore  *corev1.ConfigMap
				configMapToDelete   *corev1.ConfigMap
				secretToRestore     *corev1.Secret
				secretToDelete      *corev1.Secret
				deploymentToRestore *appsv1.Deployment
				serviceToRestore    *corev1.Service
			)

			BeforeAll(func(ctx context.Context) {
				hostClient = cluster.KubeClientFrom(ctx, constants.GetHostClusterName())
				Expect(hostClient).NotTo(BeNil())
				vClusterClient = cluster.CurrentKubeClientFrom(ctx)
				Expect(vClusterClient).NotTo(BeNil())
				vClusterName = cluster.CurrentClusterNameFrom(ctx)
				vClusterNamespace = "vcluster-" + vClusterName
			})

			deployTestResources := func(ctx context.Context, testNamespace string) {
				GinkgoHelper()
				By("Creating test namespace "+testNamespace, func() {
					_, err := vClusterClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
					}, metav1.CreateOptions{})
					Expect(err).NotTo(HaveOccurred())
				})

				configMapToRestore = &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{Name: "configmap-restore", Namespace: testNamespace, Labels: map[string]string{"snapshot": "restore"}},
					Data:       map[string]string{"somekey": "somevalue"},
				}
				configMapToDelete = &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{Name: "configmap-delete", Namespace: testNamespace, Labels: map[string]string{"snapshot": "delete"}},
					Data:       map[string]string{"somesome": "somevalue"},
				}
				secretToRestore = &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{Name: "secret-restore", Namespace: testNamespace, Labels: map[string]string{"snapshot": "restore"}},
					Data:       map[string][]byte{"BOO_BAR": []byte("hello-world")},
				}
				secretToDelete = &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{Name: "secret-delete", Namespace: testNamespace, Labels: map[string]string{"snapshot": "delete"}},
					Data:       map[string][]byte{"ANOTHER_ENV": []byte("another-hello-world")},
				}
				deploymentToRestore = &appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{Name: "deployment-restore", Namespace: testNamespace, Labels: map[string]string{"snapshot": "restore"}},
					Spec: appsv1.DeploymentSpec{
						Replicas: ptr.To(int32(1)),
						Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"snapshot": "restore"}},
						Template: corev1.PodTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"snapshot": "restore"}},
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{{Name: "example-container", Image: "nginx:1.25.0", Ports: []corev1.ContainerPort{{ContainerPort: 80}}}},
							},
						},
					},
				}
				serviceToRestore = &corev1.Service{
					ObjectMeta: metav1.ObjectMeta{Name: "snapshot-restore", Namespace: testNamespace, Labels: map[string]string{"snapshot": "restore"}},
					Spec:       corev1.ServiceSpec{Ports: []corev1.ServicePort{{Name: "https", Port: 443}}, Type: corev1.ServiceTypeClusterIP},
				}

				By("Creating test resources in "+testNamespace, func() {
					_, err := vClusterClient.CoreV1().Services(testNamespace).Create(ctx, serviceToRestore, metav1.CreateOptions{})
					Expect(err).NotTo(HaveOccurred())
					_, err = vClusterClient.CoreV1().ConfigMaps(testNamespace).Create(ctx, configMapToRestore, metav1.CreateOptions{})
					Expect(err).NotTo(HaveOccurred())
					_, err = vClusterClient.CoreV1().Secrets(testNamespace).Create(ctx, secretToRestore, metav1.CreateOptions{})
					Expect(err).NotTo(HaveOccurred())
					_, err = vClusterClient.AppsV1().Deployments(testNamespace).Create(ctx, deploymentToRestore, metav1.CreateOptions{})
					Expect(err).NotTo(HaveOccurred())
				})
			}

			// refreshClient reconnects to the vCluster after destructive operations (restore).
			// The background proxy dies when the vCluster pod restarts.
			refreshClient := func(ctx context.Context) {
				GinkgoHelper()
				By("Reconnecting to the vCluster after restore", func() {
					// The suite-level proxy is dead; re-obtain the client.
					// After restore, the vCluster client from context is stale.
					// We need to get a fresh client. Since the host client is still valid,
					// we can use it to verify the vCluster is running, then the framework
					// should have reconnected via the background proxy.
					// If the framework proxy is still alive, CurrentKubeClientFrom will work.
					// Otherwise we need manual reconnection.
					Eventually(func(g Gomega) {
						client := cluster.CurrentKubeClientFrom(ctx)
						g.Expect(client).NotTo(BeNil())
						// Test the connection
						_, err := client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
						g.Expect(err).NotTo(HaveOccurred(), "vCluster client not yet available after restore")
						vClusterClient = client
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutVeryLong).Should(Succeed())
				})
			}

			Describe("controller-based snapshot without volumes", Ordered, func() {
				const (
					testNS       = "ctrl-snapshot-test"
					snapshotPath = "container:///snapshot-data/snapshot.tar.gz"
				)

				// BeforeAll depends on the parent BeforeAll having set up hostClient/vClusterClient
				BeforeAll(func(ctx context.Context) {
					deployTestResources(ctx, testNS)
				})

				// Spec 1: create snapshot request
				It("Creates the snapshot request", func(_ context.Context) {
					createSnapshot(vClusterName, vClusterNamespace, true, snapshotPath, false)
				})

				// Spec 2 depends on spec 1: wait for completion
				It("Creates the snapshot", func(ctx context.Context) {
					waitForSnapshotToBeCreated(ctx, hostClient, vClusterNamespace)
				})

				// Spec 3 depends on spec 2: restore and verify only snapshot resources exist
				It("Verifies only snapshot resources exist after restore", func(ctx context.Context) {
					By("Creating resources that should be removed by restore", func() {
						_, err := vClusterClient.CoreV1().ConfigMaps(testNS).Create(ctx, configMapToDelete, metav1.CreateOptions{})
						Expect(err).NotTo(HaveOccurred())
						_, err = vClusterClient.CoreV1().Secrets(testNS).Create(ctx, secretToDelete, metav1.CreateOptions{})
						Expect(err).NotTo(HaveOccurred())
						serviceToDelete := &corev1.Service{
							ObjectMeta: metav1.ObjectMeta{Name: "snapshot-delete", Namespace: testNS, Labels: map[string]string{"snapshot": "delete"}},
							Spec:       corev1.ServiceSpec{Ports: []corev1.ServicePort{{Name: "http", Port: 80}}, Type: corev1.ServiceTypeClusterIP},
						}
						svcCreated, err := vClusterClient.CoreV1().Services(testNS).Create(ctx, serviceToDelete, metav1.CreateOptions{})
						Expect(err).NotTo(HaveOccurred())
						oldResourceVersion := svcCreated.ResourceVersion

						restoreVCluster(ctx, hostClient, vClusterName, vClusterNamespace, snapshotPath, true, false)
						refreshClient(ctx)

						By("Checking pre-snapshot configmap is restored", func() {
							configmaps, err := vClusterClient.CoreV1().ConfigMaps(testNS).List(ctx, metav1.ListOptions{LabelSelector: "snapshot=restore"})
							Expect(err).NotTo(HaveOccurred())
							Expect(configmaps.Items).To(HaveLen(1))
							Expect(configmaps.Items[0].Data).To(Equal(configMapToRestore.Data))

							// Verify resource version is higher (new etcd)
							newRV, err := strconv.ParseInt(configmaps.Items[0].ResourceVersion, 10, 64)
							Expect(err).NotTo(HaveOccurred())
							oldRV, err := strconv.ParseInt(oldResourceVersion, 10, 64)
							Expect(err).NotTo(HaveOccurred())
							Expect(newRV).To(BeNumerically(">", oldRV))
						})

						By("Checking pre-snapshot secret is restored", func() {
							secrets, err := vClusterClient.CoreV1().Secrets(testNS).List(ctx, metav1.ListOptions{LabelSelector: "snapshot=restore"})
							Expect(err).NotTo(HaveOccurred())
							Expect(secrets.Items).To(HaveLen(1))
							Expect(secrets.Items[0].Data).To(Equal(secretToRestore.Data))
						})

						By("Checking pre-snapshot deployment is restored", func() {
							deps, err := vClusterClient.AppsV1().Deployments(testNS).List(ctx, metav1.ListOptions{LabelSelector: "snapshot=restore"})
							Expect(err).NotTo(HaveOccurred())
							Expect(deps.Items).To(HaveLen(1))
						})

						By("Checking post-snapshot configmap is gone", func() {
							Eventually(func(g Gomega) {
								cms, err := vClusterClient.CoreV1().ConfigMaps(testNS).List(ctx, metav1.ListOptions{LabelSelector: "snapshot=delete"})
								g.Expect(err).NotTo(HaveOccurred())
								g.Expect(cms.Items).To(BeEmpty(), "post-snapshot configmap should be deleted by restore")
							}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
						})

						By("Checking post-snapshot secret is gone", func() {
							Eventually(func(g Gomega) {
								secs, err := vClusterClient.CoreV1().Secrets(testNS).List(ctx, metav1.ListOptions{LabelSelector: "snapshot=delete"})
								g.Expect(err).NotTo(HaveOccurred())
								g.Expect(secs.Items).To(BeEmpty(), "post-snapshot secret should be deleted by restore")
							}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
						})

						By("Checking post-snapshot service is gone", func() {
							Eventually(func(g Gomega) {
								svcs, err := vClusterClient.CoreV1().Services(testNS).List(ctx, metav1.ListOptions{LabelSelector: "snapshot=delete"})
								g.Expect(err).NotTo(HaveOccurred())
								g.Expect(svcs.Items).To(BeEmpty(), "post-snapshot service should be deleted by restore")
							}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
						})
					})
				})

				// Spec 4 depends on spec 2: delete resources then restore and verify they come back
				It("Verifies deleted resources are recreated after restore", func(ctx context.Context) {
					By("Deleting resources that should be restored", func() {
						err := vClusterClient.CoreV1().ConfigMaps(testNS).Delete(ctx, configMapToRestore.Name, metav1.DeleteOptions{})
						Expect(err).NotTo(HaveOccurred())
						err = vClusterClient.CoreV1().Secrets(testNS).Delete(ctx, secretToRestore.Name, metav1.DeleteOptions{})
						Expect(err).NotTo(HaveOccurred())
						err = vClusterClient.AppsV1().Deployments(testNS).Delete(ctx, deploymentToRestore.Name, metav1.DeleteOptions{})
						Expect(err).NotTo(HaveOccurred())
					})

					restoreVCluster(ctx, hostClient, vClusterName, vClusterNamespace, snapshotPath, true, false)
					refreshClient(ctx)

					By("Checking configmap is re-created", func() {
						Eventually(func(g Gomega) {
							cms, err := vClusterClient.CoreV1().ConfigMaps(testNS).List(ctx, metav1.ListOptions{LabelSelector: "snapshot=restore"})
							g.Expect(err).NotTo(HaveOccurred())
							g.Expect(cms.Items).To(HaveLen(1))
							g.Expect(cms.Items[0].Data).To(Equal(configMapToRestore.Data))
						}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
					})

					By("Checking secret is re-created", func() {
						Eventually(func(g Gomega) {
							secs, err := vClusterClient.CoreV1().Secrets(testNS).List(ctx, metav1.ListOptions{LabelSelector: "snapshot=restore"})
							g.Expect(err).NotTo(HaveOccurred())
							g.Expect(secs.Items).To(HaveLen(1))
							g.Expect(secs.Items[0].Data).To(Equal(secretToRestore.Data))
						}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
					})

					By("Checking deployment is re-created", func() {
						Eventually(func(g Gomega) {
							deps, err := vClusterClient.AppsV1().Deployments(testNS).List(ctx, metav1.ListOptions{LabelSelector: "snapshot=restore"})
							g.Expect(err).NotTo(HaveOccurred())
							g.Expect(deps.Items).To(HaveLen(1))
						}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
					})
				})

				AfterAll(func(ctx context.Context) {
					deleteSnapshotRequestConfigMaps(ctx, hostClient, vClusterNamespace)
				})
			})

			Describe("controller-based snapshot with volumes", Ordered, func() {
				const (
					testNS           = "volume-snapshots-test"
					snapshotPath     = "container:///snapshot-data/" + testNS + ".tar.gz"
					pvcToRestoreName = "test-pvc-restore"
					testFileName     = testNS + ".txt"
					pvcData          = "Hello " + testNS
				)

				BeforeAll(func(ctx context.Context) {
					deployTestResources(ctx, testNS)
					createPVCWithData(ctx, vClusterClient, testNS, pvcToRestoreName, testFileName, pvcData)
				})

				// Spec 1: create snapshot request with volumes
				It("Creates the snapshot request", func(_ context.Context) {
					createSnapshot(vClusterName, vClusterNamespace, true, snapshotPath, true)
				})

				// Spec 2 depends on spec 1
				It("Creates the snapshot", func(ctx context.Context) {
					waitForSnapshotToBeCreated(ctx, hostClient, vClusterNamespace)
				})

				// Spec 3 depends on spec 2: verify VolumeSnapshots are cleaned up
				It("Verifies VolumeSnapshots are cleaned up after snapshot completes", func(ctx context.Context) {
					vClusterRelease, err := helm.NewSecrets(hostClient).Get(ctx, vClusterName, vClusterNamespace)
					Expect(err).NotTo(HaveOccurred())
					vConfigValues, err := yaml.Marshal(vClusterRelease.Config)
					Expect(err).NotTo(HaveOccurred())
					vClusterConfig, err := vclusterconfig.ParseConfigBytes(vConfigValues, vClusterName, nil)
					Expect(err).NotTo(HaveOccurred())

					var restConfig *rest.Config
					var volumeSnapshotsNS string
					if vClusterConfig.PrivateNodes.Enabled {
						currentClusterName := cluster.CurrentClusterNameFrom(ctx)
						restConfig = cluster.From(ctx, currentClusterName).KubernetesRestConfig()
						volumeSnapshotsNS = testNS
					} else {
						restConfig = cluster.From(ctx, constants.GetHostClusterName()).KubernetesRestConfig()
						volumeSnapshotsNS = vClusterNamespace
					}
					snapshotClient, err := snapshotsv1.NewForConfig(restConfig)
					Expect(err).NotTo(HaveOccurred())

					volumeSnapshots, err := snapshotClient.SnapshotV1().VolumeSnapshots(volumeSnapshotsNS).List(ctx, metav1.ListOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(volumeSnapshots.Items).To(BeEmpty(), "VolumeSnapshots should be cleaned up after snapshot")

					volumeSnapshotContents, err := snapshotClient.SnapshotV1().VolumeSnapshotContents().List(ctx, metav1.ListOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(volumeSnapshotContents.Items).To(BeEmpty(), "VolumeSnapshotContents should be cleaned up after snapshot")
				})

				// Spec 4 depends on spec 2: delete PVC then restore
				It("Deletes the PVC with test data", func(ctx context.Context) {
					deletePVC(ctx, vClusterClient, hostClient, vClusterName, vClusterNamespace, testNS, pvcToRestoreName)
				})

				// Spec 5 depends on spec 4: restore with volumes
				It("Restores vCluster with volumes", func(ctx context.Context) {
					// PVC has been restored in previous specs but without data, so it's stuck in Pending.
					// Delete it again so it gets restored properly.
					deletePVC(ctx, vClusterClient, hostClient, vClusterName, vClusterNamespace, testNS, pvcToRestoreName)
					restoreVCluster(ctx, hostClient, vClusterName, vClusterNamespace, snapshotPath, true, true)
					refreshClient(ctx)
				})

				// Spec 6 depends on spec 5
				It("Has the restored PVC which is bound", func(ctx context.Context) {
					Eventually(func(g Gomega) {
						pvc, err := vClusterClient.CoreV1().PersistentVolumeClaims(testNS).Get(ctx, pvcToRestoreName, metav1.GetOptions{})
						g.Expect(err).NotTo(HaveOccurred())
						g.Expect(pvc.Status.Phase).To(Equal(corev1.ClaimBound),
							"PVC %s is not bound, phase: %s", pvcToRestoreName, pvc.Status.Phase)
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
				})

				// Spec 7 depends on spec 6
				It("Has the restored PVC with data from the volume snapshot", func(ctx context.Context) {
					checkPVCData(ctx, vClusterClient, testNS, pvcToRestoreName, testFileName, pvcData)
				})

				AfterAll(func(ctx context.Context) {
					deletePVC(ctx, vClusterClient, hostClient, vClusterName, vClusterNamespace, testNS, pvcToRestoreName)
					deleteSnapshotRequestConfigMaps(ctx, hostClient, vClusterNamespace)
				})
			})

			When("a snapshot is taken while the previous one is still in progress", Ordered, func() {
				const (
					testNS       = "snapshot-canceling"
					snapshotPath = "container:///snapshot-data/" + testNS + ".tar.gz"
					appCount     = 3
					appPrefix    = "test-app-"
				)

				BeforeAll(func(ctx context.Context) {
					_, err := vClusterClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{Name: testNS},
					}, metav1.CreateOptions{})
					Expect(err).NotTo(HaveOccurred())
					for i := range appCount {
						appName := fmt.Sprintf("%s%d", appPrefix, i)
						createAppWithPVC(ctx, vClusterClient, testNS, appName)
					}
					Eventually(func(g Gomega) {
						for i := range appCount {
							appName := fmt.Sprintf("%s%d", appPrefix, i)
							dep, err := vClusterClient.AppsV1().Deployments(testNS).Get(ctx, appName, metav1.GetOptions{})
							g.Expect(err).NotTo(HaveOccurred())
							g.Expect(dep.Status.AvailableReplicas).To(Equal(int32(1)),
								"deployment %s not available: %s", appName, toJSON(dep))
						}
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())

					createSnapshot(vClusterName, vClusterNamespace, true, snapshotPath, true)
					time.Sleep(time.Second)
					createSnapshot(vClusterName, vClusterNamespace, true, snapshotPath, true)
				})

				// Spec 1 depends on BeforeAll
				It("Has 2 snapshot requests", func(ctx context.Context) {
					Eventually(func(g Gomega) {
						cms, err := hostClient.CoreV1().ConfigMaps(vClusterNamespace).List(ctx, metav1.ListOptions{
							LabelSelector: pkgconstants.SnapshotRequestLabel,
						})
						g.Expect(err).NotTo(HaveOccurred())
						g.Expect(cms.Items).To(HaveLen(2))
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
				})

				// Spec 2 depends on spec 1
				It("Canceled the previous snapshot request", func(ctx context.Context) {
					vClusterRelease, err := helm.NewSecrets(hostClient).Get(ctx, vClusterName, vClusterNamespace)
					Expect(err).NotTo(HaveOccurred())
					vConfigValues, err := yaml.Marshal(vClusterRelease.Config)
					Expect(err).NotTo(HaveOccurred())
					vClusterConfig, err := vclusterconfig.ParseConfigBytes(vConfigValues, vClusterName, nil)
					Expect(err).NotTo(HaveOccurred())

					var restConfig *rest.Config
					var volumeSnapshotsNS string
					if vClusterConfig.PrivateNodes.Enabled {
						currentClusterName := cluster.CurrentClusterNameFrom(ctx)
						restConfig = cluster.From(ctx, currentClusterName).KubernetesRestConfig()
						volumeSnapshotsNS = testNS
					} else {
						restConfig = cluster.From(ctx, constants.GetHostClusterName()).KubernetesRestConfig()
						volumeSnapshotsNS = vClusterNamespace
					}
					snapshotClient, err := snapshotsv1.NewForConfig(restConfig)
					Expect(err).NotTo(HaveOccurred())

					Eventually(func(g Gomega) {
						previousReq, _ := getTwoSnapshotRequests(g, ctx, hostClient, vClusterNamespace)
						for pvcName, vsStatus := range previousReq.Status.VolumeSnapshots.Snapshots {
							pvcParts := strings.Split(pvcName, "/")
							g.Expect(pvcParts).To(HaveLen(2))
							vsName := fmt.Sprintf("%s-%s", pvcParts[1], previousReq.Name)
							_, err := snapshotClient.SnapshotV1().VolumeSnapshots(volumeSnapshotsNS).Get(ctx, vsName, metav1.GetOptions{})
							g.Expect(kerrors.IsNotFound(err)).To(BeTrue(),
								"canceled VolumeSnapshot %s should be deleted", vsName)
							g.Expect(vsStatus.Phase).To(Equal(volumes.RequestPhaseCanceled),
								"volume snapshot for PVC %s should be canceled", pvcName)
						}
						g.Expect(previousReq.Status.VolumeSnapshots.Phase).To(Equal(volumes.RequestPhaseCanceled))
						g.Expect(previousReq.Status.Phase).To(Equal(snapshot.RequestPhaseCanceled))
					}).WithPolling(constants.PollingInterval).WithTimeout(5 * time.Minute).Should(Succeed())
				})

				// Spec 3 depends on spec 1
				It("Completed the new snapshot request", func(ctx context.Context) {
					Eventually(func(g Gomega) {
						_, newerReq := getTwoSnapshotRequests(g, ctx, hostClient, vClusterNamespace)
						for pvcName, vs := range newerReq.Status.VolumeSnapshots.Snapshots {
							g.Expect(vs.Phase).To(Equal(volumes.RequestPhaseCompleted),
								"volume snapshot for PVC %s not completed: %s", pvcName, toJSON(vs))
						}
						g.Expect(newerReq.Status.VolumeSnapshots.Phase).To(Equal(volumes.RequestPhaseCompleted))
						g.Expect(newerReq.Status.Phase).To(Equal(snapshot.RequestPhaseCompleted))
					}).WithPolling(constants.PollingInterval).WithTimeout(5 * time.Minute).Should(Succeed())
				})

				AfterAll(func(ctx context.Context) {
					_ = vClusterClient.CoreV1().Namespaces().Delete(ctx, testNS, metav1.DeleteOptions{})
					deleteSnapshotRequestConfigMaps(ctx, hostClient, vClusterNamespace)
				})
			})

			When("a snapshot is deleted", Ordered, func() {
				const (
					testNS                    = "snapshot-deleting"
					snapshotPath              = "container:///snapshot-data/" + testNS + ".tar.gz"
					appCount                  = 3
					appPrefix                 = "test-app-"
					deleteSnapshotRequestName = "delete-request-" + testNS
				)

				BeforeAll(func(ctx context.Context) {
					_, err := vClusterClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{Name: testNS},
					}, metav1.CreateOptions{})
					Expect(err).NotTo(HaveOccurred())
					for i := range appCount {
						appName := fmt.Sprintf("%s%d", appPrefix, i)
						createAppWithPVC(ctx, vClusterClient, testNS, appName)
					}
					Eventually(func(g Gomega) {
						for i := range appCount {
							appName := fmt.Sprintf("%s%d", appPrefix, i)
							dep, err := vClusterClient.AppsV1().Deployments(testNS).Get(ctx, appName, metav1.GetOptions{})
							g.Expect(err).NotTo(HaveOccurred())
							g.Expect(dep.Status.AvailableReplicas).To(Equal(int32(1)),
								"deployment %s not available: %s", appName, toJSON(dep))
						}
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())

					createSnapshot(vClusterName, vClusterNamespace, true, snapshotPath, true)
				})

				// Spec 1 depends on BeforeAll: create snapshot deletion request
				It("Creates snapshot deletion request", func(ctx context.Context) {
					listOptions := metav1.ListOptions{LabelSelector: pkgconstants.SnapshotRequestLabel}

					var snapshotOptions *snapshot.Options
					Eventually(func(g Gomega) {
						secrets, err := hostClient.CoreV1().Secrets(vClusterNamespace).List(ctx, listOptions)
						g.Expect(err).NotTo(HaveOccurred())
						g.Expect(secrets.Items).To(HaveLen(1))
						snapshotOptions, err = snapshot.UnmarshalSnapshotOptions(&secrets.Items[0])
						g.Expect(err).NotTo(HaveOccurred())
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())

					waitForSnapshotToBeCreated(ctx, hostClient, vClusterNamespace)

					snapshotRequestCMs, err := hostClient.CoreV1().ConfigMaps(vClusterNamespace).List(ctx, listOptions)
					Expect(err).NotTo(HaveOccurred())
					Expect(snapshotRequestCMs.Items).To(HaveLen(1))
					snapshotRequest, err := snapshot.UnmarshalSnapshotRequest(&snapshotRequestCMs.Items[0])
					Expect(err).NotTo(HaveOccurred())

					snapshotRequest.Name = deleteSnapshotRequestName
					snapshotRequest.CreationTimestamp = metav1.Now()
					snapshotRequest.Status.Phase = snapshot.RequestPhaseDeleting

					deleteCM, err := snapshot.CreateSnapshotRequestConfigMap(vClusterNamespace, vClusterName, snapshotRequest)
					Expect(err).NotTo(HaveOccurred())
					deleteCM.Name = deleteSnapshotRequestName

					deleteSecret, err := snapshot.CreateSnapshotOptionsSecret(
						pkgconstants.SnapshotRequestLabel, vClusterNamespace, vClusterName, snapshotOptions)
					Expect(err).NotTo(HaveOccurred())
					deleteSecret.Name = deleteSnapshotRequestName

					_, err = hostClient.CoreV1().Secrets(vClusterNamespace).Create(ctx, deleteSecret, metav1.CreateOptions{})
					Expect(err).NotTo(HaveOccurred())
					_, err = hostClient.CoreV1().ConfigMaps(vClusterNamespace).Create(ctx, deleteCM, metav1.CreateOptions{})
					Expect(err).NotTo(HaveOccurred())
				})

				// Spec 2 depends on spec 1
				It("Has deleted the snapshot", func(ctx context.Context) {
					Eventually(func(g Gomega) {
						cm, err := hostClient.CoreV1().ConfigMaps(vClusterNamespace).Get(ctx, deleteSnapshotRequestName, metav1.GetOptions{})
						g.Expect(err).NotTo(HaveOccurred())
						req, err := snapshot.UnmarshalSnapshotRequest(cm)
						g.Expect(err).NotTo(HaveOccurred())
						g.Expect(req.Status.Phase).To(Equal(snapshot.RequestPhaseDeleted),
							"snapshot request %s not deleted: %s", req.Name, toJSON(req))
						g.Expect(req.Status.VolumeSnapshots.Phase).To(Equal(volumes.RequestPhaseDeleted))
						for pvcName, vs := range req.Status.VolumeSnapshots.Snapshots {
							g.Expect(vs.Phase).To(Equal(volumes.RequestPhaseDeleted),
								"volume snapshot for PVC %s not deleted: %s", pvcName, toJSON(vs))
						}
					}).WithPolling(constants.PollingInterval).WithTimeout(5 * time.Minute).Should(Succeed())
				})

				AfterAll(func(ctx context.Context) {
					_ = vClusterClient.CoreV1().Namespaces().Delete(ctx, testNS, metav1.DeleteOptions{})
				})
			})
		})
}

// --- Volume helpers ---

func createAppWithPVC(ctx context.Context, client kubernetes.Interface, namespace, name string) {
	GinkgoHelper()
	createPVC(ctx, client, namespace, name)
	createDeploymentWithVolume(ctx, client, namespace, name, name)
}

func createPVC(ctx context.Context, client kubernetes.Interface, namespace, name string) {
	GinkgoHelper()
	_, err := client.CoreV1().PersistentVolumeClaims(namespace).Create(ctx, &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{corev1.ResourceStorage: resource.MustParse("1Gi")},
			},
			StorageClassName: ptr.To("csi-hostpath-sc"),
		},
	}, metav1.CreateOptions{})
	Expect(err).NotTo(HaveOccurred())
}

func createDeploymentWithVolume(ctx context.Context, client kubernetes.Interface, namespace, deploymentName, pvcName string) {
	GinkgoHelper()
	_, err := client.AppsV1().Deployments(namespace).Create(ctx, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: deploymentName, Namespace: namespace},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptr.To(int32(1)),
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"snapshot-test-app": deploymentName}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"snapshot-test-app": deploymentName}},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name: "snapshot-test-app", Image: "busybox",
						Command:      []string{"sleep", "1000000"},
						VolumeMounts: []corev1.VolumeMount{{Name: "data", MountPath: "/data"}},
					}},
					Volumes: []corev1.Volume{{
						Name: "data",
						VolumeSource: corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: pvcName},
						},
					}},
				},
			},
		},
	}, metav1.CreateOptions{})
	Expect(err).NotTo(HaveOccurred())
}

func createPVCWithData(ctx context.Context, client kubernetes.Interface, pvcNamespace, pvcName, testFileName, testData string) {
	GinkgoHelper()
	createPVC(ctx, client, pvcNamespace, pvcName)
	deployJob(ctx, client, pvcNamespace, "write-test-data", pvcName,
		fmt.Sprintf("echo '%s' > /data/%s", testData, testFileName), testFileName)
}

func checkPVCData(ctx context.Context, client kubernetes.Interface, pvcNamespace, pvcName, testFileName, testData string) {
	GinkgoHelper()
	script := fmt.Sprintf(`actual=$(cat "/data/%s"); expected=%q;
if [ "$actual" = "$expected" ]; then
  echo "OK: content matches";
else
  echo "FAIL: expected [$expected], got [$actual]" >&2;
  exit 1;
fi`, testFileName, testData)
	deployJob(ctx, client, pvcNamespace, "check-test-data", pvcName, script, testFileName)
}

func deployJob(ctx context.Context, client kubernetes.Interface, jobNamespace, jobName, pvcName, command, testFile string) {
	GinkgoHelper()
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{GenerateName: fmt.Sprintf("%s-", jobName), Namespace: jobNamespace},
		Spec: batchv1.JobSpec{
			BackoffLimit: ptr.To(int32(0)),
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{{
						Name: "test-job", Image: "busybox:1.36",
						Command:      []string{"sh", "-c", command},
						VolumeMounts: []corev1.VolumeMount{{Name: "data", MountPath: "/data"}},
						WorkingDir:   "/data",
						ReadinessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								Exec: &corev1.ExecAction{Command: []string{"sh", "-c", "test -f /data/" + testFile}},
							},
							InitialDelaySeconds: 1, PeriodSeconds: 1, FailureThreshold: 10, TimeoutSeconds: 2,
						},
					}},
					Volumes: []corev1.Volume{{
						Name: "data",
						VolumeSource: corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: pvcName},
						},
					}},
				},
			},
		},
	}
	job, err := client.BatchV1().Jobs(jobNamespace).Create(ctx, job, metav1.CreateOptions{})
	Expect(err).NotTo(HaveOccurred())

	Eventually(func(g Gomega) {
		j, err := client.BatchV1().Jobs(jobNamespace).Get(ctx, job.Name, metav1.GetOptions{})
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(j.Status.Succeeded).To(Equal(int32(1)),
			"job %s/%s did not succeed: %s", jobNamespace, job.Name, toJSON(j))
	}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())

	// Cleanup the job
	err = client.BatchV1().Jobs(jobNamespace).Delete(ctx, job.Name, metav1.DeleteOptions{
		PropagationPolicy: ptr.To(metav1.DeletePropagationBackground),
	})
	Expect(err).NotTo(HaveOccurred())
	Eventually(func(g Gomega) {
		_, err := client.BatchV1().Jobs(jobNamespace).Get(ctx, job.Name, metav1.GetOptions{})
		g.Expect(kerrors.IsNotFound(err)).To(BeTrue(), "job %s not yet deleted", job.Name)
	}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
}

func deletePVC(ctx context.Context, vClusterClient, _ kubernetes.Interface, _, _, pvcNamespace, pvcName string) {
	GinkgoHelper()
	err := vClusterClient.CoreV1().PersistentVolumeClaims(pvcNamespace).Delete(ctx, pvcName, metav1.DeleteOptions{})
	if kerrors.IsNotFound(err) {
		return
	}
	Expect(err).NotTo(HaveOccurred())

	Eventually(func(g Gomega) {
		_, err := vClusterClient.CoreV1().PersistentVolumeClaims(pvcNamespace).Get(ctx, pvcName, metav1.GetOptions{})
		g.Expect(kerrors.IsNotFound(err)).To(BeTrue(), "PVC %s/%s not yet deleted", pvcNamespace, pvcName)
	}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
}

func getTwoSnapshotRequests(g Gomega, ctx context.Context, hostClient kubernetes.Interface, vClusterNamespace string) (*snapshot.Request, *snapshot.Request) {
	configMaps, err := hostClient.CoreV1().ConfigMaps(vClusterNamespace).List(ctx, metav1.ListOptions{
		LabelSelector: pkgconstants.SnapshotRequestLabel,
	})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(configMaps.Items).To(HaveLen(2))

	var previousCM, newerCM corev1.ConfigMap
	if configMaps.Items[0].CreationTimestamp.Time.Before(configMaps.Items[1].CreationTimestamp.Time) {
		previousCM = configMaps.Items[0]
		newerCM = configMaps.Items[1]
	} else {
		previousCM = configMaps.Items[1]
		newerCM = configMaps.Items[0]
	}
	previous, err := snapshot.UnmarshalSnapshotRequest(&previousCM)
	g.Expect(err).NotTo(HaveOccurred())
	newer, err := snapshot.UnmarshalSnapshotRequest(&newerCM)
	g.Expect(err).NotTo(HaveOccurred())

	return previous, newer
}
