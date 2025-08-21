package setup

import (
	"context"
	"fmt"
	"path/filepath"

	vclusterconfig "github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/certs"
	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/k3s"
	"github.com/loft-sh/vcluster/pkg/k8s"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/specialservices"
	"github.com/loft-sh/vcluster/pkg/telemetry"
	"github.com/loft-sh/vcluster/pkg/util/servicecidr"
	"k8s.io/klog/v2"
)

// Initialize creates the required secrets and configmaps for the control plane to start
func Initialize(ctx context.Context, options *config.VirtualClusterConfig) error {
	// start kubernetes & etcd
	err := initialize(ctx, options)
	if err != nil {
		return fmt.Errorf("initialize: %w", err)
	}

	specialservices.Default = pro.InitDNSServiceSyncing(options)
	telemetry.CollectorControlPlane.RecordStart(ctx, options)
	return nil
}

// initialize creates the required secrets and configmaps for the control plane to start
func initialize(ctx context.Context, options *config.VirtualClusterConfig) error {
	distro := options.Distro()

	// migrate from
	migrateFrom := ""
	if options.ControlPlane.BackingStore.Etcd.Embedded.Enabled && options.ControlPlane.BackingStore.Etcd.Embedded.MigrateFromDeployedEtcd {
		if options.ControlPlane.BackingStore.Etcd.Deploy.Service.Enabled {
			migrateFrom = "https://" + options.Name + "-etcd:2379"
		} else {
			migrateFrom = "https://" + options.Name + "-etcd-headless:2379"
		}
	}

	// retrieve service cidr
	serviceCIDR, err := servicecidr.GetServiceCIDR(ctx, &options.Config, options.HostClient, options.Name, options.HostNamespace)
	if err != nil {
		return fmt.Errorf("failed to get service cidr: %w", err)
	}

	// check what distro are we running
	switch distro {
	case vclusterconfig.K3SDistro:
		// its k3s, let's create the token secret
		k3sToken, err := k3s.EnsureK3SToken(ctx, options.HostClient, options.HostNamespace, options.Name, options)
		if err != nil {
			return err
		}

		// generate etcd certificates
		certificatesDir := "/data/pki"
		err = certs.Generate(ctx, serviceCIDR, certificatesDir, options)
		if err != nil {
			return err
		}

		// should start embedded etcd?
		if options.ControlPlane.BackingStore.Etcd.Embedded.Enabled {
			// we need to run this with the parent ctx as otherwise this context
			// will be cancelled by the wait loop in Initialize
			_, err = pro.StartEmbeddedEtcd(
				context.WithoutCancel(ctx),
				options.Name,
				options.HostNamespace,
				options.HostClient,
				certificatesDir,
				options.ControlPlane.BackingStore.Etcd.Embedded.SnapshotCount,
				migrateFrom,
				true,
				options.ControlPlane.BackingStore.Etcd.Embedded.ExtraArgs,
				false,
			)
			if err != nil {
				return fmt.Errorf("start embedded etcd: %w", err)
			}
		}

		// start k3s
		go func() {
			// we need to run this with the parent ctx as otherwise this context will be cancelled by the wait
			// loop in Initialize
			err := k3s.StartK3S(context.WithoutCancel(ctx), options, serviceCIDR, k3sToken)
			if err != nil {
				klog.Fatalf("Error running k3s: %v", err)
			}
		}()
	case vclusterconfig.K8SDistro:
		// migrate k3s to k8s if needed
		err := k8s.MigrateK3sToK8s(ctx, options.HostClient, options.HostNamespace, options)
		if err != nil {
			return fmt.Errorf("migrate k3s to k8s: %w", err)
		}

		// try to generate k8s certificates
		certificatesDir := filepath.Dir(options.VirtualClusterKubeConfig().ServerCACert)
		if certificatesDir == constants.PKIDir {
			err := certs.Generate(ctx, serviceCIDR, certificatesDir, options)
			if err != nil {
				return err
			}
		}

		// should start embedded etcd?
		if options.ControlPlane.BackingStore.Etcd.Embedded.Enabled {
			// start embedded etcd
			_, err := pro.StartEmbeddedEtcd(
				context.WithoutCancel(ctx),
				options.Name,
				options.HostNamespace,
				options.HostClient,
				certificatesDir,
				options.ControlPlane.BackingStore.Etcd.Embedded.SnapshotCount,
				migrateFrom,
				true,
				options.ControlPlane.BackingStore.Etcd.Embedded.ExtraArgs,
				false,
			)
			if err != nil {
				return fmt.Errorf("start embedded etcd: %w", err)
			}
		}

		// start k8s
		go func() {
			// we need to run this with the parent ctx as otherwise this context will be cancelled by the wait
			// loop in Initialize
			err := k8s.StartK8S(
				context.WithoutCancel(ctx),
				serviceCIDR,
				options,
			)
			if err != nil {
				klog.Fatalf("Error running k8s: %v", err)
			}
		}()
	case vclusterconfig.Unknown:
		certificatesDir := filepath.Dir(options.VirtualClusterKubeConfig().ServerCACert)
		if certificatesDir == "/data/pki" {
			// generate k8s certificates
			err := certs.Generate(ctx, serviceCIDR, certificatesDir, options)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
