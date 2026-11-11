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

func testVirtualPDB(name string, minAvailable, maxUnavailable *intstr.IntOrString, selector map[string]string) *policyv1.PodDisruptionBudget {
	pdb := &policyv1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       "default",
			ResourceVersion: syncertesting.FakeClientResourceVersion,
		},
		Spec: policyv1.PodDisruptionBudgetSpec{
			MinAvailable:   minAvailable,
			MaxUnavailable: maxUnavailable,
		},
	}
	if len(selector) > 0 {
		pdb.Spec.Selector = &metav1.LabelSelector{MatchLabels: selector}
	}
	return pdb
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
	})
}

func TestModifyControllerMapping(t *testing.T) {
	ctx := context.Background()
	scheme := scheme.Scheme

	pdbNginx := testVirtualPDB("pdb-nginx", nil, intVal(1), map[string]string{"app": "nginx"})
	pdbRedis := testVirtualPDB("pdb-redis", strVal("50%"), nil, map[string]string{"app": "redis"})
	pdbPassthrough := testVirtualPDB("pdb-pass", intVal(2), nil, map[string]string{"app": "nginx"})

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(pdbNginx, pdbRedis, pdbPassthrough).Build()

	// 1. Test mapWorkloadToPDBs with Pod matching pdb-nginx
	podNginx := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "pod-1", Labels: map[string]string{"app": "nginx"}},
	}
	reqs := mapWorkloadToPDBs(ctx, fakeClient, podNginx)
	assert.Equal(t, len(reqs), 1)
	assert.Equal(t, reqs[0].Name, "pdb-nginx")

	// 2. Test mapWorkloadToPDBs with Deployment matching pdb-redis
	deployRedis := testDeployment("redis-deploy", 2, map[string]string{"app": "redis"})
	reqs = mapWorkloadToPDBs(ctx, fakeClient, deployRedis)
	assert.Equal(t, len(reqs), 1)
	assert.Equal(t, reqs[0].Name, "pdb-redis")

	// 3. Test mapWorkloadToPDBs with no match
	podOther := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "pod-other", Labels: map[string]string{"app": "other"}},
	}
	reqs = mapWorkloadToPDBs(ctx, fakeClient, podOther)
	assert.Equal(t, len(reqs), 0)

	// 4. Test mapWorkloadUpdateToPDBs unioning old and new labels
	podOld := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "pod-1", Labels: map[string]string{"app": "nginx"}},
	}
	podNew := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "pod-1", Labels: map[string]string{"app": "redis"}},
	}
	reqs = mapWorkloadUpdateToPDBs(ctx, fakeClient, podOld, podNew)
	assert.Equal(t, len(reqs), 2)
	names := map[string]bool{reqs[0].Name: true, reqs[1].Name: true}
	assert.Assert(t, names["pdb-nginx"])
	assert.Assert(t, names["pdb-redis"])

	// 5. Test mapWorkloadUpdateToPDBs when new labels match nothing (relabel away from PDB)
	reqs = mapWorkloadUpdateToPDBs(ctx, fakeClient, podOld, podOther)
	assert.Equal(t, len(reqs), 1)
	assert.Equal(t, reqs[0].Name, "pdb-nginx")

	// 6. Test mapWorkloadUpdateToPDBs when both old and new match same PDB (deduplication)
	reqs = mapWorkloadUpdateToPDBs(ctx, fakeClient, podOld, podOld)
	assert.Equal(t, len(reqs), 1)
	assert.Equal(t, reqs[0].Name, "pdb-nginx")
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
