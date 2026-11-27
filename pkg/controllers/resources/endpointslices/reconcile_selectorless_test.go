package endpointslices

import (
	"testing"

	"github.com/loft-sh/vcluster/pkg/syncer"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertesting "github.com/loft-sh/vcluster/pkg/syncer/testing"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

// TestReconcileSelectorlessServiceLongName drives the full Reconcile path
// (ReconcileStart -> SyncToHost -> translate) for a selector-less Service whose
// name is long enough to overflow the 63-byte kubernetes.io/service-name label.
// The expected host object uses translate.Default.HostName, which hashes names
// past 63 bytes, so the test asserts the synced slice carries a hashed, valid
// label rather than the raw concatenation.
func TestReconcileSelectorlessServiceLongName(t *testing.T) {
	const longSvcName = "frontend-checkout-payments-orders-inventory-billing-svc" // 55 bytes
	const namespace = "test"

	// A selector-less Service: no Spec.Selector, so the host endpoint controller
	// will not regenerate its slice -- vcluster must sync it and set the label.
	vService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      longSvcName,
			Namespace: namespace,
		},
	}
	vEndpointSlice := &discoveryv1.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "eps-selectorless",
			Namespace:       namespace,
			ResourceVersion: "999",
			Labels: map[string]string{
				translate.K8sServiceNameLabel: longSvcName,
			},
		},
		AddressType: discoveryv1.AddressTypeIPv4,
		Endpoints: []discoveryv1.Endpoint{
			{Addresses: []string{"1.2.3.4"}},
		},
	}

	hostSliceName := translate.Default.HostName(nil, vEndpointSlice.Name, namespace).Name
	// The service-name label must be the HASHED host Service name (<= 63 bytes),
	// not the raw "<svc>-x-<ns>-x-<vc>" concatenation (which would be 74 bytes).
	hostSvcLabel := translate.Default.HostName(nil, longSvcName, namespace).Name

	expectedEndpointSlice := &discoveryv1.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{
			Name:      hostSliceName,
			Namespace: namespace,
			Annotations: map[string]string{
				translate.NameAnnotation:          vEndpointSlice.Name,
				translate.NamespaceAnnotation:     namespace,
				translate.KindAnnotation:          discoveryv1.SchemeGroupVersion.WithKind("EndpointSlice").String(),
				translate.HostNamespaceAnnotation: namespace,
				translate.UIDAnnotation:           "",
				translate.HostNameAnnotation:      hostSliceName,
			},
			Labels: map[string]string{
				translate.K8sServiceNameLabel: hostSvcLabel,
				translate.NamespaceLabel:      namespace,
			},
		},
		AddressType: discoveryv1.AddressTypeIPv4,
		Endpoints: []discoveryv1.Endpoint{
			{Addresses: []string{"1.2.3.4"}},
		},
	}

	syncertesting.RunTests(t, []*syncertesting.SyncTest{
		{
			Name: "Selector-less service with a long name syncs with a hashed, valid service-name label",
			InitialVirtualState: []runtime.Object{
				vEndpointSlice,
				vService,
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				discoveryv1.SchemeGroupVersion.WithKind("EndpointSlice"): {
					expectedEndpointSlice,
				},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				_, fakeSyncer := newFakeSyncer(t, ctx)
				syncController, err := syncer.NewSyncController(ctx, fakeSyncer)
				assert.NilError(t, err)

				_, err = syncController.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{
					Namespace: vEndpointSlice.Namespace,
					Name:      vEndpointSlice.Name,
				}})
				assert.NilError(t, err)
			},
		},
	})
}
