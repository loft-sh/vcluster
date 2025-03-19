package patcher

import (
	"context"
	"fmt"

	"github.com/loft-sh/vcluster/pkg/scheme"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/clienthelper"
	"github.com/loft-sh/vcluster/pkg/util/patch"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

func CreateVirtualObject(ctx *synccontext.SyncContext, pObj, vObj client.Object, eventRecorder record.EventRecorder, hasStatus bool) (ctrl.Result, error) {
	gvk, err := apiutil.GVKForObject(vObj, scheme.Scheme)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("gvk for object: %w", err)
	}

	namespaceName := pObj.GetName()
	if pObj.GetNamespace() != "" {
		namespaceName = pObj.GetNamespace() + "/" + pObj.GetName()
	}

	err = ApplyObject(ctx, nil, vObj, synccontext.SyncHostToVirtual, hasStatus)
	if err != nil {
		ctx.Log.Infof("error syncing %s %s to virtual cluster: %v", gvk.Kind, namespaceName, err)
		if eventRecorder != nil {
			eventRecorder.Eventf(vObj, "Warning", "SyncError", "Error syncing to virtual cluster: %v", err)
		}

		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func CreateHostObject(ctx *synccontext.SyncContext, vObj, pObj client.Object, eventRecorder record.EventRecorder, hasStatus bool) (ctrl.Result, error) {
	gvk, err := apiutil.GVKForObject(pObj, scheme.Scheme)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("gvk for object: %w", err)
	}

	namespaceName := vObj.GetName()
	if vObj.GetNamespace() != "" {
		namespaceName = vObj.GetNamespace() + "/" + vObj.GetName()
	}

	err = ApplyObject(ctx, nil, pObj, synccontext.SyncVirtualToHost, hasStatus)
	if err != nil {
		ctx.Log.Infof("error syncing %s %s to host cluster: %v", gvk.Kind, namespaceName, err)
		if eventRecorder != nil {
			eventRecorder.Eventf(vObj, "Warning", "SyncError", "Error syncing to host cluster: %v", err)
		}

		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func DeleteHostObjectWithOptions(ctx *synccontext.SyncContext, pObj, vObjOld client.Object, reason string, options *client.DeleteOptions) (ctrl.Result, error) {
	err := deleteObject(ctx, pObj, reason, false, options)
	if err != nil {
		return ctrl.Result{}, err
	}

	if !clienthelper.IsNilObject(vObjOld) && ctx.ObjectCache != nil {
		ctx.ObjectCache.Virtual().Delete(vObjOld)
	}
	if ctx.ObjectCache != nil {
		ctx.ObjectCache.Host().Delete(pObj)
	}

	return ctrl.Result{}, nil
}

func DeleteHostObject(ctx *synccontext.SyncContext, pObj, vObjOld client.Object, reason string) (ctrl.Result, error) {
	return DeleteHostObjectWithOptions(ctx, pObj, vObjOld, reason, nil)
}

func DeleteVirtualObjectWithOptions(ctx *synccontext.SyncContext, vObj, pObjOld client.Object, reason string, options *client.DeleteOptions) (ctrl.Result, error) {
	err := deleteObject(ctx, vObj, reason, true, options)
	if err != nil {
		return ctrl.Result{}, err
	}

	if !clienthelper.IsNilObject(pObjOld) && ctx.ObjectCache != nil {
		ctx.ObjectCache.Host().Delete(pObjOld)
	}
	if ctx.ObjectCache != nil {
		ctx.ObjectCache.Virtual().Delete(vObj)
	}

	return ctrl.Result{}, nil
}

func DeleteVirtualObject(ctx *synccontext.SyncContext, vObj, pObjOld client.Object, reason string) (ctrl.Result, error) {
	return DeleteVirtualObjectWithOptions(ctx, vObj, pObjOld, reason, nil)
}

func deleteObject(ctx *synccontext.SyncContext, obj client.Object, reason string, isVirtual bool, options *client.DeleteOptions) error {
	side := "host"
	deleteClient := ctx.PhysicalClient
	if isVirtual {
		side = "virtual"
		deleteClient = ctx.VirtualClient
	}

	accessor, err := meta.Accessor(obj)
	if err != nil {
		return err
	}

	if obj.GetNamespace() != "" {
		ctx.Log.Infof("delete %s %s/%s, because %s", side, accessor.GetNamespace(), accessor.GetName(), reason)
	} else {
		ctx.Log.Infof("delete %s %s, because %s", side, accessor.GetName(), reason)
	}
	if options != nil {
		err = deleteClient.Delete(ctx, obj, options)
	} else {
		err = deleteClient.Delete(ctx, obj)
	}
	if err != nil {
		if kerrors.IsNotFound(err) {
			return nil
		}

		if obj.GetNamespace() != "" {
			ctx.Log.Infof("error deleting %s object %s/%s in %s cluster: %v", side, accessor.GetNamespace(), accessor.GetName(), side, err)
		} else {
			ctx.Log.Infof("error deleting %s object %s in %s cluster: %v", side, accessor.GetName(), side, err)
		}
		return err
	}

	return nil
}

func ApplyObject(ctx *synccontext.SyncContext, beforeObject, afterObject client.Object, direction synccontext.SyncDirection, hasStatus bool) error {
	var (
		objPatch patch.Patch
		err      error
	)
	if clienthelper.IsNilObject(beforeObject) {
		objPatch, err = patch.ConvertObjectToPatch(afterObject)
		if err != nil {
			return err
		}

		beforeObject = afterObject
	} else {
		objPatch, err = patch.CalculateMergePatch(beforeObject, afterObject)
		if err != nil {
			return err
		}
	}

	return ApplyObjectPatch(ctx, objPatch, beforeObject, direction, hasStatus)
}

func ApplyObjectPatch(ctx *synccontext.SyncContext, objPatch patch.Patch, obj client.Object, direction synccontext.SyncDirection, hasStatus bool) error {
	if objPatch.IsEmpty() {
		return nil
	}

	// if has status then first apply patch without status and then with status
	if hasStatus {
		// first everything else then status
		noStatusPatch := objPatch.DeepCopy()
		noStatusPatch.Delete("status")
		err := applyObjectWithPatch(ctx, noStatusPatch, obj, direction, false)
		if err != nil {
			return err
		}

		// second update only status
		statusPatch := objPatch.DeepCopy()
		statusPatch.DeleteAllExcept("", "status")
		err = applyObjectWithPatch(ctx, statusPatch, obj, direction, true)
		if err != nil {
			return err
		}

		return nil
	}

	return applyObjectWithPatch(ctx, objPatch, obj, direction, false)
}

func applyObjectWithPatch(ctx *synccontext.SyncContext, objPatch patch.Patch, obj client.Object, direction synccontext.SyncDirection, isStatus bool) error {
	if objPatch.IsEmpty() {
		return nil
	}

	kubeClient := ctx.PhysicalClient
	if direction == synccontext.SyncHostToVirtual {
		kubeClient = ctx.VirtualClient
	}

	// check if we should create or update the object
	isUpdate := false
	err := kubeClient.Get(ctx, types.NamespacedName{
		Namespace: obj.GetNamespace(),
		Name:      obj.GetName(),
	}, obj.DeepCopyObject().(client.Object))
	if err != nil && !kerrors.IsNotFound(err) {
		return fmt.Errorf("get object: %w", err)
	} else if err == nil {
		isUpdate = true
	}

	// we cannot create a status only object
	if !isUpdate && isStatus {
		return fmt.Errorf("cannot create status only object")
	}

	// apply the patch when it's an update, otherwise the patch is the create
	if isUpdate {
		beforeObject := obj.DeepCopyObject().(client.Object)
		err := objPatch.Apply(obj)
		if err != nil {
			return fmt.Errorf("apply patch: %w", err)
		} else if apiequality.Semantic.DeepEqual(beforeObject, obj) {
			// nothing to patch
			return nil
		}

		logUpdate(ctx, isStatus, direction, beforeObject, obj)
	} else {
		err := patch.ConvertPatchToObject(objPatch, obj)
		if err != nil {
			return fmt.Errorf("cannot convert patch to object: %w", err)
		}

		logCreate(ctx, direction, obj)
	}

	// create / update
	afterObj := obj.DeepCopyObject().(client.Object)
	if isStatus {
		err = kubeClient.Status().Update(ctx, obj)
		if err != nil {
			return fmt.Errorf("update object status: %w", err)
		}
	} else {
		if isUpdate {
			err = kubeClient.Update(ctx, obj)
			if err != nil {
				return fmt.Errorf("update object: %w", err)
			}
		} else {
			err = kubeClient.Create(ctx, obj)
			if err != nil {
				return fmt.Errorf("create object: %w", err)
			}
		}
	}

	// set the fields correctly, but only if the update / create succeeds
	afterObj.SetUID(obj.GetUID())
	afterObj.SetGeneration(obj.GetGeneration())
	afterObj.SetResourceVersion(obj.GetResourceVersion())
	afterObj.SetCreationTimestamp(obj.GetCreationTimestamp())
	afterObj.SetDeletionTimestamp(obj.GetDeletionTimestamp())
	afterObj.SetManagedFields(obj.GetManagedFields())
	afterObj.SetDeletionGracePeriodSeconds(obj.GetDeletionGracePeriodSeconds())
	afterObj.SetGenerateName(obj.GetGenerateName())
	afterObj.SetOwnerReferences(obj.GetOwnerReferences())
	if ctx.ObjectCache != nil {
		if direction == synccontext.SyncHostToVirtual {
			ctx.ObjectCache.Virtual().Put(afterObj)
		} else if direction == synccontext.SyncVirtualToHost {
			ctx.ObjectCache.Host().Put(afterObj)
		}
	}
	return nil
}

func logCreate(ctx context.Context, direction synccontext.SyncDirection, obj client.Object) {
	directionString := "host"
	if direction == synccontext.SyncHostToVirtual {
		directionString = "virtual"
	}

	patchMessage := fmt.Sprintf("Create %s object", directionString)
	klog.FromContext(ctx).Info(patchMessage, "kind", obj.GetObjectKind().GroupVersionKind().Kind, "object", obj.GetNamespace()+"/"+obj.GetName())
}

func logUpdate(ctx context.Context, isStatus bool, direction synccontext.SyncDirection, beforeObject, afterObject client.Object) {
	directionString := "host"
	if direction == synccontext.SyncHostToVirtual {
		directionString = "virtual"
	}

	status := ""
	if isStatus {
		status = " status"
	}

	// log patch
	patchMessage := fmt.Sprintf("Apply %s%s patch", directionString, status)
	patchBytes, _ := client.MergeFrom(beforeObject).Data(afterObject)
	klog.FromContext(ctx).Info(patchMessage, "kind", afterObject.GetObjectKind().GroupVersionKind().Kind, "object", afterObject.GetNamespace()+"/"+afterObject.GetName(), "patch", string(patchBytes))
}
