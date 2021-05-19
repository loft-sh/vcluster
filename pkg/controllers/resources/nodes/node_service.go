package nodes

import (
	"context"
	"fmt"
	"github.com/loft-sh/vcluster/pkg/util/clienthelper"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	// ServiceClusterLabel identifies to which vcluster the node service belongs if there are multiple in one namespace
	ServiceClusterLabel = "vcluster.loft.sh/belongs-to"
	// ServiceNodeLabel specifies which node this service represents
	ServiceNodeLabel = "vcluster.loft.sh/node"
	// KubeletPort is the port we pretend the kubelet is running under
	KubeletPort = int32(8443)
)

type NodeServiceProvider interface {
	CleanupNodeServices(ctx context.Context, name types.NamespacedName) error
	GetNodeIP(ctx context.Context, name types.NamespacedName) (string, error)
}

func NewNodeServiceProvider(localClient client.Client) NodeServiceProvider {
	return &nodeServiceProvider{
		localClient: localClient,
	}
}

type nodeServiceProvider struct {
	localClient client.Client
}

func (n *nodeServiceProvider) CleanupNodeServices(ctx context.Context, name types.NamespacedName) error {
	namespace, err := clienthelper.CurrentNamespace()
	if err != nil {
		return errors.Wrap(err, "get current namespace")
	}

	serviceList := &corev1.ServiceList{}
	err = n.localClient.List(ctx, serviceList, client.InNamespace(namespace), client.MatchingLabels{
		ServiceClusterLabel: translate.Suffix,
		ServiceNodeLabel:    name.Name,
	})
	if err != nil {
		return errors.Wrap(err, "list services")
	}

	errors := []error{}
	for _, s := range serviceList.Items {
		klog.Infof("Cleaning up kubelet service for node %s", s.Labels[ServiceNodeLabel])
		err = n.localClient.Delete(ctx, &s)
		if err != nil {
			errors = append(errors, err)
		}
	}

	return utilerrors.NewAggregate(errors)
}

func (n *nodeServiceProvider) GetNodeIP(ctx context.Context, name types.NamespacedName) (string, error) {
	namespace, err := clienthelper.CurrentNamespace()
	if err != nil {
		return "", errors.Wrap(err, "get current namespace")
	}

	serviceList := &corev1.ServiceList{}
	err = n.localClient.List(ctx, serviceList, client.InNamespace(namespace), client.MatchingLabels{
		ServiceClusterLabel: translate.Suffix,
		ServiceNodeLabel:    name.Name,
	})
	if err != nil {
		return "", errors.Wrap(err, "list services")
	} else if len(serviceList.Items) > 0 {
		return serviceList.Items[0].Spec.ClusterIP, nil
	}

	// create a new service if we can't find one
	podName, err := clienthelper.CurrentPodName()
	if err != nil {
		return "", errors.Wrap(err, "get current pod name")
	}

	// find out the labels to select ourself
	pod := &corev1.Pod{}
	err = n.localClient.Get(ctx, types.NamespacedName{Name: podName, Namespace: namespace}, pod)
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
			Namespace:    namespace,
			GenerateName: translate.SafeConcatGenerateName(translate.Suffix, "node", name.Name),
			Labels: map[string]string{
				ServiceClusterLabel: translate.Suffix,
				ServiceNodeLabel:    name.Name,
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Port: int32(KubeletPort),
				},
			},
			Selector: labelSelector,
		},
	}

	// set owning stateful set if defined
	if translate.OwningStatefulSet != nil {
		nodeService.SetOwnerReferences([]metav1.OwnerReference{
			{
				APIVersion: appsv1.SchemeGroupVersion.String(),
				Kind:       "StatefulSet",
				Name:       translate.OwningStatefulSet.Name,
				UID:        translate.OwningStatefulSet.UID,
			},
		})
	}

	// create the service
	klog.Infof("Generating kubelet service for node %s", name.Name)
	err = n.localClient.Create(ctx, nodeService)
	if err != nil {
		return "", errors.Wrap(err, "create node service")
	}

	return nodeService.Spec.ClusterIP, nil
}
