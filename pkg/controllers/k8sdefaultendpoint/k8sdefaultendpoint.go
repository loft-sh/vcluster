package k8sdefaultendpoint

import (
	"context"
	"fmt"

	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/constants"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	"github.com/loft-sh/vcluster/pkg/specialservices"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilnet "k8s.io/utils/net"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

type provider interface {
	createClientObject() client.Object
	createOrPatch(ctx context.Context, virtualClient client.Client, vEndpoints *corev1.Endpoints) error
}

type EndpointController struct {
	ServiceName      string
	ServiceNamespace string
	ServiceClient    client.Client

	VirtualClient       client.Client
	VirtualManagerCache cache.Cache

	Log loghelper.Logger

	provider provider
}

func NewEndpointController(ctx *config.ControllerContext, provider provider) *EndpointController {
	return &EndpointController{
		VirtualClient:       ctx.VirtualManager.GetClient(),
		VirtualManagerCache: ctx.VirtualManager.GetCache(),

		ServiceName:      ctx.Config.WorkloadService,
		ServiceNamespace: ctx.Config.WorkloadNamespace,
		ServiceClient:    ctx.WorkloadNamespaceClient,

		Log:      loghelper.New("kubernetes-default-endpoint-controller"),
		provider: provider,
	}
}

func (e *EndpointController) Register(mgr ctrl.Manager) error {
	err := e.SetupWithManager(mgr)
	if err != nil {
		return fmt.Errorf("unable to setup pod security controller: %w", err)
	}

	return nil
}

func (e *EndpointController) Reconcile(ctx context.Context, _ ctrl.Request) (ctrl.Result, error) {
	err := e.syncKubernetesServiceEndpoints(ctx)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager adds the controller to the manager
func (e *EndpointController) SetupWithManager(mgr ctrl.Manager) error {
	// creating a predicate to receive reconcile requests for kubernetes endpoint only
	pPredicates := predicate.NewTypedPredicateFuncs(func(object client.Object) bool {
		return object.GetNamespace() == e.ServiceNamespace && object.GetName() == e.ServiceName
	})

	vPredicates := predicate.NewTypedPredicateFuncs(func(object client.Object) bool {
		if object.GetNamespace() == specialservices.DefaultKubernetesSvcKey.Namespace && object.GetName() == specialservices.DefaultKubernetesSvcKey.Name {
			return true
		}

		return false
	})

	return ctrl.NewControllerManagedBy(mgr).
		Named("kubernetes_default_endpoint").
		WithOptions(controller.Options{
			CacheSyncTimeout: constants.DefaultCacheSyncTimeout,
		}).
		For(&corev1.Endpoints{}, builder.WithPredicates(pPredicates)).
		WatchesRawSource(source.Kind[client.Object](e.VirtualManagerCache, &corev1.Endpoints{}, &handler.TypedEnqueueRequestForObject[client.Object]{}, vPredicates)).
		WatchesRawSource(source.Kind(e.VirtualManagerCache, e.provider.createClientObject(), &handler.EnqueueRequestForObject{}, vPredicates)).
		Complete(e)
}

func (e *EndpointController) syncKubernetesServiceEndpoints(ctx context.Context) error {
	// get physical service endpoints
	pEndpoints := &corev1.Endpoints{}
	err := e.ServiceClient.Get(ctx, types.NamespacedName{
		Namespace: e.ServiceNamespace,
		Name:      e.ServiceName,
	}, pEndpoints)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return nil
		}

		return err
	}

	vEndpoints := &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "kubernetes",
		},
	}
	result, err := controllerutil.CreateOrPatch(ctx, e.VirtualClient, vEndpoints, func() error {
		if vEndpoints.Labels == nil {
			vEndpoints.Labels = map[string]string{}
		}
		vEndpoints.Labels[discoveryv1.LabelSkipMirror] = "true"

		// build new subsets
		newSubsets := []corev1.EndpointSubset{}
		for _, subset := range pEndpoints.Subsets {
			newPorts := []corev1.EndpointPort{}
			for _, p := range subset.Ports {
				if p.Name != "https" {
					continue
				}

				newPorts = append(newPorts, p)
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

		vEndpoints.Subsets = newSubsets
		return nil
	})
	if err != nil {
		return fmt.Errorf("error patching endpoints: %w", err)
	}

	if result == controllerutil.OperationResultCreated || result == controllerutil.OperationResultUpdated {
		return e.provider.createOrPatch(ctx, e.VirtualClient, vEndpoints)
	}

	return nil
}

// allAddressesIPv6 returns true if all provided addresses are IPv6.
// From: https://github.com/kubernetes/kubernetes/blob/7380fc735aca591325ae1fabf8dab194b40367de/pkg/controlplane/reconcilers/endpointsadapter.go#L183-L196
func allAddressesIPv6(addresses []corev1.EndpointAddress) bool {
	if len(addresses) == 0 {
		return false
	}

	for _, address := range addresses {
		if !utilnet.IsIPv6String(address.IP) {
			return false
		}
	}

	return true
}
