package nodeservice

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var (
	// ServiceClusterLabel identifies to which vcluster the node service belongs if there are multiple in one namespace
	ServiceClusterLabel = "vcluster.loft.sh/belongs-to"
	// ServiceNodeLabel specifies which node this service represents
	ServiceNodeLabel = "vcluster.loft.sh/node"
	// KubeletTargetPort is the port vcluster will run under
	KubeletTargetPort = 8443
)

type Provider interface {
	sync.Locker
	// Start starts the node service garbage collector
	Start(ctx context.Context)
	// GetNodeIP returns a new fake node ip
	GetNodeIP(ctx context.Context, name string) (string, error)
}

func NewNodeServiceProvider(serviceName, currentNamespace string, currentNamespaceClient client.Client, virtualClient client.Client, uncachedVirtualClient client.Client) Provider {
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
		ServiceClusterLabel: translate.VClusterName,
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

func (n *nodeServiceProvider) GetNodeIP(ctx context.Context, name string) (string, error) {
	serviceName := translate.SafeConcatName(translate.VClusterName, "node", strings.ReplaceAll(name, ".", "-"))

	service := &corev1.Service{}
	err := n.currentNamespaceClient.Get(ctx, types.NamespacedName{Name: serviceName, Namespace: n.currentNamespace}, service)
	if err != nil && !kerrors.IsNotFound(err) {
		return "", errors.Wrap(err, "list services")
	} else if err == nil {
		if service.Spec.Selector == nil {
			err = n.updateNodeServiceEndpoints(ctx, serviceName)
			if err != nil {
				return "", errors.Wrap(err, "update node service endpoints")
			}
		}

		return service.Spec.ClusterIP, nil
	}

	// find out the labels to select ourself
	vclusterService := &corev1.Service{}
	err = n.currentNamespaceClient.Get(ctx, types.NamespacedName{Name: n.serviceName, Namespace: n.currentNamespace}, vclusterService)
	if err != nil {
		return "", errors.Wrap(err, "get vcluster service")
	}

	// create the new service
	targetPort := intstr.FromInt32(int32(KubeletTargetPort))
	if vclusterService.Spec.Selector == nil {
		targetPort = intstr.IntOrString{}
	}
	nodeService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: n.currentNamespace,
			Name:      serviceName,
			Labels: map[string]string{
				ServiceClusterLabel: translate.VClusterName,
				ServiceNodeLabel:    name,
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       "kubelet",
					Port:       constants.KubeletPort,
					TargetPort: targetPort,
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
	klog.Infof("Generating kubelet service for node %s", name)
	err = n.currentNamespaceClient.Create(ctx, nodeService)
	if err != nil {
		return "", errors.Wrap(err, "create node service")
	}

	// create endpoints if selector is empty
	if vclusterService.Spec.Selector == nil {
		err = n.updateNodeServiceEndpoints(ctx, serviceName)
		if err != nil {
			return "", errors.Wrap(err, "update node service endpoints")
		}
	}

	return nodeService.Spec.ClusterIP, nil
}

func (n *nodeServiceProvider) updateNodeServiceEndpoints(ctx context.Context, nodeServiceName string) error {
	vClusterServiceEndpoints := &corev1.Endpoints{}
	err := n.currentNamespaceClient.Get(ctx, types.NamespacedName{Name: n.serviceName, Namespace: n.currentNamespace}, vClusterServiceEndpoints)
	if err != nil {
		return errors.Wrap(err, "get vcluster service endpoints")
	}

	// filter subsets
	nodeServiceEndpoints := &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: n.currentNamespace,
			Name:      nodeServiceName,
		},
	}
	result, err := controllerutil.CreateOrPatch(ctx, n.currentNamespaceClient, nodeServiceEndpoints, func() error {
		// build new subsets
		newSubsets := []corev1.EndpointSubset{}
		for _, subset := range vClusterServiceEndpoints.Subsets {
			newPorts := []corev1.EndpointPort{}
			for _, p := range subset.Ports {
				if p.Name != "https" {
					continue
				}

				newPorts = append(newPorts, corev1.EndpointPort{
					Name:        "kubelet",
					Port:        p.Port,
					Protocol:    p.Protocol,
					AppProtocol: p.AppProtocol,
				})
			}

			newAddresses := []corev1.EndpointAddress{}
			for _, address := range subset.Addresses {
				address.Hostname = ""
				address.NodeName = nil
				address.TargetRef = nil
				newAddresses = append(newAddresses, address)
			}
			newNotReadyAddresses := []corev1.EndpointAddress{}
			for _, address := range subset.NotReadyAddresses {
				address.Hostname = ""
				address.NodeName = nil
				address.TargetRef = nil
				newNotReadyAddresses = append(newNotReadyAddresses, address)
			}

			newSubsets = append(newSubsets, corev1.EndpointSubset{
				Addresses:         newAddresses,
				NotReadyAddresses: newNotReadyAddresses,
				Ports:             newPorts,
			})
		}

		nodeServiceEndpoints.Subsets = newSubsets
		return nil
	})
	if err != nil {
		return err
	} else if result == controllerutil.OperationResultUpdated {
		klog.Infof("Updated service endpoints for node %s", nodeServiceName)
	}

	return nil
}
