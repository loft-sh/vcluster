package setup

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/loft-sh/vcluster/pkg/certs"
	"github.com/loft-sh/vcluster/pkg/k3s"
	"github.com/loft-sh/vcluster/pkg/setup/options"
	"github.com/loft-sh/vcluster/pkg/specialservices"
	"github.com/loft-sh/vcluster/pkg/util/servicecidr"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	specialservices.SetDefault(options)

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
	// check if k0s config Secret exists
	_, err := currentNamespaceClient.CoreV1().Secrets(currentNamespace).Get(ctx, servicecidr.GetK0sSecretName(vClusterName), metav1.GetOptions{})
	if err != nil && !kerrors.IsNotFound(err) {
		return err
	}
	isK0s := err == nil

	// if k0s secret was found ensure it contains service CIDR range
	var serviceCIDR string
	if isK0s {
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
	if !isK0s && certificatesDir != "/pki" {
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
		options.IsK8sDistro = true
		err = GenerateK8sCerts(ctx, currentNamespaceClient, vClusterName, currentNamespace, serviceCIDR, certificatesDir, options.ClusterDomain)
		if err != nil {
			return err
		}
	}

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

	// generate certificates
	err := certs.EnsureCerts(ctx, serviceCIDR, currentNamespace, currentNamespaceClient, vClusterName, certificatesDir, clusterDomain, etcdSans)
	if err != nil {
		return fmt.Errorf("ensure certs: %w", err)
	}

	return nil
}
