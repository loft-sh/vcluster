package scheduler

import (
	"context"
	"fmt"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	"github.com/loft-sh/vcluster/pkg/util/random"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// isDefaultStorageClass returns true if the StorageClass has the default annotation.
func isDefaultStorageClass(sc storagev1.StorageClass) bool {
	if sc.Annotations == nil {
		return false
	}
	if sc.Annotations["storageclass.kubernetes.io/is-default-class"] == "true" {
		return true
	}
	if sc.Annotations["storageclass.beta.kubernetes.io/is-default-class"] == "true" {
		return true
	}
	return false
}

func int32Ptr(i int32) *int32 {
	return &i
}

// SchedulerWaitForFirstConsumerSpec registers WaitForFirstConsumer StatefulSet tests.
func SchedulerWaitForFirstConsumerSpec() {
	Describe("Schedule a StatefulSet with WaitForFirstConsumer PVCs",
		labels.Scheduler,
		labels.Storage,
		func() {
			var (
				hostClient     kubernetes.Interface
				vClusterClient kubernetes.Interface
			)

			BeforeEach(func(ctx context.Context) {
				hostClient = cluster.KubeClientFrom(ctx, constants.GetHostClusterName())
				Expect(hostClient).NotTo(BeNil())
				vClusterClient = cluster.CurrentKubeClientFrom(ctx)
				Expect(vClusterClient).NotTo(BeNil())
			})

			It("waits for a StatefulSet with a WaitForFirstConsumer PVC to become ready", func(ctx context.Context) {
				suffix := random.String(6)
				ssName := "test-statefulset-" + suffix

				// Determine which StorageClass to use on the host.
				// The default Kind storage class is WaitForFirstConsumer.
				scList, err := hostClient.StorageV1().StorageClasses().List(ctx, metav1.ListOptions{})
				Expect(err).To(Succeed())
				Expect(scList.Items).NotTo(BeEmpty(), "expected at least one StorageClass on the host cluster")

				// Default to the first SC; prefer one with WaitForFirstConsumer, especially if default.
				scToUse := scList.Items[0]
				for _, sc := range scList.Items {
					if sc.VolumeBindingMode != nil && *sc.VolumeBindingMode == storagev1.VolumeBindingWaitForFirstConsumer {
						scToUse = sc
						if isDefaultStorageClass(sc) {
							break
						}
					}
				}

				fmt.Fprintf(GinkgoWriter, "using storageclass %q\n", scToUse.Name)

				// If no WaitForFirstConsumer SC was found, create a copy with that binding mode.
				if scToUse.VolumeBindingMode == nil || *scToUse.VolumeBindingMode != storagev1.VolumeBindingWaitForFirstConsumer {
					fmt.Fprintf(GinkgoWriter, "creating a copy of %q with WaitForFirstConsumer\n", scToUse.Name)
					newSC := &storagev1.StorageClass{
						ObjectMeta: metav1.ObjectMeta{
							Name: "test-sc-" + suffix,
						},
						Provisioner:          scToUse.Provisioner,
						Parameters:           scToUse.Parameters,
						ReclaimPolicy:        scToUse.ReclaimPolicy,
						MountOptions:         scToUse.MountOptions,
						AllowVolumeExpansion: scToUse.AllowVolumeExpansion,
					}
					waitMode := storagev1.VolumeBindingWaitForFirstConsumer
					newSC.VolumeBindingMode = &waitMode

					_, err = hostClient.StorageV1().StorageClasses().Create(ctx, newSC, metav1.CreateOptions{})
					Expect(err).To(Succeed())
					DeferCleanup(func(ctx context.Context) {
						err := hostClient.StorageV1().StorageClasses().Delete(ctx, newSC.Name, metav1.DeleteOptions{})
						if !kerrors.IsNotFound(err) {
							Expect(err).To(Succeed(), "cleaning up created storageclass %s", newSC.Name)
						}
					})
					scToUse = *newSC
				}

				// Create the StatefulSet in the vCluster.
				workload := &appsv1.StatefulSet{
					ObjectMeta: metav1.ObjectMeta{
						Name:      ssName,
						Namespace: "default",
					},
					Spec: appsv1.StatefulSetSpec{
						Replicas: int32Ptr(1),
						VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
							{
								ObjectMeta: metav1.ObjectMeta{Name: "data"},
								Spec: corev1.PersistentVolumeClaimSpec{
									AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
									Resources: corev1.VolumeResourceRequirements{
										Requests: corev1.ResourceList{
											corev1.ResourceStorage: resource.MustParse("5Gi"),
										},
									},
									StorageClassName: &scToUse.Name,
								},
							},
						},
						Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "test-" + suffix}},
						Template: corev1.PodTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{
								Labels: map[string]string{"app": "test-" + suffix},
							},
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{
									{
										Name:  "nginx",
										Image: "nginx",
										VolumeMounts: []corev1.VolumeMount{
											{Name: "data", MountPath: "/data"},
										},
									},
								},
							},
						},
					},
				}

				_, err = vClusterClient.AppsV1().StatefulSets("default").Create(ctx, workload, metav1.CreateOptions{})
				Expect(err).To(Succeed())
				DeferCleanup(func(ctx context.Context) {
					err := vClusterClient.AppsV1().StatefulSets("default").Delete(ctx, ssName, metav1.DeleteOptions{})
					if !kerrors.IsNotFound(err) {
						Expect(err).To(Succeed())
					}
					// Also clean up the PVC created by the StatefulSet volume claim template.
					pvcName := "data-" + ssName + "-0"
					err = vClusterClient.CoreV1().PersistentVolumeClaims("default").Delete(ctx, pvcName, metav1.DeleteOptions{})
					if !kerrors.IsNotFound(err) {
						Expect(err).To(Succeed())
					}
				})

				By("Waiting for the StatefulSet to have 1 ready replica", func() {
					Eventually(func(g Gomega) {
						ss, err := vClusterClient.AppsV1().StatefulSets("default").Get(ctx, ssName, metav1.GetOptions{})
						g.Expect(err).To(Succeed(), "failed to fetch statefulset %q", ssName)
						g.Expect(ss.Status.ReadyReplicas).To(Equal(int32(1)),
							"statefulset %q has %d ready replicas, waiting for 1", ssName, ss.Status.ReadyReplicas)
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
				})
			})
		},
	)
}
