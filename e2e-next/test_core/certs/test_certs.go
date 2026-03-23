package certs

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"time"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/e2e-framework/pkg/setup/suite"
	certscmd "github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/certs"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	"github.com/loft-sh/vcluster/pkg/certs"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// DescribeCertRotation registers cert rotation tests against the given vCluster.
// These tests are Ordered because they form a lifecycle: check -> rotate CA -> verify -> rotate leaf -> verify.
func DescribeCertRotation(vcluster suite.Dependency) bool {
	return Describe("vCluster cert rotation",
		Ordered,
		labels.Core,
		labels.Security,
		cluster.Use(vcluster),
		cluster.Use(clusters.HostCluster),
		func() {
			var (
				hostClient        kubernetes.Interface
				vClusterName      string
				vClusterNamespace string
			)

			BeforeAll(func(ctx context.Context) {
				hostClient = cluster.KubeClientFrom(ctx, constants.GetHostClusterName())
				Expect(hostClient).NotTo(BeNil())
				vClusterName = cluster.CurrentClusterNameFrom(ctx)
				vClusterNamespace = "vcluster-" + vClusterName
			})

			// Spec 1: verify vCluster is ready and all pods are running
			It("should have all vCluster pods running and ready", func(ctx context.Context) {
				Eventually(func(g Gomega) {
					pods, err := hostClient.CoreV1().Pods(vClusterNamespace).List(ctx, metav1.ListOptions{
						LabelSelector: "app=vcluster,release=" + vClusterName,
					})
					g.Expect(err).To(Succeed())
					g.Expect(pods.Items).NotTo(BeEmpty(), "no vcluster pods found")

					for _, pod := range pods.Items {
						g.Expect(pod.Status.ContainerStatuses).NotTo(BeEmpty(),
							"pod %s should have container statuses", pod.Name)
						for _, container := range pod.Status.ContainerStatuses {
							g.Expect(container.State.Running).NotTo(BeNil(),
								"container %s in pod %s should be running", container.Name, pod.Name)
							g.Expect(container.Ready).To(BeTrue(),
								"container %s in pod %s should be ready", container.Name, pod.Name)
						}
					}
				}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
			})

			// Spec 2 depends on spec 1: cert secret exists
			It("should have the cert secret", func(ctx context.Context) {
				_, err := hostClient.CoreV1().Secrets(vClusterNamespace).Get(ctx,
					certs.CertSecretName(vClusterName), metav1.GetOptions{})
				Expect(err).To(Succeed())
			})

			// Spec 3: check certs command works
			It("should report cert expiry via vcluster certs check", func(_ context.Context) {
				certsCmd := certscmd.NewCertsCmd(&flags.GlobalFlags{Namespace: vClusterNamespace})
				certsCmd.SetArgs([]string{"check", vClusterName})
				Expect(certsCmd.Execute()).To(Succeed())
			})

			// Spec 4 depends on spec 2: rotate CA
			It("should rotate the CA cert", func(_ context.Context) {
				certsCmd := certscmd.NewCertsCmd(&flags.GlobalFlags{Namespace: vClusterNamespace})
				certsCmd.SetArgs([]string{"rotate-ca", vClusterName})
				Expect(certsCmd.Execute()).To(Succeed())
			})

			// Spec 5 depends on spec 4: wait for vCluster to recover after CA rotation
			It("should have all pods ready after CA rotation", func(ctx context.Context) {
				Eventually(func(g Gomega) {
					pods, err := hostClient.CoreV1().Pods(vClusterNamespace).List(ctx, metav1.ListOptions{
						LabelSelector: "app=vcluster,release=" + vClusterName,
					})
					g.Expect(err).To(Succeed())
					g.Expect(pods.Items).NotTo(BeEmpty())
					for _, pod := range pods.Items {
						g.Expect(pod.Status.ContainerStatuses).NotTo(BeEmpty())
						for _, container := range pod.Status.ContainerStatuses {
							g.Expect(container.State.Running).NotTo(BeNil(),
								"container %s in pod %s not running after CA rotation", container.Name, pod.Name)
							g.Expect(container.Ready).To(BeTrue(),
								"container %s in pod %s not ready after CA rotation", container.Name, pod.Name)
						}
					}
				}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutVeryLong).Should(Succeed())
			})

			// Spec 6 depends on spec 4: verify new CA cert is valid
			It("should have a valid CA cert after rotation", func(ctx context.Context) {
				secret, err := hostClient.CoreV1().Secrets(vClusterNamespace).Get(ctx,
					certs.CertSecretName(vClusterName), metav1.GetOptions{})
				Expect(err).To(Succeed())

				block, _ := pem.Decode(secret.Data["ca.crt"])
				Expect(block).NotTo(BeNil(), "failed to decode CA cert PEM")

				cert, err := x509.ParseCertificate(block.Bytes)
				Expect(err).To(Succeed())
				Expect(cert.NotAfter.After(time.Now())).To(BeTrue(), "CA cert should be valid")
			})

			// Spec 7: rotate leaf certs
			It("should rotate the leaf certs", func(_ context.Context) {
				certsCmd := certscmd.NewCertsCmd(&flags.GlobalFlags{Namespace: vClusterNamespace})
				certsCmd.SetArgs([]string{"rotate", vClusterName})
				Expect(certsCmd.Execute()).To(Succeed())
			})

			// Spec 8 depends on spec 7: wait for vCluster to recover after leaf rotation
			It("should have all pods ready after leaf cert rotation", func(ctx context.Context) {
				Eventually(func(g Gomega) {
					pods, err := hostClient.CoreV1().Pods(vClusterNamespace).List(ctx, metav1.ListOptions{
						LabelSelector: "app=vcluster,release=" + vClusterName,
					})
					g.Expect(err).To(Succeed())
					g.Expect(pods.Items).NotTo(BeEmpty())
					for _, pod := range pods.Items {
						g.Expect(pod.Status.ContainerStatuses).NotTo(BeEmpty())
						for _, container := range pod.Status.ContainerStatuses {
							g.Expect(container.State.Running).NotTo(BeNil(),
								"container %s in pod %s not running after leaf rotation", container.Name, pod.Name)
							g.Expect(container.Ready).To(BeTrue(),
								"container %s in pod %s not ready after leaf rotation", container.Name, pod.Name)
						}
					}
				}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutVeryLong).Should(Succeed())
			})

			// Spec 9 depends on spec 7: verify new leaf cert is valid
			It("should have a valid leaf cert after rotation", func(ctx context.Context) {
				secret, err := hostClient.CoreV1().Secrets(vClusterNamespace).Get(ctx,
					certs.CertSecretName(vClusterName), metav1.GetOptions{})
				Expect(err).To(Succeed())

				block, _ := pem.Decode(secret.Data["apiserver-etcd-client.crt"])
				Expect(block).NotTo(BeNil(), "failed to decode leaf cert PEM")

				cert, err := x509.ParseCertificate(block.Bytes)
				Expect(err).To(Succeed())
				Expect(cert.NotAfter.After(time.Now())).To(BeTrue(), "leaf cert should be valid")
			})
		},
	)
}
