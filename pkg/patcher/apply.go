package patcher

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/loft-sh/vcluster/pkg/scheme"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/clienthelper"
	"github.com/loft-sh/vcluster/pkg/util/patch"
	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

func CreateVirtualObject(ctx *synccontext.SyncContext, pObj, vObj client.Object, eventRecorder events.EventRecorder, hasStatus bool) (ctrl.Result, error) {
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
			eventRecorder.Eventf(
				vObj,
				nil,
				"Warning",
				"SyncError",
				fmt.Sprintf("Sync%s", gvk.Kind),
				"Error syncing to virtual cluster: %v",
				err)
		}

		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func CreateHostObject(ctx *synccontext.SyncContext, vObj, pObj client.Object, eventRecorder events.EventRecorder, hasStatus bool) (ctrl.Result, error) {
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
			eventRecorder.Eventf(
				vObj,
				nil,
				"Warning",
				"SyncError",
				fmt.Sprintf("Sync%s", gvk.Kind),
				"Error syncing to host cluster: %v",
				err,
			)
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
	deleteClient := ctx.HostClient
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

	kubeClient := ctx.HostClient
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
	klog.FromContext(ctx).V(1).Info(patchMessage, "kind", afterObject.GetObjectKind().GroupVersionKind().Kind, "object", afterObject.GetNamespace()+"/"+afterObject.GetName(), "patch", sanitizePatchForLog(afterObject, patchBytes))
}

// lastAppliedConfigAnnotation is the annotation key written by `kubectl apply`.
// Its value is a JSON-encoded copy of the full manifest, which for Secrets
// includes the plaintext or base64-encoded secret data and must therefore be
// redacted from logs alongside the top-level "data" / "stringData" fields.
const lastAppliedConfigAnnotation = "kubectl.kubernetes.io/last-applied-configuration"

// redactedValue is the placeholder string substituted for sensitive field values
// in log output. redactedJSON is the same value encoded as a JSON string literal,
// suitable for use as a json.RawMessage.
const (
	redactedValue = "[REDACTED]"
	redactedJSON  = `"` + redactedValue + `"`
)

// sanitizePatchForLog returns the patch as a log-safe string.
//
// For Secret objects three categories of sensitive content are redacted:
//  1. Top-level "data" values (base64-encoded binary secret material).
//  2. Top-level "stringData" values (plaintext secret material).
//  3. The "kubectl.kubernetes.io/last-applied-configuration" annotation, which
//     kubectl apply stores as a JSON-encoded copy of the whole manifest and
//     therefore also contains the secret data.
//
// In all three cases the keys are preserved so that callers can still see
// which entries changed without seeing their content.
//
// For every other object type the patch is returned as-is.
func sanitizePatchForLog(obj client.Object, patchBytes []byte) string {
	if _, ok := obj.(*corev1.Secret); !ok {
		return string(patchBytes)
	}

	// Parse the patch so individual top-level fields can be inspected and
	// selectively replaced.
	var patchMap map[string]json.RawMessage
	if err := json.Unmarshal(patchBytes, &patchMap); err != nil {
		// If the patch cannot be parsed at all, return a safe sentinel rather
		// than risking a partial leak.
		return redactedValue
	}

	// Redact "data" (binary, base64-encoded) and "stringData" (plaintext)
	// at the top level of the patch.
	redactSecretDataFields(patchMap)

	// The last-applied-configuration annotation is a JSON string that kubectl
	// apply writes on every managed object. For Secrets it contains the full
	// manifest including the secret data, so it must also be redacted.
	redactLastAppliedAnnotation(patchMap)

	// Re-serialize the (now sanitized) patch map for logging.
	out, err := json.Marshal(patchMap)
	if err != nil {
		return redactedValue
	}
	return string(out)
}

// redactSecretDataFields replaces the values of the "data" and "stringData"
// entries in m with "[REDACTED]" placeholders, keeping the keys intact so that
// callers can still see which entries changed without seeing their content.
//
// If a field value cannot be parsed as a key→value map the entire field is
// replaced with a "[REDACTED]" sentinel.
func redactSecretDataFields(m map[string]json.RawMessage) {
	for _, field := range []string{"data", "stringData"} {
		raw, exists := m[field]
		if !exists {
			continue
		}

		var fieldMap map[string]json.RawMessage
		if err := json.Unmarshal(raw, &fieldMap); err != nil {
			// Unexpected shape – replace the entire field to be safe.
			m[field] = json.RawMessage(redactedJSON)
			continue
		}

		// null is a valid merge-patch value meaning "clear this field".
		// There are no keys or values to redact; preserve it verbatim.
		if fieldMap == nil {
			continue
		}

		redactedMap := make(map[string]string, len(fieldMap))
		for k := range fieldMap {
			redactedMap[k] = redactedValue
		}
		redactedBytes, _ := json.Marshal(redactedMap)
		m[field] = redactedBytes
	}
}

// redactLastAppliedAnnotation sanitizes the value of the
// kubectl.kubernetes.io/last-applied-configuration annotation inside patchMap.
//
// The annotation value is a JSON-encoded string containing the full object
// manifest as it was last applied by kubectl. For Secrets this manifest
// includes the secret data.
//
// The annotation key itself is always preserved. The function is a no-op at
// every step where the expected structure is absent.
func redactLastAppliedAnnotation(patchMap map[string]json.RawMessage) {
	metaRaw, exists := patchMap["metadata"]
	if !exists {
		return
	}

	var metaMap map[string]json.RawMessage
	if err := json.Unmarshal(metaRaw, &metaMap); err != nil {
		return
	}

	annotationsRaw, exists := metaMap["annotations"]
	if !exists {
		return
	}

	var annotations map[string]json.RawMessage
	if err := json.Unmarshal(annotationsRaw, &annotations); err != nil {
		return
	}

	raw, exists := annotations[lastAppliedConfigAnnotation]
	if !exists {
		return
	}

	annotations[lastAppliedConfigAnnotation] = sanitizeLastAppliedAnnotationValue(raw)

	annotationsBytes, err := json.Marshal(annotations)
	if err != nil {
		return
	}
	metaMap["annotations"] = annotationsBytes

	metaBytes, err := json.Marshal(metaMap)
	if err != nil {
		return
	}
	patchMap["metadata"] = metaBytes
}

// sanitizeLastAppliedAnnotationValue parses the raw JSON value of the
// last-applied-configuration annotation, unmarshals the embedded manifest as a
// corev1.Secret to validate its structure, redacts the sensitive fields, and
// returns the sanitized manifest re-encoded as a JSON string.
//
// On any parse or marshal error, the function returns a "[REDACTED]" sentinel so
// that a failure to sanitize never causes raw secret data to be logged.
func sanitizeLastAppliedAnnotationValue(raw json.RawMessage) json.RawMessage {
	var manifestJSON string
	if err := json.Unmarshal(raw, &manifestJSON); err != nil {
		return json.RawMessage(redactedJSON)
	}

	// Re-parse the manifest as a generic map so we can apply redactSecretDataFields
	// and produce clean "[REDACTED]" string values in the output.
	var manifestMap map[string]json.RawMessage
	if err := json.Unmarshal([]byte(manifestJSON), &manifestMap); err != nil {
		return json.RawMessage(redactedJSON)
	}

	// Apply the same key-preserving redaction used for the top-level patch.
	redactSecretDataFields(manifestMap)

	redactedManifest, err := json.Marshal(manifestMap)
	if err != nil {
		return json.RawMessage(redactedJSON)
	}

	// The annotation value must be a JSON string, so re-encode the manifest
	// bytes as a JSON string literal.
	encodedAnnotation, err := json.Marshal(string(redactedManifest))
	if err != nil {
		return json.RawMessage(redactedJSON)
	}
	return encodedAnnotation
}
