package patcher

import (
	"fmt"

	"github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/clienthelper"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Option interface {
	Apply(p *Patcher)
}

type optionFn func(p *Patcher)

func (o optionFn) Apply(p *Patcher) {
	o(p)
}

func TranslatePatches(translate []config.TranslatePatch, reverseExpressions bool) Option {
	return optionFn(func(p *Patcher) {
		p.patches = translate
		p.reverseExpressions = reverseExpressions
	})
}

func NoStatusSubResource() Option {
	return optionFn(func(p *Patcher) {
		p.NoStatusSubResource = true
	})
}

func NewSyncerPatcher(ctx *synccontext.SyncContext, pObj, vObj client.Object, options ...Option) (*SyncerPatcher, error) {
	// virtual cluster patcher
	vPatcher, err := NewPatcher(vObj, ctx.VirtualClient, options...)
	if err != nil {
		return nil, fmt.Errorf("create virtual patcher: %w", err)
	}
	vPatcher.direction = synccontext.SyncHostToVirtual

	// host cluster patcher
	pPatcher, err := NewPatcher(pObj, ctx.PhysicalClient, options...)
	if err != nil {
		return nil, fmt.Errorf("create virtual patcher: %w", err)
	}
	pPatcher.direction = synccontext.SyncVirtualToHost

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
	beforeObject client.Object

	vObj client.Object
	pObj client.Object

	direction synccontext.SyncDirection

	patches            []config.TranslatePatch
	reverseExpressions bool

	NoStatusSubResource bool
}

// NewPatcher returns an initialized Patcher.
func NewPatcher(obj client.Object, crClient client.Client, options ...Option) (*Patcher, error) {
	// Return early if the object is nil.
	if clienthelper.IsNilObject(obj) {
		return nil, fmt.Errorf("expected non-nil objet")
	}

	patcher := &Patcher{
		client:       crClient,
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
	if clienthelper.IsNilObject(obj) {
		return fmt.Errorf("expected non-nil object")
	}

	// apply translate patches if wanted
	if len(h.patches) > 0 {
		obj = obj.DeepCopyObject().(client.Object)
		if h.direction == synccontext.SyncVirtualToHost {
			err := pro.ApplyPatchesHostObject(ctx, h.beforeObject, obj, h.vObj, h.patches, h.reverseExpressions)
			if err != nil {
				return fmt.Errorf("apply patches host object: %w", err)
			}
		} else if h.direction == synccontext.SyncHostToVirtual {
			err := pro.ApplyPatchesVirtualObject(ctx, h.beforeObject, obj, h.pObj, h.patches, h.reverseExpressions)
			if err != nil {
				return fmt.Errorf("apply patches virtual object: %w", err)
			}
		}
	}

	return ApplyObject(ctx, h.beforeObject, obj, h.direction, !h.NoStatusSubResource)
}
