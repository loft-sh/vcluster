package patcher

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

type Option interface {
	Apply(p *Patcher)
}

type optionFn func(p *Patcher)

func (o optionFn) Apply(p *Patcher) {
	o(p)
}

func TranslatePatches(translate []config.TranslatePatch) Option {
	return optionFn(func(p *Patcher) {
		p.patches = translate
	})
}

func Direction(direction string) Option {
	return optionFn(func(p *Patcher) {
		p.direction = direction
	})
}

func NoStatusSubResource() Option {
	return optionFn(func(p *Patcher) {
		p.NoStatusSubResource = true
	})
}

func NewSyncerPatcher(ctx *synccontext.SyncContext, pObj, vObj client.Object, options ...Option) (*SyncerPatcher, error) {
	// virtual cluster patcher
	vPatcher, err := NewPatcher(vObj, ctx.VirtualClient, append([]Option{Direction("virtual")}, options...)...)
	if err != nil {
		return nil, fmt.Errorf("create virtual patcher: %w", err)
	}

	// host cluster patcher
	pPatcher, err := NewPatcher(pObj, ctx.PhysicalClient, append([]Option{Direction("host")}, options...)...)
	if err != nil {
		return nil, fmt.Errorf("create virtual patcher: %w", err)
	}

	return &SyncerPatcher{
		vPatcher: vPatcher,
		pPatcher: pPatcher,
	}, nil
}

type SyncerPatcher struct {
	vPatcher *Patcher
	pPatcher *Patcher
}

// Patch will attempt to patch the given object, including its status.
func (h *SyncerPatcher) Patch(ctx *synccontext.SyncContext, pObj, vObj client.Object) error {
	h.vPatcher.vObj = vObj
	h.vPatcher.pObj = pObj

	h.pPatcher.vObj = vObj
	h.pPatcher.pObj = pObj

	err := h.vPatcher.Patch(ctx, vObj)
	if err != nil {
		return fmt.Errorf("patch virtual object: %w", err)
	}

	err = h.pPatcher.Patch(ctx, pObj)
	if err != nil {
		return fmt.Errorf("patch host object: %w", err)
	}

	return nil
}

// Patcher is a utility for ensuring the proper patching of objects.
type Patcher struct {
	client       client.Client
	gvk          schema.GroupVersionKind
	beforeObject client.Object
	before       *unstructured.Unstructured
	after        *unstructured.Unstructured

	vObj client.Object
	pObj client.Object

	changes map[string]bool

	direction string

	patches []config.TranslatePatch

	NoStatusSubResource bool
}

// NewPatcher returns an initialized Patcher.
func NewPatcher(obj client.Object, crClient client.Client, options ...Option) (*Patcher, error) {
	// Return early if the object is nil.
	if err := checkNilObject(obj); err != nil {
		return nil, err
	}

	// Get the GroupVersionKind of the object,
	// used to validate against later on.
	gvk, err := apiutil.GVKForObject(obj, crClient.Scheme())
	if err != nil {
		return nil, err
	}

	// Convert the object to unstructured to compare against our before copy.
	unstructuredObj, err := toUnstructured(obj)
	if err != nil {
		return nil, err
	}

	patcher := &Patcher{
		client:       crClient,
		gvk:          gvk,
		before:       unstructuredObj,
		beforeObject: obj.DeepCopyObject().(client.Object),
	}

	for _, option := range options {
		option.Apply(patcher)
	}

	return patcher, nil
}

// Patch will attempt to patch the given object, including its status.
func (h *Patcher) Patch(ctx *synccontext.SyncContext, obj client.Object) error {
	// Return early if the object is nil.
	if err := checkNilObject(obj); err != nil {
		return err
	}

	// apply translate patches if wanted
	if len(h.patches) > 0 {
		obj = obj.DeepCopyObject().(client.Object)
		if h.direction == "host" {
			err := pro.ApplyPatchesHostObject(ctx, h.beforeObject, obj, h.vObj, h.patches)
			if err != nil {
				return fmt.Errorf("apply patches host object: %w", err)
			}
		} else if h.direction == "virtual" {
			err := pro.ApplyPatchesVirtualObject(ctx, h.beforeObject, obj, h.pObj, h.patches)
			if err != nil {
				return fmt.Errorf("apply patches virtual object: %w", err)
			}
		}
	}

	// Get the GroupVersionKind of the object that we want to patch.
	gvk, err := apiutil.GVKForObject(obj, h.client.Scheme())
	if err != nil {
		return err
	} else if gvk != h.gvk {
		return errors.Errorf("unmatched GroupVersionKind, expected %q got %q", h.gvk, gvk)
	}

	// Convert the object to unstructured to compare against our before copy.
	h.after, err = toUnstructured(obj)
	if err != nil {
		return err
	}

	// Calculate and store the top-level field changes (e.g. "metadata", "spec", "status") we have before/after.
	h.changes, err = h.calculateChanges(obj)
	if err != nil {
		return err
	}

	// Issue patches and return errors in an aggregate.
	var errs []error

	// check if status is there
	if h.NoStatusSubResource {
		if err := h.patchWholeObject(ctx, obj); err != nil {
			errs = append(errs, err)
		}
	} else {
		if err := h.patch(ctx, obj); err != nil {
			errs = append(errs, err)
		}

		if err := h.patchStatus(ctx, obj); err != nil {
			errs = append(errs, err)
		}
	}

	return utilerrors.NewAggregate(errs)
}

// patchWholeObject issues a patch for metadata, spec and status.
func (h *Patcher) patchWholeObject(ctx context.Context, obj client.Object) error {
	if !h.shouldPatch(nil, nil) {
		return nil
	}
	beforeObject, afterObject, err := h.calculatePatch(obj, nil, nil)
	if err != nil {
		return err
	}

	logPatch(ctx, fmt.Sprintf("Apply %s patch", h.direction), obj, beforeObject, afterObject)
	return h.client.Patch(ctx, afterObject, client.MergeFrom(beforeObject))
}

// patch issues a patch for metadata and spec.
func (h *Patcher) patch(ctx context.Context, obj client.Object) error {
	if !h.shouldPatch(nil, statusKey) {
		return nil
	}
	beforeObject, afterObject, err := h.calculatePatch(obj, nil, statusKey)
	if err != nil {
		return err
	}

	logPatch(ctx, fmt.Sprintf("Apply %s patch", h.direction), obj, beforeObject, afterObject)
	return h.client.Patch(ctx, afterObject, client.MergeFrom(beforeObject))
}

// patchStatus issues a patch if the status has changed.
func (h *Patcher) patchStatus(ctx context.Context, obj client.Object) error {
	if !h.shouldPatch(statusKey, nil) {
		return nil
	}
	beforeObject, afterObject, err := h.calculatePatch(obj, statusKey, nil)
	if err != nil {
		return err
	}

	logPatch(ctx, fmt.Sprintf("Apply %s status patch", h.direction), obj, beforeObject, afterObject)
	return h.client.Status().Patch(ctx, afterObject, client.MergeFrom(beforeObject))
}

func logPatch(ctx context.Context, patchMessage string, obj, beforeObject, afterObject client.Object) {
	// log patch
	patchBytes, _ := client.MergeFrom(beforeObject).Data(afterObject)
	klog.FromContext(ctx).Info(patchMessage, "kind", obj.GetObjectKind().GroupVersionKind().Kind, "object", obj.GetNamespace()+"/"+obj.GetName(), "patch", string(patchBytes))
}

// calculatePatch returns the before/after objects to be given in a controller-runtime patch, scoped down to the absolute necessary.
func (h *Patcher) calculatePatch(afterObj client.Object, include, exclude map[string]bool) (client.Object, client.Object, error) {
	// Get a shallow unsafe copy of the before/after object in unstructured form.
	before := unsafeUnstructuredCopy(h.before, include, exclude)
	after := unsafeUnstructuredCopy(h.after, include, exclude)

	// We've now applied all modifications to local unstructured objects,
	// make copies of the original objects and convert them back.
	beforeObj := h.beforeObject.DeepCopyObject().(client.Object)
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(before.Object, beforeObj); err != nil {
		return nil, nil, err
	}
	afterObj = afterObj.DeepCopyObject().(client.Object)
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(after.Object, afterObj); err != nil {
		return nil, nil, err
	}
	return beforeObj, afterObj, nil
}

func (h *Patcher) shouldPatch(include, exclude map[string]bool) bool {
	// Ranges over the keys of the unstructured object, think of this as the very top level of an object
	// when submitting a yaml to kubectl or a client.
	// These would be keys like `apiVersion`, `kind`, `metadata`, `spec`, `status`, etc.
	for key := range h.changes {
		// exclude
		if len(exclude) > 0 && exclude[key] {
			continue
		}

		// include
		if len(include) > 0 && !include[key] {
			continue
		}

		return true
	}

	return false
}

// calculate changes tries to build a patch from the before/after objects we have
// and store in a map which top-level fields (e.g. `metadata`, `spec`, `status`, etc.) have changed.
func (h *Patcher) calculateChanges(after client.Object) (map[string]bool, error) {
	// Calculate patch data.
	patch := client.MergeFrom(h.beforeObject)
	diff, err := patch.Data(after)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to calculate patch data")
	}

	// Unmarshal patch data into a local map.
	patchDiff := map[string]interface{}{}
	if err := json.Unmarshal(diff, &patchDiff); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal patch data into a map")
	}

	// Return the map.
	res := make(map[string]bool, len(patchDiff))
	for key := range patchDiff {
		res[key] = true
	}
	return res, nil
}

func checkNilObject(obj client.Object) error {
	if obj == nil || (reflect.ValueOf(obj).IsValid() && reflect.ValueOf(obj).IsNil()) {
		return errors.Errorf("expected non-nil object")
	}

	return nil
}
