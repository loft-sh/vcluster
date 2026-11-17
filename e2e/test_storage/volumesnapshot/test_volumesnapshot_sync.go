// Package volumesnapshot covers the sync of VolumeSnapshot,
// VolumeSnapshotContent and VolumeSnapshotClass between a tenant cluster and
// the Control Plane Cluster. The tenant never runs a CSI driver itself: it
// drives the snapshot-controller and the CSI driver on the host through these
// synced objects.
package volumesnapshot

import (
	"context"
	"io"

	volumesnapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v8/apis/volumesnapshot/v1"
	snapshotclient "github.com/kubernetes-csi/external-snapshotter/client/v8/clientset/versioned"
	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e/constants"
	"github.com/loft-sh/vcluster/e2e/labels"
	"github.com/loft-sh/vcluster/pkg/util/random"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/ptr"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// Installed on the host by setup.CSIHostpathPreSetup.
	hostStorageClass  = "csi-hostpath-sc"
	hostSnapshotClass = "csi-hostpath-snapclass"

	sourcePVCName = "source-pvc"
	writerPodName = "writer"
	readerPodName = "reader"
)

// VolumeSnapshotSyncSpec registers the spec.
func VolumeSnapshotSyncSpec() {
	// Ordered: the snapshot taken by the third spec is the source of the
	// fourth, and the fifth deletes it again.
	Describe("VolumeSnapshot sync between vCluster and host",
		labels.Storage, labels.Sync, Ordered,
		func() {
			var (
				hostClient        kubernetes.Interface
				hostSnapshots     *snapshotclient.Clientset
				vClusterClient    kubernetes.Interface
				vClusterSnapshots *snapshotclient.Clientset
				vClusterCRDs      *apiextensionsclient.Clientset
				vClusterName      string
				hostNS            string
				nsName            string
				snapshotName      string
				dataMarker        string
				boundContentName  string
			)

			BeforeAll(func(ctx context.Context) {
				suffix := random.String(6)
				nsName = "volumesnapshot-sync-" + suffix
				snapshotName = "snapshot-" + suffix
				dataMarker = "volume-snapshot-" + suffix

				hostCluster := cluster.From(ctx, constants.GetHostClusterName())
				Expect(hostCluster).NotTo(BeNil())
				hostClient = cluster.KubeClientFrom(ctx, constants.GetHostClusterName())
				Expect(hostClient).NotTo(BeNil())

				var err error
				hostSnapshots, err = snapshotclient.NewForConfig(hostCluster.KubernetesRestConfig())
				Expect(err).To(Succeed())

				vClusterClient = cluster.CurrentKubeClientFrom(ctx)
				Expect(vClusterClient).NotTo(BeNil())
				vClusterName = cluster.CurrentClusterNameFrom(ctx)
				hostNS = "vcluster-" + vClusterName

				vClusterRestConfig := cluster.CurrentClusterFrom(ctx).KubernetesRestConfig()
				vClusterSnapshots, err = snapshotclient.NewForConfig(vClusterRestConfig)
				Expect(err).To(Succeed())
				vClusterCRDs, err = apiextensionsclient.NewForConfig(vClusterRestConfig)
				Expect(err).To(Succeed())

				// The snapshot is created by one spec, consumed by the next and
				// deleted by the last, so its cleanup belongs to the container:
				// a DeferCleanup registered inside the creating It would fire
				// before the restore spec ever runs.
				DeferCleanup(func(ctx context.Context) {
					err := vClusterSnapshots.SnapshotV1().VolumeSnapshots(nsName).Delete(ctx, snapshotName, metav1.DeleteOptions{})
					Expect(ctrlclient.IgnoreNotFound(err)).To(Succeed())
				})

				By("Creating the test namespace", func() {
					_, err := vClusterClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{Name: nsName},
					}, metav1.CreateOptions{})
					Expect(err).To(Succeed())
				})
				DeferCleanup(func(ctx context.Context) {
					err := vClusterClient.CoreV1().Namespaces().Delete(ctx, nsName, metav1.DeleteOptions{})
					Expect(ctrlclient.IgnoreNotFound(err)).To(Succeed())
				})

				By("Creating the source PVC and writing data into it", func() {
					_, err := vClusterClient.CoreV1().PersistentVolumeClaims(nsName).Create(ctx,
						pvcWithStorageClass(sourcePVCName, nil), metav1.CreateOptions{})
					Expect(err).To(Succeed())
					DeferCleanup(func(ctx context.Context) {
						err := vClusterClient.CoreV1().PersistentVolumeClaims(nsName).Delete(ctx, sourcePVCName, metav1.DeleteOptions{})
						Expect(ctrlclient.IgnoreNotFound(err)).To(Succeed())
					})

					// The snapshot has to capture data that is actually on the
					// volume, so write a marker through a pod and wait for it to
					// finish before snapshotting.
					_, err = vClusterClient.CoreV1().Pods(nsName).Create(ctx, &corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{Name: writerPodName},
						Spec: corev1.PodSpec{
							RestartPolicy: corev1.RestartPolicyNever,
							Containers: []corev1.Container{
								{
									Name:    "writer",
									Image:   "busybox",
									Command: []string{"sh", "-c", "echo " + dataMarker + " > /data/test.txt"},
									VolumeMounts: []corev1.VolumeMount{
										{Name: "data", MountPath: "/data"},
									},
								},
							},
							Volumes: []corev1.Volume{
								{
									Name: "data",
									VolumeSource: corev1.VolumeSource{
										PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
											ClaimName: sourcePVCName,
										},
									},
								},
							},
						},
					}, metav1.CreateOptions{})
					Expect(err).To(Succeed())
					DeferCleanup(func(ctx context.Context) {
						err := vClusterClient.CoreV1().Pods(nsName).Delete(ctx, writerPodName, metav1.DeleteOptions{})
						Expect(ctrlclient.IgnoreNotFound(err)).To(Succeed())
					})

					waitForPodPhase(ctx, vClusterClient, nsName, writerPodName, corev1.PodSucceeded)
				})

				By("Waiting for the source PVC to be Bound", func() {
					Eventually(func(g Gomega) {
						pvc, err := vClusterClient.CoreV1().PersistentVolumeClaims(nsName).Get(ctx, sourcePVCName, metav1.GetOptions{})
						g.Expect(err).To(Succeed())
						g.Expect(pvc.Status.Phase).To(Equal(corev1.ClaimBound),
							"PVC phase is %s, not yet Bound", pvc.Status.Phase)
					}).WithContext(ctx).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutVeryLong).Should(Succeed())
				})
			})

			It("installs the VolumeSnapshot CRDs in the vCluster", func(ctx context.Context) {
				// The repro in ENGCP-1411 is that these CRDs are missing, so the
				// tenant has no way to create a snapshot at all. The mappers
				// install them from the embedded copies at syncer startup.
				for _, crdName := range []string{
					"volumesnapshots.snapshot.storage.k8s.io",
					"volumesnapshotcontents.snapshot.storage.k8s.io",
					"volumesnapshotclasses.snapshot.storage.k8s.io",
				} {
					By("Checking that "+crdName+" is established", func() {
						Eventually(func(g Gomega) {
							crd, err := vClusterCRDs.ApiextensionsV1().CustomResourceDefinitions().Get(ctx, crdName, metav1.GetOptions{})
							g.Expect(err).To(Succeed())
							g.Expect(crdEstablished(crd.Status.Conditions)).To(BeTrue(),
								"CRD %s is not Established yet", crdName)
						}).WithContext(ctx).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
					})
				}

				By("Checking that the snapshot API is usable in the vCluster", func() {
					_, err := vClusterSnapshots.SnapshotV1().VolumeSnapshots(nsName).List(ctx, metav1.ListOptions{})
					Expect(err).To(Succeed())
					_, err = vClusterSnapshots.SnapshotV1().VolumeSnapshotContents().List(ctx, metav1.ListOptions{})
					Expect(err).To(Succeed())
				})
			})

			It("syncs VolumeSnapshotClasses from the host", func(ctx context.Context) {
				hostClass, err := hostSnapshots.SnapshotV1().VolumeSnapshotClasses().Get(ctx, hostSnapshotClass, metav1.GetOptions{})
				Expect(err).To(Succeed())

				By("Waiting for the host VolumeSnapshotClass to appear in the vCluster", func() {
					Eventually(func(g Gomega) {
						vClass, err := vClusterSnapshots.SnapshotV1().VolumeSnapshotClasses().Get(ctx, hostSnapshotClass, metav1.GetOptions{})
						g.Expect(err).To(Succeed())
						// Mirrored one-way: same name, same driver and policy.
						g.Expect(vClass.Driver).To(Equal(hostClass.Driver))
						g.Expect(vClass.DeletionPolicy).To(Equal(hostClass.DeletionPolicy))
						g.Expect(vClass.Parameters).To(Equal(hostClass.Parameters))
					}).WithContext(ctx).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
				})
			})

			It("syncs a VolumeSnapshot to the host and reports it ready", func(ctx context.Context) {
				By("Creating the VolumeSnapshot in the vCluster", func() {
					_, err := vClusterSnapshots.SnapshotV1().VolumeSnapshots(nsName).Create(ctx, &volumesnapshotv1.VolumeSnapshot{
						ObjectMeta: metav1.ObjectMeta{Name: snapshotName},
						Spec: volumesnapshotv1.VolumeSnapshotSpec{
							VolumeSnapshotClassName: ptr.To(hostSnapshotClass),
							Source: volumesnapshotv1.VolumeSnapshotSource{
								PersistentVolumeClaimName: ptr.To(sourcePVCName),
							},
						},
					}, metav1.CreateOptions{})
					Expect(err).To(Succeed())
				})

				By("Verifying the host VolumeSnapshot points at the synced host PVC", func() {
					hostSnapshotName := translate.SingleNamespaceHostName(snapshotName, nsName, vClusterName)
					hostPVCName := translate.SingleNamespaceHostName(sourcePVCName, nsName, vClusterName)
					Eventually(func(g Gomega) {
						pVS, err := hostSnapshots.SnapshotV1().VolumeSnapshots(hostNS).Get(ctx, hostSnapshotName, metav1.GetOptions{})
						g.Expect(err).To(Succeed())
						g.Expect(pVS.Spec.Source.PersistentVolumeClaimName).To(HaveValue(Equal(hostPVCName)),
							"host VolumeSnapshot source was not name translated")
					}).WithContext(ctx).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
				})

				By("Waiting for readyToUse to be reflected back into the vCluster", func() {
					Eventually(func(g Gomega) {
						vVS, err := vClusterSnapshots.SnapshotV1().VolumeSnapshots(nsName).Get(ctx, snapshotName, metav1.GetOptions{})
						g.Expect(err).To(Succeed())
						g.Expect(vVS.Status).NotTo(BeNil())
						g.Expect(vVS.Status.ReadyToUse).To(HaveValue(BeTrue()),
							"VolumeSnapshot is not readyToUse yet: %+v", vVS.Status)
						g.Expect(vVS.Status.BoundVolumeSnapshotContentName).NotTo(BeNil())
						boundContentName = *vVS.Status.BoundVolumeSnapshotContentName
					}).WithContext(ctx).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutVeryLong).Should(Succeed())
				})

				By("Verifying the bound VolumeSnapshotContent is visible in the vCluster", func() {
					Eventually(func(g Gomega) {
						vVSC, err := vClusterSnapshots.SnapshotV1().VolumeSnapshotContents().Get(ctx, boundContentName, metav1.GetOptions{})
						g.Expect(err).To(Succeed())
						// The ref is translated back to the tenant names.
						g.Expect(vVSC.Spec.VolumeSnapshotRef.Name).To(Equal(snapshotName))
						g.Expect(vVSC.Spec.VolumeSnapshotRef.Namespace).To(Equal(nsName))
						g.Expect(vVSC.Status).NotTo(BeNil())
						g.Expect(vVSC.Status.ReadyToUse).To(HaveValue(BeTrue()))
					}).WithContext(ctx).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
				})
			})

			It("restores a PVC from the snapshot with its data intact", func(ctx context.Context) {
				const restoredPVCName = "restored-pvc"

				By("Creating a PVC with the VolumeSnapshot as dataSource", func() {
					_, err := vClusterClient.CoreV1().PersistentVolumeClaims(nsName).Create(ctx,
						pvcWithStorageClass(restoredPVCName, &corev1.TypedLocalObjectReference{
							APIGroup: ptr.To(volumesnapshotv1.GroupName),
							Kind:     "VolumeSnapshot",
							Name:     snapshotName,
						}), metav1.CreateOptions{})
					Expect(err).To(Succeed())
					DeferCleanup(func(ctx context.Context) {
						err := vClusterClient.CoreV1().PersistentVolumeClaims(nsName).Delete(ctx, restoredPVCName, metav1.DeleteOptions{})
						Expect(ctrlclient.IgnoreNotFound(err)).To(Succeed())
					})
				})

				By("Verifying the host PVC dataSource was name translated", func() {
					// Without the VolumeSnapshot case in the PVC syncer's
					// translate the host PVC would reference the untranslated
					// tenant name and never provision.
					hostPVCName := translate.SingleNamespaceHostName(restoredPVCName, nsName, vClusterName)
					hostSnapshotName := translate.SingleNamespaceHostName(snapshotName, nsName, vClusterName)
					Eventually(func(g Gomega) {
						pPVC, err := hostClient.CoreV1().PersistentVolumeClaims(hostNS).Get(ctx, hostPVCName, metav1.GetOptions{})
						g.Expect(err).To(Succeed())
						g.Expect(pPVC.Spec.DataSource).NotTo(BeNil())
						g.Expect(pPVC.Spec.DataSource.Name).To(Equal(hostSnapshotName))
						g.Expect(pPVC.Spec.DataSourceRef).NotTo(BeNil())
						g.Expect(pPVC.Spec.DataSourceRef.Name).To(Equal(hostSnapshotName))
					}).WithContext(ctx).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
				})

				By("Reading the restored volume back through a pod", func() {
					_, err := vClusterClient.CoreV1().Pods(nsName).Create(ctx, &corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{Name: readerPodName},
						Spec: corev1.PodSpec{
							RestartPolicy: corev1.RestartPolicyNever,
							Containers: []corev1.Container{
								{
									Name:    "reader",
									Image:   "busybox",
									Command: []string{"sh", "-c", "cat /data/test.txt"},
									VolumeMounts: []corev1.VolumeMount{
										{Name: "data", MountPath: "/data"},
									},
								},
							},
							Volumes: []corev1.Volume{
								{
									Name: "data",
									VolumeSource: corev1.VolumeSource{
										PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
											ClaimName: restoredPVCName,
										},
									},
								},
							},
						},
					}, metav1.CreateOptions{})
					Expect(err).To(Succeed())
					DeferCleanup(func(ctx context.Context) {
						err := vClusterClient.CoreV1().Pods(nsName).Delete(ctx, readerPodName, metav1.DeleteOptions{})
						Expect(ctrlclient.IgnoreNotFound(err)).To(Succeed())
					})

					waitForPodPhase(ctx, vClusterClient, nsName, readerPodName, corev1.PodSucceeded)

					pvc, err := vClusterClient.CoreV1().PersistentVolumeClaims(nsName).Get(ctx, restoredPVCName, metav1.GetOptions{})
					Expect(err).To(Succeed())
					Expect(pvc.Status.Phase).To(Equal(corev1.ClaimBound))

					logs, err := podLogs(ctx, vClusterClient, nsName, readerPodName)
					Expect(err).To(Succeed())
					Expect(logs).To(ContainSubstring(dataMarker),
						"restored volume does not carry the data written before the snapshot")
				})
			})

			It("cleans up the host VolumeSnapshot and VolumeSnapshotContent on deletion", func(ctx context.Context) {
				Expect(boundContentName).NotTo(BeEmpty(), "no bound VolumeSnapshotContent recorded")

				By("Deleting the VolumeSnapshot in the vCluster", func() {
					err := vClusterSnapshots.SnapshotV1().VolumeSnapshots(nsName).Delete(ctx, snapshotName, metav1.DeleteOptions{})
					Expect(err).To(Succeed())
				})

				By("Waiting for the host VolumeSnapshot to be deleted", func() {
					hostSnapshotName := translate.SingleNamespaceHostName(snapshotName, nsName, vClusterName)
					Eventually(func(g Gomega) {
						_, err := hostSnapshots.SnapshotV1().VolumeSnapshots(hostNS).Get(ctx, hostSnapshotName, metav1.GetOptions{})
						g.Expect(kerrors.IsNotFound(err)).To(BeTrue(),
							"host VolumeSnapshot %s/%s not yet deleted", hostNS, hostSnapshotName)
					}).WithContext(ctx).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
				})

				By("Waiting for the host VolumeSnapshotContent to be deleted", func() {
					Eventually(func(g Gomega) {
						_, err := hostSnapshots.SnapshotV1().VolumeSnapshotContents().Get(ctx, boundContentName, metav1.GetOptions{})
						g.Expect(kerrors.IsNotFound(err)).To(BeTrue(),
							"host VolumeSnapshotContent %s not yet deleted", boundContentName)
					}).WithContext(ctx).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
				})

				By("Waiting for the VolumeSnapshotContent to disappear from the vCluster", func() {
					Eventually(func(g Gomega) {
						_, err := vClusterSnapshots.SnapshotV1().VolumeSnapshotContents().Get(ctx, boundContentName, metav1.GetOptions{})
						g.Expect(kerrors.IsNotFound(err)).To(BeTrue(),
							"VolumeSnapshotContent %s not yet deleted from the vCluster", boundContentName)
					}).WithContext(ctx).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
				})
			})
		},
	)
}

func pvcWithStorageClass(name string, dataSource *corev1.TypedLocalObjectReference) *corev1.PersistentVolumeClaim {
	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: corev1.PersistentVolumeClaimSpec{
			StorageClassName: ptr.To(hostStorageClass),
			AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("100Mi"),
				},
			},
			DataSource: dataSource,
		},
	}
}

func waitForPodPhase(ctx context.Context, client kubernetes.Interface, namespace, name string, phase corev1.PodPhase) {
	GinkgoHelper()
	Eventually(func(g Gomega) {
		pod, err := client.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
		g.Expect(err).To(Succeed())
		g.Expect(pod.Status.Phase).NotTo(Equal(corev1.PodFailed), "pod %s/%s failed", namespace, name)
		g.Expect(pod.Status.Phase).To(Equal(phase),
			"pod %s/%s is %s, waiting for %s", namespace, name, pod.Status.Phase, phase)
	}).WithContext(ctx).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutVeryLong).Should(Succeed())
}

func podLogs(ctx context.Context, client kubernetes.Interface, namespace, name string) (string, error) {
	stream, err := client.CoreV1().Pods(namespace).GetLogs(name, &corev1.PodLogOptions{}).Stream(ctx)
	if err != nil {
		return "", err
	}
	defer stream.Close()

	out, err := io.ReadAll(stream)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func crdEstablished(conditions []apiextensionsv1.CustomResourceDefinitionCondition) bool {
	for _, condition := range conditions {
		if condition.Type == apiextensionsv1.Established {
			return condition.Status == apiextensionsv1.ConditionTrue
		}
	}
	return false
}
