package nodes

import (
	"context"
	goruntime "runtime"
	"testing"

	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertesting "github.com/loft-sh/vcluster/pkg/syncer/testing"
	syncer "github.com/loft-sh/vcluster/pkg/syncer/types"
	"gotest.tools/assert"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/loft-sh/vcluster/pkg/constants"
	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

func newFakeFakeSyncer(t *testing.T, ctx *synccontext.RegisterContext) (*synccontext.SyncContext, *fakeNodeSyncer) {
	// we need that index here as well otherwise we wouldn't find the related pod
	err := ctx.VirtualManager.GetFieldIndexer().IndexField(ctx, &corev1.Pod{}, constants.IndexByAssigned, func(rawObj client.Object) []string {
		pod := rawObj.(*corev1.Pod)
		return []string{pod.Spec.NodeName}
	})
	assert.NilError(t, err)

	syncContext, object := syncertesting.FakeStartSyncer(t, ctx, func(ctx *synccontext.RegisterContext) (syncer.Object, error) {
		return NewFakeSyncer(ctx, &fakeNodeServiceProvider{})
	})
	return syncContext, object.(*fakeNodeSyncer)
}

type fakeNodeServiceProvider struct{}

func (f *fakeNodeServiceProvider) Start(context.Context) {}
func (f *fakeNodeServiceProvider) Lock()                 {}
func (f *fakeNodeServiceProvider) Unlock()               {}
func (f *fakeNodeServiceProvider) GetNodeIP(context.Context, string) (string, error) {
	return "127.0.0.1", nil
}

func TestFakeSync(t *testing.T) {
	fakeGUID := newGUID()
	now := metav1.Now()
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
			Labels: map[string]string{
				"vcluster.loft.sh/fake-node": "true",
				"beta.kubernetes.io/arch":    goruntime.GOARCH,
				"beta.kubernetes.io/os":      "linux",
				"kubernetes.io/arch":         goruntime.GOARCH,
				"kubernetes.io/hostname":     "fake-" + baseName.Name,
				"kubernetes.io/os":           "linux",
			},
			Annotations: map[string]string{
				"node.alpha.kubernetes.io/ttl":                           "0",
				"volumes.kubernetes.io/controller-managed-attach-detach": "false",
			},
		},
		Status: corev1.NodeStatus{
			Capacity: corev1.ResourceList{
				corev1.ResourceCPU:                     resource.MustParse("16"),
				corev1.ResourceMemory:                  resource.MustParse("32Gi"),
				corev1.ResourceEphemeralStorage:        resource.MustParse("100Gi"),
				corev1.ResourceHugePagesPrefix + "1Gi": resource.MustParse("0"),
				corev1.ResourceHugePagesPrefix + "2Mi": resource.MustParse("0"),
				corev1.ResourcePods:                    resource.MustParse("110"),
			},
			Allocatable: corev1.ResourceList{
				corev1.ResourceCPU:                     resource.MustParse("16"),
				corev1.ResourceMemory:                  resource.MustParse("32Gi"),
				corev1.ResourceEphemeralStorage:        resource.MustParse("100Gi"),
				corev1.ResourceHugePagesPrefix + "1Gi": resource.MustParse("0"),
				corev1.ResourceHugePagesPrefix + "2Mi": resource.MustParse("0"),
				corev1.ResourcePods:                    resource.MustParse("110"),
			},
			Conditions: []corev1.NodeCondition{
				{
					Message: "kubelet has sufficient memory available",
					Reason:  "KubeletHasSufficientMemory",
					Status:  "False",
					Type:    corev1.NodeMemoryPressure,
				},
				{
					Message: "kubelet has no disk pressure",
					Reason:  "KubeletHasNoDiskPressure",
					Status:  "False",
					Type:    corev1.NodeDiskPressure,
				},
				{
					Message: "kubelet has sufficient PID available",
					Reason:  "KubeletHasSufficientPID",
					Status:  "False",
					Type:    corev1.NodePIDPressure,
				},
				{
					Message: "kubelet is posting ready status",
					Reason:  "KubeletReady",
					Status:  "True",
					Type:    corev1.NodeReady,
				},
			},
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
				Architecture:            "amd64",
				ContainerRuntimeVersion: "docker://19.3.12",
				KernelVersion:           "4.19.76-fakelinux",
				KubeProxyVersion:        "v1.16.6-beta.0", //nolint:staticcheck //deprecated, but we should continue to use it until the api removes it
				KubeletVersion:          "v1.16.6-beta.0",
				OperatingSystem:         "linux",
				OSImage:                 "Fake Kubernetes Image",
			},
			Images: []corev1.ContainerImage{},
		},
	}

	syncertesting.RunTests(t, []*syncertesting.SyncTest{
		{
			Name:                "Create test",
			InitialVirtualState: []runtime.Object{basePod},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Node"): {baseNode},
				corev1.SchemeGroupVersion.WithKind("Pod"):  {basePod},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				ctx.Config.Networking.Advanced.ProxyKubelets.ByIP = false
				syncContext, syncer := newFakeFakeSyncer(t, ctx)
				_, err := syncer.FakeSyncToVirtual(syncContext, baseName)
				assert.NilError(t, err)
			},
			Compare: func(obj1 runtime.Object, obj2 runtime.Object) bool {
				node1, ok1 := obj1.(*corev1.Node)
				node2, ok2 := obj2.(*corev1.Node)
				if ok1 && ok2 {
					for _, node := range []*corev1.Node{node1, node2} {
						for i := range node.Status.Conditions {
							node.Status.Conditions[i].LastHeartbeatTime = now
							node.Status.Conditions[i].LastTransitionTime = now
						}
						node.Status.NodeInfo.BootID = fakeGUID
						node.Status.NodeInfo.MachineID = fakeGUID
						node.Status.NodeInfo.SystemUUID = fakeGUID
						node.Status.NodeInfo.KernelVersion = fakeGUID
						node.Status.NodeInfo.KubeProxyVersion = fakeGUID //nolint:staticcheck //deprecated, but we should continue to use it until the api removes it
						node.Status.NodeInfo.KubeletVersion = fakeGUID
					}

					node1.Status.Images = node2.Status.Images
					obj1 = node1
					obj2 = node2
				}

				assert.DeepEqual(t, obj1, obj2)
				return apiequality.Semantic.DeepEqual(obj1, obj2)
			},
		},
		{
			Name:                "Delete test",
			InitialVirtualState: []runtime.Object{baseNode},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Node"): {},
				corev1.SchemeGroupVersion.WithKind("Pod"):  {},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncContext, syncer := newFakeFakeSyncer(t, ctx)

				_, err := syncer.FakeSync(syncContext, baseNode)
				assert.NilError(t, err)
			},
		},
	})
}
