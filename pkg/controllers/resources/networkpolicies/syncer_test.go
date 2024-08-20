package networkpolicies

import (
	"testing"

	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertesting "github.com/loft-sh/vcluster/pkg/syncer/testing"
	"gotest.tools/assert"
	"k8s.io/utils/ptr"

	"github.com/loft-sh/vcluster/pkg/util/translate"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestSync(t *testing.T) {
	somePorts := []networkingv1.NetworkPolicyPort{
		{
			Port: &intstr.IntOrString{Type: intstr.Int, IntVal: 32},
		},
		{
			Port:    &intstr.IntOrString{Type: intstr.Int, IntVal: 1024},
			EndPort: ptr.To(int32(2 ^ 32)),
		},
		{
			Port: &intstr.IntOrString{Type: intstr.String, StrVal: "namedport"},
		},
	}
	vObjectMeta := metav1.ObjectMeta{
		Name:      "testnetworkpolicy",
		Namespace: "test",
	}
	vBaseSpec := networkingv1.NetworkPolicySpec{
		PodSelector: metav1.LabelSelector{
			MatchLabels: map[string]string{"mykey": "mylabel"},
			MatchExpressions: []metav1.LabelSelectorRequirement{
				{
					Key:      "secondkey",
					Operator: metav1.LabelSelectorOpIn,
					Values:   []string{"label-A", "label-B"},
				},
			},
		},
	}
	pBaseSpec := networkingv1.NetworkPolicySpec{
		PodSelector: metav1.LabelSelector{
			MatchLabels: map[string]string{
				translate.HostLabel("mykey"): "mylabel",
				translate.NamespaceLabel:     vObjectMeta.Namespace,
				translate.MarkerLabel:        translate.VClusterName,
			},
			MatchExpressions: []metav1.LabelSelectorRequirement{
				{
					Key:      translate.HostLabel("secondkey"),
					Operator: metav1.LabelSelectorOpIn,
					Values:   []string{"label-A", "label-B"},
				},
			},
		},
	}
	pObjectMeta := metav1.ObjectMeta{
		Name:      translate.Default.HostName(nil, "testnetworkpolicy", "test").Name,
		Namespace: "test",
		Annotations: map[string]string{
			translate.NameAnnotation:          vObjectMeta.Name,
			translate.NamespaceAnnotation:     vObjectMeta.Namespace,
			translate.UIDAnnotation:           "",
			translate.KindAnnotation:          networkingv1.SchemeGroupVersion.WithKind("NetworkPolicy").String(),
			translate.HostNameAnnotation:      translate.Default.HostName(nil, "testnetworkpolicy", "test").Name,
			translate.HostNamespaceAnnotation: "test",
		},
		Labels: map[string]string{
			translate.MarkerLabel:    translate.VClusterName,
			translate.NamespaceLabel: vObjectMeta.Namespace,
		},
	}
	vBaseNetworkPolicy := &networkingv1.NetworkPolicy{
		ObjectMeta: vObjectMeta,
		Spec:       vBaseSpec,
	}
	pBaseNetworkPolicy := &networkingv1.NetworkPolicy{
		ObjectMeta: pObjectMeta,
		Spec:       pBaseSpec,
	}

	vnetworkPolicyNoPodSelector := vBaseNetworkPolicy.DeepCopy()
	vnetworkPolicyNoPodSelector.Spec.PodSelector = metav1.LabelSelector{}

	pnetworkPolicyNoPodSelector := pBaseNetworkPolicy.DeepCopy()
	pnetworkPolicyNoPodSelector.Spec.PodSelector = metav1.LabelSelector{
		MatchLabels: map[string]string{
			translate.NamespaceLabel: vObjectMeta.Namespace,
			translate.MarkerLabel:    translate.VClusterName,
		},
	}

	vnetworkPolicyWithIPBlock := vBaseNetworkPolicy.DeepCopy()
	vnetworkPolicyWithIPBlock.Spec.Ingress = []networkingv1.NetworkPolicyIngressRule{
		{
			Ports: somePorts,
			From: []networkingv1.NetworkPolicyPeer{{IPBlock: &networkingv1.IPBlock{
				CIDR:   "10.0.0.0/24",
				Except: []string{"10.25.0.0/30"},
			}}},
		},
	}
	pnetworkPolicyWithIPBlock := pBaseNetworkPolicy.DeepCopy()
	pnetworkPolicyWithIPBlock.Spec.Ingress = vnetworkPolicyWithIPBlock.Spec.Ingress

	vnetworkPolicyWithPodSelectorNoNs := vBaseNetworkPolicy.DeepCopy()
	vnetworkPolicyWithPodSelectorNoNs.Spec.Ingress = []networkingv1.NetworkPolicyIngressRule{
		{
			Ports: somePorts,
			From: []networkingv1.NetworkPolicyPeer{{PodSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"random-key": "value"},
			}}},
		},
	}
	pnetworkPolicyWithLabelSelectorNoNs := pBaseNetworkPolicy.DeepCopy()
	pnetworkPolicyWithLabelSelectorNoNs.Spec.Ingress = []networkingv1.NetworkPolicyIngressRule{
		{
			Ports: somePorts,
			From: []networkingv1.NetworkPolicyPeer{{PodSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					translate.HostLabel("random-key"): "value",
					translate.MarkerLabel:             translate.VClusterName,
					translate.NamespaceLabel:          vnetworkPolicyWithPodSelectorNoNs.GetNamespace(),
				},
				MatchExpressions: []metav1.LabelSelectorRequirement{},
			}}},
		},
	}

	vnetworkPolicyWithPodSelectorEmptyNs := vnetworkPolicyWithPodSelectorNoNs.DeepCopy()
	vnetworkPolicyWithPodSelectorEmptyNs.Spec.Ingress[0].From[0].NamespaceSelector = &metav1.LabelSelector{}

	pnetworkPolicyWithLabelSelectorEmptyNs := pnetworkPolicyWithLabelSelectorNoNs.DeepCopy()
	delete(pnetworkPolicyWithLabelSelectorEmptyNs.Spec.Ingress[0].From[0].PodSelector.MatchLabels, translate.NamespaceLabel)

	vnetworkPolicyWithPodSelectorNsSelector := vnetworkPolicyWithPodSelectorNoNs.DeepCopy()
	vnetworkPolicyWithPodSelectorNsSelector.Spec.Ingress[0].From[0].NamespaceSelector = &metav1.LabelSelector{
		MatchLabels: map[string]string{"nslabelkey": "abc"},
	}

	pnetworkPolicyWithLabelSelectorNsSelector := pnetworkPolicyWithLabelSelectorNoNs.DeepCopy()
	delete(pnetworkPolicyWithLabelSelectorNsSelector.Spec.Ingress[0].From[0].PodSelector.MatchLabels, translate.NamespaceLabel)
	pnetworkPolicyWithLabelSelectorNsSelector.Spec.Ingress[0].From[0].PodSelector.MatchLabels[translate.HostLabelNamespace("nslabelkey")] = "abc"

	vnetworkPolicyEgressWithPodSelectorNoNs := vBaseNetworkPolicy.DeepCopy()
	vnetworkPolicyEgressWithPodSelectorNoNs.Spec.Egress = []networkingv1.NetworkPolicyEgressRule{
		{
			Ports: somePorts,
			To:    []networkingv1.NetworkPolicyPeer{vnetworkPolicyWithPodSelectorNoNs.Spec.Ingress[0].From[0]},
		},
	}
	pnetworkPolicyEgressWithLabelSelectorNoNs := pBaseNetworkPolicy.DeepCopy()
	pnetworkPolicyEgressWithLabelSelectorNoNs.Spec.Egress = []networkingv1.NetworkPolicyEgressRule{
		{
			Ports: somePorts,
			To:    []networkingv1.NetworkPolicyPeer{pnetworkPolicyWithLabelSelectorNoNs.Spec.Ingress[0].From[0]},
		},
	}

	vnetworkPolicyWithMatchExpressions := vBaseNetworkPolicy.DeepCopy()
	vnetworkPolicyWithMatchExpressions.Spec.Ingress = []networkingv1.NetworkPolicyIngressRule{
		{
			Ports: somePorts,
			From: []networkingv1.NetworkPolicyPeer{{
				PodSelector: &metav1.LabelSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{
							Key:      "pod-expr-key",
							Operator: metav1.LabelSelectorOpExists,
							Values:   []string{"some-pod-key"},
						},
					},
				},
				NamespaceSelector: &metav1.LabelSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{
							Key:      "ns-expr-key",
							Operator: metav1.LabelSelectorOpDoesNotExist,
							Values:   []string{"forbidden-ns-key"},
						},
					},
				},
			}},
		},
	}
	pnetworkPolicyWithMatchExpressions := pBaseNetworkPolicy.DeepCopy()
	pnetworkPolicyWithMatchExpressions.Spec.Ingress = []networkingv1.NetworkPolicyIngressRule{
		{
			Ports: somePorts,
			From: []networkingv1.NetworkPolicyPeer{{
				PodSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						translate.MarkerLabel: translate.VClusterName,
					},
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{
							Key:      translate.HostLabel("pod-expr-key"),
							Operator: metav1.LabelSelectorOpExists,
							Values:   []string{"some-pod-key"},
						},
						{
							Key:      translate.HostLabelNamespace("ns-expr-key"),
							Operator: metav1.LabelSelectorOpDoesNotExist,
							Values:   []string{"forbidden-ns-key"},
						},
					},
				},
			}},
		},
	}

	syncertesting.RunTests(t, []*syncertesting.SyncTest{
		{
			Name:                "Create forward",
			InitialVirtualState: []runtime.Object{vBaseNetworkPolicy.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				networkingv1.SchemeGroupVersion.WithKind("NetworkPolicy"): {vBaseNetworkPolicy.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				networkingv1.SchemeGroupVersion.WithKind("NetworkPolicy"): {pBaseNetworkPolicy.DeepCopy()},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*networkPolicySyncer).SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(vBaseNetworkPolicy.DeepCopy()))
				assert.NilError(t, err)
			},
		},
		{
			Name:                "Create forward - empty pod selector",
			InitialVirtualState: []runtime.Object{vnetworkPolicyNoPodSelector.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				networkingv1.SchemeGroupVersion.WithKind("NetworkPolicy"): {vnetworkPolicyNoPodSelector.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				networkingv1.SchemeGroupVersion.WithKind("NetworkPolicy"): {pnetworkPolicyNoPodSelector.DeepCopy()},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*networkPolicySyncer).SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(vnetworkPolicyNoPodSelector.DeepCopy()))
				assert.NilError(t, err)
			},
		},
		{
			Name: "Update forward",
			InitialVirtualState: []runtime.Object{&networkingv1.NetworkPolicy{
				ObjectMeta: vObjectMeta,
				Spec:       vBaseSpec,
			}},
			InitialPhysicalState: []runtime.Object{&networkingv1.NetworkPolicy{
				ObjectMeta: pObjectMeta,
				Spec:       networkingv1.NetworkPolicySpec{},
			}},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				networkingv1.SchemeGroupVersion.WithKind("NetworkPolicy"): {vBaseNetworkPolicy.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				networkingv1.SchemeGroupVersion.WithKind("NetworkPolicy"): {&networkingv1.NetworkPolicy{
					ObjectMeta: pObjectMeta,
					Spec:       pBaseSpec,
				}},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				pNetworkPolicy := &networkingv1.NetworkPolicy{
					ObjectMeta: pObjectMeta,
					Spec:       networkingv1.NetworkPolicySpec{},
				}
				pNetworkPolicy.ResourceVersion = "999"

				_, err := syncer.(*networkPolicySyncer).Sync(syncCtx, synccontext.NewSyncEvent(pNetworkPolicy, &networkingv1.NetworkPolicy{
					ObjectMeta: vObjectMeta,
					Spec:       vBaseSpec,
				}))
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Update forward not needed",
			InitialVirtualState:  []runtime.Object{vBaseNetworkPolicy.DeepCopy()},
			InitialPhysicalState: []runtime.Object{pBaseNetworkPolicy.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				networkingv1.SchemeGroupVersion.WithKind("NetworkPolicy"): {vBaseNetworkPolicy.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				networkingv1.SchemeGroupVersion.WithKind("NetworkPolicy"): {pBaseNetworkPolicy.DeepCopy()},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				vNetworkPolicy := vBaseNetworkPolicy.DeepCopy()
				vNetworkPolicy.ResourceVersion = "999"

				_, err := syncer.(*networkPolicySyncer).Sync(syncCtx, synccontext.NewSyncEvent(pBaseNetworkPolicy.DeepCopy(), vNetworkPolicy))
				assert.NilError(t, err)
			},
		},
		{
			Name:                "Create forward - ingress policy that uses IPBlock",
			InitialVirtualState: []runtime.Object{vnetworkPolicyWithIPBlock.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				networkingv1.SchemeGroupVersion.WithKind("NetworkPolicy"): {vnetworkPolicyWithIPBlock},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				networkingv1.SchemeGroupVersion.WithKind("NetworkPolicy"): {pnetworkPolicyWithIPBlock},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*networkPolicySyncer).SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(vnetworkPolicyWithIPBlock.DeepCopy()))
				assert.NilError(t, err)
			},
		},
		{
			Name:                "Create forward - ingress policy that uses pod label selector",
			InitialVirtualState: []runtime.Object{vnetworkPolicyWithPodSelectorNoNs.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				networkingv1.SchemeGroupVersion.WithKind("NetworkPolicy"): {vnetworkPolicyWithPodSelectorNoNs},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				networkingv1.SchemeGroupVersion.WithKind("NetworkPolicy"): {pnetworkPolicyWithLabelSelectorNoNs},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*networkPolicySyncer).SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(vnetworkPolicyWithPodSelectorNoNs.DeepCopy()))
				assert.NilError(t, err)
			},
		},
		{
			Name:                "Create forward - ingress policy that uses pod label selector with empty namespace selector",
			InitialVirtualState: []runtime.Object{vnetworkPolicyWithPodSelectorEmptyNs.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				networkingv1.SchemeGroupVersion.WithKind("NetworkPolicy"): {vnetworkPolicyWithPodSelectorEmptyNs},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				networkingv1.SchemeGroupVersion.WithKind("NetworkPolicy"): {pnetworkPolicyWithLabelSelectorEmptyNs},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*networkPolicySyncer).SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(vnetworkPolicyWithPodSelectorEmptyNs.DeepCopy()))
				assert.NilError(t, err)
			},
		},
		{
			Name:                "Create forward - ingress policy that uses pod label selector and namespace selector",
			InitialVirtualState: []runtime.Object{vnetworkPolicyWithPodSelectorNsSelector.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				networkingv1.SchemeGroupVersion.WithKind("NetworkPolicy"): {vnetworkPolicyWithPodSelectorNsSelector},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				networkingv1.SchemeGroupVersion.WithKind("NetworkPolicy"): {pnetworkPolicyWithLabelSelectorNsSelector},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*networkPolicySyncer).SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(vnetworkPolicyWithPodSelectorNsSelector.DeepCopy()))
				assert.NilError(t, err)
			},
		},
		{
			Name:                "Create forward - ingress policy that uses pod label selector and namespace selector which use MatchExpressions",
			InitialVirtualState: []runtime.Object{vnetworkPolicyWithMatchExpressions.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				networkingv1.SchemeGroupVersion.WithKind("NetworkPolicy"): {vnetworkPolicyWithMatchExpressions},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				networkingv1.SchemeGroupVersion.WithKind("NetworkPolicy"): {pnetworkPolicyWithMatchExpressions},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*networkPolicySyncer).SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(vnetworkPolicyWithMatchExpressions.DeepCopy()))
				assert.NilError(t, err)
			},
		},
		{
			Name:                "Create forward - egress policy that uses pod label selector",
			InitialVirtualState: []runtime.Object{vnetworkPolicyEgressWithPodSelectorNoNs.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				networkingv1.SchemeGroupVersion.WithKind("NetworkPolicy"): {vnetworkPolicyEgressWithPodSelectorNoNs},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				networkingv1.SchemeGroupVersion.WithKind("NetworkPolicy"): {pnetworkPolicyEgressWithLabelSelectorNoNs},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*networkPolicySyncer).SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(vnetworkPolicyEgressWithPodSelectorNoNs.DeepCopy()))
				assert.NilError(t, err)
			},
		},
	})
}
