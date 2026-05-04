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
	"net/url"
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
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/scheme"
	setupconfig "github.com/loft-sh/vcluster/pkg/setup/config"
	"github.com/loft-sh/vcluster/pkg/snapshot/volumes"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/etcdutl/v3/snapshot"
	etcdservererrors "go.etcd.io/etcd/server/v3/etcdserver/errors"
	"go.etcd.io/etcd/server/v3/storage/backend"
	"go.etcd.io/etcd/server/v3/storage/datadir"
	"go.etcd.io/etcd/server/v3/storage/mvcc"
	"go.etcd.io/etcd/server/v3/storage/schema"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/runtime/serializer/protobuf"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/klog/v2"
)

const (
	pvPrefix  = "/registry/persistentvolumes/"
	pvcPrefix = "/registry/persistentvolumeclaims/"
)

type RestoreClient struct {
	snapshotRequest Request
	Snapshot        Options
	RestoreVolumes  bool

	etcdClient etcd.Client

	NewVCluster bool
}

var (
	podGVK = corev1.SchemeGroupVersion.WithKind("Pod")
	pvcGVK = corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim")

	// bump revision to make sure we invalidate caches. See https://github.com/kubernetes/kubernetes/issues/118501 for more details
	BumpRevision = int64(1000)
)

func NewRestoreClient(snapshotOptions Options, restoreVolumes bool, newVCluster bool) *RestoreClient {
	return &RestoreClient{
		Snapshot:       snapshotOptions,
		RestoreVolumes: restoreVolumes,
		NewVCluster:    newVCluster,
	}
}

func (o *RestoreClient) GetSnapshotRequest(ctx context.Context) (*Request, error) {
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

		var snapshotRequest Request
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

	snapshotPath, err := writeTempFile(snapshotReader)
	if err != nil {
		return fmt.Errorf("failed to write snapshot to temp file: %w", err)
	}
	defer os.Remove(snapshotPath)

	typeMetaStamp, err := gzipCommentFromFile(snapshotPath)
	if err != nil {
		return fmt.Errorf("failed to extract type meta stamp from snapshot: %w", err)
	}

	if typeMetaStamp == EtcdSnapshotTypeMetaStamp {
		if vConfig.BackingStoreType() == vclusterconfig.StoreTypeEmbeddedEtcd {
			if err := o.restoreSnapshot(ctx, vConfig, snapshotPath); err != nil {
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

func (o *RestoreClient) restoreSnapshot(ctx context.Context, vConfig *config.VirtualClusterConfig, snapshotPath string) (retErr error) {
	log := klog.FromContext(ctx)

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

	dbPath := ""
	defer func() {
		if dbPath != "" {
			_ = os.Remove(dbPath)
		}
	}()

	encoder := protobuf.NewSerializer(scheme.Scheme, scheme.Scheme)

	var skipKeys map[string]struct{}

	for {
		header, err := tarReader.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			return fmt.Errorf("failed to read tar header: %w", err)
		}

		if header.Name == DBStoreKey {
			dbPath, err = writeTempFile(tarReader)
			if err != nil {
				return fmt.Errorf("failed to write snapshot to temp file: %w", err)
			}

			continue
		}

		if strings.HasPrefix(header.Name, RequestStoreKey) {
			if o.RestoreVolumes {
				buf := &bytes.Buffer{}
				_, err = io.Copy(buf, tarReader)
				if err != nil {
					return fmt.Errorf("failed to read snapshot request: %w", err)
				}

				err = o.createRestoreRequest(ctx, vConfig, buf.Bytes(), encoder)
				if err != nil {
					return fmt.Errorf("failed to create restore request: %w", err)
				}
			}

			continue
		}

		if header.Name == SkipKeysStoreKey {
			skipKeysBytes, err := io.ReadAll(tarReader)
			if err != nil {
				return fmt.Errorf("failed to read skipKeys from tar archive: %w", err)
			}
			err = json.Unmarshal(skipKeysBytes, &skipKeys)
			if err != nil {
				return fmt.Errorf("failed to unmarshal skipKeys: %w", err)
			}

			continue
		}
	}

	if dbPath == "" {
		return fmt.Errorf("failed to find etcd snapshot in tar archive")
	}

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

	log.Info("Restoring etcd snapshot", "dbPath", dbPath)
	if err := snapshot.NewV3(zap.L().Named("restore-etcd")).Restore(snapshot.RestoreConfig{
		SnapshotPath:        dbPath,
		Name:                "default",
		OutputDataDir:       constants.EmbeddedEtcdData,
		OutputWALDir:        datadir.ToWALDir(constants.EmbeddedEtcdData),
		PeerURLs:            []string{"http://127.0.0.1:2380"},
		InitialCluster:      "default=http://127.0.0.1:2380",
		InitialClusterToken: "etcd-cluster",
		SkipHashCheck:       false,
		InitialMmapSize:     backend.InitialMmapSize,
		RevisionBump:        uint64(BumpRevision),
		MarkCompacted:       true,
	}); err != nil {
		return fmt.Errorf("restore etcd: %w", err)
	}

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

	etcdClient, err := etcd.GetEtcdClient(ctx, zap.L().Named("etcd-client"), etcdCertificates, etcdEndpoint)
	if err != nil {
		return fmt.Errorf("failed to get etcd client: %w", err)
	}
	defer etcdClient.Close()

	if err := fixSeedMemberPeerUrls(ctx, etcdClient); err != nil {
		return fmt.Errorf("failed to fix seed member peer urls: %w", err)
	}

	if len(skipKeys) > 0 {
		log.Info("Deleting skip keys")
		for key := range skipKeys {
			if _, err := etcdClient.Delete(ctx, key); ignoreKeyNotFound(err) != nil {
				return fmt.Errorf("failed to delete key %s: %w", key, err)
			}
		}
	}

	if o.NewVCluster {
		log.Info("Deleting old kube-root-ca.crt")
		if _, err := etcdClient.Delete(ctx, "/registry/configmaps/default/kube-root-ca.crt"); ignoreKeyNotFound(err) != nil {
			return fmt.Errorf("failed to delete kube-root-ca.crt: %w", err)
		}

		log.Info("Deleting old vcluster mappings")
		if _, err := etcdClient.Delete(ctx, store.MappingsPrefix, clientv3.WithPrefix(), clientv3.WithRev(int64(0))); ignoreKeyNotFound(err) != nil {
			return fmt.Errorf("failed to delete mappings prefix: %w", err)
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

		// check snapshot request
		if strings.HasPrefix(string(key), RequestStoreKey) {
			if o.RestoreVolumes {
				err = o.createRestoreRequest(ctx, vConfig, value, encoder)
				if err != nil {
					return fmt.Errorf("failed to create restore request: %w", err)
				}
			}
			continue
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

		if o.isPVCThatShouldBeRestoredInHost(string(key), vConfig) {
			value, err = unsetVolumeName(value, decoder, encoder)
			if err != nil {
				return fmt.Errorf("failed to unset volume name: %w", err)
			}
			klog.Infof("Unset volume name for PVC %s", strings.TrimPrefix(string(key), pvcPrefix))
		}

		// write the value to etcd
		if o.skipKey(string(key), vConfig) {
			klog.Infof("Skip key %s", string(key))
			continue
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

func (o *RestoreClient) createRestoreRequest(ctx context.Context, vConfig *config.VirtualClusterConfig, value []byte, encoder runtime.Encoder) error {
	klog.V(1).Infof("Found snapshot request object %s", string(value))
	var err error
	if !vConfig.ControlPlane.Standalone.Enabled {
		// init the clients
		if vConfig.HostConfig == nil || vConfig.HostNamespace == "" {
			vConfig.HostConfig, vConfig.HostNamespace, err = setupconfig.InitClientConfig()
			if err != nil {
				return fmt.Errorf("failed to init client config: %w", err)
			}
		}
		if vConfig.HostClient == nil {
			err = setupconfig.InitClients(vConfig)
			if err != nil {
				return fmt.Errorf("failed to init clients: %w", err)
			}
		}
	}

	var snapshotRequest Request
	err = json.Unmarshal(value, &snapshotRequest)
	if err != nil {
		return fmt.Errorf("failed to unmarshal snapshot request: %w", err)
	}
	klog.Infof("Found snapshot request: %s", snapshotRequest.Name)
	o.snapshotRequest = snapshotRequest

	// first create the snapshot options Secret
	namespace := vConfig.HostNamespace
	secret, err := CreateSnapshotOptionsSecret(constants.RestoreRequestLabel, namespace, vConfig.Name, &o.Snapshot)
	if err != nil {
		return fmt.Errorf("failed to create snapshot options Secret: %w", err)
	}

	if vConfig.ControlPlane.Standalone.Enabled {
		// Standalone: kube-apiserver is down during restore, so write resources directly to etcd.
		secret.Name = fmt.Sprintf("%s-restore-request-%s", vConfig.Name, rand.String(5))
		secret.TypeMeta = metav1.TypeMeta{APIVersion: "v1", Kind: "Secret"}
		secret.UID = uuid.NewUUID()
		secret.CreationTimestamp = metav1.Now()

		secretBuf := &bytes.Buffer{}
		if err = encoder.Encode(secret, secretBuf); err != nil {
			return fmt.Errorf("encode restore secret: %w", err)
		}
		if _, err = o.etcdClient.Put(ctx, fmt.Sprintf("/registry/secrets/%s/%s", namespace, secret.Name), secretBuf.Bytes()); err != nil {
			return fmt.Errorf("write restore secret to etcd: %w", err)
		}

		restoreRequest, err := NewRestoreRequest(snapshotRequest)
		if err != nil {
			return fmt.Errorf("failed to create restore request: %w", err)
		}
		restoreRequest.Name = secret.Name
		configMap, err := CreateRestoreRequestConfigMap(namespace, vConfig.Name, restoreRequest)
		if err != nil {
			return fmt.Errorf("failed to create snapshot request ConfigMap: %w", err)
		}
		configMap.Name = secret.Name
		configMap.TypeMeta = metav1.TypeMeta{APIVersion: "v1", Kind: "ConfigMap"}
		configMap.UID = uuid.NewUUID()
		configMap.CreationTimestamp = metav1.Now()

		cmBuf := &bytes.Buffer{}
		if err = encoder.Encode(configMap, cmBuf); err != nil {
			return fmt.Errorf("encode restore configmap: %w", err)
		}
		if _, err = o.etcdClient.Put(ctx, fmt.Sprintf("/registry/configmaps/%s/%s", namespace, configMap.Name), cmBuf.Bytes()); err != nil {
			return fmt.Errorf("write restore configmap to etcd: %w", err)
		}

		klog.Infof("Created restore request in etcd: %s/%s", namespace, configMap.Name)
		return nil
	}

	secret.GenerateName = fmt.Sprintf("%s-restore-request-", vConfig.Name)
	secret, err = vConfig.HostClient.CoreV1().Secrets(vConfig.HostNamespace).Create(ctx, secret, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create snapshot options Secret: %w", err)
	}

	// then create the restore request that will be reconciled by the controller
	restoreRequest, err := NewRestoreRequest(snapshotRequest)
	if err != nil {
		return fmt.Errorf("failed to create restore request: %w", err)
	}
	restoreRequest.Name = secret.Name
	configMap, err := CreateRestoreRequestConfigMap(vConfig.HostNamespace, vConfig.Name, restoreRequest)
	if err != nil {
		return fmt.Errorf("failed to create snapshot request ConfigMap: %w", err)
	}
	configMap.Name = secret.Name
	_, err = vConfig.HostClient.CoreV1().ConfigMaps(vConfig.HostNamespace).Create(ctx, configMap, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create snapshot request ConfigMap: %w", err)
	}

	klog.Infof("Created restore request: %s/%s", configMap.Namespace, configMap.Name)
	return nil
}

func (o *RestoreClient) skipKey(key string, vConfig *config.VirtualClusterConfig) bool {
	if !o.RestoreVolumes {
		return false
	}

	if vConfig.PrivateNodes.Enabled {
		// check if the snapshot exists
		if strings.HasPrefix(key, pvcPrefix) {
			pvcName := strings.TrimPrefix(key, pvcPrefix)
			status, ok := o.snapshotRequest.Status.VolumeSnapshots.Snapshots[pvcName]
			if !ok {
				return false
			}
			// skip restoring PVC if it has a snapshot, restore controller will restore it
			return status.Phase == volumes.RequestPhaseCompleted
		} else if strings.HasPrefix(key, pvPrefix) {
			volumeName := strings.TrimPrefix(key, pvPrefix)
			for _, snapshotSpec := range o.snapshotRequest.Spec.VolumeSnapshots.Requests {
				if snapshotSpec.PersistentVolumeClaim.Spec.VolumeName == volumeName {
					return true
				}
			}
		}
	} else {
		// In the shared mode, PVC restore is skipped for the ones where snapshots have failed.

		// check if the snapshot exists
		if strings.HasPrefix(key, pvcPrefix) {
			pvcName := strings.TrimPrefix(key, pvcPrefix)

			translatedPVCName := getTranslatedPVCName(pvcName)
			if translatedPVCName == "" {
				return true
			}
			status, ok := o.snapshotRequest.Status.VolumeSnapshots.Snapshots[translatedPVCName]
			if !ok {
				return false
			}
			// skip restoring PVC if it has a snapshot, restore controller will restore it
			return status.Phase != volumes.RequestPhaseCompleted
		}
	}

	return false
}

// isPVCThatShouldBeRestoredInHost checks if the key refers to a PVC that should be restored on the host cluster.
func (o *RestoreClient) isPVCThatShouldBeRestoredInHost(key string, vConfig *config.VirtualClusterConfig) bool {
	if !o.RestoreVolumes {
		return false
	}
	if !strings.HasPrefix(key, pvcPrefix) {
		// not PVC
		return false
	}
	if vConfig.PrivateNodes.Enabled {
		// restore in virtual, not in host
		return false
	}

	pvcName := strings.TrimPrefix(key, pvcPrefix)
	translatedPVCName := getTranslatedPVCName(pvcName)
	if translatedPVCName == "" {
		return true
	}
	status, ok := o.snapshotRequest.Status.VolumeSnapshots.Snapshots[translatedPVCName]
	if !ok {
		klog.V(1).Infof("Snapshot not found for PVC %s", strings.TrimPrefix(key, pvcPrefix))
		return false
	}
	// skip restoring PVC if it has a snapshot, restore controller will restore it
	klog.Infof("Snapshot found for PVC %s with phase %s", strings.TrimPrefix(key, pvcPrefix), status.Phase)
	return status.Phase == volumes.RequestPhaseCompleted
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

func unsetVolumeName(value []byte, decoder runtime.Decoder, encoder runtime.Encoder) ([]byte, error) {
	// decode value
	obj := &corev1.PersistentVolumeClaim{}
	_, _, err := decoder.Decode(value, &pvcGVK, obj)
	if err != nil {
		return nil, fmt.Errorf("decode value: %w", err)
	} else if obj.DeletionTimestamp != nil {
		klog.Infof("PVC deleted!")
		return value, nil
	}

	// unset volume name
	obj.Spec.VolumeName = ""

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
	doneChan := k8s.StartKineWithDone(kineCtx, fmt.Sprintf("sqlite://%s%s", file, k8s.SQLiteParams), constants.K8sKineEndpoint, nil, nil)

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

func getTranslatedPVCName(pvcName string) string {
	// Parse namespace and name from "namespace/name" format
	parts := strings.SplitN(pvcName, "/", 2)
	if len(parts) != 2 {
		return ""
	}

	vNamespace, vName := parts[0], parts[1]
	hostName := translate.Default.HostName(nil, vName, vNamespace)
	return hostName.Namespace + "/" + hostName.Name
}

func writeTempFile(reader io.Reader) (string, error) {
	f, err := os.CreateTemp("", "snapshot-")
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, reader); err != nil {
		return "", fmt.Errorf("write temp file: %w", err)
	}

	return f.Name(), nil
}

func gzipCommentFromFile(fileName string) (string, error) {
	f, err := os.Open(fileName)
	if err != nil {
		return "", fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	gzipReader, err := gzip.NewReader(f)
	if err != nil {
		return "", fmt.Errorf("create gzip reader: %w", err)
	}
	defer gzipReader.Close()
	return gzipReader.Comment, nil
}

func ignoreKeyNotFound(err error) error {
	if errors.Is(err, etcdservererrors.ErrKeyNotFound) {
		return nil
	}

	return err
}

// fixSeedMemberPeerUrls fixes the peer urls of the seed member to match the expected value.
// The required value is not available at the time of the WAL restore.
func fixSeedMemberPeerUrls(ctx context.Context, etcdClient *clientv3.Client) error {
	log := klog.FromContext(ctx)

	memberList, err := etcdClient.MemberList(ctx)
	if err != nil {
		return fmt.Errorf("failed to get member list: %w", err)
	}

	if len(memberList.Members) != 1 {
		log.Info("Not a single member cluster, skipping fixing seed member peer urls")
		return nil
	}

	member := memberList.Members[0]

	if len(member.PeerURLs) != 1 {
		log.Info("Unexpected number of peer urls, skipping fixing seed member peer urls", "peerUrls", member.PeerURLs)
		return nil
	}

	if member.PeerURLs[0] != "http://127.0.0.1:2380" {
		log.Info("Unexpected peer url, skipping fixing seed member peer urls", "peerUrl", member.PeerURLs[0])
		return nil
	}

	if len(member.ClientURLs) == 0 {
		log.Info("No client urls, skipping fixing seed member peer urls")
		return nil
	}

	clientURL, err := url.Parse(member.ClientURLs[0])
	if err != nil {
		return fmt.Errorf("failed to parse client url: %w", err)
	}

	peerURL := fmt.Sprintf("https://%s:2380", clientURL.Hostname())
	log.Info("Updating seed member peer url", "peerUrl", peerURL)
	if _, err := etcdClient.MemberUpdate(ctx, member.ID, []string{peerURL}); err != nil {
		return fmt.Errorf("failed to update peer url: %w", err)
	}

	return nil
}
