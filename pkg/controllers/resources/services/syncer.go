package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/patcher"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/specialservices"
	"github.com/loft-sh/vcluster/pkg/syncer"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/syncer/translator"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var ServiceBlockDeletion = "vcluster.loft.sh/block-deletion"

func New(ctx *synccontext.RegisterContext) (syncertypes.Object, error) {
	mapper, err := ctx.Mappings.ByGVK(mappings.Services())
	if err != nil {
		return nil, err
	}

	return &serviceSyncer{
		// exclude "field.cattle.io/publicEndpoints" annotation used by Rancher,
		// because if it is also installed in the host cluster, it will be
		// overriding it, which would cause endless updates back and forth.
		GenericTranslator: translator.NewGenericTranslator(ctx, "service", &corev1.Service{}, mapper),
		Importer:          pro.NewImporter(mapper),

		excludedAnnotations: []string{
			"field.cattle.io/publicEndpoints",
		},

		serviceName: ctx.Config.WorkloadService,
	}, nil
}

type serviceSyncer struct {
	syncertypes.GenericTranslator
	syncertypes.Importer

	excludedAnnotations []string

	serviceName string
}

var _ syncertypes.OptionsProvider = &serviceSyncer{}

func (s *serviceSyncer) Options() *syncertypes.Options {
	return &syncertypes.Options{
		DisableUIDDeletion: true,
	}
}

var _ syncertypes.Syncer = &serviceSyncer{}

func (s *serviceSyncer) Syncer() syncertypes.Sync[client.Object] {
	return syncer.ToGenericSyncer[*corev1.Service](s)
}

func (s *serviceSyncer) SyncToHost(ctx *synccontext.SyncContext, event *synccontext.SyncToHostEvent[*corev1.Service]) (ctrl.Result, error) {
	if event.IsDelete() || event.Virtual.DeletionTimestamp != nil {
		return syncer.DeleteVirtualObject(ctx, event.Virtual, "host object was deleted")
	}

	pObj := s.translate(ctx, event.Virtual)
	err := pro.ApplyPatchesHostObject(ctx, nil, pObj, event.Virtual, ctx.Config.Sync.ToHost.Services.Translate)
	if err != nil {
		return ctrl.Result{}, err
	}

	return syncer.CreateHostObject(ctx, event.Virtual, pObj, s.EventRecorder())
}

func (s *serviceSyncer) Sync(ctx *synccontext.SyncContext, event *synccontext.SyncEvent[*corev1.Service]) (_ ctrl.Result, retErr error) {
	// delay if we are in the middle of a switch operation
	if isSwitchingFromExternalName(event.Host, event.Virtual) {
		return ctrl.Result{RequeueAfter: time.Second * 3}, nil
	}

	// check if recreating service is necessary
	if event.Virtual.Spec.ClusterIP != event.Host.Spec.ClusterIP {
		event.Virtual.Spec.ClusterIPs = nil
		event.Virtual.Spec.ClusterIP = event.Host.Spec.ClusterIP
		ctx.Log.Infof("recreating virtual service %s/%s, because cluster ip differs %s != %s", event.Virtual.Namespace, event.Virtual.Name, event.Host.Spec.ClusterIP, event.Virtual.Spec.ClusterIP)

		// recreate the new service with the correct cluster ip
		err := recreateService(ctx, ctx.VirtualClient, event.Virtual)
		if err != nil {
			ctx.Log.Errorf("error creating virtual service: %s/%s", event.Virtual.Namespace, event.Virtual.Name)
			return ctrl.Result{}, err
		}

		return ctrl.Result{Requeue: true}, err
	}

	// patch the service
	patch, err := patcher.NewSyncerPatcher(ctx, event.Host, event.Virtual, patcher.TranslatePatches(ctx.Config.Sync.ToHost.Services.Translate))
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("new syncer patcher: %w", err)
	}
	defer func() {
		if err := patch.Patch(ctx, event.Host, event.Virtual); err != nil {
			retErr = utilerrors.NewAggregate([]error{retErr, err})
		}
		if retErr != nil {
			s.EventRecorder().Eventf(event.Virtual, "Warning", "SyncError", "Error syncing: %v", retErr)
		}
	}()

	// update spec bidirectionally
	event.TargetObject().Spec.ExternalIPs = event.SourceObject().Spec.ExternalIPs
	event.TargetObject().Spec.LoadBalancerIP = event.SourceObject().Spec.LoadBalancerIP
	event.TargetObject().Spec.Ports = event.SourceObject().Spec.Ports
	event.TargetObject().Spec.PublishNotReadyAddresses = event.SourceObject().Spec.PublishNotReadyAddresses
	event.TargetObject().Spec.Type = event.SourceObject().Spec.Type
	event.TargetObject().Spec.ExternalName = event.SourceObject().Spec.ExternalName
	event.TargetObject().Spec.ExternalTrafficPolicy = event.SourceObject().Spec.ExternalTrafficPolicy
	event.TargetObject().Spec.SessionAffinity = event.SourceObject().Spec.SessionAffinity
	event.TargetObject().Spec.SessionAffinityConfig = event.SourceObject().Spec.SessionAffinityConfig
	event.TargetObject().Spec.LoadBalancerSourceRanges = event.SourceObject().Spec.LoadBalancerSourceRanges
	event.TargetObject().Spec.HealthCheckNodePort = event.SourceObject().Spec.HealthCheckNodePort

	// update status
	event.Virtual.Status = event.Host.Status

	// check labels
	if event.Source == synccontext.SyncEventSourceHost {
		event.Virtual.Labels = translate.VirtualLabels(event.Host, event.Virtual)
	} else {
		event.Host.Labels = translate.HostLabels(event.Virtual, event.Host)
	}

	// check annotations
	updatedAnnotations := translate.HostAnnotations(event.Virtual, event.Host, s.excludedAnnotations...)

	// remove the ServiceBlockDeletion annotation if it's not needed
	if event.Virtual.Spec.ClusterIP == event.Host.Spec.ClusterIP {
		delete(updatedAnnotations, ServiceBlockDeletion)
	}
	event.Host.Annotations = updatedAnnotations

	// translate selector
	if event.Source == synccontext.SyncEventSourceHost {
		event.Virtual.Spec.Selector = translate.VirtualLabelsMap(event.Host.Spec.Selector, event.Virtual.Spec.Selector)
	} else {
		event.Host.Spec.Selector = translate.HostLabelsMap(event.Virtual.Spec.Selector, event.Host.Spec.Selector, event.Virtual.Namespace, false)
	}

	return ctrl.Result{}, nil
}

func isSwitchingFromExternalName(pService *corev1.Service, vService *corev1.Service) bool {
	return vService.Spec.Type == corev1.ServiceTypeExternalName && pService.Spec.Type != vService.Spec.Type && pService.Spec.ClusterIP != ""
}

func (s *serviceSyncer) SyncToVirtual(ctx *synccontext.SyncContext, event *synccontext.SyncToVirtualEvent[*corev1.Service]) (ctrl.Result, error) {
	// we have to delay deletion here if a vObj does not (yet) exist for a service that was just
	// created, because vcluster intercepts those calls and first creates a service inside the host
	// cluster and then inside the virtual cluster.
	if event.Host.Annotations != nil && event.Host.Annotations[ServiceBlockDeletion] == "true" {
		return ctrl.Result{Requeue: true}, nil
	}

	if event.IsDelete() || event.Host.DeletionTimestamp != nil {
		return syncer.DeleteHostObject(ctx, event.Host, "virtual object was deleted")
	}

	vObj := s.translateToVirtual(ctx, event.Host)
	err := pro.ApplyPatchesVirtualObject(ctx, nil, vObj, event.Host, ctx.Config.Sync.ToHost.Services.Translate)
	if err != nil {
		return ctrl.Result{}, err
	}

	return syncer.CreateVirtualObject(ctx, event.Host, vObj, s.EventRecorder())
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

var _ syncertypes.Starter = &serviceSyncer{}

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
