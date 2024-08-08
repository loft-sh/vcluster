package pvc

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/loft-sh/vcluster/pkg/util/random"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"github.com/loft-sh/vcluster/test/framework"
	"github.com/onsi/ginkgo/v2"
)

const (
	initialNsLabelKey   = "testing-ns-label"
	initialNsLabelValue = "testing-ns-label-value"
)

var _ = ginkgo.Describe("Persistent volume synced from host cluster", func() {
	var (
		f         *framework.Framework
		iteration int
		ns        string
	)

	ginkgo.JustBeforeEach(func() {
		f = framework.DefaultFramework
		iteration++

		ns = fmt.Sprintf("e2e-syncer-pvc-%d-%s", iteration, random.String(5))

		_, err := f.VClusterClient.CoreV1().Namespaces().Create(f.Context, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
			Name:   ns,
			Labels: map[string]string{initialNsLabelKey: initialNsLabelValue},
		}}, metav1.CreateOptions{})

		framework.ExpectNoError(err)
	})

	ginkgo.AfterEach(func() {
		err := f.DeleteTestNamespace(ns, false)
		framework.ExpectNoError(err)
	})

	ginkgo.It("Test pvc provisioned successfully and is synced back to vcluster", func() {
		pvcName := "test"
		podName := "nginx-test"

		q, err := resource.ParseQuantity("3Gi")
		framework.ExpectNoError(err)

		_, err = f.VClusterClient.CoreV1().PersistentVolumeClaims(ns).Create(f.Context, &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name: pvcName,
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{
					corev1.ReadWriteOnce,
				},
				Resources: corev1.VolumeResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: q,
					},
				},
			},
		}, metav1.CreateOptions{})

		framework.ExpectNoError(err)

		// add a pod bound to the volume as by default storage class on kind is configured with
		// volume binding mode as WaitForFirstConsumer
		_, err = f.VClusterClient.CoreV1().Pods(ns).Create(f.Context, &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: podName,
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "nginx",
						Image: "nginx",
					},
				},
				Volumes: []corev1.Volume{
					{
						Name: "nginx-pvc",
						VolumeSource: corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
								ClaimName: pvcName,
							},
						},
					},
				},
			},
		}, metav1.CreateOptions{})

		framework.ExpectNoError(err)

		err = f.WaitForPersistentVolumeClaimBound(pvcName, ns)
		framework.ExpectNoError(err, "A pvc created in the vcluster is expected to be in bound state eventually.")

		// get current status

		vpvc, err := f.VClusterClient.CoreV1().PersistentVolumeClaims(ns).Get(f.Context, pvcName, metav1.GetOptions{})
		framework.ExpectNoError(err)

		pPvcName := translate.Default.HostName(nil, pvcName, ns)
		pvc, err := f.HostClient.CoreV1().PersistentVolumeClaims(pPvcName.Namespace).Get(f.Context, pPvcName.Name, metav1.GetOptions{})
		framework.ExpectNoError(err)

		framework.ExpectEqual(vpvc.Status, pvc.Status)
	})
})
