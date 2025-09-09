package snapshot

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/loft-sh/vcluster/test/framework"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

var _ = Describe("snapshot and restore", Ordered, func() {
	const snapshotPath = "container:///snapshot-pvc/snapshot.tar"

	var (
		f                        *framework.Framework
		vClusterDefaultNamespace string
		defaultNamespace         string
		cmd                      *exec.Cmd
		configMapToRestore       *corev1.ConfigMap
		configMapToDelete        *corev1.ConfigMap
		secretToRestore          *corev1.Secret
		secretToDelete           *corev1.Secret
		deploymentToRestore      *appsv1.Deployment
		serviceToRestore         *corev1.Service
		pvc                      *corev1.PersistentVolumeClaim
	)

	beforeAll := func() {
		f = framework.DefaultFramework
		vClusterDefaultNamespace = f.VClusterNamespace
		defaultNamespace = "default"
		cmd = &exec.Cmd{}

		configMapToRestore = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "configmap-restore",
				Namespace: defaultNamespace,
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
				Namespace: defaultNamespace,
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
				Namespace: defaultNamespace,
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
				Namespace: defaultNamespace,
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
				Name:   "deployment-restore",
				Labels: map[string]string{"snapshot": "restore"},
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: intRef(1),
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
				Namespace: defaultNamespace,
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

		pods, err := f.HostClient.CoreV1().Pods(vClusterDefaultNamespace).List(f.Context, metav1.ListOptions{
			LabelSelector: "app=vcluster",
		})
		framework.ExpectNoError(err)
		framework.ExpectEqual(true, len(pods.Items) > 0)

		By("Create test resources")
		_, err = f.HostClient.CoreV1().PersistentVolumeClaims(pvc.Namespace).Create(f.Context, pvc, metav1.CreateOptions{})
		framework.ExpectNoError(err)

		// now create a service that should be there when we restore again
		_, err = f.VClusterClient.CoreV1().Services(defaultNamespace).Create(f.Context, serviceToRestore, metav1.CreateOptions{})
		framework.ExpectNoError(err)

		// now create a configmap that should be there when we restore again
		_, err = f.VClusterClient.CoreV1().ConfigMaps(defaultNamespace).Create(f.Context, configMapToRestore, metav1.CreateOptions{})
		framework.ExpectNoError(err)

		// now create a secret that should be there when we restore again
		_, err = f.VClusterClient.CoreV1().Secrets(defaultNamespace).Create(f.Context, secretToRestore, metav1.CreateOptions{})
		framework.ExpectNoError(err)

		// now create a deployment that should be there when we restore again
		_, err = f.VClusterClient.AppsV1().Deployments(defaultNamespace).Create(f.Context, deploymentToRestore, metav1.CreateOptions{})
		framework.ExpectNoError(err)

		By("Snapshot vcluster")
		// regular snapshot
		cmd := exec.Command(
			"vcluster",
			"snapshot",
			f.VClusterName,
			snapshotPath,
			"-n", f.VClusterNamespace,
			"--pod-mount", "pvc:snapshot-pvc:/snapshot-pvc",
		)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		framework.ExpectNoError(err)
	}

	afterAll := func() {
		// delete the snapshot pvc
		err := f.HostClient.CoreV1().PersistentVolumeClaims(pvc.Namespace).Delete(f.Context, pvc.Name, metav1.DeleteOptions{})
		framework.ExpectNoError(err)

		Eventually(func() error {
			// create namespace
			_, err := f.VClusterClient.CoreV1().Namespaces().Create(f.Context, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "snapshot-test",
				},
			}, metav1.CreateOptions{})
			if err != nil && !kerrors.IsAlreadyExists(err) {
				return err
			}

			// wait until the default service account gets created
			_, err = f.VClusterClient.CoreV1().ServiceAccounts("snapshot-test").Get(f.Context, "default", metav1.GetOptions{})
			if err != nil {
				return err
			}

			return nil
		}).WithPolling(time.Second).
			WithTimeout(framework.PollTimeout).
			Should(Succeed())

		// delete the namespace
		err = f.VClusterClient.CoreV1().Namespaces().Delete(f.Context, "snapshot-test", metav1.DeleteOptions{})
		framework.ExpectNoError(err)
	}

	runSpecs := func() {
		It("Verify if only the resources in snapshot are available in vCluster after restore", func() {
			// now create a configmap that should be deleted by restore
			_, err := f.VClusterClient.CoreV1().ConfigMaps(defaultNamespace).Create(f.Context, configMapToDelete, metav1.CreateOptions{})
			framework.ExpectNoError(err)

			// now create a secret that should be deleted by restore
			_, err = f.VClusterClient.CoreV1().Secrets(defaultNamespace).Create(f.Context, secretToDelete, metav1.CreateOptions{})
			framework.ExpectNoError(err)

			// now create a service that should be deleted by restore
			serviceToDelete, err := f.VClusterClient.CoreV1().Services(defaultNamespace).Create(f.Context, &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "snapshot-delete",
					Namespace: defaultNamespace,
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
			By("Restore vCluster")
			cmd = exec.Command(
				"vcluster",
				"restore",
				f.VClusterName,
				snapshotPath,
				"-n", f.VClusterNamespace,
				"--pod-mount", "pvc:snapshot-pvc:/snapshot-pvc",
			)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err = cmd.Run()
			framework.ExpectNoError(err)

			// wait until vCluster is running
			err = wait.PollUntilContextTimeout(f.Context, time.Second, time.Minute*2, false, func(ctx context.Context) (done bool, err error) {
				newPods, _ := f.HostClient.CoreV1().Pods(f.VClusterNamespace).List(ctx, metav1.ListOptions{
					LabelSelector: "app=vcluster",
				})
				p := len(newPods.Items)
				if p > 0 {
					// rp, running pod counter
					rp := 0
					for _, pod := range newPods.Items {
						if pod.Status.Phase == corev1.PodRunning {
							rp = rp + 1
						}
					}
					if rp == p {
						return true, nil
					}
				}
				return false, nil
			})
			framework.ExpectNoError(err)

			By("Verify only resources created before snapshot are available")
			// wait until all vCluster replicas are running
			Eventually(func() error {
				pods, err := f.HostClient.CoreV1().Pods(vClusterDefaultNamespace).List(f.Context, metav1.ListOptions{
					LabelSelector: "app=vcluster,release=" + f.VClusterName,
				})
				framework.ExpectNoError(err)

				for _, pod := range pods.Items {
					if len(pod.Status.ContainerStatuses) == 0 {
						return fmt.Errorf("pod %s has no container status", pod.Name)
					}

					for _, container := range pod.Status.ContainerStatuses {
						if container.State.Running == nil || !container.Ready {
							return fmt.Errorf("pod %s container %s is not running", pod.Name, container.Name)
						}
					}
				}

				return nil
			}).WithPolling(time.Second).
				WithTimeout(framework.PollTimeout).
				Should(Succeed())

			// refresh the connection
			err = f.RefreshVirtualClient()
			framework.ExpectNoError(err)

			// Check configmap created before snapshot is available
			configmaps, err := f.VClusterClient.CoreV1().ConfigMaps(defaultNamespace).List(f.Context, metav1.ListOptions{
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
			secrets, err := f.VClusterClient.CoreV1().Secrets(defaultNamespace).List(f.Context, metav1.ListOptions{
				LabelSelector: "snapshot=restore",
			})

			Expect(secrets.Items).To(HaveLen(1))
			restoredSecret := secrets.Items[0]
			Expect(restoredSecret.Data).To(Equal(secretToRestore.Data))
			framework.ExpectNoError(err)

			// Check deployment created before snapshot is available
			deployment, err := f.VClusterClient.AppsV1().Deployments(defaultNamespace).List(f.Context, metav1.ListOptions{
				LabelSelector: "snapshot=restore",
			})

			Expect(deployment.Items).To(HaveLen(1))
			framework.ExpectNoError(err)

			// Check configmap created after snapshot is not available
			Eventually(func() bool {
				configmaps, err := f.VClusterClient.CoreV1().ConfigMaps(defaultNamespace).List(f.Context, metav1.ListOptions{
					LabelSelector: "snapshot=delete",
				})

				if len(configmaps.Items) != 0 {
					return false
				}
				framework.ExpectNoError(err)
				return true
			}).WithPolling(time.Second).
				WithTimeout(framework.PollTimeout).
				Should(BeTrue())

			// Check secret created after snapshot is not available
			Eventually(func() bool {
				secrets, err := f.VClusterClient.CoreV1().Secrets(defaultNamespace).List(f.Context, metav1.ListOptions{
					LabelSelector: "snapshot=delete",
				})

				if len(secrets.Items) != 0 {
					return false
				}
				framework.ExpectNoError(err)
				return true
			}).WithPolling(time.Second).
				WithTimeout(framework.PollTimeout).
				Should(BeTrue())

			//Check service created after snapshot is not available
			Eventually(func() bool {
				deployment, err := f.VClusterClient.CoreV1().Services(defaultNamespace).List(f.Context, metav1.ListOptions{
					LabelSelector: "snapshot=delete",
				})

				if len(deployment.Items) != 0 {
					return false
				}
				framework.ExpectNoError(err)
				return true
			}).WithPolling(time.Second).
				WithTimeout(framework.PollTimeout * 2).
				Should(BeTrue())
		})

		It("Verify if deleted resources are recreated in vCluster after restore", func() {

			By("Delete resources that going to be restored")
			err := f.VClusterClient.CoreV1().ConfigMaps(defaultNamespace).Delete(f.Context, configMapToRestore.Name, metav1.DeleteOptions{})
			framework.ExpectNoError(err)

			// check configmap is deleted
			Eventually(func() error {
				_, err := f.VClusterClient.CoreV1().ConfigMaps(defaultNamespace).List(f.Context, metav1.ListOptions{
					LabelSelector: "snapshot=restore",
				})

				if err != nil {
					return nil
				}
				return err
			}).WithPolling(time.Second).
				WithTimeout(framework.PollTimeout).
				Should(Succeed())

			err = f.VClusterClient.CoreV1().Secrets(defaultNamespace).Delete(f.Context, secretToRestore.Name, metav1.DeleteOptions{})
			framework.ExpectNoError(err)

			// check secret is deleted
			Eventually(func() error {
				_, err := f.VClusterClient.CoreV1().Secrets(defaultNamespace).List(f.Context, metav1.ListOptions{
					LabelSelector: "snapshot=restore",
				})

				if err != nil {
					return nil
				}
				return err
			}).WithPolling(time.Second).
				WithTimeout(framework.PollTimeout).
				Should(Succeed())

			err = f.VClusterClient.AppsV1().Deployments(defaultNamespace).Delete(f.Context, deploymentToRestore.Name, metav1.DeleteOptions{})
			framework.ExpectNoError(err)

			// check deployment is deleted
			Eventually(func() error {
				_, err := f.VClusterClient.CoreV1().Secrets(defaultNamespace).List(f.Context, metav1.ListOptions{
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
			By("Restore vCluster")
			cmd = exec.Command(
				"vcluster",
				"restore",
				f.VClusterName,
				snapshotPath,
				"-n", f.VClusterNamespace,
				"--pod-mount", "pvc:snapshot-pvc:/snapshot-pvc",
			)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err = cmd.Run()
			framework.ExpectNoError(err)

			// wait until vCluster is running
			err = wait.PollUntilContextTimeout(f.Context, time.Second, time.Minute*2, false, func(ctx context.Context) (done bool, err error) {
				newPods, _ := f.HostClient.CoreV1().Pods(f.VClusterNamespace).List(ctx, metav1.ListOptions{
					LabelSelector: "app=vcluster",
				})
				p := len(newPods.Items)
				if p > 0 {
					// rp, running pod counter
					rp := 0
					for _, pod := range newPods.Items {
						if pod.Status.Phase == corev1.PodRunning {
							rp = rp + 1
						}
					}
					if rp == p {
						return true, nil
					}
				}
				return false, nil
			})
			framework.ExpectNoError(err)

			// wait until all vCluster replicas are running
			Eventually(func() error {
				pods, err := f.HostClient.CoreV1().Pods(vClusterDefaultNamespace).List(f.Context, metav1.ListOptions{
					LabelSelector: "app=vcluster,release=" + f.VClusterName,
				})
				framework.ExpectNoError(err)

				for _, pod := range pods.Items {
					if len(pod.Status.ContainerStatuses) == 0 {
						return fmt.Errorf("pod %s has no container status", pod.Name)
					}

					for _, container := range pod.Status.ContainerStatuses {
						if container.State.Running == nil || !container.Ready {
							return fmt.Errorf("pod %s container %s is not running", pod.Name, container.Name)
						}
					}
				}
				return nil
			}).WithPolling(time.Second).
				WithTimeout(framework.PollTimeout).
				Should(Succeed())

			// refresh the connection
			err = f.RefreshVirtualClient()
			framework.ExpectNoError(err)

			By("Verify resources delete before snapshot are available")
			Eventually(func() map[string]string {
				configmaps, err := f.VClusterClient.CoreV1().ConfigMaps(defaultNamespace).List(f.Context, metav1.ListOptions{
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
				secrets, err := f.VClusterClient.CoreV1().Secrets(defaultNamespace).List(f.Context, metav1.ListOptions{
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
				deployment, err := f.VClusterClient.AppsV1().Deployments(defaultNamespace).List(f.Context, metav1.ListOptions{
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

	BeforeAll(func() {
		beforeAll()
	})

	runSpecs()

	AfterAll(func() {
		afterAll()
	})

})

func intRef(i int32) *int32 {
	return &i
}
