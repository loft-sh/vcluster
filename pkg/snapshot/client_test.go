package snapshot

import (
	"context"
	"errors"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	snapshotapi "github.com/loft-sh/api/v4/pkg/snapshot"
	"github.com/loft-sh/api/v4/pkg/snapshot/storage/container"
	"github.com/loft-sh/vcluster/pkg/etcd"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// fakeEtcdClient is a minimal etcd.Client fake for testing code that only
// needs a handful of methods - unimplemented methods panic if called, so a
// test that exercises them accidentally fails loudly instead of silently
// returning zero values.
type fakeEtcdClient struct {
	revision    int64
	revisionErr error
	values      []etcd.Value
	listErr     error
}

func (f *fakeEtcdClient) CurrentRevision(context.Context) (int64, error) {
	return f.revision, f.revisionErr
}

func (f *fakeEtcdClient) ListStream(context.Context, string) <-chan *etcd.ValueOrError {
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

func (f *fakeEtcdClient) List(context.Context, string) ([]etcd.Value, error) {
	panic("not implemented")
}
func (f *fakeEtcdClient) Watch(context.Context, string) clientv3.WatchChan { panic("not implemented") }
func (f *fakeEtcdClient) Get(context.Context, string) (etcd.Value, error) {
	panic("not implemented")
}
func (f *fakeEtcdClient) Put(context.Context, string, []byte) (int64, error) {
	panic("not implemented")
}
func (f *fakeEtcdClient) PutAtRevision(context.Context, string, int64, []byte) (int64, error) {
	panic("not implemented")
}
func (f *fakeEtcdClient) Delete(context.Context, string) error       { panic("not implemented") }
func (f *fakeEtcdClient) DeletePrefix(context.Context, string) error { panic("not implemented") }
func (f *fakeEtcdClient) Compact(context.Context, int64) error       { panic("not implemented") }
func (f *fakeEtcdClient) Close() error                               { return nil }
func (f *fakeEtcdClient) SnapshotWithVersion(context.Context) (*clientv3.SnapshotResponse, error) {
	panic("not implemented")
}

func TestWriteKeyValueSnapshot_WritesRevision(t *testing.T) {
	t.Parallel()

	fake := &fakeEtcdClient{
		revision: 4242,
		values: []etcd.Value{
			{Key: []byte("/registry/a"), Data: []byte("va1")},
			{Key: []byte("/registry/b"), Data: []byte("vb1")},
		},
	}

	storePath := filepath.Join(t.TempDir(), "snapshot.tar.gz")
	objectStore := container.NewStore(&snapshotapi.ContainerOptions{Path: storePath})

	c := &Client{}
	if err := c.writeKeyValueSnapshot(context.Background(), fake, objectStore); err != nil {
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
}

func TestWriteKeyValueSnapshot_CurrentRevisionError(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("boom")
	fake := &fakeEtcdClient{revisionErr: sentinel}

	storePath := filepath.Join(t.TempDir(), "snapshot.tar.gz")
	objectStore := container.NewStore(&snapshotapi.ContainerOptions{Path: storePath})

	c := &Client{}
	err := c.writeKeyValueSnapshot(context.Background(), fake, objectStore)
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected CurrentRevision error to propagate, got: %v", err)
	}
}

func TestWriteKeyValueSnapshot_ListStreamError(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("boom")
	fake := &fakeEtcdClient{
		revision: 1,
		values:   []etcd.Value{{Key: []byte("/registry/a"), Data: []byte("va1")}},
		listErr:  sentinel,
	}

	storePath := filepath.Join(t.TempDir(), "snapshot.tar.gz")
	objectStore := container.NewStore(&snapshotapi.ContainerOptions{Path: storePath})

	c := &Client{}
	err := c.writeKeyValueSnapshot(context.Background(), fake, objectStore)
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
	fake := &fakeEtcdClient{revisionErr: sentinel}

	storePath := filepath.Join(t.TempDir(), "snapshot.tar.gz")
	objectStore := container.NewStore(&snapshotapi.ContainerOptions{Path: storePath})

	c := &Client{}

	runtime.GC()
	before := runtime.NumGoroutine()

	if err := c.writeKeyValueSnapshot(context.Background(), fake, objectStore); !errors.Is(err, sentinel) {
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
