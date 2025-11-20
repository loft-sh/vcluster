package snapshot

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ghodss/yaml"
	snapshotsv1 "github.com/kubernetes-csi/external-snapshotter/client/v8/clientset/versioned"
	vclusterconfig "github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/helm"
	"github.com/loft-sh/vcluster/pkg/snapshot"
	"github.com/loft-sh/vcluster/pkg/snapshot/volumes"
	"github.com/loft-sh/vcluster/test/framework"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
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

	cleanUpTestResources := func(ctx context.Context, useNewCommand bool, snapshotTestNamespace string) {
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
		}).WithContext(ctx).
			WithPolling(time.Second).
			WithTimeout(framework.PollTimeout).
			Should(Succeed())

		// delete the namespace
		err := f.VClusterClient.CoreV1().Namespaces().Delete(f.Context, snapshotTestNamespace, metav1.DeleteOptions{})
		framework.ExpectNoError(err)

		deleteSnapshotRequestConfigMaps(ctx, f)
	}

	checkTestResources := func(testNamespaceName string, useNewCommand bool, snapshotPath string) {
		It("Verify if only the resources in snapshot are available in vCluster after restore", func(ctx context.Context) {
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
			restoreVCluster(ctx, f, snapshotPath, useNewCommand, false)

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
			}).WithContext(ctx).
				WithPolling(time.Second).
				WithTimeout(framework.PollTimeout).
				Should(BeEmpty())

			// Check secret created after snapshot is not available
			Eventually(func(g Gomega, ctx context.Context) []corev1.Secret {
				secrets, err := f.VClusterClient.CoreV1().Secrets(testNamespaceName).List(ctx, metav1.ListOptions{
					LabelSelector: "snapshot=delete",
				})
				g.Expect(err).NotTo(HaveOccurred())
				return secrets.Items
			}).WithContext(ctx).
				WithPolling(time.Second).
				WithTimeout(framework.PollTimeout).
				Should(BeEmpty())

			//Check service created after snapshot is not available
			Eventually(func(g Gomega, ctx context.Context) []corev1.Service {
				serviceList, err := f.VClusterClient.CoreV1().Services(testNamespaceName).List(ctx, metav1.ListOptions{
					LabelSelector: "snapshot=delete",
				})
				g.Expect(err).NotTo(HaveOccurred())
				return serviceList.Items
			}).WithContext(ctx).
				WithPolling(time.Second).
				WithTimeout(framework.PollTimeout * 2).
				Should(BeEmpty())
		})

		It("Verify if deleted resources are recreated in vCluster after restore", func(ctx context.Context) {
			By("Delete resources that going to be restored")
			err := f.VClusterClient.CoreV1().ConfigMaps(testNamespaceName).Delete(f.Context, configMapToRestore.Name, metav1.DeleteOptions{})
			framework.ExpectNoError(err)

			// check configmap is deleted
			Eventually(func(ctx context.Context) error {
				_, err := f.VClusterClient.CoreV1().ConfigMaps(testNamespaceName).List(ctx, metav1.ListOptions{
					LabelSelector: "snapshot=restore",
				})

				if err != nil {
					return nil
				}
				return err
			}).WithContext(ctx).
				WithPolling(time.Second).
				WithTimeout(framework.PollTimeout).
				Should(Succeed())

			err = f.VClusterClient.CoreV1().Secrets(testNamespaceName).Delete(f.Context, secretToRestore.Name, metav1.DeleteOptions{})
			framework.ExpectNoError(err)

			// check secret is deleted
			Eventually(func(ctx context.Context) error {
				_, err := f.VClusterClient.CoreV1().Secrets(testNamespaceName).List(ctx, metav1.ListOptions{
					LabelSelector: "snapshot=restore",
				})

				if err != nil {
					return nil
				}
				return err
			}).WithContext(ctx).
				WithPolling(time.Second).
				WithTimeout(framework.PollTimeout).
				Should(Succeed())

			err = f.VClusterClient.AppsV1().Deployments(testNamespaceName).Delete(f.Context, deploymentToRestore.Name, metav1.DeleteOptions{})
			framework.ExpectNoError(err)

			// check deployment is deleted
			Eventually(func(ctx context.Context) error {
				_, err := f.VClusterClient.CoreV1().Secrets(testNamespaceName).List(ctx, metav1.ListOptions{
					LabelSelector: "snapshot=restore",
				})

				if err != nil {
					return nil
				}
				return err
			}).WithContext(ctx).
				WithPolling(time.Second).
				WithTimeout(framework.PollTimeout).
				Should(Succeed())

			// now restore the vCluster
			restoreVCluster(ctx, f, snapshotPath, useNewCommand, false)

			By("Verify resources delete before snapshot are available")
			Eventually(func(g Gomega, ctx context.Context) map[string]string {
				configmaps, err := f.VClusterClient.CoreV1().ConfigMaps(testNamespaceName).List(ctx, metav1.ListOptions{
					LabelSelector: "snapshot=restore",
				})
				g.Expect(err).NotTo(HaveOccurred())

				if len(configmaps.Items) != 1 {
					return map[string]string{}
				}
				restoredConfigmap := configmaps.Items[0]
				return restoredConfigmap.Data
			}).WithContext(ctx).
				WithPolling(time.Second).
				WithTimeout(framework.PollTimeout).
				Should(Equal(configMapToRestore.Data))

			Eventually(func(g Gomega, ctx context.Context) map[string][]byte {
				secrets, err := f.VClusterClient.CoreV1().Secrets(testNamespaceName).List(ctx, metav1.ListOptions{
					LabelSelector: "snapshot=restore",
				})
				g.Expect(err).NotTo(HaveOccurred())

				if len(secrets.Items) != 1 {
					return map[string][]byte{}
				}
				restoredSecret := secrets.Items[0]
				return restoredSecret.Data
			}).WithContext(ctx).
				WithPolling(time.Second).
				WithTimeout(framework.PollTimeout).
				Should(Equal(secretToRestore.Data))

			Eventually(func(g Gomega, ctx context.Context) bool {
				deployment, err := f.VClusterClient.AppsV1().Deployments(testNamespaceName).List(ctx, metav1.ListOptions{
					LabelSelector: "snapshot=restore",
				})
				g.Expect(err).NotTo(HaveOccurred())

				if len(deployment.Items) != 1 {
					return false
				}
				return len(deployment.Items) == 1
			}).WithContext(ctx).
				WithPolling(time.Second).
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

		AfterAll(func(ctx context.Context) {
			cleanUpTestResources(ctx, false, "cli-snapshot-test-afterall")
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

		It("Creates the snapshot", func(ctx context.Context) {
			waitForSnapshotToBeCreated(ctx, f)
		})

		checkTestResources(controllerTestNamespaceName, true, snapshotPath)

		AfterAll(func(ctx context.Context) {
			cleanUpTestResources(ctx, true, "controller-snapshot-test-afterall")
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

		It("Creates the snapshot", func(ctx context.Context) {
			waitForSnapshotToBeCreated(ctx, f)
		})

		It("Doesn't contain VolumeSnapshot and VolumeSnapshotContent, because they have been cleaned up", func(ctx context.Context) {
			// get vCluster config
			vClusterRelease, err := helm.NewSecrets(f.HostClient).Get(ctx, f.VClusterName, f.VClusterNamespace)
			Expect(err).NotTo(HaveOccurred())
			Expect(vClusterRelease).NotTo(BeNil())
			vConfigValues, err := yaml.Marshal(vClusterRelease.Config)
			Expect(err).NotTo(HaveOccurred())
			Expect(vConfigValues).NotTo(BeEmpty())
			vClusterConfig, err := vclusterconfig.ParseConfigBytes(vConfigValues, f.VClusterName, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(vClusterConfig).NotTo(BeNil())

			// create the snapshot client used to check if the VolumeSnapshot and VolumeSnapshotContent are deleted
			var restConfig *rest.Config
			var volumeSnapshotsNamespace string
			if vClusterConfig.PrivateNodes.Enabled {
				restConfig = f.VClusterConfig
				volumeSnapshotsNamespace = controllerTestNamespaceName
			} else {
				restConfig = f.HostConfig
				volumeSnapshotsNamespace = f.VClusterNamespace
			}
			snapshotClient, err := snapshotsv1.NewForConfig(restConfig)
			Expect(err).NotTo(HaveOccurred())
			Expect(snapshotClient).NotTo(BeNil())

			volumeSnapshots, err := snapshotClient.SnapshotV1().VolumeSnapshots(volumeSnapshotsNamespace).List(ctx, metav1.ListOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(volumeSnapshots.Items).To(BeEmpty())

			volumeSnapshotContents, err := snapshotClient.SnapshotV1().VolumeSnapshotContents().List(ctx, metav1.ListOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(volumeSnapshotContents.Items).To(BeEmpty())
		})

		It("Deletes the PVC with test data", func(ctx context.Context) {
			deletePVC(ctx, f, controllerTestNamespaceName, pvcToRestoreName)
		})

		checkTestResources(controllerTestNamespaceName, true, snapshotPath)

		It("restores vCluster with volumes", func(ctx context.Context) {
			// PVC has been restored in previous test specs, but without data, so it's stuck in Pending.
			// Therefore, delete it again, so it gets restored properly.
			deletePVC(ctx, f, controllerTestNamespaceName, pvcToRestoreName)
			// now restore the vCluster
			restoreVCluster(ctx, f, snapshotPath, true, true)
		})

		It("has the restored PVC which is bound", func(ctx context.Context) {
			var restoredPVC *corev1.PersistentVolumeClaim
			toJSON := func(pvc *corev1.PersistentVolumeClaim) string {
				if pvc == nil {
					return ""
				}
				pvcJSON, err := json.Marshal(pvc)
				if err != nil {
					return ""
				}
				return string(pvcJSON)
			}

			Eventually(func(g Gomega, ctx context.Context) corev1.PersistentVolumeClaimPhase {
				var err error
				restoredPVC, err = f.VClusterClient.CoreV1().PersistentVolumeClaims(controllerTestNamespaceName).Get(ctx, pvcToRestoreName, metav1.GetOptions{})
				g.Expect(err).NotTo(HaveOccurred())

				return restoredPVC.Status.Phase
			}).WithContext(ctx).
				WithPolling(framework.PollInterval).
				WithTimeout(framework.PollTimeoutLong).
				Should(Equal(corev1.ClaimBound), fmt.Sprintf("PVC %s is not bound, got: %s", pvcToRestoreName, toJSON(restoredPVC)))
		})

		It("has the restored PVC with data restored from the volume snapshot", func(ctx context.Context) {
			checkPVCData(ctx, f.VClusterClient, controllerTestNamespaceName, pvcToRestoreName, testFileName, pvcData)
		})

		AfterAll(func(ctx context.Context) {
			cleanUpTestResources(ctx, true, controllerTestNamespaceName)
		})
	})

	When("a snapshot is taken while the previous one is still in progress", Ordered, func() {
		const (
			testNamespaceName = "snapshot-canceling"
			snapshotPath      = "container:///snapshot-data/" + testNamespaceName + ".tar.gz"
			appCount          = 3
			appPrefix         = "test-app-"
		)

		BeforeAll(func(ctx context.Context) {
			f = framework.DefaultFramework
			deployTestNamespace(testNamespaceName)
			for i := 0; i < appCount; i++ {
				appName := fmt.Sprintf("%s%d", appPrefix, i)
				createAppWithPVC(ctx, f.VClusterClient, testNamespaceName, appName)
			}
			Eventually(func(g Gomega, ctx context.Context) {
				for i := 0; i < appCount; i++ {
					appName := fmt.Sprintf("%s%d", appPrefix, i)
					deployment, err := f.VClusterClient.AppsV1().Deployments(testNamespaceName).Get(ctx, appName, metav1.GetOptions{})
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(deployment.Status.Replicas).To(
						Equal(int32(1)),
						fmt.Sprintf("expected deployment with 1 replica, got deployment: %s", toJSON(deployment)))
					g.Expect(deployment.Status.ReadyReplicas).To(
						Equal(int32(1)),
						fmt.Sprintf("expected deployment with 1 ready replica, got deployment: %s", toJSON(deployment)))
					g.Expect(deployment.Status.AvailableReplicas).To(
						Equal(int32(1)),
						fmt.Sprintf("expected deployment with 1 available replica, got deployment: %s", toJSON(deployment)))
				}
			}).WithContext(ctx).
				WithPolling(time.Second).
				WithTimeout(framework.PollTimeoutLong).
				Should(Succeed())

			createSnapshot(f, true, snapshotPath, true)
			time.Sleep(time.Second)
			createSnapshot(f, true, snapshotPath, true)
		})

		It("has 2 snapshot requests", func(ctx context.Context) {
			Eventually(func(g Gomega, ctx context.Context) []corev1.ConfigMap {
				listOptions := metav1.ListOptions{
					LabelSelector: constants.SnapshotRequestLabel,
				}
				configMaps, err := f.HostClient.CoreV1().ConfigMaps(f.VClusterNamespace).List(ctx, listOptions)
				g.Expect(err).NotTo(HaveOccurred())
				return configMaps.Items
			}).WithContext(ctx).
				WithPolling(framework.PollInterval).
				WithTimeout(framework.PollTimeoutLong).
				Should(HaveLen(2))
		})

		It("canceled the previous snapshot request", func(ctx context.Context) {
			// get vCluster config
			vClusterRelease, err := helm.NewSecrets(f.HostClient).Get(ctx, f.VClusterName, f.VClusterNamespace)
			Expect(err).NotTo(HaveOccurred())
			Expect(vClusterRelease).NotTo(BeNil())
			vConfigValues, err := yaml.Marshal(vClusterRelease.Config)
			Expect(err).NotTo(HaveOccurred())
			Expect(vConfigValues).NotTo(BeEmpty())
			vClusterConfig, err := vclusterconfig.ParseConfigBytes(vConfigValues, f.VClusterName, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(vClusterConfig).NotTo(BeNil())

			// create the snapshot client used to check if the VolumeSnapshot and VolumeSnapshotContent are deleted
			var restConfig *rest.Config
			var volumeSnapshotsNamespace string
			if vClusterConfig.PrivateNodes.Enabled {
				restConfig = f.VClusterConfig
				volumeSnapshotsNamespace = testNamespaceName
			} else {
				restConfig = f.HostConfig
				volumeSnapshotsNamespace = f.VClusterNamespace
			}
			snapshotClient, err := snapshotsv1.NewForConfig(restConfig)
			Expect(err).NotTo(HaveOccurred())
			Expect(snapshotClient).NotTo(BeNil())

			Eventually(func(g Gomega, ctx context.Context) {
				previousSnapshotRequest, _ := getTwoSnapshotRequests(g, ctx, f)
				for pvcName, volumeSnapshotStatus := range previousSnapshotRequest.Status.VolumeSnapshots.Snapshots {
					pvcNameParts := strings.Split(pvcName, "/")
					g.Expect(pvcNameParts).To(HaveLen(2), "expected PVC name to have 2 parts separated with '/', got: %s", pvcName)
					volumeSnapshotName := fmt.Sprintf("%s-%s", pvcNameParts[1], previousSnapshotRequest.Name)
					volumeSnapshotResource, err := snapshotClient.SnapshotV1().VolumeSnapshots(volumeSnapshotsNamespace).Get(ctx, volumeSnapshotName, metav1.GetOptions{})
					g.Expect(err).To(HaveOccurred(), "expected that canceled VolumeSnapshot is not found, but found VolumeSnapshot >>>%s<<<. Canceled snapshot request is %s", toJSON(volumeSnapshotResource), toJSON(previousSnapshotRequest))
					g.Expect(kerrors.IsNotFound(err)).To(BeTrue(), "expected that canceled VolumeSnapshot is not found, but got: %v. Canceled snapshot request is %s", err, toJSON(previousSnapshotRequest))
					g.Expect(volumeSnapshotStatus.Phase).To(
						Equal(volumes.RequestPhaseCanceled),
						fmt.Sprintf("Previous volume snapshot request for PVC %s should be canceled, got volume snapshot status: %s. Canceled snapshot request is %s", pvcName, toJSON(volumeSnapshotStatus), toJSON(previousSnapshotRequest)))
				}
				g.Expect(previousSnapshotRequest.Status.VolumeSnapshots.Phase).To(
					Equal(volumes.RequestPhaseCanceled),
					fmt.Sprintf("Previous snapshot request %s is not canceled, got: %s", previousSnapshotRequest.Name, toJSON(previousSnapshotRequest)))
				g.Expect(previousSnapshotRequest.Status.Phase).To(
					Equal(snapshot.RequestPhaseCanceled),
					fmt.Sprintf("Previous snapshot request %s is not canceled, got: %s", previousSnapshotRequest.Name, toJSON(previousSnapshotRequest)))
			}).WithContext(ctx).
				WithPolling(framework.PollInterval).
				WithTimeout(5 * time.Minute).
				Should(Succeed())
		})

		It("completed new new snapshot request", func(ctx context.Context) {
			Eventually(func(g Gomega, ctx context.Context) {
				_, newerSnapshotRequest := getTwoSnapshotRequests(g, ctx, f)
				g.Expect(newerSnapshotRequest.Status.Phase).Should(
					Equal(snapshot.RequestPhaseCompleted),
					fmt.Sprintf("Newer snapshot request %s is not completed, got: %s", newerSnapshotRequest.Name, toJSON(newerSnapshotRequest)))
				for pvcName, volumeSnapshot := range newerSnapshotRequest.Status.VolumeSnapshots.Snapshots {
					g.Expect(volumeSnapshot.Phase).To(
						Equal(volumes.RequestPhaseCompleted),
						fmt.Sprintf("New volume snapshot request for PVC %s should be completed, got: %s", pvcName, toJSON(volumeSnapshot)))
				}
			}).WithContext(ctx).
				WithPolling(framework.PollInterval).
				WithTimeout(framework.PollTimeoutLong).
				Should(Succeed())
		})

		AfterAll(func(ctx context.Context) {
			// delete the namespace
			err := f.VClusterClient.CoreV1().Namespaces().Delete(ctx, testNamespaceName, metav1.DeleteOptions{})
			Expect(err).NotTo(HaveOccurred())
			deleteSnapshotRequestConfigMaps(ctx, f)
		})
	})

	When("a snapshot is deleted", Ordered, func() {
		const (
			testNamespaceName         = "snapshot-deleting"
			snapshotPath              = "container:///snapshot-data/" + testNamespaceName + ".tar.gz"
			appCount                  = 3
			appPrefix                 = "test-app-"
			deleteSnapshotRequestName = "delete-request-" + testNamespaceName
		)

		BeforeAll(func(ctx context.Context) {
			f = framework.DefaultFramework
			deployTestNamespace(testNamespaceName)
			for i := 0; i < appCount; i++ {
				appName := fmt.Sprintf("%s%d", appPrefix, i)
				createAppWithPVC(ctx, f.VClusterClient, testNamespaceName, appName)
			}
			Eventually(func(g Gomega, ctx context.Context) {
				for i := 0; i < appCount; i++ {
					appName := fmt.Sprintf("%s%d", appPrefix, i)
					deployment, err := f.VClusterClient.AppsV1().Deployments(testNamespaceName).Get(ctx, appName, metav1.GetOptions{})
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(deployment.Status.Replicas).To(
						Equal(int32(1)),
						fmt.Sprintf("expected deployment with 1 replica, got deployment: %s", toJSON(deployment)))
					g.Expect(deployment.Status.ReadyReplicas).To(
						Equal(int32(1)),
						fmt.Sprintf("expected deployment with 1 ready replica, got deployment: %s", toJSON(deployment)))
					g.Expect(deployment.Status.AvailableReplicas).To(
						Equal(int32(1)),
						fmt.Sprintf("expected deployment with 1 available replica, got deployment: %s", toJSON(deployment)))
				}
			}).WithContext(ctx).
				WithPolling(time.Second).
				WithTimeout(framework.PollTimeoutLong).
				Should(Succeed())

			createSnapshot(f, true, snapshotPath, true)
		})

		It("creates snapshot deletion request", func(ctx context.Context) {
			// get the snapshot request Secret because we need it to create the snapshot deletion request
			var snapshotOptions *snapshot.Options
			listOptions := metav1.ListOptions{
				LabelSelector: constants.SnapshotRequestLabel,
			}
			Eventually(func(g Gomega, ctx context.Context) {
				secrets, err := f.HostClient.CoreV1().Secrets(f.VClusterNamespace).List(ctx, listOptions)
				Expect(err).NotTo(HaveOccurred())
				Expect(secrets.Items).To(HaveLen(1))
				snapshotOptions, err = snapshot.UnmarshalSnapshotOptions(&secrets.Items[0])
				Expect(err).NotTo(HaveOccurred())
			}).WithContext(ctx).
				WithPolling(framework.PollInterval).
				WithTimeout(framework.PollTimeout).
				Should(Succeed())

			// now wait for the snapshot to be completed, so we can create the snapshot deletion
			// request after all volume snapshots have been created
			waitForSnapshotToBeCreated(ctx, f)

			// now, after the snapshot has been created, create the snapshot deletion request

			// first, get the completed snapshot request, as we need it to create the snapshot deletion request
			snapshotRequestConfigMaps, err := f.HostClient.CoreV1().ConfigMaps(f.VClusterNamespace).List(ctx, listOptions)
			Expect(err).NotTo(HaveOccurred())
			Expect(snapshotRequestConfigMaps.Items).To(HaveLen(1))
			snapshotRequest, err := snapshot.UnmarshalSnapshotRequest(&snapshotRequestConfigMaps.Items[0])
			Expect(err).NotTo(HaveOccurred())

			// update the snapshot request to turn it into a snapshot deletion request
			snapshotRequest.Name = deleteSnapshotRequestName
			snapshotRequest.CreationTimestamp = metav1.Now()
			snapshotRequest.Status.Phase = snapshot.RequestPhaseDeleting

			// create the new snapshot deletion request ConfigMap
			deleteSnapshotRequestConfigMap, err := snapshot.CreateSnapshotRequestConfigMap(f.VClusterNamespace, f.VClusterName, snapshotRequest)
			Expect(err).NotTo(HaveOccurred())
			Expect(deleteSnapshotRequestConfigMap).NotTo(BeNil())
			deleteSnapshotRequestConfigMap.Name = deleteSnapshotRequestName

			// create the new snapshot options Secret
			snapshotOptionsSecret, err := snapshot.CreateSnapshotOptionsSecret(
				constants.SnapshotRequestLabel,
				f.VClusterNamespace,
				f.VClusterName,
				snapshotOptions)
			Expect(err).NotTo(HaveOccurred())
			Expect(snapshotOptionsSecret).NotTo(BeNil())
			snapshotOptionsSecret.Name = deleteSnapshotRequestName

			// finally, create the snapshot deletion request resources
			_, err = f.HostClient.CoreV1().Secrets(f.VClusterNamespace).Create(ctx, snapshotOptionsSecret, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
			_, err = f.HostClient.CoreV1().ConfigMaps(f.VClusterNamespace).Create(ctx, deleteSnapshotRequestConfigMap, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
		})

		It("has deleted the snapshot", func(ctx context.Context) {
			Eventually(func(g Gomega, ctx context.Context) {
				deleteRequestConfigMap, err := f.HostClient.CoreV1().ConfigMaps(f.VClusterNamespace).Get(ctx, deleteSnapshotRequestName, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				deleteSnapshotRequest, err := snapshot.UnmarshalSnapshotRequest(deleteRequestConfigMap)
				Expect(err).NotTo(HaveOccurred())

				g.Expect(deleteSnapshotRequest.Status.Phase).To(
					Equal(snapshot.RequestPhaseDeleted),
					fmt.Sprintf("Snapshot request %s phase is not Deleted, got: %s", deleteSnapshotRequest.Name, toJSON(deleteSnapshotRequest)))
				g.Expect(deleteSnapshotRequest.Status.VolumeSnapshots.Phase).To(
					Equal(volumes.RequestPhaseDeleted),
					fmt.Sprintf("Snapshot request %s phase is not Deleted, got: %s", deleteSnapshotRequest.Name, toJSON(deleteSnapshotRequest)))
				for pvcName, volumeSnapshot := range deleteSnapshotRequest.Status.VolumeSnapshots.Snapshots {
					g.Expect(volumeSnapshot.Phase).To(
						Equal(volumes.RequestPhaseDeleted),
						fmt.Sprintf("Volume snapshot request phase for PVC %s should be Deleted, got: %s", pvcName, toJSON(volumeSnapshot)))
				}
			}).WithContext(ctx).
				WithPolling(framework.PollInterval).
				WithTimeout(5 * time.Minute).
				Should(Succeed())
		})

		AfterAll(func(ctx context.Context) {
			// delete the namespace
			err := f.VClusterClient.CoreV1().Namespaces().Delete(ctx, testNamespaceName, metav1.DeleteOptions{})
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
