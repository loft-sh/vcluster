package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/loft-sh/loftctl/v3/cmd/loftctl/cmd/use"
	proclient "github.com/loft-sh/loftctl/v3/pkg/client"
	"github.com/loft-sh/loftctl/v3/pkg/vcluster"
	"github.com/loft-sh/vcluster/pkg/procli"
	"github.com/loft-sh/vcluster/pkg/util/clihelper"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	loftctlUtil "github.com/loft-sh/loftctl/v3/pkg/util"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/app/localkubernetes"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/find"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/flags"
	"github.com/loft-sh/vcluster/pkg/lifecycle"
	"github.com/loft-sh/vcluster/pkg/upgrade"
	"github.com/loft-sh/vcluster/pkg/util/portforward"
	"github.com/loft-sh/vcluster/pkg/util/translate"
)

// ConnectCmd holds the cmd flags
type ConnectCmd struct {
	*flags.GlobalFlags
	rawConfig                 clientcmdapi.Config
	kubeClientConfig          clientcmd.ClientConfig
	Log                       log.Logger
	errorChan                 chan error
	interruptChan             chan struct{}
	restConfig                *rest.Config
	kubeClient                *kubernetes.Clientset
	ServiceAccountClusterRole string
	PodName                   string
	Address                   string
	KubeConfigContextName     string
	Server                    string
	KubeConfig                string
	Project                   string
	ServiceAccount            string
	LocalPort                 int
	ServiceAccountExpiration  int
	Print                     bool
	UpdateCurrent             bool
	BackgroundProxy           bool
	portForwarding            bool
	Insecure                  bool
}

// NewConnectCmd creates a new command
func NewConnectCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &ConnectCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}

	useLine, nameValidator := loftctlUtil.NamedPositionalArgsValidator(true, false, "VCLUSTER_NAME")

	cobraCmd := &cobra.Command{
		Use:   "connect" + useLine,
		Short: "Connect to a virtual cluster",
		Long: `
#######################################################
################## vcluster connect ###################
#######################################################
Connect to a virtual cluster

Example:
vcluster connect test --namespace test
# Open a new bash with the vcluster KUBECONFIG defined
vcluster connect test -n test -- bash
vcluster connect test -n test -- kubectl get ns
#######################################################
	`,
		Args:              nameValidator,
		ValidArgsFunction: newValidVClusterNameFunc(globalFlags),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			// Check for newer version
			upgrade.PrintNewerVersionWarning()

			return cmd.Run(cobraCmd.Context(), args)
		},
	}

	cobraCmd.Flags().StringVar(&cmd.KubeConfigContextName, "kube-config-context-name", "", "If set, will override the context name of the generated virtual cluster kube config with this name")
	cobraCmd.Flags().StringVar(&cmd.KubeConfig, "kube-config", "./kubeconfig.yaml", "Writes the created kube config to this file")
	cobraCmd.Flags().BoolVar(&cmd.UpdateCurrent, "update-current", true, "If true updates the current kube config")
	cobraCmd.Flags().BoolVar(&cmd.Print, "print", false, "When enabled prints the context to stdout")
	cobraCmd.Flags().StringVar(&cmd.PodName, "pod", "", "The pod to connect to")
	cobraCmd.Flags().StringVar(&cmd.Server, "server", "", "The server to connect to")
	cobraCmd.Flags().IntVar(&cmd.LocalPort, "local-port", 0, "The local port to forward the virtual cluster to. If empty, vcluster will use a random unused port")
	cobraCmd.Flags().StringVar(&cmd.Address, "address", "", "The local address to start port forwarding under")
	cobraCmd.Flags().StringVar(&cmd.ServiceAccount, "service-account", "", "If specified, vcluster will create a service account token to connect to the virtual cluster instead of using the default client cert / key. Service account must exist and can be used as namespace/name.")
	cobraCmd.Flags().StringVar(&cmd.ServiceAccountClusterRole, "cluster-role", "", "If specified, vcluster will create the service account if it does not exist and also add a cluster role binding for the given cluster role to it. Requires --service-account to be set")
	cobraCmd.Flags().IntVar(&cmd.ServiceAccountExpiration, "token-expiration", 0, "If specified, vcluster will create the service account token for the given duration in seconds. Defaults to eternal")
	cobraCmd.Flags().BoolVar(&cmd.Insecure, "insecure", false, "If specified, vcluster will create the kube config with insecure-skip-tls-verify")
	cobraCmd.Flags().BoolVar(&cmd.BackgroundProxy, "background-proxy", false, "If specified, vcluster will create the background proxy in docker [its mainly used for vclusters with no nodeport service.]")

	// pro
	cobraCmd.Flags().StringVar(&cmd.Project, "project", "", "[PRO] The pro project the vcluster is in")

	return cobraCmd
}

// Run executes the functionality
func (cmd *ConnectCmd) Run(ctx context.Context, args []string) error {
	vClusterName := ""
	if len(args) > 0 {
		vClusterName = args[0]
	}

	proClient, err := procli.CreateProClient()
	if err != nil {
		cmd.Log.Debugf("Error creating pro client: %v", err)
	}

	return cmd.Connect(ctx, proClient, vClusterName, args[1:])
}

func (cmd *ConnectCmd) Connect(ctx context.Context, proClient procli.Client, vClusterName string, command []string) error {
	// validate flags
	err := cmd.validateFlags()
	if err != nil {
		return err
	}

	// retrieve the vcluster
	vCluster, proVCluster, err := find.GetVCluster(ctx, proClient, cmd.Context, vClusterName, cmd.Namespace, cmd.Project, cmd.Log)
	if err != nil {
		return err
	} else if proVCluster != nil {
		return cmd.connectPro(ctx, proClient, proVCluster, command)
	}

	return cmd.connectOss(ctx, vCluster, command)
}

func (cmd *ConnectCmd) validateFlags() error {
	if cmd.ServiceAccountClusterRole != "" && cmd.ServiceAccount == "" {
		return fmt.Errorf("expected --service-account to be defined as well")
	}

	return nil
}

func (cmd *ConnectCmd) connectPro(ctx context.Context, proClient proclient.Client, vCluster *procli.VirtualClusterInstanceProject, command []string) error {
	err := cmd.validateProFlags()
	if err != nil {
		return err
	}

	// create management client
	managementClient, err := proClient.Management()
	if err != nil {
		return err
	}

	// wait for vCluster to become ready
	vCluster.VirtualCluster, err = vcluster.WaitForVirtualClusterInstance(ctx, managementClient, vCluster.VirtualCluster.Namespace, vCluster.VirtualCluster.Name, true, cmd.Log)
	if err != nil {
		return err
	}

	// retrieve vCluster kube config
	kubeConfig, err := cmd.getVClusterProKubeConfig(ctx, proClient, vCluster)
	if err != nil {
		return err
	}

	// check if we should execute command
	if len(command) > 0 {
		return cmd.executeCommand(*kubeConfig, command)
	}

	return cmd.writeKubeConfig(kubeConfig, vCluster.VirtualCluster.Name)
}

func (cmd *ConnectCmd) validateProFlags() error {
	if cmd.PodName != "" {
		return fmt.Errorf("cannot use --pod with a pro vCluster")
	}
	if cmd.Server != "" {
		return fmt.Errorf("cannot use --server with a pro vCluster")
	}
	if cmd.BackgroundProxy {
		return fmt.Errorf("cannot use --background-proxy with a pro vCluster")
	}
	if cmd.LocalPort != 0 {
		return fmt.Errorf("cannot use --local-port with a pro vCluster")
	}
	if cmd.Address != "" {
		return fmt.Errorf("cannot use --address with a pro vCluster")
	}

	return nil
}

func (cmd *ConnectCmd) connectOss(ctx context.Context, vCluster *find.VCluster, command []string) error {
	// prepare clients and find vcluster
	err := cmd.prepare(ctx, vCluster)
	if err != nil {
		return err
	}

	// retrieve vcluster kube config
	kubeConfig, err := cmd.getVClusterKubeConfig(ctx, vCluster.Name, command)
	if err != nil {
		return err
	}

	if len(command) == 0 && cmd.ServiceAccount == "" && cmd.Server == "" && cmd.BackgroundProxy && localkubernetes.IsDockerInstalledAndUpAndRunning() {
		// start background container
		server, err := localkubernetes.CreateBackgroundProxyContainer(ctx, vCluster.Name, cmd.Namespace, &cmd.rawConfig, kubeConfig, cmd.LocalPort, cmd.Log)
		if err != nil {
			cmd.Log.Warnf("Error exposing local vcluster, will fallback to port-forwarding: %v", err)
			cmd.BackgroundProxy = false
		}
		cmd.Server = server
	}

	// check if we should execute command
	if len(command) > 0 {
		if !cmd.portForwarding {
			return fmt.Errorf("command is specified, but port-forwarding isn't started")
		}
		defer close(cmd.interruptChan)

		// wait for vcluster to be ready
		err := cmd.waitForVCluster(ctx, *kubeConfig, cmd.errorChan)
		if err != nil {
			return err
		}

		// build vKubeConfig
		return cmd.executeCommand(cmd.getLocalVClusterConfig(*kubeConfig), command)
	}

	return cmd.writeKubeConfig(kubeConfig, vCluster.Name)
}

func (cmd *ConnectCmd) writeKubeConfig(kubeConfig *clientcmdapi.Config, vClusterName string) error {
	// write kube config to buffer
	out, err := clientcmd.Write(*kubeConfig)
	if err != nil {
		return err
	}

	// write kube config to file
	if cmd.UpdateCurrent {
		var clusterConfig *clientcmdapi.Cluster
		for _, c := range kubeConfig.Clusters {
			clusterConfig = c
		}

		var authConfig *clientcmdapi.AuthInfo
		for _, a := range kubeConfig.AuthInfos {
			authConfig = a
		}

		err = clihelper.UpdateKubeConfig(cmd.KubeConfigContextName, clusterConfig, authConfig, true)
		if err != nil {
			return err
		}

		cmd.Log.Donef("Switched active kube context to %s", cmd.KubeConfigContextName)
		if !cmd.BackgroundProxy && cmd.portForwarding {
			cmd.Log.Warnf("Since you are using port-forwarding to connect, you will need to leave this terminal open")
			c := make(chan os.Signal, 1)
			signal.Notify(c, os.Interrupt, syscall.SIGTERM)
			go func() {
				<-c
				kubeConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{}).RawConfig()
				if err == nil && kubeConfig.CurrentContext == cmd.KubeConfigContextName {
					err = deleteContext(&kubeConfig, cmd.KubeConfigContextName, cmd.Context)
					if err != nil {
						cmd.Log.Errorf("Error deleting context: %v", err)
					} else {
						cmd.Log.Infof("Switched back to context %v", cmd.Context)
					}
				}
				os.Exit(1)
			}()

			defer func() {
				signal.Stop(c)
			}()
			cmd.Log.WriteString(logrus.InfoLevel, "- Use CTRL+C to return to your previous kube context\n")
			cmd.Log.WriteString(logrus.InfoLevel, "- Use `kubectl get namespaces` in another terminal to access the vcluster\n")
		} else {
			cmd.Log.WriteString(logrus.InfoLevel, "- Use `vcluster disconnect` to return to your previous kube context\n")
			cmd.Log.WriteString(logrus.InfoLevel, "- Use `kubectl get namespaces` to access the vcluster\n")
		}
	} else if cmd.Print {
		_, err = os.Stdout.Write(out)
		if err != nil {
			return err
		}
	} else {
		err = os.WriteFile(cmd.KubeConfig, out, 0666)
		if err != nil {
			return errors.Wrap(err, "write kube config")
		}

		cmd.Log.Donef("Virtual cluster kube config written to: %s", cmd.KubeConfig)
		if cmd.Server == "" {
			cmd.Log.WriteString(logrus.InfoLevel, fmt.Sprintf("- Use `vcluster connect %s -n %s -- kubectl get ns` to execute a command directly within this terminal\n", vClusterName, cmd.Namespace))
		}
		cmd.Log.WriteString(logrus.InfoLevel, fmt.Sprintf("- Use `kubectl --kubeconfig %s get namespaces` to access the vcluster\n", cmd.KubeConfig))
	}

	// wait for port-forwarding if necessary
	if cmd.portForwarding {
		if cmd.Server != "" {
			// Stop port-forwarding here
			close(cmd.interruptChan)
		}

		return <-cmd.errorChan
	}

	return nil
}

func (cmd *ConnectCmd) prepare(ctx context.Context, vCluster *find.VCluster) error {
	if cmd.LocalPort == 0 {
		cmd.LocalPort = clihelper.RandomPort()
	}

	var (
		kubeConfigLoader clientcmd.ClientConfig
		err              error
	)
	kubeConfigLoader = vCluster.ClientFactory
	cmd.Context = vCluster.Context
	cmd.Namespace = vCluster.Namespace
	restConfig, err := kubeConfigLoader.ClientConfig()
	if err != nil {
		return errors.Wrap(err, "load kube config")
	}
	kubeClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return errors.Wrap(err, "create kube client")
	}
	rawConfig, err := kubeConfigLoader.RawConfig()
	if err != nil {
		return errors.Wrap(err, "load raw config")
	}
	rawConfig.CurrentContext = cmd.Context

	cmd.kubeClient = kubeClient
	cmd.restConfig = restConfig
	cmd.kubeClientConfig = kubeConfigLoader
	cmd.rawConfig = rawConfig

	// set the namespace correctly
	if cmd.Namespace == "" {
		cmd.Namespace, _, err = kubeConfigLoader.Namespace()
		if err != nil {
			return err
		}
	}

	// resume vcluster if necessary
	if vCluster != nil && vCluster.Status == find.StatusPaused {
		cmd.Log.Infof("Resume vcluster %s...", vCluster.Name)
		err = lifecycle.ResumeVCluster(ctx, cmd.kubeClient, vCluster.Name, cmd.Namespace, cmd.Log)
		if err != nil {
			return errors.Wrap(err, "resume vcluster")
		}
	}

	return nil
}

func (cmd *ConnectCmd) getVClusterProKubeConfig(ctx context.Context, proClient proclient.Client, vCluster *procli.VirtualClusterInstanceProject) (*clientcmdapi.Config, error) {
	contextOptions, err := use.CreateVirtualClusterInstanceOptions(ctx, proClient, "", vCluster.Project.Name, vCluster.VirtualCluster, false, false, cmd.Log)
	if err != nil {
		return nil, fmt.Errorf("prepare vCluster kube config: %w", err)
	}

	// make sure access key is set
	if contextOptions.Token == "" && len(contextOptions.ClientCertificateData) == 0 && len(contextOptions.ClientKeyData) == 0 {
		contextOptions.Token = proClient.Config().AccessKey
	}

	// get current context
	rawConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{
		CurrentContext: cmd.Context,
	}).RawConfig()
	if err != nil {
		return nil, err
	}

	// make sure kube context name is set
	if cmd.KubeConfigContextName == "" {
		// use parent context if this is a vcluster context
		kubeContext := rawConfig.CurrentContext
		_, _, parentContext := find.VClusterProFromContext(kubeContext)
		if parentContext == "" {
			_, _, parentContext = find.VClusterFromContext(kubeContext)
		}
		if parentContext != "" {
			kubeContext = parentContext
		}
		cmd.KubeConfigContextName = find.VClusterProContextName(vCluster.VirtualCluster.Name, vCluster.Project.Name, kubeContext)
	}

	// set insecure true?
	if cmd.Insecure {
		contextOptions.InsecureSkipTLSVerify = true
	}

	// build kube config
	kubeConfig, err := clihelper.GetProKubeConfig(contextOptions)
	if err != nil {
		return nil, err
	}

	// we want to use a service account token in the kube config
	if cmd.ServiceAccount != "" {
		// check if its enabled on the pro vcluster
		if !vCluster.VirtualCluster.Status.VirtualCluster.ForwardToken {
			return nil, fmt.Errorf("forward token is not enabled on the vCluster and hence you cannot authenticate with a service account token")
		}

		// create service account token
		token, err := cmd.createServiceAccountToken(ctx, *kubeConfig)
		if err != nil {
			return nil, err
		}

		// set service account token
		for k := range kubeConfig.AuthInfos {
			kubeConfig.AuthInfos[k] = &clientcmdapi.AuthInfo{
				Token:                token,
				Extensions:           make(map[string]runtime.Object),
				ImpersonateUserExtra: make(map[string][]string),
			}
		}
	}

	return kubeConfig, nil
}

func (cmd *ConnectCmd) getVClusterKubeConfig(ctx context.Context, vclusterName string, command []string) (*clientcmdapi.Config, error) {
	var err error
	podName := cmd.PodName
	if podName == "" {
		waitErr := wait.PollUntilContextTimeout(ctx, time.Second, time.Second*30, true, func(ctx context.Context) (bool, error) {
			// get vcluster pod name
			var pods *corev1.PodList
			pods, err = cmd.kubeClient.CoreV1().Pods(cmd.Namespace).List(ctx, metav1.ListOptions{
				LabelSelector: "app=vcluster,release=" + vclusterName,
			})
			if err != nil {
				return false, err
			} else if len(pods.Items) == 0 {
				err = fmt.Errorf("can't find a running vcluster pod in namespace %s", cmd.Namespace)
				return false, nil
			}

			// sort by newest
			sort.Slice(pods.Items, func(i, j int) bool {
				return pods.Items[i].CreationTimestamp.Unix() > pods.Items[j].CreationTimestamp.Unix()
			})
			if pods.Items[0].DeletionTimestamp != nil {
				err = fmt.Errorf("can't find a running vcluster pod in namespace %s", cmd.Namespace)
				return false, nil
			}

			podName = pods.Items[0].Name
			return true, nil
		})
		if waitErr != nil {
			return nil, fmt.Errorf("finding vcluster pod: %w - %w", waitErr, err)
		}
	}

	// get the kube config from the Secret
	kubeConfig, err := clihelper.GetKubeConfig(ctx, cmd.kubeClient, vclusterName, cmd.Namespace, cmd.Log)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse kube config")
	}

	// find out port we should listen to locally
	if len(kubeConfig.Clusters) != 1 {
		return nil, fmt.Errorf("unexpected kube config")
	}

	// exchange context name in virtual kube config
	err = cmd.exchangeContextName(kubeConfig, vclusterName)
	if err != nil {
		return nil, err
	}

	// check if the vcluster is exposed and set server
	if vclusterName != "" && cmd.Server == "" && len(command) == 0 {
		err = cmd.setServerIfExposed(ctx, vclusterName, kubeConfig)
		if err != nil {
			return nil, err
		}
	}

	// find out vcluster server port
	port := "8443"
	for k := range kubeConfig.Clusters {
		if cmd.Insecure {
			kubeConfig.Clusters[k].CertificateAuthorityData = nil
			kubeConfig.Clusters[k].InsecureSkipTLSVerify = true
		}

		if cmd.Server != "" {
			if !strings.HasPrefix(cmd.Server, "https://") {
				cmd.Server = "https://" + cmd.Server
			}

			kubeConfig.Clusters[k].Server = cmd.Server
		} else {
			splitted := strings.Split(kubeConfig.Clusters[k].Server, ":")
			if len(splitted) != 3 {
				return nil, fmt.Errorf("unexpected server in kubeconfig: %s", kubeConfig.Clusters[k].Server)
			}

			port = splitted[2]
			splitted[2] = strconv.Itoa(cmd.LocalPort)
			kubeConfig.Clusters[k].Server = strings.Join(splitted, ":")
		}
	}

	// start port forwarding
	if cmd.ServiceAccount != "" || cmd.Server == "" || len(command) > 0 {
		cmd.portForwarding = true
		cmd.interruptChan = make(chan struct{})
		cmd.errorChan = make(chan error)

		// silence port-forwarding if a command is used
		stdout := io.Writer(os.Stdout)
		stderr := io.Writer(os.Stderr)
		if len(command) > 0 || cmd.BackgroundProxy {
			stdout = io.Discard
			stderr = io.Discard
		}

		go func() {
			cmd.errorChan <- portforward.StartPortForwardingWithRestart(ctx, cmd.restConfig, cmd.Address, podName, cmd.Namespace, strconv.Itoa(cmd.LocalPort), port, cmd.interruptChan, stdout, stderr, cmd.Log)
		}()
	}

	// we want to use a service account token in the kube config
	if cmd.ServiceAccount != "" {
		token, err := cmd.createServiceAccountToken(ctx, *kubeConfig)
		if err != nil {
			return nil, err
		}

		// set service account token
		for k := range kubeConfig.AuthInfos {
			kubeConfig.AuthInfos[k] = &clientcmdapi.AuthInfo{
				Token:                token,
				Extensions:           make(map[string]runtime.Object),
				ImpersonateUserExtra: make(map[string][]string),
			}
		}
	}

	return kubeConfig, nil
}

func (cmd *ConnectCmd) setServerIfExposed(ctx context.Context, vClusterName string, vClusterConfig *clientcmdapi.Config) error {
	printedWaiting := false
	err := wait.PollUntilContextTimeout(ctx, time.Second*2, time.Minute*5, true, func(ctx context.Context) (done bool, err error) {
		// first check for load balancer service, look for the other service if it's not there
		loadBalancerMissing := false
		service, err := cmd.kubeClient.CoreV1().Services(cmd.Namespace).Get(ctx, translate.GetLoadBalancerSVCName(vClusterName), metav1.GetOptions{})
		if err != nil {
			if kerrors.IsNotFound(err) {
				loadBalancerMissing = true
			} else {
				return false, err
			}
		}
		if loadBalancerMissing {
			service, err = cmd.kubeClient.CoreV1().Services(cmd.Namespace).Get(ctx, vClusterName, metav1.GetOptions{})
			if kerrors.IsNotFound(err) {
				return true, nil
			} else if err != nil {
				return false, err
			}
		}

		// not a load balancer? Then don't wait
		if service.Spec.Type == corev1.ServiceTypeNodePort {
			server, err := localkubernetes.ExposeLocal(ctx, vClusterName, cmd.Namespace, &cmd.rawConfig, vClusterConfig, service, cmd.LocalPort, cmd.Log)
			if err != nil {
				cmd.Log.Warnf("Error exposing local vcluster, will fallback to port-forwarding: %v", err)
			}

			cmd.Server = server
			return true, nil
		} else if service.Spec.Type != corev1.ServiceTypeLoadBalancer {
			return true, nil
		}

		if len(service.Status.LoadBalancer.Ingress) == 0 {
			if !printedWaiting {
				cmd.Log.Infof("Waiting for vcluster LoadBalancer ip...")
				printedWaiting = true
			}

			return false, nil
		}

		if service.Status.LoadBalancer.Ingress[0].Hostname != "" {
			cmd.Server = service.Status.LoadBalancer.Ingress[0].Hostname
		} else if service.Status.LoadBalancer.Ingress[0].IP != "" {
			cmd.Server = service.Status.LoadBalancer.Ingress[0].IP
		}

		if cmd.Server == "" {
			return false, nil
		}

		cmd.Log.Infof("Using vcluster %s load balancer endpoint: %s", vClusterName, cmd.Server)
		return true, nil
	})
	if err != nil {
		return errors.Wrap(err, "wait for vcluster")
	}

	return nil
}

// exchangeContextName switches the context name specified in the remote kubeconfig with
// the context name specified by the user. It cannot correctly handle kubeconfigs with multiple entries
// for clusters, authInfos, contexts, but ideally this is pointed at a secret created by us.
func (cmd *ConnectCmd) exchangeContextName(kubeConfig *clientcmdapi.Config, vclusterName string) error {
	if cmd.KubeConfigContextName == "" {
		if vclusterName != "" {
			cmd.KubeConfigContextName = find.VClusterContextName(vclusterName, cmd.Namespace, cmd.rawConfig.CurrentContext)
		} else {
			cmd.KubeConfigContextName = find.VClusterContextName(cmd.PodName, cmd.Namespace, cmd.rawConfig.CurrentContext)
		}
	}

	// pick the last specified cluster (there should ideally be exactly one)
	for k := range kubeConfig.Clusters {
		kubeConfig.Clusters[cmd.KubeConfigContextName] = kubeConfig.Clusters[k]
		// delete the rest
		if k != cmd.KubeConfigContextName {
			delete(kubeConfig.Clusters, k)
		}
		break
	}

	// pick the last specified context (there should ideally be exactly one)
	for k := range kubeConfig.Contexts {
		ctx := kubeConfig.Contexts[k]
		ctx.Cluster = cmd.KubeConfigContextName
		ctx.AuthInfo = cmd.KubeConfigContextName
		kubeConfig.Contexts[cmd.KubeConfigContextName] = ctx
		// delete the rest
		if k != cmd.KubeConfigContextName {
			delete(kubeConfig.Contexts, k)
		}
		break
	}

	// pick the last specified authinfo (there should ideally be exactly one)
	for k := range kubeConfig.AuthInfos {
		kubeConfig.AuthInfos[cmd.KubeConfigContextName] = kubeConfig.AuthInfos[k]
		// delete the rest
		if k != cmd.KubeConfigContextName {
			delete(kubeConfig.AuthInfos, k)
		}
		break
	}

	// update current-context
	kubeConfig.CurrentContext = cmd.KubeConfigContextName
	return nil
}

func (cmd *ConnectCmd) executeCommand(vKubeConfig clientcmdapi.Config, command []string) error {
	// convert to local kube config
	out, err := clientcmd.Write(vKubeConfig)
	if err != nil {
		return err
	}

	// write a temporary kube file
	tempFile, err := os.CreateTemp("", "")
	if err != nil {
		return errors.Wrap(err, "create temp file")
	}
	defer func(name string) {
		_ = os.Remove(name)
	}(tempFile.Name())

	_, err = tempFile.Write(out)
	if err != nil {
		return errors.Wrap(err, "write kube config to temp file")
	}

	err = tempFile.Close()
	if err != nil {
		return errors.Wrap(err, "close temp file")
	}

	commandErrChan := make(chan error)
	execCmd := exec.Command(command[0], command[1:]...)
	execCmd.Env = os.Environ()
	execCmd.Env = append(execCmd.Env, "KUBECONFIG="+tempFile.Name())
	execCmd.Stdout = os.Stdout
	execCmd.Stdin = os.Stdin
	execCmd.Stderr = os.Stderr
	err = execCmd.Start()
	if err != nil {
		return err
	}
	if cmd.errorChan == nil {
		return execCmd.Wait()
	}

	go func() {
		commandErrChan <- execCmd.Wait()
	}()

	select {
	case err := <-cmd.errorChan:
		if execCmd.Process != nil {
			_ = execCmd.Process.Kill()
		}

		return errors.Wrap(err, "error port-forwarding")
	case err := <-commandErrChan:
		if exitError, ok := lo.ErrorsAs[*exec.ExitError](err); ok {
			cmd.Log.Errorf("Error executing command: %v", err)
			os.Exit(exitError.ExitCode())
		}

		return err
	}
}

func (cmd *ConnectCmd) getLocalVClusterConfig(vKubeConfig clientcmdapi.Config) clientcmdapi.Config {
	// wait until we can access the virtual cluster
	vKubeConfig = *vKubeConfig.DeepCopy()
	for k := range vKubeConfig.Clusters {
		vKubeConfig.Clusters[k].Server = "https://localhost:" + strconv.Itoa(cmd.LocalPort)
	}
	return vKubeConfig
}

func (cmd *ConnectCmd) getLocalVClusterClient(vKubeConfig clientcmdapi.Config) (kubernetes.Interface, error) {
	vRestConfig, err := clientcmd.NewDefaultClientConfig(cmd.getLocalVClusterConfig(vKubeConfig), &clientcmd.ConfigOverrides{}).ClientConfig()
	if err != nil {
		return nil, errors.Wrap(err, "create virtual rest config")
	}

	vKubeClient, err := kubernetes.NewForConfig(vRestConfig)
	if err != nil {
		return nil, errors.Wrap(err, "create virtual kube client")
	}

	return vKubeClient, nil
}

func (cmd *ConnectCmd) waitForVCluster(ctx context.Context, vKubeConfig clientcmdapi.Config, errorChan chan error) error {
	vKubeClient, err := cmd.getLocalVClusterClient(vKubeConfig)
	if err != nil {
		return err
	}

	err = wait.PollUntilContextTimeout(ctx, time.Millisecond*200, time.Minute*3, true, func(ctx context.Context) (bool, error) {
		select {
		case err := <-errorChan:
			return false, err
		default:
			// check if service account exists
			_, err = vKubeClient.CoreV1().ServiceAccounts("default").Get(ctx, "default", metav1.GetOptions{})
			return err == nil, nil
		}
	})
	if err != nil {
		return errors.Wrap(err, "wait for vcluster to become ready")
	}

	return nil
}

func (cmd *ConnectCmd) createServiceAccountToken(ctx context.Context, vKubeConfig clientcmdapi.Config) (string, error) {
	vKubeClient, err := cmd.getLocalVClusterClient(vKubeConfig)
	if err != nil {
		return "", err
	}

	var (
		serviceAccount          = cmd.ServiceAccount
		serviceAccountNamespace = "kube-system"
	)
	if strings.Contains(cmd.ServiceAccount, "/") {
		splitted := strings.Split(cmd.ServiceAccount, "/")
		if len(splitted) != 2 {
			return "", fmt.Errorf("unexpected service account reference, expected ServiceAccountNamespace/ServiceAccountName")
		}

		serviceAccountNamespace = splitted[0]
		serviceAccount = splitted[1]
	}

	audiences := []string{"https://kubernetes.default.svc.cluster.local", "https://kubernetes.default.svc", "https://kubernetes.default"}
	expirationSeconds := int64(10 * 365 * 24 * 60 * 60)
	if cmd.ServiceAccountExpiration > 0 {
		expirationSeconds = int64(cmd.ServiceAccountExpiration)
	}
	token := ""
	cmd.Log.Infof("Create service account token for %s/%s", serviceAccountNamespace, serviceAccount)
	err = wait.PollUntilContextTimeout(ctx, time.Second, time.Minute*3, false, func(ctx context.Context) (bool, error) {
		// check if namespace exists
		_, err := vKubeClient.CoreV1().Namespaces().Get(ctx, serviceAccountNamespace, metav1.GetOptions{})
		if err != nil {
			if kerrors.IsNotFound(err) || kerrors.IsForbidden(err) {
				return false, err
			}

			return false, nil
		}

		// check if service account exists
		_, err = vKubeClient.CoreV1().ServiceAccounts(serviceAccountNamespace).Get(ctx, serviceAccount, metav1.GetOptions{})
		if err != nil {
			if kerrors.IsNotFound(err) {
				if serviceAccount == "default" {
					return false, nil
				}

				if cmd.ServiceAccountClusterRole != "" {
					// create service account
					_, err = vKubeClient.CoreV1().ServiceAccounts(serviceAccountNamespace).Create(ctx, &corev1.ServiceAccount{
						ObjectMeta: metav1.ObjectMeta{
							Name:      serviceAccount,
							Namespace: serviceAccountNamespace,
						},
					}, metav1.CreateOptions{})
					if err != nil {
						return false, err
					}

					cmd.Log.Donef("Created service account %s/%s", serviceAccountNamespace, serviceAccount)
				} else {
					return false, err
				}
			} else if kerrors.IsForbidden(err) {
				return false, err
			} else {
				return false, nil
			}
		}

		// create service account cluster role binding
		if cmd.ServiceAccountClusterRole != "" {
			clusterRoleBindingName := translate.SafeConcatName("vcluster", "sa", serviceAccount, serviceAccountNamespace)
			clusterRoleBinding, err := vKubeClient.RbacV1().ClusterRoleBindings().Get(ctx, clusterRoleBindingName, metav1.GetOptions{})
			if err != nil {
				if kerrors.IsNotFound(err) {
					// create cluster role binding
					_, err = vKubeClient.RbacV1().ClusterRoleBindings().Create(ctx, &rbacv1.ClusterRoleBinding{
						ObjectMeta: metav1.ObjectMeta{
							Name: clusterRoleBindingName,
						},
						RoleRef: rbacv1.RoleRef{
							APIGroup: rbacv1.SchemeGroupVersion.Group,
							Kind:     "ClusterRole",
							Name:     cmd.ServiceAccountClusterRole,
						},
						Subjects: []rbacv1.Subject{
							{
								Kind:      "ServiceAccount",
								Name:      serviceAccount,
								Namespace: serviceAccountNamespace,
							},
						},
					}, metav1.CreateOptions{})
					if err != nil {
						return false, err
					}

					cmd.Log.Donef("Created cluster role binding for cluster role %s", cmd.ServiceAccountClusterRole)
				} else if kerrors.IsForbidden(err) {
					return false, err
				} else {
					return false, nil
				}
			} else {
				// if cluster role differs, recreate it
				if clusterRoleBinding.RoleRef.Name != cmd.ServiceAccountClusterRole {
					err = vKubeClient.RbacV1().ClusterRoleBindings().Delete(ctx, clusterRoleBindingName, metav1.DeleteOptions{})
					if err != nil {
						return false, err
					}

					cmd.Log.Done("Recreate cluster role binding for service account")
					// this will recreate the cluster role binding in the next iteration
					return false, nil
				}
			}
		}

		// create service account token
		result, err := vKubeClient.CoreV1().ServiceAccounts(serviceAccountNamespace).CreateToken(ctx, serviceAccount, &authenticationv1.TokenRequest{Spec: authenticationv1.TokenRequestSpec{
			Audiences:         audiences,
			ExpirationSeconds: &expirationSeconds,
		}}, metav1.CreateOptions{})
		if err != nil {
			if kerrors.IsNotFound(err) || kerrors.IsForbidden(err) {
				return false, err
			}

			return false, nil
		}

		token = result.Status.Token
		return true, nil
	})
	if err != nil {
		return "", errors.Wrap(err, "create service account token")
	}

	return token, nil
}
