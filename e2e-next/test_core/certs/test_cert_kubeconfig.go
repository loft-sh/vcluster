package certs

import (
	"context"
	"crypto/tls"
	"errors"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/e2e-framework/pkg/setup/suite"
	certscmd "github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/certs"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// DescribeCertKubeConfig registers tests that verify TLS behaviour in kubeconfigs
// after cert rotation. After leaf rotation the old TLS config must still work
// (same CA). After CA rotation the old TLS config must fail with a certificate
// verification error (new CA means old root cert is untrusted).
// Ordered because the specs form a lifecycle: establish baseline -> rotate leaf ->
// verify old TLS still works -> rotate CA -> verify old TLS fails.
func DescribeCertKubeConfig(vcluster suite.Dependency) bool {
	return Describe("vCluster cert rotation kubeconfig TLS behaviour",
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

				// restConfigBefore holds the TLS config captured at test start.
				// After leaf rotation it must remain valid; after CA rotation it must not.
				restConfigBefore *rest.Config
			)

			BeforeAll(func(ctx context.Context) context.Context {
				hostClient = cluster.KubeClientFrom(ctx, constants.GetHostClusterName())
				Expect(hostClient).NotTo(BeNil())
				vClusterName = cluster.CurrentClusterNameFrom(ctx)
				vClusterNamespace = "vcluster-" + vClusterName
				return ctx
			})

			// Establish a fresh connection and capture the initial TLS config.
			It("should connect to vCluster and capture baseline TLS config", func(ctx context.Context) {
				restConfig, vClusterClient := reconnectVCluster(ctx, vClusterName, vClusterNamespace)
				restConfigBefore = restConfig

				_, err := vClusterClient.CoreV1().Pods("default").List(ctx, metav1.ListOptions{})
				Expect(err).To(Succeed(), "baseline vCluster client should work")
			})

			// Rotate leaf certs (CA is unchanged).
			It("should rotate the leaf certs", func(_ context.Context) {
				certsCmd := certscmd.NewCertsCmd(&flags.GlobalFlags{Namespace: vClusterNamespace})
				certsCmd.SetArgs([]string{"rotate", vClusterName})
				Expect(certsCmd.Execute()).To(Succeed())
			})

			// Wait for recovery after leaf rotation.
			It("should have all pods ready after leaf rotation", func(ctx context.Context) {
				waitForPodsReady(ctx, hostClient, vClusterNamespace, vClusterName, constants.PollingTimeoutLong)
			})

			// After leaf rotation: old TLS config (same CA root) must still work.
			It("should still accept the old TLS config after leaf rotation", func(ctx context.Context) {
				newRestConfig, _ := reconnectVCluster(ctx, vClusterName, vClusterNamespace)

				// Override the TLS config with the pre-rotation one.
				newRestConfig.TLSClientConfig = restConfigBefore.TLSClientConfig

				oldTLSClient, err := kubernetes.NewForConfig(newRestConfig)
				Expect(err).To(Succeed(), "building client with old TLS config after leaf rotation")

				_, err = oldTLSClient.CoreV1().Pods("default").List(ctx, metav1.ListOptions{})
				Expect(err).To(Succeed(),
					"old TLS config should remain valid after leaf-only rotation (CA unchanged)")
			})

			// Rotate the CA (new CA means old TLS root cert becomes untrusted).
			It("should rotate the CA cert", func(_ context.Context) {
				certsCmd := certscmd.NewCertsCmd(&flags.GlobalFlags{Namespace: vClusterNamespace})
				certsCmd.SetArgs([]string{"rotate-ca", vClusterName})
				Expect(certsCmd.Execute()).To(Succeed())
			})

			// Wait for recovery after CA rotation.
			It("should have all pods ready after CA rotation", func(ctx context.Context) {
				waitForPodsReady(ctx, hostClient, vClusterNamespace, vClusterName, constants.PollingTimeoutLong)
			})

			// After CA rotation: old TLS config must fail with a certificate verification error.
			It("should reject the old TLS config after CA rotation", func(ctx context.Context) {
				newRestConfig, _ := reconnectVCluster(ctx, vClusterName, vClusterNamespace)

				// Override the TLS config with the pre-rotation one (old CA root).
				newRestConfig.TLSClientConfig = restConfigBefore.TLSClientConfig

				oldTLSClient, err := kubernetes.NewForConfig(newRestConfig)
				Expect(err).To(Succeed(), "building client with old TLS config")

				_, err = oldTLSClient.CoreV1().Pods("default").List(ctx, metav1.ListOptions{})

				var certErr *tls.CertificateVerificationError
				Expect(errors.As(err, &certErr)).To(BeTrue(),
					"expected a TLS certificate verification error after CA rotation, got: %v", err)
			})

			// Reconnect to restore a working proxy for subsequent suites.
			It("should reconnect to vCluster after CA rotation", func(ctx context.Context) {
				reconnectVCluster(ctx, vClusterName, vClusterNamespace)
			})
		},
	)
}
