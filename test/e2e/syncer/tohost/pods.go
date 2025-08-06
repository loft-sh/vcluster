package tohost

import (
	"fmt"

	"github.com/loft-sh/vcluster/pkg/util/random"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"github.com/loft-sh/vcluster/test/framework"
	"github.com/onsi/ginkgo/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testingContainerName  = "nginx"
	testingContainerImage = "nginxinc/nginx-unprivileged:stable-alpine3.20-slim"
	ipRegExp              = "(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5]).){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])"
	initialNsLabelKey     = "testing-ns-label"
	initialNsLabelValue   = "testing-ns-label-value"
)

var _ = ginkgo.Describe("Test sync NetworkPolicy from vCluster to host", ginkgo.Ordered, func() {
	var (
		f  *framework.Framework
		ns string
	)

	ginkgo.JustBeforeEach(func() {
		f = framework.DefaultFramework
		ns = fmt.Sprintf("e2e-syncer-pods-%s", random.String(5))

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

	ginkgo.It("Verify base64 encoded values in js patch expression", func() {
		podName := "test-js-patch"
		podAnnotationValue := "dGVzdGluZyBzeW5jIGZyb20gdkNsdXN0ZXIK"
		podAnnotationValueModified := "dGVzdGluZyBqcyBwYXRjaCBmcm9tIHZDbHVzdGVyCg=="
		ginkgo.By("Create a pod with annotation set.")
		vPod, err := f.VClusterClient.CoreV1().Pods(ns).Create(f.Context, &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: podName,
				Annotations: map[string]string{
					"vcluster-annotation-js": podAnnotationValue,
				},
			},
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

		pPodName := translate.Default.HostName(nil, vPod.Name, vPod.Namespace)
		pPod, err := f.HostClient.CoreV1().Pods(pPodName.Namespace).Get(f.Context, pPodName.Name, metav1.GetOptions{})
		framework.ExpectNoError(err)

		framework.ExpectEqual(podAnnotationValueModified, pPod.Annotations["vcluster-annotation-js"])

		err = f.VClusterClient.CoreV1().Pods(ns).Delete(f.Context, podName, metav1.DeleteOptions{})
		framework.ExpectNoError(err)
	})
})
