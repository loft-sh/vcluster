package cmd

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	context2 "github.com/loft-sh/vcluster/cmd/vcluster/context"
	"github.com/loft-sh/vcluster/pkg/util/blockingcacheclient"
	"github.com/loft-sh/vcluster/pkg/util/pluginhookclient"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	LogsMountPath = "/var/log/pods"
)

var (
	podMappings PhysicalPodToVirtualPodDetail
)

func init() {
	podMappings = make(PhysicalPodToVirtualPodDetail)
}

func NewLogMapperCommand() *cobra.Command {
	options := &context2.VirtualClusterOptions{}
	cmd := &cobra.Command{
		Use:   "maplogs",
		Short: "Map host to virtual pod logs",
		Args:  cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return MapHostPathLogs(options)
		},
	}

	cmd.Flags().StringVar(&options.KubeConfigContextName, "kube-config-context-name", "", "If set, will override the context name of the generated virtual cluster kube config with this name")
	// cmd.Flags().StringSliceVar(&options.Controllers, "sync", []string{}, "A list of sync controllers to enable. 'foo' enables the sync controller named 'foo', '-foo' disables the sync controller named 'foo'")
	cmd.Flags().StringVar(&options.RequestHeaderCaCert, "request-header-ca-cert", "/data/server/tls/request-header-ca.crt", "The path to the request header ca certificate")
	cmd.Flags().StringVar(&options.ClientCaCert, "client-ca-cert", "/data/server/tls/client-certificate", "The path to the client ca certificate")
	cmd.Flags().StringVar(&options.ServerCaCert, "server-ca-cert", "/data/server/tls/certificate-authority", "The path to the server ca certificate")
	cmd.Flags().StringVar(&options.ServerCaKey, "server-ca-key", "/data/server/tls/client-key", "The path to the server ca key")
	cmd.Flags().StringVar(&options.KubeConfigPath, "kube-config", "/data/server/tls/config", "The path to the virtual cluster admin kube config")
	cmd.Flags().StringSliceVar(&options.TLSSANs, "tls-san", []string{}, "Add additional hostname or IP as a Subject Alternative Name in the TLS cert")

	cmd.Flags().StringVar(&options.KubeConfigSecret, "out-kube-config-secret", "", "If specified, the virtual cluster will write the generated kube config to the given secret")
	cmd.Flags().StringVar(&options.KubeConfigSecretNamespace, "out-kube-config-secret-namespace", "", "If specified, the virtual cluster will write the generated kube config in the given namespace")
	cmd.Flags().StringVar(&options.KubeConfigServer, "out-kube-config-server", "", "If specified, the virtual cluster will use this server for the generated kube config (e.g. https://my-vcluster.domain.com)")

	cmd.Flags().StringVar(&options.TargetNamespace, "target-namespace", "", "The namespace to run the virtual cluster in (defaults to current namespace)")
	cmd.Flags().StringVar(&options.ServiceName, "service-name", "", "The service name where the vcluster proxy will be available")
	cmd.Flags().BoolVar(&options.SetOwner, "set-owner", true, "If true, will set the same owner the currently running syncer pod has on the synced resources")

	cmd.Flags().StringVar(&options.Name, "name", "vcluster", "The name of the virtual cluster")
	cmd.Flags().StringVar(&options.BindAddress, "bind-address", "0.0.0.0", "The address to bind the server to")
	cmd.Flags().IntVar(&options.Port, "port", 8443, "The port to bind to")

	cmd.Flags().BoolVar(&options.SyncAllNodes, "sync-all-nodes", false, "If enabled and --fake-nodes is false, the virtual cluster will sync all nodes instead of only the needed ones")

	cmd.Flags().StringVar(&options.ServiceAccount, "service-account", "", "If set, will set this host service account on the synced pods")

	return cmd
}

func MapHostPathLogs(options *context2.VirtualClusterOptions) error {
	inClusterConfig := ctrl.GetConfigOrDie()

	inClusterConfig.QPS = 40
	inClusterConfig.Burst = 80
	inClusterConfig.Timeout = 0

	translate.Suffix = options.Name

	var virtualClusterConfig *rest.Config
	err := wait.PollImmediate(time.Second, time.Hour, func() (bool, error) {
		virtualClusterConfig = &rest.Config{
			Host: options.Name,
			TLSClientConfig: rest.TLSClientConfig{
				ServerName: options.Name,
				CertFile:   options.ClientCaCert,
				KeyFile:    options.ServerCaKey,
				CAFile:     options.ServerCaCert,
			},
		}

		kubeClient, err := kubernetes.NewForConfig(virtualClusterConfig)
		if err != nil {
			return false, errors.Wrap(err, "create kube client")
		}

		_, err = kubeClient.Discovery().ServerVersion()
		if err != nil {
			klog.Infof("couldn't retrieve virtual cluster version (%v), will retry in 1 seconds", err)
			return false, nil
		}
		_, err = kubeClient.CoreV1().ServiceAccounts("default").Get(context.Background(), "default", metav1.GetOptions{})
		if err != nil {
			klog.Infof("default ServiceAccount is not available yet, will retry in 1 seconds")
			return false, nil
		}

		return true, nil
	})
	if err != nil {
		return err
	}

	localManager, err := ctrl.NewManager(inClusterConfig, ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: "0",
		LeaderElection:     false,
		Namespace:          options.TargetNamespace,
		NewClient:          pluginhookclient.NewPhysicalPluginClientFactory(blockingcacheclient.NewCacheClient),
	})
	if err != nil {
		return err
	}

	virtualClusterManager, err := ctrl.NewManager(virtualClusterConfig, ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: "0",
		LeaderElection:     false,
		NewClient:          pluginhookclient.NewVirtualPluginClientFactory(blockingcacheclient.NewCacheClient),
	})
	if err != nil {
		return err
	}

	ctx := context.Background()

	startManagers(ctx, localManager, virtualClusterManager)

	mapLogs(ctx, localManager, virtualClusterManager, options)

	return nil
}

func mapLogs(ctx context.Context, pManager, vManager manager.Manager, options *context2.VirtualClusterOptions) {
	wait.Forever(func() {
		podList := &v1.PodList{}
		err := pManager.GetClient().List(ctx, podList, &client.ListOptions{Namespace: options.TargetNamespace})
		if err != nil {
			klog.Errorf("unable to list pods: %v", err)
			return
		}

		fillUpPodMapping(ctx, podList)
		klog.Infof("podMapping length: %d", len(podMappings))

		vPodList := &v1.PodList{}
		err = vManager.GetClient().List(ctx, vPodList)
		if err != nil {
			klog.Errorf("unable to list pods: %v", err)
			return
		}

		for _, pod := range vPodList.Items {
			pName := translate.PhysicalName(pod.Name, pod.Namespace)

			if v, ok := podMappings[pName]; ok {
				// if v.SymLinkName == nil {
				// create symlink
				symlinkName, err := createSymlinkToPhysical(ctx, pod.Namespace, pod.Name, string(pod.UID), v.Target)
				if err != nil {
					klog.Errorf("unable to create symlink for %s: %v", v.Target, err)
				}

				v.SymLinkName = symlinkName
				// }
			}
		}

	}, time.Second*5)
}

func fillUpPodMapping(ctx context.Context, podList *v1.PodList) {
	for _, pod := range podList.Items {
		lookupName := fmt.Sprintf("%s_%s_%s", pod.Namespace, pod.Name, pod.UID)

		ok, err := checkIfPathExists(lookupName)
		if err != nil {
			klog.Errorf("error checking existence for path %s: %v", lookupName, err)
		}

		if ok {
			// check entry in podMapping
			if _, ok := podMappings[pod.Name]; !ok {
				podMappings[pod.Name] = &PodDetail{
					Target: lookupName,
				}
			}
		}
	}
}

// check if folder exists
func checkIfPathExists(path string) (bool, error) {
	fullPath := filepath.Join(LogsMountPath, path)
	if _, err := os.Stat(fullPath); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

func startManagers(ctx context.Context, pManager, vManager manager.Manager) {
	go func() {
		err := pManager.Start(ctx)
		if err != nil {
			panic(err)
		}
	}()

	go func() {
		err := vManager.Start(ctx)
		if err != nil {
			panic(err)
		}
	}()
}

func createSymlinkToPhysical(ctx context.Context, namespace, podName, UID, target string) (*string, error) {
	vPodDirName := filepath.Join(LogsMountPath, fmt.Sprintf("%s_%s_%s", namespace, podName, UID))

	target = filepath.Join(LogsMountPath, target)
	klog.Infof("creating symlink from %s -> %s", vPodDirName, target)
	err := os.Symlink(target, vPodDirName)
	if err != nil {
		if os.IsExist(err) {
			return nil, err
		}

		return nil, err
	}

	return &vPodDirName, nil
}

// map of physical pod names to the corresponding virtual pod
type PhysicalPodToVirtualPodDetail map[string]*PodDetail

type PodDetail struct {
	Target      string
	SymLinkName *string
}
