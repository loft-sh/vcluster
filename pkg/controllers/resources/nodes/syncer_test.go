package nodes

import (
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"testing"

	"github.com/loft-sh/vcluster/pkg/controllers/resources/nodes/nodeservice"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"gotest.tools/assert"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/loft-sh/vcluster/pkg/constants"
	generictesting "github.com/loft-sh/vcluster/pkg/controllers/syncer/testing"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

func newFakeSyncer(t *testing.T, ctx *synccontext.RegisterContext) (*synccontext.SyncContext, *nodeSyncer) {
	// we need that index here as well otherwise we wouldn't find the related pod
	err := ctx.VirtualManager.GetFieldIndexer().IndexField(ctx.Context, &corev1.Pod{}, constants.IndexByAssigned, func(rawObj client.Object) []string {
		pod := rawObj.(*corev1.Pod)
		return []string{pod.Spec.NodeName}
	})
	assert.NilError(t, err)

	syncContext, object := generictesting.FakeStartSyncer(t, ctx, func(ctx *synccontext.RegisterContext) (syncer.Object, error) {
		return NewSyncer(ctx, &fakeNodeServiceProvider{})
	})
	return syncContext, object.(*nodeSyncer)
}

func TestSync(t *testing.T) {
	baseName := types.NamespacedName{
		Name: "mynode",
	}
	basePod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "mypod",
		},
		Spec: corev1.PodSpec{
			NodeName: baseName.Name,
		},
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
	baseVNode := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: baseName.Name,
		},
		Status: corev1.NodeStatus{
			Addresses: []corev1.NodeAddress{
				{
					Address: "127.0.0.1",
					Type:    corev1.NodeInternalIP,
				},
			},
			DaemonEndpoints: corev1.NodeDaemonEndpoints{
				KubeletEndpoint: corev1.DaemonEndpoint{
					Port: nodeservice.KubeletPort,
				},
			},
		},
	}
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
					Address: "127.0.0.1",
					Type:    corev1.NodeInternalIP,
				},
			},
			DaemonEndpoints: corev1.NodeDaemonEndpoints{
				KubeletEndpoint: corev1.DaemonEndpoint{
					Port: nodeservice.KubeletPort,
				},
			},
			NodeInfo: corev1.NodeSystemInfo{
				Architecture: "amd64",
			},
		},
	}

	generictesting.RunTests(t, []*generictesting.SyncTest{
		{
			Name:                "Create backward",
			InitialVirtualState: []runtime.Object{basePod},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Node"): {baseNode},
				corev1.SchemeGroupVersion.WithKind("Pod"):  {basePod},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := newFakeSyncer(t, ctx)
				_, err := syncer.SyncUp(syncCtx, baseNode)
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
				_, err := syncer.SyncUp(syncCtx, baseNode)
				assert.NilError(t, err)
			},
		},
		{
			Name:                "Update backward",
			InitialVirtualState: []runtime.Object{basePod, baseNode},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Node"): {editedNode},
				corev1.SchemeGroupVersion.WithKind("Pod"):  {basePod},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := newFakeSyncer(t, ctx)
				_, err := syncer.Sync(syncCtx, editedNode, baseNode)
				assert.NilError(t, err)

				err = ctx.VirtualManager.GetClient().Get(ctx.Context, types.NamespacedName{Name: baseNode.Name}, baseNode)
				assert.NilError(t, err)

				_, err = syncer.Sync(syncCtx, editedNode, baseNode)
				assert.NilError(t, err)
			},
		},
		{
			Name:                "Update backward no change",
			InitialVirtualState: []runtime.Object{basePod, baseNode},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Node"): {baseNode},
				corev1.SchemeGroupVersion.WithKind("Pod"):  {basePod},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := newFakeSyncer(t, ctx)
				_, err := syncer.Sync(syncCtx, baseNode, baseVNode)
				assert.NilError(t, err)
			},
		},
		{
			Name:                "Delete backward",
			InitialVirtualState: []runtime.Object{baseNode},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Node"): {},
				corev1.SchemeGroupVersion.WithKind("Pod"):  {},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := newFakeSyncer(t, ctx)
				_, err := syncer.Sync(syncCtx, baseNode, baseNode)
				assert.NilError(t, err)
			},
		},
	})

	baseNode = &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: baseName.Name,
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
					Address: "127.0.0.1",
					Type:    corev1.NodeInternalIP,
				},
			},
			DaemonEndpoints: corev1.NodeDaemonEndpoints{
				KubeletEndpoint: corev1.DaemonEndpoint{
					Port: nodeservice.KubeletPort,
				},
			},
			NodeInfo: corev1.NodeSystemInfo{
				Architecture: "amd64",
			},
		},
	}

	editedNode = &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: baseName.Name,
		},
		Spec: corev1.NodeSpec{
			Taints: nil,
		},
		Status: corev1.NodeStatus{
			Addresses: []corev1.NodeAddress{
				{
					Address: "127.0.0.1",
					Type:    corev1.NodeInternalIP,
				},
			},
			DaemonEndpoints: corev1.NodeDaemonEndpoints{
				KubeletEndpoint: corev1.DaemonEndpoint{
					Port: nodeservice.KubeletPort,
				},
			},
			NodeInfo: corev1.NodeSystemInfo{
				Architecture: "amd64",
			},
		},
	}

	generictesting.RunTests(t, []*generictesting.SyncTest{
		{
			Name:                 "Taint matching Enforced Toleration",
			InitialPhysicalState: []runtime.Object{basePod, baseNode},
			InitialVirtualState:  []runtime.Object{basePod, baseNode},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Node"): {editedNode},
				corev1.SchemeGroupVersion.WithKind("Pod"):  {basePod},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				ctx.Options.Tolerations = []string{"key1=value1:NoSchedule"}
				syncCtx, syncer := newFakeSyncer(t, ctx)
				_, err := syncer.Sync(syncCtx, baseNode, baseNode)
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Taint not matching Enforced Toleration",
			InitialPhysicalState: []runtime.Object{basePod, baseNode},
			InitialVirtualState:  []runtime.Object{basePod, baseNode},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Node"): {baseNode},
				corev1.SchemeGroupVersion.WithKind("Pod"):  {basePod},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				ctx.Options.Tolerations = []string{"key2=value2:NoSchedule"}
				syncCtx, syncer := newFakeSyncer(t, ctx)
				_, err := syncer.Sync(syncCtx, baseNode, baseNode)
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Taint matching Enforced Toleration - special case of empty key with Exists operator",
			InitialPhysicalState: []runtime.Object{basePod, baseNode},
			InitialVirtualState:  []runtime.Object{basePod, baseNode},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Node"): {editedNode},
				corev1.SchemeGroupVersion.WithKind("Pod"):  {basePod},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				ctx.Options.Tolerations = []string{":NoSchedule op=Exists"}
				syncCtx, syncer := newFakeSyncer(t, ctx)
				_, err := syncer.Sync(syncCtx, baseNode, baseNode)
				assert.NilError(t, err)
			},
		},
	})

	baseName = types.NamespacedName{
		Name: "mynode",
	}
	basePod = &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "mypod",
		},
		Spec: corev1.PodSpec{
			NodeName: baseName.Name,
		},
	}
	baseNode = &corev1.Node{
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
	baseVNode = &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: baseName.Name,
		},
		Status: corev1.NodeStatus{
			Addresses: []corev1.NodeAddress{
				{
					Address: "127.0.0.1",
					Type:    corev1.NodeInternalIP,
				},
			},
			DaemonEndpoints: corev1.NodeDaemonEndpoints{
				KubeletEndpoint: corev1.DaemonEndpoint{
					Port: nodeservice.KubeletPort,
				},
			},
		},
	}
	editedNode = &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: baseName.Name,
			Labels: map[string]string{
				"test": "true",
			},
		},
		Status: corev1.NodeStatus{
			Addresses: []corev1.NodeAddress{
				{
					Address: "127.0.0.1",
					Type:    corev1.NodeInternalIP,
				},
			},
			DaemonEndpoints: corev1.NodeDaemonEndpoints{
				KubeletEndpoint: corev1.DaemonEndpoint{
					Port: nodeservice.KubeletPort,
				},
			},
			NodeInfo: corev1.NodeSystemInfo{
				Architecture: "amd64",
			},
		},
	}

	generictesting.RunTests(t, []*generictesting.SyncTest{
		{
			Name:                 "Label Matched and enforceNodeSelector false - expect node to be synced from NodeSelector",
			InitialPhysicalState: []runtime.Object{baseNode},
			InitialVirtualState:  []runtime.Object{baseVNode},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Node"): {editedNode},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				ctx.Options.EnforceNodeSelector = false
				req, _ := labels.NewRequirement("test", selection.Equals, []string{"true"})
				sel := labels.NewSelector().Add(*req)
				ctx.Options.NodeSelector = sel.String()
				syncCtx, syncer := newFakeSyncer(t, ctx)
				_, err := syncer.Sync(syncCtx, baseNode, baseNode)
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Label Not Matched and enforceNodeSelector false - expect node to be synced from pod needs",
			InitialPhysicalState: []runtime.Object{basePod, baseNode},
			InitialVirtualState:  []runtime.Object{basePod, baseVNode},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Node"): {editedNode},
				corev1.SchemeGroupVersion.WithKind("Pod"):  {basePod},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				ctx.Options.EnforceNodeSelector = false
				req, _ := labels.NewRequirement("test", selection.NotEquals, []string{"true"})
				sel := labels.NewSelector().Add(*req)
				ctx.Options.NodeSelector = sel.String()
				syncCtx, syncer := newFakeSyncer(t, ctx)
				_, err := syncer.Sync(syncCtx, baseNode, baseNode)
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "No NodeSelector LabelSet and enforceNodeSelector false - expect node to be synced from pod needs",
			InitialPhysicalState: []runtime.Object{basePod, baseNode},
			InitialVirtualState:  []runtime.Object{basePod, baseVNode},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Node"): {editedNode},
				corev1.SchemeGroupVersion.WithKind("Pod"):  {basePod},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				ctx.Options.EnforceNodeSelector = false
				syncCtx, syncer := newFakeSyncer(t, ctx)
				_, err := syncer.Sync(syncCtx, baseNode, baseNode)
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Label Not Matched and enforceNodeSelector true - expect node not to be synced",
			InitialPhysicalState: []runtime.Object{basePod, baseNode},
			InitialVirtualState:  []runtime.Object{baseVNode},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{},
			Sync: func(ctx *synccontext.RegisterContext) {
				req, _ := labels.NewRequirement("test", selection.NotEquals, []string{"true"})
				sel := labels.NewSelector().Add(*req)
				ctx.Options.NodeSelector = sel.String()
				ctx.Options.EnforceNodeSelector = true
				syncCtx, syncer := newFakeSyncer(t, ctx)
				_, err := syncer.Sync(syncCtx, baseNode, baseNode)
				assert.NilError(t, err)
			},
		},
	})

	baseName = types.NamespacedName{
		Name: "mynode",
	}

	baseNode = &corev1.Node{
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
	baseVNode = &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: baseName.Name,
		},
		Status: corev1.NodeStatus{
			Addresses: []corev1.NodeAddress{
				{
					Address: "127.0.0.1",
					Type:    corev1.NodeInternalIP,
				},
			},
			DaemonEndpoints: corev1.NodeDaemonEndpoints{
				KubeletEndpoint: corev1.DaemonEndpoint{
					Port: nodeservice.KubeletPort,
				},
			},
		},
	}
	editedNode = &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: baseName.Name,
			Labels: map[string]string{
				"test": "true",
			},
		},
		Status: corev1.NodeStatus{
			Addresses: []corev1.NodeAddress{
				{
					Address: "127.0.0.1",
					Type:    corev1.NodeInternalIP,
				},
			},
			DaemonEndpoints: corev1.NodeDaemonEndpoints{
				KubeletEndpoint: corev1.DaemonEndpoint{
					Port: nodeservice.KubeletPort,
				},
			},
			NodeInfo: corev1.NodeSystemInfo{
				Architecture: "amd64",
			},
			Images: []corev1.ContainerImage{},
		},
	}

	generictesting.RunTests(t, []*generictesting.SyncTest{
		{
			Name:                 "Clear Node Images Enabled -- Synced Node Should have no images in status.images",
			InitialPhysicalState: []runtime.Object{baseNode},
			InitialVirtualState:  []runtime.Object{baseVNode},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Node"): {editedNode},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				ctx.Options.SyncAllNodes = true
				ctx.Options.ClearNodeImages = true
				syncCtx, syncerSvc := newFakeSyncer(t, ctx)
				_, err := syncerSvc.Sync(syncCtx, baseNode, baseNode)
				assert.NilError(t, err)
			},
		},
	})

	editedNode = &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: baseName.Name,
			Labels: map[string]string{
				"test": "true",
			},
		},
		Status: corev1.NodeStatus{
			Addresses: []corev1.NodeAddress{
				{
					Address: "127.0.0.1",
					Type:    corev1.NodeInternalIP,
				},
			},
			DaemonEndpoints: corev1.NodeDaemonEndpoints{
				KubeletEndpoint: corev1.DaemonEndpoint{
					Port: nodeservice.KubeletPort,
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

	generictesting.RunTests(t, []*generictesting.SyncTest{
		{
			Name:                 "Clear Node Images Disabled -- Synced Node Should have images in status.images",
			InitialPhysicalState: []runtime.Object{baseNode},
			InitialVirtualState:  []runtime.Object{baseVNode},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Node"): {editedNode},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				ctx.Options.SyncAllNodes = true
				syncCtx, syncerSvc := newFakeSyncer(t, ctx)
				_, err := syncerSvc.Sync(syncCtx, baseNode, baseNode)
				assert.NilError(t, err)
			},
		},
	})
}
