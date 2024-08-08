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

var _ syncertypes.Syncer = &configMapSyncer{}

func (s *configMapSyncer) Syncer() syncertypes.Sync[client.Object] {
	return syncer.ToGenericSyncer[*corev1.ConfigMap](s)
}

var _ syncertypes.ControllerModifier = &configMapSyncer{}

func (s *configMapSyncer) ModifyController(ctx *synccontext.RegisterContext, builder *builder.Builder) (*builder.Builder, error) {
	return builder.WatchesRawSource(ctx.Mappings.Store().Watch(s.GroupVersionKind(), func(nameMapping synccontext.NameMapping, queue workqueue.RateLimitingInterface) {
		queue.Add(reconcile.Request{NamespacedName: nameMapping.VirtualName})
	})), nil
}

func (s *configMapSyncer) SyncToHost(ctx *synccontext.SyncContext, event *synccontext.SyncToHostEvent[*corev1.ConfigMap]) (ctrl.Result, error) {
	createNeeded, err := s.isConfigMapUsed(ctx, event.Virtual)
	if err != nil {
		return ctrl.Result{}, err
	} else if !createNeeded {
		return ctrl.Result{}, nil
	}

	if event.IsDelete() || event.Virtual.DeletionTimestamp != nil {
		return syncer.DeleteVirtualObject(ctx, event.Virtual, "host object was deleted")
	}

	pObj := translate.HostMetadata(event.Virtual, s.VirtualToHost(ctx, types.NamespacedName{Name: event.Virtual.Name, Namespace: event.Virtual.Namespace}, event.Virtual))
	err = pro.ApplyPatchesHostObject(ctx, nil, pObj, event.Virtual, ctx.Config.Sync.ToHost.ConfigMaps.Translate)
	if err != nil {
		return ctrl.Result{}, err
	}

	return syncer.CreateHostObject(ctx, event.Virtual, pObj, s.EventRecorder())
}

func (s *configMapSyncer) SyncToVirtual(ctx *synccontext.SyncContext, event *synccontext.SyncToVirtualEvent[*corev1.ConfigMap]) (_ ctrl.Result, retErr error) {
	if event.IsDelete() || event.Host.DeletionTimestamp != nil {
		// virtual object is not here anymore, so we delete
		return syncer.DeleteHostObject(ctx, event.Host, "virtual object was deleted")
	}

	vObj := translate.VirtualMetadata(event.Host, s.HostToVirtual(ctx, types.NamespacedName{Name: event.Host.Name, Namespace: event.Host.Namespace}, event.Host))
	createNeeded, err := s.isConfigMapUsed(ctx, vObj)
	if err != nil {
		return ctrl.Result{}, err
	} else if !createNeeded {
		return ctrl.Result{}, nil
	}

	err = pro.ApplyPatchesVirtualObject(ctx, nil, vObj, event.Host, ctx.Config.Sync.ToHost.ConfigMaps.Translate)
	if err != nil {
		return ctrl.Result{}, err
	}

	return syncer.CreateVirtualObject(ctx, event.Host, vObj, s.EventRecorder())
}

func (s *configMapSyncer) Sync(ctx *synccontext.SyncContext, event *synccontext.SyncEvent[*corev1.ConfigMap]) (_ ctrl.Result, retErr error) {
	used, err := s.isConfigMapUsed(ctx, event.Virtual)
	if err != nil {
		return ctrl.Result{}, err
	} else if !used {
		ctx.Log.Infof("delete physical config map %s/%s, because it is not used anymore", event.Host.GetNamespace(), event.Host.GetName())
		err = ctx.PhysicalClient.Delete(ctx, event.Host)
		if err != nil {
			ctx.Log.Infof("error deleting physical object %s/%s in physical cluster: %v", event.Host.GetNamespace(), event.Host.GetName(), err)
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	patch, err := patcher.NewSyncerPatcher(ctx, event.Host, event.Virtual, patcher.TranslatePatches(ctx.Config.Sync.ToHost.ConfigMaps.Translate))
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

	// check annotations & labels
	event.Host.Annotations = translate.HostAnnotations(event.Virtual, event.Host)

	// check labels
	if event.Source == synccontext.SyncEventSourceHost {
		event.Virtual.Labels = translate.VirtualLabels(event.Host, event.Virtual)
	} else {
		event.Host.Labels = translate.HostLabels(event.Virtual, event.Host)
	}

	// bidirectional sync
	event.TargetObject().Data = event.SourceObject().Data
	event.TargetObject().BinaryData = event.SourceObject().BinaryData
	return ctrl.Result{}, nil
}

func (s *configMapSyncer) isConfigMapUsed(ctx *synccontext.SyncContext, vObj *corev1.ConfigMap) (bool, error) {
	if vObj.Annotations[constants.SyncResourceAnnotation] == "true" {
		return true, nil
	} else if ctx.Config.Sync.ToHost.ConfigMaps.All {
		return true, nil
	}

	// retrieve references for config map
	references := ctx.Mappings.Store().ReferencesTo(ctx, synccontext.Object{
		GroupVersionKind: s.GroupVersionKind(),
		NamespacedName: types.NamespacedName{
			Namespace: vObj.Namespace,
			Name:      vObj.Name,
		},
	})

	return len(references) > 0, nil
}
