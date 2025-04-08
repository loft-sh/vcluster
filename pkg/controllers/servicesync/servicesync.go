package servicesync

import (
	"context"
	"fmt"
	"strings"

	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/services"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/mappings/store/verify"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

type syncResult struct {
	gvk        schema.GroupVersionKind
	fromObject client.Object
	toObject   client.Object
}

type ServiceSyncer struct {
	Name        string
	SyncContext *synccontext.SyncContext

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
		Named(fmt.Sprintf("servicesyncer-%s", e.Name)).
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

	var result *syncResult

	// if we should create endpoints
	if e.CreateEndpoints {
		result, err = e.syncServiceAndEndpoints(ctx, fromService, to)
	} else {
		result, err = e.syncServiceWithSelector(ctx, fromService, to)
	}
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error while trying to sync service %s: %w", req.NamespacedName.String(), err)
	}
	if result == nil {
		// skipped syncing service
		return ctrl.Result{}, nil
	}

	err = e.saveMapping(ctx, *result)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error while trying to save mapping: %w", err)
	}

	return ctrl.Result{}, nil
}

func (e *ServiceSyncer) syncServiceWithSelector(ctx context.Context, fromService *corev1.Service, to types.NamespacedName) (*syncResult, error) {
	// compare to endpoint and service
	toService := &corev1.Service{}
	err := e.To.GetClient().Get(ctx, to, toService)
	if err != nil {
		if !kerrors.IsNotFound(err) {
			return nil, fmt.Errorf("error (diffrent than NotFound) while getting target service %s: %w", to.String(), err)
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
		toService.Spec.Selector = translate.HostLabelsMap(fromService.Spec.Selector, toService.Spec.Selector, fromService.Namespace, false)
		e.Log.Infof("Create target service %s/%s because it is missing", to.Namespace, to.Name)
		err = e.To.GetClient().Create(ctx, toService)
		if err != nil {
			return nil, fmt.Errorf("error while creating target service %s/%s: %w", to.Namespace, to.Name, err)
		}
		return &syncResult{
			gvk:        mappings.Services(),
			fromObject: fromService,
			toObject:   toService,
		}, nil
	} else if toService.Labels == nil || toService.Labels[translate.ControllerLabel] != "vcluster" {
		// skip as it seems the service was user created
		return nil, nil
	}

	// rewrite selector
	targetService := toService.DeepCopy()
	targetService.Spec.Selector = translate.HostLabelsMap(fromService.Spec.Selector, toService.Spec.Selector, fromService.Namespace, false)

	// compare service ports
	if !apiequality.Semantic.DeepEqual(toService.Spec.Ports, fromService.Spec.Ports) || !apiequality.Semantic.DeepEqual(toService.Spec.Selector, targetService.Spec.Selector) {
		e.Log.Infof("Update target service %s/%s because ports or selector are different", to.Namespace, to.Name)
		toService.Spec.Ports = fromService.Spec.Ports
		toService.Spec.Selector = targetService.Spec.Selector
		err = e.To.GetClient().Update(ctx, toService)
		if err != nil {
			return nil, fmt.Errorf("error while updating target service %s/%s: %w", to.Namespace, to.Name, err)
		}
		return &syncResult{
			gvk:        mappings.Services(),
			fromObject: fromService,
			toObject:   toService,
		}, nil
	}

	return nil, nil
}

func (e *ServiceSyncer) syncServiceAndEndpoints(ctx context.Context, fromService *corev1.Service, to types.NamespacedName) (*syncResult, error) {
	// compare to endpoint and service
	toService := &corev1.Service{}
	err := e.To.GetClient().Get(ctx, to, toService)
	if err != nil {
		if !kerrors.IsNotFound(err) {
			return nil, fmt.Errorf("error (diffrent than NotFound) while getting target service %s/%s: %w", to.Namespace, to.Name, err)
		}

		// check if namespace exists
		if e.CreateNamespace {
			namespace := &corev1.Namespace{}
			err = e.To.GetClient().Get(ctx, types.NamespacedName{Name: to.Namespace}, namespace)
			if err != nil && !kerrors.IsNotFound(err) {
				return nil, fmt.Errorf("error (diffrent than NotFound) while getting namespace %s: %w", to.Namespace, err)
			} else if kerrors.IsNotFound(err) {
				// create namespace
				e.Log.Infof("Create namespace %s because it is missing", to.Namespace)
				err = e.To.GetClient().Create(ctx, &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: to.Namespace,
					},
				})
				if err != nil {
					return nil, fmt.Errorf("error (diffrent than NotFound) while creating namespace %s: %w", to.Namespace, err)
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
		err = e.To.GetClient().Create(ctx, toService)
		if err != nil {
			return nil, fmt.Errorf("error while creating target service %s/%s: %w", to.Namespace, to.Name, err)
		}
		return &syncResult{
			gvk:        mappings.Services(),
			fromObject: fromService,
			toObject:   toService,
		}, nil
	} else if toService.Labels == nil || toService.Labels[translate.ControllerLabel] != "vcluster" {
		// skip as it seems the service was user created
		return nil, nil
	}

	// sync the loadbalancer status
	if fromService.Spec.Type == corev1.ServiceTypeLoadBalancer && !apiequality.Semantic.DeepEqual(fromService.Status.LoadBalancer, toService.Status.LoadBalancer) {
		e.Log.Infof("Update target service %s/%s because the loadbalancer status changed", to.Namespace, to.Name)
		toService.Status.LoadBalancer = fromService.Status.LoadBalancer
		err = e.To.GetClient().Status().Update(ctx, toService)
		if err != nil {
			return nil, fmt.Errorf("error while updating target service %s/%s because loadbalancer status changed: %w", to.Namespace, to.Name, err)
		}
		return &syncResult{
			gvk:        mappings.Services(),
			fromObject: fromService,
			toObject:   toService,
		}, nil
	}
	// compare service ports
	if !apiequality.Semantic.DeepEqual(toService.Spec.Ports, fromService.Spec.Ports) {
		e.Log.Infof("Update target service %s/%s because ports are different", to.Namespace, to.Name)
		toService.Spec.Ports = fromService.Spec.Ports
		err = e.To.GetClient().Update(ctx, toService)
		if err != nil {
			return nil, fmt.Errorf("error while updating target service %s/%s because ports changed: %w", to.Namespace, to.Name, err)
		}
		return &syncResult{
			gvk:        mappings.Services(),
			fromObject: fromService,
			toObject:   toService,
		}, nil
	}

	// check target endpoints
	fromEndpoints := &corev1.Endpoints{}
	toEndpoints := &corev1.Endpoints{}
	err = e.To.GetClient().Get(ctx, to, toEndpoints)
	if err != nil {
		if !kerrors.IsNotFound(err) {
			return nil, fmt.Errorf("error (diffrent than NotFound) while getting target endpoints %s: %w", to.String(), err)
		}

		// copy subsets from endpoint
		subsets := []corev1.EndpointSubset{}

		if fromService.Spec.ClusterIP == corev1.ClusterIPNone {
			// fetch the corresponding endpoint and assign address from there to here
			err = e.From.GetClient().Get(ctx, types.NamespacedName{
				Name:      fromService.GetName(),
				Namespace: fromService.GetNamespace(),
			}, fromEndpoints)
			if err != nil {
				return nil, fmt.Errorf("error while getting from endpoints %s/%s: %w", fromService.Namespace, fromService.Name, err)
			}

			subsets = fromEndpoints.Subsets
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
		err = e.To.GetClient().Create(ctx, toEndpoints)
		if err != nil {
			return nil, fmt.Errorf("error while creating target endpoints %s: %w", to.String(), err)
		}
		return &syncResult{
			gvk:        mappings.Endpoints(),
			fromObject: fromEndpoints,
			toObject:   toEndpoints,
		}, nil
	}

	// check if update is needed
	var expectedSubsets []corev1.EndpointSubset
	if fromService.Spec.ClusterIP == corev1.ClusterIPNone {
		// fetch the corresponding endpoint and assign address from there to here
		err = e.From.GetClient().Get(ctx, types.NamespacedName{
			Name:      fromService.GetName(),
			Namespace: fromService.GetNamespace(),
		}, fromEndpoints)
		if err != nil {
			return nil, fmt.Errorf("error while getting from endpoints %s/%s: %w", fromService.Namespace, fromService.Name, err)
		}

		expectedSubsets = fromEndpoints.Subsets
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
		err = e.To.GetClient().Update(ctx, toEndpoints)
		if err != nil {
			return nil, fmt.Errorf("error while updating target endpoints %s/%s: %w", toEndpoints.Namespace, toEndpoints.Name, err)
		}
		return &syncResult{
			gvk:        mappings.Endpoints(),
			fromObject: fromEndpoints,
			toObject:   toEndpoints,
		}, nil
	}

	return nil, nil
}

func (e *ServiceSyncer) saveMapping(ctx context.Context, result syncResult) error {
	var pObj, vObj client.Object
	var syncDirection synccontext.SyncDirection
	if e.IsVirtualToHostSyncer {
		vObj = result.fromObject
		pObj = result.toObject
		syncDirection = synccontext.SyncVirtualToHost
	} else {
		vObj = result.toObject
		pObj = result.fromObject
		syncDirection = synccontext.SyncHostToVirtual
	}

	// Save synced service to mappings store
	mapping := synccontext.NameMapping{
		GroupVersionKind: result.gvk,
		HostName: types.NamespacedName{
			Namespace: pObj.GetNamespace(),
			Name:      pObj.GetName(),
		},
		VirtualName: types.NamespacedName{
			Namespace: vObj.GetNamespace(),
			Name:      vObj.GetName(),
		},
		SyncDirection: syncDirection,
	}

	ctx = context.WithValue(ctx, verify.SkipHostNamespaceCheck, true)
	err := e.SyncContext.Mappings.Store().AddReferenceAndSave(ctx, mapping, mapping)
	if err != nil {
		return fmt.Errorf("error while saving mapping %s: %w", mapping.String(), err)
	}

	return nil
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
