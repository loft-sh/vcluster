package configmaps

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
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func New(ctx *synccontext.RegisterContext) (syncertypes.Object, error) {
	mapper, err := ctx.Mappings.ByGVK(mappings.ConfigMaps())
	if err != nil {
		return nil, err
	}

	// only add importer if all configmaps should get synced
	importer := syncer.NewNoopImporter()
	if ctx.Config.Sync.ToHost.ConfigMaps.All {
		importer = pro.NewImporter(mapper)
	}

	return &configMapSyncer{
		GenericTranslator: translator.NewGenericTranslator(ctx, "configmap", &corev1.ConfigMap{}, mapper),
		Importer:          importer,
	}, nil
}

type configMapSyncer struct {
	syncertypes.GenericTranslator
	syncertypes.Importer
}

var _ syncertypes.OptionsProvider = &configMapSyncer{}

func (s *configMapSyncer) Options() *syncertypes.Options {
	return &syncertypes.Options{
		ObjectCaching: true,
	}
}

var _ syncertypes.Syncer = &configMapSyncer{}

func (s *configMapSyncer) Syncer() syncertypes.Sync[client.Object] {
	return syncer.ToGenericSyncer(s)
}

var _ syncertypes.ControllerModifier = &configMapSyncer{}

func (s *configMapSyncer) ModifyController(ctx *synccontext.RegisterContext, builder *builder.Builder) (*builder.Builder, error) {
	return builder.WatchesRawSource(ctx.Mappings.Store().Watch(s.GroupVersionKind(), func(nameMapping synccontext.NameMapping, queue workqueue.TypedRateLimitingInterface[ctrl.Request]) {
		queue.Add(reconcile.Request{NamespacedName: nameMapping.VirtualName})
	})), nil
}

func (s *configMapSyncer) SyncToHost(ctx *synccontext.SyncContext, event *synccontext.SyncToHostEvent[*corev1.ConfigMap]) (ctrl.Result, error) {
	createNeeded := s.isConfigMapUsed(ctx, event.Virtual)
	if !createNeeded {
		return ctrl.Result{}, nil
	}

	if event.HostOld != nil || event.Virtual.DeletionTimestamp != nil {
		return patcher.DeleteVirtualObject(ctx, event.Virtual, event.HostOld, "host object was deleted")
	}

	pObj := translate.HostMetadata(event.Virtual, s.VirtualToHost(ctx, types.NamespacedName{Name: event.Virtual.Name, Namespace: event.Virtual.Namespace}, event.Virtual))
	err := pro.ApplyPatchesHostObject(ctx, nil, pObj, event.Virtual, ctx.Config.Sync.ToHost.ConfigMaps.Patches, false)
	if err != nil {
		return ctrl.Result{}, err
	}

	return patcher.CreateHostObject(ctx, event.Virtual, pObj, s.EventRecorder(), false)
}

func (s *configMapSyncer) SyncToVirtual(ctx *synccontext.SyncContext, event *synccontext.SyncToVirtualEvent[*corev1.ConfigMap]) (_ ctrl.Result, retErr error) {
	if event.VirtualOld != nil || translate.ShouldDeleteHostObject(event.Host) {
		return patcher.DeleteHostObject(ctx, event.Host, event.VirtualOld, "virtual object was deleted")
	}

	vObj := translate.VirtualMetadata(event.Host, s.HostToVirtual(ctx, types.NamespacedName{Name: event.Host.Name, Namespace: event.Host.Namespace}, event.Host))
	createNeeded := s.isConfigMapUsed(ctx, vObj)
	if !createNeeded {
		return ctrl.Result{}, nil
	}

	err := pro.ApplyPatchesVirtualObject(ctx, nil, vObj, event.Host, ctx.Config.Sync.ToHost.ConfigMaps.Patches, false)
	if err != nil {
		return ctrl.Result{}, err
	}

	return patcher.CreateVirtualObject(ctx, event.Host, vObj, s.EventRecorder(), false)
}

func (s *configMapSyncer) Sync(ctx *synccontext.SyncContext, event *synccontext.SyncEvent[*corev1.ConfigMap]) (_ ctrl.Result, retErr error) {
	used := s.isConfigMapUsed(ctx, event.Virtual)
	if !used {
		// virtual object is not here anymore, so we delete
		return patcher.DeleteHostObject(ctx, event.Host, event.Virtual, "configmap is not used anymore")
	}

	patchHelper, err := patcher.NewSyncerPatcher(ctx, event.Host, event.Virtual, patcher.TranslatePatches(ctx.Config.Sync.ToHost.ConfigMaps.Patches, false))
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("new syncer patcher: %w", err)
	}

	defer func() {
		if err := patchHelper.Patch(ctx, event.Host, event.Virtual); err != nil {
			retErr = utilerrors.NewAggregate([]error{retErr, err})
		}
		if retErr != nil {
			s.EventRecorder().Eventf(event.Virtual, "Warning", "SyncError", "Error syncing: %v", retErr)
		}
	}()

	// bi-directional sync of annotations and labels
	event.Virtual.Annotations, event.Host.Annotations = translate.AnnotationsBidirectionalUpdate(event)
	event.Virtual.Labels, event.Host.Labels = translate.LabelsBidirectionalUpdate(event)

	// bidirectional sync
	event.Virtual.Data, event.Host.Data, err = patcher.MergeBidirectional(event.VirtualOld.Data, event.Virtual.Data, event.HostOld.Data, event.Host.Data)
	if err != nil {
		return ctrl.Result{}, err
	}
	event.Virtual.BinaryData, event.Host.BinaryData, err = patcher.MergeBidirectional(event.VirtualOld.BinaryData, event.Virtual.BinaryData, event.HostOld.BinaryData, event.Host.BinaryData)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (s *configMapSyncer) isConfigMapUsed(ctx *synccontext.SyncContext, vObj *corev1.ConfigMap) bool {
	if vObj.Annotations[constants.SyncResourceAnnotation] == "true" {
		return true
	} else if ctx.Config.Sync.ToHost.ConfigMaps.All {
		return true
	}

	// retrieve references for config map
	references := ctx.Mappings.Store().ReferencesTo(ctx, synccontext.Object{
		GroupVersionKind: s.GroupVersionKind(),
		NamespacedName: types.NamespacedName{
			Namespace: vObj.Namespace,
			Name:      vObj.Name,
		},
	})

	return len(references) > 0
}
