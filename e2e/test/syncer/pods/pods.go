package pods

import (
	"fmt"
	"time"

	"github.com/loft-sh/vcluster/e2e/framework"
	"github.com/loft-sh/vcluster/pkg/util/podhelper"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"github.com/onsi/ginkgo"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	testingContainerName  = "nginx"
	testingContainerImage = "nginx"
)

var _ = ginkgo.Describe("Pods are running in the host cluster", func() {
	var (
		f         *framework.Framework
		iteration int
		ns        string
	)

	ginkgo.JustBeforeEach(func() {
		// use default framework
		f = framework.DefaultFramework
		iteration++
		ns = fmt.Sprintf("e2e-syncer-pods-%d", iteration)
		// execute cleanup in case previous e2e test were terminated prematurely
		err := cleanup(f, ns, true)
		framework.ExpectNoError(err)

		// create test namespace
		_, err = f.VclusterClient.CoreV1().Namespaces().Create(f.Context, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}, metav1.CreateOptions{})
		framework.ExpectNoError(err)
	})

	ginkgo.AfterEach(func() {
		// delete test namespace
		err := cleanup(f, ns, false)
		framework.ExpectNoError(err)
	})

	ginkgo.It("Test pod starts successfully and status is synced back to vcluster pod resource", func() {
		podName := "test"
		_, err := f.VclusterClient.CoreV1().Pods(ns).Create(f.Context, &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: podName},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  testingContainerName,
						Image: testingContainerImage,
					},
				},
			},
		}, metav1.CreateOptions{})
		framework.ExpectNoError(err)

		err = WaitForPodRunning(f, podName, ns)
		framework.ExpectNoError(err, "A pod created in the vcluster is expected to be in the Running phase eventually.")

		// get current status
		vpod, err := f.VclusterClient.CoreV1().Pods(ns).Get(f.Context, podName, metav1.GetOptions{})
		framework.ExpectNoError(err)
		pod, err := f.HostClient.CoreV1().Pods(f.VclusterNamespace).Get(f.Context, podName+"-x-"+ns+"-x-"+f.Suffix, metav1.GetOptions{})
		framework.ExpectNoError(err)

		framework.ExpectEqual(vpod.Status, pod.Status)
	})

	ginkgo.It("Test pod starts successfully with a non-default service account", func() {
		podName := "test"
		saName := "test-account"

		// create a service account
		_, err := f.VclusterClient.CoreV1().ServiceAccounts(ns).Create(f.Context, &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name: saName,
			},
		}, metav1.CreateOptions{})
		framework.ExpectNoError(err)

		// wait until the service account exists
		err = WaitForServiceAccount(f, saName, ns)
		framework.ExpectNoError(err)

		_, err = f.VclusterClient.CoreV1().Pods(ns).Create(f.Context, &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: podName},
			Spec: corev1.PodSpec{
				ServiceAccountName: saName,
				Containers: []corev1.Container{
					{
						Name:  testingContainerName,
						Image: testingContainerImage,
					},
				},
			},
		}, metav1.CreateOptions{})
		framework.ExpectNoError(err)

		err = WaitForPodRunning(f, podName, ns)
		framework.ExpectNoError(err, "A pod created in the vcluster is expected to be in the Running phase eventually.")

		// get current state
		vpod, err := f.VclusterClient.CoreV1().Pods(ns).Get(f.Context, podName, metav1.GetOptions{})
		framework.ExpectNoError(err)

		// verify that ServiceAccountName is unchanged
		framework.ExpectEqual(vpod.Spec.ServiceAccountName, saName)
	})

	ginkgo.It("Test pod contains env vars and files defined by a ConfigMap", func() {
		podName := "test"
		cmName := "test-configmap"
		cmKey := "test-key"
		cmKeyValue := "test-value"
		envVarName := "TEST_ENVVAR"
		fileName := "test.file"
		filePath := "/test-path"

		// create a configmap
		_, err := f.VclusterClient.CoreV1().ConfigMaps(ns).Create(f.Context, &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: cmName,
			},
			Data: map[string]string{
				cmKey: cmKeyValue,
			}}, metav1.CreateOptions{})
		framework.ExpectNoError(err)

		pod, err := f.VclusterClient.CoreV1().Pods(ns).Create(f.Context, &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: podName},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  testingContainerName,
						Image: testingContainerImage,
						Env: []corev1.EnvVar{
							{
								Name: envVarName,
								ValueFrom: &corev1.EnvVarSource{
									ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: cmName,
										},
										Key: cmKey,
									},
								},
							},
						},
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      "volume-name",
								MountPath: filePath,
								ReadOnly:  true,
							},
						},
					},
				},
				Volumes: []corev1.Volume{
					{
						Name: "volume-name",
						VolumeSource: corev1.VolumeSource{
							ConfigMap: &corev1.ConfigMapVolumeSource{
								LocalObjectReference: corev1.LocalObjectReference{Name: cmName},
								Items: []corev1.KeyToPath{
									{Key: cmKey, Path: fileName},
								},
							},
						},
					},
				},
			},
		}, metav1.CreateOptions{})
		framework.ExpectNoError(err)

		err = WaitForPodRunning(f, podName, ns)
		framework.ExpectNoError(err, "A pod created in the vcluster is expected to be in the Running phase eventually.")

		// execute a command in a pod to retrieve env var value
		stdout, stderr, err := podhelper.ExecBuffered(f.HostConfig, f.VclusterNamespace, translate.ObjectPhysicalName(pod), testingContainerName, []string{"sh", "-c", "echo $" + envVarName}, nil)
		framework.ExpectNoError(err)
		framework.ExpectEqual(string(stdout), cmKeyValue+"\n") // echo adds \n in the end
		framework.ExpectEqual(string(stderr), "")

		// execute a command in a pod to retrieve file content
		stdout, stderr, err = podhelper.ExecBuffered(f.HostConfig, f.VclusterNamespace, translate.ObjectPhysicalName(pod), testingContainerName, []string{"cat", filePath + "/" + fileName}, nil)
		framework.ExpectNoError(err)
		framework.ExpectEqual(string(stdout), cmKeyValue)
		framework.ExpectEqual(string(stderr), "")
	})

	ginkgo.It("Test pod contains env vars and files defined by a Secret", func() {
		podName := "test"
		secretName := "test-secret"
		secretKey := "test-key"
		secretKeyValue := "test-value"
		envVarName := "TEST_ENVVAR"
		fileName := "test.file"
		filePath := "/test-path"

		// create a configmap
		_, err := f.VclusterClient.CoreV1().Secrets(ns).Create(f.Context, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: secretName,
			},
			Data: map[string][]byte{
				secretKey: []byte(secretKeyValue),
			}}, metav1.CreateOptions{})
		framework.ExpectNoError(err)

		pod, err := f.VclusterClient.CoreV1().Pods(ns).Create(f.Context, &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: podName},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  testingContainerName,
						Image: testingContainerImage,
						Env: []corev1.EnvVar{
							{
								Name: envVarName,
								ValueFrom: &corev1.EnvVarSource{
									SecretKeyRef: &corev1.SecretKeySelector{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: secretName,
										},
										Key: secretKey,
									},
								},
							},
						},
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      "volume-name",
								MountPath: filePath,
								ReadOnly:  true,
							},
						},
					},
				},
				Volumes: []corev1.Volume{
					{
						Name: "volume-name",
						VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{
								SecretName: secretName,
								Items: []corev1.KeyToPath{
									{Key: secretKey, Path: fileName},
								},
							},
						},
					},
				},
			},
		}, metav1.CreateOptions{})
		framework.ExpectNoError(err)

		err = WaitForPodRunning(f, podName, ns)
		framework.ExpectNoError(err, "A pod created in the vcluster is expected to be in the Running phase eventually.")

		// execute a command in a pod to retrieve env var value
		stdout, stderr, err := podhelper.ExecBuffered(f.HostConfig, f.VclusterNamespace, translate.ObjectPhysicalName(pod), testingContainerName, []string{"sh", "-c", "echo $" + envVarName}, nil)
		framework.ExpectNoError(err)
		framework.ExpectEqual(string(stdout), secretKeyValue+"\n") // echo adds \n in the end
		framework.ExpectEqual(string(stderr), "")

		// execute a command in a pod to retrieve file content
		stdout, stderr, err = podhelper.ExecBuffered(f.HostConfig, f.VclusterNamespace, translate.ObjectPhysicalName(pod), testingContainerName, []string{"cat", filePath + "/" + fileName}, nil)
		framework.ExpectNoError(err)
		framework.ExpectEqual(string(stdout), secretKeyValue)
		framework.ExpectEqual(string(stderr), "")
	})
})

func cleanup(f *framework.Framework, ns string, waitUntilDeleted bool) error {
	err := f.VclusterClient.CoreV1().Namespaces().Delete(f.Context, ns, metav1.DeleteOptions{})
	if err != nil {
		if kerrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	if !waitUntilDeleted {
		return nil
	}
	return wait.PollImmediate(time.Second, framework.PollTimeout, func() (bool, error) {
		_, err = f.VclusterClient.CoreV1().Namespaces().Get(f.Context, ns, metav1.GetOptions{})
		if kerrors.IsNotFound(err) {
			return true, nil
		}
		return false, err
	})
}
