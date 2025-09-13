package snapshot

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	vclusterconfig "github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/etcd"
	"github.com/loft-sh/vcluster/pkg/k8s"
	"github.com/loft-sh/vcluster/pkg/mappings/store"
	"github.com/loft-sh/vcluster/pkg/scheme"
	"go.etcd.io/etcd/server/v3/storage/backend"
	"go.etcd.io/etcd/server/v3/storage/mvcc"
	"go.etcd.io/etcd/server/v3/storage/schema"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/runtime/serializer/protobuf"
	"k8s.io/klog/v2"
)

type RestoreClient struct {
	Snapshot Options

	NewVCluster bool
}

var (
	podGVK = corev1.SchemeGroupVersion.WithKind("Pod")

	// bump revision to make sure we invalidate caches. See https://github.com/kubernetes/kubernetes/issues/118501 for more details
	BumpRevision = int64(1000)
)

func (o *RestoreClient) Run(ctx context.Context) (retErr error) {
	// create decoder and encoder
	decoder := serializer.NewCodecFactory(scheme.Scheme).UniversalDeserializer()
	encoder := protobuf.NewSerializer(scheme.Scheme, scheme.Scheme)

	// parse vCluster config
	vConfig, err := config.ParseConfig(constants.DefaultVClusterConfigLocation, os.Getenv("VCLUSTER_NAME"), nil)
	if err != nil {
		return err
	}

	// make sure to validate options
	err = ValidateConfigAndOptions(vConfig, &o.Snapshot, true, false)
	if err != nil {
		return err
	}

	// create store
	objectStore, err := CreateStore(ctx, &o.Snapshot)
	if err != nil {
		return fmt.Errorf("failed to create store: %w", err)
	}

	// now stream objects from object store to etcd
	reader, err := objectStore.GetObject(ctx)
	if err != nil {
		return fmt.Errorf("failed to get backup: %w", err)
	}
	defer reader.Close()

	// print log message that we start restoring
	klog.Infof("Start restoring etcd snapshot from %s...", objectStore.Target())

	// optionally decompress
	gzipReader, err := gzip.NewReader(reader)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzipReader.Close()

	// create new etcd client that will delete the existing data / recreate the database
	etcdClient, revertBackup, err := newRestoreEtcdClient(ctx, vConfig)
	if err != nil {
		revertBackup()
		return fmt.Errorf("failed to create etcd client: %w", err)
	}
	defer etcdClient.Close()

	// revert backup if there is an error
	defer func() {
		if retErr != nil {
			klog.Errorf("Reverting from backup due to error: %v", retErr)
			revertBackup()
		}
	}()

	// create a new tar reader
	tarReader := tar.NewReader(gzipReader)

	// now restore each key value
	restoredKeys := 0
	latestRevision := int64(0)
	for {
		// read from archive
		key, value, err := readKeyValue(tarReader)
		if err != nil && !errors.Is(err, io.EOF) {
			return fmt.Errorf("read etcd key/value: %w", err)
		} else if errors.Is(err, io.EOF) || len(key) == 0 {
			break
		}

		// transform value if we are restoring to a new vCluster
		if o.NewVCluster {
			// skip mappings
			splitKey := strings.Split(string(key), "/")
			if strings.HasPrefix(string(key), store.MappingsPrefix) {
				continue
			} else if len(splitKey) == 5 && splitKey[2] == "configmaps" && splitKey[4] == "kube-root-ca.crt" {
				// we will get separate certificates, so we need to skip these
				continue
			}
		}

		// transform pods to make sure they are not deleted on start
		if strings.HasPrefix(string(key), "/registry/pods/") {
			// we need to only do this in shared nodes mode as otherwise kubelet will not update the status correctly
			if !vConfig.PrivateNodes.Enabled {
				value, err = transformPod(value, decoder, encoder)
				if err != nil {
					return fmt.Errorf("transform value: %w", err)
				}
			}
		}

		// write the value to etcd
		klog.V(1).Infof("Restore key %s", string(key))
		latestRevision, err = etcdClient.Put(ctx, string(key), value)
		if err != nil {
			return fmt.Errorf("restore etcd key %s: %w", string(key), err)
		}

		// print status update
		restoredKeys++
		if restoredKeys%100 == 0 {
			klog.Infof("Restored %d keys", restoredKeys)
		}
	}

	// compact the database until that revision
	klog.Infof("Compact etcd database until revision %d", latestRevision)
	err = etcdClient.Compact(ctx, latestRevision)
	if err != nil {
		return fmt.Errorf("compact etcd database: %w", err)
	}

	klog.Infof("Successfully restored %d etcd keys from snapshot", restoredKeys)
	klog.Infof("Successfully restored snapshot from %s", objectStore.Target())
	return nil
}

func transformPod(value []byte, decoder runtime.Decoder, encoder runtime.Encoder) ([]byte, error) {
	// decode value
	obj := &corev1.Pod{}
	_, _, err := decoder.Decode(value, &podGVK, obj)
	if err != nil {
		return nil, fmt.Errorf("decode value: %w", err)
	} else if obj.DeletionTimestamp != nil {
		return value, nil
	}

	// make sure to delete nodename & status or otherwise vCluster will delete the pod on start
	obj.Spec.NodeName = ""
	obj.Status = corev1.PodStatus{}

	// encode value
	buf := &bytes.Buffer{}
	err = encoder.Encode(obj, buf)
	if err != nil {
		return nil, fmt.Errorf("encode value: %w", err)
	}

	return buf.Bytes(), nil
}

func newRestoreEtcdClient(ctx context.Context, vConfig *config.VirtualClusterConfig) (etcd.Client, func(), error) {
	revertBackup := func() {}

	// delete existing storage:
	// * embedded etcd: delete the files locally and make sure revision is not decreasing, this is important as otherwise watches will not work correctly
	// * deploy etcd: range delete request
	// * embedded database: delete the files locally and make sure revision is not decreasing, this is important as otherwise watches will not work correctly
	// * external database: we can't so we skip and then check later if there are any already
	if vConfig.BackingStoreType() == vclusterconfig.StoreTypeEmbeddedDatabase {
		if vConfig.Distro() == vclusterconfig.K8SDistro {
			// get latest revision from database
			latestRevision, err := getLatestRevisionSQLite(ctx, constants.K8sSqliteDatabase)
			if err != nil {
				return nil, revertBackup, fmt.Errorf("failed to get latest revision from database: %w", err)
			}

			// this is a little bit stupid since we cannot rename /data, so we have to snapshot the
			// individual files.
			err = backupFile(ctx, constants.K8sSqliteDatabase)
			if err != nil {
				return nil, revertBackup, fmt.Errorf("failed to backup database: %w", err)
			}
			err = backupFile(ctx, constants.K8sSqliteDatabase+"-wal")
			if err != nil {
				return nil, revertBackup, fmt.Errorf("failed to backup database: %w", err)
			}
			err = backupFile(ctx, constants.K8sSqliteDatabase+"-shm")
			if err != nil {
				return nil, revertBackup, fmt.Errorf("failed to backup database: %w", err)
			}

			// create a restore function that will restore the database in case of an error
			revertBackup = func() {
				_ = os.RemoveAll(constants.K8sSqliteDatabase)
				_ = os.RemoveAll(constants.K8sSqliteDatabase + "-wal")
				_ = os.RemoveAll(constants.K8sSqliteDatabase + "-shm")
				_ = os.Rename(constants.K8sSqliteDatabase+".backup", constants.K8sSqliteDatabase)
				_ = os.Rename(constants.K8sSqliteDatabase+"-wal.backup", constants.K8sSqliteDatabase+"-wal")
				_ = os.Rename(constants.K8sSqliteDatabase+"-shm.backup", constants.K8sSqliteDatabase+"-shm")
			}

			// set latest revision
			if latestRevision > 0 {
				err = setLatestRevisionSQLite(ctx, constants.K8sSqliteDatabase, latestRevision+BumpRevision)
				if err != nil {
					return nil, revertBackup, fmt.Errorf("failed to set latest revision: %w", err)
				}
			}
		} else if vConfig.Distro() == vclusterconfig.K3SDistro {
			// get latest revision from database
			latestRevision, err := getLatestRevisionSQLite(ctx, constants.K3sSqliteDatabase)
			if err != nil {
				return nil, revertBackup, fmt.Errorf("failed to get latest revision from database: %w", err)
			}

			// backup database
			err = backupFolder(ctx, filepath.Dir(constants.K3sSqliteDatabase))
			if err != nil {
				return nil, revertBackup, err
			}

			// create a restore function that will restore the database in case of an error
			revertBackup = func() {
				_ = os.RemoveAll(constants.K3sSqliteDatabase)
				_ = os.Rename(constants.K3sSqliteDatabase+".backup", constants.K3sSqliteDatabase)
			}

			// set latest revision
			if latestRevision > 0 {
				err = setLatestRevisionSQLite(ctx, constants.K3sSqliteDatabase, latestRevision+BumpRevision)
				if err != nil {
					return nil, revertBackup, fmt.Errorf("failed to set latest revision: %w", err)
				}
			}
		}
	} else if vConfig.BackingStoreType() == vclusterconfig.StoreTypeEmbeddedEtcd {
		// get latest revision from etcd
		etcdDBPath := filepath.Join(constants.EmbeddedEtcdData, "member", "snap", "db")
		latestRevision, err := getLatestRevisionEtcd(ctx, etcdDBPath)
		if err != nil {
			return nil, revertBackup, fmt.Errorf("failed to get latest revision from etcd: %w", err)
		}

		// backup etcd data
		err = backupFolder(ctx, constants.EmbeddedEtcdData)
		if err != nil {
			return nil, revertBackup, err
		}

		// create a restore function that will restore the database in case of an error
		revertBackup = func() {
			_ = os.RemoveAll(constants.EmbeddedEtcdData)
			_ = os.Rename(constants.EmbeddedEtcdData+".backup", constants.EmbeddedEtcdData)
		}

		// set latest revision
		if latestRevision > 0 {
			err = setLatestRevisionEtcd(ctx, vConfig, etcdDBPath, latestRevision+BumpRevision)
			if err != nil {
				return nil, revertBackup, fmt.Errorf("failed to set latest revision: %w", err)
			}
		}
	}

	// now create the etcd client
	etcdClient, err := newEtcdClient(ctx, vConfig, true)
	if err != nil {
		return nil, revertBackup, err
	}

	// delete contents in external etcd
	if vConfig.BackingStoreType() == vclusterconfig.StoreTypeDeployedEtcd || vConfig.BackingStoreType() == vclusterconfig.StoreTypeExternalEtcd {
		klog.FromContext(ctx).Info("Delete existing etcd data before restore...")
		err = etcdClient.DeletePrefix(ctx, "/")
		if err != nil {
			return nil, revertBackup, err
		}
	}

	return etcdClient, revertBackup, nil
}

func setLatestRevisionSQLite(ctx context.Context, file string, revision int64) error {
	klog.FromContext(ctx).Info("Setting latest revision for SQLite database...", "revision", revision)

	// create a new context that can be cancelled
	kineCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// start & stop kine to create the database
	doneChan := k8s.StartKineWithDone(kineCtx, fmt.Sprintf("sqlite://%s%s", file, k8s.SQLiteParams), constants.K8sKineEndpoint, nil, nil)

	// wait until file is created
	for {
		time.Sleep(1 * time.Second)
		_, err := os.Stat(file)
		if err == nil {
			break
		}
	}

	// stop kine
	cancel()

	// wait for kine to finish
	<-doneChan

	// set latest revision
	db, err := sql.Open("sqlite", file+k8s.SQLiteParams)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// set latest revision
	_, err = db.ExecContext(ctx, "UPDATE SQLITE_SEQUENCE SET seq = ? WHERE name = 'kine'", revision)
	if err != nil {
		// try insert if it doesn't exist
		_, err = db.ExecContext(ctx, "INSERT INTO SQLITE_SEQUENCE (name, seq) VALUES ('kine', ?)", revision)
		if err != nil {
			return fmt.Errorf("failed to set latest revision: %w", err)
		}
	}

	klog.FromContext(ctx).Info("Successfully set latest revision for SQLite database", "revision", revision)
	return nil
}

func getLatestRevisionSQLite(ctx context.Context, file string) (int64, error) {
	// check if file exists
	_, err := os.Stat(file)
	if os.IsNotExist(err) {
		return 0, nil
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// open sqlite database
	db, err := sql.Open("sqlite", file+k8s.SQLiteParams)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	// get latest revision
	row := db.QueryRowContext(ctx, "SELECT seq FROM SQLITE_SEQUENCE WHERE name = 'kine'")
	var revision int64
	err = row.Scan(&revision)
	if err != nil {
		return 0, err
	}

	klog.FromContext(ctx).Info("Successfully got latest revision for SQLite database", "revision", revision)
	return revision, nil
}

func setLatestRevisionEtcd(ctx context.Context, vConfig *config.VirtualClusterConfig, file string, revision int64) error {
	klog.FromContext(ctx).Info("Setting latest revision for etcd database...", "revision", revision)

	// start embedded etcd
	stop, err := startEmbeddedEtcd(ctx, vConfig)
	if err != nil {
		return fmt.Errorf("failed to start embedded etcd: %w", err)
	}

	// wait until etcd is ready
	etcdClient, err := newEtcdClient(ctx, vConfig, false)
	if err != nil {
		return fmt.Errorf("failed to create etcd client: %w", err)
	}
	etcdClient.Close()

	// stop embedded etcd
	stop()

	// set latest revision
	err = unsafeSetLatestRevisionEtcd(file, revision)
	if err != nil {
		return fmt.Errorf("failed to set latest revision: %w", err)
	}

	klog.FromContext(ctx).Info("Successfully set latest revision for etcd database", "revision", revision)
	return nil
}

func unsafeSetLatestRevisionEtcd(file string, revision int64) error {
	// code is mostly from https://github.com/etcd-io/etcd/blob/c515c6acc15574a611d0f001a03030cb0ba945e6/etcdutl/snapshot/v3_snapshot.go#L373
	be := backend.NewDefaultBackend(zap.L().Named("etcd-client"), file)
	defer func() {
		be.ForceCommit()
		be.Close()
	}()

	tx := be.BatchTx()
	tx.LockOutsideApply()
	defer tx.Unlock()

	k := mvcc.NewRevBytes()
	k = mvcc.RevToBytes(mvcc.Revision{
		Main: revision,
		Sub:  0,
	}, k)
	tx.UnsafePut(schema.Key, k, []byte{})
	return nil
}

func getLatestRevisionEtcd(ctx context.Context, file string) (int64, error) {
	_, err := os.Stat(file)
	if os.IsNotExist(err) {
		return 0, nil
	}

	// code is mostly from https://github.com/etcd-io/etcd/blob/c515c6acc15574a611d0f001a03030cb0ba945e6/etcdutl/snapshot/v3_snapshot.go#L421
	be := backend.NewDefaultBackend(zap.L().Named("etcd-client"), file)
	defer be.Close()

	tx := be.ReadTx()
	tx.RLock()
	defer tx.RUnlock()

	var latest mvcc.Revision
	err = tx.UnsafeForEach(schema.Key, func(k, _ []byte) (err error) {
		rev := mvcc.BytesToRev(k)
		if rev.GreaterThan(latest) {
			latest = rev
		}

		return nil
	})

	klog.FromContext(ctx).Info("Successfully got latest revision for etcd database", "revision", latest.Main)
	return latest.Main, err
}

func backupFile(ctx context.Context, file string) error {
	_, err := os.Stat(file)
	if os.IsNotExist(err) {
		return nil
	}

	backupName := file + ".backup"
	_, err = os.Stat(backupName)
	if err == nil {
		_ = os.RemoveAll(backupName)
	}

	klog.FromContext(ctx).Info(fmt.Sprintf("Renaming existing database from %s to %s, if something goes wrong please restore the old database", file, backupName))
	return os.Rename(file, backupName)
}

func backupFolder(ctx context.Context, dir string) error {
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		return nil
	}

	backupName := dir + ".backup"
	_, err = os.Stat(backupName)
	if err == nil {
		_ = os.RemoveAll(backupName)
	}

	klog.FromContext(ctx).Info(fmt.Sprintf("Renaming existing database from %s to %s, if something goes wrong please restore the old database", dir, backupName))
	err = os.Rename(dir, backupName)
	if err != nil {
		return err
	}

	return os.MkdirAll(dir, 0777)
}

func readKeyValue(tarReader *tar.Reader) ([]byte, []byte, error) {
	header, err := tarReader.Next()
	if err != nil {
		return nil, nil, err
	}

	buf := &bytes.Buffer{}
	_, err = io.Copy(buf, tarReader)
	if err != nil {
		return nil, nil, err
	}

	return []byte(header.Name), buf.Bytes(), nil
}
