package snapshot

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	vclusterconfig "github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/etcd"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/snapshot/types"
	"k8s.io/klog/v2"
)

const (
	RequestStoreKey           = "/vcluster/snapshot/request"
	DBStoreKey                = "/vcluster/snapshot/db"
	SkipKeysStoreKey          = "/vcluster/snapshot/skipkeys"
	EtcdSnapshotTypeMetaStamp = `{"apiVersion":"snapshot.vcluster.com/v1","kind":"EtcdSnapshot"}`
)

type Client struct {
	Request  *Request
	Options  Options
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

	// create store
	objectStore, err := CreateStore(ctx, &c.Options)
	if err != nil {
		return fmt.Errorf("failed to create store: %w", err)
	}

	// write the snapshot
	klog.Infof("Start writing etcd snapshot %s...", objectStore.Target())

	if vConfig.BackingStoreType() == vclusterconfig.StoreTypeEmbeddedEtcd {
		err = c.writeSnapshot(ctx, etcdClient, objectStore)
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

func (c *Client) List(ctx context.Context) ([]types.Snapshot, error) {
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

// writeSnapshot pulls a point-in-time snapshot from etcd, wraps it into a tar.gz archive, and stores it in the object store.
func (c *Client) writeSnapshot(ctx context.Context, etcdClient etcd.Client, objectStore types.Storage) error {
	log := klog.FromContext(ctx)

	log.Info("Getting point-in-time etcd snapshot")
	res, err := etcdClient.SnapshotWithVersion(ctx)
	if err != nil {
		return fmt.Errorf("snapshot request failed: %w", err)
	}

	dbPath, err := writeTempFile(res.Snapshot)
	if err != nil {
		return fmt.Errorf("failed to write snapshot to temp file: %w", err)
	}
	defer os.Remove(dbPath)

	log.Info("Creating snapshot archive")
	snapshotFileWrite, err := os.CreateTemp("", "snapshot-")
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

	// stamp snapshot archive with type meta
	gzipWriter.Comment = EtcdSnapshotTypeMetaStamp

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	if c.Options.Release != nil {
		log.Info("Adding vCluster config to snapshot archive")
		releaseBytes, err := json.Marshal(c.Options.Release)
		if err != nil {
			return fmt.Errorf("failed to marshal vCluster release: %w", err)
		}

		err = writeKeyValue(tarWriter, []byte(SnapshotReleaseKey), releaseBytes)
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
		key := fmt.Sprintf("%s/%s", RequestStoreKey, APIVersion)
		err = writeKeyValue(tarWriter, []byte(key), requestBytes)
		if err != nil {
			return fmt.Errorf("failed to snapshot snapshot request: %w", err)
		}
	}

	log.Info("Adding etcd snapshot to snapshot archive")
	if err := writeKeyValueFile(tarWriter, DBStoreKey, dbPath); err != nil {
		return fmt.Errorf("failed to write etcd snapshot to tar archive: %w", err)
	}

	if c.skipKeys != nil {
		log.Info("Adding skipKeys to snapshot archive")
		skipKeysBytes, err := json.Marshal(c.skipKeys)
		if err != nil {
			return fmt.Errorf("failed to marshal skipKeys: %w", err)
		}
		err = writeKeyValue(tarWriter, []byte(SkipKeysStoreKey), skipKeysBytes)
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

func (c *Client) writeKeyValueSnapshot(ctx context.Context, etcdClient etcd.Client, objectStore types.Storage) error {
	// now stream objects from etcd to object store
	errChan := make(chan error)
	reader, writer, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("failed to create pipe: %w", err)
	}
	defer writer.Close()
	go func() {
		defer reader.Close()
		errChan <- objectStore.PutObject(ctx, reader)
	}()

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

		err = writeKeyValue(tarWriter, []byte(SnapshotReleaseKey), releaseBytes)
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
		key := fmt.Sprintf("%s/%s", RequestStoreKey, APIVersion)
		err = writeKeyValue(tarWriter, []byte(key), requestBytes)
		if err != nil {
			return fmt.Errorf("failed to snapshot snapshot request: %w", err)
		}
	}

	// now write the snapshot
	backedUpKeys := 0
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context: %w", ctx.Err())
		case err := <-errChan:
			if err != nil {
				return fmt.Errorf("failed to write snapshot: %w", err)
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
				err := writeKeyValue(tarWriter, obj.Value.Key, obj.Value.Data)
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

				// close the writer to signal we are done, but wait until object store has finished writing
				_ = tarWriter.Close()
				_ = gzipWriter.Close()
				_ = writer.Close()
				return <-errChan
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

func writeKeyValue(tarWriter *tar.Writer, key, value []byte) error {
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

func writeKeyValueFile(tarWriter *tar.Writer, key, path string) error {
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
