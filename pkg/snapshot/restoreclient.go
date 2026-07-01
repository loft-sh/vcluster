package snapshot

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	snapshotapi "github.com/loft-sh/api/v4/pkg/snapshot"
	vclusterconfig "github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/etcd"
	"github.com/loft-sh/vcluster/pkg/k8s"
	"github.com/loft-sh/vcluster/pkg/mappings/store"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/scheme"
	"github.com/loft-sh/vcluster/pkg/util/clienthelper"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/mirror"
	"go.etcd.io/etcd/etcdutl/v3/snapshot"
	etcdservererrors "go.etcd.io/etcd/server/v3/etcdserver/errors"
	"go.etcd.io/etcd/server/v3/storage/backend"
	"go.etcd.io/etcd/server/v3/storage/datadir"
	"go.etcd.io/etcd/server/v3/storage/mvcc"
	"go.etcd.io/etcd/server/v3/storage/schema"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/runtime/serializer/protobuf"
	"k8s.io/klog/v2"
)

const (
	podPrefix       = "/registry/pods/"
	configMapPrefix = "/registry/configmaps/"
)

type RestoreClient struct {
	Snapshot snapshotapi.Options

	etcdClient etcd.Client

	NewVCluster bool
}

var (
	podGVK = corev1.SchemeGroupVersion.WithKind("Pod")
	// bump revision to make sure we invalidate caches. See https://github.com/kubernetes/kubernetes/issues/118501 for more details
	BumpRevision = int64(1000)
)

func NewRestoreClient(snapshotOptions snapshotapi.Options, newVCluster bool) *RestoreClient {
	return &RestoreClient{
		Snapshot:    snapshotOptions,
		NewVCluster: newVCluster,
	}
}

func (o *RestoreClient) GetSnapshotRequest(ctx context.Context) (*snapshotapi.Request, error) {
	// make sure to validate options
	err := Validate(&o.Snapshot, false)
	if err != nil {
		return nil, err
	}

	// create store
	objectStore, err := CreateStore(ctx, &o.Snapshot)
	if err != nil {
		return nil, fmt.Errorf("failed to create store: %w", err)
	}

	// now stream objects from object store to etcd
	reader, err := objectStore.GetObject(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get backup: %w", err)
	}
	defer reader.Close()

	// optionally decompress
	gzipReader, err := gzip.NewReader(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzipReader.Close()

	// create a new tar reader
	tarReader := tar.NewReader(gzipReader)

	for {
		header, err := tarReader.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			return nil, fmt.Errorf("read snapshot archive: %w", err)
		}

		if !strings.HasPrefix(header.Name, RequestStoreKey) {
			continue
		}

		value, err := io.ReadAll(tarReader)
		if err != nil {
			return nil, fmt.Errorf("failed to read snapshot request: %w", err)
		}

		var snapshotRequest snapshotapi.Request
		if err := json.Unmarshal(value, &snapshotRequest); err != nil {
			return nil, fmt.Errorf("failed to unmarshal snapshot request: %w", err)
		}

		return &snapshotRequest, nil
	}

	return nil, ErrSnapshotRequestNotFound
}

func (o *RestoreClient) Run(ctx context.Context, vConfig *config.VirtualClusterConfig) (retErr error) {
	if vConfig == nil {
		return fmt.Errorf("restore run requires vCluster config")
	}

	var err error
	revertStandaloneRestore := func() error { return nil }

	if vConfig.ControlPlane.Standalone.Enabled {
		vConfig.HostNamespace = constants.VClusterStandaloneSnapshotNamespace
		err = pro.SetStandaloneConstants(vConfig)
		if err != nil {
			return fmt.Errorf("set standalone constants: %w", err)
		}

		revertStandaloneRestore, err = pro.SetupStandaloneRestore(vConfig)
		if err != nil {
			return fmt.Errorf("setup standalone restore: %w", err)
		}

		defer func() {
			if retErr != nil {
				revertErr := revertStandaloneRestore()
				if revertErr != nil {
					retErr = errors.Join(retErr, fmt.Errorf("revert standalone restore state: %w", revertErr))
				}
			}
		}()
	}

	// make sure to validate options
	err = Validate(&o.Snapshot, false)
	if err != nil {
		return err
	}

	// create store
	objectStore, err := CreateStore(ctx, &o.Snapshot)
	if err != nil {
		return fmt.Errorf("failed to create store: %w", err)
	}

	klog.Infof("Start restoring etcd snapshot from %s...", objectStore.Target())

	snapshotReader, err := objectStore.GetObject(ctx)
	if err != nil {
		return fmt.Errorf("failed to get snapshot: %w", err)
	}
	defer snapshotReader.Close()

	snapshotPath, err := writeTempFile(o.Snapshot.SnapshotTempDir, snapshotReader)
	if err != nil {
		return fmt.Errorf("failed to write snapshot to temp file: %w", err)
	}
	defer os.Remove(snapshotPath)

	archiveKind, err := getSnapshotArchiveKind(snapshotPath)
	if err != nil {
		return fmt.Errorf("failed to determine snapshot archive kind: %w", err)
	}

	if archiveKind == EtcdSnapshotKind {
		if vConfig.BackingStoreType() == vclusterconfig.StoreTypeEmbeddedEtcd {
			if err := o.restoreEtcdSnapshot(ctx, vConfig, snapshotPath); err != nil {
				return fmt.Errorf("failed to restore etcd snapshot: %w", err)
			}
		} else {
			return fmt.Errorf("restore etcd snapshot is not supported for store type %s", vConfig.BackingStoreType())
		}
	} else {
		if err := o.restoreKeyValueSnapshot(ctx, vConfig, snapshotPath); err != nil {
			return fmt.Errorf("restore key-value snapshot: %w", err)
		}
	}

	klog.Infof("Successfully restored snapshot from %s", objectStore.Target())
	return nil
}

// restoreEtcdSnapshot restores the vCluster embedded etcd backing store as follows:
//  1. read snapshot data from the snapshot archive
//  2. restore the etcd snapshot via etcdutl snapshot package into etcd dataDir
//  3. mutate etcd data via embedded-etcd instance:
//     - remove skipped keys
//     - reset pods nodeName and status
func (o *RestoreClient) restoreEtcdSnapshot(ctx context.Context, vConfig *config.VirtualClusterConfig, snapshotPath string) (retErr error) {
	log := klog.FromContext(ctx)

	// dbPath and skipKeysBytes are snapshot data components to be extracted from the snapshot archive
	var dbPath string
	defer func() {
		if dbPath != "" {
			_ = os.Remove(dbPath)
		}
	}()
	var skipKeysBytes []byte

	log.Info("Reading snapshot archive", "snapshotPath", snapshotPath)
	reader, err := os.Open(snapshotPath)
	if err != nil {
		return fmt.Errorf("failed to get backup: %w", err)
	}
	defer reader.Close()

	gzipReader, err := gzip.NewReader(reader)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)

	for {
		header, err := tarReader.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		if header.Name == DBStoreKey {
			dbPath, err = writeTempFile(o.Snapshot.SnapshotTempDir, tarReader)
			if err != nil {
				return fmt.Errorf("failed to write snapshot to temp file: %w", err)
			}
			continue
		}

		if header.Name == SkipKeysStoreKey {
			skipKeysBytes, err = io.ReadAll(tarReader)
			if err != nil {
				return fmt.Errorf("failed to read skipKeys from tar archive: %w", err)
			}
			continue
		}
	}

	if dbPath == "" {
		return fmt.Errorf("failed to find etcd snapshot in tar archive")
	}

	// read snapshot and existing etcd dataDir status
	snapshotStatus, err := snapshot.NewV3(zap.L().Named("restore-etcd")).Status(dbPath)
	if err != nil {
		return fmt.Errorf("failed to get snapshot status: %w", err)
	}

	var latestRevision int64
	if _, err := os.Stat(datadir.ToBackendFileName(constants.EmbeddedEtcdData)); err == nil {
		existingSnapshot, err := snapshot.NewV3(zap.L().Named("restore-etcd")).Status(datadir.ToBackendFileName(constants.EmbeddedEtcdData))
		if err != nil {
			return fmt.Errorf("failed to get existing snapshot status: %w", err)
		}
		latestRevision = existingSnapshot.Revision
	}

	// backup etcd dataDir before restoring
	if err := backupFolder(ctx, constants.EmbeddedEtcdData); err != nil {
		return fmt.Errorf("failed to backup etcd dataDir: %w", err)
	}
	defer func() {
		if retErr != nil {
			if err := restoreBackupFolder(ctx, constants.EmbeddedEtcdData); err != nil {
				klog.Errorf("Failed to revert etcd dataDir: %v", err)
			}
		}
	}()

	// restoring etcd snapshot into etcd dataDir
	name, peerURL, err := nameAndPeerURLForConfig(vConfig)
	if err != nil {
		return fmt.Errorf("failed to get name and peer URL: %w", err)
	}

	log.Info("Restoring etcd snapshot", "dbPath", dbPath, "dataDir", constants.EmbeddedEtcdData)
	if err := snapshot.NewV3(zap.L().Named("restore-etcd")).Restore(snapshot.RestoreConfig{
		SnapshotPath:        dbPath,
		Name:                name,
		OutputDataDir:       constants.EmbeddedEtcdData,
		OutputWALDir:        datadir.ToWALDir(constants.EmbeddedEtcdData),
		PeerURLs:            []string{peerURL},
		InitialCluster:      name + "=" + peerURL,
		InitialClusterToken: "vcluster",
		SkipHashCheck:       false,
		InitialMmapSize:     backend.InitialMmapSize,
		RevisionBump:        snapshotRestoreBumpRevision(latestRevision, snapshotStatus.Revision, BumpRevision),
		MarkCompacted:       true,
	}); err != nil {
		return fmt.Errorf("restore etcd: %w", err)
	}

	// post-restore etcd data mutation
	if err := o.postRestoreSnapshotDataMutation(ctx, vConfig, skipKeysBytes); err != nil {
		return fmt.Errorf("failed to mutate data: %w", err)
	}

	return nil
}

func (o *RestoreClient) postRestoreSnapshotDataMutation(ctx context.Context, vConfig *config.VirtualClusterConfig, skipKeysBytes []byte) error {
	log := klog.FromContext(ctx)

	log.Info("Starting embedded etcd to reset cluster membership")
	stop, err := startEmbeddedEtcd(ctx, vConfig)
	if err != nil {
		return fmt.Errorf("failed to start embedded etcd: %w", err)
	}
	defer stop()

	log.Info("Waiting for embedded etcd to be ready")
	etcdEndpoint, etcdCertificates := etcd.GetEtcdEndpoint(vConfig)

	if err := etcd.WaitForEtcd(ctx, etcdCertificates, etcdEndpoint); err != nil {
		return fmt.Errorf("failed to wait for embedded etcd: %w", err)
	}

	etcdClient, err := etcd.GetEtcdClient(ctx, etcdCertificates, etcdEndpoint)
	if err != nil {
		return fmt.Errorf("failed to get etcd client: %w", err)
	}
	defer etcdClient.Close()

	if len(skipKeysBytes) > 0 {
		skipKeys := make(map[string]struct{})
		if err := json.Unmarshal(skipKeysBytes, &skipKeys); err != nil {
			return fmt.Errorf("failed to unmarshal skipKeys: %w", err)
		}

		for key := range skipKeys {
			log.Info("Deleting skipped key", "key", key)
			if _, err := etcdClient.Delete(ctx, key); ignoreKeyNotFound(err) != nil {
				return fmt.Errorf("failed to delete key %s: %w", key, err)
			}
		}
	}

	if o.NewVCluster {
		log.Info("Deleting old kube-root-ca.crt")
		resp, err := etcdClient.Get(ctx, configMapPrefix, clientv3.WithPrefix(), clientv3.WithKeysOnly(), clientv3.WithRev(int64(0)))
		if err != nil {
			return fmt.Errorf("failed to get config maps: %w", err)
		}

		for _, kv := range resp.Kvs {
			if strings.HasSuffix(string(kv.Key), "/kube-root-ca.crt") {
				log.Info("Deleting old kube-root-ca.crt", "key", string(kv.Key))
				if _, err := etcdClient.Delete(ctx, string(kv.Key)); err != nil {
					return fmt.Errorf("failed to delete kube-root-ca.crt %s: %w", string(kv.Key), err)
				}
			}
		}

		log.Info("Deleting old vcluster mappings")
		if _, err := etcdClient.Delete(ctx, store.MappingsPrefix, clientv3.WithPrefix(), clientv3.WithRev(int64(0))); ignoreKeyNotFound(err) != nil {
			return fmt.Errorf("failed to delete mappings prefix: %w", err)
		}
	}

	decoder := serializer.NewCodecFactory(scheme.Scheme).UniversalDeserializer()
	encoder := protobuf.NewSerializer(scheme.Scheme, scheme.Scheme)

	// transform pods to make sure they are not deleted on start
	if !vConfig.PrivateNodes.Enabled {
		podsCh, podsErrCh := mirror.NewSyncer(etcdClient, podPrefix, 0).SyncBase(ctx)
		for resp := range podsCh {
			log.Info("Resetting pods nodeName and status", "count", len(resp.Kvs))
			for _, kv := range resp.Kvs {
				value, err := transformPod(kv.Value, decoder, encoder)
				if err != nil {
					return fmt.Errorf("failed to transform pod: %w", err)
				}
				if _, err := etcdClient.Put(ctx, string(kv.Key), string(value)); err != nil {
					return fmt.Errorf("failed to put pod %s: %w", string(kv.Key), err)
				}
			}
		}

		for podsErr := range podsErrCh {
			return fmt.Errorf("failed to sync pods: %w", podsErr)
		}
	}

	return nil
}

func (o *RestoreClient) restoreKeyValueSnapshot(ctx context.Context, vConfig *config.VirtualClusterConfig, snapshotPath string) (retErr error) {
	// create decoder and encoder
	decoder := serializer.NewCodecFactory(scheme.Scheme).UniversalDeserializer()
	encoder := protobuf.NewSerializer(scheme.Scheme, scheme.Scheme)

	// set global vCluster name
	translate.VClusterName = vConfig.Name

	// now stream objects from object store to etcd
	reader, err := os.Open(snapshotPath)
	if err != nil {
		return fmt.Errorf("failed to get backup: %w", err)
	}
	defer reader.Close()

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
	o.etcdClient = etcdClient

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
		key, value, err := readArchiveEntry(tarReader)
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

	// remove stale kine socket from a previous run to avoid "address already in use" errors
	kineSocketPath := filepath.Join(constants.DataDir, "kine.sock")
	_ = os.Remove(kineSocketPath)

	// create a new context that can be cancelled
	kineCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// start & stop kine to create the database
	doneChan := k8s.StartKineWithDone(kineCtx, fmt.Sprintf("sqlite://%s%s", file, k8s.SQLiteParams), constants.K8sKineEndpoint, nil,
		// disable the kine metrics listener, not required for snapshots and would conflict on port
		[]string{"--metrics-bind-address=0"},
	)

	// wait until file is created or kine fails or timeout
	kineStartTimeout := 30 * time.Second
	timeoutTimer := time.NewTimer(kineStartTimeout)
	defer timeoutTimer.Stop()
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	var fileCreated bool
	for !fileCreated {
		select {
		case err := <-doneChan:
			// kine exited before creating the file
			if err != nil {
				return fmt.Errorf("kine exited before creating database: %w", err)
			}
			return fmt.Errorf("kine exited before creating database")
		case <-timeoutTimer.C:
			cancel()
			// drain doneChan to prevent goroutine leak from unbuffered channel send
			<-doneChan
			return fmt.Errorf("timed out waiting for kine to create database after %s", kineStartTimeout)
		case <-ticker.C:
			if _, err := os.Stat(file); err == nil {
				fileCreated = true
			}
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

func restoreBackupFolder(ctx context.Context, dir string) error {
	backupName := dir + ".backup"
	if _, err := os.Stat(backupName); os.IsNotExist(err) {
		return nil
	}

	klog.FromContext(ctx).Info(fmt.Sprintf("Restoring etcd dataDir %s from backup...", constants.EmbeddedEtcdData))
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("restoreBackupFolder: failed to remove %s: %w", dir, err)
	}

	if err := os.Rename(backupName, dir); err != nil {
		return fmt.Errorf("restoreBackupFolder: failed to rename %s: %w", dir, err)
	}

	return nil
}

func readArchiveEntry(tarReader *tar.Reader) ([]byte, []byte, error) {
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

func writeTempFile(dir string, reader io.Reader) (string, error) {
	f, err := os.CreateTemp(dir, "snapshot-")
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, reader); err != nil {
		return "", fmt.Errorf("write temp file: %w", err)
	}

	return f.Name(), nil
}

// getSnapshotArchiveKind analyzes the snapshot file and determines its archive kind (EtcdSnapshotKind or KeyValueSnapshotKind).
func getSnapshotArchiveKind(fileName string) (SnapshotKind, error) {
	f, err := os.Open(fileName)
	if err != nil {
		return UnknownSnapshotKind, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	gzipReader, err := gzip.NewReader(f)
	if err != nil {
		return UnknownSnapshotKind, fmt.Errorf("create gzip reader: %w", err)
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)

	// read the first entry header
	header, err := tarReader.Next()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return KeyValueSnapshotKind, nil
		}
		return UnknownSnapshotKind, fmt.Errorf("read tar header: %w", err)
	}

	// found release key, reading the next entry header
	if header.Name == snapshotapi.SnapshotReleaseKey {
		header, err = tarReader.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return KeyValueSnapshotKind, nil
			}
			return UnknownSnapshotKind, fmt.Errorf("failed to read tar header: %w", err)
		}
	}

	// found request store key, reading the next entry header
	if strings.HasPrefix(header.Name, RequestStoreKey) {
		header, err = tarReader.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return KeyValueSnapshotKind, nil
			}
			return UnknownSnapshotKind, fmt.Errorf("failed to read tar header: %w", err)
		}
	}

	// found the DBStoreKey, return EtcdSnapshotKind.
	if header.Name == DBStoreKey {
		return EtcdSnapshotKind, nil
	}

	return KeyValueSnapshotKind, nil
}

func ignoreKeyNotFound(err error) error {
	if errors.Is(err, etcdservererrors.ErrKeyNotFound) {
		return nil
	}

	return err
}

func nameAndPeerURLForConfig(vConfig *config.VirtualClusterConfig) (string, string, error) {
	if vConfig.ControlPlane.Standalone.Enabled {
		name := os.Getenv(constants.VClusterStandaloneIPAddressEnvVar)
		if name == "" {
			return "", "", fmt.Errorf("could not determine the IP address for the embedded etcd peer")
		}
		peerURL := fmt.Sprintf("https://%s:2380", name)
		return name, peerURL, nil
	}

	namespace := vConfig.HostNamespace
	if namespace == "" {
		var err error
		namespace, err = clienthelper.CurrentNamespace()
		if err != nil {
			return "", "", err
		}
	}

	name := fmt.Sprintf("%s-0", vConfig.Name)
	peerURL := fmt.Sprintf("https://%s.%s-headless.%s:2380", name, vConfig.Name, namespace)
	return name, peerURL, nil
}

// snapshotRestoreBumpRevision returns the RevisionBump value that incorporates the latest revision and snapshot revision.
func snapshotRestoreBumpRevision(latestRevision int64, snapshotRevision int64, bumpRevision int64) uint64 {
	if latestRevision > snapshotRevision {
		return uint64(latestRevision-snapshotRevision) + uint64(bumpRevision)
	}
	return uint64(bumpRevision)
}
