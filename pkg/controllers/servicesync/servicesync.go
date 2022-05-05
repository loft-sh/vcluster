package servicesync

import (
	"context"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/services"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"strings"
)

type ServiceSyncer struct {
	SyncServices    map[string]types.NamespacedName
	CreateNamespace bool

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
		Named("servicesync").
		For(&corev1.Service{}).
		Watches(source.NewKindWithCache(&corev1.Service{}, e.To.GetCache()), &serviceHandler{Mapping: reverseMapping}).
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
		e.Log.Infof("Delete target service %s/%s because from service is missing", to.Name, to.Namespace)
		err = e.To.GetClient().Delete(ctx, &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      to.Name,
				Namespace: to.Namespace,
			},
		})
		if err != nil && !kerrors.IsNotFound(err) && !kerrors.IsForbidden(err) {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	// make sure we don't copy the node ports
	fromService = fromService.DeepCopy()
	services.StripNodePorts(fromService)

	// compare to endpoint and service
	toService := &corev1.Service{}
	err = e.To.GetClient().Get(ctx, to, toService)
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
				Annotations: map[string]string{
					constants.SkipSyncAnnotation: "true",
				},
			},
			Spec: corev1.ServiceSpec{
				Ports:     fromService.Spec.Ports,
				ClusterIP: corev1.ClusterIPNone,
			},
		}
		e.Log.Infof("Create target service %s/%s because it is missing", to.Namespace, to.Name)
		return ctrl.Result{}, e.To.GetClient().Create(ctx, toService)
	} else if toService.Annotations == nil || toService.Annotations[constants.SkipSyncAnnotation] != "true" {
		// skip as it seems the service was user created
		return ctrl.Result{}, nil
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

		// create endpoints
		toEndpoints = &corev1.Endpoints{
			ObjectMeta: metav1.ObjectMeta{
				Name:      to.Name,
				Namespace: to.Namespace,
				Annotations: map[string]string{
					constants.SkipSyncAnnotation: "true",
				},
			},
			Subsets: []corev1.EndpointSubset{
				{
					Addresses: []corev1.EndpointAddress{
						{
							IP: fromService.Spec.ClusterIP,
						},
					},
					Ports: convertPorts(toService.Spec.Ports),
				},
			},
		}
		e.Log.Infof("Create target endpoints %s/%s because they are missing", to.Namespace, to.Name)
		return ctrl.Result{}, e.To.GetClient().Create(ctx, toEndpoints)
	}

	// check if update is needed
	expectedSubsets := []corev1.EndpointSubset{
		{
			Addresses: []corev1.EndpointAddress{
				{
					IP: fromService.Spec.ClusterIP,
				},
			},
			Ports: convertPorts(toService.Spec.Ports),
		},
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
