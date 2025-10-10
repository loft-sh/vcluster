package services

import (
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
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	ServiceBlockDeletion             = "vcluster.loft.sh/block-deletion"
	RancherPublicEndpointsAnnotation = "field.cattle.io/publicEndpoints"
)

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
			RancherPublicEndpointsAnnotation,
		},

		serviceName: ctx.Config.Name,
	}, nil
}

type serviceSyncer struct {
	syncertypes.GenericTranslator
	syncertypes.Importer
	serviceName         string
	excludedAnnotations []string
}

var _ syncertypes.OptionsProvider = &serviceSyncer{}

func (s *serviceSyncer) Options() *syncertypes.Options {
	return &syncertypes.Options{
		DisableUIDDeletion: true,
		ObjectCaching:      true,
	}
}

var _ syncertypes.Syncer = &serviceSyncer{}

func (s *serviceSyncer) Syncer() syncertypes.Sync[client.Object] {
	return syncer.ToGenericSyncer(s)
}

func (s *serviceSyncer) SyncToHost(ctx *synccontext.SyncContext, event *synccontext.SyncToHostEvent[*corev1.Service]) (ctrl.Result, error) {
	if event.HostOld != nil || event.Virtual.DeletionTimestamp != nil {
		return patcher.DeleteVirtualObject(ctx, event.Virtual, event.HostOld, "host object was deleted")
	}

	pObj := s.translate(ctx, event.Virtual)
	err := pro.ApplyPatchesHostObject(ctx, nil, pObj, event.Virtual, ctx.Config.Sync.ToHost.Services.Patches, false)
	if err != nil {
		return ctrl.Result{}, err
	}

	return patcher.CreateHostObject(ctx, event.Virtual, pObj, s.EventRecorder(), true)
}

func (s *serviceSyncer) Sync(ctx *synccontext.SyncContext, event *synccontext.SyncEvent[*corev1.Service]) (_ ctrl.Result, retErr error) {
	// delay if we are in the middle of a switch operation
	if isSwitchingFromExternalName(event.Host, event.Virtual) {
		return ctrl.Result{RequeueAfter: time.Second * 3}, nil
	}

	// check if recreating service is necessary
	if event.Virtual.Spec.ClusterIP != event.Host.Spec.ClusterIP {
		ctx.Log.Infof("recreating virtual service %s/%s, because cluster ip differs %s != %s", event.Virtual.Namespace, event.Virtual.Name, event.Host.Spec.ClusterIP, event.Virtual.Spec.ClusterIP)
		event.Virtual.Spec.ClusterIPs = nil
		event.Virtual.Spec.ClusterIP = event.Host.Spec.ClusterIP

		// recreate the new service with the correct cluster ip
		err := recreateService(ctx, ctx.VirtualClient, event.Virtual)
		if err != nil {
			ctx.Log.Errorf("error creating virtual service: %s/%s", event.Virtual.Namespace, event.Virtual.Name)
			return ctrl.Result{}, err
		}

		return ctrl.Result{Requeue: true}, err
	}

	// patch the service
	patch, err := patcher.NewSyncerPatcher(ctx, event.Host, event.Virtual, patcher.TranslatePatches(ctx.Config.Sync.ToHost.Services.Patches, false))
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

	event.Virtual.Spec.Type, event.Host.Spec.Type = patcher.CopyBidirectional(
		event.VirtualOld.Spec.Type,
		event.Virtual.Spec.Type,
		event.HostOld.Spec.Type,
		event.Host.Spec.Type,
	)

	// update spec bidirectionally
	event.Virtual.Spec.ExternalIPs, event.Host.Spec.ExternalIPs = patcher.CopyBidirectional(
		event.VirtualOld.Spec.ExternalIPs,
		event.Virtual.Spec.ExternalIPs,
		event.HostOld.Spec.ExternalIPs,
		event.Host.Spec.ExternalIPs,
	)
	event.Virtual.Spec.ExternalName, event.Host.Spec.ExternalName = patcher.CopyBidirectional(
		event.VirtualOld.Spec.ExternalName,
		event.Virtual.Spec.ExternalName,
		event.HostOld.Spec.ExternalName,
		event.Host.Spec.ExternalName,
	)
	event.Virtual.Spec.LoadBalancerIP, event.Host.Spec.LoadBalancerIP = patcher.CopyBidirectional(
		event.VirtualOld.Spec.LoadBalancerIP,
		event.Virtual.Spec.LoadBalancerIP,
		event.HostOld.Spec.LoadBalancerIP,
		event.Host.Spec.LoadBalancerIP,
	)
	event.Virtual.Spec.Ports, event.Host.Spec.Ports = patcher.CopyBidirectional(
		event.VirtualOld.Spec.Ports,
		event.Virtual.Spec.Ports,
		event.HostOld.Spec.Ports,
		event.Host.Spec.Ports,
	)
	event.Virtual.Spec.PublishNotReadyAddresses, event.Host.Spec.PublishNotReadyAddresses = patcher.CopyBidirectional(
		event.VirtualOld.Spec.PublishNotReadyAddresses,
		event.Virtual.Spec.PublishNotReadyAddresses,
		event.HostOld.Spec.PublishNotReadyAddresses,
		event.Host.Spec.PublishNotReadyAddresses,
	)
	event.Virtual.Spec.ExternalTrafficPolicy, event.Host.Spec.ExternalTrafficPolicy = patcher.CopyBidirectional(
		event.VirtualOld.Spec.ExternalTrafficPolicy,
		event.Virtual.Spec.ExternalTrafficPolicy,
		event.HostOld.Spec.ExternalTrafficPolicy,
		event.Host.Spec.ExternalTrafficPolicy,
	)
	event.Virtual.Spec.SessionAffinity, event.Host.Spec.SessionAffinity = patcher.CopyBidirectional(
		event.VirtualOld.Spec.SessionAffinity,
		event.Virtual.Spec.SessionAffinity,
		event.HostOld.Spec.SessionAffinity,
		event.Host.Spec.SessionAffinity,
	)
	event.Virtual.Spec.SessionAffinityConfig, event.Host.Spec.SessionAffinityConfig = patcher.CopyBidirectional(
		event.VirtualOld.Spec.SessionAffinityConfig,
		event.Virtual.Spec.SessionAffinityConfig,
		event.HostOld.Spec.SessionAffinityConfig,
		event.Host.Spec.SessionAffinityConfig,
	)
	event.Virtual.Spec.LoadBalancerSourceRanges, event.Host.Spec.LoadBalancerSourceRanges = patcher.CopyBidirectional(
		event.VirtualOld.Spec.LoadBalancerSourceRanges,
		event.Virtual.Spec.LoadBalancerSourceRanges,
		event.HostOld.Spec.LoadBalancerSourceRanges,
		event.Host.Spec.LoadBalancerSourceRanges,
	)
	event.Virtual.Spec.HealthCheckNodePort, event.Host.Spec.HealthCheckNodePort = patcher.CopyBidirectional(
		event.VirtualOld.Spec.HealthCheckNodePort,
		event.Virtual.Spec.HealthCheckNodePort,
		event.HostOld.Spec.HealthCheckNodePort,
		event.Host.Spec.HealthCheckNodePort,
	)

	// update status
	ensureLoadBalancerStatus(event.Host)
	event.Virtual.Status = event.Host.Status

	// bi-directional sync of annotations and labels
	event.Virtual.Annotations, event.Host.Annotations = translate.AnnotationsBidirectionalUpdate(event, s.excludedAnnotations...)
	event.Virtual.Labels, event.Host.Labels = translate.LabelsBidirectionalUpdate(event)

	// remove the ServiceBlockDeletion annotation if it's not needed
	delete(event.Host.Annotations, ServiceBlockDeletion)

	// the logic here is that when the virtual object has changed we sync the labels to the host. If the host object has changed and the virtual object has not, we sync the labels to the virtual cluster.
	// If nothing has changed, we make sure to sync the labels from the virtual cluster to the host. This is necessary because earlier versions of vCluster did sync the labels differently and rewrote them
	// so we need to make sure those are always correctly synced.
	if !apiequality.Semantic.DeepEqual(event.VirtualOld.Spec.Selector, event.Virtual.Spec.Selector) || apiequality.Semantic.DeepEqual(event.HostOld.Spec.Selector, event.Host.Spec.Selector) {
		event.Host.Spec.Selector = translate.HostLabelsMap(event.Virtual.Spec.Selector, event.Host.Spec.Selector, event.Virtual.Namespace, false)
	} else {
		event.Virtual.Spec.Selector = translate.VirtualLabelsMap(event.Host.Spec.Selector, event.Virtual.Spec.Selector)
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
	if event.Host.Annotations != nil && event.Host.Annotations[ServiceBlockDeletion] == "true" && time.Since(event.Host.CreationTimestamp.Time) < 2*time.Minute {
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	if event.VirtualOld != nil || translate.ShouldDeleteHostObject(event.Host) {
		return patcher.DeleteHostObject(ctx, event.Host, event.VirtualOld, "virtual object was deleted")
	}

	vObj := s.translateToVirtual(ctx, event.Host)
	err := pro.ApplyPatchesVirtualObject(ctx, nil, vObj, event.Host, ctx.Config.Sync.ToHost.Services.Patches, false)
	if err != nil {
		return ctrl.Result{}, err
	}

	return patcher.CreateVirtualObject(ctx, event.Host, vObj, s.EventRecorder(), true)
}

func recreateService(ctx *synccontext.SyncContext, virtualClient client.Client, vService *corev1.Service) error {
	// delete & create with correct ClusterIP
	err := virtualClient.Delete(ctx, vService)
	if err != nil && !kerrors.IsNotFound(err) {
		klog.Errorf("error deleting virtual service: %s/%s: %v", vService.Namespace, vService.Name, err)
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

// ensureLoadBalancerStatus removes any LoadBalancer-related fields from the Service
// if it is of type ClusterIP.
//
// This is necessary to ensure consistency when syncing services from a virtual
// cluster to the host cluster. ClusterIP services should not carry LoadBalancer
// settings such as LoadBalancerIP or Status.LoadBalancer.Ingress, which are
// specific to LoadBalancer-type services and can cause incorrect behavior if retained.
func ensureLoadBalancerStatus(pObj *corev1.Service) {
	if pObj.Spec.Type != corev1.ServiceTypeLoadBalancer {
		pObj.Spec.LoadBalancerIP = ""
		pObj.Status.LoadBalancer.Ingress = make([]corev1.LoadBalancerIngress, 0)
	}
}
