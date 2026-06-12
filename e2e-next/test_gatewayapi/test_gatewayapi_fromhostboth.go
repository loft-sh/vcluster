package test_gatewayapi

import (
	"context"

	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	"github.com/loft-sh/vcluster/pkg/util/random"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

const fromHostBothGatewayClassSelectorValue = "gatewayapi-fromhostboth"

func GatewayAPIFromHostBothSpec() {
	Describe("Gateway API fromHost+toHost gateways", labels.GatewayAPI, labels.GatewayClasses, func() {
		var (
			hostClient     ctrlclient.Client
			vClusterClient ctrlclient.Client
			vClusterName   string
			vClusterHostNS string
		)

		BeforeEach(func(ctx context.Context) {
			c := newGatewayAPIClients(ctx)
			hostClient = c.HostClient
			vClusterClient = c.VClusterClient
			vClusterName = c.VClusterName
			vClusterHostNS = c.VClusterHostNS

			ensureHostNamespace(ctx, hostClient, "gwapi-host-both")
		})

		It("reserves the mapped tenant Gateway name and lets a non-overlapping tenant Gateway sync to host", func(ctx context.Context) {
			suffix := random.String(6)
			class := createGatewayClass(ctx, hostClient, "gc-both-"+suffix, fromHostBothGatewayClassSelectorValue, "fromHost+toHost class")
			hostGWName := "edge-" + suffix

			By("creating the host Gateway covered by mappings and waiting for the tenant mirror", func() {
				hostGW := hostGateway("gwapi-host-both", hostGWName, class.Name)
				createHostGateway(ctx, hostClient, hostGW)
				Eventually(func(g Gomega) {
					g.Expect(vClusterClient.Get(ctx, types.NamespacedName{Namespace: "gwapi-import-both", Name: hostGWName}, &gatewayv1.Gateway{})).To(Succeed())
				}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
			})

			By("verifying the tenant cannot create a Gateway with the imported name in the mapped namespace", func() {
				colliding := tenantGateway("gwapi-import-both", hostGWName, class.Name)
				err := vClusterClient.Create(ctx, colliding)
				Expect(kerrors.IsAlreadyExists(err)).To(BeTrue(),
					"tenant must not be able to create a Gateway with the imported name; got: %v", err)
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
