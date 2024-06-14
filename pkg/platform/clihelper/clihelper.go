package clihelper

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	jsonpatch "github.com/evanphx/json-patch"
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	loftclientset "github.com/loft-sh/api/v4/pkg/clientset/versioned"
	"github.com/loft-sh/api/v4/pkg/product"
	"github.com/loft-sh/log"
	"github.com/loft-sh/log/survey"
	"github.com/loft-sh/vcluster/pkg/platform/kubeconfig"
	utilhttp "github.com/loft-sh/vcluster/pkg/util/http"
	"github.com/loft-sh/vcluster/pkg/util/portforward"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/transport/spdy"
	"k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
)

// CriticalStatus container status
var CriticalStatus = map[string]bool{
	"Error":                      true,
	"Unknown":                    true,
	"ImagePullBackOff":           true,
	"CrashLoopBackOff":           true,
	"RunContainerError":          true,
	"ErrImagePull":               true,
	"CreateContainerConfigError": true,
	"InvalidImageName":           true,
}

const defaultReleaseName = "loft"

const LoftRouterDomainSecret = "loft-router-domain"

const defaultTimeout = 10 * time.Minute

const timeoutEnvVariable = "LOFT_TIMEOUT"

var defaultDeploymentName = "loft"

func Timeout() time.Duration {
	if timeout := os.Getenv(timeoutEnvVariable); timeout != "" {
		if parsedTimeout, err := time.ParseDuration(timeout); err == nil {
			return parsedTimeout
		}
	}

	return defaultTimeout
}

func GetDisplayName(name string, displayName string) string {
	if displayName != "" {
		return displayName
	}

	return name
}

func GetTableDisplayName(name string, displayName string) string {
	if displayName != "" && displayName != name {
		return displayName + " (" + name + ")"
	}

	return name
}

func DisplayName(entityInfo *storagev1.EntityInfo) string {
	if entityInfo == nil {
		return ""
	} else if entityInfo.DisplayName != "" {
		return entityInfo.DisplayName
	} else if entityInfo.Username != "" {
		return entityInfo.Username
	}

	return entityInfo.Name
}

// GetProKubeConfig builds a pro kube config from options and client
func GetProKubeConfig(options kubeconfig.ContextOptions) (*clientcmdapi.Config, error) {
	contextName := options.Name
	cluster := clientcmdapi.NewCluster()
	cluster.Server = options.Server
	cluster.CertificateAuthorityData = options.CaData
	cluster.InsecureSkipTLSVerify = options.InsecureSkipTLSVerify

	authInfo := clientcmdapi.NewAuthInfo()
	if options.Token != "" || options.ClientCertificateData != nil || options.ClientKeyData != nil {
		authInfo.Token = options.Token
		authInfo.ClientKeyData = options.ClientKeyData
		authInfo.ClientCertificateData = options.ClientCertificateData
	}

	config := clientcmdapi.NewConfig()
	config.Clusters[contextName] = cluster
	config.AuthInfos[contextName] = authInfo

	// Update kube context
	kubeContext := clientcmdapi.NewContext()
	kubeContext.Cluster = contextName
	kubeContext.AuthInfo = contextName
	kubeContext.Namespace = options.CurrentNamespace

	config.Contexts[contextName] = kubeContext
	config.CurrentContext = contextName

	// set kind & version
	config.APIVersion = "v1"
	config.Kind = "Config"
	return config, nil
}

func GetLoftIngressHost(ctx context.Context, kubeClient kubernetes.Interface, namespace string) (string, error) {
	ingress, err := kubeClient.NetworkingV1().Ingresses(namespace).Get(ctx, "loft-ingress", metav1.GetOptions{})
	if err != nil {
		ingress, err := kubeClient.NetworkingV1beta1().Ingresses(namespace).Get(ctx, "loft-ingress", metav1.GetOptions{})
		if err != nil {
			return "", err
		}
		// find host
		for _, rule := range ingress.Spec.Rules {
			return rule.Host, nil
		}
	} else {
		// find host
		for _, rule := range ingress.Spec.Rules {
			return rule.Host, nil
		}
	}

	return "", fmt.Errorf("couldn't find any host in loft ingress '%s/loft-ingress', please make sure you have not changed any deployed resources", namespace)
}

func WaitForReadyLoftPod(ctx context.Context, kubeClient kubernetes.Interface, namespace string, log log.Logger) (*corev1.Pod, error) {
	// wait until we have a running loft pod
	now := time.Now()
	pod := &corev1.Pod{}
	err := wait.PollUntilContextTimeout(ctx, time.Second*2, Timeout(), true, func(ctx context.Context) (bool, error) {
		pods, err := kubeClient.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
			LabelSelector: "app=loft",
		})
		if err != nil {
			log.Warnf("Error trying to retrieve %s pod: %v", product.DisplayName(), err)
			return false, nil
		} else if len(pods.Items) == 0 {
			if time.Now().After(now.Add(time.Second * 10)) {
				log.Infof("Still waiting for a %s pod...", product.DisplayName())
				now = time.Now()
			}
			return false, nil
		}

		sort.Slice(pods.Items, func(i, j int) bool {
			return pods.Items[i].CreationTimestamp.After(pods.Items[j].CreationTimestamp.Time)
		})

		loftPod := &pods.Items[0]
		found := false
		for _, containerStatus := range loftPod.Status.ContainerStatuses {
			if containerStatus.State.Running != nil && containerStatus.Ready {
				if containerStatus.Name == "manager" {
					found = true
				}

				continue
			} else if containerStatus.State.Terminated != nil || (containerStatus.State.Waiting != nil && CriticalStatus[containerStatus.State.Waiting.Reason]) {
				reason := ""
				message := ""
				if containerStatus.State.Terminated != nil {
					reason = containerStatus.State.Terminated.Reason
					message = containerStatus.State.Terminated.Message
				} else if containerStatus.State.Waiting != nil {
					reason = containerStatus.State.Waiting.Reason
					message = containerStatus.State.Waiting.Message
				}

				out, err := kubeClient.CoreV1().Pods(namespace).GetLogs(loftPod.Name, &corev1.PodLogOptions{
					Container: "manager",
				}).Do(context.Background()).Raw()
				if err != nil {
					return false, fmt.Errorf("there seems to be an issue with %s starting up: %s (%s). Please reach out to our support at https://loft.sh/", product.DisplayName(), message, reason)
				}
				if strings.Contains(string(out), "register instance: Post \"https://license.loft.sh/register\": dial tcp") {
					return false, fmt.Errorf("%[1]s logs: \n%[2]v \nThere seems to be an issue with %[1]s starting up. Looks like you try to install %[1]s into an air-gapped environment, please reach out to our support at https://loft.sh/ for an offline license", product.DisplayName(), string(out))
				}

				return false, fmt.Errorf("%[1]s logs: \n%v \nThere seems to be an issue with %[1]s starting up: %[2]s (%[3]s). Please reach out to our support at https://loft.sh/", product.DisplayName(), string(out), message, reason)
			} else if containerStatus.State.Waiting != nil && time.Now().After(now.Add(time.Second*10)) {
				if containerStatus.State.Waiting.Message != "" {
					log.Infof("Please keep waiting, %s container is still starting up: %s (%s)", product.DisplayName(), containerStatus.State.Waiting.Message, containerStatus.State.Waiting.Reason)
				} else if containerStatus.State.Waiting.Reason != "" {
					log.Infof("Please keep waiting, %s container is still starting up: %s", product.DisplayName(), containerStatus.State.Waiting.Reason)
				} else {
					log.Infof("Please keep waiting, %s container is still starting up...", product.DisplayName())
				}

				now = time.Now()
			}

			return false, nil
		}

		pod = loftPod
		return found, nil
	})
	if err != nil {
		return nil, err
	}

	return pod, nil
}

func StartPortForwarding(ctx context.Context, config *rest.Config, client kubernetes.Interface, pod *corev1.Pod, localPort string, log log.Logger) (chan struct{}, error) {
	log.WriteString(logrus.InfoLevel, "\n")
	log.Infof("Starting port-forwarding to the %s pod", product.DisplayName())
	execRequest := client.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(pod.Name).
		Namespace(pod.Namespace).
		SubResource("portforward")

	t, upgrader, err := spdy.RoundTripperFor(config)
	if err != nil {
		return nil, err
	}

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: t}, "POST", execRequest.URL())
	errChan := make(chan error)
	readyChan := make(chan struct{})
	stopChan := make(chan struct{})
	targetPort := getPortForwardingTargetPort(pod)
	forwarder, err := portforward.New(dialer, []string{localPort + ":" + strconv.Itoa(targetPort)}, stopChan, readyChan, errChan, io.Discard, io.Discard)
	if err != nil {
		return nil, err
	}

	go func() {
		err := forwarder.ForwardPorts(ctx)
		if err != nil {
			errChan <- err
		}
	}()

	// wait till ready
	select {
	case err = <-errChan:
		return nil, err
	case <-readyChan:
	case <-stopChan:
		return nil, fmt.Errorf("stopped before ready")
	}

	// start watcher
	go func() {
		for {
			select {
			case <-stopChan:
				return
			case err = <-errChan:
				log.Infof("error during port forwarder: %v", err)
				close(stopChan)
				return
			}
		}
	}()

	return stopChan, nil
}

func GetLoftDefaultPassword(ctx context.Context, kubeClient kubernetes.Interface, namespace string) (string, error) {
	loftNamespace, err := kubeClient.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		if kerrors.IsNotFound(err) {
			loftNamespace, err := kubeClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespace,
				},
			}, metav1.CreateOptions{})
			if err != nil {
				return "", err
			}

			return string(loftNamespace.UID), nil
		}

		return "", err
	}

	return string(loftNamespace.UID), nil
}

type version struct {
	Version string `json:"version"`
}

func IsLoftReachable(ctx context.Context, host string) (bool, error) {
	// wait until loft is reachable at the given url
	client := &http.Client{
		Transport: utilhttp.InsecureTransport(),
	}
	url := "https://" + host + "/version"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, fmt.Errorf("error creating request with context: %w", err)
	}
	resp, err := client.Do(req)
	if err == nil && resp.StatusCode == http.StatusOK {
		out, err := io.ReadAll(resp.Body)
		if err != nil {
			return false, nil
		}

		v := &version{}
		err = json.Unmarshal(out, v)
		if err != nil {
			return false, fmt.Errorf("error decoding response from %s: %w. Try running '%s --reset'", url, err, product.StartCmd())
		} else if v.Version == "" {
			return false, fmt.Errorf("unexpected response from %s: %s. Try running '%s --reset'", url, string(out), product.StartCmd())
		}

		return true, nil
	}

	return false, nil
}

func IsLocalCluster(host string, log log.Logger) bool {
	url, err := url.Parse(host)
	if err != nil {
		log.Warnf("Couldn't parse kube context host url: %v", err)
		return false
	}

	hostname := url.Hostname()
	ip := net.ParseIP(hostname)
	if ip != nil {
		if IsPrivateIP(ip) {
			return true
		}
	}

	if hostname == "localhost" || strings.HasSuffix(hostname, ".internal") || strings.HasSuffix(hostname, ".localhost") {
		return true
	}

	return false
}

var privateIPBlocks []*net.IPNet

func init() {
	for _, cidr := range []string{
		"127.0.0.0/8",    // IPv4 loopback
		"10.0.0.0/8",     // RFC1918
		"172.16.0.0/12",  // RFC1918
		"192.168.0.0/16", // RFC1918
		"::1/128",        // IPv6 loopback
		"fe80::/10",      // IPv6 link-local
		"fc00::/7",       // IPv6 unique local addr
	} {
		_, block, _ := net.ParseCIDR(cidr)
		privateIPBlocks = append(privateIPBlocks, block)
	}
}

// IsPrivateIP checks if a given ip is private
func IsPrivateIP(ip net.IP) bool {
	for _, block := range privateIPBlocks {
		if block.Contains(ip) {
			return true
		}
	}

	return false
}

func EnterHostNameQuestion(log log.Logger) (string, error) {
	return log.Question(&survey.QuestionOptions{
		Question: fmt.Sprintf("Enter a hostname for your %s instance (e.g. loft.my-domain.tld): \n ", product.DisplayName()),
		ValidationFunc: func(answer string) error {
			u, err := url.Parse("https://" + answer)
			if err != nil || u.Path != "" || u.Port() != "" || len(strings.Split(answer, ".")) < 2 {
				return fmt.Errorf("please enter a valid hostname without protocol (https://), without path and without port, e.g. loft.my-domain.tld")
			}
			return nil
		},
	})
}

func IsLoftAlreadyInstalled(ctx context.Context, kubeClient kubernetes.Interface, namespace string) (bool, error) {
	_, err := kubeClient.AppsV1().Deployments(namespace).Get(ctx, defaultDeploymentName, metav1.GetOptions{})
	if err != nil {
		if kerrors.IsNotFound(err) {
			return false, nil
		}

		return false, fmt.Errorf("error accessing kubernetes cluster: %w", err)
	}

	return true, nil
}

func UninstallLoft(ctx context.Context, kubeClient kubernetes.Interface, restConfig *rest.Config, kubeContext, namespace string, log log.Logger) error {
	log.Infof("Uninstalling %s...", product.DisplayName())
	releaseName := defaultReleaseName
	deploy, err := kubeClient.AppsV1().Deployments(namespace).Get(ctx, defaultDeploymentName, metav1.GetOptions{})
	if err != nil && !kerrors.IsNotFound(err) {
		return err
	} else if deploy != nil && deploy.Labels != nil && deploy.Labels["release"] != "" {
		releaseName = deploy.Labels["release"]
	}

	args := []string{
		"uninstall",
		releaseName,
		"--kube-context",
		kubeContext,
		"--namespace",
		namespace,
	}
	log.Infof("Executing command: helm %s", strings.Join(args, " "))
	output, err := exec.Command("helm", args...).CombinedOutput()
	if err != nil {
		log.Errorf("error during helm command: %s (%v)", string(output), err)
	}

	// we also cleanup the validating webhook configuration and apiservice
	apiRegistrationClient, err := clientset.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	err = apiRegistrationClient.ApiregistrationV1().APIServices().Delete(ctx, "v1.management.loft.sh", metav1.DeleteOptions{})
	if err != nil && !kerrors.IsNotFound(err) {
		return err
	}

	err = deleteUser(ctx, restConfig, "admin")
	if err != nil {
		return err
	}

	err = kubeClient.CoreV1().Secrets(namespace).Delete(context.Background(), "loft-user-secret-admin", metav1.DeleteOptions{})
	if err != nil && !kerrors.IsNotFound(err) {
		return err
	}

	err = kubeClient.CoreV1().Secrets(namespace).Delete(context.Background(), LoftRouterDomainSecret, metav1.DeleteOptions{})
	if err != nil && !kerrors.IsNotFound(err) {
		return err
	}

	// we also cleanup the validating webhook configuration and apiservice
	err = kubeClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Delete(ctx, "loft-agent", metav1.DeleteOptions{})
	if err != nil && !kerrors.IsNotFound(err) {
		return err
	}

	err = apiRegistrationClient.ApiregistrationV1().APIServices().Delete(ctx, "v1alpha1.tenancy.kiosk.sh", metav1.DeleteOptions{})
	if err != nil && !kerrors.IsNotFound(err) {
		return err
	}

	err = apiRegistrationClient.ApiregistrationV1().APIServices().Delete(ctx, "v1.cluster.loft.sh", metav1.DeleteOptions{})
	if err != nil && !kerrors.IsNotFound(err) {
		return err
	}

	err = kubeClient.CoreV1().ConfigMaps(namespace).Delete(ctx, "loft-agent-controller", metav1.DeleteOptions{})
	if err != nil && !kerrors.IsNotFound(err) {
		return err
	}

	err = kubeClient.CoreV1().ConfigMaps(namespace).Delete(ctx, "loft-applied-defaults", metav1.DeleteOptions{})
	if err != nil && !kerrors.IsNotFound(err) {
		return err
	}

	log.WriteString(logrus.InfoLevel, "\n")
	log.Done(product.Replace("Successfully uninstalled Loft"))
	log.WriteString(logrus.InfoLevel, "\n")

	return nil
}

func deleteUser(ctx context.Context, restConfig *rest.Config, name string) error {
	loftClient, err := loftclientset.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	user, err := loftClient.StorageV1().Users().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil
	} else if len(user.Finalizers) > 0 {
		user.Finalizers = nil
		_, err = loftClient.StorageV1().Users().Update(ctx, user, metav1.UpdateOptions{})
		if err != nil {
			if kerrors.IsConflict(err) {
				return deleteUser(ctx, restConfig, name)
			}

			return err
		}
	}

	err = loftClient.StorageV1().Users().Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !kerrors.IsNotFound(err) {
		return err
	}

	return nil
}

func EnsureIngressController(ctx context.Context, kubeClient kubernetes.Interface, kubeContext string, log log.Logger) error {
	// first create an ingress controller
	const (
		YesOption = "Yes"
		NoOption  = "No, I already have an ingress controller installed."
	)

	answer, err := log.Question(&survey.QuestionOptions{
		Question:     "Ingress controller required. Should the nginx-ingress controller be installed?",
		DefaultValue: YesOption,
		Options: []string{
			YesOption,
			NoOption,
		},
	})
	if err != nil {
		return err
	}

	if answer == YesOption {
		args := []string{
			"install",
			"ingress-nginx",
			"ingress-nginx",
			"--repository-config=''",
			"--repo",
			"https://kubernetes.github.io/ingress-nginx",
			"--kube-context",
			kubeContext,
			"--namespace",
			"ingress-nginx",
			"--create-namespace",
			"--set-string",
			"controller.config.hsts=false",
			"--wait",
		}
		log.WriteString(logrus.InfoLevel, "\n")
		log.Infof("Executing command: helm %s\n", strings.Join(args, " "))
		log.Info("Waiting for ingress controller deployment, this can take several minutes...")
		helmCmd := exec.Command("helm", args...)
		output, err := helmCmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("error during helm command: %s (%w)", string(output), err)
		}

		list, err := kubeClient.CoreV1().Secrets("ingress-nginx").List(ctx, metav1.ListOptions{
			LabelSelector: "name=ingress-nginx,owner=helm,status=deployed",
		})
		if err != nil {
			return err
		}

		if len(list.Items) == 1 {
			secret := list.Items[0]
			originalSecret := secret.DeepCopy()
			secret.Labels["loft.sh/app"] = "true"
			if secret.Annotations == nil {
				secret.Annotations = map[string]string{}
			}

			secret.Annotations["loft.sh/url"] = "https://kubernetes.github.io/ingress-nginx"
			originalJSON, err := json.Marshal(originalSecret)
			if err != nil {
				return err
			}
			modifiedJSON, err := json.Marshal(secret)
			if err != nil {
				return err
			}
			data, err := jsonpatch.CreateMergePatch(originalJSON, modifiedJSON)
			if err != nil {
				return err
			}
			_, err = kubeClient.CoreV1().Secrets(secret.Namespace).Patch(ctx, secret.Name, types.MergePatchType, data, metav1.PatchOptions{})
			if err != nil {
				return err
			}
		}

		log.Done("Successfully installed ingress-nginx to your kubernetes cluster!")
	}

	return nil
}

func UpgradeLoft(chartName, chartRepo, kubeContext, namespace string, extraArgs []string, log log.Logger) error {
	// now we install loft
	args := []string{
		"upgrade",
		defaultReleaseName,
		chartName,
		"--install",
		"--reuse-values",
		"--create-namespace",
		"--repository-config=''",
		"--kube-context",
		kubeContext,
		"--namespace",
		namespace,
	}
	if chartRepo != "" {
		args = append(args, "--repo", chartRepo)
	}
	args = append(args, extraArgs...)

	log.WriteString(logrus.InfoLevel, "\n")
	log.Infof("Executing command: helm %s\n", strings.Join(args, " "))
	log.Info("Waiting for helm command, this can take up to several minutes...")
	helmCmd := exec.Command("helm", args...)
	if chartRepo != "" {
		helmWorkDir, err := getHelmWorkdir(chartName)
		if err != nil {
			return err
		}

		helmCmd.Dir = helmWorkDir
	}
	output, err := helmCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error during helm command: %s (%w)", string(output), err)
	}

	log.Donef("%s has been deployed to your cluster!", product.DisplayName())
	return nil
}

func GetLoftManifests(chartName, chartRepo, kubeContext, namespace string, extraArgs []string, _ log.Logger) (string, error) {
	args := []string{
		"template",
		defaultReleaseName,
		chartName,
		"--repository-config=''",
		"--kube-context",
		kubeContext,
		"--namespace",
		namespace,
	}
	if chartRepo != "" {
		args = append(args, "--repo", chartRepo)
	}
	args = append(args, extraArgs...)

	helmCmd := exec.Command("helm", args...)
	if chartRepo != "" {
		helmWorkDir, err := getHelmWorkdir(chartName)
		if err != nil {
			return "", err
		}

		helmCmd.Dir = helmWorkDir
	}
	output, err := helmCmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("error during helm command: %s (%w)", string(output), err)
	}
	return string(output), nil
}

// Return the directory where the `helm` commands should be executed or error if none can be found/created
// Uses current workdir by default unless it contains a folder with the chart name
func getHelmWorkdir(chartName string) (string, error) {
	// If chartName folder exists, check temp dir next
	if _, err := os.Stat(chartName); err == nil {
		tempDir := os.TempDir()

		// If tempDir/chartName folder exists, create temp folder
		if _, err := os.Stat(path.Join(tempDir, chartName)); err == nil {
			tempDir, err = os.MkdirTemp(tempDir, chartName)
			if err != nil {
				return "", errors.New("problematic directory `" + chartName + "` found: please execute command in a different folder")
			}
		}

		// Use tempDir
		return tempDir, nil
	}

	// Use current workdir
	return "", nil
}

// Makes sure that admin user and password secret exists
// Returns (true, nil) if everything is correct but password is different from parameter `password`
func EnsureAdminPassword(ctx context.Context, kubeClient kubernetes.Interface, restConfig *rest.Config, password string, log log.Logger) (bool, error) {
	loftClient, err := loftclientset.NewForConfig(restConfig)
	if err != nil {
		return false, err
	}

	admin, err := loftClient.StorageV1().Users().Get(ctx, "admin", metav1.GetOptions{})
	if err != nil && !kerrors.IsNotFound(err) {
		return false, err
	} else if admin == nil {
		admin, err = loftClient.StorageV1().Users().Create(ctx, &storagev1.User{
			ObjectMeta: metav1.ObjectMeta{
				Name: "admin",
			},
			Spec: storagev1.UserSpec{
				Username: "admin",
				Email:    "test@domain.tld",
				Subject:  "admin",
				Groups:   []string{"system:masters"},
				PasswordRef: &storagev1.SecretRef{
					SecretName:      "loft-user-secret-admin",
					SecretNamespace: "loft",
					Key:             "password",
				},
			},
		}, metav1.CreateOptions{})
		if err != nil {
			return false, err
		}
	} else if admin.Spec.PasswordRef == nil || admin.Spec.PasswordRef.SecretName == "" || admin.Spec.PasswordRef.SecretNamespace == "" {
		return false, nil
	}

	key := admin.Spec.PasswordRef.Key
	if key == "" {
		key = "password"
	}

	passwordHash := fmt.Sprintf("%x", sha256.Sum256([]byte(password)))

	secret, err := kubeClient.CoreV1().Secrets(admin.Spec.PasswordRef.SecretNamespace).Get(ctx, admin.Spec.PasswordRef.SecretName, metav1.GetOptions{})
	if err != nil && !kerrors.IsNotFound(err) {
		return false, err
	} else if err == nil {
		existingPasswordHash, keyExists := secret.Data[key]
		if keyExists {
			return (string(existingPasswordHash) != passwordHash), nil
		}

		secret.Data[key] = []byte(passwordHash)
		_, err = kubeClient.CoreV1().Secrets(secret.Namespace).Update(ctx, secret, metav1.UpdateOptions{})
		if err != nil {
			return false, errors.Wrap(err, "update admin password secret")
		}
		return false, nil
	}

	// create the password secret if it was not found, this can happen if you delete the loft namespace without deleting the admin user
	secret = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      admin.Spec.PasswordRef.SecretName,
			Namespace: admin.Spec.PasswordRef.SecretNamespace,
		},
		Data: map[string][]byte{
			key: []byte(passwordHash),
		},
	}
	_, err = kubeClient.CoreV1().Secrets(secret.Namespace).Create(ctx, secret, metav1.CreateOptions{})
	if err != nil {
		return false, errors.Wrap(err, "create admin password secret")
	}

	log.Info("Successfully recreated admin password secret")
	return false, nil
}

func IsLoftInstalledLocally(ctx context.Context, kubeClient kubernetes.Interface, namespace string) bool {
	_, err := kubeClient.NetworkingV1().Ingresses(namespace).Get(ctx, "loft-ingress", metav1.GetOptions{})
	if err != nil && !kerrors.IsNotFound(err) {
		_, err = kubeClient.NetworkingV1beta1().Ingresses(namespace).Get(ctx, "loft-ingress", metav1.GetOptions{})
		return kerrors.IsNotFound(err)
	}

	return kerrors.IsNotFound(err)
}

func getPortForwardingTargetPort(pod *corev1.Pod) int {
	for _, container := range pod.Spec.Containers {
		if container.Name == "manager" {
			for _, port := range container.Ports {
				if port.Name == "https" {
					return int(port.ContainerPort)
				}
			}
		}
	}

	return 10443
}
