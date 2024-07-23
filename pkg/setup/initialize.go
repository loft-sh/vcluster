package setup

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	vclusterconfig "github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/certs"
	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/k0s"
	"github.com/loft-sh/vcluster/pkg/k3s"
	"github.com/loft-sh/vcluster/pkg/k8s"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/specialservices"
	"github.com/loft-sh/vcluster/pkg/telemetry"
	"github.com/loft-sh/vcluster/pkg/util/servicecidr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

// Initialize creates the required secrets and configmaps for the control plane to start
func Initialize(ctx context.Context, options *config.VirtualClusterConfig) error {
	// Ensure that service CIDR range is written into the expected location
	err := wait.PollUntilContextTimeout(ctx, 5*time.Second, 2*time.Minute, true, func(waitCtx context.Context) (bool, error) {
		err := initialize(
			waitCtx,
			ctx,
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

	specialservices.Default = pro.InitDNSServiceSyncing(options)
	telemetry.CollectorControlPlane.RecordStart(ctx, options)
	return nil
}

// initialize creates the required secrets and configmaps for the control plane to start
func initialize(ctx context.Context, parentCtx context.Context, options *config.VirtualClusterConfig) error {
	distro := options.Distro()

	// migrate from
	migrateFrom := ""
	if options.ControlPlane.BackingStore.Etcd.Embedded.Enabled && options.ControlPlane.BackingStore.Etcd.Embedded.MigrateFromDeployedEtcd {
		migrateFrom = "https://" + options.Name + "-etcd:2379"
	}

	// retrieve service cidr
	serviceCIDR := options.ServiceCIDR
	if serviceCIDR == "" {
		var warning string
		serviceCIDR, warning = servicecidr.GetServiceCIDR(ctx, options.WorkloadClient, options.WorkloadNamespace)
		if warning != "" {
			klog.Warning(warning)
		}
	}

	// check what distro are we running
	switch distro {
	case vclusterconfig.K0SDistro:
		// only return the first cidr, because k0s don't accept coma separated ones
		serviceCIDR = strings.Split(serviceCIDR, ",")[0]

		// ensure service cidr
		err := k0s.WriteK0sConfig(serviceCIDR, options)
		if err != nil {
			return err
		}

		// create certificates if they are not there yet
		certificatesDir := "/data/k0s/pki"
		err = GenerateCerts(ctx, options.ControlPlaneClient, options.Name, options.ControlPlaneNamespace, serviceCIDR, certificatesDir, options)
		if err != nil {
			return err
		}

		// should start embedded etcd?
		if options.ControlPlane.BackingStore.Etcd.Embedded.Enabled {
			err = pro.StartEmbeddedEtcd(
				parentCtx,
				options.Name,
				options.ControlPlaneNamespace,
				certificatesDir,
				int(options.ControlPlane.StatefulSet.HighAvailability.Replicas),
				migrateFrom,
			)
			if err != nil {
				return fmt.Errorf("start embedded etcd: %w", err)
			}
		}

		// start k0s
		parentCtxWithCancel, cancel := context.WithCancel(parentCtx)
		go func() {
			// we need to run this with the parent ctx as otherwise this context will be cancelled by the wait
			// loop in Initialize
			err := k0s.StartK0S(parentCtxWithCancel, cancel, options)
			if err != nil {
				klog.Fatalf("Error running k0s: %v", err)
			}
		}()

		// try to update the certs secret with the k0s certificates
		err = UpdateSecretWithK0sCerts(ctx, options.ControlPlaneClient, options.ControlPlaneNamespace, options.Name)
		if err != nil {
			cancel()
			return err
		}
	case vclusterconfig.K3SDistro:
		// its k3s, let's create the token secret
		k3sToken, err := k3s.EnsureK3SToken(ctx, options.ControlPlaneClient, options.ControlPlaneNamespace, options.Name, options)
		if err != nil {
			return err
		}

		// generate etcd certificates
		certificatesDir := "/data/pki"
		err = GenerateCerts(ctx, options.ControlPlaneClient, options.Name, options.ControlPlaneNamespace, serviceCIDR, certificatesDir, options)
		if err != nil {
			return err
		}

		// should start embedded etcd?
		if options.ControlPlane.BackingStore.Etcd.Embedded.Enabled {
			// we need to run this with the parent ctx as otherwise this context
			// will be cancelled by the wait loop in Initialize
			err = pro.StartEmbeddedEtcd(
				parentCtx,
				options.Name,
				options.ControlPlaneNamespace,
				certificatesDir,
				int(options.ControlPlane.StatefulSet.HighAvailability.Replicas),
				migrateFrom,
			)
			if err != nil {
				return fmt.Errorf("start embedded etcd: %w", err)
			}
		}

		// start k3s
		go func() {
			// we need to run this with the parent ctx as otherwise this context will be cancelled by the wait
			// loop in Initialize
			err := k3s.StartK3S(parentCtx, options, serviceCIDR, k3sToken)
			if err != nil {
				klog.Fatalf("Error running k3s: %v", err)
			}
		}()
	case vclusterconfig.K8SDistro:
		// try to generate k8s certificates
		certificatesDir := filepath.Dir(options.VirtualClusterKubeConfig().ServerCACert)
		if certificatesDir == "/data/pki" {
			err := GenerateCerts(ctx, options.ControlPlaneClient, options.Name, options.ControlPlaneNamespace, serviceCIDR, certificatesDir, options)
			if err != nil {
				return err
			}
		}

		// should start embedded etcd?
		if options.ControlPlane.BackingStore.Etcd.Embedded.Enabled {
			// start embedded etcd
			err := pro.StartEmbeddedEtcd(
				parentCtx,
				options.Name,
				options.ControlPlaneNamespace,
				certificatesDir,
				int(options.ControlPlane.StatefulSet.HighAvailability.Replicas),
				migrateFrom,
			)
			if err != nil {
				return fmt.Errorf("start embedded etcd: %w", err)
			}
		}

		// start k8s
		go func() {
			// we need to run this with the parent ctx as otherwise this context will be cancelled by the wait
			// loop in Initialize
			var err error
			if distro == vclusterconfig.K8SDistro {
				err = k8s.StartK8S(
					parentCtx,
					serviceCIDR,
					options.ControlPlane.Distro.K8S.APIServer,
					options.ControlPlane.Distro.K8S.ControllerManager,
					options.ControlPlane.Distro.K8S.Scheduler,
					options,
				)
			}
			if err != nil {
				klog.Fatalf("Error running k8s: %v", err)
			}
		}()
	case vclusterconfig.Unknown:
		certificatesDir := filepath.Dir(options.VirtualClusterKubeConfig().ServerCACert)
		if certificatesDir == "/data/pki" {
			// generate k8s certificates
			err := GenerateCerts(ctx, options.ControlPlaneClient, options.Name, options.ControlPlaneNamespace, serviceCIDR, certificatesDir, options)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func GenerateCerts(ctx context.Context, currentNamespaceClient kubernetes.Interface, vClusterName, currentNamespace, serviceCIDR, certificatesDir string, options *config.VirtualClusterConfig) error {
	clusterDomain := options.Networking.Advanced.ClusterDomain
	// generate etcd server and peer sans
	etcdService := vClusterName + "-etcd"
	etcdSans := []string{
		"localhost",
		etcdService,
		etcdService + "." + currentNamespace,
		etcdService + "." + currentNamespace + ".svc",
	}

	// add wildcard
	for _, service := range []string{vClusterName, etcdService} {
		etcdSans = append(
			etcdSans,
			"*."+service+"-headless",
			"*."+service+"-headless"+"."+currentNamespace,
			"*."+service+"-headless"+"."+currentNamespace+".svc",
			"*."+service+"-headless"+"."+currentNamespace+".svc."+clusterDomain,
		)
	}

	// expect up to 20 etcd members, number could be lower since more
	// than 5 is generally a bad idea
	for i := range 20 {
		// this is for embedded etcd
		hostname := vClusterName + "-" + strconv.Itoa(i)
		etcdSans = append(etcdSans, hostname, hostname+"."+vClusterName+"-headless", hostname+"."+vClusterName+"-headless"+"."+currentNamespace)

		// this is for external etcd
		etcdHostname := etcdService + "-" + strconv.Itoa(i)
		etcdSans = append(etcdSans, etcdHostname, etcdHostname+"."+etcdService+"-headless", etcdHostname+"."+etcdService+"-headless"+"."+currentNamespace)
	}

	// generate certificates
	err := certs.EnsureCerts(ctx, serviceCIDR, currentNamespace, currentNamespaceClient, vClusterName, certificatesDir, etcdSans, options)
	if err != nil {
		return fmt.Errorf("ensure certs: %w", err)
	}

	return nil
}

func UpdateSecretWithK0sCerts(
	ctx context.Context,
	currentNamespaceClient kubernetes.Interface,
	currentNamespace, vClusterName string,
) error {
	// wait for k0s to generate the secrets for us
	files, err := waitForK0sFiles(ctx, "/data/k0s/pki")
	if err != nil {
		return err
	}

	// retrieve cert secret
	secret, err := currentNamespaceClient.CoreV1().Secrets(currentNamespace).Get(ctx, vClusterName+"-certs", metav1.GetOptions{})
	if err != nil {
		return err
	} else if secret.Data == nil {
		return fmt.Errorf("error while trying to update the secret, data was empty, will try to fetch it again")
	}

	// check if the secret contains the k0s files now, which would mean somebody was faster than we were
	if secretContainsK0sCerts(secret) {
		if secretIsUpToDate(secret, files) {
			return nil
		}

		return fmt.Errorf("error while trying to update the secret, it was already updated, will try to fetch it again")
	}

	// update the secret to include the k0s certs
	for fileName, content := range files {
		secret.Data[fileName] = content
	}

	// if any error we will retry from the poll loop
	_, err = currentNamespaceClient.CoreV1().Secrets(currentNamespace).Update(ctx, secret, metav1.UpdateOptions{})
	return err
}

func waitForK0sFiles(ctx context.Context, certDir string) (map[string][]byte, error) {
	for {
		filesFound := 0
		for file := range certs.K0sFiles {
			_, err := os.ReadFile(filepath.Join(certDir, file))
			if errors.Is(err, fs.ErrNotExist) {
				break
			} else if err != nil {
				return nil, err
			}

			filesFound++
		}
		if filesFound == len(certs.K0sFiles) {
			break
		}

		select {
		case <-ctx.Done():
			return nil, context.DeadlineExceeded
		case <-time.After(time.Second):
		}
	}
	return readK0sFiles(certDir)
}

func readK0sFiles(certDir string) (map[string][]byte, error) {
	files := make(map[string][]byte)
	for file := range certs.K0sFiles {
		b, err := os.ReadFile(filepath.Join(certDir, file))
		if err != nil {
			return nil, err
		}
		files[file] = b
	}

	return files, nil
}

func secretContainsK0sCerts(secret *corev1.Secret) bool {
	if secret.Data == nil {
		return false
	}
	for k := range secret.Data {
		if certs.K0sFiles[k] {
			return true
		}
	}
	return false
}

func secretIsUpToDate(secret *corev1.Secret, files map[string][]byte) bool {
	for fileName, content := range files {
		if !reflect.DeepEqual(secret.Data[fileName], content) {
			return false
		}
	}

	return true
}
