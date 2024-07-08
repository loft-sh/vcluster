package generic

import (
	"context"
	"fmt"

	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"
	"github.com/loft-sh/vcluster/pkg/log"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var fieldManager = "vcluster-syncer"

type ObjectPatcherAndMetadataTranslator interface {
	translator.MetadataTranslator
	ObjectPatcher
}

var ErrNoUpdateNeeded = errors.New("no update needed")

// ObjectPatcher is the heart of the export and import syncers. The following functions are executed based on the lifecycle:
// During Creation:
// * ServerSideApply with nil existingOtherObj
// During Update:
// * ReverseUpdate
// * ServerSideApply
type ObjectPatcher interface {
	// ServerSideApply applies the translated object into the target cluster (either host or virtual), which
	// was built from originalObj. There might be also an existingOtherObj which was server side applied before, which
	// is not guaranteed to exist as this function is called during creation as well.
	//
	// For export syncers:
	// * originalObj is the virtual object
	// * translatedObj is the translated virtual object to host (rewritten metadata)
	// * existingOtherObj is the existing host object (can be nil if there is none yet)
	//
	// For import syncers:
	// * originalObj is the host object
	// * translatedObj is the translated host object to virtual (rewritten metadata)
	// * existingOtherObj is the existing virtual object (can be nil if there is none yet)
	ServerSideApply(ctx context.Context, originalObj, translatedObj, existingOtherObj client.Object) error

	// ReverseUpdate updates the destObj before running ServerSideApply. This can be useful to sync back
	// certain fields. Be careful that everything synced through this function **needs** to be excluded in
	// the ServerSideApply function. Both objects are guaranteed to exist for this function. Users can use
	// ErrNoUpdateNeeded to skip reverse update.
	//
	// For export syncers:
	// * destObj is the virtual object
	// * sourceObj is the host object
	//
	// For import syncers:
	// * destObj is the host object
	// * sourceObj is the virtual object
	ReverseUpdate(ctx context.Context, destObj, sourceObj client.Object) error
}

func NewPatcher(fromClient, toClient client.Client, statusIsSubresource bool, log log.Logger) *Patcher {
	return &Patcher{
		fromClient:          fromClient,
		toClient:            toClient,
		log:                 log,
		statusIsSubresource: statusIsSubresource,
	}
}

type Patcher struct {
	fromClient          client.Client
	toClient            client.Client
	log                 log.Logger
	statusIsSubresource bool
}

func (s *Patcher) ApplyPatches(ctx context.Context, fromObj, toObj client.Object, modifier ObjectPatcherAndMetadataTranslator) (client.Object, error) {
	translatedObject := modifier.TranslateMetadata(ctx, fromObj)
	toObjBase, err := toUnstructured(translatedObject)
	if err != nil {
		return nil, err
	}
	toObjCopied := toObjBase.DeepCopy()

	// apply patches on from object
	err = modifier.ServerSideApply(ctx, fromObj, toObjCopied, toObj)
	if err != nil {
		if toObj != nil && errors.Is(err, ErrNoUpdateNeeded) {
			return nil, nil
		}

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
			s.log.Infof("Server side apply status of %s", toObjCopied.GetName())
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
	s.log.Infof("Server side apply %s", toObjCopied.GetName())
	outObject := toObjCopied.DeepCopy()
	err = s.toClient.Patch(ctx, outObject, client.Apply, client.ForceOwnership, client.FieldOwner(fieldManager))
	if err != nil {
		return nil, errors.Wrap(err, "apply object")
	}

	return outObject, nil
}

func (s *Patcher) ApplyReversePatches(ctx context.Context, fromObj, otherObj client.Object, modifier ObjectPatcherAndMetadataTranslator) (controllerutil.OperationResult, error) {
	originalUnstructured, err := toUnstructured(fromObj)
	if err != nil {
		return controllerutil.OperationResultNone, err
	}
	fromCopied := originalUnstructured.DeepCopy()

	// apply patches on from object
	err = modifier.ReverseUpdate(ctx, fromCopied, otherObj)
	if err != nil {
		if errors.Is(err, ErrNoUpdateNeeded) {
			return controllerutil.OperationResultNone, nil
		}

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
			s.log.Infof("Reverse update status of %s", fromCopied.GetName())
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
		s.log.Infof("Reverse update %s", fromCopied.GetName())
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
