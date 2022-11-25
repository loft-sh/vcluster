package framework

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/flags"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/log"
	logutil "github.com/loft-sh/vcluster/pkg/util/log"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	PollTimeout              = time.Minute
	DefaultVclusterName      = "vcluster"
	DefaultVclusterNamespace = "vcluster"
	DefaultClientTimeout     = 32 * time.Second // the default in client-go is 32
)

var DefaultFramework = &Framework{}

type Framework struct {
	// The context to use for testing
	Context context.Context

	// VclusterName is the name of the vcluster instance which we are testing
	VclusterName string

	// VclusterNamespace is the namespace in host cluster of the current
	// vcluster instance which we are testing
	VclusterNamespace string

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

	// VclusterConfig is the kubernetes rest config of the current
	// vcluster instance which we are testing
	VclusterConfig *rest.Config

	// VclusterClient is the kubernetes client of the current
	// vcluster instance which we are testing
	VclusterClient *kubernetes.Clientset

	// VclusterCRClient is the controller runtime client of the current
	// vcluster instance which we are testing
	VclusterCRClient client.Client

	// VclusterKubeconfigFile is a file containing kube config
	// of the current vcluster instance which we are testing.
	// This file shall be deleted in the end of the test suite execution.
	VclusterKubeconfigFile *os.File

	// Scheme is the global scheme to use
	Scheme *runtime.Scheme

	// Log is the logger that should be used
	Log log.Logger

	// ClientTimeout value used in the clients
	ClientTimeout time.Duration
}

func CreateFramework(ctx context.Context, scheme *runtime.Scheme) error {
	// setup loggers
	ctrl.SetLogger(logutil.NewLog(0))
	l := log.GetInstance()

	name := os.Getenv("VCLUSTER_NAME")
	if name == "" {
		name = DefaultVclusterName
	}
	ns := os.Getenv("VCLUSTER_NAMESPACE")
	if ns == "" {
		ns = DefaultVclusterNamespace
	}
	timeoutEnvVar := os.Getenv("VCLUSTER_CLIENT_TIMEOUT")
	var timeout time.Duration
	timeoutInt, err := strconv.Atoi(timeoutEnvVar)
	if err == nil {
		timeout = time.Duration(timeoutInt) * time.Second
	} else {
		timeout = DefaultClientTimeout
	}

	suffix := os.Getenv("VCLUSTER_SUFFIX")
	if suffix == "" {
		//TODO: maybe implement some autodiscovery of the suffix value that would work with dev and prod setups
		suffix = "vcluster"
	}
	translate.Suffix = suffix
	translate.NewSingleNamespaceTranslator(ns)
	l.Infof("Testing Vcluster named: %s in namespace: %s", name, ns)

	hostConfig, err := ctrl.GetConfig()
	if err != nil {
		return err
	}
	hostConfig.Timeout = timeout

	hostClient, err := kubernetes.NewForConfig(hostConfig)
	if err != nil {
		return err
	}

	hostCRClient, err := client.New(hostConfig, client.Options{Scheme: scheme})
	if err != nil {
		return err
	}

	// run port forwarder and retrieve kubeconfig for the vcluster
	vKubeconfigFile, err := os.CreateTemp(os.TempDir(), "vcluster_e2e_kubeconfig_")
	if err != nil {
		return fmt.Errorf("could not create a temporary file: %v", err)
	}
	// vKubeconfigFile removal is done in the Framework.Cleanup() which gets called in ginkgo's AfterSuite()

	connectCmd := cmd.ConnectCmd{
		Log: l,
		GlobalFlags: &flags.GlobalFlags{
			Namespace: ns,
			Debug:     true,
		},
		KubeConfig: vKubeconfigFile.Name(),
		LocalPort:  14550, // choosing a port that usually should be unused
	}
	err = connectCmd.Connect(name, nil)
	if err != nil {
		l.Fatalf("failed to connect to the vcluster: %v", err)
	}

	var vclusterConfig *rest.Config
	var vclusterClient *kubernetes.Clientset
	var vclusterCRClient client.Client

	err = wait.PollImmediate(time.Second, time.Minute*5, func() (bool, error) {
		output, err := os.ReadFile(vKubeconfigFile.Name())
		if err != nil {
			return false, nil
		}

		// try to parse config from file with retry because the file content might not be written
		vclusterConfig, err = clientcmd.RESTConfigFromKubeConfig(output)
		if err != nil {
			return false, err
		}
		vclusterConfig.Timeout = timeout

		// create kubernetes client using the config retry in case port forwarding is not ready yet
		vclusterClient, err = kubernetes.NewForConfig(vclusterConfig)
		if err != nil {
			return false, err
		}

		vclusterCRClient, err = client.New(vclusterConfig, client.Options{Scheme: scheme})
		if err != nil {
			return false, err
		}

		// try to use the client with retry in case port forwarding is not ready yet
		_, err = vclusterClient.CoreV1().ServiceAccounts("default").Get(ctx, "default", metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		return true, nil
	})
	if err != nil {
		return err
	}

	// create the framework
	DefaultFramework = &Framework{
		Context:                ctx,
		VclusterName:           name,
		VclusterNamespace:      ns,
		Suffix:                 suffix,
		HostConfig:             hostConfig,
		HostClient:             hostClient,
		HostCRClient:           hostCRClient,
		VclusterConfig:         vclusterConfig,
		VclusterClient:         vclusterClient,
		VclusterCRClient:       vclusterCRClient,
		VclusterKubeconfigFile: vKubeconfigFile,
		Scheme:                 scheme,
		Log:                    l,
		ClientTimeout:          timeout,
	}

	l.Done("Framework successfully initialized")
	return nil
}

func (f *Framework) Cleanup() error {
	return os.Remove(f.VclusterKubeconfigFile.Name())
}
