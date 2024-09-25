package pods

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	podtranslate "github.com/loft-sh/vcluster/pkg/controllers/resources/pods/token"
	"github.com/loft-sh/vcluster/pkg/util/podhelper"
	"github.com/loft-sh/vcluster/pkg/util/random"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"github.com/loft-sh/vcluster/test/framework"
	"github.com/onsi/ginkgo/v2"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	testingContainerName  = "nginx"
	testingContainerImage = "nginxinc/nginx-unprivileged"
	ipRegExp              = "(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5]).){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])"
	initialNsLabelKey     = "testing-ns-label"
	initialNsLabelValue   = "testing-ns-label-value"
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
		ns = fmt.Sprintf("e2e-syncer-pods-%d-%s", iteration, random.String(5))

		// create test namespace
		_, err := f.VClusterClient.CoreV1().Namespaces().Create(f.Context, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
			Name:   ns,
			Labels: map[string]string{initialNsLabelKey: initialNsLabelValue},
		}}, metav1.CreateOptions{})
		framework.ExpectNoError(err)
	})

	ginkgo.AfterEach(func() {
		// delete test namespace
		err := f.DeleteTestNamespace(ns, false)
		framework.ExpectNoError(err)
	})

	ginkgo.It("Test pod starts successfully and status is synced back to vcluster pod resource", func() {
		podName := "test"
		_, err := f.VClusterClient.CoreV1().Pods(ns).Create(f.Context, &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: podName},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:            testingContainerName,
						Image:           testingContainerImage,
						ImagePullPolicy: corev1.PullIfNotPresent,
						SecurityContext: f.GetDefaultSecurityContext(),
					},
				},
			},
		}, metav1.CreateOptions{})
		framework.ExpectNoError(err)

		err = f.WaitForPodRunning(podName, ns)
		framework.ExpectNoError(err, "A pod created in the vcluster is expected to be in the Running phase eventually.")

		// get current status
		vpod, err := f.VClusterClient.CoreV1().Pods(ns).Get(f.Context, podName, metav1.GetOptions{})
		framework.ExpectNoError(err)
		pPodName := translate.Default.HostName(nil, podName, ns)
		pod, err := f.HostClient.CoreV1().Pods(pPodName.Namespace).Get(f.Context, pPodName.Name, metav1.GetOptions{})
		framework.ExpectNoError(err)

		framework.ExpectEqual(vpod.Status, pod.Status)

		// check for ephemeralContainers subResource
		version, err := f.VClusterClient.Discovery().ServerVersion()
		framework.ExpectNoError(err)

		// version 1.22 and lesser than that needs legacy flag enabled
		if version != nil {
			i, err := strconv.Atoi(strings.Replace(version.Minor, "+", "", -1))
			framework.ExpectNoError(err)
			if i > 22 {
				vpod.Spec.EphemeralContainers = []corev1.EphemeralContainer{{
					EphemeralContainerCommon: corev1.EphemeralContainerCommon{
						Name:  "busybox",
						Image: "busybox:1.28",
					},
				}}
				// update ephemeralContainer
				vpod, err = f.VClusterClient.CoreV1().Pods(ns).UpdateEphemeralContainers(f.Context, vpod.Name, vpod, metav1.UpdateOptions{})
				framework.ExpectNoError(err)
				err = f.WaitForPodRunning(vpod.Name, vpod.Namespace)
				framework.ExpectNoError(err, "A pod created in the vcluster is expected to be in the Running phase eventually.")

				err = f.WaitForPodToComeUpWithEphemeralContainers(vpod.Name, vpod.Namespace)
				framework.ExpectNoError(err, "Count of ephemeralContainer is expected to be greater than 0")
			}
		}
	})

	ginkgo.It("Test pod starts successfully and readiness conditions are synced back to vcluster pod resource", func() {
		podName := "test"
		_, err := f.VClusterClient.CoreV1().Pods(ns).Create(f.Context, &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: podName},
			Spec: corev1.PodSpec{
				ReadinessGates: []corev1.PodReadinessGate{
					{ConditionType: "www.example.com/gate-1"},
				},
				Containers: []corev1.Container{
					{
						Name:            testingContainerName,
						Image:           testingContainerImage,
						ImagePullPolicy: corev1.PullIfNotPresent,
						SecurityContext: f.GetDefaultSecurityContext(),
					},
				},
			},
		}, metav1.CreateOptions{})
		framework.ExpectNoError(err)

		err = f.WaitForPodRunning(podName, ns)
		framework.ExpectNoError(err, "A pod created in the vcluster is expected to be in the Running phase eventually.")

		// get current status
		vpod, err := f.VClusterClient.CoreV1().Pods(ns).Get(f.Context, podName, metav1.GetOptions{})
		framework.ExpectNoError(err)
		pPodName := translate.Default.HostName(nil, podName, ns)
		pod, err := f.HostClient.CoreV1().Pods(pPodName.Namespace).Get(f.Context, pPodName.Name, metav1.GetOptions{})
		framework.ExpectNoError(err)
		framework.ExpectEqual(vpod.Status, pod.Status)

		// check for conditions
		vpod.Status.Conditions = append(vpod.Status.Conditions, corev1.PodCondition{Status: corev1.ConditionFalse, Type: "www.example.com/gate-1"})
		// update conditions
		vpod, err = f.VClusterClient.CoreV1().Pods(ns).UpdateStatus(f.Context, vpod, metav1.UpdateOptions{})
		framework.ExpectNoError(err)
		err = f.WaitForPodRunning(vpod.Name, vpod.Namespace)
		framework.ExpectNoError(err, "A pod created in the vcluster is expected to be in the Running phase eventually.")
		err = f.WaitForPodToComeUpWithReadinessConditions(vpod.Name, vpod.Namespace)
		framework.ExpectNoError(err, "Readiness Checks are expected to sync to physical pod")
	})

	ginkgo.It("Test pod starts successfully with a non-default service account", func() {
		podName := "test"
		saName := "test-account"

		// create a service account
		_, err := f.VClusterClient.CoreV1().ServiceAccounts(ns).Create(f.Context, &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name: saName,
			},
		}, metav1.CreateOptions{})
		framework.ExpectNoError(err)

		// wait until the service account exists
		err = f.WaitForServiceAccount(saName, ns)
		framework.ExpectNoError(err)

		_, err = f.VClusterClient.CoreV1().Pods(ns).Create(f.Context, &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: podName},
			Spec: corev1.PodSpec{
				ServiceAccountName: saName,
				Containers: []corev1.Container{
					{
						Name:            testingContainerName,
						Image:           testingContainerImage,
						ImagePullPolicy: corev1.PullIfNotPresent,
						SecurityContext: f.GetDefaultSecurityContext(),
					},
				},
			},
		}, metav1.CreateOptions{})
		framework.ExpectNoError(err)

		err = f.WaitForPodRunning(podName, ns)
		framework.ExpectNoError(err, "A pod created in the vcluster is expected to be in the Running phase eventually.")

		// get current state
		vpod, err := f.VClusterClient.CoreV1().Pods(ns).Get(f.Context, podName, metav1.GetOptions{})
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
		_, err := f.VClusterClient.CoreV1().ConfigMaps(ns).Create(f.Context, &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: cmName,
			},
			Data: map[string]string{
				cmKey: cmKeyValue,
			}}, metav1.CreateOptions{})
		framework.ExpectNoError(err)

		pod, err := f.VClusterClient.CoreV1().Pods(ns).Create(f.Context, &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: podName},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:            testingContainerName,
						Image:           testingContainerImage,
						ImagePullPolicy: corev1.PullIfNotPresent,
						SecurityContext: f.GetDefaultSecurityContext(),
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

		err = f.WaitForPodRunning(podName, ns)
		framework.ExpectNoError(err, "A pod created in the vcluster is expected to be in the Running phase eventually.")

		// execute a command in a pod to retrieve env var value
		pPodName := translate.Default.HostName(nil, pod.Name, pod.Namespace)
		stdout, stderr, err := podhelper.ExecBuffered(f.Context, f.HostConfig, pPodName.Namespace, pPodName.Name, testingContainerName, []string{"sh", "-c", "echo $" + envVarName}, nil)
		framework.ExpectNoError(err)
		framework.ExpectEqual(string(stdout), cmKeyValue+"\n") // echo adds \n in the end
		framework.ExpectEqual(string(stderr), "")

		// execute a command in a pod to retrieve file content
		stdout, stderr, err = podhelper.ExecBuffered(f.Context, f.HostConfig, pPodName.Namespace, pPodName.Name, testingContainerName, []string{"cat", filePath + "/" + fileName}, nil)
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
		_, err := f.VClusterClient.CoreV1().Secrets(ns).Create(f.Context, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: secretName,
			},
			Data: map[string][]byte{
				secretKey: []byte(secretKeyValue),
			}}, metav1.CreateOptions{})
		framework.ExpectNoError(err)

		pod, err := f.VClusterClient.CoreV1().Pods(ns).Create(f.Context, &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: podName},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:            testingContainerName,
						Image:           testingContainerImage,
						ImagePullPolicy: corev1.PullIfNotPresent,
						SecurityContext: f.GetDefaultSecurityContext(),
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

		err = f.WaitForPodRunning(podName, ns)
		framework.ExpectNoError(err, "A pod created in the vcluster is expected to be in the Running phase eventually.")

		// execute a command in a pod to retrieve env var value
		pPodName := translate.Default.HostName(nil, pod.Name, pod.Namespace)
		stdout, stderr, err := podhelper.ExecBuffered(f.Context, f.HostConfig, pPodName.Namespace, pPodName.Name, testingContainerName, []string{"sh", "-c", "echo $" + envVarName}, nil)
		framework.ExpectNoError(err)
		framework.ExpectEqual(string(stdout), secretKeyValue+"\n") // echo adds \n in the end
		framework.ExpectEqual(string(stderr), "")

		// execute a command in a pod to retrieve file content
		stdout, stderr, err = podhelper.ExecBuffered(f.Context, f.HostConfig, pPodName.Namespace, pPodName.Name, testingContainerName, []string{"cat", filePath + "/" + fileName}, nil)
		framework.ExpectNoError(err)
		framework.ExpectEqual(string(stdout), secretKeyValue)
		framework.ExpectEqual(string(stderr), "")
	})

	ginkgo.It("Test pod contains correct values in a dependent environment variables", func() {
		podName := "test"
		svcName := "myservice"
		svcPort := 80
		myProtocol := "https"

		_, err := f.VClusterClient.CoreV1().Services(ns).Create(f.Context, &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{Name: svcName},
			Spec: corev1.ServiceSpec{
				Selector: map[string]string{"doesnt": "matter"},
				Ports: []corev1.ServicePort{
					{Port: int32(svcPort)},
				},
			},
		}, metav1.CreateOptions{})
		framework.ExpectNoError(err)
		err = f.WaitForServiceInSyncerCache(svcName, ns)
		framework.ExpectNoError(err)

		pod, err := f.VClusterClient.CoreV1().Pods(ns).Create(f.Context, &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: podName},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:            testingContainerName,
						Image:           testingContainerImage,
						ImagePullPolicy: corev1.PullIfNotPresent,
						SecurityContext: f.GetDefaultSecurityContext(),
						Env: []corev1.EnvVar{
							{
								Name:  "FIRST",
								Value: "Hello",
							},
							{
								Name:  "SECOND",
								Value: "World",
							},
							{
								Name:  "HELLO_WORLD",
								Value: "$(FIRST) $(SECOND)",
							},
							{
								Name:  "ESCAPED_VAR",
								Value: "$$(FIRST)",
							},
							{
								Name:  "MY_PROTOCOL",
								Value: myProtocol,
							},
							{
								Name:  "MY_SERVICE",
								Value: "$(MY_PROTOCOL)://$(" + strings.ToUpper(svcName) + "_SERVICE_HOST):$(" + strings.ToUpper(svcName) + "_SERVICE_PORT)",
							},
						},
					},
				},
			},
		}, metav1.CreateOptions{})
		framework.ExpectNoError(err)

		err = f.WaitForPodRunning(podName, ns)
		framework.ExpectNoError(err, "A pod created in the vcluster is expected to be in the Running phase eventually.")

		// execute a command in a pod to retrieve env var value
		pPodName := translate.Default.HostName(nil, pod.Name, pod.Namespace)
		stdout, stderr, err := podhelper.ExecBuffered(f.Context, f.HostConfig, pPodName.Namespace, pPodName.Name, testingContainerName, []string{"sh", "-c", "echo $HELLO_WORLD"}, nil)
		framework.ExpectNoError(err)
		framework.ExpectEqual(string(stdout), "Hello World\n", "Dependent environment variable is expected to have its value based on the referenced environment variable(s)") // echo adds \n in the end
		framework.ExpectEqual(string(stderr), "")

		stdout, stderr, err = podhelper.ExecBuffered(f.Context, f.HostConfig, pPodName.Namespace, pPodName.Name, testingContainerName, []string{"sh", "-c", "echo $ESCAPED_VAR"}, nil)
		framework.ExpectNoError(err)
		framework.ExpectEqual(string(stdout), "$(FIRST)\n", "The double '$' symbol should be escaped") // echo adds \n in the end
		framework.ExpectEqual(string(stderr), "")

		stdout, stderr, err = podhelper.ExecBuffered(f.Context, f.HostConfig, pPodName.Namespace, pPodName.Name, testingContainerName, []string{"sh", "-c", "echo $MY_SERVICE"}, nil)
		framework.ExpectNoError(err)
		framework.ExpectMatchRegexp(string(stdout), fmt.Sprintf("^%s://%s:%d\n$", myProtocol, ipRegExp, svcPort), "Service host and port environment variables should be resolved in a dependent environment variable")
		framework.ExpectEqual(string(stderr), "")
	})

	ginkgo.It("Test pod contains namespace labels", func() {
		podName := "test"
		pod, err := f.VClusterClient.CoreV1().Pods(ns).Create(f.Context, &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: podName},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:            testingContainerName,
						Image:           testingContainerImage,
						ImagePullPolicy: corev1.PullIfNotPresent,
						SecurityContext: f.GetDefaultSecurityContext(),
					},
				},
			},
		}, metav1.CreateOptions{})
		framework.ExpectNoError(err)

		err = f.WaitForPodRunning(podName, ns)
		framework.ExpectNoError(err, "A pod created in the vcluster is expected to be in the Running phase eventually.")

		// get current physical Pod resource
		pPodName := translate.Default.HostName(nil, pod.Name, pod.Namespace)
		pPod, err := f.HostClient.CoreV1().Pods(pPodName.Namespace).Get(f.Context, pPodName.Name, metav1.GetOptions{})
		framework.ExpectNoError(err)
		pKey := translate.HostLabelNamespace(initialNsLabelKey)
		framework.ExpectHaveKey(pPod.GetLabels(), pKey)
		framework.ExpectEqual(pPod.GetLabels()[pKey], initialNsLabelValue)

		namespace, err := f.VClusterClient.CoreV1().Namespaces().Get(f.Context, ns, metav1.GetOptions{})
		framework.ExpectNoError(err)
		additionalLabelKey := "another-one"
		additionalLabelValue := "good-syncer"
		labels := namespace.GetLabels()
		labels[additionalLabelKey] = additionalLabelValue
		namespace.SetLabels(labels)

		updated := false
		err = wait.PollUntilContextTimeout(f.Context, time.Second, framework.PollTimeout, true, func(ctx context.Context) (bool, error) {
			if !updated {
				namespace, err = f.VClusterClient.CoreV1().Namespaces().Update(ctx, namespace, metav1.UpdateOptions{})
				if err != nil && !kerrors.IsConflict(err) {
					return false, err
				}
				updated = true
			}
			pPodName := translate.Default.HostName(nil, pod.Name, pod.Namespace)
			pPod, err = f.HostClient.CoreV1().Pods(pPodName.Namespace).Get(ctx, pPodName.Name, metav1.GetOptions{})
			framework.ExpectNoError(err)
			pKey = translate.HostLabelNamespace(additionalLabelKey)
			if value, ok := pPod.GetLabels()[pKey]; ok {
				framework.ExpectEqual(value, additionalLabelValue)
				return true, nil
			}
			return false, nil
		})
		framework.ExpectNoError(err)
	})

	ginkgo.It("Test if service account tokens are synced and mounted through secrets", func() {
		podName := "test-nginx"

		pod, err := f.VClusterClient.CoreV1().Pods(ns).Create(f.Context, &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      podName,
				Namespace: ns,
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  podName,
						Image: "nginx",
					},
				},
			},
		}, metav1.CreateOptions{})

		framework.ExpectNoError(err)
		err = f.WaitForPodRunning(podName, ns)

		framework.ExpectNoError(err, "A pod created in the vcluster is expected to be in the Running phase eventually.")

		// get current physical Pod resource
		pPodName := translate.Default.HostName(nil, pod.Name, pod.Namespace)
		pPod, err := f.HostClient.CoreV1().Pods(pPodName.Namespace).Get(f.Context, pPodName.Name, metav1.GetOptions{})
		framework.ExpectNoError(err)

		// make sure service account token annotation is not present
		_, ok := pPod.GetAnnotations()[podtranslate.PodServiceAccountTokenSecretName]
		framework.ExpectEqual(ok, false, "service account token annotation should not be present")

		// make sure the secret is created in host cluster
		secretName := translate.Default.HostName(nil, fmt.Sprintf("%s-sa-token", pod.Name), ns)
		_, err = f.HostClient.CoreV1().Secrets(secretName.Namespace).Get(f.Context, secretName.Name, metav1.GetOptions{})
		framework.ExpectNoError(err)

		// make sure the project volume for path 'token' is now using a secret instead of service account
		for _, volume := range pPod.Spec.Volumes {
			if volume.Projected != nil {
				for _, source := range volume.Projected.Sources {
					if source.Secret != nil {
						framework.ExpectEqual(source.Secret.Name, secretName.Name)
					}
				}
			}
		}
	})
})
