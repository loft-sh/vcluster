package kubeconfig

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/loft-sh/vcluster/pkg/cli/config"
	"k8s.io/client-go/pkg/apis/clientauthentication/v1beta1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

type ContextOptions struct {
	Name                             string
	Server                           string
	CaData                           []byte
	ConfigPath                       string
	InsecureSkipTLSVerify            bool
	DirectClusterEndpointEnabled     bool
	VirtualClusterAccessPointEnabled bool

	Token                 string
	ClientKeyData         []byte
	ClientCertificateData []byte

	CurrentNamespace string
	SetActive        bool
}

func SpaceInstanceContextName(projectName, spaceInstanceName string) string {
	return "vcluster-platform_" + spaceInstanceName + "_" + projectName
}

func VirtualClusterInstanceContextName(projectName, virtualClusterInstance string) string {
	return "vcluster-platform-vcluster_" + virtualClusterInstance + "_" + projectName
}

func virtualClusterInstanceProjectAndNameFromContextName(contextName string) (string, string) {
	return strings.Split(contextName, "_")[2], strings.Split(contextName, "_")[1]
}

func SpaceContextName(clusterName, namespaceName string) string {
	contextName := "vcluster-platform_"
	if namespaceName != "" {
		contextName += namespaceName + "_"
	}

	contextName += clusterName
	return contextName
}

func VirtualClusterContextName(clusterName, namespaceName, virtualClusterName string) string {
	return "vcluster-platform-vcluster_" + virtualClusterName + "_" + namespaceName + "_" + clusterName
}

func ManagementContextName() string {
	return "vcluster-platform-management"
}

func ParseContext(contextName string) (isPlatformContext bool, cluster string, namespace string, vCluster string) {
	splitted := strings.Split(contextName, "_")
	if len(splitted) == 0 || (splitted[0] != "vcluster-platform" && splitted[0] != "vcluster-platform-vcluster") {
		return false, "", "", ""
	}

	// cluster or space context
	if splitted[0] == "vcluster-platform" {
		if len(splitted) > 3 || len(splitted) == 1 {
			return false, "", "", ""
		} else if len(splitted) == 2 {
			return true, splitted[1], "", ""
		}

		return true, splitted[2], splitted[1], ""
	}

	// vCluster context
	if len(splitted) != 4 {
		return false, "", "", ""
	}

	return true, splitted[3], splitted[2], splitted[1]
}

func CurrentContext() (string, error) {
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{}).RawConfig()
	if err != nil {
		return "", err
	}

	return config.CurrentContext, nil
}

// DeleteContext deletes the context with the given name from the kube config
func DeleteContext(contextName string) error {
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{}).RawConfig()
	if err != nil {
		return err
	}

	delete(config.Contexts, contextName)
	delete(config.Clusters, contextName)
	delete(config.AuthInfos, contextName)

	if config.CurrentContext == contextName {
		config.CurrentContext = ""
		for name := range config.Contexts {
			config.CurrentContext = name
			break
		}
	}

	// Save the config
	return clientcmd.ModifyConfig(clientcmd.NewDefaultClientConfigLoadingRules(), config, false)
}

func updateKubeConfig(contextName string, cluster *api.Cluster, authInfo *api.AuthInfo, namespaceName string, setActive bool, cfg *config.CLI) error {
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
	context.Namespace = namespaceName

	config.Contexts[contextName] = context
	if setActive {
		if !strings.HasPrefix(config.CurrentContext, "vcluster-platform") {
			cfg.PreviousContext = config.CurrentContext
			if err := cfg.Save(); err != nil {
				return fmt.Errorf("save config: %w", err)
			}
		}

		config.CurrentContext = contextName
	}

	// Save the config
	return clientcmd.ModifyConfig(clientcmd.NewDefaultClientConfigLoadingRules(), config, false)
}

func printKubeConfigTo(contextName string, cluster *api.Cluster, authInfo *api.AuthInfo, namespaceName string, writer io.Writer) error {
	config := api.NewConfig()

	config.Clusters[contextName] = cluster
	config.AuthInfos[contextName] = authInfo

	// Update kube context
	context := api.NewContext()
	context.Cluster = contextName
	context.AuthInfo = contextName
	context.Namespace = namespaceName

	config.Contexts[contextName] = context
	config.CurrentContext = contextName

	// set kind & version
	config.APIVersion = "v1"
	config.Kind = "Config"

	out, err := clientcmd.Write(*config)
	if err != nil {
		return err
	}

	_, err = writer.Write(out)
	return err
}

// UpdateKubeConfig updates the kube config and adds the virtual cluster context
func UpdateKubeConfig(options ContextOptions, cfg *config.CLI) error {
	contextName, cluster, authInfo, err := createContext(options)
	if err != nil {
		return err
	}

	// we don't want to set the space name here as the default namespace in the virtual cluster, because it couldn't exist
	return updateKubeConfig(contextName, cluster, authInfo, options.CurrentNamespace, options.SetActive, cfg)
}

// PrintKubeConfigTo prints the given config to the writer
func PrintKubeConfigTo(options ContextOptions, writer io.Writer) error {
	contextName, cluster, authInfo, err := createContext(options)
	if err != nil {
		return err
	}

	// we don't want to set the space name here as the default namespace in the virtual cluster, because it couldn't exist
	return printKubeConfigTo(contextName, cluster, authInfo, options.CurrentNamespace, writer)
}

// PrintTokenKubeConfig writes the kube config to the os.Stdout
func PrintTokenKubeConfig(restConfig *rest.Config, token string) error {
	contextName, cluster, authInfo := createTokenContext(restConfig, token)

	return printKubeConfigTo(contextName, cluster, authInfo, "", os.Stdout)
}

// WriteTokenKubeConfig writes the kube config to the io.Writer
func WriteTokenKubeConfig(restConfig *rest.Config, token string, w io.Writer) error {
	contextName, cluster, authInfo := createTokenContext(restConfig, token)

	return printKubeConfigTo(contextName, cluster, authInfo, "", w)
}

func createTokenContext(restConfig *rest.Config, token string) (string, *api.Cluster, *api.AuthInfo) {
	contextName := "default"

	cluster := api.NewCluster()
	cluster.Server = restConfig.Host
	cluster.InsecureSkipTLSVerify = restConfig.Insecure
	cluster.CertificateAuthority = restConfig.CAFile
	cluster.CertificateAuthorityData = restConfig.CAData
	cluster.TLSServerName = restConfig.ServerName

	authInfo := api.NewAuthInfo()
	authInfo.Token = token

	return contextName, cluster, authInfo
}

func createContext(options ContextOptions) (string, *api.Cluster, *api.AuthInfo, error) {
	contextName := options.Name
	cluster := api.NewCluster()
	cluster.Server = options.Server
	cluster.CertificateAuthorityData = options.CaData
	cluster.InsecureSkipTLSVerify = options.InsecureSkipTLSVerify

	authInfo := api.NewAuthInfo()
	if options.Token != "" || options.ClientCertificateData != nil || options.ClientKeyData != nil {
		authInfo.Token = options.Token
		authInfo.ClientKeyData = options.ClientKeyData
		authInfo.ClientCertificateData = options.ClientCertificateData
	} else {
		command, err := os.Executable()
		if err != nil {
			return "", nil, nil, err
		}

		absConfigPath, err := filepath.Abs(options.ConfigPath)
		if err != nil {
			return "", nil, nil, err
		}

		if options.VirtualClusterAccessPointEnabled {
			projectName, virtualClusterName := virtualClusterInstanceProjectAndNameFromContextName(contextName)
			authInfo.Exec = &api.ExecConfig{
				APIVersion: v1beta1.SchemeGroupVersion.String(),
				Command:    command,
				Args:       []string{"platform", "token", "--silent", "--project", projectName, "--virtual-cluster", virtualClusterName},
			}
		} else {
			authInfo.Exec = &api.ExecConfig{
				APIVersion: v1beta1.SchemeGroupVersion.String(),
				Command:    command,
				Args:       []string{"platform", "token", "--silent", "--config", absConfigPath},
			}
			if options.DirectClusterEndpointEnabled {
				authInfo.Exec.Args = append(authInfo.Exec.Args, "--direct-cluster-endpoint")
			}
		}
	}

	return contextName, cluster, authInfo, nil
}
