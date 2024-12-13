package start

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	"github.com/loft-sh/api/v4/pkg/product"
	"github.com/loft-sh/log"
	"github.com/loft-sh/log/survey"
	"github.com/loft-sh/log/terminal"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/platform"
	"github.com/loft-sh/vcluster/pkg/platform/clihelper"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/kubectl/pkg/util/term"
)

var (
	scheme = runtime.NewScheme()

	_ = clientgoscheme.AddToScheme(scheme)
	_ = storagev1.AddToScheme(scheme)
)

type Options struct {
	*flags.GlobalFlags
	// for logging
	CommandName string
	Log         log.Logger
	// Will be filled later
	KubeClient kubernetes.Interface
	RestConfig *rest.Config

	// cli options common to both start and destroy
	Context   string
	Namespace string
}

// StartOptions holds the cmd flags
type StartOptions struct { //nolint:revive // linter suggests renaming to options which already exists
	Options
	// cli options
	Values           string
	LocalPort        string
	Version          string
	DockerImage      string
	Password         string
	Host             string
	Email            string
	ChartRepo        string
	Product          string
	ChartName        string
	ChartPath        string
	DockerArgs       []string
	Reset            bool
	NoPortForwarding bool
	NoTunnel         bool
	NoLogin          bool
	NoWait           bool
	Upgrade          bool
	ReuseValues      bool
	Docker           bool
}

func NewLoftStarter(options StartOptions) *LoftStarter {
	return &LoftStarter{
		StartOptions: options,
	}
}

type LoftStarter struct {
	StartOptions
}

// Start executes the functionality "loft start"
func (l *LoftStarter) Start(ctx context.Context) error {
	// start in Docker?
	if l.Docker {
		return l.startDocker(ctx, "loft")
	}

	// only set local port by default in kubernetes installation
	if l.LocalPort == "" {
		l.LocalPort = "9898"
	}

	l.Log.WriteString(logrus.InfoLevel, "\n")

	// Uninstall already existing Loft instance
	if l.Reset {
		err := clihelper.UninstallLoft(ctx, l.KubeClient, l.RestConfig, l.Context, l.Namespace, l.Log)
		if err != nil {
			return err
		}
	}

	// Is already installed?
	isInstalled, err := clihelper.IsLoftAlreadyInstalled(ctx, l.KubeClient, l.Namespace)
	if err != nil {
		return err
	}

	// Use default password if none is set
	if l.Password == "" {
		defaultPassword, err := clihelper.GetLoftDefaultPassword(ctx, l.KubeClient, l.Namespace)
		if err != nil {
			return err
		}

		l.Password = defaultPassword
	}

	// Upgrade Loft if already installed
	if isInstalled {
		return l.handleAlreadyExistingInstallation(ctx)
	}

	// Install Loft
	l.Log.Info(product.Replace("Welcome to Loft!"))
	l.Log.Info(product.Replace("This installer will help you configure and deploy Loft."))

	err = l.upgradeLoft()
	if err != nil {
		return err
	}

	return l.success(ctx)
}

// Prepare initializes clients, verifies the existense of binaries, and ensures we are starting with the right kube context
func (l *Options) Prepare() error {
	platformClient := platform.NewClientFromConfig(l.LoadedConfig(l.Log))

	platformConfig := platformClient.Config().Platform

	// first load the kube config
	kubeClientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{})

	// load the raw config
	kubeConfig, err := kubeClientConfig.RawConfig()
	if err != nil {
		return fmt.Errorf("there is an error loading your current kube config (%w), please make sure you have access to a kubernetes cluster and the command `kubectl get namespaces` is working", err)
	}

	// we switch the context to the install config
	contextToLoad := kubeConfig.CurrentContext
	if l.Context != "" {
		contextToLoad = l.Context
	} else if platformConfig.LastInstallContext != "" && platformConfig.LastInstallContext != contextToLoad {
		if terminal.IsTerminalIn {
			contextToLoad, err = l.Log.Question(&survey.QuestionOptions{
				Question:     product.Replace(fmt.Sprintf("Seems like you try to use 'vcluster %s' with a different kubernetes context than before. Please choose which kubernetes context you want to use", l.CommandName)),
				DefaultValue: contextToLoad,
				Options:      []string{contextToLoad, platformConfig.LastInstallContext},
			})
			if err != nil {
				return err
			}
		}
	}

	l.Context = contextToLoad

	platformConfig.LastInstallContext = contextToLoad
	if err := platformClient.Save(); err != nil {
		return fmt.Errorf("save vCluster config: %w", err)
	}

	// kube client config
	kubeClientConfig = clientcmd.NewNonInteractiveClientConfig(kubeConfig, contextToLoad, &clientcmd.ConfigOverrides{}, clientcmd.NewDefaultClientConfigLoadingRules())

	// test for helm and kubectl
	_, err = exec.LookPath("helm")
	if err != nil {
		return fmt.Errorf("seems like helm is not installed. Helm is required for the installation of vCluster Platform. Please visit https://helm.sh/docs/intro/install/ for install instructions")
	}

	output, err := exec.Command("helm", "version").CombinedOutput()
	if err != nil {
		return fmt.Errorf("seems like there are issues with your helm client: \n\n%s", output)
	}

	_, err = exec.LookPath("kubectl")
	if err != nil {
		return fmt.Errorf("seems like kubectl is not installed. Kubectl is required for the installation of vCluster Platform. Please visit https://kubernetes.io/docs/tasks/tools/install-kubectl/ for install instructions")
	}

	output, err = exec.Command("kubectl", "version", "--context", contextToLoad).CombinedOutput()
	if err != nil {
		return fmt.Errorf("seems like kubectl cannot connect to your Kubernetes cluster: \n\n%s", output)
	}

	l.RestConfig, err = kubeClientConfig.ClientConfig()
	if err != nil {
		return fmt.Errorf("there is an error loading your current kube config (%w), please make sure you have access to a kubernetes cluster and the command `kubectl get namespaces` is working", err)
	}
	l.KubeClient, err = kubernetes.NewForConfig(l.RestConfig)
	if err != nil {
		return fmt.Errorf("there is an error loading your current kube config (%w), please make sure you have access to a kubernetes cluster and the command `kubectl get namespaces` is working", err)
	}

	// Check if cluster has RBAC correctly configured
	_, err = l.KubeClient.RbacV1().ClusterRoles().Get(context.Background(), "cluster-admin", metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error retrieving cluster role 'cluster-admin': %w. Please make sure RBAC is correctly configured in your cluster", err)
	}

	return nil
}

func (l *LoftStarter) handleAlreadyExistingInstallation(ctx context.Context) error {
	enableIngress := false

	// Only ask if ingress should be enabled if --upgrade flag is not provided
	if !l.Upgrade && term.IsTerminal(os.Stdin) {
		l.Log.Info(product.Replace("Existing Loft instance found."))

		// Check if Loft is installed in a local cluster
		isLocal := clihelper.IsLoftInstalledLocally(ctx, l.KubeClient, l.Namespace)

		// Skip question if --host flag is provided
		if l.Host != "" {
			enableIngress = true
		}

		if enableIngress {
			if isLocal {
				// Confirm with user if this is a local cluster
				const (
					YesOption = "Yes"
					NoOption  = "No, my cluster is running not locally (GKE, EKS, Bare Metal, etc.)"
				)

				answer, err := l.Log.Question(&survey.QuestionOptions{
					Question:     "Seems like your cluster is running locally (docker desktop, minikube, kind etc.). Is that correct?",
					DefaultValue: YesOption,
					Options: []string{
						YesOption,
						NoOption,
					},
				})
				if err != nil {
					return err
				}

				isLocal = answer == YesOption
			}

			if isLocal {
				// Confirm with user if ingress should be installed in local cluster
				var (
					YesOption = product.Replace("Yes, enable the ingress for Loft anyway")
					NoOption  = "No"
				)

				answer, err := l.Log.Question(&survey.QuestionOptions{
					Question:     product.Replace("Enabling ingress is usually only useful for remote clusters. Do you still want to deploy the ingress for Loft to your local cluster?"),
					DefaultValue: NoOption,
					Options: []string{
						NoOption,
						YesOption,
					},
				})
				if err != nil {
					return err
				}

				enableIngress = answer == YesOption
			}
		}

		// Check if we need to enable ingress
		if enableIngress {
			// Ask for hostname if --host flag is not provided
			if l.Host == "" {
				host, err := clihelper.EnterHostNameQuestion(l.Log)
				if err != nil {
					return err
				}

				l.Host = host
			} else {
				l.Log.Info(product.Replace("Will enable Loft ingress with hostname: ") + l.Host)
			}

			if term.IsTerminal(os.Stdin) {
				err := clihelper.EnsureIngressController(ctx, l.KubeClient, l.Context, l.Log)
				if err != nil {
					return errors.Wrap(err, "install ingress controller")
				}
			}
		}
	}

	// Only upgrade if --upgrade flag is present or user decided to enable ingress
	if l.Upgrade || enableIngress {
		err := l.upgradeLoft()
		if err != nil {
			return err
		}
	}

	return l.success(ctx)
}
