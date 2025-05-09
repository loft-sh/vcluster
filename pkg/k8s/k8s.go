package k8s

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	vclusterconfig "github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/etcd"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/util/commandwriter"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
)

func StartK8S(
	ctx context.Context,
	serviceCIDR string,
	apiServer vclusterconfig.DistroContainerEnabled,
	controllerManager vclusterconfig.DistroContainerEnabled,
	scheduler vclusterconfig.DistroContainer,
	vConfig *config.VirtualClusterConfig,
) error {
	errChan := make(chan error, 1)

	// start the backing store
	etcdEndpoints, etcdCertificates, err := StartBackingStore(ctx, vConfig)
	if err != nil {
		return err
	}

	// start api server first
	if apiServer.Enabled {
		go func() {
			// build flags
			issuer := "https://kubernetes.default.svc.cluster.local"

			if vConfig.Networking.Advanced.ClusterDomain != "" {
				issuer = "https://kubernetes.default.svc." + vConfig.Networking.Advanced.ClusterDomain
			}

			args := []string{}
			if len(apiServer.Command) > 0 {
				args = append(args, apiServer.Command...)
			} else {
				args = append(args, "/binaries/kube-apiserver")
				args = append(args, "--advertise-address=127.0.0.1")
				args = append(args, "--service-cluster-ip-range="+serviceCIDR)
				args = append(args, "--bind-address=127.0.0.1")
				args = append(args, "--allow-privileged=true")
				args = append(args, "--authorization-mode=RBAC")
				args = append(args, "--client-ca-file="+vConfig.VirtualClusterKubeConfig().ClientCACert)
				args = append(args, "--enable-bootstrap-token-auth=true")
				args = append(args, "--etcd-servers="+etcdEndpoints)
				if etcdCertificates != nil {
					args = append(args, "--etcd-cafile="+etcdCertificates.CaCert)
					args = append(args, "--etcd-certfile="+etcdCertificates.ServerCert)
					args = append(args, "--etcd-keyfile="+etcdCertificates.ServerKey)
				}
				args = append(args, "--proxy-client-cert-file=/data/pki/front-proxy-client.crt")
				args = append(args, "--proxy-client-key-file=/data/pki/front-proxy-client.key")
				args = append(args, "--requestheader-allowed-names=front-proxy-client")
				args = append(args, "--requestheader-client-ca-file="+vConfig.VirtualClusterKubeConfig().RequestHeaderCACert)
				args = append(args, "--requestheader-extra-headers-prefix=X-Remote-Extra-")
				args = append(args, "--requestheader-group-headers=X-Remote-Group")
				args = append(args, "--requestheader-username-headers=X-Remote-User")
				args = append(args, "--secure-port=6443")
				args = append(args, "--service-account-issuer="+issuer)
				args = append(args, "--service-account-key-file=/data/pki/sa.pub")
				args = append(args, "--service-account-signing-key-file=/data/pki/sa.key")
				args = append(args, "--tls-cert-file=/data/pki/apiserver.crt")
				args = append(args, "--tls-private-key-file=/data/pki/apiserver.key")
				args = append(args, "--endpoint-reconciler-type=none")
				args = append(args, "--profiling=false")
			}

			// add extra args
			args = append(args, apiServer.ExtraArgs...)

			// wait until etcd is up and running
			err := etcd.WaitForEtcd(ctx, etcdCertificates, etcdEndpoints)
			if err != nil {
				errChan <- err
				return
			}

			// now start the api server
			errChan <- RunCommand(ctx, args, "apiserver")
		}()
	}

	// wait for api server to be up as otherwise controller and scheduler might fail
	err = waitForAPI(ctx)
	if err != nil {
		return fmt.Errorf("waited until timeout for the api to be up: %w", err)
	}

	// start controller command
	if controllerManager.Enabled {
		go func() {
			// build flags
			args := []string{}
			if len(controllerManager.Command) > 0 {
				args = append(args, controllerManager.Command...)
			} else {
				args = append(args, "/binaries/kube-controller-manager")
				args = append(args, "--service-cluster-ip-range="+serviceCIDR)
				args = append(args, "--authentication-kubeconfig=/data/pki/controller-manager.conf")
				args = append(args, "--authorization-kubeconfig=/data/pki/controller-manager.conf")
				args = append(args, "--bind-address=127.0.0.1")
				args = append(args, "--client-ca-file="+vConfig.VirtualClusterKubeConfig().ClientCACert)
				args = append(args, "--cluster-name=kubernetes")
				args = append(args, "--cluster-signing-cert-file="+vConfig.VirtualClusterKubeConfig().ServerCACert)
				args = append(args, "--cluster-signing-key-file="+vConfig.VirtualClusterKubeConfig().ServerCAKey)
				args = append(args, "--horizontal-pod-autoscaler-sync-period=60s")
				args = append(args, "--kubeconfig=/data/pki/controller-manager.conf")
				args = append(args, "--node-monitor-grace-period=180s")
				args = append(args, "--node-monitor-period=30s")
				args = append(args, "--pvclaimbinder-sync-period=60s")
				args = append(args, "--requestheader-client-ca-file="+vConfig.VirtualClusterKubeConfig().RequestHeaderCACert)
				args = append(args, "--root-ca-file="+vConfig.VirtualClusterKubeConfig().ServerCACert)
				args = append(args, "--service-account-private-key-file=/data/pki/sa.key")
				args = append(args, "--use-service-account-credentials=true")
				if vConfig.ControlPlane.StatefulSet.HighAvailability.Replicas > 1 {
					args = append(args, "--leader-elect=true")
				} else {
					args = append(args, "--leader-elect=false")
				}
				if vConfig.ControlPlane.Advanced.VirtualScheduler.Enabled {
					args = append(args, "--controllers=*,-nodeipam,-persistentvolume-binder,-attachdetach,-persistentvolume-expander,-cloud-node-lifecycle,-ttl")
					args = append(args, "--node-monitor-grace-period=1h")
					args = append(args, "--node-monitor-period=1h")
				} else {
					args = append(args, "--controllers=*,-nodeipam,-nodelifecycle,-persistentvolume-binder,-attachdetach,-persistentvolume-expander,-cloud-node-lifecycle,-ttl")
				}
			}

			// add extra args
			args = append(args, controllerManager.ExtraArgs...)
			errChan <- RunCommand(ctx, args, "controller-manager")
		}()
	}

	// start scheduler command
	if vConfig.ControlPlane.Advanced.VirtualScheduler.Enabled {
		go func() {
			// build flags
			args := []string{}
			if len(scheduler.Command) > 0 {
				args = append(args, scheduler.Command...)
			} else {
				args = append(args, "/binaries/kube-scheduler")
				args = append(args, "--authentication-kubeconfig=/data/pki/scheduler.conf")
				args = append(args, "--authorization-kubeconfig=/data/pki/scheduler.conf")
				args = append(args, "--bind-address=127.0.0.1")
				args = append(args, "--kubeconfig=/data/pki/scheduler.conf")
				if vConfig.ControlPlane.StatefulSet.HighAvailability.Replicas > 1 {
					args = append(args, "--leader-elect=true")
				} else {
					args = append(args, "--leader-elect=false")
				}
			}

			// add extra args
			args = append(args, scheduler.ExtraArgs...)
			errChan <- RunCommand(ctx, args, "scheduler")
		}()
	}

	return <-errChan
}

func StartKine(ctx context.Context, dataSource, listenAddress string, certificates *etcd.Certificates) {
	// start embedded mode
	go func() {
		args := []string{}
		args = append(args, "/usr/local/bin/kine")
		args = append(args, "--endpoint="+dataSource)
		if certificates != nil {
			if certificates.CaCert != "" {
				args = append(args, "--ca-file="+certificates.CaCert)
			}
			if certificates.ServerKey != "" {
				args = append(args, "--key-file="+certificates.ServerKey)
			}
			if certificates.ServerCert != "" {
				args = append(args, "--cert-file="+certificates.ServerCert)
			}
		}
		args = append(args, "--metrics-bind-address=0")
		args = append(args, "--listen-address="+listenAddress)

		// now start kine
		err := RunCommand(ctx, args, "kine")
		if err != nil {
			klog.Fatal("could not run kine", err)
		}
	}()
}

func StartBackingStore(ctx context.Context, vConfig *config.VirtualClusterConfig) (string, *etcd.Certificates, error) {
	// start kine embedded or external
	var (
		etcdEndpoints    string
		etcdCertificates *etcd.Certificates
	)
	if vConfig.EmbeddedDatabase() {
		dataSource := vConfig.ControlPlane.BackingStore.Database.Embedded.DataSource
		if dataSource == "" {
			dataSource = fmt.Sprintf("sqlite://%s?_journal=WAL&cache=shared&_busy_timeout=30000&_txlock=immediate", constants.K8sSqliteDatabase)
		}

		StartKine(ctx, dataSource, constants.K8sKineEndpoint, &etcd.Certificates{
			CaCert:     vConfig.ControlPlane.BackingStore.Database.Embedded.CaFile,
			ServerKey:  vConfig.ControlPlane.BackingStore.Database.Embedded.KeyFile,
			ServerCert: vConfig.ControlPlane.BackingStore.Database.Embedded.CertFile,
		})

		etcdEndpoints = constants.K8sKineEndpoint
	} else if vConfig.ControlPlane.BackingStore.Database.External.Enabled {
		// we check for an empty datasource string here because the platform connect
		// process may overwrite an empty datasource string with a platform supplied
		// one. At this point the platform connect process is assumed to have happened.
		if vConfig.ControlPlane.BackingStore.Database.External.DataSource == "" {
			return "", nil, fmt.Errorf("external datasource cannot be empty if external database is enabled")
		}

		// call out to the pro code
		var err error
		etcdEndpoints, etcdCertificates, err = pro.ConfigureExternalDatabase(ctx, constants.K8sKineEndpoint, vConfig, true)
		if err != nil {
			return "", nil, fmt.Errorf("configure external database: %w", err)
		}
	} else if vConfig.BackingStoreType() == vclusterconfig.StoreTypeExternalEtcd {
		etcdCertificates = &etcd.Certificates{
			CaCert:     vConfig.ControlPlane.BackingStore.Etcd.External.TLS.CaFile,
			ServerCert: vConfig.ControlPlane.BackingStore.Etcd.External.TLS.CertFile,
			ServerKey:  vConfig.ControlPlane.BackingStore.Etcd.External.TLS.KeyFile,
		}
		etcdEndpoints = "https://" + strings.TrimPrefix(vConfig.ControlPlane.BackingStore.Etcd.External.Endpoint, "https://")
	} else {
		// embedded or deployed etcd
		etcdCertificates = &etcd.Certificates{
			CaCert:     "/data/pki/etcd/ca.crt",
			ServerCert: "/data/pki/apiserver-etcd-client.crt",
			ServerKey:  "/data/pki/apiserver-etcd-client.key",
		}

		if vConfig.ControlPlane.BackingStore.Etcd.Embedded.Enabled {
			etcdEndpoints = "https://127.0.0.1:2379"
		} else if vConfig.ControlPlane.BackingStore.Etcd.Deploy.Service.Enabled {
			etcdEndpoints = "https://" + vConfig.Name + "-etcd:2379"
		} else {
			etcdEndpoints = "https://" + vConfig.Name + "-etcd-headless:2379"
		}
	}

	return etcdEndpoints, etcdCertificates, nil
}

func RunCommand(ctx context.Context, command []string, component string) error {
	writer, err := commandwriter.NewCommandWriter(component, false)
	if err != nil {
		return err
	}
	defer writer.Writer()

	// start the command
	klog.InfoS("Starting "+component, "args", strings.Join(command, " "))
	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
	cmd.Stdout = writer.Writer()
	cmd.Stderr = writer.Writer()
	cmd.Cancel = func() error {
		err := cmd.Process.Signal(os.Interrupt)
		if err != nil {
			return fmt.Errorf("signal %s: %w", command[0], err)
		}

		state, err := cmd.Process.Wait()
		if err == nil && state.Pid() > 0 {
			time.Sleep(2 * time.Second)
		}

		err = cmd.Process.Kill()
		if err != nil {
			return fmt.Errorf("kill %s: %w", command[0], err)
		}

		return nil
	}
	err = cmd.Run()

	// make sure we wait for scanner to be done
	writer.CloseAndWait(ctx, err)
	return fmt.Errorf("error running command %s: %w", command[0], err)
}

// waits for the api to be up, ignoring certs and calling it
// localhost
func waitForAPI(ctx context.Context) error {
	client := &http.Client{
		Timeout: 2 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	// sometimes the etcd pod takes a very long time to be ready,
	// we might want to fine tune how long we wait later
	var lastErr error
	err := wait.PollUntilContextTimeout(ctx, time.Second*2, time.Minute*5, true, func(ctx context.Context) (done bool, err error) {
		// build the request
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://127.0.0.1:6443/readyz", nil)
		if err != nil {
			lastErr = err
			klog.Errorf("could not create the request to wait for the api: %s", err.Error())
			return false, nil
		}

		// do the request
		response, err := client.Do(req)
		if err != nil {
			lastErr = err
			klog.Info("error while targeting the api on localhost, this is expected during the vCluster creation, will retry after 2 seconds:", err)
			return false, nil
		}

		// check if we got a ok response status code
		if response.StatusCode != http.StatusOK {
			bytes, _ := io.ReadAll(response.Body)
			klog.FromContext(ctx).Info("api server not ready yet", "reason", string(bytes))
			lastErr = fmt.Errorf("api server not ready yet, reason: %s", string(bytes))
			return false, nil
		}

		return true, nil
	})
	if err != nil {
		return fmt.Errorf("error waiting for API server: %w", lastErr)
	}

	return nil
}
