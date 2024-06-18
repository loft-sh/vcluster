package kubeconfig

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/loft-sh/vcluster/pkg/util/translate"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/apis/clientauthentication"
	"k8s.io/client-go/pkg/apis/clientauthentication/install"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	DefaultSecretPrefix     = "vc-"
	KubeconfigSecretKey     = "config"
	CADataSecretKey         = "certificate-authority"
	CertificateSecretKey    = "client-certificate"
	CertificateKeySecretKey = "client-key"
)

func WriteKubeConfig(ctx context.Context, currentNamespaceClient client.Client, secretName, secretNamespace string, config *clientcmdapi.Config, isRemote bool) error {
	out, err := clientcmd.Write(*config)
	if err != nil {
		return err
	}

	// try to write the kubeconfig file for backwards compatibility with older vcluster cli versions
	// intentionally ignore errors as this may fail for non root user, or if securityContext.readOnlyRootFilesystem = true
	err = os.MkdirAll("/root/.kube", 0755)
	if err == nil {
		err = os.WriteFile("/root/.kube/config", out, 0666)
		if err != nil {
			klog.Infof("Failed writing /root/.kube/config file: %v ; This error might be expected if you are running as non-root user or securityContext.readOnlyRootFilesystem = true", err)
		}
	} else {
		if !os.IsPermission(err) {
			klog.Infof("Failed to create /root/.kube folder for writing kube config: %v ; This error might be expected if you are running as non-root user or securityContext.readOnlyRootFilesystem = true", err)
		}
	}

	if secretName != "" {
		clientCmdConfig := clientcmd.NewDefaultClientConfig(*config, &clientcmd.ConfigOverrides{CurrentContext: config.CurrentContext})
		clientConfig, err := clientCmdConfig.ClientConfig()
		if err != nil {
			return err
		}

		caData := clientConfig.CAData
		cert := clientConfig.CertData
		key := clientConfig.KeyData

		kubeConfigSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: secretNamespace,
			},
		}
		result, err := controllerutil.CreateOrPatch(ctx, currentNamespaceClient, kubeConfigSecret, func() error {
			kubeConfigSecret.Type = corev1.SecretTypeOpaque
			if kubeConfigSecret.Data == nil {
				kubeConfigSecret.Data = map[string][]byte{}
			}
			kubeConfigSecret.Data[KubeconfigSecretKey] = out
			kubeConfigSecret.Data[CADataSecretKey] = caData
			kubeConfigSecret.Data[CertificateSecretKey] = cert
			kubeConfigSecret.Data[CertificateKeySecretKey] = key

			// set owner reference
			if !isRemote && translate.Owner != nil && translate.Owner.GetNamespace() == kubeConfigSecret.Namespace {
				kubeConfigSecret.OwnerReferences = translate.GetOwnerReference(nil)
			}
			return nil
		})
		if err != nil {
			return errors.Wrap(err, "apply generated kube config secret")
		} else if result != controllerutil.OperationResultNone {
			klog.Infof("Applied kube config secret %s/%s", kubeConfigSecret.Namespace, kubeConfigSecret.Name)
		}
	}

	return nil
}

func ReadKubeConfig(ctx context.Context, client *kubernetes.Clientset, suffix, namespace string) (*clientcmdapi.Config, error) {
	secret, err := client.CoreV1().Secrets(namespace).Get(ctx, GetDefaultSecretName(suffix), metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("could not Get the %s secret in order to read kubeconfig: %w", GetDefaultSecretName(suffix), err)
	}
	config, found := secret.Data[KubeconfigSecretKey]
	if !found {
		return nil, fmt.Errorf("could not find the kube config (%s key) in the %s secret", KubeconfigSecretKey, GetDefaultSecretName(suffix))
	}
	return clientcmd.Load(config)
}

func GetDefaultSecretName(suffix string) string {
	return DefaultSecretPrefix + suffix
}

func ConvertRestConfigToClientConfig(config *rest.Config) (clientcmd.ClientConfig, error) {
	contextName := "local"
	kubeConfig := clientcmdapi.NewConfig()
	kubeConfig.Contexts = map[string]*clientcmdapi.Context{
		contextName: {
			Cluster:  contextName,
			AuthInfo: contextName,
		},
	}
	kubeConfig.Clusters = map[string]*clientcmdapi.Cluster{
		contextName: {
			Server:                   config.Host,
			InsecureSkipTLSVerify:    config.Insecure,
			CertificateAuthorityData: config.CAData,
			CertificateAuthority:     config.CAFile,
		},
	}
	kubeConfig.AuthInfos = map[string]*clientcmdapi.AuthInfo{
		contextName: {
			Token:                 config.BearerToken,
			TokenFile:             config.BearerTokenFile,
			Impersonate:           config.Impersonate.UserName,
			ImpersonateGroups:     config.Impersonate.Groups,
			ImpersonateUserExtra:  config.Impersonate.Extra,
			ClientCertificate:     config.CertFile,
			ClientCertificateData: config.CertData,
			ClientKey:             config.KeyFile,
			ClientKeyData:         config.KeyData,
			Username:              config.Username,
			Password:              config.Password,
			AuthProvider:          config.AuthProvider,
			Exec:                  config.ExecProvider,
		},
	}
	kubeConfig.CurrentContext = contextName

	// resolve certificate
	if kubeConfig.Clusters[contextName].CertificateAuthorityData == nil && kubeConfig.Clusters[contextName].CertificateAuthority != "" {
		o, err := os.ReadFile(kubeConfig.Clusters[contextName].CertificateAuthority)
		if err != nil {
			return nil, err
		}

		kubeConfig.Clusters[contextName].CertificateAuthority = ""
		kubeConfig.Clusters[contextName].CertificateAuthorityData = o
	}

	// fill in data
	if kubeConfig.AuthInfos[contextName].ClientCertificateData == nil && kubeConfig.AuthInfos[contextName].ClientCertificate != "" {
		o, err := os.ReadFile(kubeConfig.AuthInfos[contextName].ClientCertificate)
		if err != nil {
			return nil, err
		}

		kubeConfig.AuthInfos[contextName].ClientCertificate = ""
		kubeConfig.AuthInfos[contextName].ClientCertificateData = o
	}
	if kubeConfig.AuthInfos[contextName].ClientKeyData == nil && kubeConfig.AuthInfos[contextName].ClientKey != "" {
		o, err := os.ReadFile(kubeConfig.AuthInfos[contextName].ClientKey)
		if err != nil {
			return nil, err
		}

		kubeConfig.AuthInfos[contextName].ClientKey = ""
		kubeConfig.AuthInfos[contextName].ClientKeyData = o
	}

	return clientcmd.NewDefaultClientConfig(*kubeConfig, &clientcmd.ConfigOverrides{}), nil
}

func ResolveKubeConfig(rawConfig clientcmd.ClientConfig) ([]byte, error) {
	restConfig, err := rawConfig.ClientConfig()
	if err != nil {
		return nil, err
	}

	// convert exec auth
	if restConfig.ExecProvider != nil {
		err = resolveExecCredentials(restConfig)
		if err != nil {
			return nil, fmt.Errorf("resolve exec credentials: %w", err)
		}
	}
	if restConfig.AuthProvider != nil {
		return nil, fmt.Errorf("auth provider is not supported")
	}

	retConfig, err := ConvertRestConfigToClientConfig(restConfig)
	if err != nil {
		return nil, err
	}

	retRawConfig, err := retConfig.RawConfig()
	if err != nil {
		return nil, err
	}

	return clientcmd.Write(retRawConfig)
}

func resolveExecCredentials(restConfig *rest.Config) error {
	cred := &clientauthentication.ExecCredential{
		Spec: clientauthentication.ExecCredentialSpec{
			Interactive: false,
		},
	}

	execProvider := restConfig.ExecProvider
	if execProvider.ProvideClusterInfo {
		var err error
		cred.Spec.Cluster, err = rest.ConfigToExecCluster(restConfig)
		if err != nil {
			return err
		}
	}

	env := os.Environ()
	for _, e := range execProvider.Env {
		env = append(env, e.Name+"="+e.Value)
	}

	groupVersion, err := schema.ParseGroupVersion(execProvider.APIVersion)
	if err != nil {
		return err
	}

	scheme := runtime.NewScheme()
	codecs := serializer.NewCodecFactory(scheme)
	install.Install(scheme)
	data, err := runtime.Encode(codecs.LegacyCodec(groupVersion), cred)
	if err != nil {
		return fmt.Errorf("encode ExecCredentials: %w", err)
	}
	env = append(env, fmt.Sprintf("%s=%s", "KUBERNETES_EXEC_INFO", data))

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd := exec.Command(execProvider.Command, execProvider.Args...)
	cmd.Env = env
	cmd.Stderr = stderr
	cmd.Stdout = stdout

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("error executing exec provider: %s %s %w", stderr.String(), stdout.String(), err)
	}

	_, gvk, err := codecs.UniversalDecoder(groupVersion).Decode(stdout.Bytes(), nil, cred)
	if err != nil {
		return fmt.Errorf("decoding stdout: %w", err)
	}
	if gvk.Group != groupVersion.Group || gvk.Version != groupVersion.Version {
		return fmt.Errorf("exec plugin is configured to use API version %s, plugin returned version %s",
			groupVersion, schema.GroupVersion{Group: gvk.Group, Version: gvk.Version})
	}

	if cred.Status == nil {
		return fmt.Errorf("exec plugin didn't return a status field")
	}
	if cred.Status.Token == "" && cred.Status.ClientCertificateData == "" && cred.Status.ClientKeyData == "" {
		return fmt.Errorf("exec plugin didn't return a token or cert/key pair")
	}
	if (cred.Status.ClientCertificateData == "") != (cred.Status.ClientKeyData == "") {
		return fmt.Errorf("exec plugin returned only certificate or key, not both")
	}

	if cred.Status.Token != "" {
		restConfig.BearerToken = cred.Status.Token
	} else if cred.Status.ClientKeyData != "" && cred.Status.ClientCertificateData != "" {
		restConfig.KeyData = []byte(cred.Status.ClientKeyData)
		restConfig.CertData = []byte(cred.Status.ClientCertificateData)
	}

	restConfig.ExecProvider = nil
	return nil
}
