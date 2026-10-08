package patcher

import (
	"fmt"

	"github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/clienthelper"
	"github.com/loft-sh/vcluster/pkg/util/patch"
	"k8s.io/apimachinery/pkg/types"
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

func SkipHostPatch() Option {
	return optionFn(func(p *Patcher) {
		p.SkipHostPatch = true
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
	pPatcher, err := NewPatcher(pObj, ctx.HostClient, options...)
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
func (h *SyncerPatcher) Patch(ctx *synccontext.SyncContext, pObj, vObj client.Object) (retErr error) {
	h.vPatcher.vObj = vObj
	h.vPatcher.pObj = pObj

	h.pPatcher.vObj = vObj
	h.pPatcher.pObj = pObj

	if ctx.ObjectCache != nil {
		vRebase := newCacheRebase(ctx.ObjectCache.Virtual(), h.vPatcher.beforeObject)
		pRebase := newCacheRebase(ctx.ObjectCache.Host(), h.pPatcher.beforeObject)
		defer func() {
			if retErr != nil {
				vRebase.apply(ctx)
				pRebase.apply(ctx)
			}
		}()
	}

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

// cacheRebase fixes the object cache entry of one side after a sync that failed part way.
//
// The next sync detects changes by comparing each object with its cache entry, and a
// successful write moves the cache entry to the written object. That object also holds
// changes made on the same side that the sync had not delivered to the other side yet,
// for example a condition an in-cluster controller set on the virtual pod while the host
// pod write then conflicted. Left like this, the next sync would treat those changes as
// synced and copy the older values from the other side over them.
//
// Restoring the old cache entry is not enough either: the values the syncer copied from the
// other side would then look like changes on this side and be written back over newer
// values. So the cache entry becomes the old entry plus only what the syncer wrote.
type cacheRebase struct {
	cache *synccontext.ObjectCache
	key   types.NamespacedName

	// before is the object as read at the start of the sync
	before client.Object
	// cached is the cache entry before the sync wrote anything, nil if there was none
	cached client.Object
}

func newCacheRebase(cache *synccontext.ObjectCache, before client.Object) *cacheRebase {
	key := client.ObjectKeyFromObject(before)
	cached, _ := cache.Get(key)
	return &cacheRebase{
		cache:  cache,
		key:    key,
		before: before.DeepCopyObject().(client.Object),
		cached: cached,
	}
}

func (r *cacheRebase) apply(ctx *synccontext.SyncContext) {
	written, ok := r.cache.Get(r.key)
	if !ok || written == r.cached {
		// nothing was written
		return
	} else if r.cached == nil {
		r.cache.Delete(written)
		return
	}

	syncerPatch, err := patch.CalculateMergePatch(r.before, written)
	if err != nil {
		ctx.Log.Errorf("rebase object cache entry %s: %v", r.key.String(), err)
		return
	}

	rebased := r.cached.DeepCopyObject().(client.Object)
	err = syncerPatch.Apply(rebased)
	if err != nil {
		ctx.Log.Errorf("rebase object cache entry %s: %v", r.key.String(), err)
		return
	}

	// keep the written resource version, so the next sync waits until the informer has
	// seen the write
	rebased.SetResourceVersion(written.GetResourceVersion())
	r.cache.Put(rebased)
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

	SkipHostPatch bool
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
	if h.SkipHostPatch && h.direction == synccontext.SyncVirtualToHost {
		return nil
	}
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
