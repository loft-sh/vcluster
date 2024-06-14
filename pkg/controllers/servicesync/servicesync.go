package servicesync

import (
	"context"
	"strings"

	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/services"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

type ServiceSyncer struct {
	SyncServices map[string]types.NamespacedName

	IsVirtualToHostSyncer bool
	CreateNamespace       bool
	CreateEndpoints       bool

	From ctrl.Manager
	To   ctrl.Manager

	Log loghelper.Logger
}

func (e *ServiceSyncer) Register() error {
	reverseMapping := map[string]types.NamespacedName{}
	for k, v := range e.SyncServices {
		splitted := strings.Split(k, "/")
		reverseMapping[v.Namespace+"/"+v.Name] = types.NamespacedName{
			Namespace: splitted[0],
			Name:      splitted[1],
		}
	}

	return ctrl.NewControllerManagedBy(e.From).
		WithOptions(controller.Options{
			CacheSyncTimeout: constants.DefaultCacheSyncTimeout,
		}).
		Named("servicesync").
		For(&corev1.Service{}).
		WatchesRawSource(source.Kind(e.To.GetCache(), &corev1.Service{}, handler.TypedEnqueueRequestsFromMapFunc(func(_ context.Context, object *corev1.Service) []reconcile.Request {
			if object == nil {
				return nil
			}

			from, ok := reverseMapping[object.GetNamespace()+"/"+object.GetName()]
			if !ok {
				return nil
			}

			return []reconcile.Request{{NamespacedName: from}}
		}))).
		WatchesRawSource(source.Kind(e.From.GetCache(), &corev1.Endpoints{}, handler.TypedEnqueueRequestsFromMapFunc(func(_ context.Context, object *corev1.Endpoints) []reconcile.Request {
			if object == nil {
				return nil
			}

			_, ok := e.SyncServices[object.GetNamespace()+"/"+object.GetName()]
			if !ok {
				return nil
			}

			return []reconcile.Request{{
				NamespacedName: types.NamespacedName{Namespace: object.GetNamespace(), Name: object.GetName()},
			}}
		}))).
		Complete(e)
}

func (e *ServiceSyncer) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	from := req.Namespace + "/" + req.Name
	to, ok := e.SyncServices[from]
	if !ok {
		return ctrl.Result{}, nil
	}

	// check if from service still exists
	fromService := &corev1.Service{}
	err := e.From.GetClient().Get(ctx, req.NamespacedName, fromService)
	if err != nil {
		if !kerrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}

		// make sure the to service is deleted
		e.Log.Infof("Delete target service %s/%s because from service is missing", to.Namespace, to.Name)
		err = e.To.GetClient().Delete(ctx, &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      to.Name,
				Namespace: to.Namespace,
			},
		})
		if err != nil && !kerrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	// make sure we don't copy the node ports
	fromService = fromService.DeepCopy()
	services.StripNodePorts(fromService)

	// if we should create endpoints
	if e.CreateEndpoints {
		return e.syncServiceAndEndpoints(ctx, fromService, to)
	}

	return e.syncServiceWithSelector(ctx, fromService, to)
}

func (e *ServiceSyncer) syncServiceWithSelector(ctx context.Context, fromService *corev1.Service, to types.NamespacedName) (ctrl.Result, error) {
	// compare to endpoint and service
	toService := &corev1.Service{}
	err := e.To.GetClient().Get(ctx, to, toService)
	if err != nil {
		if !kerrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}

		// create service
		toService = &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      to.Name,
				Namespace: to.Namespace,
				Labels: map[string]string{
					translate.ControllerLabel: "vcluster",
				},
			},
			Spec: corev1.ServiceSpec{
				Ports: fromService.Spec.Ports,
			},
		}

		// case for a headless service
		if fromService.Spec.ClusterIP == corev1.ClusterIPNone {
			toService.Spec.ClusterIP = corev1.ClusterIPNone
		}

		if e.IsVirtualToHostSyncer {
			e.Log.Infof("Add owner reference to host target service %s", to.Name)
			toService.OwnerReferences = translate.GetOwnerReference(nil)
		}
		toService.Spec.Selector = translate.Default.TranslateLabels(fromService.Spec.Selector, fromService.Namespace, nil)
		e.Log.Infof("Create target service %s/%s because it is missing", to.Namespace, to.Name)
		return ctrl.Result{}, e.To.GetClient().Create(ctx, toService)
	} else if toService.Labels == nil || toService.Labels[translate.ControllerLabel] != "vcluster" {
		// skip as it seems the service was user created
		return ctrl.Result{}, nil
	}

	// rewrite selector
	targetService := toService.DeepCopy()
	targetService.Spec.Selector = translate.Default.TranslateLabels(fromService.Spec.Selector, fromService.Namespace, nil)

	// compare service ports
	if !apiequality.Semantic.DeepEqual(toService.Spec.Ports, fromService.Spec.Ports) || !apiequality.Semantic.DeepEqual(toService.Spec.Selector, targetService.Spec.Selector) {
		e.Log.Infof("Update target service %s/%s because ports or selector are different", to.Namespace, to.Name)
		toService.Spec.Ports = fromService.Spec.Ports
		toService.Spec.Selector = targetService.Spec.Selector
		return ctrl.Result{}, e.To.GetClient().Update(ctx, toService)
	}

	return ctrl.Result{}, nil
}

func (e *ServiceSyncer) syncServiceAndEndpoints(ctx context.Context, fromService *corev1.Service, to types.NamespacedName) (ctrl.Result, error) {
	// compare to endpoint and service
	toService := &corev1.Service{}
	err := e.To.GetClient().Get(ctx, to, toService)
	if err != nil {
		if !kerrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}

		// check if namespace exists
		if e.CreateNamespace {
			namespace := &corev1.Namespace{}
			err = e.To.GetClient().Get(ctx, types.NamespacedName{Name: to.Namespace}, namespace)
			if err != nil && !kerrors.IsNotFound(err) {
				return ctrl.Result{}, err
			} else if kerrors.IsNotFound(err) {
				// create namespace
				e.Log.Infof("Create namespace %s because it is missing", to.Namespace)
				err = e.To.GetClient().Create(ctx, &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: to.Namespace,
					},
				})
				if err != nil {
					return ctrl.Result{}, err
				}
			}
		}

		// create service
		toService = &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      to.Name,
				Namespace: to.Namespace,
				Labels: map[string]string{
					translate.ControllerLabel: "vcluster",
				},
			},
			Spec: corev1.ServiceSpec{
				Ports:     fromService.Spec.Ports,
				ClusterIP: corev1.ClusterIPNone,
			},
		}

		if e.IsVirtualToHostSyncer {
			e.Log.Infof("Add owner reference to host target service %s", to.Name)
			toService.OwnerReferences = translate.GetOwnerReference(nil)
		}
		e.Log.Infof("Create target service %s/%s because it is missing", to.Namespace, to.Name)
		return ctrl.Result{}, e.To.GetClient().Create(ctx, toService)
	} else if toService.Labels == nil || toService.Labels[translate.ControllerLabel] != "vcluster" {
		// skip as it seems the service was user created
		return ctrl.Result{}, nil
	}

	// sync the loadbalancer status
	if fromService.Spec.Type == corev1.ServiceTypeLoadBalancer && !apiequality.Semantic.DeepEqual(fromService.Status.LoadBalancer, toService.Status.LoadBalancer) {
		e.Log.Infof("Update target service %s/%s because the loadbalancer status changed", to.Namespace, to.Name)
		toService.Status.LoadBalancer = fromService.Status.LoadBalancer
		return ctrl.Result{}, e.To.GetClient().Status().Update(ctx, toService)
	}
	// compare service ports
	if !apiequality.Semantic.DeepEqual(toService.Spec.Ports, fromService.Spec.Ports) {
		e.Log.Infof("Update target service %s/%s because ports are different", to.Namespace, to.Name)
		toService.Spec.Ports = fromService.Spec.Ports
		return ctrl.Result{}, e.To.GetClient().Update(ctx, toService)
	}

	// check target endpoints
	toEndpoints := &corev1.Endpoints{}
	err = e.To.GetClient().Get(ctx, to, toEndpoints)
	if err != nil {
		if !kerrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}

		// copy subsets from endpoint
		subsets := []corev1.EndpointSubset{}

		if fromService.Spec.ClusterIP == corev1.ClusterIPNone {
			// fetch the corresponding endpoint and assign address from there to here
			fromEndpoint := &corev1.Endpoints{}
			err = e.From.GetClient().Get(ctx, types.NamespacedName{
				Name:      fromService.GetName(),
				Namespace: fromService.GetNamespace(),
			}, fromEndpoint)
			if err != nil {
				return ctrl.Result{}, err
			}

			subsets = fromEndpoint.Subsets
		} else {
			subsets = append(subsets, corev1.EndpointSubset{
				Addresses: []corev1.EndpointAddress{
					{
						IP: fromService.Spec.ClusterIP,
					},
				},
				Ports: convertPorts(toService.Spec.Ports),
			})
		}

		// create endpoints
		toEndpoints = &corev1.Endpoints{
			ObjectMeta: metav1.ObjectMeta{
				Name:      to.Name,
				Namespace: to.Namespace,
				Labels: map[string]string{
					translate.ControllerLabel: "vcluster",
				},
			},
			Subsets: subsets,
		}

		e.Log.Infof("Create target endpoints %s/%s because they are missing", to.Namespace, to.Name)
		return ctrl.Result{}, e.To.GetClient().Create(ctx, toEndpoints)
	}

	// check if update is needed
	var expectedSubsets []corev1.EndpointSubset
	if fromService.Spec.ClusterIP == corev1.ClusterIPNone {
		// fetch the corresponding endpoint and assign address from there to here
		fromEndpoint := &corev1.Endpoints{}
		err = e.From.GetClient().Get(ctx, types.NamespacedName{
			Name:      fromService.GetName(),
			Namespace: fromService.GetNamespace(),
		}, fromEndpoint)
		if err != nil {
			return ctrl.Result{}, err
		}

		expectedSubsets = fromEndpoint.Subsets
	} else {
		expectedSubsets = []corev1.EndpointSubset{
			{
				Addresses: []corev1.EndpointAddress{
					{
						IP: fromService.Spec.ClusterIP,
					},
				},
				Ports: convertPorts(toService.Spec.Ports),
			},
		}
	}

	if !apiequality.Semantic.DeepEqual(toEndpoints.Subsets, expectedSubsets) {
		e.Log.Infof("Update target endpoints %s/%s because subsets are different", to.Namespace, to.Name)
		toEndpoints.Subsets = expectedSubsets
		return ctrl.Result{}, e.To.GetClient().Update(ctx, toEndpoints)
	}

	return ctrl.Result{}, nil
}

func convertPorts(servicePorts []corev1.ServicePort) []corev1.EndpointPort {
	endpointPorts := []corev1.EndpointPort{}
	for _, p := range servicePorts {
		endpointPorts = append(endpointPorts, corev1.EndpointPort{
			Name:        p.Name,
			Port:        p.Port,
			Protocol:    p.Protocol,
			AppProtocol: p.AppProtocol,
		})
	}
	return endpointPorts
}
