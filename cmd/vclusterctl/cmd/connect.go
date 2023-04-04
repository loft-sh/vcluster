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

	"github.com/pkg/errors"
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
	"k8s.io/client-go/tools/clientcmd/api"

	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/app/localkubernetes"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/find"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/flags"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/log"
	"github.com/loft-sh/vcluster/pkg/lifecycle"
	"github.com/loft-sh/vcluster/pkg/upgrade"
	"github.com/loft-sh/vcluster/pkg/util/portforward"
	"github.com/loft-sh/vcluster/pkg/util/translate"
)

// ConnectCmd holds the cmd flags
type ConnectCmd struct {
	*flags.GlobalFlags

	KubeConfigContextName string
	KubeConfig            string
	PodName               string
	UpdateCurrent         bool
	Print                 bool
	BackgroundProxy       bool
	LocalPort             int
	Address               string

	ServiceAccount            string
	ServiceAccountClusterRole string
	ServiceAccountExpiration  int

	Server   string
	Insecure bool

	Log log.Logger

	kubeClientConfig clientcmd.ClientConfig
	kubeClient       *kubernetes.Clientset
	restConfig       *rest.Config
	rawConfig        api.Config

	portForwarding bool
	interruptChan  chan struct{}
	errorChan      chan error
}

// NewConnectCmd creates a new command
func NewConnectCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &ConnectCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}

	cobraCmd := &cobra.Command{
		Use:   "connect [flags] vcluster_name",
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
		Args:              cobra.MinimumNArgs(1),
		ValidArgsFunction: newValidVClusterNameFunc(globalFlags),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			// Check for newer version
			upgrade.PrintNewerVersionWarning()

			return cmd.Run(args)
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
	return cobraCmd
}

// Run executes the functionality
func (cmd *ConnectCmd) Run(args []string) error {
	vclusterName := ""
	if len(args) > 0 {
		vclusterName = args[0]
	}

	return cmd.Connect(vclusterName, args[1:])
}

func (cmd *ConnectCmd) Connect(vclusterName string, command []string) error {
	if vclusterName == "" && cmd.PodName == "" {
		return fmt.Errorf("please specify either --pod or a name for the vcluster")
	}

	// prepare clients and find vcluster
	err := cmd.prepare(vclusterName)
	if err != nil {
		return err
	}

	// retrieve vcluster kube config
	kubeConfig, err := cmd.getVClusterKubeConfig(vclusterName, command)
	if err != nil {
		return err
	}

	if len(command) == 0 && cmd.ServiceAccount == "" && cmd.Server == "" && cmd.BackgroundProxy && localkubernetes.IsDockerInstalledAndUpAndRunning() {
		// start background container
		server, err := localkubernetes.CreateBackgroundProxyContainer(vclusterName, cmd.Namespace, &cmd.rawConfig, kubeConfig, cmd.LocalPort, cmd.Log)
		if err != nil {
			cmd.Log.Warnf("Error exposing local vcluster, will fallback to port-forwarding: %v", err)
			cmd.BackgroundProxy = false
		}
		cmd.Server = server
	}

	// check if we should execute command
	if len(command) > 0 {
		return cmd.executeCommand(*kubeConfig, command)
	}

	// write kube config to buffer
	out, err := clientcmd.Write(*kubeConfig)
	if err != nil {
		return err
	}

	// write kube config to file
	if cmd.UpdateCurrent {
		var clusterConfig *api.Cluster
		for _, c := range kubeConfig.Clusters {
			clusterConfig = c
		}

		var authConfig *api.AuthInfo
		for _, a := range kubeConfig.AuthInfos {
			authConfig = a
		}

		err = updateKubeConfig(cmd.KubeConfigContextName, clusterConfig, authConfig, true)
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
			cmd.Log.WriteString("- Use CTRL+C to return to your previous kube context\n")
			cmd.Log.WriteString("- Use `kubectl get namespaces` in another terminal to access the vcluster\n")
		} else {
			cmd.Log.WriteString("- Use `vcluster disconnect` to return to your previous kube context\n")
			cmd.Log.WriteString("- Use `kubectl get namespaces` to access the vcluster\n")
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
			cmd.Log.WriteString(fmt.Sprintf("- Use `vcluster connect %s -n %s -- kubectl get ns` to execute a command directly within this terminal\n", vclusterName, cmd.Namespace))
		}
		cmd.Log.WriteString(fmt.Sprintf("- Use `kubectl --kubeconfig %s get namespaces` to access the vcluster\n", cmd.KubeConfig))
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

func (cmd *ConnectCmd) prepare(vclusterName string) error {
	if cmd.LocalPort == 0 {
		cmd.LocalPort = randomPort()
	}

	if cmd.ServiceAccountClusterRole != "" && cmd.ServiceAccount == "" {
		return fmt.Errorf("expected --service-account to be defined as well")
	}

	var (
		kubeConfigLoader clientcmd.ClientConfig
		vCluster         *find.VCluster
		err              error
	)
	if vclusterName != "" {
		vCluster, err = find.GetVCluster(cmd.Context, vclusterName, cmd.Namespace)
		if err != nil {
			return err
		}

		kubeConfigLoader = vCluster.ClientFactory
		cmd.Context = vCluster.Context
		cmd.Namespace = vCluster.Namespace
	} else {
		kubeConfigLoader = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{
			CurrentContext: cmd.Context,
		})
	}

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
		err = lifecycle.ResumeVCluster(cmd.kubeClient, vclusterName, cmd.Namespace, cmd.Log)
		if err != nil {
			return errors.Wrap(err, "resume vcluster")
		}
	}

	return nil
}

func (cmd *ConnectCmd) getVClusterKubeConfig(vclusterName string, command []string) (*api.Config, error) {
	var err error
	podName := cmd.PodName
	if podName == "" {
		waitErr := wait.PollImmediate(time.Second, time.Second*6, func() (bool, error) {
			// get vcluster pod name
			var pods *corev1.PodList
			pods, err = cmd.kubeClient.CoreV1().Pods(cmd.Namespace).List(context.Background(), metav1.ListOptions{
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
			return nil, fmt.Errorf("finding vcluster pod: %v - %v", waitErr, err)
		}
	}

	// get the kube config from the Secret
	kubeConfig, err := GetKubeConfig(context.Background(), cmd.kubeClient, vclusterName, cmd.Namespace, cmd.Log)
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
		err = cmd.setServerIfExposed(vclusterName, kubeConfig)
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
			cmd.errorChan <- portforward.StartPortForwardingWithRestart(cmd.restConfig, cmd.Address, podName, cmd.Namespace, strconv.Itoa(cmd.LocalPort), port, cmd.interruptChan, stdout, stderr, cmd.Log)
		}()
	}

	// we want to use a service account token in the kube config
	if cmd.ServiceAccount != "" {
		token, err := cmd.createServiceAccountToken(*kubeConfig)
		if err != nil {
			return nil, err
		}

		// set service account token
		for k := range kubeConfig.AuthInfos {
			kubeConfig.AuthInfos[k] = &api.AuthInfo{
				Token:                token,
				Extensions:           make(map[string]runtime.Object),
				ImpersonateUserExtra: make(map[string][]string),
			}
		}
	}

	return kubeConfig, nil
}

func (cmd *ConnectCmd) setServerIfExposed(vClusterName string, vClusterConfig *api.Config) error {
	printedWaiting := false
	err := wait.PollImmediate(time.Second*2, time.Minute*5, func() (done bool, err error) {
		// first check for load balancer service, look for the other service if it's not there
		loadBalancerMissing := false
		service, err := cmd.kubeClient.CoreV1().Services(cmd.Namespace).Get(context.TODO(), translate.GetLoadBalancerSVCName(vClusterName), metav1.GetOptions{})
		if err != nil {
			if kerrors.IsNotFound(err) {
				loadBalancerMissing = true
			} else {
				return false, err

			}
		}
		if loadBalancerMissing {
			service, err = cmd.kubeClient.CoreV1().Services(cmd.Namespace).Get(context.TODO(), vClusterName, metav1.GetOptions{})
			if kerrors.IsNotFound(err) {
				return true, nil
			} else if err != nil {
				return false, err
			}

		}

		// not a load balancer? Then don't wait
		if service.Spec.Type == corev1.ServiceTypeNodePort {
			server, err := localkubernetes.ExposeLocal(vClusterName, cmd.Namespace, &cmd.rawConfig, vClusterConfig, service, cmd.LocalPort, cmd.Log)
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

func (cmd *ConnectCmd) exchangeContextName(kubeConfig *api.Config, vclusterName string) error {
	if cmd.KubeConfigContextName == "" {
		if vclusterName != "" {
			cmd.KubeConfigContextName = find.VClusterContextName(vclusterName, cmd.Namespace, cmd.rawConfig.CurrentContext)
		} else {
			cmd.KubeConfigContextName = find.VClusterContextName(cmd.PodName, cmd.Namespace, cmd.rawConfig.CurrentContext)
		}
	}

	// update cluster
	for k := range kubeConfig.Clusters {
		kubeConfig.Clusters[cmd.KubeConfigContextName] = kubeConfig.Clusters[k]
		delete(kubeConfig.Clusters, k)
		break
	}

	// update context
	for k := range kubeConfig.Contexts {
		ctx := kubeConfig.Contexts[k]
		ctx.Cluster = cmd.KubeConfigContextName
		ctx.AuthInfo = cmd.KubeConfigContextName
		kubeConfig.Contexts[cmd.KubeConfigContextName] = ctx
		delete(kubeConfig.Contexts, k)
		break
	}

	// update authInfo
	for k := range kubeConfig.AuthInfos {
		kubeConfig.AuthInfos[cmd.KubeConfigContextName] = kubeConfig.AuthInfos[k]
		delete(kubeConfig.AuthInfos, k)
		break
	}

	// update current-context
	kubeConfig.CurrentContext = cmd.KubeConfigContextName
	return nil
}

func (cmd *ConnectCmd) executeCommand(vKubeConfig api.Config, command []string) error {
	if !cmd.portForwarding {
		return fmt.Errorf("command is specified, but port-forwarding isn't started")
	}

	defer close(cmd.interruptChan)

	// wait for vcluster to be ready
	err := cmd.waitForVCluster(vKubeConfig, cmd.errorChan)
	if err != nil {
		return err
	}

	// convert to local kube config
	vKubeConfig = cmd.getLocalVClusterConfig(vKubeConfig)
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
		if exitError, ok := err.(*exec.ExitError); ok {
			cmd.Log.Errorf("Error executing command: %v", err)
			os.Exit(exitError.ExitCode())
		}

		return err
	}
}

func (cmd *ConnectCmd) getLocalVClusterConfig(vKubeConfig api.Config) api.Config {
	// wait until we can access the virtual cluster
	vKubeConfig = *vKubeConfig.DeepCopy()
	for k := range vKubeConfig.Clusters {
		vKubeConfig.Clusters[k].Server = "https://localhost:" + strconv.Itoa(cmd.LocalPort)
	}
	return vKubeConfig
}

func (cmd *ConnectCmd) getLocalVClusterClient(vKubeConfig api.Config) (kubernetes.Interface, error) {
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

func (cmd *ConnectCmd) waitForVCluster(vKubeConfig api.Config, errorChan chan error) error {
	vKubeClient, err := cmd.getLocalVClusterClient(vKubeConfig)
	if err != nil {
		return err
	}

	err = wait.PollImmediate(time.Millisecond*200, time.Minute*3, func() (bool, error) {
		select {
		case err := <-errorChan:
			return false, err
		default:
			// check if service account exists
			_, err = vKubeClient.CoreV1().ServiceAccounts("default").Get(context.TODO(), "default", metav1.GetOptions{})
			return err == nil, nil
		}
	})
	if err != nil {
		return errors.Wrap(err, "wait for vcluster to become ready")
	}

	return nil
}

func (cmd *ConnectCmd) createServiceAccountToken(vKubeConfig api.Config) (string, error) {
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
	err = wait.Poll(time.Second, time.Minute*3, func() (bool, error) {
		// check if namespace exists
		_, err := vKubeClient.CoreV1().Namespaces().Get(context.TODO(), serviceAccountNamespace, metav1.GetOptions{})
		if err != nil {
			if kerrors.IsNotFound(err) || kerrors.IsForbidden(err) {
				return false, err
			}

			return false, nil
		}

		// check if service account exists
		_, err = vKubeClient.CoreV1().ServiceAccounts(serviceAccountNamespace).Get(context.TODO(), serviceAccount, metav1.GetOptions{})
		if err != nil {
			if kerrors.IsNotFound(err) {
				if serviceAccount == "default" {
					return false, nil
				}

				if cmd.ServiceAccountClusterRole != "" {
					// create service account
					_, err = vKubeClient.CoreV1().ServiceAccounts(serviceAccountNamespace).Create(context.TODO(), &corev1.ServiceAccount{
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
			clusterRoleBinding, err := vKubeClient.RbacV1().ClusterRoleBindings().Get(context.TODO(), clusterRoleBindingName, metav1.GetOptions{})
			if err != nil {
				if kerrors.IsNotFound(err) {
					// create cluster role binding
					_, err = vKubeClient.RbacV1().ClusterRoleBindings().Create(context.TODO(), &rbacv1.ClusterRoleBinding{
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
					err = vKubeClient.RbacV1().ClusterRoleBindings().Delete(context.TODO(), clusterRoleBindingName, metav1.DeleteOptions{})
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
		result, err := vKubeClient.CoreV1().ServiceAccounts(serviceAccountNamespace).CreateToken(context.TODO(), serviceAccount, &authenticationv1.TokenRequest{Spec: authenticationv1.TokenRequestSpec{
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
