package test_gatewayapi

import (
	"context"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	gatewayv1alpha3 "sigs.k8s.io/gateway-api/apis/v1alpha3"
	gatewayv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

type gatewayAPIClients struct {
	HostClient     ctrlclient.Client
	VClusterClient ctrlclient.Client
	VClusterName   string
	VClusterHostNS string
}

func newGatewayAPIClients(ctx context.Context, installExtendedGatewaySchemes bool) gatewayAPIClients {
	GinkgoHelper()

	scheme := runtime.NewScheme()
	Expect(corev1.AddToScheme(scheme)).To(Succeed())
	Expect(gatewayv1.Install(scheme)).To(Succeed())
	if installExtendedGatewaySchemes {
		Expect(gatewayv1alpha2.Install(scheme)).To(Succeed())
		Expect(gatewayv1alpha3.Install(scheme)).To(Succeed())
		Expect(gatewayv1beta1.Install(scheme)).To(Succeed())
	}

	hostClient, err := ctrlclient.New(cluster.From(ctx, constants.GetHostClusterName()).KubernetesRestConfig(), ctrlclient.Options{Scheme: scheme})
	Expect(err).To(Succeed())
	vClusterClient, err := ctrlclient.New(cluster.CurrentClusterFrom(ctx).KubernetesRestConfig(), ctrlclient.Options{Scheme: scheme})
	Expect(err).To(Succeed())

	vClusterName := cluster.CurrentClusterNameFrom(ctx)
	return gatewayAPIClients{
		HostClient:     hostClient,
		VClusterClient: vClusterClient,
		VClusterName:   vClusterName,
		VClusterHostNS: "vcluster-" + vClusterName,
	}
}
