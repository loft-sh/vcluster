package snapshot

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"

	snapshotapi "github.com/loft-sh/api/v4/pkg/snapshot"
	"github.com/loft-sh/api/v4/pkg/snapshot/storage/types"
	vclusterconfig "github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/etcd"
	"github.com/loft-sh/vcluster/pkg/pro"
	"k8s.io/klog/v2"
)

type SnapshotKind string

const (
	UnknownSnapshotKind  SnapshotKind = "Unknown"
	EtcdSnapshotKind     SnapshotKind = "EtcdSnapshot"
	KeyValueSnapshotKind SnapshotKind = "KeyValueSnapshot"

	RequestStoreKey  = "/vcluster/snapshot/request"
	DBStoreKey       = "/vcluster/snapshot/db"
	SkipKeysStoreKey = "/vcluster/snapshot/skipkeys"
	// RevisionStoreKey holds the backing store's revision at the time the
	// snapshot was taken (decimal-encoded int64).
	RevisionStoreKey = "/vcluster/snapshot/revision"
)

type Client struct {
	Request  *snapshotapi.Request
	Options  snapshotapi.Options
	skipKeys map[string]struct{}
}

func (c *Client) Run(ctx context.Context, vConfig *config.VirtualClusterConfig) error {
	if vConfig == nil {
		return fmt.Errorf("snapshot client requires vCluster config")
	}

	var err error
	if vConfig.ControlPlane.Standalone.Enabled {
		vConfig.HostNamespace = constants.VClusterStandaloneSnapshotNamespace
		err = pro.SetStandaloneConstants(vConfig)
		if err != nil {
			return fmt.Errorf("set standalone constants: %w", err)
		}
	}

	// make sure to validate options
	err = Validate(&c.Options, false)
	if err != nil {
		return err
	}

	// create new etcd client
	etcdClient, err := newEtcdClient(ctx, vConfig, false)
	if err != nil {
		return fmt.Errorf("failed to create etcd client: %w", err)
	}
	defer etcdClient.Close()

	// create store
	objectStore, err := CreateStore(ctx, &c.Options)
	if err != nil {
		return fmt.Errorf("failed to create store: %w", err)
	}

	// write the snapshot
	klog.Infof("Start writing etcd snapshot %s...", objectStore.Target())

	if vConfig.BackingStoreType() == vclusterconfig.StoreTypeEmbeddedEtcd {
		err = c.writeEtcdSnapshot(ctx, etcdClient, objectStore)
		if err != nil {
			return err
		}
	} else {
		err = c.writeKeyValueSnapshot(ctx, etcdClient, objectStore)
		if err != nil {
			return err
		}
	}

	klog.Infof("Successfully wrote snapshot to %s", objectStore.Target())
	return nil
}

func (c *Client) List(ctx context.Context) ([]snapshotapi.Snapshot, error) {
	var err error
	// make sure to validate options
	err = Validate(&c.Options, true)
	if err != nil {
		return nil, err
	}

	// create store
	objectStore, err := CreateStore(ctx, &c.Options)
	if err != nil {
		return nil, fmt.Errorf("failed to create store: %w", err)
	}

	// list snapshots
	return objectStore.List(ctx)
}

func (c *Client) Delete(ctx context.Context) error {
	var err error
	// make sure to validate options
	err = Validate(&c.Options, false)
	if err != nil {
		return err
	}

	// create store
	objectStore, err := CreateStore(ctx, &c.Options)
	if err != nil {
		return fmt.Errorf("failed to create store: %w", err)
	}

	// delete snapshot
	if err := objectStore.Delete(ctx); err != nil {
		return err
	}

	klog.Infof("Successfully deleted snapshot %s", objectStore.Target())
	return nil
}

// writeEtcdSnapshot pulls a point-in-time snapshot from etcd, wraps it into a tar.gz archive, and stores it in the object store.
func (c *Client) writeEtcdSnapshot(ctx context.Context, etcdClient etcd.Client, objectStore types.Storage) error {
	log := klog.FromContext(ctx)

	log.Info("Getting point-in-time etcd snapshot")
	res, err := etcdClient.SnapshotWithVersion(ctx)
	if err != nil {
		return fmt.Errorf("snapshot request failed: %w", err)
	}
	defer res.Snapshot.Close()

	dbPath, err := writeTempFile(c.Options.SnapshotTempDir, res.Snapshot)
	if err != nil {
		return fmt.Errorf("failed to write snapshot to temp file: %w", err)
	}
	defer os.Remove(dbPath)

	log.Info("Creating snapshot archive")
	snapshotFileWrite, err := os.CreateTemp(c.Options.SnapshotTempDir, "snapshot-")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer snapshotFileWrite.Close()
	defer os.Remove(snapshotFileWrite.Name())

	// don't compress with default level as this can get quite cpu heavy
	gzipWriter, err := gzip.NewWriterLevel(snapshotFileWrite, 3)
	if err != nil {
		return fmt.Errorf("failed to create gzip writer: %w", err)
	}
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	if c.Options.Release != nil {
		// The vcluster create with restore command expects the SnapshotReleaseKey key to be the first key in the tar archive
		log.Info("Adding vCluster config to snapshot archive")
		releaseBytes, err := json.Marshal(c.Options.Release)
		if err != nil {
			return fmt.Errorf("failed to marshal vCluster release: %w", err)
		}

		err = writeArchiveEntry(tarWriter, []byte(snapshotapi.SnapshotReleaseKey), releaseBytes)
		if err != nil {
			return fmt.Errorf("failed to snapshot vCluster release: %w", err)
		}
	}

	if c.Request != nil {
		log.Info("Adding snapshot request to snapshot archive")
		requestBytes, err := json.Marshal(c.Request)
		if err != nil {
			return fmt.Errorf("failed to marshal snapshot request: %w", err)
		}
		key := fmt.Sprintf("%s/%s", RequestStoreKey, snapshotapi.APIVersion)
		err = writeArchiveEntry(tarWriter, []byte(key), requestBytes)
		if err != nil {
			return fmt.Errorf("failed to snapshot request: %w", err)
		}
	}

	// DBStoreKey as the third entry (or first, if no release or snapshot request) marks this archive
	// as an EtcdSnapshot; otherwise it is treated as a KeyValueSnapshot.
	// Keep getSnapshotArchiveKind in sync with any structure changes.
	log.Info("Adding etcd snapshot to snapshot archive")
	if err := writeArchiveFileEntry(tarWriter, DBStoreKey, dbPath); err != nil {
		return fmt.Errorf("failed to write etcd snapshot to tar archive: %w", err)
	}

	if c.skipKeys != nil {
		log.Info("Adding skipKeys to snapshot archive")
		skipKeysBytes, err := json.Marshal(c.skipKeys)
		if err != nil {
			return fmt.Errorf("failed to marshal skipKeys: %w", err)
		}
		err = writeArchiveEntry(tarWriter, []byte(SkipKeysStoreKey), skipKeysBytes)
		if err != nil {
			return fmt.Errorf("failed to snapshot skipKeys: %w", err)
		}
	}

	log.Info("Closing snapshot archive")
	if err := tarWriter.Close(); err != nil {
		return fmt.Errorf("failed to close tar writer: %w", err)
	}

	if err := gzipWriter.Close(); err != nil {
		return fmt.Errorf("failed to close gzip writer: %w", err)
	}

	if err := snapshotFileWrite.Close(); err != nil {
		return fmt.Errorf("failed to close snapshot archive writer: %w", err)
	}

	log.Info("Storing snapshot archive into the object store")
	snapshotFileRead, err := os.Open(snapshotFileWrite.Name())
	if err != nil {
		return fmt.Errorf("failed to open snapshot file for reading: %w", err)
	}
	defer snapshotFileRead.Close()

	if err := objectStore.PutObject(ctx, snapshotFileRead); err != nil {
		return fmt.Errorf("failed to write snapshot to object store: %w", err)
	}

	return nil
}

// writeKeyValueSnapshot streams etcd key/value pairs into objectStore as a
// gzipped tar archive. err is a named return so the deferred cleanup below
// can abort the in-flight upload with the real failure (instead of a clean
// EOF that would look like a successful, truncated upload) on every early
// return, and always drain the upload goroutine's result.
func (c *Client) writeKeyValueSnapshot(ctx context.Context, etcdClient etcd.Client, objectStore types.Storage) (err error) {
	// now stream objects from etcd to object store
	errChan := make(chan error, 1)
	reader, writer := io.Pipe()
	go func() {
		defer reader.Close()
		errChan <- objectStore.PutObject(ctx, reader)
	}()
	// uploadResultReceived tracks whether the select loop below already
	// consumed errChan's single buffered value, so the cleanup defer never
	// tries to receive from it a second time (which would block forever).
	uploadResultReceived := false
	defer func() {
		_ = writer.CloseWithError(err)
		if !uploadResultReceived {
			if uploadErr := <-errChan; uploadErr != nil && err == nil {
				err = fmt.Errorf("failed to write snapshot: %w", uploadErr)
			}
		}
	}()

	// pin the revision before listing so it reflects the state being snapshotted
	revision, err := etcdClient.CurrentRevision(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current revision: %w", err)
	}

	// start listing the keys
	listChan := etcdClient.ListStream(ctx, "/")

	// don't compress with default level as this can get quite cpu heavy
	gzipWriter, _ := gzip.NewWriterLevel(writer, 3)
	defer gzipWriter.Close()

	// create a new tar write
	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	// write the vCluster config as first thing
	if c.Options.Release != nil {
		releaseBytes, err := json.Marshal(c.Options.Release)
		if err != nil {
			return fmt.Errorf("failed to marshal vCluster release: %w", err)
		}

		err = writeArchiveEntry(tarWriter, []byte(snapshotapi.SnapshotReleaseKey), releaseBytes)
		if err != nil {
			return fmt.Errorf("failed to snapshot vCluster release: %w", err)
		}
	}

	// write the snapshot request
	if c.Request != nil {
		requestBytes, err := json.Marshal(c.Request)
		if err != nil {
			return fmt.Errorf("failed to marshal snapshot request: %w", err)
		}
		key := fmt.Sprintf("%s/%s", RequestStoreKey, snapshotapi.APIVersion)
		err = writeArchiveEntry(tarWriter, []byte(key), requestBytes)
		if err != nil {
			return fmt.Errorf("failed to snapshot request: %w", err)
		}
	}

	// write the pinned revision
	err = writeArchiveEntry(tarWriter, []byte(RevisionStoreKey), []byte(strconv.FormatInt(revision, 10)))
	if err != nil {
		return fmt.Errorf("failed to snapshot revision: %w", err)
	}

	// now write the snapshot
	backedUpKeys := 0
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context: %w", ctx.Err())
		case uploadErr := <-errChan:
			uploadResultReceived = true
			if uploadErr != nil {
				return fmt.Errorf("failed to write snapshot: %w", uploadErr)
			}
			return nil
		case obj := <-listChan:
			// check if error or object to write
			if obj != nil {
				if obj.Error != nil {
					return fmt.Errorf("failed to retrieve etcd items: %w", obj.Error)
				}

				key := string(obj.Value.Key)
				if _, ok := c.skipKeys[key]; ok {
					klog.Infof("Skipping key %s", key)
					continue
				}
				// write the object into the store
				klog.V(1).Infof("Snapshot key %s", key)
				err := writeArchiveEntry(tarWriter, obj.Value.Key, obj.Value.Data)
				if err != nil {
					return fmt.Errorf("failed to snapshot key %s: %w", key, err)
				}

				// print status update
				backedUpKeys++
				if backedUpKeys%100 == 0 {
					klog.Infof("Backed up %d keys", backedUpKeys)
				}
			} else {
				klog.Infof("Successfully backed up %d etcd keys", backedUpKeys)

				// flush the archive; the deferred cleanup closes the pipe
				// writer and waits for the upload to finish
				_ = tarWriter.Close()
				_ = gzipWriter.Close()
				return nil
			}
		}
	}
}

func (c *Client) addResourceToSkip(kindPlural, namespacedName string) {
	if c.skipKeys == nil {
		c.skipKeys = make(map[string]struct{})
	}

	c.skipKeys[fmt.Sprintf("/registry/%s/%s", kindPlural, namespacedName)] = struct{}{}
}

func writeArchiveEntry(tarWriter *tar.Writer, key, value []byte) error {
	err := tarWriter.WriteHeader(&tar.Header{
		Typeflag: tar.TypeReg,
		Name:     string(key),
		Size:     int64(len(value)),
		Mode:     0666,
	})
	if err != nil {
		return err
	}

	// write value to tar archive
	_, err = tarWriter.Write(value)
	if err != nil {
		return err
	}

	return nil
}

func writeArchiveFileEntry(tarWriter *tar.Writer, key, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	if err := tarWriter.WriteHeader(&tar.Header{
		Typeflag: tar.TypeReg,
		Name:     key,
		Size:     stat.Size(),
		Mode:     0666,
	}); err != nil {
		return err
	}

	if _, err := io.Copy(tarWriter, f); err != nil {
		return fmt.Errorf("failed to write file to tar archive: %w", err)
	}

	return nil
}
