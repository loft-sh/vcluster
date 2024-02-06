package setup

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"time"

	"github.com/loft-sh/vcluster/pkg/certs"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/k0s"
	"github.com/loft-sh/vcluster/pkg/k3s"
	"github.com/loft-sh/vcluster/pkg/k8s"
	"github.com/loft-sh/vcluster/pkg/options"
	"github.com/loft-sh/vcluster/pkg/specialservices"
	"github.com/loft-sh/vcluster/pkg/telemetry"
	"github.com/loft-sh/vcluster/pkg/util/servicecidr"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

// Initialize creates the required secrets and configmaps for the control plane to start
func Initialize(
	ctx context.Context,
	workspaceNamespaceClient,
	currentNamespaceClient kubernetes.Interface,
	workspaceNamespace,
	currentNamespace,
	vClusterName string,
	options *options.VirtualClusterOptions,
) error {
	// Ensure that service CIDR range is written into the expected location
	err := wait.PollUntilContextTimeout(ctx, 5*time.Second, 2*time.Minute, true, func(waitCtx context.Context) (bool, error) {
		err := initialize(
			waitCtx,
			ctx,
			workspaceNamespaceClient,
			currentNamespaceClient,
			workspaceNamespace,
			currentNamespace,
			vClusterName,
			options,
		)
		if err != nil {
			klog.Errorf("error initializing service cidr, certs and token: %v", err)
			return false, nil
		}

		return true, nil
	})
	if err != nil {
		return err
	}

	specialservices.SetDefault()
	telemetry.Collector.RecordStart(ctx)
	return nil
}

// initialize creates the required secrets and configmaps for the control plane to start
func initialize(
	ctx context.Context,
	parentCtx context.Context,
	workspaceNamespaceClient,
	currentNamespaceClient kubernetes.Interface,
	workspaceNamespace,
	currentNamespace,
	vClusterName string,
	options *options.VirtualClusterOptions,
) error {
	distro := constants.GetVClusterDistro()

	// retrieve service cidr
	var serviceCIDR string
	if distro != constants.K0SDistro {
		var warning string
		serviceCIDR, warning = servicecidr.GetServiceCIDR(ctx, currentNamespaceClient, currentNamespace)
		if warning != "" {
			klog.Warning(warning)
		}
	}

	// check what distro are we running
	switch distro {
	case constants.K0SDistro:
		// ensure service cidr
		_, err := servicecidr.EnsureServiceCIDRInK0sSecret(ctx, workspaceNamespaceClient, currentNamespaceClient, workspaceNamespace, currentNamespace, vClusterName)
		if err != nil {
			return err
		}

		// start k0s
		go func() {
			// we need to run this with the parent ctx as otherwise this context will be cancelled by the wait
			// loop in Initialize
			err := k0s.StartK0S(parentCtx)
			if err != nil {
				klog.Fatalf("Error running k0s: %v", err)
			}
		}()
	case constants.K3SDistro:
		// its k3s, let's create the token secret
		k3sToken, err := k3s.EnsureK3SToken(ctx, currentNamespaceClient, currentNamespace, vClusterName)
		if err != nil {
			return err
		}

		// start k3s
		go func() {
			// we need to run this with the parent ctx as otherwise this context will be cancelled by the wait
			// loop in Initialize
			err := k3s.StartK3S(parentCtx, serviceCIDR, k3sToken)
			if err != nil {
				klog.Fatalf("Error running k3s: %v", err)
			}
		}()
	case constants.K8SDistro, constants.EKSDistro:
		// try to generate k8s certificates
		certificatesDir := filepath.Dir(options.ServerCaCert)
		if certificatesDir == "/pki" {
			err := GenerateK8sCerts(ctx, currentNamespaceClient, vClusterName, currentNamespace, serviceCIDR, certificatesDir, options.ClusterDomain)
			if err != nil {
				return err
			}
		}

		// start k8s
		go func() {
			// we need to run this with the parent ctx as otherwise this context will be cancelled by the wait
			// loop in Initialize
			err := k8s.StartK8S(parentCtx, serviceCIDR)
			if err != nil {
				klog.Fatalf("Error running k8s: %v", err)
			}
		}()
	case constants.Unknown:
		certificatesDir := filepath.Dir(options.ServerCaCert)
		if certificatesDir == "/pki" {
			// generate k8s certificates
			err := GenerateK8sCerts(ctx, currentNamespaceClient, vClusterName, currentNamespace, serviceCIDR, certificatesDir, options.ClusterDomain)
			if err != nil {
				return err
			}
		}
	}

	klog.Info("finished running initialize")
	return nil
}

func GenerateK8sCerts(ctx context.Context, currentNamespaceClient kubernetes.Interface, vClusterName, currentNamespace, serviceCIDR, certificatesDir, clusterDomain string) error {
	// generate etcd server and peer sans
	etcdService := vClusterName + "-etcd"
	etcdSans := []string{
		"localhost",
		etcdService,
		etcdService + "." + currentNamespace,
		etcdService + "." + currentNamespace + ".svc",
		"*." + etcdService + "-headless",
		"*." + etcdService + "-headless" + "." + currentNamespace,
	}

	//expect up to 20 etcd members, number could be lower since more
	//than 5 is generally a bad idea
	for i := 0; i < 20; i++ {
		// this is for embedded etcd
		hostname := vClusterName + "-" + strconv.Itoa(i)
		etcdSans = append(etcdSans, hostname, hostname+"."+vClusterName+"-headless", hostname+"."+vClusterName+"-headless"+"."+currentNamespace)
		// this is for external etcd
		etcdHostname := etcdService + "-" + strconv.Itoa(i)
		etcdSans = append(etcdSans, etcdHostname, etcdHostname+"."+etcdService+"-headless", etcdHostname+"."+etcdService+"-headless"+"."+currentNamespace)
	}

	// generate certificates
	err := certs.EnsureCerts(ctx, serviceCIDR, currentNamespace, currentNamespaceClient, vClusterName, certificatesDir, clusterDomain, etcdSans)
	if err != nil {
		return fmt.Errorf("ensure certs: %w", err)
	}

	return nil
}
