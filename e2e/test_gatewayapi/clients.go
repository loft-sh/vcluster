package test_gatewayapi

import (
	"context"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e/constants"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

type gatewayAPIClients struct {
	HostClient     ctrlclient.Client
	VClusterClient ctrlclient.Client
	VClusterName   string
	VClusterHostNS string
}

func newGatewayAPIClients(ctx context.Context) gatewayAPIClients {
	GinkgoHelper()

	scheme := runtime.NewScheme()
	Expect(corev1.AddToScheme(scheme)).To(Succeed())
	Expect(gatewayv1.Install(scheme)).To(Succeed())

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
