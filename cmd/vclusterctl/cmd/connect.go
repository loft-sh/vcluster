package cmd

import (
	"context"
	"fmt"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"io"
	"io/ioutil"
	authenticationv1 "k8s.io/api/authentication/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/loft-sh/vcluster/pkg/upgrade"
	"github.com/loft-sh/vcluster/pkg/util/kubeconfig"
	"github.com/loft-sh/vcluster/pkg/util/podhelper"
	"github.com/loft-sh/vcluster/pkg/util/portforward"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/loft-sh/vcluster/cmd/vclusterctl/flags"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

// ConnectCmd holds the cmd flags
type ConnectCmd struct {
	*flags.GlobalFlags

	KubeConfig    string
	PodName       string
	UpdateCurrent bool
	Print         bool
	LocalPort     int
	Address       string

	ServiceAccount            string
	ServiceAccountClusterRole string
	ServiceAccountExpiration  int

	Server   string
	Insecure bool

	Log log.Logger
}

// NewConnectCmd creates a new command
func NewConnectCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &ConnectCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}

	cobraCmd := &cobra.Command{
		Use:   "connect",
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
		Args: cobra.MinimumNArgs(1),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			// Check for newer version
			upgrade.PrintNewerVersionWarning()

			return cmd.Run(args)
		},
	}

	cobraCmd.Flags().StringVar(&cmd.KubeConfig, "kube-config", "./kubeconfig.yaml", "Writes the created kube config to this file")
	cobraCmd.Flags().BoolVar(&cmd.UpdateCurrent, "update-current", false, "If true updates the current kube config")
	cobraCmd.Flags().BoolVar(&cmd.Print, "print", false, "When enabled prints the context to stdout")
	cobraCmd.Flags().StringVar(&cmd.PodName, "pod", "", "The pod to connect to")
	cobraCmd.Flags().StringVar(&cmd.Server, "server", "", "The server to connect to")
	cobraCmd.Flags().IntVar(&cmd.LocalPort, "local-port", 0, "The local port to forward the virtual cluster to. If empty, vcluster will use a random unused port")
	cobraCmd.Flags().StringVar(&cmd.Address, "address", "", "The local address to start port forwarding under")
	cobraCmd.Flags().StringVar(&cmd.ServiceAccount, "service-account", "", "If specified, vcluster will create a service account token to connect to the virtual cluster instead of using the default client cert / key. Service account must exist and can be used as namespace/name.")
	cobraCmd.Flags().StringVar(&cmd.ServiceAccountClusterRole, "cluster-role", "", "If specified, vcluster will create the service account if it does not exist and also add a cluster role binding for the given cluster role to it. Requires --service-account to be set")
	cobraCmd.Flags().IntVar(&cmd.ServiceAccountExpiration, "token-expiration", 0, "If specified, vcluster will create the service account token for the given duration in seconds. Defaults to eternal")
	cobraCmd.Flags().BoolVar(&cmd.Insecure, "insecure", false, "If specified, vcluster will create the kube config with insecure-skip-tls-verify")
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

func randomPort() int {
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < 10; i++ {
		port := 10000 + rand.Intn(3000)
		s, err := checkPort(port)
		if s && err == nil {
			return port
		}
	}

	// just try another port
	return 10000 + rand.Intn(3000)
}

func checkPort(port int) (status bool, err error) {
	// Concatenate a colon and the port
	host := "localhost:" + strconv.Itoa(port)

	// Try to create a server with the port
	server, err := net.Listen("tcp", host)

	// if it fails then the port is likely taken
	if err != nil {
		return false, err
	}

	// close the server
	_ = server.Close()

	// we successfully used and closed the port
	// so it's now available to be used again
	return true, nil
}

func (cmd *ConnectCmd) Connect(vclusterName string, command []string) error {
	if cmd.LocalPort == 0 {
		cmd.LocalPort = randomPort()
	}

	if cmd.ServiceAccountClusterRole != "" && cmd.ServiceAccount == "" {
		return fmt.Errorf("expected --service-account to be defined as well")
	}

	kubeConfigLoader := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{
		CurrentContext: cmd.Context,
	})
	restConfig, err := kubeConfigLoader.ClientConfig()
	if err != nil {
		return errors.Wrap(err, "load kube config")
	}
	kubeClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return errors.Wrap(err, "create kube client")
	}

	// set the namespace correctly
	if cmd.Namespace == "" {
		cmd.Namespace, _, err = kubeConfigLoader.Namespace()
		if err != nil {
			return err
		}
	}

	if vclusterName == "" && cmd.PodName == "" {
		return fmt.Errorf("please specify either --pod or a name for the vcluster")
	}

	podName := cmd.PodName
	if podName == "" {
		waitErr := wait.PollImmediate(time.Second, time.Second*6, func() (bool, error) {
			// get vcluster pod name
			var pods *corev1.PodList
			pods, err = kubeClient.CoreV1().Pods(cmd.Namespace).List(context.Background(), metav1.ListOptions{
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
			return fmt.Errorf("finding vcluster pod: %v - %v", waitErr, err)
		}
	}

	// get the kube config from the Secret
	kubeConfig, err := GetKubeConfig(context.Background(), kubeClient, vclusterName, restConfig, podName, cmd.Namespace, cmd.Log)
	if err != nil {
		return errors.Wrap(err, "failed to parse kube config")
	}

	// find out port we should listen to locally
	if len(kubeConfig.Clusters) != 1 {
		return fmt.Errorf("unexpected kube config")
	}

	// check if the vcluster is exposed
	if vclusterName != "" && cmd.Server == "" && len(command) == 0 {
		printedWaiting := false
		err = wait.PollImmediate(time.Second*2, time.Minute*5, func() (done bool, err error) {
			service, err := kubeClient.CoreV1().Services(cmd.Namespace).Get(context.TODO(), vclusterName, metav1.GetOptions{})
			if err != nil {
				if kerrors.IsNotFound(err) {
					return true, nil
				}

				return false, err
			}

			// not a load balancer? Then don't wait
			if service.Spec.Type != corev1.ServiceTypeLoadBalancer {
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

			cmd.Log.Infof("Using vcluster %s load balancer endpoint: %s", vclusterName, cmd.Server)
			return true, nil
		})
		if err != nil {
			return errors.Wrap(err, "wait for vcluster")
		}
	}

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
				return fmt.Errorf("unexpected server in kubeconfig: %s", kubeConfig.Clusters[k].Server)
			}

			port = splitted[2]
			splitted[2] = strconv.Itoa(cmd.LocalPort)
			kubeConfig.Clusters[k].Server = strings.Join(splitted, ":")
		}
	}

	// start port forwarding
	var (
		interruptChan chan struct{}
		errorChan     chan error
	)
	if cmd.ServiceAccount != "" || cmd.Server == "" || len(command) > 0 {
		interruptChan = make(chan struct{})
		errorChan = make(chan error)

		// silence port-forwarding if a command is used
		stdout := io.Writer(os.Stdout)
		stderr := io.Writer(os.Stderr)
		if len(command) > 0 {
			stdout = ioutil.Discard
			stderr = ioutil.Discard
		}

		go func() {
			errorChan <- portforward.StartPortForwardingWithRestart(restConfig, cmd.Address, podName, cmd.Namespace, strconv.Itoa(cmd.LocalPort), port, interruptChan, stdout, stderr, cmd.Log)
		}()
	}

	// we want to use a service account token in the kube config
	if cmd.ServiceAccount != "" {
		token, err := cmd.createServiceAccountToken(*kubeConfig)
		if err != nil {
			return err
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

	// check if we should execute command
	if len(command) > 0 {
		return cmd.executeCommand(*kubeConfig, command, interruptChan, errorChan)
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

		contextName := ""
		if vclusterName != "" {
			contextName = "vcluster_" + cmd.Namespace + "_" + vclusterName
		} else {
			contextName = "vcluster_" + cmd.Namespace + "_" + cmd.PodName
		}
		err = updateKubeConfig(contextName, clusterConfig, authConfig, false)
		if err != nil {
			return err
		}

		cmd.Log.Donef("Successfully created kube context %s. You can access the vcluster with `kubectl get namespaces --context %s`", contextName, contextName)
	} else if cmd.Print {
		_, err = os.Stdout.Write(out)
		if err != nil {
			return err
		}
	} else {
		err = ioutil.WriteFile(cmd.KubeConfig, out, 0666)
		if err != nil {
			return errors.Wrap(err, "write kube config")
		}

		cmd.Log.Infof("Use `vcluster connect %s -n %s -- kubectl get ns` to execute a command directly within this terminal", vclusterName, cmd.Namespace)
		cmd.Log.Donef("Virtual cluster kube config written to: %s. You can access the cluster via `kubectl --kubeconfig %s get namespaces`", cmd.KubeConfig, cmd.KubeConfig)
	}

	if cmd.Server != "" {
		if interruptChan == nil {
			return nil
		}

		// Stop port-forwarding here
		close(interruptChan)
		return <-errorChan
	}

	return <-errorChan
}

func (cmd *ConnectCmd) executeCommand(vKubeConfig api.Config, command []string, interruptChan chan struct{}, errorChan chan error) error {
	defer close(interruptChan)

	// wait for vcluster to be ready
	err := cmd.waitForVCluster(vKubeConfig, errorChan)
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
	tempFile, err := ioutil.TempFile("", "")
	if err != nil {
		return errors.Wrap(err, "create temp file")
	}
	defer os.Remove(tempFile.Name())

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
	case err := <-errorChan:
		if execCmd.Process != nil {
			_ = execCmd.Process.Kill()
		}

		return errors.Wrap(err, "error port-forwarding")
	case err := <-commandErrChan:
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
			return "", fmt.Errorf("unexpected service account reference, expected ServiceAccountNamspace/ServiceAccountName")
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

func updateKubeConfig(contextName string, cluster *api.Cluster, authInfo *api.AuthInfo, setActive bool) error {
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{}).RawConfig()
	if err != nil {
		return err
	}

	config.Clusters[contextName] = cluster
	config.AuthInfos[contextName] = authInfo

	// Update kube context
	context := api.NewContext()
	context.Cluster = contextName
	context.AuthInfo = contextName

	config.Contexts[contextName] = context
	if setActive {
		config.CurrentContext = contextName
	}

	// Save the config
	return clientcmd.ModifyConfig(clientcmd.NewDefaultClientConfigLoadingRules(), config, false)
}

// GetKubeConfig attempts to read the kubeconfig from the default Secret and
// falls back to reading from filesystem if the Secret is not read successfully.
// Reading from filesystem is implemented for the backward compatibility and
// can be eventually removed in the future.
//
// This is retried until the kube config is successfully retrieve, or until 10 minute timeout is reached.
func GetKubeConfig(ctx context.Context, kubeClient *kubernetes.Clientset, vclusterName string, restConfig *rest.Config, podName, namespace string, log log.Logger) (*api.Config, error) {
	var kubeConfig *api.Config

	printedWaiting := false
	err := wait.PollImmediate(time.Second*2, time.Minute*10, func() (done bool, err error) {
		kubeConfig, err = kubeconfig.ReadKubeConfig(ctx, kubeClient, vclusterName, namespace)
		if err != nil {
			// try to obtain the kube config the old way
			stdout, _, err := podhelper.ExecBuffered(restConfig, namespace, podName, "syncer", []string{"cat", "/root/.kube/config"}, nil)
			if err != nil {
				if !printedWaiting {
					log.Infof("Waiting for vcluster to come up...")
					printedWaiting = true
				}
				return false, nil
			}

			kubeConfig, err = clientcmd.Load(stdout)
			if err != nil {
				return false, fmt.Errorf("failed to parse kube config: %v", err)
			}
			log.Info("Falling back to reading the kube config from the syncer pod.")
			return true, nil

		}
		return true, nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "wait for vcluster")
	}

	return kubeConfig, nil
}
