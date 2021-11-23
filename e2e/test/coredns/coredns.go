package coredns

import (
	"fmt"

	"github.com/loft-sh/vcluster/e2e/framework"
	"github.com/loft-sh/vcluster/pkg/util/podhelper"
	"github.com/onsi/ginkgo"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = ginkgo.Describe("CoreDNS resolves host names correctly", func() {
	var (
		f                 *framework.Framework
		iteration         int
		ns                string
		curlPodName       = "curl"
		curlContainerName = "curl"
	)

	ginkgo.JustBeforeEach(func() {
		// use default framework
		f = framework.DefaultFramework
		iteration++
		ns = fmt.Sprintf("e2e-coredns-%d", iteration)
		// execute cleanup in case previous e2e test were terminated prematurely
		err := f.DeleteTestNamespace(ns, true)
		framework.ExpectNoError(err)

		// create test namespace
		_, err = f.VclusterClient.CoreV1().Namespaces().Create(f.Context, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}, metav1.CreateOptions{})
		framework.ExpectNoError(err)

		_, err = f.VclusterClient.CoreV1().Pods(ns).Create(f.Context, &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: curlPodName},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:            curlContainerName,
						Image:           "curlimages/curl",
						ImagePullPolicy: corev1.PullIfNotPresent,
						SecurityContext: f.GetDefaultSecurityContext(),
						Command:         []string{"sleep"},
						Args:            []string{"9999"},
					},
				},
			},
		}, metav1.CreateOptions{})
		framework.ExpectNoError(err)

		err = f.WaitForPodRunning(curlPodName, ns)
		framework.ExpectNoError(err, "A pod created in the vcluster is expected to be in the Running phase eventually.")
	})

	ginkgo.AfterEach(func() {
		// delete test namespace
		err := f.DeleteTestNamespace(ns, false)
		framework.ExpectNoError(err)
	})

	ginkgo.It("Test Service is reachable via it's hostname", func() {
		podName := "httpbin"
		serviceName := "myservice"
		labels := map[string]string{"app": "httpbin"}
		_, err := f.VclusterClient.CoreV1().Services(ns).Create(f.Context, &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      serviceName,
				Namespace: ns,
			},
			Spec: corev1.ServiceSpec{
				Selector: labels,
				Ports: []corev1.ServicePort{
					{Port: 8080},
				},
			},
		}, metav1.CreateOptions{})
		framework.ExpectNoError(err)

		_, err = f.VclusterClient.CoreV1().Pods(ns).Create(f.Context, &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:   podName,
				Labels: labels,
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:            podName,
						Image:           "nginxinc/nginx-unprivileged",
						ImagePullPolicy: corev1.PullIfNotPresent,
						SecurityContext: f.GetDefaultSecurityContext(),
					},
				},
			},
		}, metav1.CreateOptions{})
		framework.ExpectNoError(err)

		err = f.WaitForPodRunning(podName, ns)
		framework.ExpectNoError(err, "A pod created in the vcluster is expected to be in the Running phase eventually.")

		url := fmt.Sprintf("http://%s:8080/", serviceName)
		cmd := []string{"curl", "-s", "--show-error", "-o", "/dev/null", "-w", "%{http_code}", url}
		stdoutBuffer, stderrBuffer, err := podhelper.ExecBuffered(f.VclusterConfig, ns, curlPodName, curlContainerName, cmd, nil)
		framework.ExpectNoError(err)
		framework.ExpectEmpty(stderrBuffer)
		framework.ExpectEqual(string(stdoutBuffer), "200")
	})

	ginkgo.It("Test nodes (fake) kubelet is reachable via node hostname", func() {
		nodes, err := f.VclusterClient.CoreV1().Nodes().List(f.Context, metav1.ListOptions{})
		framework.ExpectNoError(err)
		for _, node := range nodes.Items {
			hostname := node.Name
			for _, address := range node.Status.Addresses {
				if address.Type == corev1.NodeHostName {
					hostname = address.Address
					break
				}
			}
			url := fmt.Sprintf("https://%s:%d/healthz", hostname, node.Status.DaemonEndpoints.KubeletEndpoint.Port)
			cmd := []string{"curl", "-k", "-s", "--show-error", url}
			stdoutBuffer, stderrBuffer, err := podhelper.ExecBuffered(f.VclusterConfig, ns, curlPodName, curlContainerName, cmd, nil)
			framework.ExpectNoError(err)
			framework.ExpectEmpty(stderrBuffer)
			framework.ExpectEqual(string(stdoutBuffer), "ok")
		}
	})
})
