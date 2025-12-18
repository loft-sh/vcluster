package etcd

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"go.etcd.io/etcd/api/v3/etcdserverpb"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"gotest.tools/v3/assert"
)

// This test ensures ListStream keeps a consistent snapshot across pages.
//
// etcd's Get response header contains the store's latest revision (not per-key revisions).
// A correct paginated snapshot must:
// - use WithRev(0) only for the first page (choose the snapshot revision),
// - reuse the snapshot revision for subsequent pages (WithRev(snapshotRev)),
// - never include keys created after snapshotRev, even if they appear between page requests.
func TestListStream_SnapshotAcrossPages(t *testing.T) {
	t.Parallel()

	const (
		prefix         = "test/"
		objectsLimit   = 1000
		snapshotKeys   = 1200
		newKeys        = 300
		totalKeys      = snapshotKeys + newKeys
		firstPageKeys  = objectsLimit
		secondPageKeys = snapshotKeys - firstPageKeys
	)

	// In etcd, ModRevision is the revision where a key was last modified.
	// A snapshot read at revision N must not return keys with ModRevision > N.
	snapshotRev := int64(snapshotKeys)
	afterSnapshotRev := int64(totalKeys)

	firstKVs := makeKVs(prefix, 0, firstPageKeys, 1)
	secondKVs := makeKVs(prefix, firstPageKeys, snapshotKeys, int64(firstPageKeys+1))
	newKVs := makeKVs(prefix, snapshotKeys, totalKeys, int64(snapshotKeys+1))

	var (
		callCount int
		gotOps    []clientv3.Op
	)

	get := func(_ context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
		callCount++
		op := clientv3.OpGet(key, opts...)
		gotOps = append(gotOps, op)

		switch callCount {
		case 1:
			assert.Equal(t, prefix, key, "first page must start at the provided prefix")
			return &clientv3.GetResponse{
				// etcd returns the latest revision in the header (not a per-object revision).
				Header: &etcdserverpb.ResponseHeader{Revision: snapshotRev},
				Kvs:    firstKVs,
				More:   true,
			}, nil
		case 2:
			wantStartKey := nextStartKey(firstKVs[len(firstKVs)-1].Key)
			assert.Equal(t, wantStartKey, key, "second page must start after the last key of the first page")
			// Simulate keys being added between the first and second page.
			// If the implementation doesn't keep the snapshot revision across pages (e.g. uses WithRev(0)),
			// the second page would include keys created after snapshotRev (newKVs).
			if op.Rev() == 0 {
				return &clientv3.GetResponse{
					Header: &etcdserverpb.ResponseHeader{Revision: afterSnapshotRev},
					Kvs:    append(secondKVs, newKVs...),
					More:   false,
				}, nil
			}

			assert.Equal(t, snapshotRev, op.Rev(), "second page must reuse the snapshot revision from the first page")
			return &clientv3.GetResponse{
				Header: &etcdserverpb.ResponseHeader{Revision: afterSnapshotRev},
				Kvs:    secondKVs,
				More:   false,
			}, nil
		default:
			return nil, fmt.Errorf("unexpected call %d", callCount)
		}
	}

	ch := listStream(context.Background(), prefix, get)
	gotKeys, gotMods, gotErrs := readListStream(ch)
	assert.Equal(t, 0, gotErrs, "snapshot stream must not emit errors")
	assert.Equal(t, snapshotKeys, len(gotKeys), "snapshot stream must not include keys created after the snapshot revision")

	seen := map[string]struct{}{}
	for i := 0; i < snapshotKeys; i++ {
		wantKey := fmt.Sprintf("%s%04d", prefix, i)
		assert.Equal(t, wantKey, gotKeys[i], "streamed key order must match snapshot order")
		_, ok := seen[gotKeys[i]]
		assert.Assert(t, !ok, "key must not be duplicated across pages: %q", gotKeys[i])
		seen[gotKeys[i]] = struct{}{}
		assert.Assert(t, gotMods[i] != 0, "key must have a non-zero ModRevision: index=%d key=%q", i, gotKeys[i])
	}
	assert.Equal(t, snapshotRev, gotMods[len(gotMods)-1], "last key in snapshot must have ModRevision == snapshotRev")

	assert.Equal(t, 2, callCount, "snapshot pagination should require exactly 2 Get calls")
	rangeEnd := clientv3.GetPrefixRangeEnd(prefix)
	assert.Equal(t, 2, len(gotOps), "expected exactly 2 captured operations")
	assert.Equal(t, prefix, string(gotOps[0].KeyBytes()), "first request key mismatch")
	assert.Equal(t, nextStartKey(firstKVs[len(firstKVs)-1].Key), string(gotOps[1].KeyBytes()), "second request key mismatch")
	assert.Equal(t, int64(objectsLimit), gotOps[0].Limit(), "first request must set limit to objectsLimit")
	assert.Equal(t, int64(objectsLimit), gotOps[1].Limit(), "second request must set limit to objectsLimit")
	assert.Equal(t, rangeEnd, string(gotOps[0].RangeBytes()), "first request must use prefix range end")
	assert.Equal(t, rangeEnd, string(gotOps[1].RangeBytes()), "second request must use prefix range end")
	assert.Equal(t, int64(0), gotOps[0].Rev(), "first request must use Rev=0 to select snapshot revision")
	assert.Equal(t, snapshotRev, gotOps[1].Rev(), "second request must use the snapshot revision")
}

// This test verifies that an empty range (no keys under the prefix) returns no values,
// emits no errors, and closes the channel after a single Get.
func TestListStream_Empty(t *testing.T) {
	t.Parallel()

	const prefix = "empty/"

	var callCount int
	get := func(_ context.Context, key string, _ ...clientv3.OpOption) (*clientv3.GetResponse, error) {
		callCount++
		assert.Equal(t, prefix, key, "empty list should query the provided prefix")
		return &clientv3.GetResponse{
			Header: &etcdserverpb.ResponseHeader{Revision: 1},
			Kvs:    nil,
		}, nil
	}

	gotKeys, _, gotErrs := readListStream(listStream(context.Background(), prefix, get))
	assert.Equal(t, 0, gotErrs, "empty list must not emit errors")
	assert.Equal(t, 0, len(gotKeys), "empty list must not emit values")
	assert.Equal(t, 1, callCount, "empty list should perform exactly one Get")
}

// This test verifies that a Get error on the first page is forwarded as a single error
// on the output channel and then the stream terminates.
func TestListStream_ErrorFirstPage(t *testing.T) {
	t.Parallel()

	const prefix = "err/"

	sentinel := errors.New("boom")
	var callCount int
	get := func(_ context.Context, _ string, _ ...clientv3.OpOption) (*clientv3.GetResponse, error) {
		callCount++
		return nil, sentinel
	}

	ch := listStream(context.Background(), prefix, get)
	_, _, gotErrs := readListStream(ch)
	assert.Equal(t, 1, gotErrs, "first-page error must be forwarded to the stream")
	assert.Equal(t, 1, callCount, "first-page error should stop pagination")
}

// This test verifies that if the first page succeeds and a later page fails, the stream:
// - emits all values from the successful page(s),
// - emits a single error,
// - and then terminates (channel is closed).
func TestListStream_ErrorAfterFirstPage(t *testing.T) {
	t.Parallel()

	const (
		prefix         = "partial/"
		firstHeaderRev = int64(7)
	)

	firstKVs := makeKVs(prefix, 0, 1000, 100)

	sentinel := errors.New("second get failed")
	var callCount int
	get := func(_ context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
		callCount++
		switch callCount {
		case 1:
			return &clientv3.GetResponse{
				Header: &etcdserverpb.ResponseHeader{Revision: firstHeaderRev},
				Kvs:    firstKVs,
				More:   true,
			}, nil
		case 2:
			op := clientv3.OpGet(key, opts...)
			assert.Equal(t, firstHeaderRev, op.Rev(), "second page must reuse header revision from first page")
			return nil, sentinel
		default:
			return nil, fmt.Errorf("unexpected call %d", callCount)
		}
	}

	gotKeys, _, gotErrs := readListStream(listStream(context.Background(), prefix, get))
	assert.Equal(t, 1, gotErrs, "pagination error must be forwarded to the stream")
	assert.Equal(t, len(firstKVs), len(gotKeys), "successful first page values must be emitted before the error")
	assert.Equal(t, 2, callCount, "error on second page should stop pagination")
}

func makeKVs(prefix string, start, end int, modRevStart int64) []*mvccpb.KeyValue {
	kvs := make([]*mvccpb.KeyValue, 0, end-start)
	for i := start; i < end; i++ {
		kvs = append(kvs, &mvccpb.KeyValue{
			Key:         []byte(fmt.Sprintf("%s%04d", prefix, i)),
			Value:       []byte(fmt.Sprintf("val-%04d", i)),
			ModRevision: modRevStart + int64(i-start),
		})
	}
	return kvs
}

func readListStream(ch <-chan *ValueOrError) (keys []string, modified []int64, errs int) {
	for v := range ch {
		if v == nil {
			continue
		}
		if v.Error != nil {
			errs++
			continue
		}
		keys = append(keys, string(v.Value.Key))
		modified = append(modified, v.Value.Modified)
	}
	return keys, modified, errs
}
