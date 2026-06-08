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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

const rgDisabledGatewayClassSelectorValue = "gatewayapi-rgdisabled"

// GatewayAPIReferenceGrantDisabledSpec registers tests asserting that tenant
// ReferenceGrants stay tenant-local when referenceGrants.enabled is false
// (TC-04d).
func GatewayAPIReferenceGrantDisabledSpec() {
	Describe("Gateway API referenceGrants disabled", labels.GatewayAPI, func() {
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
			Expect(gatewayv1beta1.Install(scheme)).To(Succeed())

			hostClient, err = ctrlclient.New(cluster.From(ctx, constants.GetHostClusterName()).KubernetesRestConfig(), ctrlclient.Options{Scheme: scheme})
			Expect(err).To(Succeed())
			vClusterClient, err = ctrlclient.New(cluster.CurrentClusterFrom(ctx).KubernetesRestConfig(), ctrlclient.Options{Scheme: scheme})
			Expect(err).To(Succeed())
			vClusterName = cluster.CurrentClusterNameFrom(ctx)
			vClusterHostNS = "vcluster-" + vClusterName
		})

		It("does not sync tenant-created ReferenceGrants when referenceGrants.enabled is false", func(ctx context.Context) {
			suffix := random.String(6)
			class := createGatewayClass(ctx, hostClient, "gc-rgdis-"+suffix, rgDisabledGatewayClassSelectorValue, "rg-disabled class")
			Eventually(func(g Gomega) {
				g.Expect(vClusterClient.Get(ctx, types.NamespacedName{Name: class.Name}, &gatewayv1.GatewayClass{})).To(Succeed())
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())

			backend := createTenantNamespace(ctx, vClusterClient, "rgdis-backend-"+suffix)
			grant := &gatewayv1beta1.ReferenceGrant{
				ObjectMeta: metav1.ObjectMeta{Name: "allow-" + suffix, Namespace: backend.Name},
				Spec: gatewayv1beta1.ReferenceGrantSpec{
					From: []gatewayv1beta1.ReferenceGrantFrom{{Group: gatewayv1.Group(gatewayv1.GroupName), Kind: gatewayv1.Kind("HTTPRoute"), Namespace: gatewayv1.Namespace("rgdis-frontend-" + suffix)}},
					To:   []gatewayv1beta1.ReferenceGrantTo{{Group: gatewayv1.Group(""), Kind: gatewayv1.Kind("Service")}},
				},
			}
			Expect(vClusterClient.Create(ctx, grant)).To(Succeed())
			DeferCleanup(func(ctx context.Context) {
				Expect(ctrlclient.IgnoreNotFound(vClusterClient.Delete(ctx, grant))).To(Succeed())
			})

			hostGrantName := translate.SafeConcatName(grant.Name, "x", backend.Name, "x", vClusterName)
			Consistently(func(g Gomega) {
				err := hostClient.Get(ctx, types.NamespacedName{Namespace: vClusterHostNS, Name: hostGrantName}, &gatewayv1beta1.ReferenceGrant{})
				g.Expect(kerrors.IsNotFound(err)).To(BeTrue(), "tenant ReferenceGrant must not sync when referenceGrants.enabled is false")
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutShort).Should(Succeed())

			By("confirming the tenant still owns the ReferenceGrant", func() {
				got := &gatewayv1beta1.ReferenceGrant{}
				Expect(vClusterClient.Get(ctx, ctrlclient.ObjectKeyFromObject(grant), got)).To(Succeed())
				Expect(got.Spec.From).To(HaveLen(1))
			})
		})
	})
}
