package ingressclasses

import (
	"testing"

	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertesting "github.com/loft-sh/vcluster/pkg/syncer/testing"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"gotest.tools/assert"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestSync(t *testing.T) {
	vObjectMeta := metav1.ObjectMeta{
		Name: "test-ingc",
		Annotations: map[string]string{
			translate.NameAnnotation: "test-ingc",
			translate.UIDAnnotation:  "",
			translate.KindAnnotation: networkingv1.SchemeGroupVersion.WithKind("IngressClass").String(),
		},
		Labels: map[string]string{
			translate.MarkerLabel: translate.VClusterName,
		},
		ResourceVersion: "999",
	}

	vObj := &networkingv1.IngressClass{
		ObjectMeta: vObjectMeta,
		Spec: networkingv1.IngressClassSpec{
			Controller: "test-controller",
		},
	}

	pObj := &networkingv1.IngressClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: vObjectMeta.Name,
			Labels: map[string]string{
				translate.MarkerLabel: translate.VClusterName,
			},
			Annotations: map[string]string{
				translate.NameAnnotation: "test-ingc",
				translate.UIDAnnotation:  "",
				translate.KindAnnotation: networkingv1.SchemeGroupVersion.WithKind("IngressClass").String(),
			},
		},
		Spec: networkingv1.IngressClassSpec{
			Controller: "test-controller",
		},
	}

	vObjUpdated := &networkingv1.IngressClass{
		ObjectMeta: vObjectMeta,
		Spec: networkingv1.IngressClassSpec{
			Controller: "test-controller",
			Parameters: &networkingv1.IngressClassParametersReference{
				APIGroup: strRef("test-group"),
				Kind:     "test-kind",
				Name:     "test-ingc-param",
			},
		},
	}

	pObjUpdated := &networkingv1.IngressClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: translate.Default.HostNameCluster(vObjectMeta.Name),
			Labels: map[string]string{
				translate.MarkerLabel: translate.VClusterName,
			},
			Annotations: map[string]string{
				translate.NameAnnotation: "test-ingc",
				translate.UIDAnnotation:  "",
				translate.KindAnnotation: networkingv1.SchemeGroupVersion.WithKind("IngressClass").String(),
			},
		},
		Spec: networkingv1.IngressClassSpec{
			Controller: "test-controller",
			Parameters: &networkingv1.IngressClassParametersReference{
				APIGroup: strRef("test-group"),
				Kind:     "test-kind",
				Name:     "test-ingc-param",
			},
		},
	}

	syncertesting.RunTests(t, []*syncertesting.SyncTest{
		{
			Name:                 "Sync Up",
			InitialVirtualState:  []runtime.Object{},
			InitialPhysicalState: []runtime.Object{pObj},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				networkingv1.SchemeGroupVersion.WithKind("IngressClass"): {vObj},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				networkingv1.SchemeGroupVersion.WithKind("IngressClass"): {pObj},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*ingressClassSyncer).SyncToVirtual(syncCtx, synccontext.NewSyncToVirtualEvent(pObj))
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
				_, err := syncer.(*ingressClassSyncer).SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(vObj))
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Sync",
			InitialVirtualState:  []runtime.Object{vObj},
			InitialPhysicalState: []runtime.Object{pObjUpdated},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				networkingv1.SchemeGroupVersion.WithKind("IngressClass"): {vObjUpdated},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				networkingv1.SchemeGroupVersion.WithKind("IngressClass"): {pObjUpdated},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*ingressClassSyncer).Sync(syncCtx, synccontext.NewSyncEvent(pObjUpdated, vObj))
				assert.NilError(t, err)
			},
		},
	})
}

func strRef(str string) *string {
	return &str
}
