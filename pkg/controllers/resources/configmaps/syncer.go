package configmaps

import (
	"fmt"

	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/patcher"
	"github.com/loft-sh/vcluster/pkg/syncer"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/syncer/translator"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
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

	return &configMapSyncer{
		GenericTranslator: translator.NewGenericTranslator(ctx, "configmap", &corev1.ConfigMap{}, mapper),

		syncAllConfigMaps: ctx.Config.Sync.ToHost.ConfigMaps.All,
	}, nil
}

type configMapSyncer struct {
	syncertypes.GenericTranslator

	syncAllConfigMaps bool
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

	if event.IsDelete() {
		return syncer.DeleteVirtualObject(ctx, event.Virtual, "host object was deleted")
	}

	pObj := translate.HostMetadata(ctx, event.Virtual, s.VirtualToHost(ctx, types.NamespacedName{Name: event.Virtual.Name, Namespace: event.Virtual.Namespace}, event.Virtual))
	return syncer.CreateHostObject(ctx, event.Virtual, pObj, s.EventRecorder())
}

func (s *configMapSyncer) SyncToVirtual(ctx *synccontext.SyncContext, event *synccontext.SyncToVirtualEvent[*corev1.ConfigMap]) (_ ctrl.Result, retErr error) {
	// virtual object is not here anymore, so we delete
	return syncer.DeleteHostObject(ctx, event.Host, "virtual object was deleted")
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

	patch, err := patcher.NewSyncerPatcher(ctx, event.Host, event.Virtual)
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
	event.Host.Labels = translate.HostLabels(ctx, event.Virtual, event.Host)

	// bidirectional sync
	event.TargetObject().Data = event.SourceObject().Data
	event.TargetObject().BinaryData = event.SourceObject().BinaryData
	return ctrl.Result{}, nil
}

func (s *configMapSyncer) isConfigMapUsed(ctx *synccontext.SyncContext, vObj runtime.Object) (bool, error) {
	configMap, ok := vObj.(*corev1.ConfigMap)
	if !ok || configMap == nil {
		return false, fmt.Errorf("%#v is not a config map", vObj)
	} else if configMap.Annotations != nil && configMap.Annotations[constants.SyncResourceAnnotation] == "true" {
		return true, nil
	} else if s.syncAllConfigMaps {
		return true, nil
	}

	// retrieve references for config map
	references := ctx.Mappings.Store().ReferencesTo(ctx, synccontext.Object{
		GroupVersionKind: s.GroupVersionKind(),
		NamespacedName: types.NamespacedName{
			Namespace: configMap.Namespace,
			Name:      configMap.Name,
		},
	})

	return len(references) > 0, nil
}
