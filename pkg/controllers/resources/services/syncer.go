package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/patcher"
	"github.com/loft-sh/vcluster/pkg/specialservices"
	"github.com/loft-sh/vcluster/pkg/syncer"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/syncer/translator"
	"github.com/loft-sh/vcluster/pkg/syncer/types"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var ServiceBlockDeletion = "vcluster.loft.sh/block-deletion"

func New(ctx *synccontext.RegisterContext) (types.Object, error) {
	mapper, err := ctx.Mappings.ByGVK(mappings.Services())
	if err != nil {
		return nil, err
	}

	return &serviceSyncer{
		// exclude "field.cattle.io/publicEndpoints" annotation used by Rancher,
		// because if it is also installed in the host cluster, it will be
		// overriding it, which would cause endless updates back and forth.
		GenericTranslator: translator.NewGenericTranslator(ctx, "service", &corev1.Service{}, mapper),

		excludedAnnotations: []string{
			"field.cattle.io/publicEndpoints",
		},

		serviceName: ctx.Config.WorkloadService,
	}, nil
}

type serviceSyncer struct {
	types.GenericTranslator

	excludedAnnotations []string

	serviceName string
}

var _ types.OptionsProvider = &serviceSyncer{}

func (s *serviceSyncer) Options() *types.Options {
	return &types.Options{
		DisableUIDDeletion: true,
	}
}

func (s *serviceSyncer) SyncToHost(ctx *synccontext.SyncContext, vObj client.Object) (ctrl.Result, error) {
	if ctx.IsDelete {
		return syncer.DeleteVirtualObject(ctx, vObj, "host object was deleted")
	}

	return syncer.CreateHostObject(ctx, vObj, s.translate(ctx, vObj.(*corev1.Service)), s.EventRecorder())
}

func (s *serviceSyncer) Sync(ctx *synccontext.SyncContext, pObj client.Object, vObj client.Object) (_ ctrl.Result, retErr error) {
	vService := vObj.(*corev1.Service)
	pService := pObj.(*corev1.Service)

	// delay if we are in the middle of a switch operation
	if isSwitchingFromExternalName(pService, vService) {
		return ctrl.Result{RequeueAfter: time.Second * 3}, nil
	}

	// check if recreating service is necessary
	if vService.Spec.ClusterIP != pService.Spec.ClusterIP {
		vService.Spec.ClusterIPs = nil
		vService.Spec.ClusterIP = pService.Spec.ClusterIP
		ctx.Log.Infof("recreating virtual service %s/%s, because cluster ip differs %s != %s", vService.Namespace, vService.Name, pService.Spec.ClusterIP, vService.Spec.ClusterIP)

		// recreate the new service with the correct cluster ip
		err := recreateService(ctx, ctx.VirtualClient, vService)
		if err != nil {
			ctx.Log.Errorf("error creating virtual service: %s/%s", vService.Namespace, vService.Name)
			return ctrl.Result{}, err
		}

		return ctrl.Result{Requeue: true}, err
	}

	// patch the service
	patch, err := patcher.NewSyncerPatcher(ctx, pService, vService)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("new syncer patcher: %w", err)
	}
	defer func() {
		if err := patch.Patch(ctx, pService, vService); err != nil {
			retErr = utilerrors.NewAggregate([]error{retErr, err})
		}
		if retErr != nil {
			s.EventRecorder().Eventf(vObj, "Warning", "SyncError", "Error syncing: %v", retErr)
		}
	}()

	// check
	sourceService, targetService := synccontext.SyncSourceTarget(ctx, pService, vService)

	// update spec bidirectionally
	targetService.Spec.ExternalIPs = sourceService.Spec.ExternalIPs
	targetService.Spec.LoadBalancerIP = sourceService.Spec.LoadBalancerIP
	targetService.Spec.Ports = sourceService.Spec.Ports
	targetService.Spec.PublishNotReadyAddresses = sourceService.Spec.PublishNotReadyAddresses
	targetService.Spec.Type = sourceService.Spec.Type
	targetService.Spec.ExternalName = sourceService.Spec.ExternalName
	targetService.Spec.ExternalTrafficPolicy = sourceService.Spec.ExternalTrafficPolicy
	targetService.Spec.SessionAffinity = sourceService.Spec.SessionAffinity
	targetService.Spec.SessionAffinityConfig = sourceService.Spec.SessionAffinityConfig
	targetService.Spec.LoadBalancerSourceRanges = sourceService.Spec.LoadBalancerSourceRanges
	targetService.Spec.HealthCheckNodePort = sourceService.Spec.HealthCheckNodePort

	// update status
	vService.Status = pService.Status

	// check annotations & labels
	pService.Labels = translate.HostLabels(ctx, vObj, pObj)
	updatedAnnotations := translate.HostAnnotations(vObj, pObj, s.excludedAnnotations...)

	// remove the ServiceBlockDeletion annotation if it's not needed
	if vService.Spec.ClusterIP == pService.Spec.ClusterIP {
		delete(updatedAnnotations, ServiceBlockDeletion)
	}
	pService.Annotations = updatedAnnotations

	// translate selector
	pService.Spec.Selector = translate.Default.HostLabels(vService.Spec.Selector, pService.Spec.Selector, vService.Namespace, nil)
	return ctrl.Result{}, nil
}

func isSwitchingFromExternalName(pService *corev1.Service, vService *corev1.Service) bool {
	return vService.Spec.Type == corev1.ServiceTypeExternalName && pService.Spec.Type != vService.Spec.Type && pService.Spec.ClusterIP != ""
}

var _ types.ToVirtualSyncer = &serviceSyncer{}

func (s *serviceSyncer) SyncToVirtual(ctx *synccontext.SyncContext, pObj client.Object) (ctrl.Result, error) {
	isManaged, err := s.IsManaged(ctx, pObj)
	if err != nil {
		return ctrl.Result{}, err
	} else if !isManaged {
		return ctrl.Result{}, nil
	}

	// we have to delay deletion here if a vObj does not (yet) exist for a service that was just
	// created, because vcluster intercepts those calls and first creates a service inside the host
	// cluster and then inside the virtual cluster.
	pService := pObj.(*corev1.Service)
	if pService.Annotations != nil && pService.Annotations[ServiceBlockDeletion] == "true" {
		return ctrl.Result{Requeue: true}, nil
	}

	return syncer.DeleteHostObject(ctx, pObj, "virtual object was deleted")
}

func recreateService(ctx context.Context, virtualClient client.Client, vService *corev1.Service) error {
	// delete & create with correct ClusterIP
	err := virtualClient.Delete(ctx, vService)
	if err != nil && !kerrors.IsNotFound(err) {
		return err
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
		return err
	}

	return nil
}

var _ types.Starter = &serviceSyncer{}

func (s *serviceSyncer) ReconcileStart(ctx *synccontext.SyncContext, req ctrl.Request) (bool, error) {
	// don't do anything for the kubernetes service
	if specialservices.Default == nil {
		return false, errors.New("specialservices default not initialized yet")
	}

	specialServices := specialservices.Default.SpecialServicesToSync()
	if svc, ok := specialServices[req.NamespacedName]; ok {
		return true, svc(ctx, ctx.CurrentNamespace, s.serviceName, req.NamespacedName, TranslateServicePorts)
	}

	return false, nil
}

func (s *serviceSyncer) ReconcileEnd() {}

func TranslateServicePorts(ports []corev1.ServicePort) []corev1.ServicePort {
	retPorts := []corev1.ServicePort{}
	for _, p := range ports {
		if p.Name == "kubelet" {
			continue
		}

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
