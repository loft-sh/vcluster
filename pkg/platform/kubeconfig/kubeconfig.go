package kubeconfig

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/loft-sh/vcluster/pkg/cli/config"
	"k8s.io/client-go/pkg/apis/clientauthentication/v1beta1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

type ContextOptions struct {
	Name                  string
	Server                string
	CaData                []byte
	ConfigPath            string
	InsecureSkipTLSVerify bool

	Token string

	CurrentNamespace string
	SetActive        bool
}

func SpaceInstanceContextName(projectName, spaceInstanceName string) string {
	return "vcluster-platform_" + spaceInstanceName + "_" + projectName
}

func VirtualClusterInstanceContextName(projectName, virtualClusterInstance string) string {
	return "vcluster-platform-vcluster_" + virtualClusterInstance + "_" + projectName
}

func SpaceContextName(clusterName, namespaceName string) string {
	contextName := "vcluster-platform_"
	if namespaceName != "" {
		contextName += namespaceName + "_"
	}

	contextName += clusterName
	return contextName
}

func ManagementContextName() string {
	return "vcluster-platform-management"
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

func createContext(options ContextOptions) (string, *api.Cluster, *api.AuthInfo, error) {
	contextName := options.Name
	cluster := api.NewCluster()
	cluster.Server = options.Server
	cluster.CertificateAuthorityData = options.CaData
	cluster.InsecureSkipTLSVerify = options.InsecureSkipTLSVerify

	authInfo := api.NewAuthInfo()
	if options.Token != "" {
		authInfo.Token = options.Token
	} else {
		command, err := os.Executable()
		if err != nil {
			return "", nil, nil, err
		}

		absConfigPath, err := filepath.Abs(options.ConfigPath)
		if err != nil {
			return "", nil, nil, err
		}

		authInfo.Exec = &api.ExecConfig{
			APIVersion: v1beta1.SchemeGroupVersion.String(),
			Command:    command,
			Args:       []string{"platform", "token", "--silent", "--config", absConfigPath},
		}
	}

	return contextName, cluster, authInfo, nil
}
