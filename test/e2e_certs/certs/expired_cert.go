package certs

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"time"

	certscmd "github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/certs"
	"github.com/loft-sh/vcluster/pkg/certs"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/test/framework"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = ginkgo.Describe("vCluster cert rotation tests", ginkgo.Ordered, func() {
	var (
		f *framework.Framework
	)

	ginkgo.JustBeforeEach(func() {
		f = framework.DefaultFramework
	})

	ginkgo.It("should obtain the current cert secret of vCluster", func() {
		_, err := f.HostClient.CoreV1().Secrets(f.VClusterNamespace).Get(f.Context, certs.CertSecretName(f.VClusterName), metav1.GetOptions{})
		framework.ExpectNoError(err)
	})

	ginkgo.It("setting validity of ca cert of vCluster to 30 seconds", func() {
		os.Setenv("DEVELOPMENT", "true")
		os.Setenv("VCLUSTER_CERTS_VALIDITYPERIOD", "30s")
		defer os.Unsetenv("DEVELOPMENT")
		defer os.Unsetenv("VCLUSTER_CERTS_VALIDITYPERIOD")

		certsCmd := certscmd.NewCertsCmd(&flags.GlobalFlags{Namespace: f.VClusterNamespace})
		certsCmd.SetArgs([]string{"rotate-ca", f.VClusterName})

		err := certsCmd.Execute()
		framework.ExpectNoError(err)
	})

	ginkgo.It("should check if CA cert of vCluster is expired", func() {
		gomega.Eventually(func() error {
			secret, err := f.HostClient.CoreV1().Secrets(f.VClusterNamespace).Get(
				f.Context, certs.CertSecretName(f.VClusterName), metav1.GetOptions{})
			if err != nil {
				return err
			}

			certPEM := secret.Data["ca.crt"]
			block, _ := pem.Decode(certPEM)
			if block == nil {
				return fmt.Errorf("failed to decode PEM block")
			}

			cert, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				return err
			}

			if cert.NotAfter.Before(time.Now()) {
				return nil
			}
			return fmt.Errorf("CA cert not expired yet (expires at %s)", cert.NotAfter)
		}).
			WithPolling(time.Second).
			WithTimeout(framework.PollTimeoutLong).
			Should(gomega.Succeed())
	})

	ginkgo.It("Checking Expired Status of vCluster CA cert", func() {
		certsCmd := certscmd.NewCertsCmd(&flags.GlobalFlags{Namespace: f.VClusterNamespace})
		certsCmd.SetArgs([]string{"check", f.VClusterName})

		err := certsCmd.Execute()
		framework.ExpectNoError(err)
	})

	ginkgo.It("Rotating expired CA cert of vCluster", func() {
		certsCmd := certscmd.NewCertsCmd(&flags.GlobalFlags{Namespace: f.VClusterNamespace})
		certsCmd.SetArgs([]string{"rotate-ca", f.VClusterName})

		err := certsCmd.Execute()
		framework.ExpectNoError(err)
	})

	ginkgo.It("should wait until the vCluster is ready again", func() {
		framework.ExpectNoError(f.WaitForVClusterReady())
		gomega.Eventually(func() error {
			pods, err := f.HostClient.CoreV1().Pods(f.VClusterNamespace).List(f.Context, metav1.ListOptions{
				LabelSelector: "app=vcluster,release=" + f.VClusterName,
			})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(pods.Items).NotTo(gomega.BeEmpty())

			for _, pod := range pods.Items {
				gomega.Expect(pod.Status.ContainerStatuses).NotTo(gomega.BeEmpty(),
					"pod %s should have container statuses", pod.Name)

				for i, container := range pod.Status.ContainerStatuses {
					gomega.Expect(container.State.Running).NotTo(gomega.BeNil(),
						"container %d in pod %s should be running", i, pod.Name)
					gomega.Expect(container.Ready).To(gomega.BeTrue(),
						"container %d in pod %s should be ready", i, pod.Name)
				}
			}

			return nil
		}).WithPolling(time.Second).
			WithTimeout(framework.PollTimeoutLong).
			Should(gomega.Succeed())
	})

	ginkgo.It("Checking new Expiry Date and time of vCluster CA cert", func() {
		certsCmd := certscmd.NewCertsCmd(&flags.GlobalFlags{Namespace: f.VClusterNamespace})
		certsCmd.SetArgs([]string{"check", f.VClusterName})

		err := certsCmd.Execute()
		framework.ExpectNoError(err)
	})

	ginkgo.It("Checking new validity date of CA cert of vCluster", func() {
		secret, err := f.HostClient.CoreV1().Secrets(f.VClusterNamespace).Get(
			f.Context, certs.CertSecretName(f.VClusterName), metav1.GetOptions{})
		framework.ExpectNoError(err)

		certPEM := secret.Data["ca.crt"]

		block, _ := pem.Decode(certPEM)
		gomega.Expect(block).NotTo(gomega.BeNil(), "Failed to decode PEM block")

		cert, err := x509.ParseCertificate(block.Bytes)
		framework.ExpectNoError(err)

		gomega.Expect(cert.NotAfter.After(time.Now())).To(gomega.BeTrue(), "CA cert is valid")
	})
})
