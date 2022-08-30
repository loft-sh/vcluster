package nodes

import (
	"context"
	"fmt"

	"github.com/loft-sh/vcluster/pkg/controllers/resources/nodes/nodeservice"
	podtranslate "github.com/loft-sh/vcluster/pkg/controllers/resources/pods/translate"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/util/random"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	// FakeNodesVersion is the default version that will be used for fake nodes
	FakeNodesVersion = "v1.19.1"
)

func NewFakeSyncer(ctx *synccontext.RegisterContext, nodeService nodeservice.NodeServiceProvider) (syncer.Object, error) {
	return &fakeNodeSyncer{
		nodeServiceProvider: nodeService,
	}, nil
}

type fakeNodeSyncer struct {
	nodeServiceProvider nodeservice.NodeServiceProvider
}

func (r *fakeNodeSyncer) Resource() client.Object {
	return &corev1.Node{}
}

func (r *fakeNodeSyncer) Name() string {
	return "fake-node"
}

var _ syncer.IndicesRegisterer = &fakeNodeSyncer{}

func (r *fakeNodeSyncer) RegisterIndices(ctx *synccontext.RegisterContext) error {
	return registerIndices(ctx)
}

var _ syncer.ControllerModifier = &fakeNodeSyncer{}

func (r *fakeNodeSyncer) ModifyController(ctx *synccontext.RegisterContext, builder *builder.Builder) (*builder.Builder, error) {
	return modifyController(ctx, r.nodeServiceProvider, builder)
}

var _ syncer.FakeSyncer = &fakeNodeSyncer{}

func (r *fakeNodeSyncer) FakeSyncUp(ctx *synccontext.SyncContext, name types.NamespacedName) (ctrl.Result, error) {
	needed, err := r.nodeNeeded(ctx, name.Name)
	if err != nil {
		return ctrl.Result{}, err
	} else if !needed {
		return ctrl.Result{}, nil
	}

	ctx.Log.Infof("Create fake node %s", name.Name)
	return ctrl.Result{}, CreateFakeNode(ctx.Context, r.nodeServiceProvider, ctx.VirtualClient, name)
}

func (r *fakeNodeSyncer) FakeSync(ctx *synccontext.SyncContext, vObj client.Object) (ctrl.Result, error) {
	node, ok := vObj.(*corev1.Node)
	if !ok || node == nil {
		return ctrl.Result{}, fmt.Errorf("%#v is not a node", vObj)
	}

	needed, err := r.nodeNeeded(ctx, node.Name)
	if err != nil {
		return ctrl.Result{}, err
	} else if needed {
		return ctrl.Result{}, nil
	}

	ctx.Log.Infof("Delete fake node %s as it is not needed anymore", vObj.GetName())
	return ctrl.Result{}, ctx.VirtualClient.Delete(ctx.Context, vObj)
}

func (r *fakeNodeSyncer) nodeNeeded(ctx *synccontext.SyncContext, nodeName string) (bool, error) {
	return isNodeNeededByPod(ctx.Context, ctx.VirtualClient, ctx.PhysicalClient, nodeName)
}

// this is not a real guid, but it doesn't really matter because it should just look right and not be an actual guid
func newGUID() string {
	return random.RandomString(8) + "-" + random.RandomString(4) + "-" + random.RandomString(4) + "-" + random.RandomString(4) + "-" + random.RandomString(12)
}

func CreateFakeNode(ctx context.Context, nodeServiceProvider nodeservice.NodeServiceProvider, virtualClient client.Client, name types.NamespacedName) error {
	nodeServiceProvider.Lock()
	defer nodeServiceProvider.Unlock()

	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: name.Name,
			Labels: map[string]string{
				"vcluster.loft.sh/fake-node": "true",
				"beta.kubernetes.io/arch":    "amd64",
				"beta.kubernetes.io/os":      "linux",
				"kubernetes.io/arch":         "amd64",
				"kubernetes.io/hostname":     translate.SafeConcatName("fake", name.Name),
				"kubernetes.io/os":           "linux",
			},
			Annotations: map[string]string{
				"node.alpha.kubernetes.io/ttl":                           "0",
				"volumes.kubernetes.io/controller-managed-attach-detach": "false",
			},
		},
	}

	err := virtualClient.Create(ctx, node)
	if err != nil {
		return err
	}

	nodeIP, err := nodeServiceProvider.GetNodeIP(ctx, name)
	if err != nil {
		return errors.Wrap(err, "create fake node ip")
	}

	orig := node.DeepCopy()
	node.Status = corev1.NodeStatus{
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
				LastHeartbeatTime:  metav1.Now(),
				LastTransitionTime: metav1.Now(),
				Message:            "kubelet has sufficient memory available",
				Reason:             "KubeletHasSufficientMemory",
				Status:             "False",
				Type:               corev1.NodeMemoryPressure,
			},
			{
				LastHeartbeatTime:  metav1.Now(),
				LastTransitionTime: metav1.Now(),
				Message:            "kubelet has no disk pressure",
				Reason:             "KubeletHasNoDiskPressure",
				Status:             "False",
				Type:               corev1.NodeDiskPressure,
			},
			{
				LastHeartbeatTime:  metav1.Now(),
				LastTransitionTime: metav1.Now(),
				Message:            "kubelet has sufficient PID available",
				Reason:             "KubeletHasSufficientPID",
				Status:             "False",
				Type:               corev1.NodePIDPressure,
			},
			{
				LastHeartbeatTime:  metav1.Now(),
				LastTransitionTime: metav1.Now(),
				Message:            "kubelet is posting ready status",
				Reason:             "KubeletReady",
				Status:             "True",
				Type:               corev1.NodeReady,
			},
		},
		Addresses: []corev1.NodeAddress{
			{
				Address: nodeIP,
				Type:    corev1.NodeInternalIP,
			},
		},
		DaemonEndpoints: corev1.NodeDaemonEndpoints{
			KubeletEndpoint: corev1.DaemonEndpoint{
				Port: nodeservice.KubeletPort,
			},
		},
		NodeInfo: corev1.NodeSystemInfo{
			Architecture:            "amd64",
			BootID:                  newGUID(),
			ContainerRuntimeVersion: "docker://19.3.12",
			KernelVersion:           "4.19.76-fakelinux",
			KubeProxyVersion:        FakeNodesVersion,
			KubeletVersion:          FakeNodesVersion,
			MachineID:               newGUID(),
			SystemUUID:              newGUID(),
			OperatingSystem:         "linux",
			OSImage:                 "Fake Kubernetes Image",
		},
		Images: []corev1.ContainerImage{},
	}
	err = virtualClient.Status().Patch(ctx, node, client.MergeFrom(orig))
	if err != nil {
		return err
	}

	// remove not ready taints
	orig = node.DeepCopy()
	node.Spec.Taints = []corev1.Taint{}
	err = virtualClient.Patch(ctx, node, client.MergeFrom(orig))
	if err != nil {
		return err
	}

	return nil
}

// Filter away  virtual DaemonSet Pods using OwnerReferences to enable scale down
func filterOutVirtualDaemonSets(pl *corev1.PodList) []corev1.Pod {
	var podsNoDaemonSets []corev1.Pod

	for _, item := range pl.Items {
		var isDaemonSet bool

		// ensure pod has owner references
		if len(item.OwnerReferences) > 0 {
			// cover edge case with multiple owner refs
			for _, ownerRef := range item.OwnerReferences {
				if ownerRef.APIVersion == "apps/v1" && ownerRef.Kind == "DaemonSet" {
					isDaemonSet = true
				}
			}
		}
		if !isDaemonSet {
			podsNoDaemonSets = append(podsNoDaemonSets, item)
		}
	}

	return podsNoDaemonSets
}

// Filter away physical DaemonSet Pods using annotations to enable scale down
func filterOutPhysicalDaemonSets(pl *corev1.PodList) []corev1.Pod {
	var podsNoDaemonSets []corev1.Pod

	for _, item := range pl.Items {
		if item.Annotations == nil || item.Annotations[podtranslate.OwnerSetKind] != "DaemonSet" {
			podsNoDaemonSets = append(podsNoDaemonSets, item)
		}
	}
	return podsNoDaemonSets
}
