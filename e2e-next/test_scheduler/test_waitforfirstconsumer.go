package test_scheduler

import (
	"context"
	"fmt"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	"github.com/loft-sh/vcluster/pkg/platform/random"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/ptr"
)

var _ = Describe("WaitForFirstConsumer StatefulSet scheduling",
	labels.Scheduler,
	labels.Storage,
	cluster.Use(clusters.SchedulerVCluster),
	cluster.Use(clusters.HostCluster),
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

		It("creates a StatefulSet with WaitForFirstConsumer PVC and becomes ready", func(ctx context.Context) {
			suffix := random.String(6)
			stsName := "scheduler-sts-" + suffix
			nsName := "default"

			By("Discovering a WaitForFirstConsumer StorageClass on the host", func() {
				scList, err := hostClient.StorageV1().StorageClasses().List(ctx, metav1.ListOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(scList.Items).NotTo(BeEmpty(), "Expected at least one StorageClass on host")

				// Default to the first StorageClass; prefer WaitForFirstConsumer, especially if it's the default
				scToUse := scList.Items[0]
				for _, sc := range scList.Items {
					if sc.VolumeBindingMode != nil && *sc.VolumeBindingMode == storagev1.VolumeBindingWaitForFirstConsumer {
						scToUse = sc
						if sc.Annotations["storageclass.kubernetes.io/is-default-class"] == "true" {
							break
						}
					}
				}

				GinkgoWriter.Printf("Using StorageClass %q\n", scToUse.Name)

				// If no WaitForFirstConsumer StorageClass exists, create one from a copy
				if scToUse.VolumeBindingMode == nil || *scToUse.VolumeBindingMode != storagev1.VolumeBindingWaitForFirstConsumer {
					GinkgoWriter.Printf("Creating a WaitForFirstConsumer copy of %q\n", scToUse.Name)
					newSC := scToUse.DeepCopy()
					translate.ResetObjectMetadata(&newSC.ObjectMeta)
					newSC.Name = "scheduler-test-sc-" + suffix
					wffc := storagev1.VolumeBindingWaitForFirstConsumer
					newSC.VolumeBindingMode = &wffc

					_, err = hostClient.StorageV1().StorageClasses().Create(ctx, newSC, metav1.CreateOptions{})
					Expect(err).NotTo(HaveOccurred())
					DeferCleanup(func(ctx context.Context) {
						err := hostClient.StorageV1().StorageClasses().Delete(ctx, newSC.Name, metav1.DeleteOptions{})
						if !kerrors.IsNotFound(err) {
							Expect(err).NotTo(HaveOccurred())
						}
					})

					scToUse = *newSC
				}

				By("Creating a StatefulSet with a PVC template in the vcluster", func() {
					sts := &appsv1.StatefulSet{
						ObjectMeta: metav1.ObjectMeta{
							Name:      stsName,
							Namespace: nsName,
						},
						Spec: appsv1.StatefulSetSpec{
							Replicas: ptr.To[int32](1),
							Selector: &metav1.LabelSelector{
								MatchLabels: map[string]string{"app": "scheduler-test-" + suffix},
							},
							VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
								{
									ObjectMeta: metav1.ObjectMeta{Name: "data"},
									Spec: corev1.PersistentVolumeClaimSpec{
										AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
										StorageClassName: &scToUse.Name,
										Resources: corev1.VolumeResourceRequirements{
											Requests: corev1.ResourceList{
												corev1.ResourceStorage: resource.MustParse("5Gi"),
											},
										},
									},
								},
							},
							Template: corev1.PodTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Labels: map[string]string{"app": "scheduler-test-" + suffix},
								},
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										{
											Name:  "nginx",
											Image: "nginx",
											VolumeMounts: []corev1.VolumeMount{
												{
													Name:      "data",
													MountPath: "/data",
												},
											},
										},
									},
								},
							},
						},
					}

					_, err := vClusterClient.AppsV1().StatefulSets(nsName).Create(ctx, sts, metav1.CreateOptions{})
					Expect(err).NotTo(HaveOccurred())
					DeferCleanup(func(ctx context.Context) {
						err := vClusterClient.AppsV1().StatefulSets(nsName).Delete(ctx, stsName, metav1.DeleteOptions{})
						if !kerrors.IsNotFound(err) {
							Expect(err).NotTo(HaveOccurred())
						}
					})
				})

				By("Waiting for the StatefulSet to have 1 ready replica", func() {
					Eventually(func(g Gomega) {
						ss, err := vClusterClient.AppsV1().StatefulSets(nsName).Get(ctx, stsName, metav1.GetOptions{})
						g.Expect(err).NotTo(HaveOccurred(), "Failed to get StatefulSet")
						g.Expect(ss.Status.ReadyReplicas).To(Equal(int32(1)),
							fmt.Sprintf("Expected 1 ready replica, got %d (current: %d, updated: %d)",
								ss.Status.ReadyReplicas, ss.Status.CurrentReplicas, ss.Status.UpdatedReplicas))
					}).WithPolling(constants.PollingInterval).
						WithTimeout(constants.PollingTimeoutLong).
						Should(Succeed())
				})
			})
		})
	},
)
