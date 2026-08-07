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
	"strings"

	"github.com/coreos/go-semver/semver"
	snapshotapi "github.com/loft-sh/api/v4/pkg/snapshot"
	"github.com/loft-sh/vcluster/pkg/etcd"
	clientv3 "go.etcd.io/etcd/client/v3"
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
	}

	if kind != EtcdSnapshotKind {
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

	kvSource := &mvccKeyValueSource{store: store, revision: status.Revision}
	listCtx, cancelList := context.WithCancel(ctx)
	// runs before the store teardown deferred above: the list goroutine reads
	// that store, so it has to be stopped and joined while it is still open
	defer func() {
		cancelList()
		kvSource.wait()
	}()

	// reuse the live snapshot path's archive writer so a converted archive is
	// byte-compatible with one taken from a running cluster by construction
	return writeKeyValueSnapshot(listCtx, kvSource,
		putObjectFunc(func(_ context.Context, body io.Reader) error {
			_, err := io.Copy(dst, body)
			return err
		}),
		parsed.archiveMetadata,
	)
}

// parsedEtcdSnapshotArchive is the result of parseEtcdSnapshotArchive. DBPath
// names a temp file under the caller-supplied tempDir - the caller must
// remove it.
type parsedEtcdSnapshotArchive struct {
	archiveMetadata
	DBPath string
}

// parseEtcdSnapshotArchive extracts the raw etcd db file (to a temp file under
// tempDir - caller must remove it), the optional release/request passthrough
// entries, and the optional skip-keys set from an EtcdSnapshotKind archive.
func parseEtcdSnapshotArchive(srcPath, tempDir string) (result *parsedEtcdSnapshotArchive, err error) {
	result = &parsedEtcdSnapshotArchive{}
	reader, err := os.Open(srcPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open source snapshot: %w", err)
	}
	defer reader.Close()

	gzipReader, err := gzip.NewReader(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
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
			return nil, fmt.Errorf("failed to read tar header: %w", nextErr)
		}

		switch {
		case header.Name == snapshotapi.SnapshotReleaseKey:
			result.ReleaseBytes, err = io.ReadAll(tarReader)
			if err != nil {
				return nil, fmt.Errorf("failed to read release: %w", err)
			}
		case strings.HasPrefix(header.Name, RequestStoreKey):
			result.RequestKey = header.Name
			result.RequestBytes, err = io.ReadAll(tarReader)
			if err != nil {
				return nil, fmt.Errorf("failed to read request: %w", err)
			}
		case header.Name == DBStoreKey:
			// there should only be one dbFile; if not => probably malformed archive
			if dbFile != "" {
				return nil, fmt.Errorf("archive contains multiple %q entries", DBStoreKey)
			}
			dbFile, err = writeTempFile(tempDir, tarReader)
			if err != nil {
				return nil, fmt.Errorf("failed to write etcd snapshot to temp file: %w", err)
			}
		case header.Name == SkipKeysStoreKey:
			skipKeysBytes, readErr := io.ReadAll(tarReader)
			if readErr != nil {
				return nil, fmt.Errorf("failed to read skipKeys: %w", readErr)
			}
			result.SkipKeys = make(map[string]struct{})
			if err := json.Unmarshal(skipKeysBytes, &result.SkipKeys); err != nil {
				return nil, fmt.Errorf("failed to unmarshal skipKeys: %w", err)
			}
		}
	}

	if dbFile == "" {
		return nil, fmt.Errorf("failed to find etcd snapshot in source archive")
	}

	result.DBPath = dbFile
	return result, nil
}

// mvccKeyValueSource exposes a store built over a restored etcd snapshot as a
// keyValueSource, so a converted snapshot goes through the same archive writer
// as a live one. revision is the snapshot's own recorded revision.
//
// The listing runs in its own goroutine that reads store, so the owner of that
// store MUST cancel the listing context and then call wait before closing it -
// mvcc reads racing a close panic rather than erroring.
type mvccKeyValueSource struct {
	store    mvcc.KV
	revision int64

	// done is closed once the list goroutine has exited; nil until ListStream
	// is called, which never happens if listing is skipped or fails earlier.
	done chan struct{}
}

func (s *mvccKeyValueSource) CurrentRevision(context.Context) (int64, error) {
	return s.revision, nil
}

// wait blocks until the list goroutine has exited. Cancel the listing context
// first, or a producer parked on a send never returns.
func (s *mvccKeyValueSource) wait() {
	if s.done != nil {
		<-s.done
	}
}

// ListStream pages through every live key/value under prefix (tombstoned and
// superseded revisions already resolved away by the MVCC engine), mirroring
// etcd.Client's ListStream: one pinned revision across all pages, results
// delivered on a buffered channel, errors delivered in-band.
func (s *mvccKeyValueSource) ListStream(ctx context.Context, prefix string) <-chan *etcd.ValueOrError {
	retChan := make(chan *etcd.ValueOrError, etcd.EtcdPaginationLimit)
	s.done = make(chan struct{})

	go func() {
		// closed after retChan, so a caller returning from wait knows the
		// listing is fully finished and the store is free to close
		defer close(s.done)
		defer close(retChan)

		// a consumer that gives up early stops draining, so never block on a
		// send: the store this goroutine reads is closed once the conversion
		// returns
		send := func(v *etcd.ValueOrError) bool {
			select {
			case retChan <- v:
				return true
			case <-ctx.Done():
				return false
			}
		}

		startKey, rangeEnd := mvccRange(prefix)
		var pinnedRev int64

		for {
			txn := s.store.Read(mvcc.ConcurrentReadTxMode, traceutil.TODO())
			result, err := txn.Range(ctx, startKey, rangeEnd, mvcc.RangeOptions{
				Limit: etcd.EtcdPaginationLimit,
				Rev:   pinnedRev,
			})
			txn.End()
			if err != nil {
				send(&etcd.ValueOrError{Error: fmt.Errorf("failed to range over restored keys: %w", err)})
				return
			}
			if pinnedRev == 0 {
				pinnedRev = result.Rev
			}

			for _, kv := range result.KVs {
				if !send(&etcd.ValueOrError{Value: etcd.Value{
					Key:      kv.Key,
					Data:     kv.Value,
					Modified: kv.ModRevision,
				}}) {
					return
				}
			}

			if len(result.KVs) < etcd.EtcdPaginationLimit {
				return
			}
			// advance past the last key to avoid duplicates
			startKey = append(append([]byte{}, result.KVs[len(result.KVs)-1].Key...), 0x00)
		}
	}()

	return retChan
}

// mvccRange translates a key prefix into an mvcc range. An empty prefix means
// every key: clientv3's all-keys sentinel is a "\x00" range end, which mvcc
// instead spells as an empty (open) end.
func mvccRange(prefix string) (startKey, rangeEnd []byte) {
	if prefix == "" {
		return []byte{0}, []byte{}
	}
	return []byte(prefix), []byte(clientv3.GetPrefixRangeEnd(prefix))
}
