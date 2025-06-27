package framework

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd"
	"github.com/loft-sh/vcluster/pkg/cli"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/scheme"
	"github.com/loft-sh/vcluster/pkg/upgrade"
	logutil "github.com/loft-sh/vcluster/pkg/util/log"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	PollInterval             = 5 * time.Second
	PollTimeout              = time.Minute
	PollTimeoutLong          = 2 * time.Minute
	DefaultVClusterName      = "vcluster"
	DefaultVClusterNamespace = "vcluster"
	DefaultClientTimeout     = 100 * time.Second // the default in client-go is 32 (which is not enough, as we are occasionally experiencing client-side throttling in e2e tests)
	DefaultClientBurst       = 100               // the default in client-go is 10 (which is not enough, as we are occasionally experiencing client-side throttling in e2e tests)
	DefaultClientQPS         = 50                // the default in client-go is 5 (which is not enough, as we are occasionally experiencing client-side throttling in e2e tests)
)

var DefaultFramework = &Framework{}

type Framework struct {
	// The context to use for testing
	Context context.Context

	// VClusterName is the name of the vcluster instance which we are testing
	VClusterName string

	// VClusterNamespace is the namespace in host cluster of the current
	// vcluster instance which we are testing
	VClusterNamespace string

	// The suffix to append to the synced resources in the host namespace
	Suffix string

	// HostConfig is the kubernetes rest config of the
	// host kubernetes cluster were we are testing in
	HostConfig *rest.Config

	// HostClient is the kubernetes client of the current
	// host kubernetes cluster were we are testing in
	HostClient *kubernetes.Clientset

	// HostCRClient is the controller runtime client of the current
	// host kubernetes cluster were we are testing in
	HostCRClient client.Client

	// VClusterConfig is the kubernetes rest config of the current
	// vcluster instance which we are testing
	VClusterConfig *rest.Config

	// VClusterClient is the kubernetes client of the current
	// vcluster instance which we are testing
	VClusterClient *kubernetes.Clientset

	// VClusterCRClient is the controller runtime client of the current
	// vcluster instance which we are testing
	VClusterCRClient client.Client

	// VClusterKubeConfigFile is a file containing kube config
	// of the current vcluster instance which we are testing.
	// This file shall be deleted in the end of the test suite execution.
	VClusterKubeConfigFile *os.File

	// Log is the logger that should be used
	Log log.Logger

	// ClientTimeout value used in the clients
	ClientTimeout time.Duration

	// ClientBurst value used in the clients
	ClientBurst int

	// ClientQPS value used in the clients
	ClientQPS float32
}

func CreateFramework(ctx context.Context) error {
	// setup loggers
	ctrl.SetLogger(logutil.NewLog(0))
	l := log.GetInstance()

	name := os.Getenv("VCLUSTER_NAME")
	if name == "" {
		name = DefaultVClusterName
	}
	ns := os.Getenv("VCLUSTER_NAMESPACE")
	if ns == "" {
		ns = DefaultVClusterNamespace
	}
	timeoutEnvVar := os.Getenv("VCLUSTER_CLIENT_TIMEOUT")
	var timeout time.Duration
	timeoutInt, err := strconv.Atoi(timeoutEnvVar)
	if err == nil {
		timeout = time.Duration(timeoutInt) * time.Second
	} else {
		timeout = DefaultClientTimeout
	}

	clientBurstEnvVar := os.Getenv("VCLUSTER_CLIENT_BURST")
	var clientBurst int
	clientBurst, err = strconv.Atoi(clientBurstEnvVar)
	if err != nil {
		clientBurst = DefaultClientBurst
	}

	clientQPSEnvVar := os.Getenv("VCLUSTER_CLIENT_QPS")
	var clientQPS int
	clientQPS, err = strconv.Atoi(clientQPSEnvVar)
	if err != nil {
		clientQPS = DefaultClientQPS
	}

	suffix := os.Getenv("VCLUSTER_SUFFIX")
	if suffix == "" {
		// TODO: maybe implement some autodiscovery of the suffix value that would work with dev and prod setups
		suffix = "vcluster"
	}
	translate.VClusterName = suffix

	translate.Default = translate.NewSingleNamespaceTranslator(ns)

	l.Infof("Testing vCluster named: %s in namespace: %s", name, ns)
	hostConfig, err := ctrl.GetConfig()
	if err != nil {
		return err
	}
	hostConfig.Timeout = timeout
	hostConfig.Burst = clientBurst
	hostConfig.QPS = float32(clientQPS)

	hostClient, err := kubernetes.NewForConfig(hostConfig)
	if err != nil {
		return err
	}

	hostCRClient, err := client.New(hostConfig, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		return err
	}

	// create the framework
	DefaultFramework = &Framework{
		Context:           ctx,
		VClusterName:      name,
		VClusterNamespace: ns,
		Suffix:            suffix,
		HostConfig:        hostConfig,
		HostClient:        hostClient,
		HostCRClient:      hostCRClient,
		Log:               l,
		ClientTimeout:     timeout,
		ClientBurst:       clientBurst,
		ClientQPS:         float32(clientQPS),
	}

	// init virtual client
	if os.Getenv("VCLUSTER_SKIP_CONNECT") != "true" {
		err = DefaultFramework.RefreshVirtualClient()
		if err != nil {
			return err
		}
	}

	l.Done("Framework successfully initialized")
	return nil
}

func (f *Framework) RefreshVirtualClient() error {
	// run port forwarder and retrieve kubeconfig for the vcluster
	vKubeconfigFile, err := os.CreateTemp(os.TempDir(), "vcluster_e2e_kubeconfig_")
	if err != nil {
		return fmt.Errorf("could not create a temporary file: %w", err)
	}

	// vKubeConfigFile removal is done in the Framework.Cleanup() which gets called in ginkgo's AfterSuite()
	connectCmd := cmd.ConnectCmd{
		CobraCmd: &cobra.Command{},
		Log:      f.Log,
		GlobalFlags: &flags.GlobalFlags{
			Namespace: f.VClusterNamespace,
			Debug:     true,
		},
		ConnectOptions: cli.ConnectOptions{
			KubeConfig:           vKubeconfigFile.Name(),
			LocalPort:            14550, // choosing a port that usually should be unused
			BackgroundProxy:      true,
			BackgroundProxyImage: constants.DefaultBackgroundProxyImage(upgrade.GetVersion()),
		},
	}
	err = connectCmd.Run(f.Context, []string{f.VClusterName})
	if err != nil {
		f.Log.Fatalf("failed to connect to the vcluster: %v", err)
	}

	var vClusterConfig *rest.Config
	var vClusterClient *kubernetes.Clientset
	var vClusterCRClient client.Client

	err = wait.PollUntilContextTimeout(f.Context, time.Second, time.Minute*5, false, func(ctx context.Context) (bool, error) {
		output, err := os.ReadFile(vKubeconfigFile.Name())
		if err != nil {
			return false, nil
		}

		// try to parse config from file with retry because the file content might not be written
		vClusterConfig, err = clientcmd.RESTConfigFromKubeConfig(output)
		if err != nil {
			return false, err
		}
		vClusterConfig.Timeout = f.ClientTimeout
		vClusterConfig.Burst = f.ClientBurst
		vClusterConfig.QPS = f.ClientQPS

		// create kubernetes client using the config retry in case port forwarding is not ready yet
		vClusterClient, err = kubernetes.NewForConfig(vClusterConfig)
		if err != nil {
			return false, err
		}

		vClusterCRClient, err = client.New(vClusterConfig, client.Options{Scheme: scheme.Scheme})
		if err != nil {
			return false, err
		}

		// try to use the client with retry in case port forwarding is not ready yet
		_, err = vClusterClient.CoreV1().ServiceAccounts("default").Get(ctx, "default", metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		return true, nil
	})
	if err != nil {
		return err
	}

	f.VClusterConfig = vClusterConfig
	f.VClusterClient = vClusterClient
	f.VClusterCRClient = vClusterCRClient
	f.VClusterKubeConfigFile = vKubeconfigFile
	return nil
}

func (f *Framework) Cleanup() error {
	return os.Remove(f.VClusterKubeConfigFile.Name())
}
