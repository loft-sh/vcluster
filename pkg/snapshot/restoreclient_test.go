package snapshot

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	snapshotapi "github.com/loft-sh/api/v4/pkg/snapshot"
	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/etcd"
	"github.com/loft-sh/vcluster/pkg/k8s"
	_ "modernc.org/sqlite"
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
			name: "etcd snapshot - release, request then DBStoreKey",
			setup: func(t *testing.T) string {
				return newTestArchive(t,
					archiveEntry{key: snapshotapi.SnapshotReleaseKey, value: []byte("{}")},
					archiveEntry{key: RequestStoreKey + "/v1", value: []byte("{}")},
					archiveEntry{key: DBStoreKey, value: []byte("db-bytes")},
				)
			},
			wantKind: EtcdSnapshotKind,
		},
		{
			name: "etcd snapshot - request then DBStoreKey (no release)",
			setup: func(t *testing.T) string {
				return newTestArchive(t,
					archiveEntry{key: RequestStoreKey + "/v1", value: []byte("{}")},
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
			name: "kv snapshot - only request key, EOF after",
			setup: func(t *testing.T) string {
				return newTestArchive(t,
					archiveEntry{key: RequestStoreKey + "/v1", value: []byte("{}")},
				)
			},
			wantKind: KeyValueSnapshotKind,
		},
		{
			name: "kv snapshot - release then request key, EOF after",
			setup: func(t *testing.T) string {
				return newTestArchive(t,
					archiveEntry{key: snapshotapi.SnapshotReleaseKey, value: []byte("{}")},
					archiveEntry{key: RequestStoreKey + "/v1", value: []byte("{}")},
				)
			},
			wantKind: KeyValueSnapshotKind,
		},
		{
			name: "kv snapshot - release, request then non-DB key",
			setup: func(t *testing.T) string {
				return newTestArchive(t,
					archiveEntry{key: snapshotapi.SnapshotReleaseKey, value: []byte("{}")},
					archiveEntry{key: RequestStoreKey + "/v1", value: []byte("{}")},
					archiveEntry{key: "/registry/pods/default/foo", value: []byte("v")},
				)
			},
			wantKind: KeyValueSnapshotKind,
		},
		{
			name: "kv snapshot - exact RequestStoreKey (no suffix)",
			setup: func(t *testing.T) string {
				return newTestArchive(t,
					archiveEntry{key: RequestStoreKey, value: []byte("{}")},
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

func TestGetLatestRevisionSQLite(t *testing.T) {
	tests := []struct {
		name         string
		setup        func(t *testing.T) string
		wantRevision int64
		wantErr      bool
	}{
		{
			name: "database file does not exist",
			setup: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "state.db")
			},
			wantRevision: 0,
		},
		{
			name: "database file exists but has no schema",
			setup: func(t *testing.T) string {
				file := filepath.Join(t.TempDir(), "state.db")
				f, err := os.Create(file)
				if err != nil {
					t.Fatalf("failed to create database file: %v", err)
				}
				if err := f.Close(); err != nil {
					t.Fatalf("failed to close database file: %v", err)
				}
				return file
			},
			wantRevision: 0,
		},
		{
			name: "kine table exists but is empty",
			setup: func(t *testing.T) string {
				return newTestKineDatabase(t, 0)
			},
			wantRevision: 0,
		},
		{
			name: "kine table has keys",
			setup: func(t *testing.T) string {
				return newTestKineDatabase(t, 3)
			},
			wantRevision: 3,
		},
		{
			name:    "database file is not a SQLite database",
			setup:   newTestGarbageFile,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			revision, err := getLatestRevisionSQLite(context.Background(), tt.setup(t))
			if tt.wantErr {
				if err == nil {
					t.Fatal("getLatestRevisionSQLite() expected an error, got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("getLatestRevisionSQLite() unexpected error: %v", err)
			}
			if revision != tt.wantRevision {
				t.Errorf("getLatestRevisionSQLite() = %d; want %d", revision, tt.wantRevision)
			}
		})
	}
}

func TestWriteLatestRevisionSQLite(t *testing.T) {
	tests := []struct {
		name string
		keys int
	}{
		{
			name: "kine table is empty so SQLITE_SEQUENCE has no row to update",
			keys: 0,
		},
		{
			name: "kine table has keys so SQLITE_SEQUENCE row is updated",
			keys: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := newTestKineDatabase(t, tt.keys)

			if err := writeLatestRevisionSQLite(context.Background(), file, 1234); err != nil {
				t.Fatalf("writeLatestRevisionSQLite() unexpected error: %v", err)
			}

			revision, err := getLatestRevisionSQLite(context.Background(), file)
			if err != nil {
				t.Fatalf("getLatestRevisionSQLite() unexpected error: %v", err)
			}
			if revision != 1234 {
				t.Errorf("latest revision = %d; want 1234", revision)
			}
		})
	}
}

func TestKineSchemaCreated(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(t *testing.T) string
		wantExists bool
		wantErr    bool
	}{
		{
			name: "database file without schema",
			setup: func(t *testing.T) string {
				file := filepath.Join(t.TempDir(), "state.db")
				if err := os.WriteFile(file, nil, 0o600); err != nil {
					t.Fatalf("failed to create database file: %v", err)
				}
				return file
			},
			wantExists: false,
		},
		{
			name: "database with the kine schema",
			setup: func(t *testing.T) string {
				return newTestKineDatabase(t, 0)
			},
			wantExists: true,
		},
		{
			name:    "database file is not a SQLite database",
			setup:   newTestGarbageFile,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exists, err := kineSchemaCreated(context.Background(), tt.setup(t))
			if tt.wantErr {
				if err == nil {
					t.Fatal("kineSchemaCreated() expected an error, got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("kineSchemaCreated() unexpected error: %v", err)
			}
			if exists != tt.wantExists {
				t.Errorf("kineSchemaCreated() = %t; want %t", exists, tt.wantExists)
			}
		})
	}
}

// newTestGarbageFile creates a file SQLite cannot read as a database.
func newTestGarbageFile(t *testing.T) string {
	t.Helper()

	file := filepath.Join(t.TempDir(), "state.db")
	if err := os.WriteFile(file, []byte(strings.Repeat("not a sqlite database ", 64)), 0o600); err != nil {
		t.Fatalf("failed to write garbage file: %v", err)
	}
	return file
}

// newTestKineDatabase creates a SQLite database with kine's schema and the given
// number of keys, and returns its path.
func newTestKineDatabase(t *testing.T, keys int) string {
	t.Helper()

	file := filepath.Join(t.TempDir(), "state.db")
	db, err := sql.Open("sqlite", file+k8s.SQLiteParams)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS kine (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT, value BLOB)`); err != nil {
		t.Fatalf("failed to create kine table: %v", err)
	}

	for i := range keys {
		if _, err := db.Exec(`INSERT INTO kine (name, value) VALUES (?, ?)`, fmt.Sprintf("/registry/key-%d", i), []byte("value")); err != nil {
			t.Fatalf("failed to insert kine key: %v", err)
		}
	}

	return file
}

// A kine that still serves would keep writing into the files the restore moves aside.
func TestNewRestoreEtcdClientRefusesServingEmbeddedDatabase(t *testing.T) {
	dbFile := newTestKineDatabase(t, 1)
	origDB, origProbe := constants.K8sSqliteDatabase, backingStoreReachable
	t.Cleanup(func() {
		constants.K8sSqliteDatabase = origDB
		backingStoreReachable = origProbe
	})
	constants.K8sSqliteDatabase = dbFile

	var probed string
	backingStoreReachable = func(_ context.Context, endpoint string, _ *etcd.Certificates) bool {
		probed = endpoint
		return true
	}

	client, _, err := newRestoreEtcdClient(context.Background(), &config.VirtualClusterConfig{})
	if !errors.Is(err, errBackingStoreServing) {
		t.Fatalf("newRestoreEtcdClient error = %v; want %v", err, errBackingStoreServing)
	}
	if client != nil {
		t.Errorf("newRestoreEtcdClient returned a client despite refusing the restore")
	}
	if probed != constants.K8sKineEndpoint {
		t.Errorf("probed endpoint = %q; want the kine endpoint %q", probed, constants.K8sKineEndpoint)
	}
	if _, err := os.Stat(dbFile); err != nil {
		t.Errorf("database was touched by the refused restore: %v", err)
	}
	if _, err := os.Stat(dbFile + ".backup"); !os.IsNotExist(err) {
		t.Errorf("refused restore moved the database aside: stat backup err = %v", err)
	}
}
