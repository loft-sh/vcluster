package nodeservice

import (
	"context"
	"fmt"
	"github.com/loft-sh/vcluster/pkg/util/clienthelper"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sync"
	"time"
)

var (
	// ServiceClusterLabel identifies to which vcluster the node service belongs if there are multiple in one namespace
	ServiceClusterLabel = "vcluster.loft.sh/belongs-to"
	// ServiceNodeLabel specifies which node this service represents
	ServiceNodeLabel = "vcluster.loft.sh/node"
	// KubeletPort is the port we pretend the kubelet is running under
	KubeletPort = int32(10250)
	// KubeletTargetPort is the port vcluster will run under
	KubeletTargetPort = 8443
)

type NodeServiceProvider interface {
	sync.Locker
	// Start starts the node service garbage collector
	Start(ctx context.Context)
	// GetNodeIP returns a new fake node ip
	GetNodeIP(ctx context.Context, name types.NamespacedName) (string, error)
}

func NewNodeServiceProvider(currentNamespace string, currentNamespaceClient client.Client, virtualClient client.Client, uncachedVirtualClient client.Client) NodeServiceProvider {
	return &nodeServiceProvider{
		currentNamespace:       currentNamespace,
		currentNamespaceClient: currentNamespaceClient,
		virtualClient:          virtualClient,
		uncachedVirtualClient:  uncachedVirtualClient,
	}
}

type nodeServiceProvider struct {
	currentNamespace       string
	currentNamespaceClient client.Client

	virtualClient         client.Client
	uncachedVirtualClient client.Client

	serviceMutex sync.Mutex
}

func (n *nodeServiceProvider) Start(ctx context.Context) {
	wait.Until(func() {
		err := n.cleanupNodeServices(ctx)
		if err != nil {
			klog.Errorf("error cleaning up node services: %v", err)
		}
	}, time.Second*4, ctx.Done())
}

func (n *nodeServiceProvider) cleanupNodeServices(ctx context.Context) error {
	n.serviceMutex.Lock()
	defer n.serviceMutex.Unlock()

	serviceList := &corev1.ServiceList{}
	err := n.currentNamespaceClient.List(ctx, serviceList, client.InNamespace(n.currentNamespace), client.MatchingLabels{
		ServiceClusterLabel: translate.Suffix,
	})
	if err != nil {
		return errors.Wrap(err, "list services")
	}

	errors := []error{}
	for _, s := range serviceList.Items {
		exist := false
		if s.Labels[ServiceNodeLabel] != "" {
			// check if node still exists
			err = n.virtualClient.Get(ctx, client.ObjectKey{Name: s.Labels[ServiceNodeLabel]}, &corev1.Node{})
			if err != nil {
				if kerrors.IsNotFound(err) == false {
					klog.Infof("error retrieving node %s: %v", s.Labels[ServiceNodeLabel], err)
					continue
				}

				// make sure node really does not exist
				err = n.uncachedVirtualClient.Get(ctx, client.ObjectKey{Name: s.Labels[ServiceNodeLabel]}, &corev1.Node{})
				if err == nil {
					exist = true
				}
			} else {
				exist = true
			}
		}

		if exist == false {
			klog.Infof("Cleaning up kubelet service for node %s", s.Labels[ServiceNodeLabel])
			err = n.currentNamespaceClient.Delete(ctx, &s)
			if err != nil {
				errors = append(errors, err)
			}
		}
	}

	return utilerrors.NewAggregate(errors)
}

func (n *nodeServiceProvider) Lock() {
	n.serviceMutex.Lock()
}

func (n *nodeServiceProvider) Unlock() {
	n.serviceMutex.Unlock()
}

func (n *nodeServiceProvider) GetNodeIP(ctx context.Context, name types.NamespacedName) (string, error) {
	serviceName := translate.SafeConcatName(translate.Suffix, "node", name.Name)

	service := &corev1.Service{}
	err := n.currentNamespaceClient.Get(ctx, types.NamespacedName{Name: serviceName, Namespace: n.currentNamespace}, service)
	if err != nil && !kerrors.IsNotFound(err) {
		return "", errors.Wrap(err, "list services")
	} else if err == nil {
		return service.Spec.ClusterIP, nil
	}

	// create a new service if we can't find one
	podName, err := clienthelper.CurrentPodName()
	if err != nil {
		return "", errors.Wrap(err, "get current pod name")
	}

	// find out the labels to select ourself
	pod := &corev1.Pod{}
	err = n.currentNamespaceClient.Get(ctx, types.NamespacedName{Name: podName, Namespace: n.currentNamespace}, pod)
	if err != nil {
		return "", errors.Wrap(err, "get pod")
	} else if len(pod.Labels) == 0 {
		return "", fmt.Errorf("vcluster pod has no labels to select it")
	}

	// create label selector
	labelSelector := map[string]string{}
	for k, v := range pod.Labels {
		if k == "controller-revision-hash" || k == "statefulset.kubernetes.io/pod-name" || k == "pod-template-hash" {
			continue
		}

		labelSelector[k] = v
	}

	// create the new service
	nodeService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: n.currentNamespace,
			Name:      serviceName,
			Labels: map[string]string{
				ServiceClusterLabel: translate.Suffix,
				ServiceNodeLabel:    name.Name,
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Port:       int32(KubeletPort),
					TargetPort: intstr.FromInt(KubeletTargetPort),
				},
			},
			Selector: labelSelector,
		},
	}

	// set owning stateful set if defined
	if translate.Owner != nil {
		nodeService.SetOwnerReferences(translate.GetOwnerReference())
	}

	// create the service
	klog.Infof("Generating kubelet service for node %s", name.Name)
	err = n.currentNamespaceClient.Create(ctx, nodeService)
	if err != nil {
		return "", errors.Wrap(err, "create node service")
	}

	return nodeService.Spec.ClusterIP, nil
}
