package snapshot

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/loft-sh/vcluster/pkg/util/translate"
	"github.com/loft-sh/vcluster/test/framework"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

var _ = ginkgo.Describe("Snapshot and restore VCluster", ginkgo.Ordered, func() {
	f := framework.DefaultFramework
	vClusterDefaultNamespace := f.VClusterNamespace
	defaultNamespace := "default"
	cmd := &exec.Cmd{}

	configMapToRestore := &corev1.ConfigMap{
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

	configMapToDelete := &corev1.ConfigMap{
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

	secretToRestore := &corev1.Secret{
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

	secretToDelete := &corev1.Secret{
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

	deploymentToRestore := &appsv1.Deployment{
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

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "snapshot-pvc",
			Namespace: f.VClusterNamespace,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("5Gi"),
				},
			},
		},
	}

	ginkgo.BeforeAll(func() {
		ginkgo.By("run vCluster snapshot")
		pods, err := f.HostClient.CoreV1().Pods(f.VClusterNamespace).List(f.Context, metav1.ListOptions{
			LabelSelector: "app=vcluster",
		})
		framework.ExpectNoError(err)
		framework.ExpectEqual(true, len(pods.Items) > 0)

		// skip restore if k0s
		for _, pod := range pods.Items {
			for _, container := range pod.Spec.InitContainers {
				if strings.Contains(container.Image, "k0s") {
					ginkgo.Skip("Skip restore for k0s.")
				}
			}
		}

		// create a pvc we will use to store the snapshot
		pvc := &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "snapshot-pvc",
				Namespace: f.VClusterNamespace,
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
		_, err = f.HostClient.CoreV1().PersistentVolumeClaims(pvc.Namespace).Create(f.Context, pvc, metav1.CreateOptions{})
		framework.ExpectNoError(err)

		if f.MultiNamespaceMode {
			vClusterDefaultNamespace = translate.NewMultiNamespaceTranslator(f.VClusterNamespace).HostNamespace(nil, defaultNamespace)
		}

		// now create a service that should be there when we restore again
		_, err = f.VClusterClient.CoreV1().Services(defaultNamespace).Create(f.Context, &corev1.Service{
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
		}, metav1.CreateOptions{})
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

		ginkgo.By("Snapshot vcluster")
		// regular snapshot
		cmd := exec.Command(
			"vcluster",
			"snapshot",
			f.VClusterName,
			"file:///snapshot-pvc/snapshot.tar",
			"-n", f.VClusterNamespace,
			"--pod-mount", "pvc:snapshot-pvc:/snapshot-pvc",
		)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		framework.ExpectNoError(err)

		// now create a configmap that should be deleted by restore
		_, err = f.VClusterClient.CoreV1().ConfigMaps(defaultNamespace).Create(f.Context, configMapToDelete, metav1.CreateOptions{})
		framework.ExpectNoError(err)

		// now create a secret that should be deleted by restore
		_, err = f.VClusterClient.CoreV1().Secrets(defaultNamespace).Create(f.Context, secretToDelete, metav1.CreateOptions{})
		framework.ExpectNoError(err)

		// delete deployment that should be restored
		err = f.VClusterClient.AppsV1().Deployments(defaultNamespace).Delete(f.Context, deploymentToRestore.Name, metav1.DeleteOptions{})
		framework.ExpectNoError(err)

	})

	ginkgo.It("Restore should overwrite data", func() {
		// now delete the service
		err := f.VClusterClient.CoreV1().Services(defaultNamespace).Delete(f.Context, "snapshot-restore", metav1.DeleteOptions{})
		framework.ExpectNoError(err)

		// now create a service that should not be there when we restore
		_, err = f.VClusterClient.CoreV1().Services(defaultNamespace).Create(f.Context, &corev1.Service{
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

		// check for the service getting deleted
		gomega.Eventually(func() int {
			services, err := f.HostClient.CoreV1().Services(vClusterDefaultNamespace).List(f.Context, metav1.ListOptions{
				LabelSelector: "snapshot=delete",
			})
			framework.ExpectNoError(err)
			return len(services.Items)
		}).WithPolling(time.Second).
			WithTimeout(framework.PollTimeout).
			Should(gomega.Equal(1))

		// now restore the vCluster
		ginkgo.By("Restore vCluster")
		cmd = exec.Command(
			"vcluster",
			"restore",
			f.VClusterName,
			"file:///snapshot-pvc/snapshot.tar",
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

		// delete the snapshot pvc
		err = f.HostClient.CoreV1().PersistentVolumeClaims(pvc.Namespace).Delete(f.Context, pvc.Name, metav1.DeleteOptions{})
		framework.ExpectNoError(err)

		// check for the service getting deleted
		gomega.Eventually(func() int {
			services, err := f.HostClient.CoreV1().Services(vClusterDefaultNamespace).List(f.Context, metav1.ListOptions{
				LabelSelector: "snapshot=delete",
			})
			framework.ExpectNoError(err)
			return len(services.Items)
		}).WithPolling(time.Second).
			WithTimeout(framework.PollTimeout).
			Should(gomega.Equal(0))

		// check for the service getting created
		gomega.Eventually(func() int {
			services, err := f.HostClient.CoreV1().Services(vClusterDefaultNamespace).List(f.Context, metav1.ListOptions{
				LabelSelector: "snapshot=restore",
			})
			framework.ExpectNoError(err)
			return len(services.Items)
		}).WithPolling(time.Second).
			WithTimeout(framework.PollTimeout).
			Should(gomega.Equal(1))

		// wait until all vCluster replicas are running
		gomega.Eventually(func() error {
			pods, err := f.HostClient.CoreV1().Pods(f.VClusterNamespace).List(f.Context, metav1.ListOptions{
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
			Should(gomega.Succeed())

		// refresh the connection
		err = f.RefreshVirtualClient()
		framework.ExpectNoError(err)
	})

	ginkgo.It("Should override by snapshot objects created after snapshot", func() {

		gomega.Eventually(func() error {
			_, err := f.VClusterClient.CoreV1().ConfigMaps(defaultNamespace).List(f.Context, metav1.ListOptions{
				LabelSelector: "snapshot=delete",
			})

			if err != nil {
				return nil
			}
			return err
		}).WithPolling(time.Second).
			WithTimeout(framework.PollTimeout).
			Should(gomega.Succeed())

		gomega.Eventually(func() error {
			_, err := f.VClusterClient.CoreV1().Secrets(defaultNamespace).List(f.Context, metav1.ListOptions{
				LabelSelector: "snapshot=delete",
			})

			if err != nil {
				return nil
			}
			return err
		}).WithPolling(time.Second).
			WithTimeout(framework.PollTimeout).
			Should(gomega.Succeed())

	})

	ginkgo.It("Should contain previously created objects", func() {
		gomega.Eventually(func() map[string]string {
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
			Should(gomega.Equal(configMapToRestore.Data))

		gomega.Eventually(func() map[string][]byte {
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
			Should(gomega.Equal(secretToRestore.Data))

		gomega.Eventually(func() bool {
			deployment, err := f.VClusterClient.AppsV1().Deployments(defaultNamespace).List(f.Context, metav1.ListOptions{
				LabelSelector: "snapshot=restore",
			})

			if len(deployment.Items) != 1 {
				fmt.Println(deployment.Items)
				fmt.Println(err)
				return false
			}
			framework.ExpectNoError(err)
			return len(deployment.Items) == 1
		}).WithPolling(time.Second).
			WithTimeout(framework.PollTimeout * 2).
			Should(gomega.BeTrue())
	})

})

func intRef(i int32) *int32 {
	return &i
}
