package generic

import (
	"context"
	"fmt"

	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/log"
	"github.com/loft-sh/vcluster/pkg/patches"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var (
	fieldManager = "vcluster-syncer"
)

type patcher struct {
	fromClient client.Client
	toClient   client.Client

	statusIsSubresource bool
	log                 log.Logger
}

func (s *patcher) ApplyPatches(ctx context.Context, fromObj, toObj client.Object, patchesConfig, reversePatchesConfig []*config.Patch, translateMetadata func(vObj client.Object) (client.Object, error), nameResolver patches.NameResolver) (client.Object, error) {
	translatedObject, err := translateMetadata(fromObj)
	if err != nil {
		return nil, errors.Wrap(err, "translate object")
	}

	toObjBase, err := toUnstructured(translatedObject)
	if err != nil {
		return nil, err
	}
	toObjCopied := toObjBase.DeepCopy()

	// apply patches on from object
	err = patches.ApplyPatches(toObjCopied, toObj, patchesConfig, reversePatchesConfig, nameResolver)
	if err != nil {
		return nil, fmt.Errorf("error applying patches: %w", err)
	}

	// compare status
	if s.statusIsSubresource && toObj != nil && toObj.GetUID() != "" {
		_, hasAfterStatus, err := unstructured.NestedFieldCopy(toObjCopied.Object, "status")
		if err != nil {
			return nil, err
		}

		// always apply status if it's there
		if hasAfterStatus {
			s.log.Infof("Apply status of %s during patching", toObjCopied.GetName())
			o := &client.SubResourcePatchOptions{PatchOptions: client.PatchOptions{FieldManager: fieldManager, Force: ptr.To(true)}}
			err = s.toClient.Status().Patch(ctx, toObjCopied.DeepCopy(), client.Apply, o)
			if err != nil {
				return nil, errors.Wrap(err, "apply status")
			}
		}

		if hasAfterStatus {
			unstructured.RemoveNestedField(toObjCopied.Object, "status")
		}
	}

	// always apply object
	s.log.Infof("Apply %s during patching", toObjCopied.GetName())
	outObject := toObjCopied.DeepCopy()
	err = s.toClient.Patch(ctx, outObject, client.Apply, client.ForceOwnership, client.FieldOwner(fieldManager))
	if err != nil {
		return nil, errors.Wrap(err, "apply object")
	}

	return outObject, nil
}

func (s *patcher) ApplyReversePatches(ctx context.Context, fromObj, otherObj client.Object, reversePatchConfig []*config.Patch, nameResolver patches.NameResolver) (controllerutil.OperationResult, error) {
	originalUnstructured, err := toUnstructured(fromObj)
	if err != nil {
		return controllerutil.OperationResultNone, err
	}
	fromCopied := originalUnstructured.DeepCopy()

	// apply patches on from object
	err = patches.ApplyPatches(fromCopied, otherObj, reversePatchConfig, nil, nameResolver)
	if err != nil {
		return controllerutil.OperationResultNone, fmt.Errorf("error applying reverse patches: %w", err)
	}

	// compare status
	if s.statusIsSubresource {
		beforeStatus, hasBeforeStatus, err := unstructured.NestedFieldCopy(originalUnstructured.Object, "status")
		if err != nil {
			return controllerutil.OperationResultNone, err
		}
		afterStatus, hasAfterStatus, err := unstructured.NestedFieldCopy(fromCopied.Object, "status")
		if err != nil {
			return controllerutil.OperationResultNone, err
		}

		// update status
		if (hasBeforeStatus || hasAfterStatus) && !equality.Semantic.DeepEqual(beforeStatus, afterStatus) {
			s.log.Infof("Update status of %s during reverse patching", fromCopied.GetName())
			err = s.fromClient.Status().Update(ctx, fromCopied)
			if err != nil {
				return controllerutil.OperationResultNone, errors.Wrap(err, "update reverse status")
			}

			return controllerutil.OperationResultUpdatedStatusOnly, nil
		}

		if hasBeforeStatus {
			unstructured.RemoveNestedField(originalUnstructured.Object, "status")
		}
		if hasAfterStatus {
			unstructured.RemoveNestedField(fromCopied.Object, "status")
		}
	}

	// compare rest of the object
	if !equality.Semantic.DeepEqual(originalUnstructured, fromCopied) {
		s.log.Infof("Update %s during reverse patching", fromCopied.GetName())
		err = s.fromClient.Update(ctx, fromCopied)
		if err != nil {
			return controllerutil.OperationResultNone, errors.Wrap(err, "update reverse")
		}

		return controllerutil.OperationResultUpdated, nil
	}

	return controllerutil.OperationResultNone, nil
}

func toUnstructured(obj client.Object) (*unstructured.Unstructured, error) {
	fromCopied, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj.DeepCopyObject())
	if err != nil {
		return nil, err
	}

	return &unstructured.Unstructured{Object: fromCopied}, nil
}
