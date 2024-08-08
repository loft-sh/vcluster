package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/loft-sh/vcluster/pkg/util/translate"

	"github.com/loft-sh/vcluster/test/framework"
	"github.com/onsi/ginkgo/v2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
)

var _ = ginkgo.Describe("Schedule a Statefulset with WaitForFirstConsumer PVCs", func() {
	// the default kind storageclass is WaitForFirtConsumer
	f := framework.DefaultFramework
	// create statefulset with pvc
	workload := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-statefulset",
			Namespace: "default",
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: int32Ref(1),
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
					},
				},
			},
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "test"}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "test"},
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

	ginkgo.It("Wait for Statefulset to become ready", func() {

		// determine storageclass to use
		scList := &storagev1.StorageClassList{}
		err := f.HostCRClient.List(f.Context, scList)
		framework.ExpectNoError(err)
		framework.ExpectNotEmpty(scList.Items)

		// default to first storageclass
		scToUse := scList.Items[0]
		// look for a storageclasss with WaitForFirstConsumer, preferably default
		for _, sc := range scList.Items {
			if sc.VolumeBindingMode != nil && *sc.VolumeBindingMode == storagev1.VolumeBindingWaitForFirstConsumer {
				scToUse = sc
				if framework.IsDefaultAnnotation(sc.ObjectMeta) {
					break
				}
			}
		}

		fmt.Fprintf(ginkgo.GinkgoWriter, "using storageclass %q\n", scToUse.Name)
		// if selected WaitForConsumer storageclass can't be found, attempt to make one from a copy
		if scToUse.VolumeBindingMode == nil || *scToUse.VolumeBindingMode != storagev1.VolumeBindingWaitForFirstConsumer {
			fmt.Fprintf(ginkgo.GinkgoWriter, "creating a copy of %q with WaitForFirstConsumer\n", scToUse.Name)
			newSC := scToUse.DeepCopy()
			translate.ResetObjectMetadata(&newSC.ObjectMeta)
			newSC.Name = "test-sc"
			err = f.HostCRClient.Create(f.Context, newSC)
			framework.ExpectNoError(err)
			defer func() {
				err := f.HostCRClient.Delete(f.Context, newSC)
				framework.ExpectNoError(err, "cleaning up created storageclass")
			}()
			scToUse = *newSC
		}

		// ceate
		workload.Spec.VolumeClaimTemplates[0].Spec.StorageClassName = &scToUse.Name
		err = f.VClusterCRClient.Create(f.Context, workload)
		framework.ExpectNoError(err)
		// wait for it to start running
		err = wait.PollUntilContextTimeout(f.Context, time.Second, time.Minute*2, false, func(ctx context.Context) (bool, error) {
			ss := &appsv1.StatefulSet{}
			err := f.VClusterCRClient.Get(ctx, types.NamespacedName{Name: workload.Name, Namespace: workload.Namespace}, ss)
			if err != nil {
				fmt.Fprintf(ginkgo.GinkgoWriter, "failed to fetch statefulset %q with err err: %v\n", workload.Name, err)
				return false, nil
			}
			fmt.Fprintf(ginkgo.GinkgoWriter, "statefulset.status.readyreplicas: %d\n", ss.Status.ReadyReplicas)
			return ss.Status.ReadyReplicas == 1, nil
		})
		framework.ExpectNoError(err)
	})
})

func int32Ref(i int32) *int32 {
	return &i
}
