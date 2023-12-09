package setup

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/loft-sh/vcluster/pkg/certs"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/k0s"
	"github.com/loft-sh/vcluster/pkg/k3s"
	"github.com/loft-sh/vcluster/pkg/setup/options"
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
	// check if we should create certificates
	certificatesDir := ""
	if strings.HasPrefix(options.ServerCaCert, "/pki/") {
		certificatesDir = "/pki"
	}

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
			certificatesDir,
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
	certificatesDir string,
) error {
	var err error
	distro := constants.GetVClusterDistro()

	// if k0s secret was found ensure it contains service CIDR range
	var serviceCIDR string
	if distro == constants.K0SDistro {
		klog.Info("k0s config secret detected, syncer will ensure that it contains service CIDR")
		serviceCIDR, err = servicecidr.EnsureServiceCIDRInK0sSecret(ctx, workspaceNamespaceClient, currentNamespaceClient, workspaceNamespace, currentNamespace, vClusterName)
		if err != nil {
			return err
		}
	} else {
		// in all other cases ensure that a valid CIDR range is in the designated ConfigMap
		serviceCIDR, err = servicecidr.EnsureServiceCIDRConfigmap(ctx, workspaceNamespaceClient, currentNamespaceClient, workspaceNamespace, currentNamespace, vClusterName)
		if err != nil {
			return fmt.Errorf("failed to ensure that service CIDR range is written into the expected location: %w", err)
		}
	}

	// check if k3s
	if distro == constants.K0SDistro {
		// start k0s
		go func() {
			// we need to run this with the parent ctx as otherwise this context will be cancelled by the wait
			// loop in Initialize
			err := k0s.StartK0S(parentCtx)
			if err != nil {
				klog.Fatalf("Error running k0s: %v", err)
			}
		}()
	} else if distro == constants.K3SDistro {
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
	} else if certificatesDir != "" {
		// generate k8s certificates
		err = GenerateK8sCerts(ctx, currentNamespaceClient, vClusterName, currentNamespace, serviceCIDR, certificatesDir, options.ClusterDomain, options.EtcdReplicas, options.EtcdEmbedded)
		if err != nil {
			return err
		}
	}

	return nil
}

func GenerateK8sCerts(ctx context.Context, currentNamespaceClient kubernetes.Interface, vClusterName, currentNamespace, serviceCIDR, certificatesDir, clusterDomain string, etcdReplicaCount int, etcdEmbedded bool) error {
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
	for i := 0; i < etcdReplicaCount; i++ {
		if etcdEmbedded {
			// this is for embedded etcd
			hostname := vClusterName + "-" + strconv.Itoa(i)
			etcdSans = append(etcdSans, hostname, hostname+"."+vClusterName+"-headless", hostname+"."+vClusterName+"-headless"+"."+currentNamespace)
		} else {
			// this is for external etcd
			etcdHostname := etcdService + "-" + strconv.Itoa(i)
			etcdSans = append(etcdSans, etcdHostname, etcdHostname+"."+etcdService+"-headless", etcdHostname+"."+etcdService+"-headless"+"."+currentNamespace)
		}
	}

	// generate certificates
	err := certs.EnsureCerts(ctx, serviceCIDR, currentNamespace, currentNamespaceClient, vClusterName, certificatesDir, clusterDomain, etcdSans, "-api")
	if err != nil {
		return fmt.Errorf("ensure certs: %w", err)
	}

	return nil
}
