package cmd

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
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
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	LogsMountPath = "/var/log/pods"

	HostLogFolderPattern = `(?P<namespace>[\w][^_]+)_(?P<podname>[\w][^_]+)_(?P<UID>[\w|\d|-]+)`
)

var (
	podMappings PhysicalPodToVirtualPodDetail

	hostLogFolderRegex *regexp.Regexp
)

func init() {
	podMappings = make(PhysicalPodToVirtualPodDetail)

	hostLogFolderRegex = regexp.MustCompile(HostLogFolderPattern)
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

	startManagers(context.TODO(), localManager, virtualClusterManager)

	mapLogs(context.TODO(), localManager, virtualClusterManager, options)

	return nil
}

func mapLogs(ctx context.Context, pManager, vManager manager.Manager, options *context2.VirtualClusterOptions) {
	wait.Forever(func() {
		podList := &v1.PodList{}
		err := vManager.GetClient().List(ctx, podList)
		if err != nil {
			klog.Errorf("unable to list pods: %v", err)
			return
		}

		fillUpPodMapping(ctx, podList)
		klog.Info("podMapping length: ", len(podMappings))

		podsOnNode, err := ioutil.ReadDir(LogsMountPath)
		if err != nil {
			klog.Errorf("error encountered during file walk: %v", err)
		}

		for _, podOnNode := range podsOnNode {
			// klog.Infoln("found dir", item.Name())
			matches := hostLogFolderRegex.FindAllStringSubmatch(podOnNode.Name(), -1)
			namespace, podName, UID := hostLogFolderRegex.SubexpIndex("namespace"), hostLogFolderRegex.SubexpIndex("podname"), hostLogFolderRegex.SubexpIndex("UID")
			for _, match := range matches {
				// klog.Infof("found\n\tnamespace: %s\n\tpodName: %s\n\tUID: %s\n", match[namespace], match[podName], match[UID])
				if pod, ok := podMappings[match[podName]]; ok {
					klog.Infof("found a matching pod: %s/%s", match[namespace], match[podName])
					klog.Infof("physical pod UID: %s", match[UID])
					klog.Infof("virtualPodUID: %s", pod.PodObj.UID)
					klog.Infoln()

					// check if symlink exists
					if pod.SymLinkName == nil {
						// create symlink
						symlinkName, err := createSymlinkToPhysical(ctx,
							pod.PodObj.Namespace,
							pod.PodObj.Name,
							string(pod.PodObj.UID),
							podOnNode.Name())
						if err != nil {
							klog.Errorf("error creating symlink: %v", err)
						}

						pod.SymLinkName = symlinkName
					}
				}
			}
		}
	}, time.Second*5)
}

func fillUpPodMapping(ctx context.Context, podList *v1.PodList) {
	for _, pod := range podList.Items {
		// klog.Infof("pod: %s:%s", pod.Namespace, pod.Name)
		pname := translate.PhysicalName(pod.Name, pod.Namespace)

		// add to pod mappings
		podMappings[pname] = &PodDetail{
			PodObj: pod,
		}
	}
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
	vPodDirName := fmt.Sprintf("%s_%s_%s", namespace, podName, UID)

	err := os.Symlink(target, vPodDirName)
	if err != nil {
		if os.IsExist(err) {
			return nil, nil
		}

		return nil, err
	}

	return &vPodDirName, nil
}

// map of physical pod names to the corresponding virtual pod
type PhysicalPodToVirtualPodDetail map[string]*PodDetail

type PodDetail struct {
	PodObj      v1.Pod
	SymLinkName *string
}
