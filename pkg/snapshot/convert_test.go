package snapshot

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	snapshotapi "github.com/loft-sh/api/v4/pkg/snapshot"
	"github.com/loft-sh/vcluster/pkg/etcd"
	"go.etcd.io/etcd/etcdutl/v3/snapshot"
	"go.etcd.io/etcd/pkg/v3/traceutil"
	"go.etcd.io/etcd/server/v3/lease"
	"go.etcd.io/etcd/server/v3/storage/backend"
	"go.etcd.io/etcd/server/v3/storage/mvcc"
	"go.uber.org/zap"
)

// buildRawEtcdSnapshot creates a real bbolt-backed etcd snapshot file (no
// server, no ports) with the given key/value pairs applied in order - "" as
// a value means delete the key - and returns the raw db file bytes, exactly
// as etcdClient.SnapshotWithVersion would stream them (no hash trailer).
func buildRawEtcdSnapshot(t *testing.T, ops []struct{ key, value string }) []byte {
	t.Helper()

	dir := t.TempDir()
	path := dir + "/db"
	lg := zap.NewNop()

	func() {
		be := backend.NewDefaultBackend(lg, path)
		defer be.Close()

		le := lease.NewLessor(lg, be, noopCluster{}, lease.LessorConfig{MinLeaseTTL: 60})
		defer le.Stop()

		store := mvcc.New(lg, be, le, mvcc.StoreConfig{})
		defer store.Close()

		for _, op := range ops {
			w := store.Write(traceutil.TODO())
			if op.value == "" {
				w.DeleteRange([]byte(op.key), nil)
			} else {
				w.Put([]byte(op.key), []byte(op.value), lease.NoLease)
			}
			w.End()
		}
		store.Commit()
	}()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read raw snapshot file: %v", err)
	}
	return data
}

func newTestArchiveBytes(t *testing.T, entries ...archiveEntry) []byte {
	t.Helper()
	path := newTestArchive(t, entries...)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read test archive: %v", err)
	}
	return data
}

func TestConvertEtcdSnapshotToKeyValueSnapshot(t *testing.T) {
	t.Parallel()

	dbBytes := buildRawEtcdSnapshot(t, []struct{ key, value string }{
		{"/registry/a", "va1"},
		{"/registry/b", "vb1"},
		{"/registry/c", "vc1"},
		{"/registry/b", ""}, // delete
		{"/registry/b", "vb2-recreated"},
	})

	srcBytes := newTestArchiveBytes(t, archiveEntry{key: DBStoreKey, value: dbBytes})

	var out bytes.Buffer
	err := ConvertEtcdSnapshotToKeyValueSnapshot(t.Context(), t.TempDir(), bytes.NewReader(srcBytes), &out)
	if err != nil {
		t.Fatalf("ConvertEtcdSnapshotToKeyValueSnapshot failed: %v", err)
	}

	outPath := writeBytesToTempFile(t, out.Bytes())
	kind, err := getSnapshotArchiveKind(outPath)
	if err != nil {
		t.Fatalf("getSnapshotArchiveKind failed: %v", err)
	}
	if kind != KeyValueSnapshotKind {
		t.Fatalf("expected KeyValueSnapshotKind, got %s", kind)
	}

	entries := readAllArchiveEntries(t, outPath)

	want := map[string]string{
		"/registry/a": "va1",
		"/registry/b": "vb2-recreated",
		"/registry/c": "vc1",
	}
	got := map[string]string{}
	for k, v := range entries {
		if k == RevisionStoreKey {
			continue
		}
		got[k] = string(v)
	}
	if len(got) != len(want) {
		t.Fatalf("expected %d live keys, got %d: %v", len(want), len(got), got)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("key %s: expected %q, got %q", k, v, got[k])
		}
	}

	wantStatus, err := snapshot.NewV3(zap.NewNop()).Status(writeBytesToTempFile(t, dbBytes))
	if err != nil {
		t.Fatalf("failed to get expected snapshot status: %v", err)
	}

	revBytes, ok := entries[RevisionStoreKey]
	if !ok {
		t.Fatal("expected a RevisionStoreKey entry in the output archive")
	}
	gotRevision, err := strconv.ParseInt(string(revBytes), 10, 64)
	if err != nil {
		t.Fatalf("failed to parse recorded revision %q: %v", revBytes, err)
	}
	if gotRevision != wantStatus.Revision {
		t.Errorf("expected recorded revision %d, got %d", wantStatus.Revision, gotRevision)
	}
}

func TestConvertEtcdSnapshotToKeyValueSnapshot_SkipKeys(t *testing.T) {
	t.Parallel()

	dbBytes := buildRawEtcdSnapshot(t, []struct{ key, value string }{
		{"/registry/keep", "v1"},
		{"/registry/skip", "v2"},
	})

	skipKeysBytes, err := json.Marshal(map[string]struct{}{"/registry/skip": {}})
	if err != nil {
		t.Fatalf("failed to marshal skipKeys: %v", err)
	}

	srcBytes := newTestArchiveBytes(t,
		archiveEntry{key: DBStoreKey, value: dbBytes},
		archiveEntry{key: SkipKeysStoreKey, value: skipKeysBytes},
	)

	var out bytes.Buffer
	if err := ConvertEtcdSnapshotToKeyValueSnapshot(t.Context(), t.TempDir(), bytes.NewReader(srcBytes), &out); err != nil {
		t.Fatalf("ConvertEtcdSnapshotToKeyValueSnapshot failed: %v", err)
	}

	entries := readAllArchiveEntries(t, writeBytesToTempFile(t, out.Bytes()))
	if _, ok := entries["/registry/skip"]; ok {
		t.Error("expected /registry/skip to be excluded from the converted archive")
	}
	if _, ok := entries["/registry/keep"]; !ok {
		t.Error("expected /registry/keep to be present in the converted archive")
	}
}

func TestConvertEtcdSnapshotToKeyValueSnapshot_ReleaseAndRequestPassthrough(t *testing.T) {
	t.Parallel()

	dbBytes := buildRawEtcdSnapshot(t, []struct{ key, value string }{{"/registry/a", "v1"}})

	releaseBytes := []byte(`{"name":"my-release"}`)
	requestKey := RequestStoreKey + "/" + snapshotapi.APIVersion
	requestBytes := []byte(`{"apiVersion":"v1beta1"}`)

	srcBytes := newTestArchiveBytes(t,
		archiveEntry{key: snapshotapi.SnapshotReleaseKey, value: releaseBytes},
		archiveEntry{key: requestKey, value: requestBytes},
		archiveEntry{key: DBStoreKey, value: dbBytes},
	)

	var out bytes.Buffer
	if err := ConvertEtcdSnapshotToKeyValueSnapshot(t.Context(), t.TempDir(), bytes.NewReader(srcBytes), &out); err != nil {
		t.Fatalf("ConvertEtcdSnapshotToKeyValueSnapshot failed: %v", err)
	}

	entries := readAllArchiveEntries(t, writeBytesToTempFile(t, out.Bytes()))
	if !bytes.Equal(entries[snapshotapi.SnapshotReleaseKey], releaseBytes) {
		t.Errorf("release passthrough mismatch: got %q", entries[snapshotapi.SnapshotReleaseKey])
	}
	if !bytes.Equal(entries[requestKey], requestBytes) {
		t.Errorf("request passthrough mismatch: got %q", entries[requestKey])
	}
}

// TestConvertEtcdSnapshotToKeyValueSnapshot_SkipsPollutedMetadataKey covers a
// source cluster that was itself restored from a snapshot, leaving a metadata
// key live in etcd. The converted archive must carry the freshly recorded
// revision, not the stale copy read back out of the store.
func TestConvertEtcdSnapshotToKeyValueSnapshot_SkipsPollutedMetadataKey(t *testing.T) {
	t.Parallel()

	dbBytes := buildRawEtcdSnapshot(t, []struct{ key, value string }{
		{"/registry/a", "v1"},
		{RevisionStoreKey, "999"},
	})
	srcBytes := newTestArchiveBytes(t, archiveEntry{key: DBStoreKey, value: dbBytes})

	var out bytes.Buffer
	if err := ConvertEtcdSnapshotToKeyValueSnapshot(t.Context(), t.TempDir(), bytes.NewReader(srcBytes), &out); err != nil {
		t.Fatalf("ConvertEtcdSnapshotToKeyValueSnapshot failed: %v", err)
	}

	wantStatus, err := snapshot.NewV3(zap.NewNop()).Status(writeBytesToTempFile(t, dbBytes))
	if err != nil {
		t.Fatalf("failed to get expected snapshot status: %v", err)
	}

	// readAllArchiveEntries maps by key, so a duplicate would silently
	// overwrite; assert the single surviving value is the freshly pinned one.
	entries := readAllArchiveEntries(t, writeBytesToTempFile(t, out.Bytes()))
	want := strconv.FormatInt(wantStatus.Revision, 10)
	if string(entries[RevisionStoreKey]) != want {
		t.Errorf("expected recorded revision %s, got %q", want, entries[RevisionStoreKey])
	}
}

// TestConvertEtcdSnapshotToKeyValueSnapshot_ExcludesNonPrefixedKeys pins parity
// with the live snapshot path, which lists under the "/" prefix. A key outside
// that prefix is unreachable for a live KV snapshot and must not appear here
// either.
func TestConvertEtcdSnapshotToKeyValueSnapshot_ExcludesNonPrefixedKeys(t *testing.T) {
	t.Parallel()

	dbBytes := buildRawEtcdSnapshot(t, []struct{ key, value string }{
		{"registry/no-leading-slash", "v1"},
		{"/registry/a", "v2"},
	})
	srcBytes := newTestArchiveBytes(t, archiveEntry{key: DBStoreKey, value: dbBytes})

	var out bytes.Buffer
	if err := ConvertEtcdSnapshotToKeyValueSnapshot(t.Context(), t.TempDir(), bytes.NewReader(srcBytes), &out); err != nil {
		t.Fatalf("ConvertEtcdSnapshotToKeyValueSnapshot failed: %v", err)
	}

	entries := readAllArchiveEntries(t, writeBytesToTempFile(t, out.Bytes()))
	if _, ok := entries["registry/no-leading-slash"]; ok {
		t.Error("expected a key outside the / prefix to be excluded")
	}
	if _, ok := entries["/registry/a"]; !ok {
		t.Error("expected /registry/a to be present in the converted archive")
	}
}

func TestConvertEtcdSnapshotToKeyValueSnapshot_EmptyDatabase(t *testing.T) {
	t.Parallel()

	dbBytes := buildRawEtcdSnapshot(t, nil)
	srcBytes := newTestArchiveBytes(t, archiveEntry{key: DBStoreKey, value: dbBytes})

	var out bytes.Buffer
	if err := ConvertEtcdSnapshotToKeyValueSnapshot(t.Context(), t.TempDir(), bytes.NewReader(srcBytes), &out); err != nil {
		t.Fatalf("ConvertEtcdSnapshotToKeyValueSnapshot failed: %v", err)
	}

	entries := readAllArchiveEntries(t, writeBytesToTempFile(t, out.Bytes()))
	liveKeys := 0
	for k := range entries {
		if k != RevisionStoreKey {
			liveKeys++
		}
	}
	if liveKeys != 0 {
		t.Errorf("expected zero live keys for an empty database, got %d: %v", liveKeys, entries)
	}
}

func TestConvertEtcdSnapshotToKeyValueSnapshot_AlreadyKeyValueKind(t *testing.T) {
	t.Parallel()

	srcBytes := newTestArchiveBytes(t, archiveEntry{key: "/registry/pods/default/foo", value: []byte("bar")})

	var out bytes.Buffer
	err := ConvertEtcdSnapshotToKeyValueSnapshot(t.Context(), t.TempDir(), bytes.NewReader(srcBytes), &out)
	if err == nil {
		t.Fatal("expected an error when converting an already KeyValueSnapshotKind archive")
	}
}

// TestParseEtcdSnapshotArchive_MissingDBStoreKey exercises parseEtcdSnapshotArchive's
// dbFile == "" sentinel directly. Going through the public
// ConvertEtcdSnapshotToKeyValueSnapshot entry point can't reach this: any
// archive lacking DBStoreKey makes getSnapshotArchiveKind return
// KeyValueSnapshotKind, so the caller's kind != EtcdSnapshotKind guard errors
// out first - the same path TestConvertEtcdSnapshotToKeyValueSnapshot_AlreadyKeyValueKind
// already covers.
func TestParseEtcdSnapshotArchive_MissingDBStoreKey(t *testing.T) {
	t.Parallel()

	srcPath := newTestArchive(t, archiveEntry{key: snapshotapi.SnapshotReleaseKey, value: []byte("{}")})

	_, err := parseEtcdSnapshotArchive(srcPath, t.TempDir())
	if err == nil {
		t.Fatal("expected an error for an archive without a DBStoreKey entry")
	}
	if !strings.Contains(err.Error(), "failed to find etcd snapshot in source archive") {
		t.Errorf("expected the missing-db sentinel error, got: %v", err)
	}
}

// TestConvertEtcdSnapshotToKeyValueSnapshot_Pagination exercises
// writeLiveKeyValues' multi-page path: startKey advancement via
// lastKey+0x00, cross-page pinnedRev pinning, and the exact-limit boundary
// (len(result.KVs) < etcd.EtcdPaginationLimit) - none of which any
// single-page test (every other case here seeds a handful of keys) reaches.
func TestConvertEtcdSnapshotToKeyValueSnapshot_Pagination(t *testing.T) {
	t.Parallel()

	for _, n := range []int{etcd.EtcdPaginationLimit - 1, etcd.EtcdPaginationLimit, etcd.EtcdPaginationLimit + 1} {
		t.Run(fmt.Sprintf("%d_keys", n), func(t *testing.T) {
			t.Parallel()

			ops := make([]struct{ key, value string }, n)
			for i := range n {
				key := fmt.Sprintf("/registry/pagination/%05d", i)
				ops[i] = struct{ key, value string }{key: key, value: key + "-value"}
			}
			dbBytes := buildRawEtcdSnapshot(t, ops)
			srcBytes := newTestArchiveBytes(t, archiveEntry{key: DBStoreKey, value: dbBytes})

			var out bytes.Buffer
			if err := ConvertEtcdSnapshotToKeyValueSnapshot(t.Context(), t.TempDir(), bytes.NewReader(srcBytes), &out); err != nil {
				t.Fatalf("ConvertEtcdSnapshotToKeyValueSnapshot failed: %v", err)
			}

			entries := readAllArchiveEntries(t, writeBytesToTempFile(t, out.Bytes()))
			liveKeys := 0
			for k := range entries {
				if k != RevisionStoreKey {
					liveKeys++
				}
			}
			if liveKeys != n {
				t.Fatalf("expected %d live keys, got %d", n, liveKeys)
			}
			for i := range n {
				key := fmt.Sprintf("/registry/pagination/%05d", i)
				want := key + "-value"
				got, ok := entries[key]
				if !ok {
					t.Errorf("missing key %s", key)
					continue
				}
				if string(got) != want {
					t.Errorf("key %s: expected %q, got %q", key, want, got)
				}
			}
		})
	}
}

func TestParseEtcdSnapshotArchive_CleansUpDBFileOnLaterError(t *testing.T) {
	t.Parallel()

	dbBytes := buildRawEtcdSnapshot(t, []struct{ key, value string }{{"/registry/a", "v1"}})

	// DBStoreKey is written to a temp file successfully, then the malformed
	// SkipKeysStoreKey entry that follows it fails - this exercises the leak
	// that used to occur when an error on a later archive entry discarded the
	// path to the already-written db temp file.
	srcPath := newTestArchive(t,
		archiveEntry{key: DBStoreKey, value: dbBytes},
		archiveEntry{key: SkipKeysStoreKey, value: []byte("not-json")},
	)

	tempDir := t.TempDir()
	_, err := parseEtcdSnapshotArchive(srcPath, tempDir)
	if err == nil {
		t.Fatal("expected an error from a malformed skipKeys entry")
	}

	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("failed to read temp dir: %v", err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "snapshot-") {
			t.Errorf("expected the db temp file to be cleaned up on error, found leaked file %q", e.Name())
		}
	}
}

// TestParseEtcdSnapshotArchive_DuplicateDBStoreKey verifies a malformed
// archive with two DBStoreKey entries is rejected rather than silently
// overwriting dbFile, and that the first temp file is not leaked.
func TestParseEtcdSnapshotArchive_DuplicateDBStoreKey(t *testing.T) {
	t.Parallel()

	dbBytes := buildRawEtcdSnapshot(t, []struct{ key, value string }{{"/registry/a", "v1"}})

	srcPath := newTestArchive(t,
		archiveEntry{key: DBStoreKey, value: dbBytes},
		archiveEntry{key: DBStoreKey, value: dbBytes},
	)

	tempDir := t.TempDir()
	_, err := parseEtcdSnapshotArchive(srcPath, tempDir)
	if err == nil {
		t.Fatal("expected an error for an archive with multiple DBStoreKey entries")
	}
	if !strings.Contains(err.Error(), "multiple") {
		t.Errorf("expected a duplicate-entry error, got: %v", err)
	}

	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("failed to read temp dir: %v", err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "snapshot-") {
			t.Errorf("expected the first db temp file to be cleaned up, found leaked file %q", e.Name())
		}
	}
}

// failingWriter fails every write, so the archive writer abandons the list
// channel while the list goroutine is still paging the store.
type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("sink is gone") }

// TestConvertEtcdSnapshotToKeyValueSnapshot_SinkFailureDoesNotOutliveStore
// covers the teardown ordering: the list goroutine reads the mvcc store, and
// an mvcc read racing that store's close panics rather than erroring, taking
// the whole process down from a goroutine no recover can reach. Enough keys to
// span several pages so the listing is still running when the sink gives up.
func TestConvertEtcdSnapshotToKeyValueSnapshot_SinkFailureDoesNotOutliveStore(t *testing.T) {
	t.Parallel()

	n := 5 * etcd.EtcdPaginationLimit
	ops := make([]struct{ key, value string }, n)
	for i := range n {
		key := fmt.Sprintf("/registry/teardown/%05d", i)
		ops[i] = struct{ key, value string }{key: key, value: "v"}
	}

	dbBytes := buildRawEtcdSnapshot(t, ops)
	srcBytes := newTestArchiveBytes(t, archiveEntry{key: DBStoreKey, value: dbBytes})

	err := ConvertEtcdSnapshotToKeyValueSnapshot(t.Context(), t.TempDir(), bytes.NewReader(srcBytes), failingWriter{})
	if err == nil {
		t.Fatal("expected a failing sink to surface as an error")
	}
}

func writeBytesToTempFile(t *testing.T, data []byte) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "out-")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer f.Close()
	if _, err := f.Write(data); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	return f.Name()
}
