package patcher

import (
	"testing"

	"gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
)

func TestCacheRebase(t *testing.T) {
	key := types.NamespacedName{Name: "cm", Namespace: "ns"}
	configMap := func(resourceVersion string, data map[string]string) *corev1.ConfigMap {
		return &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: key.Name, Namespace: key.Namespace, ResourceVersion: resourceVersion},
			Data:       data,
		}
	}

	// last synced state
	cached := configMap("1", map[string]string{"a": "1"})
	// read at the start of the sync, with a change made on this side that is not synced yet
	before := configMap("2", map[string]string{"a": "1", "own": "x"})
	// written by the sync, with a value copied from the other side
	written := configMap("3", map[string]string{"a": "1", "own": "x", "copied": "y"})

	tests := []struct {
		name   string
		cached *corev1.ConfigMap
		write  bool
		// expected cache entry, nil if there should be none
		expected *corev1.ConfigMap
	}{
		{
			name:     "keeps own changes unsynced and adopts what the syncer wrote",
			cached:   cached,
			write:    true,
			expected: configMap("3", map[string]string{"a": "1", "copied": "y"}),
		},
		{
			name:     "leaves the cache entry alone if nothing was written",
			cached:   cached,
			expected: cached,
		},
		{
			name:  "removes the cache entry if there was none before",
			write: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := synccontext.NewBidirectionalObjectCache(&corev1.ConfigMap{}, nil).Virtual()
			if tt.cached != nil {
				cache.Put(tt.cached.DeepCopy())
			}

			rebase := newCacheRebase(cache, before.DeepCopy())
			if tt.write {
				cache.Put(written.DeepCopy())
			}
			rebase.apply(&synccontext.SyncContext{})

			got, ok := cache.Get(key)
			if tt.expected == nil {
				assert.Assert(t, !ok, "expected no cache entry, got %v", got)
				return
			}
			assert.Assert(t, ok, "expected a cache entry")
			assert.Equal(t, got.GetResourceVersion(), tt.expected.ResourceVersion)
			assert.DeepEqual(t, got.(*corev1.ConfigMap).Data, tt.expected.Data)
		})
	}
}
