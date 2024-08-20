package poddisruptionbudgets

import (
	"testing"

	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertesting "github.com/loft-sh/vcluster/pkg/syncer/testing"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	"gotest.tools/assert"

	"github.com/loft-sh/vcluster/pkg/util/translate"

	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestSync(t *testing.T) {
	translate.Default = translate.NewSingleNamespaceTranslator(testingutil.DefaultTestTargetNamespace)
	vObjectMeta := metav1.ObjectMeta{
		Name:            "testPDB",
		Namespace:       "default",
		ResourceVersion: syncertesting.FakeClientResourceVersion,
	}
	pObjectMeta := metav1.ObjectMeta{
		Name:      translate.Default.HostName(nil, "testPDB", vObjectMeta.Namespace).Name,
		Namespace: "test",
		Annotations: map[string]string{
			translate.NameAnnotation:          vObjectMeta.Name,
			translate.NamespaceAnnotation:     vObjectMeta.Namespace,
			translate.UIDAnnotation:           "",
			translate.KindAnnotation:          policyv1.SchemeGroupVersion.WithKind("PodDisruptionBudget").String(),
			translate.HostNameAnnotation:      translate.Default.HostName(nil, "testPDB", vObjectMeta.Namespace).Name,
			translate.HostNamespaceAnnotation: "test",
		},
		Labels: map[string]string{
			translate.NamespaceLabel: vObjectMeta.Namespace,
			translate.MarkerLabel:    translate.VClusterName,
		},
		ResourceVersion: syncertesting.FakeClientResourceVersion,
	}

	vclusterPDB := &policyv1.PodDisruptionBudget{
		ObjectMeta: vObjectMeta,
		Spec: policyv1.PodDisruptionBudgetSpec{
			MinAvailable: &intstr.IntOrString{Type: intstr.Int, IntVal: int32(10)},
		},
	}

	hostClusterSyncedPDB := &policyv1.PodDisruptionBudget{
		ObjectMeta: pObjectMeta,
		Spec:       vclusterPDB.Spec,
	}

	vclusterUpdatedPDB := &policyv1.PodDisruptionBudget{
		ObjectMeta: vclusterPDB.ObjectMeta,
		Spec: policyv1.PodDisruptionBudgetSpec{
			MaxUnavailable: &intstr.IntOrString{Type: intstr.Int, IntVal: int32(5)},
		},
	}

	hostClusterSyncedUpdatedPDB := &policyv1.PodDisruptionBudget{
		ObjectMeta: hostClusterSyncedPDB.ObjectMeta,
		Spec:       vclusterUpdatedPDB.Spec,
	}

	vclusterUpdatedSelectorPDB := &policyv1.PodDisruptionBudget{
		ObjectMeta: vclusterPDB.ObjectMeta,
		Spec: policyv1.PodDisruptionBudgetSpec{
			MaxUnavailable: &intstr.IntOrString{Type: intstr.Int, IntVal: int32(5)},
			Selector:       &metav1.LabelSelector{MatchLabels: map[string]string{"app": "nginx"}},
		},
	}

	hostClusterSyncedUpdatedSelectorPDB := &policyv1.PodDisruptionBudget{
		ObjectMeta: hostClusterSyncedPDB.ObjectMeta,
		Spec: policyv1.PodDisruptionBudgetSpec{
			MaxUnavailable: vclusterUpdatedSelectorPDB.Spec.MaxUnavailable,
			Selector:       translate.HostLabelSelector(vclusterUpdatedSelectorPDB.Spec.Selector),
		},
	}

	syncertesting.RunTests(t, []*syncertesting.SyncTest{
		{
			Name: "Create Host Cluster PodDisruptionBudget",
			InitialVirtualState: []runtime.Object{
				vclusterPDB.DeepCopy(),
			},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				policyv1.SchemeGroupVersion.WithKind("PodDisruptionBudget"): {vclusterPDB.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				policyv1.SchemeGroupVersion.WithKind("PodDisruptionBudget"): {hostClusterSyncedPDB.DeepCopy()},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*pdbSyncer).SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(vclusterPDB.DeepCopy()))
				assert.NilError(t, err)
			},
		},
		{
			Name: "Update Host Cluster PodDisruptionBudget's Spec",
			InitialVirtualState: []runtime.Object{
				vclusterUpdatedPDB.DeepCopy(),
			},
			InitialPhysicalState: []runtime.Object{
				hostClusterSyncedPDB.DeepCopy(),
			},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				policyv1.SchemeGroupVersion.WithKind("PodDisruptionBudget"): {vclusterUpdatedPDB.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				policyv1.SchemeGroupVersion.WithKind("PodDisruptionBudget"): {hostClusterSyncedUpdatedPDB.DeepCopy()},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*pdbSyncer).Sync(syncCtx, synccontext.NewSyncEvent(hostClusterSyncedPDB.DeepCopy(), vclusterUpdatedPDB.DeepCopy()))
				assert.NilError(t, err)
			},
		},
		{
			Name: "Update Host Cluster PodDisruptionBudget's Selector",
			InitialVirtualState: []runtime.Object{
				vclusterUpdatedSelectorPDB.DeepCopy(),
			},
			InitialPhysicalState: []runtime.Object{
				hostClusterSyncedPDB.DeepCopy(),
			},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				policyv1.SchemeGroupVersion.WithKind("PodDisruptionBudget"): {vclusterUpdatedSelectorPDB.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				policyv1.SchemeGroupVersion.WithKind("PodDisruptionBudget"): {hostClusterSyncedUpdatedSelectorPDB.DeepCopy()},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*pdbSyncer).Sync(syncCtx, synccontext.NewSyncEvent(hostClusterSyncedPDB.DeepCopy(), vclusterUpdatedSelectorPDB.DeepCopy()))
				assert.NilError(t, err)
			},
		},
	})
}
