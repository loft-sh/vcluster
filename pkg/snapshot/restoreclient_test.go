package snapshot

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	snapshotapi "github.com/loft-sh/api/v4/pkg/snapshot"
)

func TestSnapshotRestoreBumpRevision(t *testing.T) {
	tests := []struct {
		name             string
		latestRevision   int64
		snapshotRevision int64
		bumpRevision     int64
		expected         uint64
	}{
		{
			name:             "latest greater than snapshot",
			latestRevision:   1500,
			snapshotRevision: 1000,
			bumpRevision:     1000,
			expected:         1500,
		},
		{
			name:             "snapshot greater than latest",
			latestRevision:   800,
			snapshotRevision: 1000,
			bumpRevision:     1000,
			expected:         1000,
		},
		{
			name:             "latest equal to snapshot",
			latestRevision:   1000,
			snapshotRevision: 1000,
			bumpRevision:     1000,
			expected:         1000,
		},
		{
			name:             "all zeros",
			latestRevision:   0,
			snapshotRevision: 0,
			bumpRevision:     1000,
			expected:         1000,
		},
		{
			name:             "snapshot and bump zero",
			latestRevision:   500,
			snapshotRevision: 0,
			bumpRevision:     1000,
			expected:         1500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := snapshotRestoreBumpRevision(tt.latestRevision, tt.snapshotRevision, tt.bumpRevision)
			if result != tt.expected {
				t.Errorf("snapshotRestoreBumpRevision(%d, %d, %d) = %d; want %d",
					tt.latestRevision, tt.snapshotRevision, tt.bumpRevision, result, tt.expected)
			}
		})
	}
}

func TestGetSnapshotArchiveKind(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(t *testing.T) string
		wantKind   SnapshotKind
		wantErr    bool
		wantErrSub string
	}{
		{
			name: "etcd snapshot - DBStoreKey first",
			setup: func(t *testing.T) string {
				return newTestArchive(t,
					archiveEntry{key: DBStoreKey, value: []byte("db-bytes")},
				)
			},
			wantKind: EtcdSnapshotKind,
		},
		{
			name: "etcd snapshot - release then DBStoreKey",
			setup: func(t *testing.T) string {
				return newTestArchive(t,
					archiveEntry{key: snapshotapi.SnapshotReleaseKey, value: []byte("{}")},
					archiveEntry{key: DBStoreKey, value: []byte("db-bytes")},
				)
			},
			wantKind: EtcdSnapshotKind,
		},
		{
			name: "kv snapshot - registry key first",
			setup: func(t *testing.T) string {
				return newTestArchive(t,
					archiveEntry{key: "/registry/pods/default/foo", value: []byte("v")},
				)
			},
			wantKind: KeyValueSnapshotKind,
		},
		{
			name: "kv snapshot - release then registry key",
			setup: func(t *testing.T) string {
				return newTestArchive(t,
					archiveEntry{key: snapshotapi.SnapshotReleaseKey, value: []byte("{}")},
					archiveEntry{key: "/registry/configmaps/default/x", value: []byte("v")},
				)
			},
			wantKind: KeyValueSnapshotKind,
		},
		{
			name: "kv snapshot - request key first",
			setup: func(t *testing.T) string {
				return newTestArchive(t,
					archiveEntry{key: RequestStoreKey + "/v1", value: []byte("{}")},
				)
			},
			wantKind: KeyValueSnapshotKind,
		},
		{
			name: "file does not exist",
			setup: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "missing.tar.gz")
			},
			wantKind:   UnknownSnapshotKind,
			wantErr:    true,
			wantErrSub: "open file",
		},
		{
			name: "not gzip",
			setup: func(t *testing.T) string {
				p := filepath.Join(t.TempDir(), "plain.txt")
				if err := os.WriteFile(p, []byte("not gzip data"), 0o644); err != nil {
					t.Fatalf("write file: %v", err)
				}
				return p
			},
			wantKind:   UnknownSnapshotKind,
			wantErr:    true,
			wantErrSub: "create gzip reader",
		},
		{
			name: "gzip but not tar",
			setup: func(t *testing.T) string {
				p := filepath.Join(t.TempDir(), "garbage.tar.gz")
				f, err := os.Create(p)
				if err != nil {
					t.Fatalf("create file: %v", err)
				}
				defer f.Close()
				gw := gzip.NewWriter(f)
				if _, err := gw.Write([]byte("this is not a tar stream")); err != nil {
					t.Fatalf("write gzip: %v", err)
				}
				if err := gw.Close(); err != nil {
					t.Fatalf("close gzip: %v", err)
				}
				return p
			},
			wantKind:   UnknownSnapshotKind,
			wantErr:    true,
			wantErrSub: "read tar header",
		},
		{
			name: "empty tar.gz",
			setup: func(t *testing.T) string {
				return newTestArchive(t)
			},
			wantKind: KeyValueSnapshotKind,
		},
		{
			name: "only release key, no second entry",
			setup: func(t *testing.T) string {
				return newTestArchive(t,
					archiveEntry{key: snapshotapi.SnapshotReleaseKey, value: []byte("{}")},
				)
			},
			wantKind: KeyValueSnapshotKind,
		},
		{
			name: "empty file",
			setup: func(t *testing.T) string {
				p := filepath.Join(t.TempDir(), "empty.bin")
				if err := os.WriteFile(p, nil, 0o644); err != nil {
					t.Fatalf("write file: %v", err)
				}
				return p
			},
			wantKind:   UnknownSnapshotKind,
			wantErr:    true,
			wantErrSub: "create gzip reader",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup(t)
			got, err := getSnapshotArchiveKind(path)
			if got != tt.wantKind {
				t.Errorf("kind: got %q, want %q", got, tt.wantKind)
			}
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if tt.wantErrSub != "" && !strings.Contains(err.Error(), tt.wantErrSub) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.wantErrSub)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

type archiveEntry struct {
	key   string
	value []byte
}

func newTestArchive(t *testing.T, entries ...archiveEntry) string {
	t.Helper()

	tempFile, err := os.CreateTemp(t.TempDir(), "test-")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	defer func() {
		if err := tempFile.Close(); err != nil {
			t.Fatalf("failed to close temp file: %v", err)
		}
	}()

	// use same compression level as etcd snapshot creation process
	gzipWriter, err := gzip.NewWriterLevel(tempFile, 3)
	if err != nil {
		t.Fatalf("failed to create gzip writer: %v", err)
	}

	tarWriter := tar.NewWriter(gzipWriter)

	for _, e := range entries {
		if err := tarWriter.WriteHeader(&tar.Header{
			Typeflag: tar.TypeReg,
			Name:     e.key,
			Size:     int64(len(e.value)),
			Mode:     0666,
		}); err != nil {
			t.Fatalf("failed to write header: %v", err)
		}

		if _, err := tarWriter.Write(e.value); err != nil {
			t.Fatalf("failed to write value: %v", err)
		}
	}

	if err := tarWriter.Close(); err != nil {
		t.Fatalf("failed to close tar writer: %v", err)
	}

	if err := gzipWriter.Close(); err != nil {
		t.Fatalf("failed to close gzip writer: %v", err)
	}

	return tempFile.Name()
}
