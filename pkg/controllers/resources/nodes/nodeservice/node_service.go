package nodeservice

import (
	"context"
	"strings"
	"sync"
	"time"

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

func NewNodeServiceProvider(serviceName, currentNamespace string, currentNamespaceClient client.Client, virtualClient client.Client, uncachedVirtualClient client.Client) NodeServiceProvider {
	return &nodeServiceProvider{
		serviceName:            serviceName,
		currentNamespace:       currentNamespace,
		currentNamespaceClient: currentNamespaceClient,
		virtualClient:          virtualClient,
		uncachedVirtualClient:  uncachedVirtualClient,
	}
}

type nodeServiceProvider struct {
	serviceName            string
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
				if !kerrors.IsNotFound(err) {
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

		if !exist {
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
	serviceName := translate.SafeConcatName(translate.Suffix, "node", strings.Replace(name.Name, ".", "-", -1))

	service := &corev1.Service{}
	err := n.currentNamespaceClient.Get(ctx, types.NamespacedName{Name: serviceName, Namespace: n.currentNamespace}, service)
	if err != nil && !kerrors.IsNotFound(err) {
		return "", errors.Wrap(err, "list services")
	} else if err == nil {
		return service.Spec.ClusterIP, nil
	}

	// find out the labels to select ourself
	vclusterService := &corev1.Service{}
	err = n.currentNamespaceClient.Get(ctx, types.NamespacedName{Name: n.serviceName, Namespace: n.currentNamespace}, vclusterService)
	if err != nil {
		return "", errors.Wrap(err, "get vcluster service")
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
			Selector: vclusterService.Spec.Selector,
		},
	}

	// set owner if defined
	if translate.Owner != nil {
		nodeService.SetOwnerReferences(translate.GetOwnerReference(nil))
	}

	// create the service
	klog.Infof("Generating kubelet service for node %s", name.Name)
	err = n.currentNamespaceClient.Create(ctx, nodeService)
	if err != nil {
		return "", errors.Wrap(err, "create node service")
	}

	return nodeService.Spec.ClusterIP, nil
}
