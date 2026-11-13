package poddisruptionbudgets

import (
	"context"
	"testing"

	"github.com/loft-sh/vcluster/pkg/scheme"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertesting "github.com/loft-sh/vcluster/pkg/syncer/testing"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"gotest.tools/assert"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

func intVal(v int32) *intstr.IntOrString  { return ptr.To(intstr.FromInt32(v)) }
func strVal(s string) *intstr.IntOrString { return ptr.To(intstr.FromString(s)) }

func testDeployment(name string, replicas int32, labels map[string]string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       "default",
			UID:             types.UID(name + "-uid"),
			ResourceVersion: syncertesting.FakeClientResourceVersion,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptr.To(replicas),
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
			},
		},
	}
}

func testStatefulSet(name string, replicas int32, labels map[string]string) *appsv1.StatefulSet {
	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       "default",
			UID:             types.UID(name + "-uid"),
			ResourceVersion: syncertesting.FakeClientResourceVersion,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: ptr.To(replicas),
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
			},
		},
	}
}

func testReplicaSet(name, deployName string, replicas int32, labels map[string]string) *appsv1.ReplicaSet {
	return &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       "default",
			UID:             types.UID(name + "-uid"),
			OwnerReferences: []metav1.OwnerReference{{APIVersion: "apps/v1", Kind: "Deployment", Name: deployName, UID: types.UID(deployName + "-uid"), Controller: ptr.To(true)}},
			ResourceVersion: syncertesting.FakeClientResourceVersion,
		},
		Spec: appsv1.ReplicaSetSpec{
			Replicas: ptr.To(replicas),
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{Labels: labels}},
		},
	}
}

func testPod(name, rsName string, labels map[string]string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       "default",
			Labels:          labels,
			OwnerReferences: []metav1.OwnerReference{{APIVersion: "apps/v1", Kind: "ReplicaSet", Name: rsName, UID: types.UID(rsName + "-uid"), Controller: ptr.To(true)}},
			ResourceVersion: syncertesting.FakeClientResourceVersion,
		},
	}
}

func testHostSelector(sel *metav1.LabelSelector) *metav1.LabelSelector {
	return translate.HostLabelSelector(sel)
}

func testVirtualPDBWithSelector(name string, minAvailable, maxUnavailable *intstr.IntOrString, selector *metav1.LabelSelector) *policyv1.PodDisruptionBudget {
	return &policyv1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       "default",
			ResourceVersion: syncertesting.FakeClientResourceVersion,
		},
		Spec: policyv1.PodDisruptionBudgetSpec{
			MinAvailable:   minAvailable,
			MaxUnavailable: maxUnavailable,
			Selector:       selector,
		},
	}
}

func testVirtualPDB(name string, minAvailable, maxUnavailable *intstr.IntOrString, selector map[string]string) *policyv1.PodDisruptionBudget {
	var labelSelector *metav1.LabelSelector
	if selector != nil {
		labelSelector = &metav1.LabelSelector{MatchLabels: selector}
	}
	return testVirtualPDBWithSelector(name, minAvailable, maxUnavailable, labelSelector)
}

func testHostPDB(vPDB *policyv1.PodDisruptionBudget, minAvailable, maxUnavailable *intstr.IntOrString) *policyv1.PodDisruptionBudget {
	pMeta := metav1.ObjectMeta{
		Name:      translate.Default.HostName(nil, vPDB.Name, vPDB.Namespace).Name,
		Namespace: "test",
		Annotations: map[string]string{
			translate.NameAnnotation:          vPDB.Name,
			translate.NamespaceAnnotation:     vPDB.Namespace,
			translate.UIDAnnotation:           "",
			translate.KindAnnotation:          policyv1.SchemeGroupVersion.WithKind("PodDisruptionBudget").String(),
			translate.HostNameAnnotation:      translate.Default.HostName(nil, vPDB.Name, vPDB.Namespace).Name,
			translate.HostNamespaceAnnotation: "test",
		},
		Labels: map[string]string{
			translate.NamespaceLabel: vPDB.Namespace,
			translate.MarkerLabel:    translate.VClusterName,
		},
		ResourceVersion: syncertesting.FakeClientResourceVersion,
	}
	pdb := &policyv1.PodDisruptionBudget{
		ObjectMeta: pMeta,
		Spec: policyv1.PodDisruptionBudgetSpec{
			MinAvailable:   minAvailable,
			MaxUnavailable: maxUnavailable,
		},
	}
	if vPDB.Spec.Selector != nil {
		pdb.Spec.Selector = testHostSelector(vPDB.Spec.Selector)
	}
	return pdb
}

func pdbState(pdbs ...runtime.Object) map[schema.GroupVersionKind][]runtime.Object {
	return map[schema.GroupVersionKind][]runtime.Object{
		policyv1.SchemeGroupVersion.WithKind("PodDisruptionBudget"): pdbs,
	}
}

func TestSync(t *testing.T) {
	translate.Default = translate.NewSingleNamespaceTranslator(testingutil.DefaultTestTargetNamespace)
	nginxLabels := map[string]string{"app": "nginx"}
	dbLabels := map[string]string{"app": "db"}

	// Running pod whose controller owner is a Deployment that is not created.
	missingControllerPod := func(name string) *corev1.Pod {
		return &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: "default",
				Labels:    nginxLabels,
				OwnerReferences: []metav1.OwnerReference{{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Name:       "deploy-b",
					UID:        types.UID("deploy-b-uid"),
					Controller: ptr.To(true),
				}},
				ResourceVersion: syncertesting.FakeClientResourceVersion,
			},
			Status: corev1.PodStatus{Phase: corev1.PodRunning},
		}
	}

	// PDB whose status looks fresh but disagrees with spec.replicas.
	freshDesiredHealthyPDB := func() *policyv1.PodDisruptionBudget {
		pdb := testVirtualPDB("testPDB", nil, intVal(1), nginxLabels)
		pdb.Generation = 1
		pdb.Status = policyv1.PodDisruptionBudgetStatus{
			ObservedGeneration: 1,
			DesiredHealthy:     2,
			ExpectedPods:       3,
			Conditions: []metav1.Condition{{
				Type:   policyv1.DisruptionAllowedCondition,
				Status: metav1.ConditionTrue,
				Reason: policyv1.SufficientPodsReason,
			}},
		}
		return pdb
	}

	// PDB with only a previous expectedPods, and nothing for the walk to count.
	expectedPodsOnlyPDB := func() *policyv1.PodDisruptionBudget {
		pdb := testVirtualPDB("testPDB", nil, intVal(1), nginxLabels)
		pdb.Generation = 1
		pdb.Status = policyv1.PodDisruptionBudgetStatus{
			ObservedGeneration: 1,
			ExpectedPods:       4,
		}
		return pdb
	}

	vclusterPDB := testVirtualPDB("testPDB", intVal(10), nil, nil)
	hostClusterSyncedPDB := testHostPDB(vclusterPDB, intVal(10), nil)

	vclusterUpdatedPDB := testVirtualPDB("testPDB", nil, intVal(5), nil)
	hostClusterSyncedUpdatedPDB := testHostPDB(vclusterUpdatedPDB, intVal(0), nil)

	vclusterUpdatedSelectorPDB := testVirtualPDB("testPDB", nil, intVal(5), nginxLabels)
	hostClusterSyncedUpdatedSelectorPDB := testHostPDB(vclusterUpdatedSelectorPDB, intVal(0), nil)

	syncertesting.RunTests(t, []*syncertesting.SyncTest{
		{
			Name:                  "Create Host Cluster PodDisruptionBudget",
			InitialVirtualState:   []runtime.Object{vclusterPDB.DeepCopy()},
			ExpectedVirtualState:  pdbState(vclusterPDB.DeepCopy()),
			ExpectedPhysicalState: pdbState(hostClusterSyncedPDB.DeepCopy()),
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*pdbSyncer).SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(vclusterPDB.DeepCopy()))
				assert.NilError(t, err)
			},
		},
		{
			Name:                  "Update Host Cluster PodDisruptionBudget's Spec",
			InitialVirtualState:   []runtime.Object{vclusterUpdatedPDB.DeepCopy()},
			InitialPhysicalState:  []runtime.Object{hostClusterSyncedPDB.DeepCopy()},
			ExpectedVirtualState:  pdbState(vclusterUpdatedPDB.DeepCopy()),
			ExpectedPhysicalState: pdbState(hostClusterSyncedUpdatedPDB.DeepCopy()),
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*pdbSyncer).Sync(syncCtx, synccontext.NewSyncEventWithOld(
					hostClusterSyncedPDB.DeepCopy(),
					hostClusterSyncedPDB.DeepCopy(),
					vclusterUpdatedPDB.DeepCopy(),
					vclusterUpdatedPDB.DeepCopy(),
				))
				assert.NilError(t, err)
			},
		},
		{
			Name:                  "Update Host Cluster PodDisruptionBudget's ByName",
			InitialVirtualState:   []runtime.Object{vclusterUpdatedSelectorPDB.DeepCopy()},
			InitialPhysicalState:  []runtime.Object{hostClusterSyncedPDB.DeepCopy()},
			ExpectedVirtualState:  pdbState(vclusterUpdatedSelectorPDB.DeepCopy()),
			ExpectedPhysicalState: pdbState(hostClusterSyncedUpdatedSelectorPDB.DeepCopy()),
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*pdbSyncer).Sync(syncCtx, synccontext.NewSyncEventWithOld(
					hostClusterSyncedPDB.DeepCopy(),
					hostClusterSyncedPDB.DeepCopy(),
					vclusterUpdatedSelectorPDB.DeepCopy(),
					vclusterUpdatedSelectorPDB.DeepCopy(),
				))
				assert.NilError(t, err)
			},
		},
		{
			Name:                  "Integer minAvailable Passthrough on Update",
			InitialVirtualState:   []runtime.Object{testVirtualPDB("testPDB", intVal(3), nil, nil)},
			InitialPhysicalState:  []runtime.Object{hostClusterSyncedPDB.DeepCopy()},
			ExpectedVirtualState:  pdbState(testVirtualPDB("testPDB", intVal(3), nil, nil)),
			ExpectedPhysicalState: pdbState(testHostPDB(vclusterPDB, intVal(3), nil)),
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				vUpdated := testVirtualPDB("testPDB", intVal(3), nil, nil)
				_, err := syncer.(*pdbSyncer).Sync(syncCtx, synccontext.NewSyncEventWithOld(
					hostClusterSyncedPDB.DeepCopy(),
					hostClusterSyncedPDB.DeepCopy(),
					vUpdated,
					vUpdated,
				))
				assert.NilError(t, err)
			},
		},
		{
			Name: "Integer maxUnavailable Converted with Virtual Deployment",
			InitialVirtualState: []runtime.Object{
				testDeployment("nginx-deploy", 3, nginxLabels),
				testVirtualPDB("testPDB", nil, intVal(1), nginxLabels),
			},
			ExpectedVirtualState:  pdbState(testVirtualPDB("testPDB", nil, intVal(1), nginxLabels)),
			ExpectedPhysicalState: pdbState(testHostPDB(testVirtualPDB("testPDB", nil, intVal(1), nginxLabels), intVal(2), nil)),
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				vPDB := testVirtualPDB("testPDB", nil, intVal(1), nginxLabels)
				_, err := syncer.(*pdbSyncer).SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(vPDB))
				assert.NilError(t, err)
			},
		},
		{
			Name: "Percentage maxUnavailable with Rounding Check",
			InitialVirtualState: []runtime.Object{
				testDeployment("nginx-deploy", 3, nginxLabels),
				testVirtualPDB("testPDB", nil, strVal("50%"), nginxLabels),
			},
			ExpectedVirtualState: pdbState(testVirtualPDB("testPDB", nil, strVal("50%"), nginxLabels)),
			ExpectedPhysicalState: pdbState(
				// 50% of 3 rounded up = 2; desiredHealthy = 3 - 2 = 1
				testHostPDB(testVirtualPDB("testPDB", nil, strVal("50%"), nginxLabels), intVal(1), nil),
			),
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				vPDB := testVirtualPDB("testPDB", nil, strVal("50%"), nginxLabels)
				_, err := syncer.(*pdbSyncer).SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(vPDB))
				assert.NilError(t, err)
			},
		},
		{
			Name: "Percentage minAvailable with Rounding Check",
			InitialVirtualState: []runtime.Object{
				testStatefulSet("db-sts", 3, dbLabels),
				testVirtualPDB("testPDB", strVal("50%"), nil, dbLabels),
			},
			ExpectedVirtualState: pdbState(testVirtualPDB("testPDB", strVal("50%"), nil, dbLabels)),
			ExpectedPhysicalState: pdbState(
				// 50% of 3 rounded up = 2; desiredHealthy = 2
				testHostPDB(testVirtualPDB("testPDB", strVal("50%"), nil, dbLabels), intVal(2), nil),
			),
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				vPDB := testVirtualPDB("testPDB", strVal("50%"), nil, dbLabels)
				_, err := syncer.(*pdbSyncer).SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(vPDB))
				assert.NilError(t, err)
			},
		},
		{
			Name: "Workload Scale-Up Recomputes Host minAvailable",
			InitialVirtualState: []runtime.Object{
				testDeployment("nginx-deploy", 5, nginxLabels),
				testVirtualPDB("testPDB", nil, intVal(1), nginxLabels),
			},
			InitialPhysicalState: []runtime.Object{
				testHostPDB(testVirtualPDB("testPDB", nil, intVal(1), nginxLabels), intVal(2), nil),
			},
			ExpectedVirtualState: pdbState(testVirtualPDB("testPDB", nil, intVal(1), nginxLabels)),
			ExpectedPhysicalState: pdbState(
				// 5 replicas - 1 maxUnavailable = 4 minAvailable
				testHostPDB(testVirtualPDB("testPDB", nil, intVal(1), nginxLabels), intVal(4), nil),
			),
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				vPDB := testVirtualPDB("testPDB", nil, intVal(1), nginxLabels)
				pPDB := testHostPDB(vPDB, intVal(2), nil)
				_, err := syncer.(*pdbSyncer).Sync(syncCtx, synccontext.NewSyncEventWithOld(
					pPDB.DeepCopy(),
					pPDB.DeepCopy(),
					vPDB.DeepCopy(),
					vPDB.DeepCopy(),
				))
				assert.NilError(t, err)
			},
		},
		{
			Name: "Workload Scale-Down to 1 Replica Clamps to minAvailable 0",
			InitialVirtualState: []runtime.Object{
				testDeployment("nginx-deploy", 1, nginxLabels),
				testVirtualPDB("testPDB", nil, intVal(1), nginxLabels),
			},
			InitialPhysicalState: []runtime.Object{
				testHostPDB(testVirtualPDB("testPDB", nil, intVal(1), nginxLabels), intVal(4), nil),
			},
			ExpectedVirtualState: pdbState(testVirtualPDB("testPDB", nil, intVal(1), nginxLabels)),
			ExpectedPhysicalState: pdbState(
				// 1 replica - 1 maxUnavailable = 0 minAvailable
				testHostPDB(testVirtualPDB("testPDB", nil, intVal(1), nginxLabels), intVal(0), nil),
			),
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				vPDB := testVirtualPDB("testPDB", nil, intVal(1), nginxLabels)
				pPDB := testHostPDB(vPDB, intVal(4), nil)
				_, err := syncer.(*pdbSyncer).Sync(syncCtx, synccontext.NewSyncEventWithOld(
					pPDB.DeepCopy(),
					pPDB.DeepCopy(),
					vPDB.DeepCopy(),
					vPDB.DeepCopy(),
				))
				assert.NilError(t, err)
			},
		},
		{
			Name: "Rolling Update ReplicaSet Deduplication via Deployment",
			InitialVirtualState: []runtime.Object{
				testDeployment("nginx-deploy", 3, nginxLabels),
				testReplicaSet("nginx-rs-1", "nginx-deploy", 1, nginxLabels),
				testReplicaSet("nginx-rs-2", "nginx-deploy", 3, nginxLabels),
				testPod("pod-old", "nginx-rs-1", nginxLabels),
				testPod("pod-new-1", "nginx-rs-2", nginxLabels),
				testPod("pod-new-2", "nginx-rs-2", nginxLabels),
				testVirtualPDB("testPDB", nil, intVal(1), nginxLabels),
			},
			ExpectedVirtualState: pdbState(testVirtualPDB("testPDB", nil, intVal(1), nginxLabels)),
			ExpectedPhysicalState: pdbState(
				// ExpectedCount resolved from Deployment = 3 (not 1+3=4 or pods count); 3 - 1 = 2
				testHostPDB(testVirtualPDB("testPDB", nil, intVal(1), nginxLabels), intVal(2), nil),
			),
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				vPDB := testVirtualPDB("testPDB", nil, intVal(1), nginxLabels)
				_, err := syncer.(*pdbSyncer).SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(vPDB))
				assert.NilError(t, err)
			},
		},
		{
			Name: "Clamping When maxUnavailable Exceeds expectedCount",
			InitialVirtualState: []runtime.Object{
				testDeployment("nginx-deploy", 2, nginxLabels),
				testVirtualPDB("testPDB", nil, intVal(5), nginxLabels),
			},
			ExpectedVirtualState: pdbState(testVirtualPDB("testPDB", nil, intVal(5), nginxLabels)),
			ExpectedPhysicalState: pdbState(
				// 2 replicas - 5 maxUnavailable = -3 clamped to 0
				testHostPDB(testVirtualPDB("testPDB", nil, intVal(5), nginxLabels), intVal(0), nil),
			),
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				vPDB := testVirtualPDB("testPDB", nil, intVal(5), nginxLabels)
				_, err := syncer.(*pdbSyncer).SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(vPDB))
				assert.NilError(t, err)
			},
		},
		{
			Name: "Nil Selector with maxUnavailable Ignores Workloads in Namespace",
			InitialVirtualState: []runtime.Object{
				testDeployment("nginx-deploy", 3, nginxLabels),
				testVirtualPDBWithSelector("testPDB", nil, intVal(1), nil),
			},
			ExpectedVirtualState: pdbState(testVirtualPDBWithSelector("testPDB", nil, intVal(1), nil)),
			ExpectedPhysicalState: pdbState(
				testHostPDB(testVirtualPDBWithSelector("testPDB", nil, intVal(1), nil), intVal(0), nil),
			),
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				vPDB := testVirtualPDBWithSelector("testPDB", nil, intVal(1), nil)
				_, err := syncer.(*pdbSyncer).SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(vPDB))
				assert.NilError(t, err)
			},
		},
		{
			Name: "Empty Selector with maxUnavailable Across Multiple Workloads and Namespace Isolation",
			InitialVirtualState: []runtime.Object{
				testDeployment("nginx-deploy", 3, nginxLabels),
				testStatefulSet("db-sts", 2, dbLabels),
				func() *appsv1.Deployment {
					d := testDeployment("other-deploy", 4, map[string]string{"app": "other"})
					d.Namespace = "other"
					return d
				}(),
				testVirtualPDBWithSelector("testPDB", nil, intVal(1), &metav1.LabelSelector{}),
			},
			ExpectedVirtualState: pdbState(testVirtualPDBWithSelector("testPDB", nil, intVal(1), &metav1.LabelSelector{})),
			ExpectedPhysicalState: pdbState(
				// (3 nginx + 2 db) - 1 maxUnavailable = 4 minAvailable; 'other' deployment ignored
				testHostPDB(testVirtualPDBWithSelector("testPDB", nil, intVal(1), &metav1.LabelSelector{}), intVal(4), nil),
			),
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				vPDB := testVirtualPDBWithSelector("testPDB", nil, intVal(1), &metav1.LabelSelector{})
				_, err := syncer.(*pdbSyncer).SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(vPDB))
				assert.NilError(t, err)
			},
		},
		{
			Name: "Empty Selector with Percentage minAvailable Across Workloads",
			InitialVirtualState: []runtime.Object{
				testDeployment("nginx-deploy", 3, nginxLabels),
				testStatefulSet("db-sts", 2, dbLabels),
				testVirtualPDBWithSelector("testPDB", strVal("50%"), nil, &metav1.LabelSelector{}),
			},
			ExpectedVirtualState: pdbState(testVirtualPDBWithSelector("testPDB", strVal("50%"), nil, &metav1.LabelSelector{})),
			ExpectedPhysicalState: pdbState(
				// 50% of 5 expected pods = ceil(2.5) = 3 minAvailable
				testHostPDB(testVirtualPDBWithSelector("testPDB", strVal("50%"), nil, &metav1.LabelSelector{}), intVal(3), nil),
			),
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				vPDB := testVirtualPDBWithSelector("testPDB", strVal("50%"), nil, &metav1.LabelSelector{})
				_, err := syncer.(*pdbSyncer).SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(vPDB))
				assert.NilError(t, err)
			},
		},
		{
			Name: "Empty Selector in Mixed Namespace with Deployment and Bare Pod",
			InitialVirtualState: []runtime.Object{
				testDeployment("nginx-deploy", 3, nginxLabels),
				testReplicaSet("nginx-rs-1", "nginx-deploy", 3, nginxLabels),
				testPod("pod-1", "nginx-rs-1", nginxLabels),
				testPod("pod-2", "nginx-rs-1", nginxLabels),
				testPod("pod-3", "nginx-rs-1", nginxLabels),
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "bare-pod",
						Namespace:       "default",
						ResourceVersion: syncertesting.FakeClientResourceVersion,
					},
				},
				testVirtualPDBWithSelector("testPDB", nil, intVal(1), &metav1.LabelSelector{}),
			},
			ExpectedVirtualState: pdbState(testVirtualPDBWithSelector("testPDB", nil, intVal(1), &metav1.LabelSelector{})),
			ExpectedPhysicalState: pdbState(
				// 3 deployment replicas + 1 bare pod = 4 pods; 4 - 1 = 3 minAvailable
				testHostPDB(testVirtualPDBWithSelector("testPDB", nil, intVal(1), &metav1.LabelSelector{}), intVal(3), nil),
			),
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				vPDB := testVirtualPDBWithSelector("testPDB", nil, intVal(1), &metav1.LabelSelector{})
				_, err := syncer.(*pdbSyncer).SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(vPDB))
				assert.NilError(t, err)
			},
		},
		{
			Name: "Empty Selector in Empty Namespace Results in minAvailable 0",
			InitialVirtualState: []runtime.Object{
				testVirtualPDBWithSelector("testPDB", nil, intVal(1), &metav1.LabelSelector{}),
			},
			ExpectedVirtualState: pdbState(testVirtualPDBWithSelector("testPDB", nil, intVal(1), &metav1.LabelSelector{})),
			ExpectedPhysicalState: pdbState(
				// 0 pods, 0 workloads -> expectedCount 0 -> minAvailable 0
				testHostPDB(testVirtualPDBWithSelector("testPDB", nil, intVal(1), &metav1.LabelSelector{}), intVal(0), nil),
			),
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				vPDB := testVirtualPDBWithSelector("testPDB", nil, intVal(1), &metav1.LabelSelector{})
				_, err := syncer.(*pdbSyncer).SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(vPDB))
				assert.NilError(t, err)
			},
		},
		{
			Name: "Empty Selector Refreshes on Deployment Scale-Up",
			InitialVirtualState: []runtime.Object{
				testDeployment("nginx-deploy", 5, nginxLabels),
				testVirtualPDBWithSelector("testPDB", nil, intVal(1), &metav1.LabelSelector{}),
			},
			InitialPhysicalState: []runtime.Object{
				testHostPDB(testVirtualPDBWithSelector("testPDB", nil, intVal(1), &metav1.LabelSelector{}), intVal(2), nil),
			},
			ExpectedVirtualState: pdbState(testVirtualPDBWithSelector("testPDB", nil, intVal(1), &metav1.LabelSelector{})),
			ExpectedPhysicalState: pdbState(
				// 5 replicas - 1 maxUnavailable = 4 minAvailable
				testHostPDB(testVirtualPDBWithSelector("testPDB", nil, intVal(1), &metav1.LabelSelector{}), intVal(4), nil),
			),
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				vPDB := testVirtualPDBWithSelector("testPDB", nil, intVal(1), &metav1.LabelSelector{})
				pPDB := testHostPDB(vPDB, intVal(2), nil)
				_, err := syncer.(*pdbSyncer).Sync(syncCtx, synccontext.NewSyncEventWithOld(
					pPDB.DeepCopy(),
					pPDB.DeepCopy(),
					vPDB.DeepCopy(),
					vPDB.DeepCopy(),
				))
				assert.NilError(t, err)
			},
		},
		{
			Name: "Integer minAvailable with Empty Selector Passthrough Unchanged",
			InitialVirtualState: []runtime.Object{
				testDeployment("nginx-deploy", 5, nginxLabels),
				testVirtualPDBWithSelector("testPDB", intVal(2), nil, &metav1.LabelSelector{}),
			},
			ExpectedVirtualState: pdbState(testVirtualPDBWithSelector("testPDB", intVal(2), nil, &metav1.LabelSelector{})),
			ExpectedPhysicalState: pdbState(
				// Integer minAvailable passes through unchanged regardless of replica count
				testHostPDB(testVirtualPDBWithSelector("testPDB", intVal(2), nil, &metav1.LabelSelector{}), intVal(2), nil),
			),
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				vPDB := testVirtualPDBWithSelector("testPDB", intVal(2), nil, &metav1.LabelSelector{})
				_, err := syncer.(*pdbSyncer).SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(vPDB))
				assert.NilError(t, err)
			},
		},
		{
			Name: "Labeled Selector in Multi-Workload Namespace Counts Only Matching Workload",
			InitialVirtualState: []runtime.Object{
				testDeployment("nginx-deploy", 3, nginxLabels),
				testStatefulSet("db-sts", 2, dbLabels),
				testVirtualPDB("testPDB", nil, intVal(1), nginxLabels),
			},
			ExpectedVirtualState: pdbState(testVirtualPDB("testPDB", nil, intVal(1), nginxLabels)),
			ExpectedPhysicalState: pdbState(
				// Only nginx-deploy matches (3 - 1 = 2); db-sts is not counted
				testHostPDB(testVirtualPDB("testPDB", nil, intVal(1), nginxLabels), intVal(2), nil),
			),
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				vPDB := testVirtualPDB("testPDB", nil, intVal(1), nginxLabels)
				_, err := syncer.(*pdbSyncer).SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(vPDB))
				assert.NilError(t, err)
			},
		},
		{
			Name: "Empty Selector with Sibling Workload Having Zero Pods",
			InitialVirtualState: []runtime.Object{
				testDeployment("deploy-a", 3, nginxLabels),
				testReplicaSet("rs-a", "deploy-a", 3, nginxLabels),
				testPod("pod-a1", "rs-a", nginxLabels),
				testPod("pod-a2", "rs-a", nginxLabels),
				testPod("pod-a3", "rs-a", nginxLabels),
				testDeployment("deploy-b", 2, dbLabels),
				testVirtualPDBWithSelector("testPDB", nil, intVal(1), &metav1.LabelSelector{}),
			},
			ExpectedVirtualState: pdbState(testVirtualPDBWithSelector("testPDB", nil, intVal(1), &metav1.LabelSelector{})),
			ExpectedPhysicalState: pdbState(
				// 3 (deploy-a) + 2 (deploy-b with no pods yet) - 1 = 4 minAvailable
				testHostPDB(testVirtualPDBWithSelector("testPDB", nil, intVal(1), &metav1.LabelSelector{}), intVal(4), nil),
			),
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				vPDB := testVirtualPDBWithSelector("testPDB", nil, intVal(1), &metav1.LabelSelector{})
				_, err := syncer.(*pdbSyncer).SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(vPDB))
				assert.NilError(t, err)
			},
		},
		{
			Name: "Missing Controller Pods Still Count When Another Workload Resolved",
			InitialVirtualState: []runtime.Object{
				testDeployment("deploy-a", 3, nginxLabels),
				missingControllerPod("orphan-1"),
				missingControllerPod("orphan-2"),
				testVirtualPDB("testPDB", nil, intVal(1), nginxLabels),
			},
			ExpectedVirtualState: pdbState(testVirtualPDB("testPDB", nil, intVal(1), nginxLabels)),
			ExpectedPhysicalState: pdbState(
				// deploy-a scale 3 plus 2 pods whose Deployment cannot be read; (3 + 2) - 1 = 4
				testHostPDB(testVirtualPDB("testPDB", nil, intVal(1), nginxLabels), intVal(4), nil),
			),
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				vPDB := testVirtualPDB("testPDB", nil, intVal(1), nginxLabels)
				_, err := syncer.(*pdbSyncer).SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(vPDB))
				assert.NilError(t, err)
			},
		},
		{
			Name: "Missing ReplicaSet Does Not Double Count Deployment Replicas",
			InitialVirtualState: []runtime.Object{
				testDeployment("nginx-deploy", 3, nginxLabels),
				// nginx-rs is not created. The pods still name it as their controller.
				testPod("pod-1", "nginx-rs", nginxLabels),
				testPod("pod-2", "nginx-rs", nginxLabels),
				testVirtualPDB("testPDB", nil, intVal(1), nginxLabels),
			},
			ExpectedVirtualState: pdbState(testVirtualPDB("testPDB", nil, intVal(1), nginxLabels)),
			ExpectedPhysicalState: pdbState(
				// 2 pods whose ReplicaSet cannot be read. The Deployment was not
				// resolved from that owner chain, so its 3 replicas are not added.
				// 2 - 1 maxUnavailable = 1. Counting both would be 2+3-1 = 4.
				testHostPDB(testVirtualPDB("testPDB", nil, intVal(1), nginxLabels), intVal(1), nil),
			),
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				vPDB := testVirtualPDB("testPDB", nil, intVal(1), nginxLabels)
				_, err := syncer.(*pdbSyncer).SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(vPDB))
				assert.NilError(t, err)
			},
		},
		{
			Name: "Resolved Deployment Covers Pods Whose ReplicaSet Is Gone",
			InitialVirtualState: []runtime.Object{
				testDeployment("nginx-deploy", 3, nginxLabels),
				testReplicaSet("nginx-rs-new", "nginx-deploy", 3, nginxLabels),
				testPod("pod-new-1", "nginx-rs-new", nginxLabels),
				testPod("pod-new-2", "nginx-rs-new", nginxLabels),
				// Old ReplicaSet was deleted. Its pod is still selected.
				testPod("pod-old", "nginx-rs-old", nginxLabels),
				testVirtualPDB("testPDB", nil, intVal(1), nginxLabels),
			},
			ExpectedVirtualState: pdbState(testVirtualPDB("testPDB", nil, intVal(1), nginxLabels)),
			ExpectedPhysicalState: pdbState(
				// Owner chain resolves the Deployment (3). The pod from the missing
				// ReplicaSet is already covered by that scale, so it is not added.
				// 3 - 1 = 2. Counting both would be 3+1-1 = 3.
				testHostPDB(testVirtualPDB("testPDB", nil, intVal(1), nginxLabels), intVal(2), nil),
			),
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				vPDB := testVirtualPDB("testPDB", nil, intVal(1), nginxLabels)
				_, err := syncer.(*pdbSyncer).SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(vPDB))
				assert.NilError(t, err)
			},
		},
		{
			Name: "Replica Count Wins Over Fresh DesiredHealthy",
			InitialVirtualState: []runtime.Object{
				testDeployment("nginx-deploy", 5, nginxLabels),
				freshDesiredHealthyPDB(),
			},
			ExpectedVirtualState: pdbState(freshDesiredHealthyPDB()),
			ExpectedPhysicalState: pdbState(
				// spec.replicas 5 with maxUnavailable 1 is host minAvailable 4. status.desiredHealthy is 2.
				testHostPDB(testVirtualPDB("testPDB", nil, intVal(1), nginxLabels), intVal(4), nil),
			),
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*pdbSyncer).SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(freshDesiredHealthyPDB()))
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "ExpectedPods Used Only When The Walk Finds Nothing",
			InitialVirtualState:  []runtime.Object{expectedPodsOnlyPDB()},
			ExpectedVirtualState: pdbState(expectedPodsOnlyPDB()),
			ExpectedPhysicalState: pdbState(
				// no pods and no workloads; expectedPods 4 - maxUnavailable 1 = 3
				testHostPDB(testVirtualPDB("testPDB", nil, intVal(1), nginxLabels), intVal(3), nil),
			),
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*pdbSyncer).SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(expectedPodsOnlyPDB()))
				assert.NilError(t, err)
			},
		},
	})
}

func TestModifyControllerMapping(t *testing.T) {
	ctx := context.Background()
	scheme := scheme.Scheme

	pdbNginx := testVirtualPDB("pdb-nginx", nil, intVal(1), map[string]string{"app": "nginx"})
	pdbRedis := testVirtualPDB("pdb-redis", strVal("50%"), nil, map[string]string{"app": "redis"})
	pdbPassthrough := testVirtualPDB("pdb-pass", intVal(2), nil, map[string]string{"app": "nginx"})

	pdbEmptyMaxUnavail := testVirtualPDBWithSelector("pdb-empty-max", nil, intVal(1), &metav1.LabelSelector{})
	pdbEmptyPctMinAvail := testVirtualPDBWithSelector("pdb-empty-pct", strVal("50%"), nil, &metav1.LabelSelector{})
	pdbEmptyIntMinAvail := testVirtualPDBWithSelector("pdb-empty-int", intVal(2), nil, &metav1.LabelSelector{})
	pdbNilSelector := testVirtualPDBWithSelector("pdb-nil-sel", nil, intVal(1), nil)
	pdbOtherNsEmpty := testVirtualPDBWithSelector("pdb-other-ns", nil, intVal(1), &metav1.LabelSelector{})
	pdbOtherNsEmpty.Namespace = "other"

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		pdbNginx, pdbRedis, pdbPassthrough,
		pdbEmptyMaxUnavail, pdbEmptyPctMinAvail, pdbEmptyIntMinAvail,
		pdbNilSelector, pdbOtherNsEmpty,
	).Build()

	// 1. Test mapWorkloadToPDBs with Pod matching pdb-nginx (also matches empty selector convertible PDBs)
	podNginx := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "pod-1", Labels: map[string]string{"app": "nginx"}},
	}
	reqs := mapWorkloadToPDBs(ctx, fakeClient, podNginx)
	assert.Equal(t, len(reqs), 3)
	reqMap := map[string]bool{}
	for _, r := range reqs {
		reqMap[r.Name] = true
	}
	assert.Assert(t, reqMap["pdb-nginx"])
	assert.Assert(t, reqMap["pdb-empty-max"])
	assert.Assert(t, reqMap["pdb-empty-pct"])
	assert.Assert(t, !reqMap["pdb-pass"])
	assert.Assert(t, !reqMap["pdb-empty-int"])
	assert.Assert(t, !reqMap["pdb-nil-sel"])
	assert.Assert(t, !reqMap["pdb-other-ns"])

	// 2. Test mapWorkloadToPDBs with Deployment matching pdb-redis (also matches empty selector convertible PDBs)
	deployRedis := testDeployment("redis-deploy", 2, map[string]string{"app": "redis"})
	reqs = mapWorkloadToPDBs(ctx, fakeClient, deployRedis)
	assert.Equal(t, len(reqs), 3)
	reqMap = map[string]bool{}
	for _, r := range reqs {
		reqMap[r.Name] = true
	}
	assert.Assert(t, reqMap["pdb-redis"])
	assert.Assert(t, reqMap["pdb-empty-max"])
	assert.Assert(t, reqMap["pdb-empty-pct"])

	// 3. Test mapWorkloadToPDBs with Pod with unrelated labels (matches only empty selector convertible PDBs)
	podOther := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "pod-other", Labels: map[string]string{"app": "other"}},
	}
	reqs = mapWorkloadToPDBs(ctx, fakeClient, podOther)
	assert.Equal(t, len(reqs), 2)
	reqMap = map[string]bool{}
	for _, r := range reqs {
		reqMap[r.Name] = true
	}
	assert.Assert(t, reqMap["pdb-empty-max"])
	assert.Assert(t, reqMap["pdb-empty-pct"])

	// 4. Test mapWorkloadToPDBs with unlabeled Pod (nil labels, matches only empty selector convertible PDBs)
	podUnlabeled := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "pod-unlabeled"},
	}
	reqs = mapWorkloadToPDBs(ctx, fakeClient, podUnlabeled)
	assert.Equal(t, len(reqs), 2)
	reqMap = map[string]bool{}
	for _, r := range reqs {
		reqMap[r.Name] = true
	}
	assert.Assert(t, reqMap["pdb-empty-max"])
	assert.Assert(t, reqMap["pdb-empty-pct"])

	// 5. Test mapWorkloadUpdateToPDBs unioning old and new labels (deduplicated)
	podOld := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "pod-1", Labels: map[string]string{"app": "nginx"}},
	}
	podNew := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "pod-1", Labels: map[string]string{"app": "redis"}},
	}
	reqs = mapWorkloadUpdateToPDBs(ctx, fakeClient, podOld, podNew)
	assert.Equal(t, len(reqs), 4)
	reqMap = map[string]bool{}
	for _, r := range reqs {
		reqMap[r.Name] = true
	}
	assert.Assert(t, reqMap["pdb-nginx"])
	assert.Assert(t, reqMap["pdb-redis"])
	assert.Assert(t, reqMap["pdb-empty-max"])
	assert.Assert(t, reqMap["pdb-empty-pct"])
	assert.Assert(t, !reqMap["pdb-nil-sel"])

	// 6. Test mapWorkloadUpdateToPDBs when new labels match nothing specific (relabel away from PDB)
	reqs = mapWorkloadUpdateToPDBs(ctx, fakeClient, podOld, podOther)
	assert.Equal(t, len(reqs), 3)
	reqMap = map[string]bool{}
	for _, r := range reqs {
		reqMap[r.Name] = true
	}
	assert.Assert(t, reqMap["pdb-nginx"])
	assert.Assert(t, reqMap["pdb-empty-max"])
	assert.Assert(t, reqMap["pdb-empty-pct"])

	// 7. Test mapWorkloadUpdateToPDBs when both old and new match same PDB (deduplication)
	reqs = mapWorkloadUpdateToPDBs(ctx, fakeClient, podOld, podOld)
	assert.Equal(t, len(reqs), 3)
	reqMap = map[string]bool{}
	for _, r := range reqs {
		reqMap[r.Name] = true
	}
	assert.Assert(t, reqMap["pdb-nginx"])
	assert.Assert(t, reqMap["pdb-empty-max"])
	assert.Assert(t, reqMap["pdb-empty-pct"])

	// 8. Test mapWorkloadUpdateToPDBs for Deployment replica change
	deployRedisScaled := testDeployment("redis-deploy", 5, map[string]string{"app": "redis"})
	reqs = mapWorkloadUpdateToPDBs(ctx, fakeClient, deployRedis, deployRedisScaled)
	assert.Equal(t, len(reqs), 3)
	reqMap = map[string]bool{}
	for _, r := range reqs {
		reqMap[r.Name] = true
	}
	assert.Assert(t, reqMap["pdb-redis"])
	assert.Assert(t, reqMap["pdb-empty-max"])
	assert.Assert(t, reqMap["pdb-empty-pct"])
	assert.Assert(t, !reqMap["pdb-nil-sel"])
}

func TestPredicates(t *testing.T) {
	wPred := workloadPredicate()
	pPred := podPredicate()

	// Workload predicate tests
	deployOld := testDeployment("deploy", 3, map[string]string{"app": "nginx"})
	deployNew := testDeployment("deploy", 3, map[string]string{"app": "nginx"})

	// Status-only or identical: false
	assert.Assert(t, !wPred.Update(event.UpdateEvent{ObjectOld: deployOld, ObjectNew: deployNew}))

	// Replicas change: true
	deployNewReplicas := testDeployment("deploy", 5, map[string]string{"app": "nginx"})
	assert.Assert(t, wPred.Update(event.UpdateEvent{ObjectOld: deployOld, ObjectNew: deployNewReplicas}))

	// Template labels change: true
	deployNewLabels := testDeployment("deploy", 3, map[string]string{"app": "nginx2"})
	assert.Assert(t, wPred.Update(event.UpdateEvent{ObjectOld: deployOld, ObjectNew: deployNewLabels}))

	// DeletionTimestamp change: true
	deployDeleting := deployOld.DeepCopy()
	now := metav1.Now()
	deployDeleting.DeletionTimestamp = &now
	assert.Assert(t, wPred.Update(event.UpdateEvent{ObjectOld: deployOld, ObjectNew: deployDeleting}))

	// StatefulSet replicas change: true
	stsOld := testStatefulSet("sts", 2, map[string]string{"app": "db"})
	stsNew := testStatefulSet("sts", 4, map[string]string{"app": "db"})
	assert.Assert(t, wPred.Update(event.UpdateEvent{ObjectOld: stsOld, ObjectNew: stsNew}))

	// Pod predicate tests
	podOld := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "pod-1",
			Labels:    map[string]string{"app": "nginx"},
		},
		Status: corev1.PodStatus{Phase: corev1.PodRunning},
	}
	podNew := podOld.DeepCopy()

	// Identical pod: false
	assert.Assert(t, !pPred.Update(event.UpdateEvent{ObjectOld: podOld, ObjectNew: podNew}))

	// Labels change: true
	podRelabeled := podOld.DeepCopy()
	podRelabeled.Labels = map[string]string{"app": "nginx2"}
	assert.Assert(t, pPred.Update(event.UpdateEvent{ObjectOld: podOld, ObjectNew: podRelabeled}))

	// OwnerReferences change: true
	podOwnerChanged := podOld.DeepCopy()
	podOwnerChanged.OwnerReferences = []metav1.OwnerReference{{APIVersion: "apps/v1", Kind: "ReplicaSet", Name: "rs-1", UID: "uid-1"}}
	assert.Assert(t, pPred.Update(event.UpdateEvent{ObjectOld: podOld, ObjectNew: podOwnerChanged}))

	// DeletionTimestamp change: true
	podDeleting := podOld.DeepCopy()
	podDeleting.DeletionTimestamp = &now
	assert.Assert(t, pPred.Update(event.UpdateEvent{ObjectOld: podOld, ObjectNew: podDeleting}))

	// Non-terminal phase change (Pending -> Running): false
	podPending := podOld.DeepCopy()
	podPending.Status.Phase = corev1.PodPending
	assert.Assert(t, !pPred.Update(event.UpdateEvent{ObjectOld: podPending, ObjectNew: podOld}))

	// Terminal phase change (Running -> Succeeded): true
	podSucceeded := podOld.DeepCopy()
	podSucceeded.Status.Phase = corev1.PodSucceeded
	assert.Assert(t, pPred.Update(event.UpdateEvent{ObjectOld: podOld, ObjectNew: podSucceeded}))

	// Terminal phase change (Running -> Failed): true
	podFailed := podOld.DeepCopy()
	podFailed.Status.Phase = corev1.PodFailed
	assert.Assert(t, pPred.Update(event.UpdateEvent{ObjectOld: podOld, ObjectNew: podFailed}))
}
