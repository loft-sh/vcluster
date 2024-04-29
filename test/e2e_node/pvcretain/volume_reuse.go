package pvcretain

import (
	"context"
	"time"

	"github.com/loft-sh/vcluster/test/framework"
	"github.com/onsi/ginkgo/v2"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

var _ = ginkgo.Describe("pvc with retain tests", ginkgo.Ordered, func() {
	var (
		f          = framework.DefaultFramework
		volumeName = ""
	)

	ginkgo.BeforeAll(func() {
		// create a sc with retain
		retain := corev1.PersistentVolumeReclaimRetain
		waitFor := storagev1.VolumeBindingWaitForFirstConsumer
		storageClass := storagev1.StorageClass{
			ObjectMeta: metav1.ObjectMeta{
				Name: "retain",
			},
			Provisioner:       "rancher.io/local-path",
			ReclaimPolicy:     &retain,
			VolumeBindingMode: &waitFor,
		}
		_, err := f.HostClient.StorageV1().StorageClasses().Create(f.Context, &storageClass, metav1.CreateOptions{})

		framework.ExpectNoError(err)
	})

	ginkgo.It("should create a pvc , pv and attach pv to a pod", func() {
		ctx := f.Context

		retain := "retain"
		pvc, err := f.VclusterClient.CoreV1().PersistentVolumeClaims("default").Create(
			ctx,
			&corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{Name: "pvc", Namespace: "default"},
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{
							"storage": resource.MustParse("1Gi"),
						},
					},
					StorageClassName: &retain,
				},
			},
			metav1.CreateOptions{})
		framework.ExpectNoError(err)
		volumeName = pvc.Spec.VolumeName
		_, err = f.VclusterClient.CoreV1().Pods("default").Create(ctx, &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "test"},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "nginx",
						Image: "nginx",
						VolumeMounts: []corev1.VolumeMount{
							{Name: "mypd", MountPath: "/tmp/"},
						},
					},
				},
				Volumes: []corev1.Volume{
					{
						Name: "mypd",
						VolumeSource: corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: "pvc"},
						},
					},
				},
			},
		}, metav1.CreateOptions{})
		framework.ExpectNoError(err)
		waitErr := wait.PollUntilContextTimeout(ctx, time.Millisecond*500, framework.PollTimeout, true, func(ctx context.Context) (done bool, err error) {
			pod, err := f.VclusterClient.CoreV1().Pods("default").Get(ctx, "test", metav1.GetOptions{})
			if kerrors.IsNotFound(err) {
				return false, nil
			}
			if err != nil {
				f.Log.Error(err)
				return false, err
			}
			if len(pod.Status.ContainerStatuses) > 0 && pod.Status.ContainerStatuses[0].State.Running != nil {
				return true, nil
			}

			return false, nil
		})
		framework.ExpectNoError(waitErr)
		pvc, err = f.VclusterClient.CoreV1().PersistentVolumeClaims("default").Get(ctx, "pvc", metav1.GetOptions{})
		framework.ExpectNoError(err)
		framework.ExpectNotEqual(pvc.Spec.VolumeName, "")
		volumeName = pvc.Spec.VolumeName
	})

	ginkgo.It("pvc should go to released after the pod is deleted and pvc is deleted", func() {
		waitErr := wait.PollUntilContextTimeout(f.Context, time.Millisecond*500, framework.PollTimeout, true, func(ctx context.Context) (done bool, err error) {
			err = f.VclusterClient.CoreV1().Pods("default").Delete(ctx, "test", metav1.DeleteOptions{})
			if err != nil {
				f.Log.Error(err)
				return false, err
			}

			return true, nil
		})
		framework.ExpectNoError(waitErr)

		waitErr = wait.PollUntilContextTimeout(f.Context, time.Millisecond*500, framework.PollTimeout, true, func(ctx context.Context) (done bool, err error) {
			err = f.VclusterClient.CoreV1().PersistentVolumeClaims("default").Delete(ctx, "pvc", metav1.DeleteOptions{})
			if err != nil {
				f.Log.Error(err)
				return false, err
			}

			return true, nil
		})
		framework.ExpectNoError(waitErr)

		waitErr = wait.PollUntilContextTimeout(f.Context, time.Millisecond*500, framework.PollTimeout, true, func(ctx context.Context) (done bool, err error) {
			pv, err := f.VclusterClient.CoreV1().PersistentVolumes().Get(ctx, volumeName, metav1.GetOptions{})
			if err != nil {
				f.Log.Error(err)
				return false, err
			}
			if pv.Status.Phase != corev1.VolumeReleased {
				return false, nil
			}

			return true, nil
		})
		framework.ExpectNoError(waitErr)
	})

	ginkgo.It("pvc should go to available after the claimref is removed", func() {
		var pv *corev1.PersistentVolume
		var err error
		pv, err = f.VclusterClient.CoreV1().PersistentVolumes().Get(f.Context, volumeName, metav1.GetOptions{})
		if err != nil {
			f.Log.Error(err)
		}
		framework.ExpectNoError(err)

		pv.Spec.ClaimRef = nil
		_, err = f.VclusterClient.CoreV1().PersistentVolumes().Update(f.Context, pv, metav1.UpdateOptions{})
		if err != nil {
			f.Log.Error(err)
		}
		framework.ExpectNoError(err)

		waitErr := wait.PollUntilContextTimeout(f.Context, time.Millisecond*500, framework.PollTimeout, true, func(ctx context.Context) (done bool, err error) {
			pv, err = f.VclusterClient.CoreV1().PersistentVolumes().Get(ctx, volumeName, metav1.GetOptions{})
			if err != nil {
				f.Log.Error(err)
				return false, err
			}
			if pv.Status.Phase != corev1.VolumeAvailable {
				f.Log.Info(pv.Status.Phase)
				return false, nil
			}

			return true, nil
		})
		framework.ExpectNoError(waitErr)
	})
	ginkgo.It("should be able to bind the pod back to the volume", func() {
		retain := "retain"
		_, err := f.VclusterClient.CoreV1().PersistentVolumeClaims("default").Create(
			f.Context,
			&corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{Name: "pvc", Namespace: "default"},
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{
							"storage": resource.MustParse("1Gi"),
						},
					},
					StorageClassName: &retain,
					VolumeName:       volumeName,
				},
			},
			metav1.CreateOptions{})
		framework.ExpectNoError(err)

		_, err = f.VclusterClient.CoreV1().Pods("default").Create(f.Context, &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "test"},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "nginx",
						Image: "nginx",
						VolumeMounts: []corev1.VolumeMount{
							{Name: "mypd", MountPath: "/tmp/"},
						},
					},
				},
				Volumes: []corev1.Volume{
					{
						Name: "mypd",
						VolumeSource: corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: "pvc"},
						},
					},
				},
			},
		}, metav1.CreateOptions{})
		framework.ExpectNoError(err)

		waitErr := wait.PollUntilContextTimeout(f.Context, time.Millisecond*500, framework.PollTimeout, true, func(ctx context.Context) (done bool, err error) {
			pod, err := f.VclusterClient.CoreV1().Pods("default").Get(ctx, "test", metav1.GetOptions{})
			if kerrors.IsNotFound(err) {
				return false, nil
			}
			if err != nil {
				f.Log.Error(err)
				return false, err
			}
			if len(pod.Status.ContainerStatuses) > 0 && pod.Status.ContainerStatuses[0].State.Running != nil {
				return true, nil
			}

			return false, nil
		})
		framework.ExpectNoError(waitErr)
	})
})
