package services

import (
	"context"
	"time"

	"github.com/loft-sh/vcluster/pkg/controllers/syncer"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"

	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var ServiceBlockDeletion = "vcluster.loft.sh/block-deletion"

func New(ctx *synccontext.RegisterContext) (syncer.Object, error) {
	return &serviceSyncer{
		// exclude "field.cattle.io/publicEndpoints" annotation used by Rancher,
		// because if it is also installed in the host cluster, it will be
		// overriding it, which would cause endless updates back and forth.
		NamespacedTranslator: translator.NewNamespacedTranslator(ctx, "service", &corev1.Service{}, "field.cattle.io/publicEndpoints"),

		serviceName: ctx.Options.ServiceName,
	}, nil
}

type serviceSyncer struct {
	translator.NamespacedTranslator

	serviceName string
}

func (s *serviceSyncer) SyncDown(ctx *synccontext.SyncContext, vObj client.Object) (ctrl.Result, error) {
	return s.SyncDownCreate(ctx, vObj, s.translate(vObj.(*corev1.Service)))
}

func (s *serviceSyncer) Sync(ctx *synccontext.SyncContext, pObj client.Object, vObj client.Object) (ctrl.Result, error) {
	vService := vObj.(*corev1.Service)
	pService := pObj.(*corev1.Service)

	// delay if we are in the middle of a switch operation
	if isSwitchingFromExternalName(pService, vService) {
		return ctrl.Result{RequeueAfter: time.Second * 3}, nil
	}

	// check if backwards update is necessary
	newService := s.translateUpdateBackwards(pService, vService)
	if newService != nil {
		if vService.Spec.ClusterIP != pService.Spec.ClusterIP {
			newService.Spec.ClusterIPs = nil
			ctx.Log.Infof("recreating virtual service %s/%s, because cluster ip differs %s != %s", vService.Namespace, vService.Name, pService.Spec.ClusterIP, vService.Spec.ClusterIP)

			// recreate the new service with the correct cluster ip
			_, err := recreateService(ctx.Context, ctx.VirtualClient, newService)
			if err != nil {
				ctx.Log.Errorf("error creating virtual service: %s/%s", vService.Namespace, vService.Name)
				return ctrl.Result{}, err
			}
		} else {
			// update with correct ports
			ctx.Log.Infof("update virtual service %s/%s, because spec is out of sync", vService.Namespace, vService.Name)
			err := ctx.VirtualClient.Update(ctx.Context, newService)
			if err != nil {
				return ctrl.Result{}, err
			}
		}

		// we will requeue anyways
		return ctrl.Result{Requeue: true}, nil
	}

	// check if backwards status update is necessary
	if !equality.Semantic.DeepEqual(vService.Status, pService.Status) {
		newService := vService.DeepCopy()
		newService.Status = pService.Status
		ctx.Log.Infof("update virtual service %s/%s, because status is out of sync", vService.Namespace, vService.Name)
		translator.PrintChanges(vService, newService, ctx.Log)
		err := ctx.VirtualClient.Status().Update(ctx.Context, newService)
		if err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{Requeue: true}, nil
	}

	// forward update
	newService = s.translateUpdate(pService, vService)
	if newService != nil {
		translator.PrintChanges(pService, newService, ctx.Log)
	}

	return s.SyncDownUpdate(ctx, vObj, newService)
}

func isSwitchingFromExternalName(pService *corev1.Service, vService *corev1.Service) bool {
	return vService.Spec.Type == corev1.ServiceTypeExternalName && pService.Spec.Type != vService.Spec.Type && pService.Spec.ClusterIP != ""
}

var _ syncer.UpSyncer = &serviceSyncer{}

func (s *serviceSyncer) SyncUp(ctx *synccontext.SyncContext, pObj client.Object) (ctrl.Result, error) {
	if !translate.Default.IsManaged(pObj) {
		return ctrl.Result{}, nil
	}

	// we have to delay deletion here if a vObj does not (yet) exist for a service that was just
	// created, because vcluster intercepts those calls and first creates a service inside the host
	// cluster and then inside the virtual cluster.
	pService := pObj.(*corev1.Service)
	if pService.Annotations != nil && pService.Annotations[ServiceBlockDeletion] == "true" {
		return ctrl.Result{Requeue: true}, nil
	}

	return syncer.DeleteObject(ctx, pObj)
}

func recreateService(ctx context.Context, virtualClient client.Client, vService *corev1.Service) (*corev1.Service, error) {
	// delete & create with correct ClusterIP
	err := virtualClient.Delete(ctx, vService)
	if err != nil && !kerrors.IsNotFound(err) {
		return nil, err
	}

	// make sure we don't set the resource version during create
	vService = vService.DeepCopy()
	vService.ResourceVersion = ""
	vService.UID = ""
	vService.DeletionTimestamp = nil
	vService.Generation = 0

	// create the new service with the correct cluster ip
	err = virtualClient.Create(ctx, vService)
	if err != nil {
		klog.Errorf("error recreating virtual service: %s/%s: %v", vService.Namespace, vService.Name, err)
		return nil, err
	}

	return vService, nil
}

var _ syncer.Starter = &serviceSyncer{}

func (s *serviceSyncer) ReconcileStart(ctx *synccontext.SyncContext, req ctrl.Request) (bool, error) {
	// don't do anything for the kubernetes service
	if req.Name == "kubernetes" && req.Namespace == "default" {
		return true, SyncKubernetesService(ctx.Context, ctx.VirtualClient, ctx.CurrentNamespaceClient, ctx.CurrentNamespace, s.serviceName)
	}

	return false, nil
}

func (s *serviceSyncer) ReconcileEnd() {}

func SyncKubernetesService(ctx context.Context, virtualClient client.Client, localClient client.Client, serviceNamespace, serviceName string) error {
	// get physical service
	pObj := &corev1.Service{}
	err := localClient.Get(ctx, types.NamespacedName{
		Namespace: serviceNamespace,
		Name:      serviceName,
	}, pObj)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return nil
		}

		return err
	}

	// get virtual service
	vObj := &corev1.Service{}
	err = virtualClient.Get(ctx, types.NamespacedName{
		Namespace: "default",
		Name:      "kubernetes",
	}, vObj)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return nil
		}

		return err
	}

	translatedPorts := translateKubernetesServicePorts(pObj.Spec.Ports)
	if vObj.Spec.ClusterIP != pObj.Spec.ClusterIP || !equality.Semantic.DeepEqual(vObj.Spec.Ports, translatedPorts) {
		newService := vObj.DeepCopy()
		newService.Spec.ClusterIP = pObj.Spec.ClusterIP
		newService.Spec.Ports = translatedPorts
		if vObj.Spec.ClusterIP != pObj.Spec.ClusterIP {
			newService.Spec.ClusterIPs = nil

			// delete & create with correct ClusterIP
			err = virtualClient.Delete(ctx, vObj)
			if err != nil {
				return err
			}

			// make sure we don't set the resource version during create
			newService.ResourceVersion = ""

			// create the new service with the correct cluster ip
			err = virtualClient.Create(ctx, newService)
			if err != nil {
				return err
			}
		} else {
			// delete & create with correct ClusterIP
			err = virtualClient.Update(ctx, newService)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func translateKubernetesServicePorts(ports []corev1.ServicePort) []corev1.ServicePort {
	retPorts := []corev1.ServicePort{}
	for _, p := range ports {
		// Delete the NodePort
		retPorts = append(retPorts, corev1.ServicePort{
			Name:        p.Name,
			Protocol:    p.Protocol,
			AppProtocol: p.AppProtocol,
			Port:        p.Port,
			TargetPort:  p.TargetPort,
		})
	}

	return retPorts
}
