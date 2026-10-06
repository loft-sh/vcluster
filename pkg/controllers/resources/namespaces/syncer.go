package namespaces

import (
	"fmt"

	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/patcher"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/syncer"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/syncer/translator"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// Unsafe annotations based on the docs here:
// https://kubernetes.io/docs/reference/labels-annotations-taints/
var excludedAnnotations = []string{
	"scheduler.alpha.kubernetes.io/node-selector",
	"scheduler.alpha.kubernetes.io/defaultTolerations",
}

func New(ctx *synccontext.RegisterContext) (syncertypes.Object, error) {
	mapper, err := ctx.Mappings.ByGVK(mappings.Namespaces())
	if err != nil {
		return nil, err
	}

	namespaceLabels := map[string]string{}
	for k, v := range ctx.Config.Sync.ToHost.Namespaces.ExtraLabels {
		namespaceLabels[k] = v
	}
	namespaceLabels[constants.VClusterNameAnnotation] = ctx.Config.Name
	namespaceLabels[constants.VClusterNamespaceAnnotation] = ctx.CurrentNamespace

	return &namespaceSyncer{
		GenericTranslator:          translator.NewGenericTranslator(ctx, "namespace", &corev1.Namespace{}, mapper),
		workloadServiceAccountName: ctx.Config.ControlPlane.Advanced.WorkloadServiceAccount.Name,
		Importer:                   pro.NewImporter(mapper),
		excludedAnnotations:        excludedAnnotations,

		namespaceLabels: namespaceLabels,
	}, nil
}

type namespaceSyncer struct {
	syncertypes.GenericTranslator
	syncertypes.Importer

	namespaceLabels            map[string]string
	workloadServiceAccountName string
	excludedAnnotations        []string
}

var _ syncertypes.Syncer = &namespaceSyncer{}
var _ syncertypes.OptionsProvider = &namespaceSyncer{}

func (s *namespaceSyncer) Options() *syncertypes.Options {
	return &syncertypes.Options{
		ObjectCaching:      true,
		DisableUIDDeletion: true,
	}
}

func (s *namespaceSyncer) Syncer() syncertypes.Sync[client.Object] {
	return syncer.ToGenericSyncer(s)
}

func (s *namespaceSyncer) SyncToHost(ctx *synccontext.SyncContext, event *synccontext.SyncToHostEvent[*corev1.Namespace]) (ctrl.Result, error) {
	if event.HostOld != nil || event.Virtual.DeletionTimestamp != nil {
		return patcher.DeleteVirtualObject(ctx, event.Virtual, event.HostOld, "host object was deleted")
	}

	newNamespace := s.translateToHost(ctx, event.Virtual)
	ctx.Log.Infof("create physical namespace %s", newNamespace.Name)

	err := pro.ApplyPatchesHostObject(ctx, nil, newNamespace, event.Virtual, ctx.Config.Sync.ToHost.Namespaces.Patches, false)
	if err != nil {
		return ctrl.Result{}, err
	}

	return patcher.CreateHostObject(ctx, event.Virtual, newNamespace, s.EventRecorder(), true)
}

func (s *namespaceSyncer) Sync(ctx *synccontext.SyncContext, event *synccontext.SyncEvent[*corev1.Namespace]) (_ ctrl.Result, retErr error) {
	patch, err := patcher.NewSyncerPatcher(ctx, event.Host, event.Virtual)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("new syncer patcher: %w", err)
	}

	defer func() {
		if err := patch.Patch(ctx, event.Host, event.Virtual); err != nil {
			retErr = utilerrors.NewAggregate([]error{retErr, err})
		}
	}()

	s.translateUpdate(event.Host, event.Virtual)
	return ctrl.Result{}, s.EnsureWorkloadServiceAccount(ctx, event.Host.Name)
}

func (s *namespaceSyncer) SyncToVirtual(ctx *synccontext.SyncContext, event *synccontext.SyncToVirtualEvent[*corev1.Namespace]) (_ ctrl.Result, retErr error) {
	// virtual object is not here anymore, so we delete
	if event.VirtualOld != nil || event.Host.DeletionTimestamp != nil {
		// first, lets check if host object was imported - if so, we don't delete it
		if event.Host.Annotations != nil && event.Host.Annotations[translate.ImportedMarkerAnnotation] == "true" {
			ctx.Log.Infof("host object %s/%s was imported, not deleting it", event.Host.Namespace, event.Host.Name)
			return ctrl.Result{}, nil
		}

		return patcher.DeleteHostObject(ctx, event.Host, event.VirtualOld, "virtual object was deleted")
	}

	// add marker annotation to host object and update it
	_, err := controllerutil.CreateOrPatch(ctx, ctx.HostClient, event.Host, func() error {
		if event.Host.Annotations == nil {
			event.Host.Annotations = map[string]string{}
		}
		event.Host.Annotations[translate.ImportedMarkerAnnotation] = "true"
		return nil
	})
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("create or patch host object: %w", err)
	}

	newNamespace := s.translateToVirtual(ctx, event.Host)
	ctx.Log.Infof("create virtual namespace %s", newNamespace.Name)

	err = pro.ApplyPatchesVirtualObject(ctx, nil, newNamespace, event.Host, ctx.Config.Sync.ToHost.Namespaces.Patches, false)
	if err != nil {
		return ctrl.Result{}, err
	}
	return patcher.CreateVirtualObject(ctx, event.Host, newNamespace, s.EventRecorder(), true)
}

func (s *namespaceSyncer) EnsureWorkloadServiceAccount(ctx *synccontext.SyncContext, pNamespace string) error {
	if s.workloadServiceAccountName == "" {
		return nil
	}

	svc := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: pNamespace,
			Name:      s.workloadServiceAccountName,
		},
	}
	_, err := controllerutil.CreateOrPatch(ctx, ctx.HostClient, svc, func() error { return nil })
	return err
}
