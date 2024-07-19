package nodes

import (
	"testing"

	"github.com/loft-sh/vcluster/pkg/syncer"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertesting "github.com/loft-sh/vcluster/pkg/syncer/testing"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"gotest.tools/assert"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/loft-sh/vcluster/pkg/constants"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

var (
	baseName types.NamespacedName = types.NamespacedName{
		Name: "mynode",
	}
	basePod corev1.Pod = corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "mypod",
		},
		Spec: corev1.PodSpec{
			NodeName: baseName.Name,
		},
	}

	baseNode corev1.Node = corev1.Node{
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
	baseVNode corev1.Node = corev1.Node{
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

func TestNodeDeletion(t *testing.T) {
	baseName := types.NamespacedName{
		Name: "mynode",
	}
	baseNode := &corev1.Node{
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

	syncertesting.RunTests(t, []*syncertesting.SyncTest{
		{
			Name:                 "Delete unused node backwards",
			InitialVirtualState:  []runtime.Object{baseNode},
			InitialPhysicalState: []runtime.Object{baseNode},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Node"): {},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				_, nodesSyncer := newFakeSyncer(t, ctx)
				syncController, err := syncer.NewSyncController(ctx, nodesSyncer)
				assert.NilError(t, err)

				_, err = syncController.Reconcile(ctx, controllerruntime.Request{NamespacedName: baseName})
				assert.NilError(t, err)
			},
		},
	})
}

func TestSync(t *testing.T) {
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

	syncertesting.RunTests(t, []*syncertesting.SyncTest{
		{
			Name:                "Create backward",
			InitialVirtualState: []runtime.Object{basePod.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Node"): {baseNode.DeepCopy()},
				corev1.SchemeGroupVersion.WithKind("Pod"):  {basePod.DeepCopy()},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := newFakeSyncer(t, ctx)
				_, err := syncer.SyncToVirtual(syncCtx, baseNode.DeepCopy())
				assert.NilError(t, err)
			},
		},
		{
			Name:                "Create backward not needed",
			InitialVirtualState: []runtime.Object{},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Node"): {},
				corev1.SchemeGroupVersion.WithKind("Pod"):  {},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := newFakeSyncer(t, ctx)
				_, err := syncer.SyncToVirtual(syncCtx, baseNode.DeepCopy())
				assert.NilError(t, err)
			},
		},
		{
			Name:                "Update backward",
			InitialVirtualState: []runtime.Object{basePod.DeepCopy(), baseNode.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Node"): {editedNode.DeepCopy()},
				corev1.SchemeGroupVersion.WithKind("Pod"):  {basePod.DeepCopy()},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				ctx.Config.Networking.Advanced.ProxyKubelets.ByIP = false
				syncCtx, syncer := newFakeSyncer(t, ctx)
				_, err := syncer.Sync(syncCtx, editedNode.DeepCopy(), baseNode.DeepCopy())
				assert.NilError(t, err)

				err = ctx.VirtualManager.GetClient().Get(ctx, types.NamespacedName{Name: baseNode.Name}, baseNode.DeepCopy())
				assert.NilError(t, err)

				_, err = syncer.Sync(syncCtx, editedNode.DeepCopy(), baseNode.DeepCopy())
				assert.NilError(t, err)
			},
		},
		{
			Name:                "Update backward no change",
			InitialVirtualState: []runtime.Object{basePod.DeepCopy(), baseNode.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Node"): {baseNode.DeepCopy()},
				corev1.SchemeGroupVersion.WithKind("Pod"):  {basePod.DeepCopy()},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				ctx.Config.Networking.Advanced.ProxyKubelets.ByIP = false
				syncCtx, syncer := newFakeSyncer(t, ctx)
				_, err := syncer.Sync(syncCtx, baseNode.DeepCopy(), baseVNode.DeepCopy())
				assert.NilError(t, err)
			},
		},
		{
			Name:                "Delete backward",
			InitialVirtualState: []runtime.Object{baseNode.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Node"): {},
				corev1.SchemeGroupVersion.WithKind("Pod"):  {},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := newFakeSyncer(t, ctx)
				_, err := syncer.Sync(syncCtx, baseNode.DeepCopy(), baseNode.DeepCopy())
				assert.NilError(t, err)
			},
		},
	})
}

func TestTaints(t *testing.T) {
	baseNode := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: baseName.Name,
			Annotations: map[string]string{
				TaintsAnnotation: "[\"{\\\"key\\\":\\\"key1\\\",\\\"value\\\":\\\"value1\\\",\\\"effect\\\":\\\"NoSchedule\\\"}\"]",
			},
		},
		Spec: corev1.NodeSpec{
			Taints: []corev1.Taint{
				{
					Key:    "key1",
					Value:  "value1",
					Effect: "NoSchedule",
				},
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

	editedNode := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: baseName.Name,
			Annotations: map[string]string{
				TaintsAnnotation: "[\"{\\\"key\\\":\\\"key1\\\",\\\"value\\\":\\\"value1\\\",\\\"effect\\\":\\\"NoSchedule\\\"}\"]",
			},
		},
		Spec: corev1.NodeSpec{
			Taints: nil,
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

	syncertesting.RunTests(t, []*syncertesting.SyncTest{
		{
			Name:                 "Taint matching Enforced Toleration",
			InitialPhysicalState: []runtime.Object{basePod.DeepCopy(), baseNode.DeepCopy()},
			InitialVirtualState:  []runtime.Object{basePod.DeepCopy(), baseNode.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Node"): {editedNode.DeepCopy()},
				corev1.SchemeGroupVersion.WithKind("Pod"):  {basePod.DeepCopy()},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				ctx.Config.Sync.ToHost.Pods.EnforceTolerations = []string{"key1=value1:NoSchedule"}
				ctx.Config.Networking.Advanced.ProxyKubelets.ByIP = false
				syncCtx, syncer := newFakeSyncer(t, ctx)
				_, err := syncer.Sync(syncCtx, baseNode.DeepCopy(), baseNode.DeepCopy())
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Taint not matching Enforced Toleration",
			InitialPhysicalState: []runtime.Object{basePod.DeepCopy(), baseNode.DeepCopy()},
			InitialVirtualState:  []runtime.Object{basePod.DeepCopy(), baseNode.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Node"): {baseNode.DeepCopy()},
				corev1.SchemeGroupVersion.WithKind("Pod"):  {basePod.DeepCopy()},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				ctx.Config.Sync.ToHost.Pods.EnforceTolerations = []string{"key2=value2:NoSchedule"}
				ctx.Config.Networking.Advanced.ProxyKubelets.ByIP = false
				syncCtx, syncer := newFakeSyncer(t, ctx)
				_, err := syncer.Sync(syncCtx, baseNode.DeepCopy(), baseNode.DeepCopy())
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Taint matching Enforced Toleration - special case of empty key with Exists operator",
			InitialPhysicalState: []runtime.Object{basePod.DeepCopy(), baseNode.DeepCopy()},
			InitialVirtualState:  []runtime.Object{basePod.DeepCopy(), baseNode.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Node"): {editedNode.DeepCopy()},
				corev1.SchemeGroupVersion.WithKind("Pod"):  {basePod.DeepCopy()},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				ctx.Config.Sync.ToHost.Pods.EnforceTolerations = []string{":NoSchedule op=Exists"}
				ctx.Config.Networking.Advanced.ProxyKubelets.ByIP = false
				syncCtx, syncer := newFakeSyncer(t, ctx)
				_, err := syncer.Sync(syncCtx, baseNode.DeepCopy(), baseNode.DeepCopy())
				assert.NilError(t, err)
			},
		},
	})
}

func TestLabelSelector(t *testing.T) {
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
			Labels: map[string]string{
				"test": "true",
			},
		},
		Status: corev1.NodeStatus{
			DaemonEndpoints: corev1.NodeDaemonEndpoints{
				KubeletEndpoint: corev1.DaemonEndpoint{
					Port: 0,
				},
			},
			NodeInfo: corev1.NodeSystemInfo{
				Architecture: "amd64",
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
			NodeInfo: corev1.NodeSystemInfo{
				Architecture: "amd64",
			},
		},
	}
	editedNode := corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: baseName.Name,
			Annotations: map[string]string{
				translate.ManagedLabelsAnnotation: "test",
			},
			Labels: map[string]string{
				"test": "true",
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

	syncertesting.RunTests(t, []*syncertesting.SyncTest{
		{
			Name:                 "Label Matched and enforceNodeSelector false - expect node to be synced from NodeSelector",
			InitialPhysicalState: []runtime.Object{baseNode.DeepCopy()},
			InitialVirtualState:  []runtime.Object{baseVNode.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Node"): {editedNode.DeepCopy()},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				ctx.Config.Networking.Advanced.ProxyKubelets.ByIP = false
				ctx.Config.Sync.FromHost.Nodes.Selector.Labels = map[string]string{
					"test": "true",
				}
				syncCtx, syncer := newFakeSyncer(t, ctx)
				_, err := syncer.Sync(syncCtx, baseNode.DeepCopy(), baseVNode.DeepCopy())
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Label Not Matched and enforceNodeSelector false - expect node to be synced from pod needs",
			InitialPhysicalState: []runtime.Object{basePod.DeepCopy(), baseNode.DeepCopy()},
			InitialVirtualState:  []runtime.Object{basePod.DeepCopy(), baseVNode.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Node"): {editedNode.DeepCopy()},
				corev1.SchemeGroupVersion.WithKind("Pod"):  {basePod.DeepCopy()},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				ctx.Config.Networking.Advanced.ProxyKubelets.ByIP = false
				ctx.Config.Sync.FromHost.Nodes.Selector.Labels = map[string]string{
					"test": "true",
				}
				syncCtx, syncer := newFakeSyncer(t, ctx)
				_, err := syncer.Sync(syncCtx, baseNode.DeepCopy(), baseVNode.DeepCopy())
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "No NodeSelector LabelSet and enforceNodeSelector false - expect node to be synced from pod needs",
			InitialPhysicalState: []runtime.Object{basePod.DeepCopy(), baseNode.DeepCopy()},
			InitialVirtualState:  []runtime.Object{basePod.DeepCopy(), baseVNode.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Node"): {editedNode.DeepCopy()},
				corev1.SchemeGroupVersion.WithKind("Pod"):  {basePod.DeepCopy()},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				ctx.Config.Networking.Advanced.ProxyKubelets.ByIP = false
				syncCtx, syncer := newFakeSyncer(t, ctx)
				_, err := syncer.Sync(syncCtx, baseNode.DeepCopy(), baseVNode.DeepCopy())
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Label Not Matched and enforceNodeSelector true - expect node not to be synced",
			InitialPhysicalState: []runtime.Object{basePod.DeepCopy(), baseNode.DeepCopy()},
			InitialVirtualState:  []runtime.Object{baseVNode.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{},
			Sync: func(ctx *synccontext.RegisterContext) {
				ctx.Config.Networking.Advanced.ProxyKubelets.ByIP = false
				ctx.Config.Sync.FromHost.Nodes.Selector.Labels = map[string]string{
					"test": "true",
				}
				syncCtx, syncer := newFakeSyncer(t, ctx)
				_, err := syncer.Sync(syncCtx, baseNode.DeepCopy(), baseVNode.DeepCopy())
				assert.NilError(t, err)
			},
		},
	})
}

func TestClearNode(t *testing.T) {
	baseNode = corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: baseName.Name,
			Labels: map[string]string{
				"test": "true",
			},
		},
		Status: corev1.NodeStatus{
			DaemonEndpoints: corev1.NodeDaemonEndpoints{
				KubeletEndpoint: corev1.DaemonEndpoint{
					Port: 0,
				},
			},
			NodeInfo: corev1.NodeSystemInfo{
				Architecture: "amd64",
			},
			Images: []corev1.ContainerImage{
				{
					Names: []string{"ghcr.io/jetpack/calico"},
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
			NodeInfo: corev1.NodeSystemInfo{
				Architecture: "amd64",
			},
		},
	}
	editedNode := corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: baseName.Name,
			Annotations: map[string]string{
				translate.ManagedLabelsAnnotation: "test",
			},
			Labels: map[string]string{
				"test": "true",
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
			Images: []corev1.ContainerImage{},
		},
	}

	syncertesting.RunTests(t, []*syncertesting.SyncTest{
		{
			Name:                 "Clear Node Images Enabled -- Synced Node Should have no images in status.images",
			InitialPhysicalState: []runtime.Object{baseNode.DeepCopy()},
			InitialVirtualState:  []runtime.Object{baseVNode.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Node"): {editedNode.DeepCopy()},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				ctx.Config.Networking.Advanced.ProxyKubelets.ByIP = false
				ctx.Config.Sync.FromHost.Nodes.Selector.All = true
				ctx.Config.Sync.FromHost.Nodes.ClearImageStatus = true
				syncCtx, syncerSvc := newFakeSyncer(t, ctx)
				_, err := syncerSvc.Sync(syncCtx, baseNode.DeepCopy(), baseVNode.DeepCopy())
				assert.NilError(t, err)
			},
		},
	})
}

func TestClearNodeImageDisabled(t *testing.T) {
	editedNode := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: baseName.Name,
			Annotations: map[string]string{
				translate.ManagedLabelsAnnotation: "test",
			},
			Labels: map[string]string{
				"test": "true",
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
			Images: []corev1.ContainerImage{
				{
					Names: []string{"ghcr.io/jetpack/calico"},
				},
			},
		},
	}

	syncertesting.RunTests(t, []*syncertesting.SyncTest{
		{
			Name:                 "Clear Node Images Disabled -- Synced Node Should have images in status.images",
			InitialPhysicalState: []runtime.Object{baseNode.DeepCopy()},
			InitialVirtualState:  []runtime.Object{baseVNode.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Node"): {editedNode.DeepCopy()},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				ctx.Config.Networking.Advanced.ProxyKubelets.ByIP = false
				ctx.Config.Sync.FromHost.Nodes.Selector.All = true
				syncCtx, syncerSvc := newFakeSyncer(t, ctx)
				_, err := syncerSvc.Sync(syncCtx, baseNode.DeepCopy(), baseVNode.DeepCopy())
				assert.NilError(t, err)
			},
		},
	})
}
