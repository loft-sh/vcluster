package nodes

import (
	"context"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/nodes/nodeservice"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"

	"github.com/loft-sh/vcluster/pkg/constants"
	generictesting "github.com/loft-sh/vcluster/pkg/controllers/generic/testing"
	"github.com/loft-sh/vcluster/pkg/util/locks"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

func newFakeSyncer(ctx context.Context, lockFactory locks.LockFactory, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient) (*syncer, error) {
	err := vClient.IndexField(ctx, &corev1.Pod{}, constants.IndexByAssigned, func(rawObj client.Object) []string {
		pod := rawObj.(*corev1.Pod)
		return []string{pod.Spec.NodeName}
	})
	if err != nil {
		return nil, err
	}

	return &syncer{
		sharedNodesMutex:    lockFactory.GetLock("ingress-controller"),
		nodeServiceProvider: &fakeNodeServiceProvider{},
		virtualClient:       vClient,
		podCache:            pClient,
		localClient:         pClient,
		scheme:              testingutil.NewScheme(),
		useFakeKubelets:     true,
	}, nil
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
	lockFactory := locks.NewDefaultLockFactory()

	generictesting.RunTests(t, []*generictesting.SyncTest{
		{
			Name:                "Create backward",
			InitialVirtualState: []runtime.Object{basePod},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Node"): {baseNode},
				corev1.SchemeGroupVersion.WithKind("Pod"):  {basePod},
			},
			Sync: func(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient, scheme *runtime.Scheme, log loghelper.Logger) {
				syncer, err := newFakeSyncer(ctx, lockFactory, pClient, vClient)
				if err != nil {
					t.Fatal(err)
				}

				_, err = syncer.Backward(ctx, baseNode, log)
				if err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			Name:                "Create backward not needed",
			InitialVirtualState: []runtime.Object{},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Node"): {},
				corev1.SchemeGroupVersion.WithKind("Pod"):  {},
			},
			Sync: func(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient, scheme *runtime.Scheme, log loghelper.Logger) {
				syncer, err := newFakeSyncer(ctx, lockFactory, pClient, vClient)
				if err != nil {
					t.Fatal(err)
				}

				_, err = syncer.Backward(ctx, baseNode, log)
				if err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			Name:                "Update backward",
			InitialVirtualState: []runtime.Object{basePod, baseNode},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Node"): {editedNode},
				corev1.SchemeGroupVersion.WithKind("Pod"):  {basePod},
			},
			Sync: func(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient, scheme *runtime.Scheme, log loghelper.Logger) {
				syncer, err := newFakeSyncer(ctx, lockFactory, pClient, vClient)
				if err != nil {
					t.Fatal(err)
				}

				_, err = syncer.Update(ctx, editedNode, baseNode, log)
				if err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			Name:                "Update backward no change",
			InitialVirtualState: []runtime.Object{basePod, baseNode},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Node"): {baseNode},
				corev1.SchemeGroupVersion.WithKind("Pod"):  {basePod},
			},
			Sync: func(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient, scheme *runtime.Scheme, log loghelper.Logger) {
				syncer, err := newFakeSyncer(ctx, lockFactory, pClient, vClient)
				if err != nil {
					t.Fatal(err)
				}

				_, err = syncer.Update(ctx, baseNode, baseVNode, log)
				if err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			Name:                "Delete backward",
			InitialVirtualState: []runtime.Object{baseNode},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Node"): {},
				corev1.SchemeGroupVersion.WithKind("Pod"):  {},
			},
			Sync: func(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient, scheme *runtime.Scheme, log loghelper.Logger) {
				syncer, err := newFakeSyncer(ctx, lockFactory, pClient, vClient)
				if err != nil {
					t.Fatal(err)
				}

				_, err = syncer.Update(ctx, baseNode, baseNode, log)
				if err != nil {
					t.Fatal(err)
				}
			},
		},
	})
}
