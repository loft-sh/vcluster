package setup

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"time"

	vclusterconfig "github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/certs"
	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/k3s"
	"github.com/loft-sh/vcluster/pkg/k8s"
	"github.com/loft-sh/vcluster/pkg/kubeadm"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/specialservices"
	"github.com/loft-sh/vcluster/pkg/telemetry"
	"github.com/loft-sh/vcluster/pkg/util/servicecidr"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
)

// Initialize creates the required secrets and configmaps for the control plane to start
func Initialize(ctx context.Context, options *config.VirtualClusterConfig) error {
	// Ensure that service CIDR range is written into the expected location
	err := wait.PollUntilContextTimeout(ctx, 5*time.Second, 2*time.Minute, true, func(waitCtx context.Context) (bool, error) {
		err := initialize(waitCtx, options)
		if err != nil {
			klog.Errorf("error initializing service cidr, certs and token: %v", err)
			return false, nil
		}

		return true, nil
	})
	if err != nil {
		return err
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
	serviceCIDR, err := servicecidr.GetServiceCIDR(ctx, &options.Config, options.WorkloadClient, options.WorkloadService, options.WorkloadNamespace)
	if err != nil {
		return fmt.Errorf("failed to get service cidr: %w", err)
	}

	// check what distro are we running
	switch distro {
	case vclusterconfig.K3SDistro:
		// its k3s, let's create the token secret
		k3sToken, err := k3s.EnsureK3SToken(ctx, options.ControlPlaneClient, options.ControlPlaneNamespace, options.Name, options)
		if err != nil {
			return err
		}

		// generate etcd certificates
		certificatesDir := "/data/pki"
		err = GenerateCerts(ctx, serviceCIDR, certificatesDir, options)
		if err != nil {
			return err
		}

		// should start embedded etcd?
		if options.ControlPlane.BackingStore.Etcd.Embedded.Enabled {
			// we need to run this with the parent ctx as otherwise this context
			// will be cancelled by the wait loop in Initialize
			err = pro.StartEmbeddedEtcd(
				context.WithoutCancel(ctx),
				options.Name,
				options.ControlPlaneNamespace,
				options.ControlPlaneClient,
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
		err := k8s.MigrateK3sToK8s(ctx, options.ControlPlaneClient, options.ControlPlaneNamespace, options)
		if err != nil {
			return fmt.Errorf("migrate k3s to k8s: %w", err)
		}

		// try to generate k8s certificates
		certificatesDir := filepath.Dir(options.VirtualClusterKubeConfig().ServerCACert)
		if certificatesDir == constants.PKIDir {
			err := GenerateCerts(ctx, serviceCIDR, certificatesDir, options)
			if err != nil {
				return err
			}
		}

		// should start embedded etcd?
		if options.ControlPlane.BackingStore.Etcd.Embedded.Enabled {
			// start embedded etcd
			err := pro.StartEmbeddedEtcd(
				context.WithoutCancel(ctx),
				options.Name,
				options.ControlPlaneNamespace,
				options.ControlPlaneClient,
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
			err := GenerateCerts(ctx, serviceCIDR, certificatesDir, options)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func GenerateCerts(ctx context.Context, serviceCIDR, certificatesDir string, options *config.VirtualClusterConfig) error {
	clusterDomain := options.Networking.Advanced.ClusterDomain
	currentNamespace := options.ControlPlaneNamespace
	currentNamespaceClient := options.ControlPlaneClient

	// generate etcd server and peer sans
	etcdService := options.Name + "-etcd"
	extraSans := []string{
		"localhost",
		etcdService,
		etcdService + "-headless",
		etcdService + "." + currentNamespace,
		etcdService + "." + currentNamespace + ".svc",
	}

	// add wildcard
	for _, service := range []string{options.Name, etcdService} {
		extraSans = append(
			extraSans,
			"*."+service+"-headless",
			"*."+service+"-headless"+"."+currentNamespace,
			"*."+service+"-headless"+"."+currentNamespace+".svc",
			"*."+service+"-headless"+"."+currentNamespace+".svc."+clusterDomain,
		)
	}

	// expect up to 5 etcd members
	for i := range 5 {
		// this is for embedded etcd
		hostname := options.Name + "-" + strconv.Itoa(i)
		extraSans = append(extraSans, hostname, hostname+"."+options.Name+"-headless", hostname+"."+options.Name+"-headless"+"."+currentNamespace)

		// this is for external etcd
		etcdHostname := etcdService + "-" + strconv.Itoa(i)
		extraSans = append(extraSans, etcdHostname, etcdHostname+"."+etcdService+"-headless", etcdHostname+"."+etcdService+"-headless"+"."+currentNamespace)
	}

	// create kubeadm config
	kubeadmConfig, err := kubeadm.InitKubeadmConfig(options, "", "127.0.0.1:6443", serviceCIDR, certificatesDir, extraSans)
	if err != nil {
		return fmt.Errorf("create kubeadm config: %w", err)
	}

	// generate certificates
	err = certs.EnsureCerts(ctx, currentNamespace, currentNamespaceClient, certificatesDir, options, kubeadmConfig)
	if err != nil {
		return fmt.Errorf("ensure certs: %w", err)
	}

	return nil
}
