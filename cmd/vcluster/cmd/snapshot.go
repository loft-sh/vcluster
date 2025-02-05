package cmd

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	vclusterconfig "github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/etcd"
	"github.com/loft-sh/vcluster/pkg/k8s"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/setup"
	"github.com/loft-sh/vcluster/pkg/snapshot"
	"github.com/loft-sh/vcluster/pkg/snapshot/file"
	"github.com/loft-sh/vcluster/pkg/snapshot/s3"
	"github.com/loft-sh/vcluster/pkg/util/servicecidr"
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
)

type Storage interface {
	Target() string
	PutObject(body io.Reader) error
	GetObject() (io.ReadCloser, error)
}

type SnapshotOptions struct {
	S3   s3.Options
	File file.Options

	Compress bool
	Storage  string
	Prefix   string
	Config   string
}

func NewSnapshotCommand() *cobra.Command {
	options := &SnapshotOptions{}
	envOptions, err := parseOptionsFromEnv()
	if err != nil {
		klog.Warningf("Error parsing environment variables: %v", err)
	} else {
		options.S3 = envOptions.S3
		options.File = envOptions.File
	}

	cmd := &cobra.Command{
		Use:   "snapshot",
		Short: "snapshot a vCluster",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return options.Run(cmd.Context())
		},
	}

	cmd.Flags().StringVar(&options.Config, "config", constants.DefaultVClusterConfigLocation, "The path where to find the vCluster config to load")
	cmd.Flags().StringVar(&options.Prefix, "prefix", "/", "The prefix to use to snapshot the etcd")
	cmd.Flags().StringVar(&options.Storage, "storage", "s3", "The storage to snapshot to. Can be either s3 or file")
	cmd.Flags().BoolVar(&options.Compress, "compress", false, "If the snapshot should be compressed")

	// add storage flags
	file.AddFileFlags(cmd.Flags(), &options.File)
	s3.AddS3Flags(cmd.Flags(), &options.S3)
	return cmd
}

func (o *SnapshotOptions) Run(ctx context.Context) error {
	// parse vCluster config
	vConfig, err := config.ParseConfig(o.Config, os.Getenv("VCLUSTER_NAME"), nil)
	if err != nil {
		return err
	}

	// make sure to validate options
	err = validateOptions(vConfig, o.Storage, &o.S3, &o.File)
	if err != nil {
		return err
	}

	// create new etcd client
	etcdClient, err := newEtcdClient(ctx, vConfig, false)
	if err != nil {
		return fmt.Errorf("failed to create etcd client: %w", err)
	}

	// create store
	objectStore, err := createStore(ctx, o.Storage, &o.S3, &o.File)
	if err != nil {
		return fmt.Errorf("failed to create store: %w", err)
	}

	// write the snapshot
	klog.Infof("Start writing etcd snapshot...")
	err = o.writeSnapshot(ctx, etcdClient, objectStore)
	if err != nil {
		return err
	}

	klog.Infof("Successfully wrote snapshot to %s", objectStore.Target())
	return nil
}

func (o *SnapshotOptions) writeSnapshot(ctx context.Context, etcdClient etcd.Client, objectStore Storage) error {
	// now stream objects from etcd to object store
	errChan := make(chan error)
	reader, writer, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("failed to create pipe: %w", err)
	}
	defer writer.Close()
	go func() {
		defer reader.Close()
		errChan <- objectStore.PutObject(reader)
	}()

	// start listing the keys
	listChan := etcdClient.ListStream(ctx, o.Prefix)

	// optionally compress
	gzipWriter := io.WriteCloser(writer)
	if o.Compress {
		gzipWriter = gzip.NewWriter(writer)
		defer gzipWriter.Close()
	}

	// create a new tar write
	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

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

				// write the object into the store
				klog.V(1).Infof("Snapshot key %s", string(obj.Value.Key))
				err := writeKeyValue(tarWriter, obj.Value.Key, obj.Value.Data)
				if err != nil {
					return fmt.Errorf("failed to snapshot key %s: %w", string(obj.Value.Key), err)
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

func validateOptions(vConfig *config.VirtualClusterConfig, storage string, s3Options *s3.Options, fileOptions *file.Options) error {
	// storage needs to be either s3 or file
	if storage == "s3" {
		if s3Options.Key == "" {
			return fmt.Errorf("--s3-key must be specified")
		}
		if s3Options.Bucket == "" {
			return fmt.Errorf("--s3-bucket must be specified")
		}
	} else if storage == "file" {
		if fileOptions.Path == "" {
			return fmt.Errorf("--file-path must be specified")
		}
	} else {
		return fmt.Errorf("--storage must be either 's3' or 'file'")
	}

	// only support k3s and k8s distro
	if vConfig.Distro() != vclusterconfig.K8SDistro && vConfig.Distro() != vclusterconfig.K3SDistro {
		return fmt.Errorf("unsupported distro: %s", vConfig.Distro())
	}

	return nil
}

func newEtcdClient(ctx context.Context, vConfig *config.VirtualClusterConfig, startEmbedded bool) (etcd.Client, error) {
	// get etcd endpoint
	etcdEndpoint, etcdCertificates := etcd.GetEtcdEndpoint(vConfig)

	// we need to start etcd ourselves when it's embedded etcd or kine based
	if vConfig.BackingStoreType() == vclusterconfig.StoreTypeEmbeddedDatabase || vConfig.BackingStoreType() == vclusterconfig.StoreTypeEmbeddedEtcd {
		if startEmbedded && !isEtcdReachable(ctx, etcdEndpoint, etcdCertificates) {
			klog.FromContext(ctx).Info(fmt.Sprintf("Embedded backing store %s is not reachable", etcdEndpoint))
			err := startEmbeddedBackingStore(ctx, vConfig)
			if err != nil {
				return nil, fmt.Errorf("start embedded backing store: %w", err)
			}
		}
	} else if vConfig.BackingStoreType() == vclusterconfig.StoreTypeExternalDatabase {
		if !isEtcdReachable(ctx, etcdEndpoint, etcdCertificates) {
			klog.FromContext(ctx).Info(fmt.Sprintf("External database backing store %s is not reachable, starting...", etcdEndpoint))
			err := startExternalDatabaseBackingStore(ctx, vConfig)
			if err != nil {
				return nil, fmt.Errorf("start external database backing store: %w", err)
			}
		}
	}

	// create the etcd client
	klog.Info("Creating etcd client...")
	etcdClient, err := etcd.NewFromConfig(ctx, vConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create etcd client: %w", err)
	}

	return etcdClient, nil
}

func isEtcdReachable(ctx context.Context, endpoint string, certificates *etcd.Certificates) bool {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	etcdClient, err := etcd.GetEtcdClient(ctx, certificates, endpoint)
	if err == nil {
		defer func() {
			_ = etcdClient.Close()
		}()

		_, err = etcdClient.MemberList(ctx)
		if err == nil {
			return true
		}
	}

	return false
}

func startExternalDatabaseBackingStore(ctx context.Context, vConfig *config.VirtualClusterConfig) error {
	kineAddress := constants.K8sKineEndpoint
	if vConfig.Distro() == vclusterconfig.K3SDistro {
		kineAddress = constants.K3sKineEndpoint
	}

	// call out to the pro code
	_, _, err := pro.ConfigureExternalDatabase(ctx, kineAddress, vConfig, false)
	if err != nil {
		return err
	}

	return nil
}

func startEmbeddedBackingStore(ctx context.Context, vConfig *config.VirtualClusterConfig) error {
	// embedded database
	if vConfig.EmbeddedDatabase() {
		if vConfig.Distro() == vclusterconfig.K8SDistro {
			klog.FromContext(ctx).Info("Starting k8s kine embedded database...")
			_, _, err := k8s.StartBackingStore(ctx, vConfig)
			if err != nil {
				return fmt.Errorf("failed to start backing store: %w", err)
			}
		} else if vConfig.Distro() == vclusterconfig.K3SDistro {
			klog.FromContext(ctx).Info("Starting k3s kine embedded database...")
			err := os.MkdirAll(filepath.Dir(constants.K3sSqliteDatabase), 0777)
			if err != nil {
				return fmt.Errorf("failed to create directory %s: %w", filepath.Dir(constants.K3sSqliteDatabase), err)
			}

			k8s.StartKine(ctx, fmt.Sprintf("sqlite://%s?_journal=WAL&cache=shared&_busy_timeout=30000", constants.K3sSqliteDatabase), constants.K3sKineEndpoint, nil)
		} else {
			return fmt.Errorf("unsupported distro: %s", vConfig.Distro())
		}

		return nil
	}

	// embedded etcd
	if vConfig.BackingStoreType() == vclusterconfig.StoreTypeEmbeddedEtcd {
		var err error
		klog.FromContext(ctx).Info("Starting embedded etcd...")

		// init the clients
		vConfig.ControlPlaneConfig, vConfig.ControlPlaneNamespace, vConfig.ControlPlaneService, vConfig.WorkloadConfig, vConfig.WorkloadNamespace, vConfig.WorkloadService, err = pro.GetRemoteClient(vConfig)
		if err != nil {
			return err
		}
		err = setup.InitClients(vConfig)
		if err != nil {
			return err
		}

		// retrieve service cidr
		serviceCIDR := vConfig.ServiceCIDR
		if serviceCIDR == "" {
			var warning string
			serviceCIDR, warning = servicecidr.GetServiceCIDR(ctx, vConfig.WorkloadClient, vConfig.WorkloadNamespace)
			if warning != "" {
				klog.Warning(warning)
			}
		}

		// generate etcd certificates
		certificatesDir := "/data/pki"
		err = setup.GenerateCerts(ctx, vConfig.ControlPlaneClient, vConfig.Name, vConfig.ControlPlaneNamespace, serviceCIDR, certificatesDir, vConfig)
		if err != nil {
			return err
		}

		// we need to run this with the parent ctx as otherwise this context
		// will be cancelled by the wait loop in Initialize
		err = pro.StartEmbeddedEtcd(
			context.WithoutCancel(ctx),
			vConfig.Name,
			vConfig.ControlPlaneNamespace,
			certificatesDir,
			int(vConfig.ControlPlane.StatefulSet.HighAvailability.Replicas),
			"",
			false,
		)
		if err != nil {
			return fmt.Errorf("start embedded etcd: %w", err)
		}
	}

	return nil
}

func createStore(ctx context.Context, storage string, s3Options *s3.Options, fileOptions *file.Options) (Storage, error) {
	if storage == "s3" {
		objectStore := s3.NewObjectStore(klog.FromContext(ctx))
		err := objectStore.Init(s3Options)
		if err != nil {
			return nil, fmt.Errorf("failed to init s3 object store: %w", err)
		}

		return objectStore, nil
	} else if storage == "file" {
		return file.NewFileStore(fileOptions), nil
	}

	return nil, fmt.Errorf("unknown storage: %s", storage)
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

func parseOptionsFromEnv() (*snapshot.Options, error) {
	snapshotOptions := os.Getenv("VCLUSTER_STORAGE_OPTIONS")
	if snapshotOptions == "" {
		return &snapshot.Options{}, nil
	}

	decoded, err := base64.StdEncoding.DecodeString(snapshotOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to decode storage options from env: %w", err)
	}

	options := &snapshot.Options{}
	err = json.Unmarshal(decoded, options)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal storage options from env: %w", err)
	}

	return options, nil
}
