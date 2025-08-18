package k8s

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	vclusterconfig "github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/etcd"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/util/command"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
)

const (
	SQLiteParams = "?_journal=WAL&cache=shared&_busy_timeout=30000&_txlock=immediate"
)

func StartK8S(ctx context.Context, serviceCIDR string, vConfig *config.VirtualClusterConfig) error {
	// start the backing store
	etcdEndpoints, etcdCertificates, err := StartBackingStore(ctx, vConfig)
	if err != nil {
		return err
	}

	// start api server first
	apiServer := vConfig.ControlPlane.Distro.K8S.APIServer
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
				args = append(args, constants.K8sAPIServerBinary)
				args = append(args, "--service-cluster-ip-range="+serviceCIDR)
				args = append(args, "--bind-address=127.0.0.1")
				args = append(args, "--allow-privileged=true")
				args = append(args, "--authorization-mode=Node,RBAC")
				args = append(args, "--client-ca-file="+vConfig.VirtualClusterKubeConfig().ClientCACert)
				args = append(args, "--enable-bootstrap-token-auth=true")
				args = append(args, "--etcd-servers="+etcdEndpoints)
				if etcdCertificates != nil {
					args = append(args, "--etcd-cafile="+etcdCertificates.CaCert)
					args = append(args, "--etcd-certfile="+etcdCertificates.ServerCert)
					args = append(args, "--etcd-keyfile="+etcdCertificates.ServerKey)
				}
				args = append(args, "--proxy-client-cert-file="+constants.FrontProxyClientCert)
				args = append(args, "--proxy-client-key-file="+constants.FrontProxyClientKey)
				args = append(args, "--requestheader-allowed-names=front-proxy-client")
				args = append(args, "--requestheader-client-ca-file="+vConfig.VirtualClusterKubeConfig().RequestHeaderCACert)
				args = append(args, "--requestheader-extra-headers-prefix=X-Remote-Extra-")
				args = append(args, "--requestheader-group-headers=X-Remote-Group")
				args = append(args, "--requestheader-username-headers=X-Remote-User")
				args = append(args, "--secure-port=6443")
				args = append(args, "--service-account-issuer="+issuer)
				args = append(args, "--service-account-key-file="+constants.SACert)
				args = append(args, "--service-account-signing-key-file="+constants.SAKey)
				args = append(args, "--tls-cert-file="+constants.APIServerCert)
				args = append(args, "--tls-private-key-file="+constants.APIServerKey)
				args = append(args, "--profiling=false")

				// this is a hack since we want to set this ourselves and k8s does not support setting a custom port for this
				args = append(args, "--advertise-address=127.0.0.1")
				args = append(args, "--endpoint-reconciler-type=none")

				if vConfig.PrivateNodes.Enabled {
					args = append(args, "--kubelet-client-certificate="+constants.APIServerKubeletClientCert)
					args = append(args, "--kubelet-client-key="+constants.APIServerKubeletClientKey)
					args = append(args, "--enable-admission-plugins=NodeRestriction")
					args = append(args, "--endpoint-reconciler-type=none")

					// if konnectivity is enabled, we need to write the egress config
					if vConfig.ControlPlane.Advanced.Konnectivity.Server.Enabled {
						egressConfig, err := pro.WriteKonnectivityEgressConfig()
						if err != nil {
							klog.Fatalf("error writing konnectivity egress config: %s", err.Error())
							return
						}

						args = append(args, "--egress-selector-config-file="+egressConfig)
					}
				}
			}

			// add extra args
			args = command.MergeArgs(args, apiServer.ExtraArgs)

			// wait until etcd is up and running
			err := etcd.WaitForEtcd(ctx, etcdCertificates, etcdEndpoints)
			if err != nil {
				klog.Fatalf("error waiting for etcd to be up: %s", err.Error())
				return
			}

			// now start the api server
			err = command.RunCommand(ctx, args, "apiserver")
			if err != nil {
				klog.Fatalf("error running apiserver: %s", err.Error())
				return
			}
			klog.Info("apiserver finished")
			os.Exit(0)
		}()
	}

	// wait for api server to be up as otherwise controller and scheduler might fail
	err = waitForAPI(ctx, vConfig.VirtualClusterKubeConfig().KubeConfig)
	if err != nil {
		return fmt.Errorf("waited until timeout for the api to be up: %w", err)
	}

	// start controller command
	controllerManager := vConfig.ControlPlane.Distro.K8S.ControllerManager
	if controllerManager.Enabled {
		go func() {
			// build flags
			args := []string{}
			if len(controllerManager.Command) > 0 {
				args = append(args, controllerManager.Command...)
			} else {
				args = append(args, constants.K8sControllerManagerBinary)
				args = append(args, "--service-cluster-ip-range="+serviceCIDR)
				args = append(args, "--authentication-kubeconfig="+constants.ControllerManagerConf)
				args = append(args, "--authorization-kubeconfig="+constants.ControllerManagerConf)
				args = append(args, "--bind-address=127.0.0.1")
				args = append(args, "--client-ca-file="+vConfig.VirtualClusterKubeConfig().ClientCACert)
				args = append(args, "--cluster-name=kubernetes")
				args = append(args, "--cluster-signing-cert-file="+vConfig.VirtualClusterKubeConfig().ServerCACert)
				args = append(args, "--cluster-signing-key-file="+vConfig.VirtualClusterKubeConfig().ServerCAKey)
				args = append(args, "--kubeconfig="+constants.ControllerManagerConf)
				args = append(args, "--requestheader-client-ca-file="+vConfig.VirtualClusterKubeConfig().RequestHeaderCACert)
				args = append(args, "--root-ca-file="+vConfig.VirtualClusterKubeConfig().ServerCACert)
				args = append(args, "--service-account-private-key-file="+constants.SAKey)
				args = append(args, "--use-service-account-credentials=true")
				args = append(args, "--leader-elect=true")

				if vConfig.PrivateNodes.Enabled {
					args = append(args, "--controllers=*,bootstrapsigner,tokencleaner")
					args = append(args, "--allocate-node-cidrs=true")
					args = append(args, "--cluster-cidr="+vConfig.Networking.PodCIDR)
					// we set cloud provider to external as we either want to use an external cloud controller manager
					// such as AWS or GCP or we fallback to our in-built cloud controller manager.
					args = append(args, "--cloud-provider=external")
				} else if vConfig.IsVirtualSchedulerEnabled() {
					args = append(args, "--controllers=*,-nodeipam,-persistentvolume-binder,-attachdetach,-persistentvolume-expander,-cloud-node-lifecycle,-ttl")
					args = append(args, "--node-monitor-grace-period=1h")
					args = append(args, "--node-monitor-period=1h")
					args = append(args, "--pvclaimbinder-sync-period=60s")
					args = append(args, "--horizontal-pod-autoscaler-sync-period=60s")
				} else {
					args = append(args, "--controllers=*,-nodeipam,-nodelifecycle,-persistentvolume-binder,-attachdetach,-persistentvolume-expander,-cloud-node-lifecycle,-ttl")
					args = append(args, "--node-monitor-grace-period=180s")
					args = append(args, "--node-monitor-period=30s")
					args = append(args, "--pvclaimbinder-sync-period=60s")
					args = append(args, "--horizontal-pod-autoscaler-sync-period=60s")
				}
			}

			// add extra args
			args = command.MergeArgs(args, controllerManager.ExtraArgs)
			err = command.RunCommand(ctx, args, "controller-manager")
			if err != nil {
				klog.Fatalf("error running controller-manager: %s", err.Error())
				return
			}
			klog.Info("controller-manager finished")
			os.Exit(0)
		}()
	}

	// start scheduler command
	scheduler := vConfig.ControlPlane.Distro.K8S.Scheduler
	if vConfig.IsVirtualSchedulerEnabled() || vConfig.PrivateNodes.Enabled {
		go func() {
			// build flags
			args := []string{}
			if len(scheduler.Command) > 0 {
				args = append(args, scheduler.Command...)
			} else {
				args = append(args, constants.K8sSchedulerBinary)
				args = append(args, "--authentication-kubeconfig="+constants.SchedulerConf)
				args = append(args, "--authorization-kubeconfig="+constants.SchedulerConf)
				args = append(args, "--bind-address=127.0.0.1")
				args = append(args, "--kubeconfig="+constants.SchedulerConf)
				args = append(args, "--leader-elect=true")
			}

			// add extra args
			args = command.MergeArgs(args, scheduler.ExtraArgs)
			err = command.RunCommand(ctx, args, "scheduler")
			if err != nil {
				klog.Fatalf("error running scheduler: %s", err.Error())
				return
			}
			klog.Info("scheduler finished")
			os.Exit(0)
		}()
	}

	<-ctx.Done()
	return ctx.Err()
}

func StartKine(ctx context.Context, dataSource, listenAddress string, certificates *etcd.Certificates, extraArgs []string) {
	// start kine
	doneChan := StartKineWithDone(ctx, dataSource, listenAddress, certificates, extraArgs)

	// wait for kine to finish
	go func() {
		err := <-doneChan
		if err != nil {
			klog.Fatalf("could not run kine: %s", err.Error())
		}
		klog.Info("kine finished")
		os.Exit(0)
	}()
}

func StartKineWithDone(ctx context.Context, dataSource, listenAddress string, certificates *etcd.Certificates, extraArgs []string) <-chan error {
	doneChan := make(chan error)

	// start embedded mode
	go func() {
		args := []string{}
		args = append(args, constants.KineBinary)
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
		args = command.MergeArgs(args, extraArgs)

		// now start kine
		err := command.RunCommand(ctx, args, "kine")
		doneChan <- err
	}()

	return doneChan
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
			dataSource = fmt.Sprintf("sqlite://%s%s", constants.K8sSqliteDatabase, SQLiteParams)
		}

		StartKine(ctx, dataSource, constants.K8sKineEndpoint, &etcd.Certificates{
			CaCert:     vConfig.ControlPlane.BackingStore.Database.Embedded.CaFile,
			ServerKey:  vConfig.ControlPlane.BackingStore.Database.Embedded.KeyFile,
			ServerCert: vConfig.ControlPlane.BackingStore.Database.Embedded.CertFile,
		}, vConfig.ControlPlane.BackingStore.Database.Embedded.ExtraArgs)

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
			CaCert:     filepath.Join(constants.PKIDir, "etcd", "ca.crt"),
			ServerCert: filepath.Join(constants.PKIDir, "apiserver-etcd-client.crt"),
			ServerKey:  filepath.Join(constants.PKIDir, "apiserver-etcd-client.key"),
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

func ExecTemplate(templateContents string, name, namespace string, values *vclusterconfig.Config) ([]byte, error) {
	out, err := json.Marshal(values)
	if err != nil {
		return nil, err
	}

	rawValues := map[string]interface{}{}
	err = json.Unmarshal(out, &rawValues)
	if err != nil {
		return nil, err
	}

	t, err := template.New("").Parse(templateContents)
	if err != nil {
		return nil, err
	}

	b := &bytes.Buffer{}
	err = t.Execute(b, map[string]interface{}{
		"Values": rawValues,
		"Release": map[string]interface{}{
			"Name":      name,
			"Namespace": namespace,
		},
	})
	if err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

// waits for the api to be up, ignoring certs and calling it
// localhost
func waitForAPI(ctx context.Context, kubeConfig string) error {
	rawConfig, err := clientcmd.LoadFromFile(kubeConfig)
	if err != nil {
		return fmt.Errorf("error loading kube config: %w", err)
	}

	restConfig, err := clientcmd.NewDefaultClientConfig(*rawConfig, &clientcmd.ConfigOverrides{}).ClientConfig()
	if err != nil {
		return fmt.Errorf("error creating rest client: %w", err)
	}

	clientSet, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return fmt.Errorf("error creating clientset: %w", err)
	}
	restClient := clientSet.DiscoveryClient.RESTClient()

	// sometimes the etcd pod takes a very long time to be ready,
	// we might want to fine tune how long we wait later
	var lastErr error
	err = wait.PollUntilContextTimeout(ctx, time.Second*2, time.Minute*5, true, func(ctx context.Context) (done bool, err error) {
		// do the request
		_, err = restClient.Get().AbsPath("/readyz").DoRaw(ctx)
		if err != nil {
			lastErr = err
			klog.Warningf("could not create the request to wait for the api: %s", err.Error())
			return false, nil
		}

		return true, nil
	})
	if err != nil {
		return fmt.Errorf("error waiting for API server: %w", lastErr)
	}

	return nil
}
