package endpointslices

import (
	"testing"

	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertesting "github.com/loft-sh/vcluster/pkg/syncer/testing"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"gotest.tools/assert"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestTranslateSelectorlessServiceLongName covers the one path where the
// kubernetes.io/service-name label is load-bearing: a selector-less Service, for
// which vcluster sets the label itself instead of letting the host endpoint
// controller regenerate it. It drives the real translate() path and asserts the
// label stays within the 63-byte limit and still resolves to the host Service
// name, even when the virtual Service name is long enough that a naive
// concatenation would overflow.
func TestTranslateSelectorlessServiceLongName(t *testing.T) {
	// 55 chars: short enough to be a legal virtual Service name (<= 63), long
	// enough that the naive "<svc>-x-<ns>-x-<vc>" concat overflows 63 bytes.
	const longSvcName = "frontend-checkout-payments-orders-inventory-billing-svc"
	const namespace = "default"

	// EndpointSlice as it arrives for a selector-less Service: the only link back
	// to its Service is the kubernetes.io/service-name label.
	vEndpointSlice := &discoveryv1.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "eps-selectorless",
			Namespace: namespace,
			Labels: map[string]string{
				translate.K8sServiceNameLabel: longSvcName,
			},
		},
		AddressType: discoveryv1.AddressTypeIPv4,
		Endpoints: []discoveryv1.Endpoint{
			{Addresses: []string{"1.2.3.4"}},
		},
	}

	syncertesting.RunTests(t, []*syncertesting.SyncTest{
		{
			Name: "Selector-less service with a long name keeps a valid service-name label",
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, fakeSyncer := newFakeSyncer(t, ctx)

				hostEndpointSlice := fakeSyncer.translate(syncCtx, vEndpointSlice)
				gotLabel := hostEndpointSlice.GetLabels()[translate.K8sServiceNameLabel]

				// The label must stay within the 63-byte limit, or the host
				// apiserver rejects the EndpointSlice and it never syncs.
				assert.Assert(t, len(gotLabel) <= 63, "service-name label is %d bytes, exceeds the 63-byte limit: %q", len(gotLabel), gotLabel)

				// It must also resolve to the actual host Service name, so the
				// valid-length slice still points at the right Service.
				wantLabel := translate.Default.HostName(nil, longSvcName, namespace).Name
				assert.Equal(t, gotLabel, wantLabel)
			},
		},
	})
}
