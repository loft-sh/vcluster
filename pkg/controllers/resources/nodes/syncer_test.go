package nodes

import (
	"github.com/loft-sh/vcluster/pkg/controllers/resources/nodes/nodeservice"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"gotest.tools/assert"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"

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
}
