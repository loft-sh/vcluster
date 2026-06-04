package scheme

import (
	agentclusterv1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/cluster/v1"
	agentstoragev1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/storage/v1"
	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	"github.com/loft-sh/vcluster/pkg/apis"
	apidiscoveryv2 "k8s.io/api/apidiscovery/v2"
	apidiscoveryv2beta1 "k8s.io/api/apidiscovery/v2beta1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	apiregistrationv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	metricsv1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	gatewayv1alpha3 "sigs.k8s.io/gateway-api/apis/v1alpha3"
	gatewayv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

var Scheme = runtime.NewScheme()

func init() {
	_ = clientgoscheme.AddToScheme(Scheme)
	// API extensions are not in the above scheme set,
	// and must thus be added separately.
	_ = apiextensionsv1beta1.AddToScheme(Scheme)
	_ = apiextensionsv1.AddToScheme(Scheme)
	_ = apiregistrationv1.AddToScheme(Scheme)
	_ = apidiscoveryv2beta1.AddToScheme(Scheme)
	_ = apidiscoveryv2.AddToScheme(Scheme)
	_ = metricsv1beta1.AddToScheme(Scheme)

	// Register the fake conversions
	_ = apis.RegisterConversions(Scheme)

	// Register GatewayAPI CRDs
	_ = gatewayv1.Install(Scheme)
	_ = gatewayv1alpha2.Install(Scheme)
	_ = gatewayv1alpha3.Install(Scheme)
	_ = gatewayv1beta1.Install(Scheme)

	// Register Loft CRDs
	_ = agentstoragev1.AddToScheme(Scheme)
	_ = agentclusterv1.AddToScheme(Scheme)
	_ = managementv1.AddToScheme(Scheme)
}
