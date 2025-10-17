package snapshot

import (
	"context"
	"strconv"
	"time"

	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/test/framework"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

var _ = Describe("snapshot and restore", Ordered, func() {
	var (
		f                        *framework.Framework
		vClusterDefaultNamespace string
		configMapToRestore       *corev1.ConfigMap
		configMapToDelete        *corev1.ConfigMap
		secretToRestore          *corev1.Secret
		secretToDelete           *corev1.Secret
		deploymentToRestore      *appsv1.Deployment
		serviceToRestore         *corev1.Service
		pvc                      *corev1.PersistentVolumeClaim
	)

	deployTestNamespace := func(testNamespace string) {
		f = framework.DefaultFramework
		vClusterDefaultNamespace = f.VClusterNamespace

		testNamespaceObj := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: testNamespace,
			},
		}

		// create the test namespace
		_, err := f.VClusterClient.CoreV1().Namespaces().Create(f.Context, testNamespaceObj, metav1.CreateOptions{})
		framework.ExpectNoError(err)
	}

	deployTestResources := func(testNamespace string, useNewCommand bool) {
		f = framework.DefaultFramework
		vClusterDefaultNamespace = f.VClusterNamespace

		configMapToRestore = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "configmap-restore",
				Namespace: testNamespace,
				Labels: map[string]string{
					"snapshot": "restore",
				},
			},
			Data: map[string]string{
				"somekey": "somevalue",
			},
		}

		configMapToDelete = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "configmap-delete",
				Namespace: testNamespace,
				Labels: map[string]string{
					"snapshot": "delete",
				},
			},
			Data: map[string]string{
				"somesome": "somevalue",
			},
		}

		secretToRestore = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "secret-restore",
				Namespace: testNamespace,
				Labels: map[string]string{
					"snapshot": "restore",
				},
			},
			Data: map[string][]byte{
				"BOO_BAR": []byte("hello-world"),
			},
		}

		secretToDelete = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "secret-delete",
				Namespace: testNamespace,
				Labels: map[string]string{
					"snapshot": "delete",
				},
			},
			Data: map[string][]byte{
				"ANOTHER_ENV": []byte("another-hello-world"),
			},
		}

		deploymentToRestore = &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "deployment-restore",
				Namespace: testNamespace,
				Labels:    map[string]string{"snapshot": "restore"},
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: ptr.To(int32(1)),
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"snapshot": "restore",
					},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"snapshot": "restore",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "example-container",
								Image: "nginx:1.25.0",
								Ports: []corev1.ContainerPort{
									{
										ContainerPort: 80,
									},
								},
							},
						},
					},
				},
			},
		}

		serviceToRestore = &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "snapshot-restore",
				Namespace: testNamespace,
				Labels: map[string]string{
					"snapshot": "restore",
				},
			},
			Spec: corev1.ServiceSpec{
				Ports: []corev1.ServicePort{
					{
						Name: "https",
						Port: 443,
					},
				},
				Type: corev1.ServiceTypeClusterIP,
			},
		}

		if !useNewCommand {
			pvc = &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "snapshot-pvc",
					Namespace: vClusterDefaultNamespace,
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("5Gi"),
						},
					},
				},
			}
		}

		pods, err := f.HostClient.CoreV1().Pods(vClusterDefaultNamespace).List(f.Context, metav1.ListOptions{
			LabelSelector: "app=vcluster",
		})
		framework.ExpectNoError(err)
		framework.ExpectEqual(true, len(pods.Items) > 0)

		By("Create test resources")
		if !useNewCommand {
			_, err = f.HostClient.CoreV1().PersistentVolumeClaims(pvc.Namespace).Create(f.Context, pvc, metav1.CreateOptions{})
			framework.ExpectNoError(err)
		}

		// now create a service that should be there when we restore again
		_, err = f.VClusterClient.CoreV1().Services(testNamespace).Create(f.Context, serviceToRestore, metav1.CreateOptions{})
		framework.ExpectNoError(err)

		// now create a configmap that should be there when we restore again
		_, err = f.VClusterClient.CoreV1().ConfigMaps(testNamespace).Create(f.Context, configMapToRestore, metav1.CreateOptions{})
		framework.ExpectNoError(err)

		// now create a secret that should be there when we restore again
		_, err = f.VClusterClient.CoreV1().Secrets(testNamespace).Create(f.Context, secretToRestore, metav1.CreateOptions{})
		framework.ExpectNoError(err)

		// now create a deployment that should be there when we restore again
		_, err = f.VClusterClient.AppsV1().Deployments(testNamespace).Create(f.Context, deploymentToRestore, metav1.CreateOptions{})
		framework.ExpectNoError(err)
	}

	cleanUpTestResources := func(useNewCommand bool, snapshotTestNamespace string) {
		if !useNewCommand {
			// delete the snapshot pvc
			err := f.HostClient.CoreV1().PersistentVolumeClaims(pvc.Namespace).Delete(f.Context, pvc.Name, metav1.DeleteOptions{})
			framework.ExpectNoError(err)
		}

		Eventually(func(ctx context.Context) error {
			// create namespace
			_, err := f.VClusterClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: snapshotTestNamespace,
				},
			}, metav1.CreateOptions{})
			if err != nil && !kerrors.IsAlreadyExists(err) {
				return err
			}

			// wait until the default service account gets created
			_, err = f.VClusterClient.CoreV1().ServiceAccounts(snapshotTestNamespace).Get(ctx, "default", metav1.GetOptions{})
			return err
		}).WithPolling(time.Second).
			WithTimeout(framework.PollTimeout).
			Should(Succeed())

		// delete the namespace
		err := f.VClusterClient.CoreV1().Namespaces().Delete(f.Context, snapshotTestNamespace, metav1.DeleteOptions{})
		framework.ExpectNoError(err)

		// delete snapshot request config maps
		deleteOptions := metav1.DeleteOptions{}
		listOptions := metav1.ListOptions{
			LabelSelector: constants.SnapshotRequestLabel,
		}
		err = f.HostClient.CoreV1().ConfigMaps(f.VClusterNamespace).DeleteCollection(f.Context, deleteOptions, listOptions)
		framework.ExpectNoError(err)
	}

	checkTestResources := func(testNamespaceName string, useNewCommand bool, snapshotPath string) {
		It("Verify if only the resources in snapshot are available in vCluster after restore", func() {
			// now create a configmap that should be deleted by restore
			_, err := f.VClusterClient.CoreV1().ConfigMaps(testNamespaceName).Create(f.Context, configMapToDelete, metav1.CreateOptions{})
			framework.ExpectNoError(err)

			// now create a secret that should be deleted by restore
			_, err = f.VClusterClient.CoreV1().Secrets(testNamespaceName).Create(f.Context, secretToDelete, metav1.CreateOptions{})
			framework.ExpectNoError(err)

			// now create a service that should be deleted by restore
			serviceToDelete, err := f.VClusterClient.CoreV1().Services(testNamespaceName).Create(f.Context, &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "snapshot-delete",
					Namespace: testNamespaceName,
					Labels: map[string]string{
						"snapshot": "delete",
					},
				},
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{
						{
							Name: "http",
							Port: 80,
						},
					},
					Type: corev1.ServiceTypeClusterIP,
				},
			}, metav1.CreateOptions{})
			framework.ExpectNoError(err)

			// now restore the vCluster
			restoreVCluster(f, snapshotPath, useNewCommand, false)

			// Check configmap created before snapshot is available
			configmaps, err := f.VClusterClient.CoreV1().ConfigMaps(testNamespaceName).List(f.Context, metav1.ListOptions{
				LabelSelector: "snapshot=restore",
			})

			Expect(configmaps.Items).To(HaveLen(1))
			restoredConfigmap := configmaps.Items[0]
			Expect(restoredConfigmap.Data).To(Equal(configMapToRestore.Data))
			framework.ExpectNoError(err)

			// make sure the new resourceVersion is bigger than the latest old one
			newResourceVersion, err := strconv.ParseInt(restoredConfigmap.ResourceVersion, 10, 64)
			framework.ExpectNoError(err)
			oldResourceVersion, err := strconv.ParseInt(serviceToDelete.ResourceVersion, 10, 64)
			framework.ExpectNoError(err)
			Expect(newResourceVersion).To(BeNumerically(">", oldResourceVersion))

			// Check secret created before snapshot is available
			secrets, err := f.VClusterClient.CoreV1().Secrets(testNamespaceName).List(f.Context, metav1.ListOptions{
				LabelSelector: "snapshot=restore",
			})

			Expect(secrets.Items).To(HaveLen(1))
			restoredSecret := secrets.Items[0]
			Expect(restoredSecret.Data).To(Equal(secretToRestore.Data))
			framework.ExpectNoError(err)

			// Check deployment created before snapshot is available
			deployment, err := f.VClusterClient.AppsV1().Deployments(testNamespaceName).List(f.Context, metav1.ListOptions{
				LabelSelector: "snapshot=restore",
			})

			Expect(deployment.Items).To(HaveLen(1))
			framework.ExpectNoError(err)

			// Check configmap created after snapshot is not available
			Eventually(func(g Gomega, ctx context.Context) []corev1.ConfigMap {
				configmaps, err := f.VClusterClient.CoreV1().ConfigMaps(testNamespaceName).List(ctx, metav1.ListOptions{
					LabelSelector: "snapshot=delete",
				})
				g.Expect(err).NotTo(HaveOccurred())
				return configmaps.Items
			}).WithPolling(time.Second).
				WithTimeout(framework.PollTimeout).
				Should(BeEmpty())

			// Check secret created after snapshot is not available
			Eventually(func(g Gomega) []corev1.Secret {
				secrets, err := f.VClusterClient.CoreV1().Secrets(testNamespaceName).List(f.Context, metav1.ListOptions{
					LabelSelector: "snapshot=delete",
				})
				g.Expect(err).NotTo(HaveOccurred())
				return secrets.Items
			}).WithPolling(time.Second).
				WithTimeout(framework.PollTimeout).
				Should(BeEmpty())

			//Check service created after snapshot is not available
			Eventually(func(g Gomega) []corev1.Service {
				serviceList, err := f.VClusterClient.CoreV1().Services(testNamespaceName).List(f.Context, metav1.ListOptions{
					LabelSelector: "snapshot=delete",
				})
				g.Expect(err).NotTo(HaveOccurred())
				return serviceList.Items
			}).WithPolling(time.Second).
				WithTimeout(framework.PollTimeout * 2).
				Should(BeEmpty())
		})

		It("Verify if deleted resources are recreated in vCluster after restore", func() {

			By("Delete resources that going to be restored")
			err := f.VClusterClient.CoreV1().ConfigMaps(testNamespaceName).Delete(f.Context, configMapToRestore.Name, metav1.DeleteOptions{})
			framework.ExpectNoError(err)

			// check configmap is deleted
			Eventually(func() error {
				_, err := f.VClusterClient.CoreV1().ConfigMaps(testNamespaceName).List(f.Context, metav1.ListOptions{
					LabelSelector: "snapshot=restore",
				})

				if err != nil {
					return nil
				}
				return err
			}).WithPolling(time.Second).
				WithTimeout(framework.PollTimeout).
				Should(Succeed())

			err = f.VClusterClient.CoreV1().Secrets(testNamespaceName).Delete(f.Context, secretToRestore.Name, metav1.DeleteOptions{})
			framework.ExpectNoError(err)

			// check secret is deleted
			Eventually(func() error {
				_, err := f.VClusterClient.CoreV1().Secrets(testNamespaceName).List(f.Context, metav1.ListOptions{
					LabelSelector: "snapshot=restore",
				})

				if err != nil {
					return nil
				}
				return err
			}).WithPolling(time.Second).
				WithTimeout(framework.PollTimeout).
				Should(Succeed())

			err = f.VClusterClient.AppsV1().Deployments(testNamespaceName).Delete(f.Context, deploymentToRestore.Name, metav1.DeleteOptions{})
			framework.ExpectNoError(err)

			// check deployment is deleted
			Eventually(func() error {
				_, err := f.VClusterClient.CoreV1().Secrets(testNamespaceName).List(f.Context, metav1.ListOptions{
					LabelSelector: "snapshot=restore",
				})

				if err != nil {
					return nil
				}
				return err
			}).WithPolling(time.Second).
				WithTimeout(framework.PollTimeout).
				Should(Succeed())

			// now restore the vCluster
			restoreVCluster(f, snapshotPath, useNewCommand, false)

			By("Verify resources delete before snapshot are available")
			Eventually(func() map[string]string {
				configmaps, err := f.VClusterClient.CoreV1().ConfigMaps(testNamespaceName).List(f.Context, metav1.ListOptions{
					LabelSelector: "snapshot=restore",
				})

				if len(configmaps.Items) != 1 {
					return map[string]string{}
				}
				restoredConfigmap := configmaps.Items[0]
				framework.ExpectNoError(err)
				return restoredConfigmap.Data
			}).WithPolling(time.Second).
				WithTimeout(framework.PollTimeout).
				Should(Equal(configMapToRestore.Data))

			Eventually(func() map[string][]byte {
				secrets, err := f.VClusterClient.CoreV1().Secrets(testNamespaceName).List(f.Context, metav1.ListOptions{
					LabelSelector: "snapshot=restore",
				})

				if len(secrets.Items) != 1 {
					return map[string][]byte{}
				}
				restoredSecret := secrets.Items[0]
				framework.ExpectNoError(err)
				return restoredSecret.Data
			}).WithPolling(time.Second).
				WithTimeout(framework.PollTimeout).
				Should(Equal(secretToRestore.Data))

			Eventually(func() bool {
				deployment, err := f.VClusterClient.AppsV1().Deployments(testNamespaceName).List(f.Context, metav1.ListOptions{
					LabelSelector: "snapshot=restore",
				})

				if len(deployment.Items) != 1 {
					return false
				}
				framework.ExpectNoError(err)
				return len(deployment.Items) == 1
			}).WithPolling(time.Second).
				WithTimeout(framework.PollTimeout * 2).
				Should(BeTrue())
		})
	}

	Describe("CLI-based snapshot", Ordered, func() {
		const (
			cliTestNamespaceName = "cli-snapshot-test"
			snapshotPath         = "container:///snapshot-pvc/snapshot.tar"
		)

		BeforeAll(func() {
			deployTestNamespace(cliTestNamespaceName)
			deployTestResources(cliTestNamespaceName, false)
		})

		It("Creates the snapshot", func() {
			createSnapshot(f, false, snapshotPath, false)
		})

		checkTestResources(cliTestNamespaceName, false, snapshotPath)

		AfterAll(func() {
			cleanUpTestResources(false, "cli-snapshot-test-afterall")
		})
	})

	Describe("controller-based snapshot without volumes", Ordered, func() {
		const (
			controllerTestNamespaceName = "controller-snapshot-test"
			snapshotPath                = "container:///snapshot-data/snapshot.tar.gz"
		)

		BeforeAll(func() {
			deployTestNamespace(controllerTestNamespaceName)
			deployTestResources(controllerTestNamespaceName, true)
		})

		It("Creates the snapshot request", func() {
			createSnapshot(f, true, snapshotPath, false)
		})

		It("Creates the snapshot", func() {
			waitForSnapshotToBeCreated(f)
		})

		checkTestResources(controllerTestNamespaceName, true, snapshotPath)

		AfterAll(func() {
			cleanUpTestResources(true, "controller-snapshot-test-afterall")
		})
	})

	Describe("controller-based snapshot with volumes", Ordered, func() {
		const (
			controllerTestNamespaceName = "volume-snapshots-test"
			snapshotPath                = "container:///snapshot-data/" + controllerTestNamespaceName + ".tar.gz"
			pvcToRestoreName            = "test-pvc-restore"
			testFileName                = controllerTestNamespaceName + ".txt"
			pvcData                     = "Hello " + controllerTestNamespaceName
		)

		BeforeAll(func(ctx context.Context) {
			f = framework.DefaultFramework
			deployTestNamespace(controllerTestNamespaceName)
			createPVCWithData(ctx, f.VClusterClient, controllerTestNamespaceName, pvcToRestoreName, testFileName, pvcData)
			deployTestResources(controllerTestNamespaceName, true)
		})

		It("Creates the snapshot request", func() {
			createSnapshot(f, true, snapshotPath, true)
		})

		It("Creates the snapshot", func() {
			waitForSnapshotToBeCreated(f)
		})

		It("Deletes the PVC with test data", func(ctx context.Context) {
			deletePVC(ctx, f.VClusterClient, controllerTestNamespaceName, pvcToRestoreName)
		})

		checkTestResources(controllerTestNamespaceName, true, snapshotPath)

		It("restores vCluster with volumes", func(ctx context.Context) {
			// PVC has been restored in previous test specs, but without data, so it's stuck in Pending.
			// Therefore, delete it again, so it gets restored properly.
			deletePVC(ctx, f.VClusterClient, controllerTestNamespaceName, pvcToRestoreName)

			// now restore the vCluster
			restoreVCluster(f, snapshotPath, true, true)
		})

		It("has the restored PVC with data restored from the volume snapshot", func(ctx context.Context) {
			checkPVCData(ctx, f.VClusterClient, controllerTestNamespaceName, pvcToRestoreName, testFileName, pvcData)
		})

		AfterAll(func() {
			cleanUpTestResources(true, controllerTestNamespaceName)
		})
	})
})
