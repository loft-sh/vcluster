package snapshot

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/coreos/go-semver/semver"
	snapshotapi "github.com/loft-sh/api/v4/pkg/snapshot"
	"github.com/loft-sh/vcluster/pkg/etcd"
	"go.etcd.io/etcd/etcdutl/v3/snapshot"
	"go.etcd.io/etcd/pkg/v3/traceutil"
	"go.etcd.io/etcd/server/v3/lease"
	"go.etcd.io/etcd/server/v3/storage/backend"
	"go.etcd.io/etcd/server/v3/storage/datadir"
	"go.etcd.io/etcd/server/v3/storage/mvcc"
	"go.uber.org/zap"
)

// noopCluster satisfies lease.Lessor's unexported cluster dependency for a
// throwaway lessor that never actually joins a cluster or checkpoints leases.
type noopCluster struct{}

func (noopCluster) Version() *semver.Version { return &semver.Version{Major: 3, Minor: 6} }

// ConvertEtcdSnapshotToKeyValueSnapshot reads a raw etcd binary snapshot
// archive (EtcdSnapshotKind, as produced for an embedded-etcd backing store)
// and writes an equivalent KeyValueSnapshotKind archive to dst. This lets a
// snapshot taken from an embedded-etcd tenant cluster be restored into a
// backing store that only supports KV-style restore (external database,
// deployed/external etcd) - e.g. migrating a tenant cluster from embedded
// etcd to an external database.
//
// The raw snapshot is decoded by restoring it into a scratch data directory
// (never the real one) and reading the resulting bbolt file directly through
// the etcd MVCC engine - no etcd server, ports, or exec'd binary involved.
func ConvertEtcdSnapshotToKeyValueSnapshot(ctx context.Context, tempDir string, src io.Reader, dst io.Writer) error {
	srcPath, err := writeTempFile(tempDir, src)
	if err != nil {
		return fmt.Errorf("failed to write source snapshot to temp file: %w", err)
	}
	defer os.Remove(srcPath)

	kind, err := getSnapshotArchiveKind(srcPath)
	if err != nil {
		return fmt.Errorf("failed to determine snapshot archive kind: %w", err)
	} else if kind != EtcdSnapshotKind {
		return fmt.Errorf("source snapshot is not an etcd snapshot (kind: %s)", kind)
	}

	parsed, err := parseEtcdSnapshotArchive(srcPath, tempDir)
	if err != nil {
		return err
	}
	defer os.Remove(parsed.DBPath)

	lg := zap.NewNop()

	// the snapshot's own recorded revision - captured for parity with live KV
	// snapshots; using it as a restore-time floor is separate, later work.
	status, err := snapshot.NewV3(lg).Status(parsed.DBPath)
	if err != nil {
		return fmt.Errorf("failed to get snapshot status: %w", err)
	}

	scratchDir, err := os.MkdirTemp(tempDir, "convert-etcd-")
	if err != nil {
		return fmt.Errorf("failed to create scratch directory: %w", err)
	}
	defer os.RemoveAll(scratchDir)

	const (
		scratchName    = "convert"
		scratchPeerURL = "https://127.0.0.1:2380"
	)
	if err := snapshot.NewV3(lg).Restore(snapshot.RestoreConfig{
		SnapshotPath:        parsed.DBPath,
		Name:                scratchName,
		OutputDataDir:       scratchDir,
		OutputWALDir:        datadir.ToWALDir(scratchDir),
		PeerURLs:            []string{scratchPeerURL},
		InitialCluster:      scratchName + "=" + scratchPeerURL,
		InitialClusterToken: "vcluster-convert",
		// Snapshots taken via etcdClient.SnapshotWithVersion (vcluster's own
		// "vcluster snapshot create") never carry the sha256 trailer that
		// etcdctl's separate "snapshot save" CLI path appends - requiring one
		// here would reject the primary input this tool exists to handle.
		// Status() above already performs a full bbolt structural integrity
		// check independent of this trailer.
		SkipHashCheck: true,
	}); err != nil {
		return fmt.Errorf("failed to restore etcd snapshot into scratch directory: %w", err)
	}

	be := backend.NewDefaultBackend(lg, datadir.ToBackendFileName(scratchDir))
	defer be.Close()

	lessor := lease.NewLessor(lg, be, noopCluster{}, lease.LessorConfig{MinLeaseTTL: 60})
	defer lessor.Stop()

	store := mvcc.New(lg, be, lessor, mvcc.StoreConfig{})
	defer store.Close()

	gzipWriter, err := gzip.NewWriterLevel(dst, 3)
	if err != nil {
		return fmt.Errorf("failed to create gzip writer: %w", err)
	}
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	if parsed.ReleaseBytes != nil {
		if err := writeArchiveEntry(tarWriter, []byte(snapshotapi.SnapshotReleaseKey), parsed.ReleaseBytes); err != nil {
			return fmt.Errorf("failed to write release: %w", err)
		}
	}
	if parsed.RequestBytes != nil {
		if err := writeArchiveEntry(tarWriter, []byte(parsed.RequestKey), parsed.RequestBytes); err != nil {
			return fmt.Errorf("failed to write request: %w", err)
		}
	}
	if err := writeArchiveEntry(tarWriter, []byte(RevisionStoreKey), []byte(strconv.FormatInt(status.Revision, 10))); err != nil {
		return fmt.Errorf("failed to write revision: %w", err)
	}

	if err := writeLiveKeyValues(ctx, tarWriter, store, parsed.SkipKeys); err != nil {
		return err
	}

	if err := tarWriter.Close(); err != nil {
		return fmt.Errorf("failed to close tar writer: %w", err)
	}
	if err := gzipWriter.Close(); err != nil {
		return fmt.Errorf("failed to close gzip writer: %w", err)
	}

	return nil
}

// parsedEtcdSnapshotArchive is the result of parseEtcdSnapshotArchive. DBPath
// names a temp file under the caller-supplied tempDir - the caller must
// remove it.
type parsedEtcdSnapshotArchive struct {
	DBPath       string
	ReleaseBytes []byte
	RequestKey   string
	RequestBytes []byte
	SkipKeys     map[string]struct{}
}

// parseEtcdSnapshotArchive extracts the raw etcd db file (to a temp file under
// tempDir - caller must remove it), the optional release/request passthrough
// entries, and the optional skip-keys set from an EtcdSnapshotKind archive.
func parseEtcdSnapshotArchive(srcPath, tempDir string) (result parsedEtcdSnapshotArchive, err error) {
	reader, err := os.Open(srcPath)
	if err != nil {
		return parsedEtcdSnapshotArchive{}, fmt.Errorf("failed to open source snapshot: %w", err)
	}
	defer reader.Close()

	gzipReader, err := gzip.NewReader(reader)
	if err != nil {
		return parsedEtcdSnapshotArchive{}, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzipReader.Close()

	// dbFile tracks the temp file written for DBStoreKey independently of
	// result.DBPath, which every error path below leaves unset - without
	// this, a failure on a later tar entry would lose the path to a file that
	// already exists on disk, leaking it.
	var dbFile string
	defer func() {
		if err != nil && dbFile != "" {
			_ = os.Remove(dbFile)
		}
	}()

	tarReader := tar.NewReader(gzipReader)
	for {
		header, nextErr := tarReader.Next()
		if nextErr != nil {
			if errors.Is(nextErr, io.EOF) {
				break
			}
			return parsedEtcdSnapshotArchive{}, fmt.Errorf("failed to read tar header: %w", nextErr)
		}

		switch {
		case header.Name == snapshotapi.SnapshotReleaseKey:
			result.ReleaseBytes, err = io.ReadAll(tarReader)
			if err != nil {
				return parsedEtcdSnapshotArchive{}, fmt.Errorf("failed to read release: %w", err)
			}
		case strings.HasPrefix(header.Name, RequestStoreKey):
			result.RequestKey = header.Name
			result.RequestBytes, err = io.ReadAll(tarReader)
			if err != nil {
				return parsedEtcdSnapshotArchive{}, fmt.Errorf("failed to read request: %w", err)
			}
		case header.Name == DBStoreKey:
			dbFile, err = writeTempFile(tempDir, tarReader)
			if err != nil {
				return parsedEtcdSnapshotArchive{}, fmt.Errorf("failed to write etcd snapshot to temp file: %w", err)
			}
		case header.Name == SkipKeysStoreKey:
			skipKeysBytes, readErr := io.ReadAll(tarReader)
			if readErr != nil {
				return parsedEtcdSnapshotArchive{}, fmt.Errorf("failed to read skipKeys: %w", readErr)
			}
			result.SkipKeys = make(map[string]struct{})
			if err := json.Unmarshal(skipKeysBytes, &result.SkipKeys); err != nil {
				return parsedEtcdSnapshotArchive{}, fmt.Errorf("failed to unmarshal skipKeys: %w", err)
			}
		}
	}

	if dbFile == "" {
		return parsedEtcdSnapshotArchive{}, fmt.Errorf("failed to find etcd snapshot in source archive")
	}

	result.DBPath = dbFile
	return result, nil
}

// writeLiveKeyValues pages through every live key/value in store (tombstoned/
// superseded revisions already resolved away by the MVCC engine) and writes
// each one not present in skipKeys into tarWriter, mirroring how
// writeKeyValueSnapshot streams a live cluster's keys.
func writeLiveKeyValues(ctx context.Context, tarWriter *tar.Writer, store mvcc.KV, skipKeys map[string]struct{}) error {
	startKey := []byte{0}
	var pinnedRev int64

	for {
		txn := store.Read(mvcc.ConcurrentReadTxMode, traceutil.TODO())
		result, err := txn.Range(ctx, startKey, []byte{}, mvcc.RangeOptions{
			Limit: etcd.EtcdPaginationLimit,
			Rev:   pinnedRev,
		})
		txn.End()
		if err != nil {
			return fmt.Errorf("failed to range over restored keys: %w", err)
		}
		if pinnedRev == 0 {
			pinnedRev = result.Rev
		}

		for _, kv := range result.KVs {
			if _, ok := skipKeys[string(kv.Key)]; ok {
				continue
			}
			if err := writeArchiveEntry(tarWriter, kv.Key, kv.Value); err != nil {
				return fmt.Errorf("failed to write key %s: %w", kv.Key, err)
			}
		}

		if len(result.KVs) < etcd.EtcdPaginationLimit {
			return nil
		}
		startKey = append(append([]byte{}, result.KVs[len(result.KVs)-1].Key...), 0x00)
	}
}
