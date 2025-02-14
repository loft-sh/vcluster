package fromhost

import (
	"reflect"
	"strings"
	"time"

	"github.com/loft-sh/vcluster/test/framework"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = ginkgo.Describe("Secrets are synced to host and can be used in Pods", ginkgo.Ordered, func() {
	var (
		f                       *framework.Framework
		secret1                 *corev1.Secret
		secret2                 *corev1.Secret
		secret3                 *corev1.Secret
		secret1Name             = "dummy"
		secret1HostNamespace    = "from-host-sync-test-2"
		secretsVirtualNamespace = "barfoo2"
		secret2HostNamespace    = "default"
		secret2HostName         = "my-secret"
		secret2VirtualName      = "secret-my"
		secret3HostName         = "my-secret-in-default-ns"
		secret3VirtualName      = "secret-from-default-ns"
		podName                 = "my-pod"
	)

	ginkgo.BeforeAll(func() {
		f = framework.DefaultFramework
		secret1 = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secret1Name,
				Namespace: secret1HostNamespace,
			},
			Data: map[string][]byte{
				"BOO_BAR":     []byte("hello-world"),
				"ANOTHER_ENV": []byte("another-hello-world"),
			},
		}
		secret2 = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secret2HostName,
				Namespace: secret2HostNamespace,
			},
			Data: map[string][]byte{
				"ENV_FROM_DEFAULT_NS":         []byte("one"),
				"ANOTHER_ENV_FROM_DEFAULT_NS": []byte("two"),
			},
		}
		secret3 = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secret3HostName,
				Namespace: f.VClusterNamespace,
			},
			Data: map[string][]byte{
				"dummy":   []byte("one"),
				"dummy-2": []byte("two"),
			},
		}

	})

	ginkgo.AfterAll(func() {
		framework.ExpectNoError(f.HostClient.CoreV1().Secrets(secret1.GetNamespace()).Delete(f.Context, secret1.GetName(), metav1.DeleteOptions{}))
		framework.ExpectNoError(f.HostClient.CoreV1().Secrets(secret2.GetNamespace()).Delete(f.Context, secret2.GetName(), metav1.DeleteOptions{}))
		framework.ExpectNoError(f.HostClient.CoreV1().Secrets(secret3.GetNamespace()).Delete(f.Context, secret3.GetName(), metav1.DeleteOptions{}))

		framework.ExpectNoError(f.VClusterClient.CoreV1().Pods(secretsVirtualNamespace).Delete(f.Context, podName, metav1.DeleteOptions{}))
		// verify whether secrets got deleted from virtual too
		_, err := f.VClusterClient.CoreV1().Secrets(secretsVirtualNamespace).Get(f.Context, secret1Name, metav1.GetOptions{})
		framework.ExpectError(err, "expected secret to be deleted")
		_, err = f.VClusterClient.CoreV1().Secrets(secretsVirtualNamespace).Get(f.Context, secret2VirtualName, metav1.GetOptions{})
		framework.ExpectError(err, "expected secret to be deleted")

		framework.ExpectNoError(f.HostClient.CoreV1().Namespaces().Delete(f.Context, secret1HostNamespace, metav1.DeleteOptions{}))
	})

	ginkgo.It("create secrets in host", func() {
		_, err := f.HostClient.CoreV1().Secrets(secret1.GetNamespace()).Create(f.Context, secret1, metav1.CreateOptions{})
		framework.ExpectNoError(err)
		_, err = f.HostClient.CoreV1().Secrets(secret2.GetNamespace()).Create(f.Context, secret2, metav1.CreateOptions{})
		framework.ExpectNoError(err)
		_, err = f.HostClient.CoreV1().Secrets(secret3.GetNamespace()).Create(f.Context, secret3, metav1.CreateOptions{})
		framework.ExpectNoError(err)
	})

	ginkgo.It("Secrets are synced to virtual", func() {
		gomega.Eventually(func() bool {
			virtual1, err := f.VClusterClient.CoreV1().Secrets(secretsVirtualNamespace).Get(f.Context, secret1Name, metav1.GetOptions{})
			if err != nil {
				return false
			}
			if !reflect.DeepEqual(virtual1.Data, secret1.Data) {
				f.Log.Errorf("expected %#v in virtual.Data got %#v", secret1.Data, virtual1.Data)
				return false
			}
			return true
		}).
			WithPolling(time.Second).
			WithTimeout(framework.PollTimeout / 4).
			Should(gomega.BeTrue())

		gomega.Eventually(func() bool {
			virtual2, err := f.VClusterClient.CoreV1().Secrets(secretsVirtualNamespace).Get(f.Context, secret2VirtualName, metav1.GetOptions{})
			if err != nil {
				return false
			}
			if !reflect.DeepEqual(virtual2.Data, secret2.Data) {
				f.Log.Errorf("expected %#v in virtual.Data got %#v", secret2.Data, virtual2.Data)
				return false
			}
			return true
		}).
			WithPolling(time.Second).
			WithTimeout(framework.PollTimeout / 4).
			Should(gomega.BeTrue())

		gomega.Eventually(func() bool {
			virtual3, err := f.VClusterClient.CoreV1().Secrets(secretsVirtualNamespace).Get(f.Context, secret3VirtualName, metav1.GetOptions{})
			if err != nil {
				return false
			}
			if !reflect.DeepEqual(virtual3.Data, secret3.Data) {
				f.Log.Errorf("expected %#v in virtual.Data got %#v", secret3.Data, virtual3.Data)
				return false
			}
			return true
		}).
			WithPolling(time.Second).
			WithTimeout(framework.PollTimeout / 4).
			Should(gomega.BeTrue())
	})

	ginkgo.It("update in host secret should get synced to virtual", func() {
		freshHostSecret, err := f.HostClient.CoreV1().Secrets(secret1.GetNamespace()).Get(f.Context, secret1Name, metav1.GetOptions{})
		framework.ExpectNoError(err)
		freshHostSecret.Data["UPDATED_ENV"] = []byte("one")
		if freshHostSecret.Labels == nil {
			freshHostSecret.Labels = make(map[string]string, 1)
		}
		freshHostSecret.Labels["updated-label"] = "updated-value"
		if freshHostSecret.Annotations == nil {
			freshHostSecret.Annotations = make(map[string]string, 1)
		}
		freshHostSecret.Annotations["updated-annotation"] = "updated-value"
		_, err = f.HostClient.CoreV1().Secrets(freshHostSecret.GetNamespace()).Update(f.Context, freshHostSecret, metav1.UpdateOptions{})
		framework.ExpectNoError(err)
		gomega.Eventually(func() bool {
			updatedCm1, err := f.VClusterClient.CoreV1().Secrets(secretsVirtualNamespace).Get(f.Context, secret1Name, metav1.GetOptions{})
			if err != nil {
				return false
			}
			return string(updatedCm1.Data["UPDATED_ENV"]) == "one" && updatedCm1.Labels["updated-label"] == "updated-value" && updatedCm1.Annotations["updated-annotation"] == "updated-value"

		}).
			WithPolling(time.Second).
			WithTimeout(framework.PollTimeout).
			Should(gomega.BeTrue())
	})

	ginkgo.It("synced secret can be used as env source for pod", func() {
		optional := false
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      podName,
				Namespace: secretsVirtualNamespace,
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:            "default",
						Image:           "nginxinc/nginx-unprivileged",
						ImagePullPolicy: corev1.PullIfNotPresent,
						SecurityContext: f.GetDefaultSecurityContext(),
						EnvFrom: []corev1.EnvFromSource{
							{
								SecretRef: &corev1.SecretEnvSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: secret1Name,
									},
									Optional: &optional,
								},
							},
							{
								SecretRef: &corev1.SecretEnvSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: secret2VirtualName,
									},
									Optional: &optional,
								},
							},
						},
					},
				},
			},
		}
		createdPod, err := f.VClusterClient.CoreV1().Pods(pod.GetNamespace()).Create(f.Context, pod, metav1.CreateOptions{})
		framework.ExpectNoError(err)

		framework.ExpectNoError(f.WaitForPodRunning(createdPod.GetName(), createdPod.GetNamespace()))

		output, err := f.ExecCommandInThePod(createdPod.GetName(), createdPod.GetNamespace(), []string{"sh", "-c", "printenv"})
		framework.ExpectNoError(err, output)

		envVars := strings.Split(strings.TrimSpace(output), "\n")
		envs := make(map[string]string, len(envVars))
		for _, envVar := range envVars {
			parts := strings.Split(envVar, "=")
			envs[parts[0]] = strings.ReplaceAll(parts[1], "\r", "")
		}

		gomega.Expect(envs["ANOTHER_ENV_FROM_DEFAULT_NS"]).To(gomega.Equal("two"))
		gomega.Expect(envs["UPDATED_ENV"]).To(gomega.Equal("one"))
		gomega.Expect(envs["ANOTHER_ENV"]).To(gomega.Equal("another-hello-world"))
		gomega.Expect(envs["BOO_BAR"]).To(gomega.Equal("hello-world"))
		gomega.Expect(envs["ENV_FROM_DEFAULT_NS"]).To(gomega.Equal("one"))
	})

})
