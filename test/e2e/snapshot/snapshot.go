package snapshot

import (
	"context"
	"os/exec"
	"time"

	"github.com/loft-sh/vcluster/test/framework"
	"github.com/onsi/ginkgo/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

var _ = ginkgo.Describe("Snapshot VCluster", func() {
	f := framework.DefaultFramework
	ginkgo.It("run vcluster snapshot and vcluster restore", func() {
		ginkgo.By("Make sure vcluster pods are running")
		pods, err := f.HostClient.CoreV1().Pods(f.VclusterNamespace).List(f.Context, metav1.ListOptions{
			LabelSelector: "app=vcluster",
		})
		framework.ExpectNoError(err)
		framework.ExpectEqual(true, len(pods.Items) > 0)

		// create a pvc we will use to store the snapshot
		pvc := &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "snapshot-pvc",
				Namespace: f.VclusterNamespace,
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
		cmd := exec.Command(
			"vcluster",
			"snapshot",
			f.VclusterName,
			"-n", f.VclusterNamespace,
			"--storage", "file",
			"--file-path", "/snapshot-pvc/snapshot.tar",
			"--pod-mount", "pvc:snapshot-pvc:/snapshot-pvc",
		)
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

		ginkgo.By("Restore vcluster")
		cmd = exec.Command(
			"vcluster",
			"restore",
			f.VclusterName,
			"-n", f.VclusterNamespace,
			"--storage", "file",
			"--file-path", "/snapshot-pvc/snapshot.tar",
			"--pod-mount", "pvc:snapshot-pvc:/snapshot-pvc",
		)
		err = cmd.Run()
		framework.ExpectNoError(err)

		// wait until vCluster is running
		err = wait.PollUntilContextTimeout(f.Context, time.Second, time.Minute*2, false, func(ctx context.Context) (done bool, err error) {
			newPods, _ := f.HostClient.CoreV1().Pods(f.VclusterNamespace).List(ctx, metav1.ListOptions{
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
		services, err := f.HostClient.CoreV1().Services(f.VclusterNamespace).List(f.Context, metav1.ListOptions{
			LabelSelector: "snapshot=delete",
		})
		framework.ExpectNoError(err)
		framework.ExpectEqual(len(services.Items), 0)

		// check for the secret getting created
		services, err = f.HostClient.CoreV1().Services(f.VclusterNamespace).List(f.Context, metav1.ListOptions{
			LabelSelector: "snapshot=restore",
		})
		framework.ExpectNoError(err)
		framework.ExpectEqual(len(services.Items), 1)
	})
})
