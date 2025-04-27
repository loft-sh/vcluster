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

var _ = ginkgo.Describe("Test fromHost sync of configmaps", ginkgo.Ordered, func() {
	var (
		f                   *framework.Framework
		configMap1          *corev1.ConfigMap
		configMap2          *corev1.ConfigMap
		configMap3          *corev1.ConfigMap
		configMap4          *corev1.ConfigMap
		configMap5          *corev1.ConfigMap
		configMap6          *corev1.ConfigMap
		configMap7          *corev1.ConfigMap
		cm1Name             = "dummy"
		cm1HostNamespace    = "from-host-sync-test"
		cmsVirtualNamespace = "barfoo"
		cm2HostNamespace    = "default"
		cm2HostName         = "my.cm"
		cm2VirtualName      = "cm-my"
		cm3Name             = "from-vcluster-ns"
		cm3HostNamespace    = "vcluster"
		cm3virtualNamespace = "my-new-ns"
		cm4Name             = "my-cm-4"
		cm4HostNamespace    = "vcluster"
		podName             = "my-pod"
		cm5Name             = "specific-cm"
		cm5HostNamespace    = "vcluster"
		cm5virtualNamespace = "my-virtual-namespace"
		cm6Name             = "to-be-deleted"
		cm6Namespace        = "my-ns"
		cm7Name             = "cm-same-ns"
		cm7Namespace        = "same-ns"
	)

	ginkgo.BeforeAll(func() {
		f = framework.DefaultFramework
		configMap1 = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cm1Name,
				Namespace: cm1HostNamespace,
			},
			Data: map[string]string{
				"BOO_BAR":     "hello-world",
				"ANOTHER_ENV": "another-hello-world",
			},
		}
		configMap2 = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cm2HostName,
				Namespace: cm2HostNamespace,
			},
			Data: map[string]string{
				"ENV_FROM_DEFAULT_NS":         "one",
				"ANOTHER_ENV_FROM_DEFAULT_NS": "two",
			},
		}
		configMap3 = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cm3Name,
				Namespace: cm3HostNamespace,
			},
			Data: map[string]string{
				"VAL1": "abcdef",
				"VAL2": "defgh",
			},
		}
		configMap4 = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cm4Name,
				Namespace: cm4HostNamespace,
			},
			Data: map[string]string{
				"VAL3": "ghijkl",
				"VAL4": "defg",
			},
		}
		configMap5 = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cm5Name,
				Namespace: cm5HostNamespace,
			},
			Data: map[string]string{
				"key5": "value5",
				"key6": "value6",
			},
		}
		configMap6 = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cm6Name,
				Namespace: cm6Namespace,
			},
			Data: map[string]string{
				"key6": "value6",
			},
		}
		configMap7 = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cm7Name,
				Namespace: cm7Namespace,
			},
			Data: map[string]string{
				"key7": "value7",
			},
		}

	})

	ginkgo.AfterAll(func() {
		framework.ExpectNoError(f.HostClient.CoreV1().ConfigMaps(configMap1.GetNamespace()).Delete(f.Context, configMap1.GetName(), metav1.DeleteOptions{}))
		framework.ExpectNoError(f.HostClient.CoreV1().ConfigMaps(configMap2.GetNamespace()).Delete(f.Context, configMap2.GetName(), metav1.DeleteOptions{}))
		framework.ExpectNoError(f.HostClient.CoreV1().ConfigMaps(configMap4.GetNamespace()).Delete(f.Context, configMap4.GetName(), metav1.DeleteOptions{}))

		framework.ExpectNoError(f.VClusterClient.CoreV1().Pods(cmsVirtualNamespace).Delete(f.Context, podName, metav1.DeleteOptions{}))
		// verify whether config maps got deleted from virtual too
		_, err := f.VClusterClient.CoreV1().ConfigMaps(cmsVirtualNamespace).Get(f.Context, cm1Name, metav1.GetOptions{})
		framework.ExpectError(err, "expected config map to be deleted")
		_, err = f.VClusterClient.CoreV1().ConfigMaps(cmsVirtualNamespace).Get(f.Context, cm2VirtualName, metav1.GetOptions{})
		framework.ExpectError(err, "expected config map to be deleted")
		_, err = f.VClusterClient.CoreV1().ConfigMaps(cmsVirtualNamespace).Get(f.Context, cm4Name, metav1.GetOptions{})
		framework.ExpectError(err, "expected config map to be deleted")

		framework.ExpectNoError(f.HostClient.CoreV1().Namespaces().Delete(f.Context, cm1HostNamespace, metav1.DeleteOptions{}))
	})

	ginkgo.It("create config maps in host", func() {
		_, err := f.HostClient.CoreV1().ConfigMaps(configMap1.GetNamespace()).Create(f.Context, configMap1, metav1.CreateOptions{})
		framework.ExpectNoError(err)
		_, err = f.HostClient.CoreV1().ConfigMaps(configMap2.GetNamespace()).Create(f.Context, configMap2, metav1.CreateOptions{})
		framework.ExpectNoError(err)
		_, err = f.HostClient.CoreV1().ConfigMaps(configMap4.GetNamespace()).Create(f.Context, configMap4, metav1.CreateOptions{})
		framework.ExpectNoError(err)
	})

	ginkgo.It("ConfigMaps are synced to virtual", func() {
		gomega.Eventually(func() bool {
			virtual1, err := f.VClusterClient.CoreV1().ConfigMaps(cmsVirtualNamespace).Get(f.Context, cm1Name, metav1.GetOptions{})
			if err != nil {
				return false
			}
			if !reflect.DeepEqual(virtual1.Data, configMap1.Data) {
				f.Log.Errorf("expected %#v in virtual.Data got %#v", configMap1.Data, virtual1.Data)
				return false
			}
			return true
		}).
			WithPolling(time.Second).
			WithTimeout(framework.PollTimeout / 4).
			Should(gomega.BeTrue())

		gomega.Eventually(func() bool {
			virtual2, err := f.VClusterClient.CoreV1().ConfigMaps(cmsVirtualNamespace).Get(f.Context, cm2VirtualName, metav1.GetOptions{})
			if err != nil {
				return false
			}
			if !reflect.DeepEqual(virtual2.Data, configMap2.Data) {
				f.Log.Errorf("expected %#v in virtual.Data got %#v", configMap2.Data, virtual2.Data)
				return false
			}
			return true
		}).
			WithPolling(time.Second).
			WithTimeout(framework.PollTimeout / 4).
			Should(gomega.BeTrue())

		gomega.Eventually(func() bool {
			virtual4, err := f.VClusterClient.CoreV1().ConfigMaps(cmsVirtualNamespace).Get(f.Context, cm4Name, metav1.GetOptions{})
			if err != nil {
				return false
			}
			if !reflect.DeepEqual(virtual4.Data, configMap4.Data) {
				f.Log.Errorf("expected %#v in virtual.Data got %#v", configMap4.Data, virtual4.Data)
				return false
			}
			return true
		}).
			WithPolling(time.Second).
			WithTimeout(framework.PollTimeout / 4).
			Should(gomega.BeTrue())
	})

	ginkgo.It("update in host config map should get synced to virtual", func() {
		freshHostConfigMap, err := f.HostClient.CoreV1().ConfigMaps(configMap1.GetNamespace()).Get(f.Context, configMap1.GetName(), metav1.GetOptions{})
		framework.ExpectNoError(err)
		freshHostConfigMap.Data["UPDATED_ENV"] = "one"
		if freshHostConfigMap.Labels == nil {
			freshHostConfigMap.Labels = make(map[string]string, 1)
		}
		freshHostConfigMap.Labels["updated-label"] = "updated-value"
		if freshHostConfigMap.Annotations == nil {
			freshHostConfigMap.Annotations = make(map[string]string, 1)
		}
		freshHostConfigMap.Annotations["updated-annotation"] = "updated-value"
		_, err = f.HostClient.CoreV1().ConfigMaps(freshHostConfigMap.GetNamespace()).Update(f.Context, freshHostConfigMap, metav1.UpdateOptions{})
		framework.ExpectNoError(err)
		gomega.Eventually(func() bool {
			updatedCm1, err := f.VClusterClient.CoreV1().ConfigMaps(cmsVirtualNamespace).Get(f.Context, cm1Name, metav1.GetOptions{})
			if err != nil {
				return false
			}
			return updatedCm1.Data["UPDATED_ENV"] == "one" && updatedCm1.Labels["updated-label"] == "updated-value" && updatedCm1.Annotations["updated-annotation"] == "updated-value"

		}).
			WithPolling(time.Second).
			WithTimeout(framework.PollTimeout).
			Should(gomega.BeTrue())
	})

	ginkgo.It("synced config maps can be used as env source for pod", func() {
		// make sure that default service account exists before creating a pod.
		// this is purely to avoid test flakes.
		framework.ExpectNoError(f.WaitForServiceAccount("default", cmsVirtualNamespace))
		optional := false
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      podName,
				Namespace: cmsVirtualNamespace,
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:            "default",
						Image:           "nginxinc/nginx-unprivileged:stable-alpine3.20-slim",
						ImagePullPolicy: corev1.PullIfNotPresent,
						SecurityContext: f.GetDefaultSecurityContext(),
						EnvFrom: []corev1.EnvFromSource{
							{
								ConfigMapRef: &corev1.ConfigMapEnvSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: cm1Name,
									},
									Optional: &optional,
								},
							},
							{
								ConfigMapRef: &corev1.ConfigMapEnvSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: cm2VirtualName,
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

	ginkgo.It("Delete configmap in vCluster should be sync again by host", func() {
		oldConfigMap, _ := f.VClusterClient.CoreV1().ConfigMaps(cmsVirtualNamespace).Get(f.Context, cm2VirtualName, metav1.GetOptions{})

		uidBeforeDeletion := oldConfigMap.UID

		framework.ExpectNoError(f.VClusterClient.CoreV1().ConfigMaps(cmsVirtualNamespace).Delete(f.Context, cm2VirtualName, metav1.DeleteOptions{}))

		gomega.Eventually(func() bool {
			newConfigMap, err := f.VClusterClient.CoreV1().ConfigMaps(cmsVirtualNamespace).Get(f.Context, cm2VirtualName, metav1.GetOptions{})
			if err != nil {
				return false
			}
			return newConfigMap.Data["ENV_FROM_DEFAULT_NS"] == "one" && newConfigMap.Data["ANOTHER_ENV_FROM_DEFAULT_NS"] == "two" && uidBeforeDeletion != newConfigMap.UID
		}).
			WithPolling(time.Second).
			WithTimeout(framework.PollTimeout).
			Should(gomega.BeTrue())
	})

	ginkgo.It("confimap edited in vCluster should be overwritten by host", func() {
		vClusterMap, err := f.VClusterClient.CoreV1().ConfigMaps(cmsVirtualNamespace).Get(f.Context, cm2VirtualName, metav1.GetOptions{})
		framework.ExpectNoError(err)

		vClusterMap.Data = make(map[string]string, 1)

		vClusterMap.Data["new-value"] = "will-be-overwritten"

		_, err = f.VClusterClient.CoreV1().ConfigMaps(cmsVirtualNamespace).Update(f.Context, vClusterMap, metav1.UpdateOptions{})
		framework.ExpectNoError(err)

		gomega.Eventually(func() bool {
			defaultCmValues, err := f.VClusterClient.CoreV1().ConfigMaps(cmsVirtualNamespace).Get(f.Context, cm2VirtualName, metav1.GetOptions{})
			if err != nil {
				return false
			}
			_, updatedExist := defaultCmValues.Data["new-value"]
			return defaultCmValues.Data["ENV_FROM_DEFAULT_NS"] == "one" && defaultCmValues.Data["ANOTHER_ENV_FROM_DEFAULT_NS"] == "two" && !updatedExist
		}).
			WithPolling(time.Second).
			WithTimeout(framework.PollTimeout).
			Should(gomega.BeTrue())
	})

	ginkgo.It("Syncs all ConfigMaps from vCluster's host namespace to my-new-ns in vCluster", func() {
		_, err := f.HostClient.CoreV1().ConfigMaps(configMap3.GetNamespace()).Create(f.Context, configMap3, metav1.CreateOptions{})
		framework.ExpectNoError(err)

		gomega.Eventually(func() bool {
			defaultCmValues, err := f.VClusterClient.CoreV1().ConfigMaps(cm3virtualNamespace).Get(f.Context, cm3Name, metav1.GetOptions{})
			if err != nil {
				return false
			}
			return defaultCmValues.Data["VAL1"] == "abcdef" && defaultCmValues.Data["VAL2"] == "defgh"
		}).
			WithPolling(time.Second).
			WithTimeout(framework.PollTimeout).
			Should(gomega.BeTrue())
	})

	ginkgo.It("Sync specific config maps from host namespace to a different namespace in vCluster", func() {
		_, err := f.HostClient.CoreV1().ConfigMaps(configMap5.GetNamespace()).Create(f.Context, configMap5, metav1.CreateOptions{})
		framework.ExpectNoError(err)

		gomega.Eventually(func() bool {
			cmValues, err := f.VClusterClient.CoreV1().ConfigMaps(cm5virtualNamespace).Get(f.Context, cm5Name, metav1.GetOptions{})
			if err != nil {
				return false
			}
			return cmValues.Data["key6"] == "value6" && cmValues.Data["key5"] == "value5"
		}).
			WithPolling(time.Second).
			WithTimeout(framework.PollTimeout).
			Should(gomega.BeTrue())
	})

	ginkgo.It("Delete configmap in Host, configmap in vCluster is also deleted", func() {
		_, err := f.HostClient.CoreV1().Namespaces().Create(f.Context, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: cm6Namespace}}, metav1.CreateOptions{})
		framework.ExpectNoError(err)

		_, err = f.HostClient.CoreV1().ConfigMaps(configMap6.GetNamespace()).Create(f.Context, configMap6, metav1.CreateOptions{})
		framework.ExpectNoError(err)

		gomega.Eventually(func() bool {
			cmValues, err := f.VClusterClient.CoreV1().ConfigMaps(cm6Namespace).Get(f.Context, cm6Name, metav1.GetOptions{})
			if err != nil {
				return false
			}
			return cmValues.Data["key6"] == "value6"
		}).
			WithPolling(time.Second).
			WithTimeout(framework.PollTimeout).
			Should(gomega.BeTrue())

		framework.ExpectNoError(f.HostClient.CoreV1().ConfigMaps(configMap6.GetNamespace()).Delete(f.Context, configMap6.GetName(), metav1.DeleteOptions{}))
		gomega.Eventually(func() bool {
			_, err := f.VClusterClient.CoreV1().ConfigMaps(cm6Namespace).Get(f.Context, cm6Name, metav1.GetOptions{})
			return err != nil
		}).
			WithPolling(time.Second).
			WithTimeout(framework.PollTimeout).
			Should(gomega.BeTrue())

		_, err = f.VClusterClient.CoreV1().ConfigMaps(cm6Namespace).Get(f.Context, cm6Name, metav1.GetOptions{})
		framework.ExpectError(err)

		framework.ExpectNoError(f.HostClient.CoreV1().Namespaces().Delete(f.Context, cm6Namespace, metav1.DeleteOptions{}))
	})

	ginkgo.It("Sync all config maps from host namespaces to the same namespace in vCluster", func() {
		_, err := f.HostClient.CoreV1().Namespaces().Create(f.Context, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: cm7Namespace}}, metav1.CreateOptions{})
		framework.ExpectNoError(err)

		_, err = f.HostClient.CoreV1().ConfigMaps(configMap7.GetNamespace()).Create(f.Context, configMap7, metav1.CreateOptions{})
		framework.ExpectNoError(err)

		gomega.Eventually(func() bool {
			cmValues, err := f.VClusterClient.CoreV1().ConfigMaps(cm7Namespace).Get(f.Context, cm7Name, metav1.GetOptions{})
			if err != nil {
				return false
			}
			return cmValues.Data["key7"] == "value7"
		}).
			WithPolling(time.Second).
			WithTimeout(framework.PollTimeout).
			Should(gomega.BeTrue())

		framework.ExpectNoError(f.HostClient.CoreV1().ConfigMaps(configMap7.GetNamespace()).Delete(f.Context, configMap7.GetName(), metav1.DeleteOptions{}))

		framework.ExpectNoError(f.HostClient.CoreV1().Namespaces().Delete(f.Context, cm7Namespace, metav1.DeleteOptions{}))
	})

})
