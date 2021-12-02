package services

import (
	"context"
	context2 "github.com/loft-sh/vcluster/cmd/vcluster/context"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/generic"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

func RegisterIndices(ctx *context2.ControllerContext) error {
	err := generic.RegisterSyncerIndices(ctx, &corev1.Service{})
	if err != nil {
		return err
	}

	return nil
}

func Register(ctx *context2.ControllerContext, eventBroadcaster record.EventBroadcaster) error {
	recorder := eventBroadcaster.NewRecorder(ctx.VirtualManager.GetScheme(), corev1.EventSource{Component: "service-syncer"})
	return generic.RegisterSyncer(ctx, "service", &syncer{
		Translator: generic.NewNamespacedTranslator(ctx.Options.TargetNamespace, ctx.VirtualManager.GetClient(), &corev1.Service{}),

		currentNamespaceClient: ctx.CurrentNamespaceClient,
		currentNamespace:       ctx.CurrentNamespace,
		serviceName:            ctx.Options.ServiceName,
		
		localClient:   ctx.LocalManager.GetClient(),
		virtualClient: ctx.VirtualManager.GetClient(),

		creator:    generic.NewGenericCreator(ctx.LocalManager.GetClient(), recorder, "service"),
		translator: translate.NewDefaultTranslator(ctx.Options.TargetNamespace),
	})
}

type syncer struct {
	generic.Translator

	currentNamespaceClient client.Client
	currentNamespace       string
	serviceName            string

	localClient   client.Client
	virtualClient client.Client

	creator    *generic.GenericCreator
	translator translate.Translator
}

func (s *syncer) New() client.Object {
	return &corev1.Service{}
}

func (s *syncer) Forward(ctx context.Context, vObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	pObj, err := s.translate(vObj.(*corev1.Service))
	if err != nil {
		return ctrl.Result{}, err
	}

	return s.creator.Create(ctx, vObj, pObj, log)
}

func (s *syncer) Update(ctx context.Context, pObj client.Object, vObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
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
			log.Infof("recreating virtual service %s/%s, because cluster ip differs %s != %s", vService.Namespace, vService.Name, pService.Spec.ClusterIP, vService.Spec.ClusterIP)

			// recreate the new service with the correct cluster ip
			_, err := recreateService(ctx, s.virtualClient, newService)
			if err != nil {
				log.Errorf("error creating virtual service: %s/%s", vService.Namespace, vService.Name)
				return ctrl.Result{}, err
			}
		} else {
			// update with correct ports
			log.Infof("update virtual service %s/%s, because spec is out of sync", vService.Namespace, vService.Name)
			err := s.virtualClient.Update(ctx, newService)
			if err != nil {
				return ctrl.Result{}, err
			}
		}

		// we will requeue anyways
		return ctrl.Result{}, nil
	}

	// check if backwards status update is necessary
	if !equality.Semantic.DeepEqual(vService.Status, pService.Status) {
		newService := vService.DeepCopy()
		newService.Status = pService.Status
		log.Infof("update virtual service %s/%s, because status is out of sync", vService.Namespace, vService.Name)
		err := s.virtualClient.Status().Update(ctx, newService)
		if err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	// forward update
	return s.creator.Update(ctx, vObj, s.translateUpdate(pService, vService), log)
}

func isSwitchingFromExternalName(pService *corev1.Service, vService *corev1.Service) bool {
	return vService.Spec.Type == corev1.ServiceTypeExternalName && pService.Spec.Type != vService.Spec.Type && pService.Spec.ClusterIP != ""
}

var _ generic.BackwardSyncer = &syncer{}

func (s *syncer) Backward(ctx context.Context, pObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	if !translate.IsManaged(pObj) {
		return ctrl.Result{}, nil
	}

	// we have to delay deletion here if a vObj does not (yet) exist for a service that was just
	// created, because vcluster intercepts those calls and first creates a service inside the host
	// cluster and then inside the virtual cluster.
	//
	// We also don't need to care about the forwarding part deleting the physical object, because as soon as
	// that controller gets a delete event for a virtual service, we can safely delete the physical object.
	pService := pObj.(*corev1.Service)
	if pService.DeletionTimestamp == nil && pService.CreationTimestamp.Add(time.Second*180).After(time.Now()) {
		return ctrl.Result{RequeueAfter: time.Second * 10}, nil
	}

	return generic.DeleteObject(ctx, s.localClient, pObj, log)
}

func recreateService(ctx context.Context, virtualClient client.Client, vService *corev1.Service) (*corev1.Service, error) {
	// delete & create with correct ClusterIP
	err := virtualClient.Delete(ctx, vService)
	if err != nil && kerrors.IsNotFound(err) == false {
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

var _ generic.Starter = &syncer{}

func (s *syncer) ReconcileStart(ctx context.Context, req ctrl.Request) (bool, error) {
	// don't do anything for the kubernetes service
	if req.Name == "kubernetes" && req.Namespace == "default" {
		return true, SyncKubernetesService(ctx, s.virtualClient, s.currentNamespaceClient, s.currentNamespace, s.serviceName)
	}

	return false, nil
}

func (s *syncer) ReconcileEnd() {}

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
