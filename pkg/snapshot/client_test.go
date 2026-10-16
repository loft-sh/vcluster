package snapshot

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	snapshotapi "github.com/loft-sh/api/v4/pkg/snapshot"
	"github.com/loft-sh/api/v4/pkg/snapshot/storage/container"
	"github.com/loft-sh/vcluster/pkg/etcd"
)

// fakeKeyValueSource is a keyValueSource fake serving a fixed set of values.
type fakeKeyValueSource struct {
	revision    int64
	revisionErr error
	values      []etcd.Value
	listErr     error
}

func (f *fakeKeyValueSource) CurrentRevision(context.Context) (int64, error) {
	return f.revision, f.revisionErr
}

func (f *fakeKeyValueSource) ListStream(context.Context, string) <-chan *etcd.ValueOrError {
	ch := make(chan *etcd.ValueOrError, len(f.values)+1)
	for _, v := range f.values {
		ch <- &etcd.ValueOrError{Value: v}
	}
	if f.listErr != nil {
		ch <- &etcd.ValueOrError{Error: f.listErr}
	}
	close(ch)
	return ch
}

// the real etcd client must keep satisfying the narrowed read surface
var _ keyValueSource = etcd.Client(nil)

func TestWriteKeyValueSnapshot_WritesRevision(t *testing.T) {
	t.Parallel()

	fake := &fakeKeyValueSource{
		revision: 4242,
		values: []etcd.Value{
			{Key: []byte("/registry/a"), Data: []byte("va1")},
			{Key: []byte("/registry/b"), Data: []byte("vb1")},
		},
	}

	storePath := filepath.Join(t.TempDir(), "snapshot.tar.gz")
	objectStore := container.NewStore(&snapshotapi.ContainerOptions{Path: storePath})

	if err := writeKeyValueSnapshot(t.Context(), fake, objectStore, archiveMetadata{}); err != nil {
		t.Fatalf("writeKeyValueSnapshot failed: %v", err)
	}

	entries := readAllArchiveEntries(t, storePath)

	revBytes, ok := entries[RevisionStoreKey]
	if !ok {
		t.Fatal("expected a RevisionStoreKey entry")
	}
	if string(revBytes) != "4242" {
		t.Errorf("expected revision 4242, got %q", revBytes)
	}

	if string(entries["/registry/a"]) != "va1" {
		t.Errorf("expected /registry/a=va1, got %q", entries["/registry/a"])
	}
	if string(entries["/registry/b"]) != "vb1" {
		t.Errorf("expected /registry/b=vb1, got %q", entries["/registry/b"])
	}

	// the leading RevisionStoreKey must not be mistaken for a metadata header
	// that flips kind detection; a KV snapshot has no DBStoreKey so it stays
	// KeyValueSnapshotKind
	kind, err := getSnapshotArchiveKind(storePath)
	if err != nil {
		t.Fatalf("getSnapshotArchiveKind failed: %v", err)
	}
	if kind != KeyValueSnapshotKind {
		t.Fatalf("expected KeyValueSnapshotKind, got %s", kind)
	}
}

// TestWriteKeyValueSnapshot_DetectedAsKeyValueKind_WithReleaseAndRequest pins
// the ordering with both optional headers present, so RevisionStoreKey lands
// in the third slot getSnapshotArchiveKind reserves for DBStoreKey. It must
// still fall through to KeyValueSnapshotKind, not be read as an EtcdSnapshot.
func TestWriteKeyValueSnapshot_DetectedAsKeyValueKind_WithReleaseAndRequest(t *testing.T) {
	t.Parallel()

	fake := &fakeKeyValueSource{
		revision: 7,
		values:   []etcd.Value{{Key: []byte("/registry/a"), Data: []byte("va1")}},
	}

	storePath := filepath.Join(t.TempDir(), "snapshot.tar.gz")
	objectStore := container.NewStore(&snapshotapi.ContainerOptions{Path: storePath})

	c := &Client{
		Request: &snapshotapi.Request{},
		Options: snapshotapi.Options{Release: &snapshotapi.HelmRelease{}},
	}
	meta, err := c.buildArchiveMetadata()
	if err != nil {
		t.Fatalf("buildArchiveMetadata failed: %v", err)
	}
	if err := writeKeyValueSnapshot(t.Context(), fake, objectStore, meta); err != nil {
		t.Fatalf("writeKeyValueSnapshot failed: %v", err)
	}

	kind, err := getSnapshotArchiveKind(storePath)
	if err != nil {
		t.Fatalf("getSnapshotArchiveKind failed: %v", err)
	}
	if kind != KeyValueSnapshotKind {
		t.Fatalf("expected KeyValueSnapshotKind, got %s", kind)
	}
}

// TestSnapshotMetadataKeysCarryPrefix guards the invariant the backup/restore
// skip logic relies on: every archived metadata key starts with
// SnapshotMetadataPrefix. Local keys derive from the prefix by construction;
// this also pins snapshotapi.SnapshotReleaseKey, which is defined out-of-repo
// and only happens to share the prefix.
func TestSnapshotMetadataKeysCarryPrefix(t *testing.T) {
	t.Parallel()

	keys := []string{
		snapshotapi.SnapshotReleaseKey,
		RequestStoreKey,
		DBStoreKey,
		SkipKeysStoreKey,
		RevisionStoreKey,
	}
	for _, k := range keys {
		if !strings.HasPrefix(k, SnapshotMetadataPrefix) {
			t.Errorf("metadata key %q must start with %q or the skip guards miss it", k, SnapshotMetadataPrefix)
		}
	}
}

// TestWriteKeyValueSnapshot_SkipsPollutedMetadataKey guards that a metadata
// key a previous restore accidentally persisted into etcd is not re-emitted
// from the list stream, so the archive holds exactly one revision entry.
func TestWriteKeyValueSnapshot_SkipsPollutedMetadataKey(t *testing.T) {
	t.Parallel()

	fake := &fakeKeyValueSource{
		revision: 4242,
		values: []etcd.Value{
			{Key: []byte("/registry/a"), Data: []byte("va1")},
			// stale copy from an old buggy restore
			{Key: []byte(RevisionStoreKey), Data: []byte("999")},
		},
	}

	storePath := filepath.Join(t.TempDir(), "snapshot.tar.gz")
	objectStore := container.NewStore(&snapshotapi.ContainerOptions{Path: storePath})

	if err := writeKeyValueSnapshot(t.Context(), fake, objectStore, archiveMetadata{}); err != nil {
		t.Fatalf("writeKeyValueSnapshot failed: %v", err)
	}

	// readAllArchiveEntries maps by key, so a duplicate would silently
	// overwrite; assert the single surviving value is the freshly pinned one.
	entries := readAllArchiveEntries(t, storePath)
	if string(entries[RevisionStoreKey]) != "4242" {
		t.Errorf("expected pinned revision 4242, got %q", entries[RevisionStoreKey])
	}
}

// TestWriteKeyValueSnapshot_CancelledSourceNeverReportsSuccess guards against
// a truncated archive being reported as a complete one. A source that aborts
// on cancellation just closes its channel, and that close races the ctx.Done()
// arm of the select loop - so the run is repeated to give the losing side of
// that race a chance to slip through.
func TestWriteKeyValueSnapshot_CancelledSourceNeverReportsSuccess(t *testing.T) {
	t.Parallel()

	for i := range 200 {
		fake := &fakeKeyValueSource{
			revision: 1,
			values: []etcd.Value{
				{Key: []byte("/registry/a"), Data: []byte("va1")},
				{Key: []byte("/registry/b"), Data: []byte("vb1")},
			},
		}

		storePath := filepath.Join(t.TempDir(), "snapshot.tar.gz")
		objectStore := container.NewStore(&snapshotapi.ContainerOptions{Path: storePath})

		// the source stopped early, so the values above are an incomplete listing
		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		if err := writeKeyValueSnapshot(ctx, fake, objectStore, archiveMetadata{}); err == nil {
			t.Fatalf("run %d: cancelled source reported success; the archive is truncated", i)
		}
	}
}

func TestWriteKeyValueSnapshot_CurrentRevisionError(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("boom")
	fake := &fakeKeyValueSource{revisionErr: sentinel}

	storePath := filepath.Join(t.TempDir(), "snapshot.tar.gz")
	objectStore := container.NewStore(&snapshotapi.ContainerOptions{Path: storePath})

	err := writeKeyValueSnapshot(t.Context(), fake, objectStore, archiveMetadata{})
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected CurrentRevision error to propagate, got: %v", err)
	}
}

func TestWriteKeyValueSnapshot_ListStreamError(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("boom")
	fake := &fakeKeyValueSource{
		revision: 1,
		values:   []etcd.Value{{Key: []byte("/registry/a"), Data: []byte("va1")}},
		listErr:  sentinel,
	}

	storePath := filepath.Join(t.TempDir(), "snapshot.tar.gz")
	objectStore := container.NewStore(&snapshotapi.ContainerOptions{Path: storePath})

	err := writeKeyValueSnapshot(t.Context(), fake, objectStore, archiveMetadata{})
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected ListStream error to propagate, got: %v", err)
	}
}

// TestWriteKeyValueSnapshot_NoGoroutineLeakOnEarlyError guards against the
// upload goroutine blocking forever on an unbuffered errChan send when
// writeKeyValueSnapshot returns early (e.g. before etcd is even listed).
// Deliberately not run in parallel with other tests: it compares
// process-wide goroutine counts and needs a quiet baseline to be reliable.
func TestWriteKeyValueSnapshot_NoGoroutineLeakOnEarlyError(t *testing.T) {
	sentinel := errors.New("boom")
	fake := &fakeKeyValueSource{revisionErr: sentinel}

	storePath := filepath.Join(t.TempDir(), "snapshot.tar.gz")
	objectStore := container.NewStore(&snapshotapi.ContainerOptions{Path: storePath})

	runtime.GC()
	before := runtime.NumGoroutine()

	if err := writeKeyValueSnapshot(t.Context(), fake, objectStore, archiveMetadata{}); !errors.Is(err, sentinel) {
		t.Fatalf("expected CurrentRevision error to propagate, got: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		after := runtime.NumGoroutine()
		if after <= before {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("upload goroutine appears leaked: goroutines before=%d, after=%d", before, after)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

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
