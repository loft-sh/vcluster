package generic

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/loft-sh/vcluster/pkg/mappings/generic"
	"github.com/loft-sh/vcluster/pkg/syncer"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/syncer/translator"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"

	"github.com/loft-sh/vcluster/pkg/log"

	vclusterconfig "github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func CreateExporters(ctx *synccontext.ControllerContext) error {
	exporterConfig := ctx.Config.Experimental.GenericSync
	if len(exporterConfig.Exports) == 0 {
		return nil
	}

	registerCtx := ctx.ToRegisterContext()
	for _, exportConfig := range exporterConfig.Exports {
		_, hasStatusSubresource, err := translate.EnsureCRDFromPhysicalCluster(
			registerCtx,
			registerCtx.PhysicalManager.GetConfig(),
			registerCtx.VirtualManager.GetConfig(),
			schema.FromAPIVersionAndKind(exportConfig.APIVersion, exportConfig.Kind))
		if err != nil {
			if exportConfig.Optional {
				klog.Infof("error ensuring CRD %s(%s) from host cluster: %v. Skipping exportSyncer as resource is optional", exportConfig.Kind, exportConfig.APIVersion, err)
				continue
			}

			return fmt.Errorf("error creating %s(%s) syncer: %w", exportConfig.Kind, exportConfig.APIVersion, err)
		}

		reversePatches := []*vclusterconfig.Patch{
			{
				Operation: vclusterconfig.PatchTypeCopyFromObject,
				FromPath:  "status",
				Path:      "status",
			},
		}
		reversePatches = append(reversePatches, exportConfig.ReversePatches...)
		exportConfig.ReversePatches = reversePatches

		s, err := createExporterFromConfig(registerCtx, exportConfig, hasStatusSubresource)
		klog.Infof("creating exporter for %s/%s", exportConfig.APIVersion, exportConfig.Kind)
		if err != nil {
			return fmt.Errorf("error creating %s(%s) syncer: %w", exportConfig.Kind, exportConfig.APIVersion, err)
		}

		err = syncer.RegisterSyncer(registerCtx, s)
		klog.Infof("registering export syncer for %s/%s", exportConfig.APIVersion, exportConfig.Kind)
		if err != nil {
			return fmt.Errorf("error registering syncer %w", err)
		}
	}

	return nil
}

func createExporterFromConfig(ctx *synccontext.RegisterContext, config *vclusterconfig.Export, hasStatusSubresource bool) (syncertypes.Syncer, error) {
	obj := &unstructured.Unstructured{}
	obj.SetKind(config.Kind)
	obj.SetAPIVersion(config.APIVersion)

	err := validateExportConfig(config)
	if err != nil {
		return nil, fmt.Errorf("invalid configuration for %s(%s) mapping: %w", config.Kind, config.APIVersion, err)
	}

	var selector labels.Selector
	if config.Selector != nil {
		selector, err = metav1.LabelSelectorAsSelector(metav1.SetAsLabelSelector(config.Selector.LabelSelector))
		if err != nil {
			return nil, fmt.Errorf("invalid selector in configuration for %s(%s) mapping: %w", config.Kind, config.APIVersion, err)
		}
	}

	gvk := schema.FromAPIVersionAndKind(config.APIVersion, config.Kind)
	controllerID := fmt.Sprintf("%s/%s/GenericExport", strings.ToLower(gvk.Kind), strings.ToLower(gvk.Group))
	mapper, err := generic.NewMapper(ctx, obj, translate.Default.HostName)
	if err != nil {
		return nil, err
	}

	return &exporter{
		GenericTranslator: translator.NewGenericTranslator(ctx, controllerID, obj, mapper),
		ObjectPatcher: &exportPatcher{
			config: config,
			gvk:    gvk,
		},

		patcher:  NewPatcher(ctx.VirtualManager.GetClient(), ctx.PhysicalManager.GetClient(), hasStatusSubresource, log.New(controllerID)),
		gvk:      gvk,
		selector: selector,
		name:     controllerID,

		replaceWhenInvalid: config.ReplaceWhenInvalid,
	}, nil
}

var _ syncertypes.Syncer = &exporter{}

type exporter struct {
	syncertypes.GenericTranslator
	ObjectPatcher

	patcher            *Patcher
	gvk                schema.GroupVersionKind
	selector           labels.Selector
	replaceWhenInvalid bool

	name string
}

func (f *exporter) Syncer() syncertypes.Sync[client.Object] {
	return syncer.ToGenericSyncer[*unstructured.Unstructured](f)
}

func (f *exporter) SyncToHost(ctx *synccontext.SyncContext, event *synccontext.SyncToHostEvent[*unstructured.Unstructured]) (ctrl.Result, error) {
	// check if selector matches
	if !f.objectMatches(event.Virtual) {
		return ctrl.Result{}, nil
	}

	// delete object if host was deleted
	if event.IsDelete() {
		return syncer.DeleteVirtualObject(ctx, event.Virtual, "host object was deleted")
	}

	// apply object to physical cluster
	ctx.Log.Infof("Create physical %s %s/%s, since it is missing, but virtual object exists", f.gvk.Kind, event.Virtual.GetNamespace(), event.Virtual.GetName())
	pObj, err := f.patcher.ApplyPatches(ctx, event.Virtual, nil, f)
	if kerrors.IsConflict(err) {
		return ctrl.Result{Requeue: true}, nil
	}
	if err != nil {
		if err := IgnoreAcceptableErrors(err); err != nil {
			return ctrl.Result{}, nil
		}

		f.EventRecorder().Eventf(event.Virtual, "Warning", "SyncError", "Error syncing to physical cluster: %v", err)
		return ctrl.Result{}, fmt.Errorf("error applying patches: %w", err)
	} else if pObj == nil {
		return ctrl.Result{}, nil
	}

	// wait here for vObj to be created
	err = wait.PollUntilContextTimeout(ctx, time.Millisecond*10, time.Second, true, func(pollContext context.Context) (done bool, err error) {
		err = ctx.PhysicalClient.Get(pollContext, types.NamespacedName{
			Namespace: pObj.GetNamespace(),
			Name:      pObj.GetName(),
		}, f.Resource())
		if err != nil {
			if kerrors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}

		return true, nil
	})
	if err != nil {
		return ctrl.Result{}, err
	}

	// all good we can return safely
	return ctrl.Result{}, nil
}

func (f *exporter) Sync(ctx *synccontext.SyncContext, event *synccontext.SyncEvent[*unstructured.Unstructured]) (ctrl.Result, error) {
	// check if virtual object is not matching anymore
	if !f.objectMatches(event.Virtual) {
		ctx.Log.Infof("delete physical %s %s/%s, because it is not used anymore", f.gvk.Kind, event.Host.GetNamespace(), event.Host.GetName())
		err := ctx.PhysicalClient.Delete(ctx, event.Host, &client.DeleteOptions{
			GracePeriodSeconds: &[]int64{0}[0],
		})
		if err != nil {
			ctx.Log.Infof("error deleting physical %s %s/%s in physical cluster: %v", f.gvk.Kind, event.Host.GetNamespace(), event.Host.GetName(), err)
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	// check if either object is getting deleted
	if event.Virtual.GetDeletionTimestamp() != nil || event.Host.GetDeletionTimestamp() != nil {
		if event.Host.GetDeletionTimestamp() == nil {
			ctx.Log.Infof("delete physical object %s/%s, because the virtual object is being deleted", event.Host.GetNamespace(), event.Host.GetName())
			if err := ctx.PhysicalClient.Delete(ctx, event.Host); err != nil {
				return ctrl.Result{}, err
			}
		} else if event.Virtual.GetDeletionTimestamp() == nil {
			ctx.Log.Infof("delete virtual object %s/%s, because physical object %s/%s is being deleted", event.Virtual.GetNamespace(), event.Virtual.GetName(), event.Host.GetNamespace(), event.Host.GetName())
			if err := ctx.VirtualClient.Delete(ctx, event.Virtual); err != nil {
				return ctrl.Result{}, nil
			}
		}

		return ctrl.Result{}, nil
	}

	// apply reverse patches
	result, err := f.patcher.ApplyReversePatches(ctx, event.Virtual, event.Host, f)
	if err != nil {
		if kerrors.IsConflict(err) {
			return ctrl.Result{Requeue: true}, nil
		}
		if kerrors.IsInvalid(err) {
			ctx.Log.Infof("Warning: this message could indicate a timing issue with no significant impact, or a bug. Please report this if your resource never reaches the expected state. Error message: failed to patch virtual %s %s/%s: %v", f.gvk.Kind, event.Virtual.GetNamespace(), event.Virtual.GetName(), err)
			// this happens when some field is being removed shortly after being added, which suggest it's a timing issue
			// it doesn't seem to have any negative consequence besides the logged error message
			return ctrl.Result{Requeue: true}, nil
		}

		f.EventRecorder().Eventf(event.Virtual, "Warning", "SyncError", "Error syncing to virtual cluster: %v", err)
		return ctrl.Result{}, fmt.Errorf("failed to patch virtual %s %s/%s: %w", f.gvk.Kind, event.Virtual.GetNamespace(), event.Virtual.GetName(), err)
	} else if result == controllerutil.OperationResultUpdated || result == controllerutil.OperationResultUpdatedStatus || result == controllerutil.OperationResultUpdatedStatusOnly {
		// a change will trigger reconciliation anyway, and at that point we can make
		// a more accurate updates(reverse patches) to the virtual resource
		return ctrl.Result{}, nil
	}

	// apply patches
	pObj, err := f.patcher.ApplyPatches(ctx, event.Virtual, event.Host, f)
	err = IgnoreAcceptableErrors(err)
	if err != nil {
		// when invalid, auto delete and recreate to recover
		if kerrors.IsInvalid(err) && f.replaceWhenInvalid {
			// Replace the object
			ctx.Log.Infof("Replace physical object, because apply failed: %v", err)
			err = ctx.PhysicalClient.Delete(ctx, event.Host, &client.DeleteOptions{
				GracePeriodSeconds: &[]int64{0}[0],
			})
			if err != nil {
				return ctrl.Result{}, err
			}

			return ctrl.Result{}, nil
		}
		if kerrors.IsConflict(err) {
			return ctrl.Result{Requeue: true}, nil
		}

		f.EventRecorder().Eventf(event.Virtual, "Warning", "SyncError", "Error syncing to physical cluster: %v", err)
		return ctrl.Result{}, fmt.Errorf("error applying patches: %w", err)
	} else if pObj == nil {
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}

func (f *exporter) SyncToVirtual(ctx *synccontext.SyncContext, event *synccontext.SyncToVirtualEvent[*unstructured.Unstructured]) (ctrl.Result, error) {
	isManaged, err := f.GenericTranslator.IsManaged(ctx, event.Host)
	if err != nil {
		return ctrl.Result{}, err
	} else if !isManaged {
		return ctrl.Result{}, nil
	}

	// delete physical object because virtual one is missing
	return syncer.DeleteHostObject(ctx, event.Host, fmt.Sprintf("delete physical %s because virtual is missing", event.Host.GetName()))
}

func (f *exporter) Name() string {
	return f.name
}

// TranslateMetadata converts the virtual object into a physical object
func (f *exporter) TranslateMetadata(ctx *synccontext.SyncContext, vObj client.Object) client.Object {
	pObj := translate.HostMetadata(vObj, f.VirtualToHost(ctx, types.NamespacedName{Name: vObj.GetName(), Namespace: vObj.GetNamespace()}, vObj))
	if pObj == nil {
		return nil
	}

	if pObj.GetAnnotations() == nil {
		pObj.SetAnnotations(map[string]string{translate.ControllerLabel: f.Name()})
	} else {
		a := pObj.GetAnnotations()
		a[translate.ControllerLabel] = f.Name()
		pObj.SetAnnotations(a)
	}

	return pObj
}

func (f *exporter) objectMatches(obj client.Object) bool {
	return f.selector == nil || f.selector.Matches(labels.Set(obj.GetLabels()))
}
