package snapshot

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"testing"

	snapshotapi "github.com/loft-sh/api/v4/pkg/snapshot"
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
	err := ConvertEtcdSnapshotToKeyValueSnapshot(context.Background(), t.TempDir(), bytes.NewReader(srcBytes), &out)
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

	revBytes, ok := entries[RevisionStoreKey]
	if !ok {
		t.Fatal("expected a RevisionStoreKey entry in the output archive")
	}
	if string(revBytes) == "" {
		t.Fatal("expected a non-empty recorded revision")
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
	if err := ConvertEtcdSnapshotToKeyValueSnapshot(context.Background(), t.TempDir(), bytes.NewReader(srcBytes), &out); err != nil {
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
	if err := ConvertEtcdSnapshotToKeyValueSnapshot(context.Background(), t.TempDir(), bytes.NewReader(srcBytes), &out); err != nil {
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

func TestConvertEtcdSnapshotToKeyValueSnapshot_EmptyDatabase(t *testing.T) {
	t.Parallel()

	dbBytes := buildRawEtcdSnapshot(t, nil)
	srcBytes := newTestArchiveBytes(t, archiveEntry{key: DBStoreKey, value: dbBytes})

	var out bytes.Buffer
	if err := ConvertEtcdSnapshotToKeyValueSnapshot(context.Background(), t.TempDir(), bytes.NewReader(srcBytes), &out); err != nil {
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
	err := ConvertEtcdSnapshotToKeyValueSnapshot(context.Background(), t.TempDir(), bytes.NewReader(srcBytes), &out)
	if err == nil {
		t.Fatal("expected an error when converting an already KeyValueSnapshotKind archive")
	}
}

func TestConvertEtcdSnapshotToKeyValueSnapshot_MissingDBStoreKey(t *testing.T) {
	t.Parallel()

	// no DBStoreKey entry at all, but also no ordinary key either - forces
	// getSnapshotArchiveKind down the EtcdSnapshotKind-only path is not
	// possible without DBStoreKey, so use a release-only archive to still
	// exercise the "malformed etcd snapshot" error from parseEtcdSnapshotArchive
	// by asserting kind detection directly instead.
	srcBytes := newTestArchiveBytes(t, archiveEntry{key: snapshotapi.SnapshotReleaseKey, value: []byte("{}")})

	var out bytes.Buffer
	err := ConvertEtcdSnapshotToKeyValueSnapshot(context.Background(), t.TempDir(), bytes.NewReader(srcBytes), &out)
	if err == nil {
		t.Fatal("expected an error for an archive without a DBStoreKey entry")
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

// readAllArchiveEntries reads every tar entry of a gzip'd tar file at path
// into a map keyed by entry name.
func readAllArchiveEntries(t *testing.T, path string) map[string][]byte {
	t.Helper()

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("failed to open archive: %v", err)
	}
	defer f.Close()

	gzipReader, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("failed to create gzip reader: %v", err)
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	entries := map[string][]byte{}
	for {
		key, value, err := readArchiveEntry(tarReader)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			t.Fatalf("failed to read archive entry: %v", err)
		}
		entries[string(key)] = value
	}
	return entries
}
