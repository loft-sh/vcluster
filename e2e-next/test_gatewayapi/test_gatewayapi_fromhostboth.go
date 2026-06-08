package test_gatewayapi

import (
	"context"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	"github.com/loft-sh/vcluster/pkg/util/random"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

const fromHostBothGatewayClassSelectorValue = "gatewayapi-fromhostboth"

// GatewayAPIFromHostBothSpec registers tests that exercise the case where
// fromHost.gateways and toHost.gatewayApi.gateways are both enabled
// (TC-34a/b). The imported Gateway's mapped tenant name must be reserved —
// tenant users cannot create a Gateway with the same namespace/name — while
// a tenant-created Gateway in a different namespace must still sync to the
// host. Depends on ENGNODE-556; until that ships the vCluster won't start
// and these specs will fail in BeforeAll.
func GatewayAPIFromHostBothSpec() {
	Describe("Gateway API fromHost+toHost gateways", labels.GatewayAPI, labels.GatewayClasses, func() {
		var (
			hostClient     ctrlclient.Client
			vClusterClient ctrlclient.Client
			vClusterName   string
			vClusterHostNS string
		)

		BeforeEach(func(ctx context.Context) {
			var err error
			scheme := runtime.NewScheme()
			Expect(corev1.AddToScheme(scheme)).To(Succeed())
			Expect(gatewayv1.Install(scheme)).To(Succeed())

			hostClient, err = ctrlclient.New(cluster.From(ctx, constants.GetHostClusterName()).KubernetesRestConfig(), ctrlclient.Options{Scheme: scheme})
			Expect(err).To(Succeed())
			vClusterClient, err = ctrlclient.New(cluster.CurrentClusterFrom(ctx).KubernetesRestConfig(), ctrlclient.Options{Scheme: scheme})
			Expect(err).To(Succeed())
			vClusterName = cluster.CurrentClusterNameFrom(ctx)
			vClusterHostNS = "vcluster-" + vClusterName

			ensureHostNamespace(ctx, hostClient, "gwapi-host-both")
		})

		It("reserves the mapped tenant Gateway name and lets a non-overlapping tenant Gateway sync to host", func(ctx context.Context) {
			suffix := random.String(6)
			class := createGatewayClass(ctx, hostClient, "gc-both-"+suffix, fromHostBothGatewayClassSelectorValue, "fromHost+toHost class")

			By("creating the host Gateway covered by mappings and waiting for the tenant mirror", func() {
				hostGW := hostGateway("gwapi-host-both", "edge", class.Name)
				createHostGateway(ctx, hostClient, hostGW)
				Eventually(func(g Gomega) {
					g.Expect(vClusterClient.Get(ctx, types.NamespacedName{Namespace: "gwapi-import-both", Name: "edge"}, &gatewayv1.Gateway{})).To(Succeed())
				}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
			})

			By("verifying the tenant cannot create a Gateway with the imported name in the mapped namespace", func() {
				createTenantNamespace(ctx, vClusterClient, "gwapi-import-both")
				colliding := tenantGateway("gwapi-import-both", "edge", class.Name)
				err := vClusterClient.Create(ctx, colliding)
				if err == nil {
					DeferCleanup(func(ctx context.Context) {
						Expect(ctrlclient.IgnoreNotFound(vClusterClient.Delete(ctx, colliding))).To(Succeed())
					})
				}
				Expect(err).To(HaveOccurred(), "tenant must not be able to create a Gateway with the imported name")
				Expect(kerrors.IsAlreadyExists(err) || kerrors.IsForbidden(err) || kerrors.IsInvalid(err)).To(BeTrue(),
					"expected AlreadyExists/Forbidden/Invalid, got: %v", err)
			})

			By("verifying the tenant can create a Gateway in a different namespace and it syncs to host", func() {
				otherNS := createTenantNamespace(ctx, vClusterClient, "gwapi-other-"+suffix)
				localGW := tenantGateway(otherNS.Name, "edge-"+suffix, class.Name)
				Expect(vClusterClient.Create(ctx, localGW)).To(Succeed())
				DeferCleanup(func(ctx context.Context) {
					Expect(ctrlclient.IgnoreNotFound(vClusterClient.Delete(ctx, localGW))).To(Succeed())
				})

				hostGWName := translate.SafeConcatName(localGW.Name, "x", otherNS.Name, "x", vClusterName)
				Eventually(func(g Gomega) {
					g.Expect(hostClient.Get(ctx, types.NamespacedName{Namespace: vClusterHostNS, Name: hostGWName}, &gatewayv1.Gateway{})).To(Succeed())
				}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
			})
		})
	})
}
