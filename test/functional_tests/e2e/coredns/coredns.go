package coredns

import (
	"fmt"

	"github.com/loft-sh/vcluster/pkg/coredns"
	"github.com/loft-sh/vcluster/pkg/util/podhelper"
	"github.com/loft-sh/vcluster/pkg/util/random"
	"github.com/loft-sh/vcluster/test/framework"
	"github.com/onsi/ginkgo/v2"
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
		ns = fmt.Sprintf("e2e-coredns-%d-%s", iteration, random.String(5))

		// create test namespace
		_, err := f.VClusterClient.CoreV1().Namespaces().Create(f.Context, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}, metav1.CreateOptions{})
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
		framework.DefaultFramework.TestServiceIsEventuallyReachable(curlPod, service)
	})

	ginkgo.It("Test nodes (fake) kubelet is reachable via node hostname", func() {
		nodes, err := f.VClusterClient.CoreV1().Nodes().List(f.Context, metav1.ListOptions{})
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
			url := fmt.Sprintf("https://%s:%d/healthz", hostname, node.Status.DaemonEndpoints.KubeletEndpoint.Port)
			cmd := []string{"curl", "-k", "-s", "--show-error", url}
			stdoutBuffer, stderrBuffer, err := podhelper.ExecBuffered(f.Context, f.VClusterConfig, ns, curlPod.GetName(), curlPod.Spec.Containers[0].Name, cmd, nil)
			framework.ExpectNoError(err)
			framework.ExpectEmpty(stderrBuffer)
			framework.ExpectEqual(string(stdoutBuffer), "ok")
		}
	})
	ginkgo.It("Test coredns uses pinned image version", func() {
		coreDNSName, coreDNSNamespace := "coredns", "kube-system"
		coreDNSDeployment, err := f.VClusterClient.AppsV1().Deployments(coreDNSNamespace).Get(f.Context, coreDNSName, metav1.GetOptions{})
		framework.ExpectNoError(err)
		framework.ExpectEqual(len(coreDNSDeployment.Spec.Template.Spec.Containers), 1)
		framework.ExpectEqual(coreDNSDeployment.Spec.Template.Spec.Containers[0].Image, coredns.DefaultImage)
		// these are images with known security vulnerabilities.
		framework.ExpectNotEqual(coreDNSDeployment.Spec.Template.Spec.Containers[0].Image, "1.11.1")
		framework.ExpectNotEqual(coreDNSDeployment.Spec.Template.Spec.Containers[0].Image, "1.11.0")
	})
})
