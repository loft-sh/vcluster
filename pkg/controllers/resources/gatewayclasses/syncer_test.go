package gatewayclasses

import (
	"maps"
	"testing"

	rootconfig "github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertesting "github.com/loft-sh/vcluster/pkg/syncer/testing"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"gotest.tools/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func TestSync(t *testing.T) {
	selector := rootconfig.StandardLabelSelector{MatchLabels: map[string]string{"sync": "yes"}}
	adjustSelector := func(vConfig *config.VirtualClusterConfig) {
		vConfig.Sync.FromHost.GatewayClasses.Selector = selector
	}

	gwClassGVK := schema.GroupVersionKind{
		Group:   gatewayv1.GroupVersion.Group,
		Version: gatewayv1.GroupVersion.Version,
		Kind:    "GatewayClass",
	}

	vObjectMeta := metav1.ObjectMeta{
		Name: "test-gwc",
		Annotations: map[string]string{
			translate.NameAnnotation: "test-gwc",
			translate.UIDAnnotation:  "",
			translate.KindAnnotation: gwClassGVK.String(),
		},
		ResourceVersion: "999",
	}

	vObj := &gatewayv1.GatewayClass{
		ObjectMeta: vObjectMeta,
		Spec: gatewayv1.GatewayClassSpec{
			ControllerName: gatewayv1.GatewayController("example.com/gateway-controller"),
		},
	}

	pObj := &gatewayv1.GatewayClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: vObjectMeta.Name,
			Annotations: map[string]string{
				translate.NameAnnotation: "test-gwc",
				translate.UIDAnnotation:  "",
				translate.KindAnnotation: gwClassGVK.String(),
			},
		},
		Spec: gatewayv1.GatewayClassSpec{
			ControllerName: gatewayv1.GatewayController("example.com/gateway-controller"),
		},
	}

	updatedLabels := map[string]string{"app": "my-gateway"}
	vObjectMetaUpdated := vObjectMeta
	vObjectMetaUpdated.Labels = maps.Clone(updatedLabels)

	description := "GatewayClass for tests"
	parametersNamespace := gatewayv1.Namespace("gateway-system")
	updatedSpec := gatewayv1.GatewayClassSpec{
		ControllerName: gatewayv1.GatewayController("example.com/gateway-controller"),
		Description:    &description,
		ParametersRef: &gatewayv1.ParametersReference{
			Group:     gatewayv1.Group("example.com"),
			Kind:      gatewayv1.Kind("GatewayClassConfig"),
			Name:      "test-gwc-param",
			Namespace: &parametersNamespace,
		},
	}
	updatedStatus := gatewayv1.GatewayClassStatus{
		Conditions: []metav1.Condition{
			{
				Type:   string(gatewayv1.GatewayClassConditionStatusAccepted),
				Status: metav1.ConditionTrue,
				Reason: string(gatewayv1.GatewayClassReasonAccepted),
			},
		},
		SupportedFeatures: []gatewayv1.SupportedFeature{
			{Name: gatewayv1.FeatureName("HTTPRoute")},
		},
	}

	vObjWithStatus := vObj.DeepCopy()
	vObjWithStatus.Status = updatedStatus

	pObjWithStatus := pObj.DeepCopy()
	pObjWithStatus.Status = updatedStatus

	vObjUpdated := &gatewayv1.GatewayClass{
		ObjectMeta: vObjectMetaUpdated,
		Spec:       updatedSpec,
		Status:     updatedStatus,
	}

	pObjUpdated := &gatewayv1.GatewayClass{
		ObjectMeta: metav1.ObjectMeta{
			Name:   vObjectMeta.Name,
			Labels: updatedLabels,
			Annotations: map[string]string{
				translate.NameAnnotation: "test-gwc",
				translate.UIDAnnotation:  "",
				translate.KindAnnotation: gwClassGVK.String(),
			},
		},
		Spec:   updatedSpec,
		Status: updatedStatus,
	}

	syncertesting.RunTests(t, []*syncertesting.SyncTest{
		{
			Name:                 "Sync Up",
			InitialVirtualState:  []runtime.Object{},
			InitialPhysicalState: []runtime.Object{pObj},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				gwClassGVK: {vObj},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				gwClassGVK: {pObj},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*gatewayClassSyncer).SyncToVirtual(syncCtx, synccontext.NewSyncToVirtualEvent(pObj.DeepCopy()))
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Sync Up with status",
			InitialVirtualState:  []runtime.Object{vObjWithStatus},
			InitialPhysicalState: []runtime.Object{pObjWithStatus},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				gwClassGVK: {vObjWithStatus},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				gwClassGVK: {pObjWithStatus},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)

				// InitialVirtualState must include vObjWithStatus to register GatewayClass as a
				// status-subresource type in the fake client (via WithStatusSubresource). This
				// enforces that Create() strips the status field, so only the Status().Patch()
				// path in patcher.CreateVirtualObject(hasStatus=true) can write it.
				// We then delete the seeded object so SyncToVirtual can create it from scratch.
				err := syncCtx.VirtualClient.Delete(syncCtx, vObjWithStatus.DeepCopy())
				assert.NilError(t, err)

				_, err = syncer.(*gatewayClassSyncer).SyncToVirtual(syncCtx, synccontext.NewSyncToVirtualEvent(pObjWithStatus.DeepCopy()))
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Sync Up selector mismatch",
			InitialVirtualState:  []runtime.Object{},
			InitialPhysicalState: []runtime.Object{pObj},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				gwClassGVK: {},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				gwClassGVK: {pObj},
			},
			AdjustConfig: adjustSelector,
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*gatewayClassSyncer).SyncToVirtual(syncCtx, synccontext.NewSyncToVirtualEvent(pObj.DeepCopy()))
				assert.NilError(t, err)
			},
		},
		{
			Name:                  "Sync Down",
			InitialVirtualState:   []runtime.Object{vObj},
			ExpectedVirtualState:  map[schema.GroupVersionKind][]runtime.Object{},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*gatewayClassSyncer).SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(vObj.DeepCopy()))
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Sync selector mismatch",
			InitialVirtualState:  []runtime.Object{vObj},
			InitialPhysicalState: []runtime.Object{pObj},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				gwClassGVK: {},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				gwClassGVK: {pObj},
			},
			AdjustConfig: adjustSelector,
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*gatewayClassSyncer).Sync(syncCtx, synccontext.NewSyncEvent(pObj.DeepCopy(), vObj.DeepCopy()))
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Sync",
			InitialVirtualState:  []runtime.Object{vObj},
			InitialPhysicalState: []runtime.Object{pObjUpdated},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				gwClassGVK: {vObjUpdated},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				gwClassGVK: {pObjUpdated},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*gatewayClassSyncer).Sync(syncCtx, synccontext.NewSyncEvent(pObjUpdated.DeepCopy(), vObj.DeepCopy()))
				assert.NilError(t, err)
			},
		},
	})
}
