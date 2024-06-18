package cli

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

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/find"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/cli/localkubernetes"
	"github.com/loft-sh/vcluster/pkg/lifecycle"
	"github.com/loft-sh/vcluster/pkg/util/clihelper"
	"github.com/loft-sh/vcluster/pkg/util/portforward"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
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
)

type ConnectOptions struct {
	Driver string

	ServiceAccountClusterRole string
	PodName                   string
	Address                   string
	KubeConfigContextName     string
	Server                    string
	KubeConfig                string
	ServiceAccount            string
	LocalPort                 int
	ServiceAccountExpiration  int
	Print                     bool
	UpdateCurrent             bool
	BackgroundProxy           bool
	Insecure                  bool

	Project string
}

type connectHelm struct {
	*flags.GlobalFlags
	*ConnectOptions

	portForwarding   bool
	rawConfig        clientcmdapi.Config
	kubeClientConfig clientcmd.ClientConfig
	errorChan        chan error
	interruptChan    chan struct{}
	restConfig       *rest.Config
	kubeClient       *kubernetes.Clientset

	Log log.Logger
}

func ConnectHelm(ctx context.Context, options *ConnectOptions, globalFlags *flags.GlobalFlags, vClusterName string, command []string, log log.Logger) error {
	cmd := &connectHelm{
		GlobalFlags:    globalFlags,
		ConnectOptions: options,
		Log:            log,
	}

	// retrieve the vcluster
	vCluster, err := find.GetVCluster(ctx, cmd.Context, vClusterName, cmd.Namespace, cmd.Log)
	if err != nil {
		return err
	}

	return cmd.connect(ctx, vCluster, command)
}

func (cmd *connectHelm) connect(ctx context.Context, vCluster *find.VCluster, command []string) error {
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
		return executeCommand(getLocalVClusterConfig(*kubeConfig, cmd.ConnectOptions), command, cmd.errorChan, cmd.Log)
	}

	// write kube config
	err = writeKubeConfig(kubeConfig, vCluster.Name, cmd.ConnectOptions, cmd.GlobalFlags, cmd.portForwarding, cmd.Log)
	if err != nil {
		return err
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

func writeKubeConfig(kubeConfig *clientcmdapi.Config, vClusterName string, options *ConnectOptions, globalFlags *flags.GlobalFlags, portForwarding bool, log log.Logger) error {
	// write kube config to buffer
	out, err := clientcmd.Write(*kubeConfig)
	if err != nil {
		return err
	}

	// write kube config to file
	if options.Print {
		_, err = os.Stdout.Write(out)
		if err != nil {
			return err
		}
	} else if options.UpdateCurrent {
		var clusterConfig *clientcmdapi.Cluster
		for _, c := range kubeConfig.Clusters {
			clusterConfig = c
		}

		var authConfig *clientcmdapi.AuthInfo
		for _, a := range kubeConfig.AuthInfos {
			authConfig = a
		}

		err = clihelper.UpdateKubeConfig(options.KubeConfigContextName, clusterConfig, authConfig, true)
		if err != nil {
			return err
		}

		log.Donef("Switched active kube context to %s", options.KubeConfigContextName)
		if !options.BackgroundProxy && portForwarding {
			log.Warnf("Since you are using port-forwarding to connect, you will need to leave this terminal open")
			c := make(chan os.Signal, 1)
			signal.Notify(c, os.Interrupt, syscall.SIGTERM)
			go func() {
				<-c
				kubeConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{}).RawConfig()
				if err == nil && kubeConfig.CurrentContext == options.KubeConfigContextName {
					err = deleteContext(&kubeConfig, options.KubeConfigContextName, globalFlags.Context)
					if err != nil {
						log.Errorf("Error deleting context: %v", err)
					} else {
						log.Infof("Switched back to context %v", globalFlags.Context)
					}
				}
				os.Exit(1)
			}()

			defer func() {
				signal.Stop(c)
			}()
			log.WriteString(logrus.InfoLevel, "- Use CTRL+C to return to your previous kube context\n")
			log.WriteString(logrus.InfoLevel, "- Use `kubectl get namespaces` in another terminal to access the vcluster\n")
		} else {
			log.WriteString(logrus.InfoLevel, "- Use `vcluster disconnect` to return to your previous kube context\n")
			log.WriteString(logrus.InfoLevel, "- Use `kubectl get namespaces` to access the vcluster\n")
		}
	} else {
		err = os.WriteFile(options.KubeConfig, out, 0666)
		if err != nil {
			return fmt.Errorf("write kube config: %w", err)
		}

		log.Donef("Virtual cluster kube config written to: %s", options.KubeConfig)
		if options.Server == "" {
			log.WriteString(logrus.InfoLevel, fmt.Sprintf("- Use `vcluster connect %s -n %s -- kubectl get ns` to execute a command directly within this terminal\n", vClusterName, globalFlags.Namespace))
		}
		log.WriteString(logrus.InfoLevel, fmt.Sprintf("- Use `kubectl --kubeconfig %s get namespaces` to access the vcluster\n", options.KubeConfig))
	}

	return nil
}

func (cmd *connectHelm) prepare(ctx context.Context, vCluster *find.VCluster) error {
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
		return fmt.Errorf("load kube config: %w", err)
	}
	kubeClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return fmt.Errorf("create kube client: %w", err)
	}
	rawConfig, err := kubeConfigLoader.RawConfig()
	if err != nil {
		return fmt.Errorf("load raw config: %w", err)
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

	// resume vCluster if necessary
	if vCluster.Status == find.StatusPaused {
		cmd.Log.Infof("Resume vcluster %s...", vCluster.Name)
		err = lifecycle.ResumeVCluster(ctx, cmd.kubeClient, vCluster.Name, cmd.Namespace, cmd.Log)
		if err != nil {
			return fmt.Errorf("resume vcluster: %w", err)
		}
	}

	return nil
}

func (cmd *connectHelm) getVClusterKubeConfig(ctx context.Context, vclusterName string, command []string) (*clientcmdapi.Config, error) {
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
		return nil, fmt.Errorf("failed to parse kube config: %w", err)
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

		// check if we should start a background proxy
		if cmd.Server == "" && cmd.BackgroundProxy {
			if localkubernetes.IsDockerInstalledAndUpAndRunning() {
				// start background container
				server, err := localkubernetes.CreateBackgroundProxyContainer(ctx, vclusterName, cmd.Namespace, cmd.kubeClientConfig, kubeConfig, cmd.LocalPort, cmd.Log)
				if err != nil {
					cmd.Log.Warnf("Error exposing local vcluster, will fallback to port-forwarding: %v", err)
					cmd.BackgroundProxy = false
				}
				cmd.Server = server
			} else {
				cmd.Log.Debugf("Docker is not installed, so skip background proxy")
			}
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
		token, err := createServiceAccountToken(ctx, *kubeConfig, cmd.ConnectOptions, cmd.Log)
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

func (cmd *connectHelm) setServerIfExposed(ctx context.Context, vClusterName string, vClusterConfig *clientcmdapi.Config) error {
	printedWaiting := false
	err := wait.PollUntilContextTimeout(ctx, time.Second*2, time.Minute*5, true, func(ctx context.Context) (done bool, err error) {
		// first check for load balancer service, look for the other service if it's not there
		loadBalancerMissing := false
		service, err := cmd.kubeClient.CoreV1().Services(cmd.Namespace).Get(ctx, vClusterName, metav1.GetOptions{})
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
		return fmt.Errorf("wait for vcluster: %w", err)
	}

	return nil
}

// exchangeContextName switches the context name specified in the remote kubeconfig with
// the context name specified by the user. It cannot correctly handle kubeconfigs with multiple entries
// for clusters, authInfos, contexts, but ideally this is pointed at a secret created by us.
func (cmd *connectHelm) exchangeContextName(kubeConfig *clientcmdapi.Config, vclusterName string) error {
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

func executeCommand(vKubeConfig clientcmdapi.Config, command []string, errorChan chan error, log log.Logger) error {
	// convert to local kube config
	out, err := clientcmd.Write(vKubeConfig)
	if err != nil {
		return err
	}

	// write a temporary kube file
	tempFile, err := os.CreateTemp("", "")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer func(name string) {
		_ = os.Remove(name)
	}(tempFile.Name())

	_, err = tempFile.Write(out)
	if err != nil {
		return fmt.Errorf("write kube config to temp file: %w", err)
	}

	err = tempFile.Close()
	if err != nil {
		return fmt.Errorf("close temp file: %w", err)
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
	if errorChan == nil {
		return execCmd.Wait()
	}

	go func() {
		commandErrChan <- execCmd.Wait()
	}()

	select {
	case err := <-errorChan:
		if execCmd.Process != nil {
			_ = execCmd.Process.Kill()
		}

		return fmt.Errorf("error port-forwarding: %w", err)
	case err := <-commandErrChan:
		if exitError, ok := lo.ErrorsAs[*exec.ExitError](err); ok {
			log.Errorf("Error executing command: %v", err)
			os.Exit(exitError.ExitCode())
		}

		return err
	}
}

func getLocalVClusterConfig(vKubeConfig clientcmdapi.Config, options *ConnectOptions) clientcmdapi.Config {
	// wait until we can access the virtual cluster
	vKubeConfig = *vKubeConfig.DeepCopy()

	// update vCluster server address in case of OSS vClusters only
	if options.LocalPort != 0 {
		for k := range vKubeConfig.Clusters {
			vKubeConfig.Clusters[k].Server = "https://localhost:" + strconv.Itoa(options.LocalPort)
		}
	}

	return vKubeConfig
}

func getLocalVClusterClient(vKubeConfig clientcmdapi.Config, options *ConnectOptions) (kubernetes.Interface, error) {
	vRestConfig, err := clientcmd.NewDefaultClientConfig(getLocalVClusterConfig(vKubeConfig, options), &clientcmd.ConfigOverrides{}).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("create virtual rest config: %w", err)
	}

	vKubeClient, err := kubernetes.NewForConfig(vRestConfig)
	if err != nil {
		return nil, fmt.Errorf("create virtual kube client: %w", err)
	}

	return vKubeClient, nil
}

func (cmd *connectHelm) waitForVCluster(ctx context.Context, vKubeConfig clientcmdapi.Config, errorChan chan error) error {
	vKubeClient, err := getLocalVClusterClient(vKubeConfig, cmd.ConnectOptions)
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
		return fmt.Errorf("wait for vcluster to become ready: %w", err)
	}

	return nil
}

func createServiceAccountToken(ctx context.Context, vKubeConfig clientcmdapi.Config, options *ConnectOptions, log log.Logger) (string, error) {
	vKubeClient, err := getLocalVClusterClient(vKubeConfig, options)
	if err != nil {
		return "", err
	}

	var (
		serviceAccount          = options.ServiceAccount
		serviceAccountNamespace = "kube-system"
	)
	if strings.Contains(options.ServiceAccount, "/") {
		splitted := strings.Split(options.ServiceAccount, "/")
		if len(splitted) != 2 {
			return "", fmt.Errorf("unexpected service account reference, expected ServiceAccountNamespace/ServiceAccountName")
		}

		serviceAccountNamespace = splitted[0]
		serviceAccount = splitted[1]
	}

	audiences := []string{"https://kubernetes.default.svc.cluster.local", "https://kubernetes.default.svc", "https://kubernetes.default"}
	expirationSeconds := int64(10 * 365 * 24 * 60 * 60)
	if options.ServiceAccountExpiration > 0 {
		expirationSeconds = int64(options.ServiceAccountExpiration)
	}
	token := ""
	log.Infof("Create service account token for %s/%s", serviceAccountNamespace, serviceAccount)
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

				if options.ServiceAccountClusterRole != "" {
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

					log.Donef("Created service account %s/%s", serviceAccountNamespace, serviceAccount)
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
		if options.ServiceAccountClusterRole != "" {
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
							Name:     options.ServiceAccountClusterRole,
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

					log.Donef("Created cluster role binding for cluster role %s", options.ServiceAccountClusterRole)
				} else if kerrors.IsForbidden(err) {
					return false, err
				} else {
					return false, nil
				}
			} else {
				// if cluster role differs, recreate it
				if clusterRoleBinding.RoleRef.Name != options.ServiceAccountClusterRole {
					err = vKubeClient.RbacV1().ClusterRoleBindings().Delete(ctx, clusterRoleBindingName, metav1.DeleteOptions{})
					if err != nil {
						return false, err
					}

					log.Done("Recreate cluster role binding for service account")
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
		return "", fmt.Errorf("create service account token: %w", err)
	}

	return token, nil
}
