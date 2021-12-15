package nodes

import (
	"context"
	"fmt"
	"sync"

	"github.com/loft-sh/vcluster/pkg/controllers/resources/generic"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/nodes/nodeservice"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	context2 "github.com/loft-sh/vcluster/cmd/vcluster/context"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/util/random"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	// FakeNodesVersion is the default version that will be used for fake nodes
	FakeNodesVersion = "v1.19.1"
)

func RegisterFakeSyncer(ctx *context2.ControllerContext) error {
	return generic.RegisterFakeSyncerWithOptions(ctx, "fake-node", &fakeSyncer{
		sharedNodesMutex:    ctx.LockFactory.GetLock("nodes-controller"),
		nodeServiceProvider: ctx.NodeServiceProvider,
		virtualClient:       ctx.VirtualManager.GetClient(),
	}, &generic.SyncerOptions{
		ModifyController: func(builder *builder.Builder) *builder.Builder {
			return builder.Watches(&source.Kind{Type: &corev1.Pod{}}, handler.EnqueueRequestsFromMapFunc(func(object client.Object) []reconcile.Request {
				pod, ok := object.(*corev1.Pod)
				if !ok || pod == nil {
					return []reconcile.Request{}
				}

				return []reconcile.Request{
					{
						NamespacedName: types.NamespacedName{
							Name: pod.Spec.NodeName,
						},
					},
				}
			}))
		},
	})
}

type fakeSyncer struct {
	sharedNodesMutex    sync.Locker
	virtualClient       client.Client
	nodeServiceProvider nodeservice.NodeServiceProvider
}

func (r *fakeSyncer) New() client.Object {
	return &corev1.Node{}
}

func (r *fakeSyncer) ReconcileStart(ctx context.Context, req ctrl.Request) (bool, error) {
	r.sharedNodesMutex.Lock()
	return false, nil
}

func (r *fakeSyncer) ReconcileEnd() {
	r.sharedNodesMutex.Unlock()
}

func (r *fakeSyncer) Create(ctx context.Context, name types.NamespacedName, log loghelper.Logger) (ctrl.Result, error) {
	needed, err := r.nodeNeeded(ctx, name.Name)
	if err != nil {
		return ctrl.Result{}, err
	} else if !needed {
		return ctrl.Result{}, nil
	}

	log.Infof("Create fake node %s", name.Name)
	return ctrl.Result{}, CreateFakeNode(ctx, r.nodeServiceProvider, r.virtualClient, name)
}

func (r *fakeSyncer) Update(ctx context.Context, vObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
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

	log.Infof("Delete fake node %s as it is not needed anymore", vObj.GetName())
	return ctrl.Result{}, r.virtualClient.Delete(ctx, vObj)
}

func (r *fakeSyncer) nodeNeeded(ctx context.Context, nodeName string) (bool, error) {
	podList := &corev1.PodList{}
	err := r.virtualClient.List(ctx, podList, client.MatchingFields{constants.IndexByAssigned: nodeName})
	if err != nil {
		return false, err
	}

	return len(podList.Items) > 0, nil
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
				"kubernetes.io/hostname":     "fake-" + name.Name,
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
