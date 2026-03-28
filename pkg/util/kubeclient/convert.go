package kubeclient

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/pkg/apis/clientauthentication"
	"k8s.io/client-go/pkg/apis/clientauthentication/install"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// ConvertRestConfigToClientConfig converts a *rest.Config into a ClientConfig,
// resolving file-backed certificate and key fields to inline data.
func ConvertRestConfigToClientConfig(config *rest.Config) (clientcmd.ClientConfig, error) {
	contextName := "local"
	kubeConfig := clientcmdapi.NewConfig()
	kubeConfig.Contexts = map[string]*clientcmdapi.Context{
		contextName: {
			Cluster:  contextName,
			AuthInfo: contextName,
		},
	}

	cluster := &clientcmdapi.Cluster{
		Server:                   config.Host,
		InsecureSkipTLSVerify:    config.Insecure,
		CertificateAuthorityData: config.CAData,
		CertificateAuthority:     config.CAFile,
	}
	kubeConfig.Clusters = map[string]*clientcmdapi.Cluster{contextName: cluster}

	authInfo := &clientcmdapi.AuthInfo{
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
	}
	kubeConfig.AuthInfos = map[string]*clientcmdapi.AuthInfo{contextName: authInfo}
	kubeConfig.CurrentContext = contextName

	// resolve file-backed CA
	if cluster.CertificateAuthorityData == nil && cluster.CertificateAuthority != "" {
		data, err := os.ReadFile(cluster.CertificateAuthority)
		if err != nil {
			return nil, err
		}
		cluster.CertificateAuthority = ""
		cluster.CertificateAuthorityData = data
	}

	// resolve file-backed client cert/key
	if authInfo.ClientCertificateData == nil && authInfo.ClientCertificate != "" {
		data, err := os.ReadFile(authInfo.ClientCertificate)
		if err != nil {
			return nil, err
		}
		authInfo.ClientCertificate = ""
		authInfo.ClientCertificateData = data
	}
	if authInfo.ClientKeyData == nil && authInfo.ClientKey != "" {
		data, err := os.ReadFile(authInfo.ClientKey)
		if err != nil {
			return nil, err
		}
		authInfo.ClientKey = ""
		authInfo.ClientKeyData = data
	}

	return clientcmd.NewDefaultClientConfig(*kubeConfig, &clientcmd.ConfigOverrides{}), nil
}

// ResolveKubeConfig resolves any exec or auth-provider credentials in rawConfig
// into a concrete clientcmdapi.Config (token or cert/key), suitable for
// serialisation and embedding in a kubeconfig Secret.
func ResolveKubeConfig(rawConfig clientcmd.ClientConfig) (clientcmdapi.Config, error) {
	restConfig, err := rawConfig.ClientConfig()
	if err != nil {
		return clientcmdapi.Config{}, err
	}

	if restConfig.ExecProvider != nil {
		if err := resolveExecCredentials(restConfig); err != nil {
			return clientcmdapi.Config{}, fmt.Errorf("resolve exec credentials: %w", err)
		}
	}
	if restConfig.AuthProvider != nil {
		return clientcmdapi.Config{}, fmt.Errorf("auth provider is not supported")
	}

	resolved, err := ConvertRestConfigToClientConfig(restConfig)
	if err != nil {
		return clientcmdapi.Config{}, err
	}

	return resolved.RawConfig()
}

// resolveExecCredentials executes the exec credential plugin referenced by
// restConfig and writes the resulting token or cert/key back into restConfig.
func resolveExecCredentials(restConfig *rest.Config) error {
	cred := &clientauthentication.ExecCredential{
		Spec: clientauthentication.ExecCredentialSpec{Interactive: false},
	}

	execProvider := restConfig.ExecProvider
	if execProvider == nil {
		return fmt.Errorf("exec provider is missing")
	}
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
	env = append(env, "KUBERNETES_EXEC_INFO="+string(data))

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd := exec.Command(execProvider.Command, execProvider.Args...)
	cmd.Env = env
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if err := cmd.Run(); err != nil {
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
	} else {
		restConfig.KeyData = []byte(cred.Status.ClientKeyData)
		restConfig.CertData = []byte(cred.Status.ClientCertificateData)
	}
	restConfig.ExecProvider = nil
	return nil
}
