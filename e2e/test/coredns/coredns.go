package coredns

import (
	"fmt"
	"time"

	"github.com/loft-sh/vcluster/e2e/framework"
	"github.com/loft-sh/vcluster/pkg/util/podhelper"
	"github.com/loft-sh/vcluster/pkg/util/random"
	"github.com/onsi/ginkgo"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = ginkgo.Describe("CoreDNS resolves host names correctly", func() {
	var (
		f         *framework.Framework
		iteration int
		ns        string
		curlPod   *corev1.Pod
	)

	ginkgo.JustBeforeEach(func() {
		// use default framework
		f = framework.DefaultFramework
		iteration++
		ns = fmt.Sprintf("e2e-coredns-%d-%s", iteration, random.RandomString(5))

		// create test namespace
		_, err := f.VclusterClient.CoreV1().Namespaces().Create(f.Context, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}, metav1.CreateOptions{})
		framework.ExpectNoError(err)

		curlPod, err = f.CreateCurlPod(ns)
		framework.ExpectNoError(err)

		err = f.WaitForPodRunning(curlPod.GetName(), ns)
		framework.ExpectNoError(err, "A pod created in the vcluster is expected to be in the Running phase eventually.")
	})

	ginkgo.AfterEach(func() {
		// delete test namespace
		err := f.DeleteTestNamespace(ns, false)
		framework.ExpectNoError(err)
	})

	ginkgo.It("Test Service is reachable via it's hostname", func() {
		pod, service, err := f.CreateNginxPodAndService(ns)
		framework.ExpectNoError(err)

		err = f.WaitForPodRunning(pod.GetName(), ns)
		framework.ExpectNoError(err, "A pod created in the vcluster is expected to be in the Running phase eventually.")

		// sleep to reduce the rate of pod/exec calls made when checking if service is reacheable
		time.Sleep(time.Second * 10)
		framework.DefaultFramework.TestServiceIsEventuallyReachable(curlPod, service)
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
			// sleep to reduce the rate of pod/exec calls
			time.Sleep(time.Second * 10)
			url := fmt.Sprintf("https://%s:%d/healthz", hostname, node.Status.DaemonEndpoints.KubeletEndpoint.Port)
			cmd := []string{"curl", "-k", "-s", "--show-error", url}
			stdoutBuffer, stderrBuffer, err := podhelper.ExecBuffered(f.VclusterConfig, ns, curlPod.GetName(), curlPod.Spec.Containers[0].Name, cmd, nil)
			framework.ExpectNoError(err)
			framework.ExpectEmpty(stderrBuffer)
			framework.ExpectEqual(string(stdoutBuffer), "ok")
		}
	})
})
