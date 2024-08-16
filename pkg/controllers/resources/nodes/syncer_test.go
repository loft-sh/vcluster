package nodes

import (
	"testing"

	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertesting "github.com/loft-sh/vcluster/pkg/syncer/testing"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"gotest.tools/assert"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/loft-sh/vcluster/pkg/constants"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

var (
	baseName = types.NamespacedName{
		Name: "mynode",
	}
	basePod = corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "mypod",
		},
		Spec: corev1.PodSpec{
			NodeName: baseName.Name,
		},
	}

	baseNode = corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: baseName.Name,
		},
		Status: corev1.NodeStatus{
			DaemonEndpoints: corev1.NodeDaemonEndpoints{
				KubeletEndpoint: corev1.DaemonEndpoint{
					Port: 0,
				},
			},
		},
	}
	baseVNode = corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: baseName.Name,
		},
		Status: corev1.NodeStatus{
			Addresses: []corev1.NodeAddress{
				{
					Address: GetNodeHost(baseName.Name),
					Type:    corev1.NodeHostName,
				},
			},
			DaemonEndpoints: corev1.NodeDaemonEndpoints{
				KubeletEndpoint: corev1.DaemonEndpoint{
					Port: constants.KubeletPort,
				},
			},
		},
	}
)

func TestSyncToVirtual(t *testing.T) {
	testCases := []struct {
		name           string
		withVirtualPod bool
		expectNode     bool
	}{
		{
			name:           "Create backward",
			withVirtualPod: true,
			expectNode:     true,
		},
		{
			name: "Create backward not needed",
		},
	}
	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			baseNode := baseNode.DeepCopy()

			initialObjects := []runtime.Object{}
			expectedVirtualObjects := map[schema.GroupVersionKind][]runtime.Object{}

			if tC.withVirtualPod {
				initialObjects = append(initialObjects, basePod.DeepCopy())
				expectedVirtualObjects[corev1.SchemeGroupVersion.WithKind("Pod")] = []runtime.Object{basePod.DeepCopy()}
			}

			if tC.expectNode {
				expectedVirtualObjects[corev1.SchemeGroupVersion.WithKind("Node")] = []runtime.Object{baseNode.DeepCopy()}
			}

			test := syncertesting.SyncTest{
				Name:                 tC.name,
				InitialVirtualState:  initialObjects,
				ExpectedVirtualState: expectedVirtualObjects,
				Sync: func(ctx *synccontext.RegisterContext) {
					ctx.Config.Networking.Advanced.ProxyKubelets.ByIP = false
					syncCtx, syncer := newFakeSyncer(t, ctx)
					_, err := syncer.SyncToVirtual(syncCtx, synccontext.NewSyncToVirtualEvent(baseNode.DeepCopy()))
					assert.NilError(t, err)
				},
			}

			test.Run(t, syncertesting.NewFakeRegisterContext)
		})
	}
}

func TestSyncBothExist(t *testing.T) {
	editedNode := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: baseName.Name,
			Labels: map[string]string{
				"test": "true",
			},
			Annotations: map[string]string{
				"test":                                 "true",
				translate.ManagedAnnotationsAnnotation: "test",
				translate.ManagedLabelsAnnotation:      "test",
			},
		},
		Status: corev1.NodeStatus{
			Addresses: []corev1.NodeAddress{
				{
					Address: GetNodeHost(baseName.Name),
					Type:    corev1.NodeHostName,
				},
			},
			DaemonEndpoints: corev1.NodeDaemonEndpoints{
				KubeletEndpoint: corev1.DaemonEndpoint{
					Port: constants.KubeletPort,
				},
			},
			NodeInfo: corev1.NodeSystemInfo{
				Architecture: "amd64",
			},
		},
	}
	testCases := []struct {
		name                      string
		withVirtualPod            bool
		withPhysicalPod           bool
		virtualNodeExists         bool
		modifiedPhysical          bool
		expectNoVNode             bool
		syncNodes                 bool
		syncFromHostLabel         map[string]string
		hostLabel                 map[string]string
		virtualFinalLabels        map[string]string
		virtualInitialAnnotations map[string]string
		virtualFinalAnnotations   map[string]string
		physicalAnnotations       map[string]string
		imagesPhysicalNode        []corev1.ContainerImage
		enforceTolerations        []string
		clearImage                bool
		physicalTaints            []corev1.Taint
		expectedTaints            []corev1.Taint
	}{
		{
			name:              "Update backward no change",
			withVirtualPod:    true,
			virtualNodeExists: true,
			modifiedPhysical:  false,
		},
		{
			name:              "Update backward",
			withVirtualPod:    true,
			virtualNodeExists: true,
			modifiedPhysical:  true,
		},
		{
			name:              "Delete backward",
			virtualNodeExists: true,
			expectNoVNode:     true,
		},
		{
			name:                    "Label Matched and enforceNodeSelector false - expect node to be synced from NodeSelector",
			virtualNodeExists:       true,
			syncFromHostLabel:       map[string]string{"test": "true"},
			hostLabel:               map[string]string{"test": "true"},
			virtualFinalAnnotations: map[string]string{translate.ManagedLabelsAnnotation: "test"},
			virtualFinalLabels:      map[string]string{"test": "true"},
		},
		{
			name:              "Label Not Matched and enforceNodeSelector true - expect node not to be synced",
			virtualNodeExists: true,
			syncFromHostLabel: map[string]string{"test": "true"},
			expectNoVNode:     true,
		},
		{
			name:               "Clear Nodes",
			virtualNodeExists:  true,
			withVirtualPod:     true,
			clearImage:         true,
			imagesPhysicalNode: []corev1.ContainerImage{{Names: []string{"ghcr.io/jetpack/calico"}}},
		},
		{
			name:               "Don't Clear Nodes",
			virtualNodeExists:  true,
			withVirtualPod:     true,
			imagesPhysicalNode: []corev1.ContainerImage{{Names: []string{"ghcr.io/jetpack/calico"}}},
		},
		{
			name: "Matching taints",
			physicalAnnotations: map[string]string{
				TaintsAnnotation: "[\"{\\\"key\\\":\\\"key1\\\",\\\"value\\\":\\\"value1\\\",\\\"effect\\\":\\\"NoSchedule\\\"}\"]",
			},
			virtualFinalAnnotations: map[string]string{
				TaintsAnnotation: "[\"{\\\"key\\\":\\\"key1\\\",\\\"value\\\":\\\"value1\\\",\\\"effect\\\":\\\"NoSchedule\\\"}\"]",
			},
			virtualNodeExists:  true,
			withVirtualPod:     true,
			enforceTolerations: []string{":NoSchedule op=Exists"},
			physicalTaints: []corev1.Taint{
				{
					Key:    "key1",
					Value:  "value1",
					Effect: "NoSchedule",
				},
			},
		},
		{
			name: "Not Matching taints",
			physicalAnnotations: map[string]string{
				TaintsAnnotation: "[\"{\\\"key\\\":\\\"key1\\\",\\\"value\\\":\\\"value1\\\",\\\"effect\\\":\\\"NoSchedule\\\"}\"]",
			},
			virtualFinalAnnotations: map[string]string{
				TaintsAnnotation: "[\"{\\\"key\\\":\\\"key1\\\",\\\"value\\\":\\\"value1\\\",\\\"effect\\\":\\\"NoSchedule\\\"}\"]",
			},
			virtualNodeExists:  true,
			withVirtualPod:     true,
			enforceTolerations: []string{"key2=value2:NoSchedule"},
			physicalTaints: []corev1.Taint{
				{
					Key:    "key1",
					Value:  "value1",
					Effect: "NoSchedule",
				},
			},
			expectedTaints: []corev1.Taint{
				{
					Key:    "key1",
					Value:  "value1",
					Effect: "NoSchedule",
				},
			},
		},
		{
			name: "Taint matching Enforced Toleration - special case of empty key with Exists operator",
			physicalAnnotations: map[string]string{
				TaintsAnnotation: "[\"{\\\"key\\\":\\\"key1\\\",\\\"value\\\":\\\"value1\\\",\\\"effect\\\":\\\"NoSchedule\\\"}\"]",
			},
			virtualFinalAnnotations: map[string]string{
				TaintsAnnotation: "[\"{\\\"key\\\":\\\"key1\\\",\\\"value\\\":\\\"value1\\\",\\\"effect\\\":\\\"NoSchedule\\\"}\"]",
			},
			virtualNodeExists:  true,
			withVirtualPod:     true,
			enforceTolerations: []string{":NoSchedule op=Exists"},
			physicalTaints: []corev1.Taint{
				{
					Key:    "key1",
					Value:  "value1",
					Effect: "NoSchedule",
				},
			},
		},
		{
			name:              "Nodes syncing enabled -- Ignore updates to Rancher managed annotations",
			withVirtualPod:    true,
			withPhysicalPod:   true,
			virtualNodeExists: true,
			syncNodes:         true,
			physicalAnnotations: map[string]string{
				RancherAgentPodRequestsAnnotation: "{\"pods\":\"3\"}",
				RancherAgentPodLimitsAnnotation:   "{\"pods\":\"10\"}",
			},
			virtualInitialAnnotations: map[string]string{
				RancherAgentPodRequestsAnnotation: "{\"pods\":\"1\"}",
				RancherAgentPodLimitsAnnotation:   "{\"pods\":\"5\"}",
			},
			virtualFinalAnnotations: map[string]string{
				RancherAgentPodRequestsAnnotation: "{\"pods\":\"1\"}",
				RancherAgentPodLimitsAnnotation:   "{\"pods\":\"5\"}",
			},
		},
	}
	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			initialVNode := baseVNode.DeepCopy()
			expectedVNode := baseVNode.DeepCopy()
			physical := baseNode.DeepCopy()

			initialObjects := []runtime.Object{}
			expectedVirtualObjects := map[schema.GroupVersionKind][]runtime.Object{}

			if tC.withVirtualPod {
				initialObjects = append(initialObjects, basePod.DeepCopy())
				expectedVirtualObjects[corev1.SchemeGroupVersion.WithKind("Pod")] = []runtime.Object{basePod.DeepCopy()}
			}

			if tC.virtualNodeExists {
				node := initialVNode.DeepCopy()
				if len(tC.virtualInitialAnnotations) > 0 {
					node.Annotations = tC.virtualInitialAnnotations
				}

				initialObjects = append(initialObjects, node)
			}

			physical.Labels = tC.hostLabel
			physical.Status.Images = tC.imagesPhysicalNode
			if !tC.clearImage && tC.imagesPhysicalNode != nil {
				expectedVNode.Status.Images = tC.imagesPhysicalNode
			}
			physical.Spec.Taints = tC.physicalTaints

			if !tC.expectNoVNode {
				expectedVirtualObjects[corev1.SchemeGroupVersion.WithKind("Node")] = []runtime.Object{expectedVNode}
				expectedVNode.Labels = tC.virtualFinalLabels
				expectedVNode.Annotations = tC.virtualFinalAnnotations
			}
			expectedVNode.Spec.Taints = tC.expectedTaints

			if tC.modifiedPhysical {
				expectedVirtualObjects[corev1.SchemeGroupVersion.WithKind("Node")] = []runtime.Object{editedNode.DeepCopy()}
				physical = editedNode.DeepCopy()
			}

			test := syncertesting.SyncTest{
				Name:                 tC.name,
				InitialVirtualState:  initialObjects,
				ExpectedVirtualState: expectedVirtualObjects,
			}

			// setting up the clients
			pClient, vClient, vConfig := test.Setup()
			registerContext := syncertesting.NewFakeRegisterContext(vConfig, pClient, vClient)

			registerContext.Config.Networking.Advanced.ProxyKubelets.ByIP = false
			registerContext.Config.Sync.FromHost.Nodes.Enabled = tC.syncNodes
			registerContext.Config.Sync.FromHost.Nodes.ClearImageStatus = tC.clearImage
			registerContext.Config.Sync.ToHost.Pods.EnforceTolerations = tC.enforceTolerations
			registerContext.Config.Sync.FromHost.Nodes.Selector.Labels = tC.syncFromHostLabel

			syncCtx, syncer := newFakeSyncer(t, registerContext)
			_, err := syncer.Sync(syncCtx, synccontext.NewSyncEvent(physical, initialVNode.DeepCopy()))
			assert.NilError(t, err)

			test.Validate(t)
		})
	}
}

func newFakeSyncer(t *testing.T, ctx *synccontext.RegisterContext) (*synccontext.SyncContext, *nodeSyncer) {
	// we need that index here as well otherwise we wouldn't find the related pod
	err := ctx.VirtualManager.GetFieldIndexer().IndexField(ctx, &corev1.Pod{}, constants.IndexByAssigned, func(rawObj client.Object) []string {
		pod := rawObj.(*corev1.Pod)
		return []string{pod.Spec.NodeName}
	})
	assert.NilError(t, err)

	syncContext, object := syncertesting.FakeStartSyncer(t, ctx, func(ctx *synccontext.RegisterContext) (syncertypes.Object, error) {
		return NewSyncer(ctx, &fakeNodeServiceProvider{})
	})
	return syncContext, object.(*nodeSyncer)
}
