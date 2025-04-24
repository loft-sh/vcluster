package k8sdefaultendpoint

import (
	"fmt"
	"time"

	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func SyncKubernetesServiceDedicated(ctx *synccontext.SyncContext) error {
	// get the ip and port
	ip, port, err := GetVClusterDedicatedControlPlaneEndpoint(ctx)
	if err != nil {
		return err
	}

	// create or update endpoints
	vObj := &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubernetes",
			Namespace: "default",
		},
	}
	_, err = controllerutil.CreateOrPatch(ctx, ctx.VirtualClient, vObj, func() error {
		vObj.Subsets = []corev1.EndpointSubset{
			{
				Addresses: []corev1.EndpointAddress{
					{
						IP: ip,
					},
				},
				Ports: []corev1.EndpointPort{
					{
						Port:     port,
						Name:     "https",
						Protocol: corev1.ProtocolTCP,
					},
				},
			},
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to create or patch endpoints: %w", err)
	}

	// create or patch endpoint slice
	provider := &v1Provider{}
	err = provider.createOrPatch(ctx, ctx.VirtualClient, vObj)
	if err != nil {
		return fmt.Errorf("failed to create or patch endpoint slice: %w", err)
	}

	return err
}

func GetVClusterDedicatedControlPlaneEndpoint(ctx *synccontext.SyncContext) (string, int32, error) {
	// get physical service
	pObj := &corev1.Service{}
	err := ctx.CurrentNamespaceClient.Get(ctx, types.NamespacedName{
		Namespace: ctx.Config.WorkloadNamespace,
		Name:      ctx.Config.WorkloadService,
	}, pObj)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return "", 0, nil
		}
		return "", 0, err
	}

	// get the ip and port
	ip := pObj.Spec.ClusterIP
	port := pObj.Spec.Ports[0].Port
	if pObj.Spec.Type == corev1.ServiceTypeLoadBalancer {
		// wait for the load balancer to get an ip
		if len(pObj.Status.LoadBalancer.Ingress) == 0 {
			time.Sleep(time.Second)
			klog.Infof("Waiting for load balancer ingress to get an ip...")
			return GetVClusterDedicatedControlPlaneEndpoint(ctx)
		}

		if pObj.Status.LoadBalancer.Ingress[0].IP == "" {
			return "", 0, fmt.Errorf("load balancer ingress ip is not set")
		}

		ip = pObj.Status.LoadBalancer.Ingress[0].IP
	}
	// TODO: handle node port services

	return ip, port, nil
}
