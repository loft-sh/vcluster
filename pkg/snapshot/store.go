package snapshot

import (
	"context"
	"fmt"
	"io"
	"time"

	vclusterconfig "github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/certs"
	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/etcd"
	"github.com/loft-sh/vcluster/pkg/k8s"
	"github.com/loft-sh/vcluster/pkg/pro"
	setupconfig "github.com/loft-sh/vcluster/pkg/setup/config"
	"github.com/loft-sh/vcluster/pkg/util/servicecidr"
	"go.uber.org/zap"
	"google.golang.org/grpc/grpclog"
	"k8s.io/klog/v2"
)

func newEtcdClient(ctx context.Context, vConfig *config.VirtualClusterConfig, isRestore bool) (etcd.Client, error) {
	// get etcd endpoint
	etcdEndpoint, etcdCertificates := etcd.GetEtcdEndpoint(vConfig)

	// we need to start etcd ourselves when it's embedded etcd or kine based
	if vConfig.BackingStoreType() == vclusterconfig.StoreTypeEmbeddedDatabase || vConfig.BackingStoreType() == vclusterconfig.StoreTypeEmbeddedEtcd {
		if isRestore && !isEtcdReachable(ctx, etcdEndpoint, etcdCertificates) {
			klog.FromContext(ctx).Info(fmt.Sprintf("Embedded backing store %s is not reachable", etcdEndpoint))
			err := startEmbeddedBackingStore(ctx, vConfig)
			if err != nil {
				return nil, fmt.Errorf("start embedded backing store: %w", err)
			}
		} else if !isRestore && vConfig.BackingStoreType() == vclusterconfig.StoreTypeEmbeddedEtcd && !isEtcdReachable(ctx, etcdEndpoint, etcdCertificates) { // this is needed for embedded etcd
			etcdEndpoint = "https://" + vConfig.Name + "-0." + vConfig.Name + "-headless:2379"
		}
	} else if vConfig.BackingStoreType() == vclusterconfig.StoreTypeExternalDatabase {
		if !isEtcdReachable(ctx, etcdEndpoint, etcdCertificates) {
			klog.FromContext(ctx).Info(fmt.Sprintf("External database backing store %s is not reachable, starting...", etcdEndpoint))
			err := startExternalDatabaseBackingStore(ctx, vConfig)
			if err != nil {
				return nil, fmt.Errorf("start external database backing store: %w", err)
			}
		}
	} else if vConfig.BackingStoreType() == vclusterconfig.StoreTypeDeployedEtcd {
		_, err := generateCertificates(ctx, vConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to get certificates: %w", err)
		}
	}

	// create the etcd client
	klog.Info("Creating etcd client...")
	etcdClient, err := etcd.New(ctx, etcdCertificates, etcdEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to create etcd client: %w", err)
	}

	return etcdClient, nil
}

func isEtcdReachable(ctx context.Context, endpoint string, certificates *etcd.Certificates) bool {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if !klog.V(1).Enabled() {
		// prevent etcd client messages from showing
		grpclog.SetLoggerV2(grpclog.NewLoggerV2(io.Discard, io.Discard, io.Discard))
	}
	etcdClient, err := etcd.GetEtcdClient(ctx, zap.L().Named("etcd-client"), certificates, endpoint)
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

	// make sure to start license if using a connector
	if vConfig.ControlPlane.BackingStore.Database.External.Connector != "" {
		// make sure clients and config are correctly initialized
		_, err := generateCertificates(ctx, vConfig)
		if err != nil {
			return fmt.Errorf("failed to get certificates: %w", err)
		}

		// license init
		err = pro.LicenseInit(ctx, vConfig)
		if err != nil {
			return fmt.Errorf("failed to get license: %w", err)
		}
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
		klog.FromContext(ctx).Info("Starting k8s kine embedded database...")
		_, _, err := k8s.StartBackingStore(ctx, vConfig)
		if err != nil {
			return fmt.Errorf("failed to start backing store: %w", err)
		}

		return nil
	}

	// embedded etcd
	if vConfig.BackingStoreType() == vclusterconfig.StoreTypeEmbeddedEtcd {
		_, err := startEmbeddedEtcd(context.WithoutCancel(ctx), vConfig)
		if err != nil {
			return fmt.Errorf("start embedded etcd: %w", err)
		}
	}

	return nil
}

func startEmbeddedEtcd(ctx context.Context, vConfig *config.VirtualClusterConfig) (func(), error) {
	klog.FromContext(ctx).Info("Starting embedded etcd...")
	certificatesDir, err := generateCertificates(ctx, vConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to get certificates: %w", err)
	}

	stop, err := pro.StartEmbeddedEtcd(
		ctx,
		vConfig,
		certificatesDir,
		"",
		false,
		true,
	)
	if err != nil {
		return nil, fmt.Errorf("start embedded etcd: %w", err)
	}

	return stop, nil
}

func generateCertificates(ctx context.Context, vConfig *config.VirtualClusterConfig) (string, error) {
	var err error

	// init the clients
	vConfig.HostConfig, vConfig.HostNamespace, err = setupconfig.InitClientConfig()
	if err != nil {
		return "", err
	}
	err = setupconfig.InitClients(vConfig)
	if err != nil {
		return "", err
	}

	// retrieve service cidr
	serviceCIDR, err := servicecidr.GetServiceCIDR(ctx, &vConfig.Config, vConfig.HostClient, vConfig.Name, vConfig.HostNamespace)
	if err != nil {
		return "", fmt.Errorf("failed to get service cidr: %w", err)
	}

	// generate etcd certificates
	certificatesDir := constants.PKIDir
	err = certs.Generate(ctx, serviceCIDR, certificatesDir, vConfig)
	if err != nil {
		return "", err
	}

	return certificatesDir, nil
}
