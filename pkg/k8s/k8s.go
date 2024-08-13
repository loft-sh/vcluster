package k8s

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"

	vclusterconfig "github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/etcd"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/util/commandwriter"
	"golang.org/x/sync/errgroup"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
)

const KineEndpoint = "unix:///data/kine.sock"

func StartK8S(
	ctx context.Context,
	serviceCIDR string,
	apiServer vclusterconfig.DistroContainerEnabled,
	controllerManager vclusterconfig.DistroContainerEnabled,
	scheduler vclusterconfig.DistroContainer,
	vConfig *config.VirtualClusterConfig,
) error {
	eg := &errgroup.Group{}

	// start kine embedded or external
	var (
		etcdEndpoints    string
		etcdCertificates *etcd.Certificates
	)
	if vConfig.EmbeddedDatabase() {
		dataSource := vConfig.ControlPlane.BackingStore.Database.External.DataSource
		if dataSource == "" {
			dataSource = "sqlite:///data/state.db?_journal=WAL&cache=shared&_busy_timeout=30000"
		}

		// start embedded mode
		go func() {
			args := []string{}
			args = append(args, "/usr/local/bin/kine")
			args = append(args, "--endpoint="+dataSource)
			args = append(args, "--ca-file="+vConfig.ControlPlane.BackingStore.Database.External.CaFile)
			args = append(args, "--key-file="+vConfig.ControlPlane.BackingStore.Database.External.KeyFile)
			args = append(args, "--cert-file="+vConfig.ControlPlane.BackingStore.Database.External.CertFile)
			args = append(args, "--metrics-bind-address=0")
			args = append(args, "--listen-address="+KineEndpoint)

			// now start kine
			err := RunCommand(ctx, args, "kine")
			if err != nil {
				klog.Fatal("could not run kine", err)
			}
		}()

		etcdEndpoints = KineEndpoint
	} else if vConfig.ControlPlane.BackingStore.Database.External.Enabled {
		// we check for an empty datasource string here because the platform connect
		// process may overwrite an empty datasource string with a platform supplied
		// one. At this point the platform connect process is assumed to have happened.
		if vConfig.ControlPlane.BackingStore.Database.External.DataSource == "" {
			return fmt.Errorf("external datasource cannot be empty if external database is enabled")
		}

		// call out to the pro code
		var err error
		etcdEndpoints, etcdCertificates, err = pro.ConfigureExternalDatabase(ctx, vConfig)
		if err != nil {
			return fmt.Errorf("configure external database: %w", err)
		}
	} else {
		// embedded or deployed etcd
		etcdCertificates = &etcd.Certificates{
			CaCert:     "/data/pki/etcd/ca.crt",
			ServerCert: "/data/pki/apiserver-etcd-client.crt",
			ServerKey:  "/data/pki/apiserver-etcd-client.key",
		}

		if vConfig.ControlPlane.BackingStore.Etcd.Embedded.Enabled {
			etcdEndpoints = "https://127.0.0.1:2379"
		} else {
			etcdEndpoints = "https://" + vConfig.Name + "-etcd:2379"
		}
	}

	// start api server first
	if apiServer.Enabled {
		eg.Go(func() error {
			// build flags
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
				args = append(args, "--service-account-issuer=https://kubernetes.default.svc.cluster.local")
				args = append(args, "--service-account-key-file=/data/pki/sa.pub")
				args = append(args, "--service-account-signing-key-file=/data/pki/sa.key")
				args = append(args, "--tls-cert-file=/data/pki/apiserver.crt")
				args = append(args, "--tls-private-key-file=/data/pki/apiserver.key")
				args = append(args, "--watch-cache=false")
				args = append(args, "--endpoint-reconciler-type=none")
			}

			// add extra args
			args = append(args, apiServer.ExtraArgs...)

			// wait until etcd is up and running
			_, err := etcd.WaitForEtcdClient(ctx, etcdCertificates, etcdEndpoints)
			if err != nil {
				return err
			}

			// now start the api server
			return RunCommand(ctx, args, "apiserver")
		})
	}

	// wait for api server to be up as otherwise controller and scheduler might fail
	err := waitForAPI(ctx)
	if err != nil {
		return fmt.Errorf("waited until timeout for the api to be up: %w", err)
	}

	// start controller command
	if controllerManager.Enabled {
		eg.Go(func() error {
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
			return RunCommand(ctx, args, "controller-manager")
		})
	}

	// start scheduler command
	if vConfig.ControlPlane.Advanced.VirtualScheduler.Enabled {
		eg.Go(func() error {
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
			return RunCommand(ctx, args, "scheduler")
		})
	}

	// regular stop case, will return as soon as a component returns an error.
	// we don't expect the components to stop by themselves since they're supposed
	// to run until killed or until they fail
	err = eg.Wait()
	if err == nil || err.Error() == "signal: killed" {
		return nil
	}
	return err
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
	err = cmd.Run()

	// make sure we wait for scanner to be done
	writer.CloseAndWait(ctx, err)
	return err
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
