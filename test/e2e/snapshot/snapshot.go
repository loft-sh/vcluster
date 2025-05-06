package snapshot

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/loft-sh/vcluster/pkg/util/translate"
	"github.com/loft-sh/vcluster/test/framework"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

var _ = ginkgo.Describe("Snapshot VCluster", func() {
	f := framework.DefaultFramework
	ginkgo.It("run vCluster snapshot and vCluster restore", func() {
		ginkgo.By("Make sure vcluster pods are running")
		pods, err := f.HostClient.CoreV1().Pods(f.VClusterNamespace).List(f.Context, metav1.ListOptions{
			LabelSelector: "app=vcluster",
		})
		framework.ExpectNoError(err)
		framework.ExpectEqual(true, len(pods.Items) > 0)

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

		// get vCluster default namespace
		vClusterDefaultNamespace := f.VClusterNamespace
		if f.MultiNamespaceMode {
			vClusterDefaultNamespace = translate.NewMultiNamespaceTranslator(f.VClusterNamespace).HostNamespace(nil, "default")
		}

		// now create a service that should be there when we restore again
		_, err = f.VClusterClient.CoreV1().Services("default").Create(f.Context, &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "snapshot-restore",
				Namespace: "default",
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

		ginkgo.By("Snapshot vcluster")
		// regular snapshot
		cmd := exec.Command(
			"vcluster",
			"snapshot",
			f.VClusterName,
			"container:///snapshot-pvc/snapshot.tar",
			"-n", f.VClusterNamespace,
			"--pod-mount", "pvc:snapshot-pvc:/snapshot-pvc",
		)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		framework.ExpectNoError(err)

		// now delete the service
		err = f.VClusterClient.CoreV1().Services("default").Delete(f.Context, "snapshot-restore", metav1.DeleteOptions{})
		framework.ExpectNoError(err)

		// now create a service that should not be there when we restore
		_, err = f.VClusterClient.CoreV1().Services("default").Create(f.Context, &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "snapshot-delete",
				Namespace: "default",
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
			"container:///snapshot-pvc/snapshot.tar",
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

		// create new namespace and wait until the default service account gets created
		gomega.Eventually(func() error {
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
			Should(gomega.Succeed())

		// delete the namespace
		err = f.VClusterClient.CoreV1().Namespaces().Delete(f.Context, "snapshot-test", metav1.DeleteOptions{})
		framework.ExpectNoError(err)
	})
})
