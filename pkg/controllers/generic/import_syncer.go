package generic

import (
	"context"
	"fmt"
	"strings"
	"time"

	vclusterconfig "github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/config"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"

	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/log"
	syncertypes "github.com/loft-sh/vcluster/pkg/types"
	"github.com/loft-sh/vcluster/pkg/util/clienthelper"
	util "github.com/loft-sh/vcluster/pkg/util/context"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type HostToVirtual func(ctx context.Context, req types.NamespacedName, pObj client.Object) types.NamespacedName

type VirtualToHost func(ctx context.Context, req types.NamespacedName, vObj client.Object) types.NamespacedName

func CreateImporters(ctx *config.ControllerContext) error {
	cfg := ctx.Config.Experimental.GenericSync
	if len(cfg.Imports) == 0 {
		return nil
	}

	registerCtx := util.ToRegisterContext(ctx)
	if !registerCtx.Config.Experimental.MultiNamespaceMode.Enabled {
		return fmt.Errorf("invalid configuration, 'import' type sync of the generic CRDs is allowed only in the multi-namespace mode")
	}

	for _, importConfig := range cfg.Imports {
		gvk := schema.FromAPIVersionAndKind(importConfig.APIVersion, importConfig.Kind)

		// don't skip even if scheme.Recognizes(gvk) to ensure scope for builtin
		// cluster scoped resources is registered and set properly
		isClusterScoped, hasStatusSubresource, err := translate.EnsureCRDFromPhysicalCluster(
			registerCtx.Context,
			registerCtx.PhysicalManager.GetConfig(),
			registerCtx.VirtualManager.GetConfig(),
			gvk)
		if err != nil {
			if importConfig.Optional {
				klog.Infof("error ensuring CRD %s(%s) from host cluster: %v, Skipping importSyncer as resource is optional", importConfig.Kind, importConfig.APIVersion, err)
				continue
			}

			return fmt.Errorf("error syncronizing CRD %s(%s) from the host cluster into vcluster: %w", importConfig.Kind, importConfig.APIVersion, err)
		}

		s, err := createImporter(registerCtx, importConfig, isClusterScoped, hasStatusSubresource)
		klog.Infof("creating importer for %s/%s", importConfig.APIVersion, importConfig.Kind)
		if err != nil {
			return fmt.Errorf("error creating %s(%s) syncer: %w", importConfig.Kind, importConfig.APIVersion, err)
		}

		err = syncer.RegisterSyncer(registerCtx, s)
		klog.Infof("registering import syncer for %s/%s", importConfig.APIVersion, importConfig.Kind)
		if err != nil {
			return fmt.Errorf("error registering syncer %w", err)
		}
	}

	return nil
}

func createImporter(ctx *synccontext.RegisterContext, config *vclusterconfig.Import, isClusterScoped, hasStatusSubresource bool) (syncertypes.Syncer, error) {
	gvk := schema.FromAPIVersionAndKind(config.APIVersion, config.Kind)
	controllerID := fmt.Sprintf("%s/%s/GenericImport", strings.ToLower(gvk.Kind), strings.ToLower(gvk.GroupVersion().String()))
	return &importer{
		ObjectPatcher: &importPatcher{
			config:        config,
			virtualClient: ctx.VirtualManager.GetClient(),
		},

		patcher: NewPatcher(ctx.PhysicalManager.GetClient(), ctx.VirtualManager.GetClient(), hasStatusSubresource, log.New(controllerID)),
		gvk:     gvk,

		replaceWhenInvalid: config.ReplaceWhenInvalid,

		virtualClient: ctx.VirtualManager.GetClient(),

		name: controllerID,
		syncerOptions: &syncertypes.Options{
			DisableUIDDeletion:   true,
			IsClusterScopedCRD:   isClusterScoped,
			HasStatusSubresource: hasStatusSubresource,
		},
	}, nil
}

func BuildCustomImporter(
	registerCtx *synccontext.RegisterContext,
	controllerID string,
	objectPatcher ObjectPatcher,
	hostToVirtual HostToVirtual,
	virtualToHost VirtualToHost,
	gvk schema.GroupVersionKind,
	replaceWhenInvalid bool,
) (syncertypes.Object, error) {
	isClusterScoped, hasStatusSubresource, err := translate.EnsureCRDFromPhysicalCluster(
		registerCtx.Context,
		registerCtx.PhysicalManager.GetConfig(),
		registerCtx.VirtualManager.GetConfig(),
		gvk)
	if err != nil {
		return nil, fmt.Errorf("error creating %s(%s) syncer: %w", gvk.Kind, gvk.GroupVersion().String(), err)
	}

	return &importer{
		ObjectPatcher: objectPatcher,

		hostToVirtual: hostToVirtual,
		virtualToHost: virtualToHost,

		patcher: NewPatcher(registerCtx.PhysicalManager.GetClient(), registerCtx.VirtualManager.GetClient(), hasStatusSubresource, log.New(controllerID)),
		gvk:     gvk,
		name:    controllerID,
		syncerOptions: &syncertypes.Options{
			DisableUIDDeletion:   true,
			IsClusterScopedCRD:   isClusterScoped,
			HasStatusSubresource: hasStatusSubresource,
		},

		virtualClient: registerCtx.VirtualManager.GetClient(),

		replaceWhenInvalid: replaceWhenInvalid,
	}, nil
}

type importer struct {
	ObjectPatcher

	hostToVirtual HostToVirtual
	virtualToHost VirtualToHost

	patcher            *Patcher
	replaceWhenInvalid bool

	virtualClient client.Client

	syncerOptions *syncertypes.Options
	gvk           schema.GroupVersionKind
	name          string
}

func (s *importer) Resource() client.Object {
	obj := &unstructured.Unstructured{}
	obj.SetKind(s.gvk.Kind)
	obj.SetAPIVersion(s.gvk.GroupVersion().String())
	return obj
}

func (s *importer) Name() string {
	return s.name
}

var _ syncertypes.OptionsProvider = &importer{}

func (s *importer) WithOptions() *syncertypes.Options {
	return s.syncerOptions
}

var _ syncertypes.ToVirtualSyncer = &importer{}

func (s *importer) SyncToVirtual(ctx *synccontext.SyncContext, pObj client.Object) (ctrl.Result, error) {
	// check if annotation is already present
	pAnnotations := pObj.GetAnnotations()
	if pAnnotations != nil && pAnnotations[translate.ControllerLabel] == s.Name() && !s.syncerOptions.IsClusterScopedCRD { // only delete pObj if its not cluster scoped
		ctx.Log.Infof("Delete physical %s %s/%s, since virtual is missing, but physical object was already synced", s.gvk.Kind, pObj.GetNamespace(), pObj.GetName())
		err := ctx.PhysicalClient.Delete(ctx.Context, pObj)
		if err != nil && !kerrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	// apply object to virtual cluster
	ctx.Log.Infof("Create virtual %s, since it is missing, but physical object %s/%s exists", s.gvk.Kind, pObj.GetNamespace(), pObj.GetName())
	vObj, err := s.patcher.ApplyPatches(ctx.Context, pObj, nil, s)
	if err != nil {
		// TODO: add eventRecorder?
		// s.EventRecorder().Eventf(vObj, "Warning", "SyncError", "Error syncing to virtual cluster: %v", err)
		return ctrl.Result{}, fmt.Errorf("error applying patches: %w", err)
	}

	// add annotation to physical resource to mark it as controlled by this syncer
	err = s.addAnnotationsToPhysicalObject(ctx, pObj, vObj)
	if err != nil {
		return ctrl.Result{}, err
	}

	// wait here for vObj to be created
	err = wait.PollUntilContextTimeout(ctx.Context, time.Millisecond*10, time.Second, true, func(syncContext context.Context) (done bool, err error) {
		err = ctx.VirtualClient.Get(syncContext, types.NamespacedName{
			Namespace: vObj.GetNamespace(),
			Name:      vObj.GetName(),
		}, s.Resource())
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

var _ syncertypes.Syncer = &importer{}

func (s *importer) SyncToHost(ctx *synccontext.SyncContext, vObj client.Object) (ctrl.Result, error) {
	// ignore all virtual resources that were not created by this controller
	if !s.isVirtualManaged(vObj) {
		return ctrl.Result{}, nil
	}

	// should we delete the object?
	if vObj.GetDeletionTimestamp() == nil {
		ctx.Log.Infof("remove virtual %s %s/%s, because object should get deleted", s.gvk.Kind, vObj.GetNamespace(), vObj.GetName())
		return ctrl.Result{}, ctx.VirtualClient.Delete(ctx.Context, vObj)
	}

	// remove finalizers if there are any
	if len(vObj.GetFinalizers()) > 0 {
		// delete the finalizer here so that the object can be deleted
		vObj.SetFinalizers([]string{})
		ctx.Log.Infof("remove virtual %s %s/%s finalizers, because object should get deleted", s.gvk.Kind, vObj.GetNamespace(), vObj.GetName())
		return ctrl.Result{}, ctx.VirtualClient.Update(ctx.Context, vObj)
	}

	// force deletion
	err := ctx.VirtualClient.Delete(ctx.Context, vObj, &client.DeleteOptions{
		GracePeriodSeconds: &[]int64{0}[0],
	})
	if kerrors.IsNotFound(err) {
		return ctrl.Result{}, nil
	}
	return ctrl.Result{}, err
}

func (s *importer) Sync(ctx *synccontext.SyncContext, pObj client.Object, vObj client.Object) (ctrl.Result, error) {
	// check if physical object is managed by this import controller
	managed, err := s.IsManaged(ctx.Context, pObj)
	if err != nil {
		return ctrl.Result{}, err
	} else if !managed {
		return ctrl.Result{}, nil
	}

	// check if either object is getting deleted
	if vObj.GetDeletionTimestamp() != nil || pObj.GetDeletionTimestamp() != nil {
		if pObj.GetDeletionTimestamp() == nil && !s.syncerOptions.IsClusterScopedCRD {
			ctx.Log.Infof("delete physical object %s/%s, because the virtual object is being deleted", pObj.GetNamespace(), pObj.GetName())
			if err := ctx.PhysicalClient.Delete(ctx.Context, pObj); err != nil {
				return ctrl.Result{}, err
			}
		} else if vObj.GetDeletionTimestamp() == nil {
			ctx.Log.Infof("delete virtual object %s/%s, because physical object is being deleted", vObj.GetNamespace(), vObj.GetName())
			if err := ctx.VirtualClient.Delete(ctx.Context, vObj); err != nil {
				return ctrl.Result{}, nil
			}
		}

		return ctrl.Result{}, nil
	}

	// execute reverse patches
	result, err := s.patcher.ApplyReversePatches(ctx.Context, pObj, vObj, s)
	if err != nil {
		if kerrors.IsInvalid(err) {
			ctx.Log.Infof("Warning: this message could indicate a timing issue with no significant impact, or a bug. Please report this if your resource never reaches the expected state. Error message: failed to patch virtual %s %s/%s: %v", s.gvk.Kind, vObj.GetNamespace(), vObj.GetName(), err)
			// this happens when some field is being removed shortly after being added, which suggest it's a timing issue
			// it doesn't seem to have any negative consequence besides the logged error message
			return ctrl.Result{Requeue: true}, nil
		}

		return ctrl.Result{}, fmt.Errorf("failed to apply reverse patch on physical %s %s/%s: %w", s.gvk.Kind, vObj.GetNamespace(), vObj.GetName(), err)
	} else if result == controllerutil.OperationResultUpdated || result == controllerutil.OperationResultUpdatedStatus || result == controllerutil.OperationResultUpdatedStatusOnly {
		// a change will trigger reconciliation anyway, and at that point we can make
		// a more accurate updates(reverse patches) to the virtual resource
		return ctrl.Result{}, nil
	}

	// apply patches
	_, err = s.patcher.ApplyPatches(ctx.Context, pObj, vObj, s)
	if err != nil {
		// when invalid, auto delete and recreate to recover
		if kerrors.IsInvalid(err) && s.replaceWhenInvalid {
			// Replace the object
			ctx.Log.Infof("Replace virtual object, because of apply failed: %v", err)
			err = ctx.VirtualClient.Delete(ctx.Context, vObj, &client.DeleteOptions{
				GracePeriodSeconds: &[]int64{0}[0],
			})
			if err != nil {
				return ctrl.Result{}, err
			}

			return ctrl.Result{}, nil
		}
		if kerrors.IsNotFound(err) {
			return ctrl.Result{Requeue: true}, nil
		}

		return ctrl.Result{}, fmt.Errorf("error applying patches: %w", err)
	}

	// ensure that annotation on physical resource to mark it as controlled by this syncer is present
	return ctrl.Result{}, s.addAnnotationsToPhysicalObject(ctx, pObj, vObj)
}

var _ syncertypes.ObjectExcluder = &importer{}

func (s *importer) ExcludeVirtual(vObj client.Object) bool {
	return s.excludeObject(vObj)
}

func (s *importer) ExcludePhysical(pObj client.Object) bool {
	return s.excludeObject(pObj)
}

func (s *importer) excludeObject(obj client.Object) bool {
	// check if back sync is disabled eg. for service account token secrets
	if obj.GetAnnotations() != nil && obj.GetAnnotations()[translate.SkipBackSyncInMultiNamespaceMode] == "true" {
		return true
	}
	if obj.GetLabels() != nil && obj.GetLabels()[translate.ControllerLabel] != "" {
		return true
	}
	if obj.GetAnnotations() != nil && obj.GetAnnotations()[translate.ControllerLabel] != "" && obj.GetAnnotations()[translate.ControllerLabel] != s.Name() {
		// make sure kind matches
		splitted := strings.Split(obj.GetAnnotations()[translate.ControllerLabel], "/")
		if len(splitted) != 3 {
			return true
		} else if splitted[0] != strings.ToLower(obj.GetObjectKind().GroupVersionKind().Kind) || splitted[1] != strings.ToLower(obj.GetObjectKind().GroupVersionKind().Group) {
			return false
		}

		return true
	}

	return false
}

func (s *importer) isVirtualManaged(vObj client.Object) bool {
	return vObj.GetAnnotations() != nil && vObj.GetAnnotations()[translate.ControllerLabel] != "" && vObj.GetAnnotations()[translate.ControllerLabel] == s.Name()
}

func (s *importer) IsManaged(_ context.Context, pObj client.Object) (bool, error) {
	if s.syncerOptions.IsClusterScopedCRD {
		return true, nil
	}
	if s.excludeObject(pObj) {
		return false, nil
	}

	// check if the pObj belong to a namespace managed by this vcluster
	if !translate.Default.IsTargetedNamespace(pObj.GetNamespace()) {
		return false, nil
	}

	// check that it is not managed by a non-generic syncer
	annotations := pObj.GetAnnotations()
	if annotations != nil && annotations[translate.ControllerLabel] == "" && annotations[translate.NameAnnotation] != "" {
		return false, nil
	}

	return true, nil
}

func (s *importer) VirtualToHost(ctx context.Context, req types.NamespacedName, vObj client.Object) types.NamespacedName {
	if s.virtualToHost != nil {
		return s.virtualToHost(ctx, req, vObj)
	}

	return types.NamespacedName{Name: translate.Default.PhysicalName(req.Name, req.Namespace), Namespace: translate.Default.PhysicalNamespace(req.Namespace)}
}

func (s *importer) HostToVirtual(ctx context.Context, req types.NamespacedName, pObj client.Object) types.NamespacedName {
	if s.syncerOptions.IsClusterScopedCRD {
		return types.NamespacedName{
			Name: req.Name,
		}
	}

	// in multi-namespace mode we just query the target namespace
	if !translate.Default.SingleNamespaceTarget() {
		vNamespace := &corev1.Namespace{}
		err := clienthelper.GetByIndex(ctx, s.virtualClient, vNamespace, constants.IndexByPhysicalName, req.Namespace)
		if err != nil {
			return types.NamespacedName{}
		}

		return types.NamespacedName{Name: req.Name, Namespace: vNamespace.GetName()}
	}

	// this is a little bit more tricky
	// check if we made annotations already
	if pObj != nil && pObj.GetAnnotations() != nil && pObj.GetAnnotations()[translate.NameAnnotation] != "" && pObj.GetAnnotations()[translate.NamespaceAnnotation] != "" {
		return types.NamespacedName{Name: pObj.GetAnnotations()[translate.NameAnnotation], Namespace: pObj.GetAnnotations()[translate.NamespaceAnnotation]}
	}

	return s.hostToVirtual(ctx, req, pObj)
}

func (s *importer) TranslateMetadata(ctx context.Context, pObj client.Object) client.Object {
	vObj := pObj.DeepCopyObject().(client.Object)
	vObj.SetResourceVersion("")
	vObj.SetUID("")
	vObj.SetManagedFields(nil)
	vObj.SetOwnerReferences(nil)
	vObj.SetFinalizers(nil)
	vObj.SetAnnotations(s.updateVirtualAnnotations(vObj.GetAnnotations()))
	nn := s.HostToVirtual(ctx, types.NamespacedName{Name: pObj.GetName(), Namespace: pObj.GetNamespace()}, pObj)
	vObj.SetName(nn.Name)
	vObj.SetNamespace(nn.Namespace)
	return vObj
}

// TranslateMetadataUpdate translates the object's metadata annotations and labels and determines
// if they have changed between the physical and virtual object
func (s *importer) TranslateMetadataUpdate(_ context.Context, vObj client.Object, pObj client.Object) (changed bool, annotations map[string]string, labels map[string]string) {
	updatedAnnotations := s.updateVirtualAnnotations(pObj.GetAnnotations())
	updatedLabels := pObj.GetLabels()
	return !equality.Semantic.DeepEqual(updatedAnnotations, vObj.GetAnnotations()) || !equality.Semantic.DeepEqual(updatedLabels, vObj.GetLabels()), updatedAnnotations, updatedLabels
}

func (s *importer) updateVirtualAnnotations(a map[string]string) map[string]string {
	if a == nil {
		return map[string]string{translate.ControllerLabel: s.Name()}
	}

	a[translate.ControllerLabel] = s.Name()
	delete(a, translate.NameAnnotation)
	delete(a, translate.NamespaceAnnotation)
	delete(a, translate.UIDAnnotation)
	delete(a, corev1.LastAppliedConfigAnnotation)
	return a
}

func (s *importer) addAnnotationsToPhysicalObject(ctx *synccontext.SyncContext, pObj client.Object, vObj client.Object) error {
	if s.syncerOptions.IsClusterScopedCRD {
		// do not add annotations to physical object
		return nil
	}

	originalObject := pObj.DeepCopyObject().(client.Object)
	annotations := pObj.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}
	annotations[translate.NameAnnotation] = vObj.GetName()
	annotations[translate.NamespaceAnnotation] = vObj.GetNamespace()
	annotations[translate.UIDAnnotation] = string(vObj.GetUID())
	annotations[translate.ControllerLabel] = s.Name()
	pObj.SetAnnotations(annotations)

	patch := client.MergeFrom(originalObject)
	patchBytes, err := patch.Data(pObj)
	if err != nil {
		return err
	} else if string(patchBytes) == "{}" {
		return nil
	}

	ctx.Log.Infof("Patch controlled-by annotation on %s %s/%s", s.gvk.Kind, pObj.GetNamespace(), pObj.GetName())
	return ctx.PhysicalClient.Patch(ctx.Context, pObj, patch)
}
