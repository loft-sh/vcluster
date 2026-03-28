package kubeclient

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/loft-sh/vcluster/pkg/cli/config"
	"k8s.io/client-go/pkg/apis/clientauthentication/v1beta1"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// ContextOptions describes the kubeconfig context to create.
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

// --- Context name generators ---

// ContextName returns the kubeconfig context name for a standalone vCluster.
func ContextName(vclusterName, namespace, parentContext string) string {
	return "vcluster_" + vclusterName + "_" + namespace + "_" + parentContext
}

// PlatformContextName returns the kubeconfig context name for a platform-managed vCluster.
func PlatformContextName(vclusterName, project, parentContext string) string {
	return "vcluster-platform_" + vclusterName + "_" + project + "_" + parentContext
}

// SpaceContextName returns the kubeconfig context name for a host cluster or space.
func SpaceContextName(clusterName, namespaceName string) string {
	name := "vcluster-platform_"
	if namespaceName != "" {
		name += namespaceName + "_"
	}
	return name + clusterName
}

// SpaceInstanceContextName returns the kubeconfig context name for a space instance.
func SpaceInstanceContextName(projectName, spaceInstanceName string) string {
	return "vcluster-platform_" + spaceInstanceName + "_" + projectName
}

// VirtualClusterInstanceContextName returns the kubeconfig context name for a platform vCluster instance.
func VirtualClusterInstanceContextName(projectName, virtualClusterInstance string) string {
	return "vcluster-platform-vcluster_" + virtualClusterInstance + "_" + projectName
}

// ManagementContextName returns the kubeconfig context name for the platform management API.
func ManagementContextName() string {
	return "vcluster-platform_management"
}

var nonAllowedCharsRe = regexp.MustCompile(`[^a-zA-Z0-9\-_]+`)

// BackgroundProxyName returns the name used for the background proxy process for a vCluster.
func BackgroundProxyName(vclusterName, namespace, parentContext string) string {
	return nonAllowedCharsRe.ReplaceAllString(ContextName(vclusterName, namespace, parentContext)+"_background_proxy", "")
}

// --- Context name parsers ---

// FromContext parses a standalone vCluster context name into its components.
// Returns empty strings if the context is not a vCluster context.
func FromContext(originalContext string) (name, namespace, parentContext string) {
	if !strings.HasPrefix(originalContext, "vcluster_") {
		return "", "", ""
	}
	parts := strings.Split(originalContext, "_")
	if len(parts) >= 4 {
		return parts[1], parts[2], strings.Join(parts[3:], "_")
	}
	return originalContext, "", ""
}

// PlatformFromContext parses a platform vCluster context name into its components.
// Returns empty strings if the context is not a platform vCluster context.
func PlatformFromContext(originalContext string) (name, project, parentContext string) {
	if !strings.HasPrefix(originalContext, "vcluster-platform_") {
		return "", "", ""
	}
	parts := strings.Split(originalContext, "_")
	if len(parts) >= 4 {
		return parts[1], parts[2], strings.Join(parts[3:], "_")
	}
	return originalContext, "", ""
}

// DockerFromContext parses a Docker vCluster context name and returns the vCluster name.
// Returns an empty string if the context is not a Docker vCluster context.
func DockerFromContext(originalContext string) (name string) {
	if !strings.HasPrefix(originalContext, "vcluster-docker_") {
		return ""
	}
	parts := strings.Split(originalContext, "_")
	if len(parts) == 2 {
		return parts[1]
	}
	return ""
}

// --- Kubeconfig operations ---

// GetKubeConfig builds an in-memory *clientcmdapi.Config from ContextOptions.
// It does not write anything to disk.
func GetKubeConfig(opts ContextOptions) (*clientcmdapi.Config, error) {
	contextName := opts.Name
	cluster := clientcmdapi.NewCluster()
	cluster.Server = opts.Server
	cluster.CertificateAuthorityData = opts.CaData
	cluster.InsecureSkipTLSVerify = opts.InsecureSkipTLSVerify

	authInfo := clientcmdapi.NewAuthInfo()
	if opts.Token != "" {
		authInfo.Token = opts.Token
	} else if opts.ConfigPath != "" {
		command, err := os.Executable()
		if err != nil {
			return nil, err
		}
		absConfigPath, err := filepath.Abs(opts.ConfigPath)
		if err != nil {
			return nil, err
		}
		authInfo.Exec = &clientcmdapi.ExecConfig{
			APIVersion: v1beta1.SchemeGroupVersion.String(),
			Command:    command,
			Args:       []string{"platform", "token", "--silent", "--config", absConfigPath},
		}
	}

	cfg := clientcmdapi.NewConfig()
	cfg.Clusters[contextName] = cluster
	cfg.AuthInfos[contextName] = authInfo

	kubeContext := clientcmdapi.NewContext()
	kubeContext.Cluster = contextName
	kubeContext.AuthInfo = contextName
	kubeContext.Namespace = opts.CurrentNamespace

	cfg.Contexts[contextName] = kubeContext
	cfg.CurrentContext = contextName
	cfg.APIVersion = "v1"
	cfg.Kind = "Config"
	return cfg, nil
}

// UpdateKubeConfig merges a context built from opts into the local kubeconfig file.
// When opts.SetActive is true the context is made current and the previous context
// is saved to cfg so it can be restored by disconnect.
func UpdateKubeConfig(opts ContextOptions, cfg *config.CLI) error {
	contextName, cluster, authInfo, err := buildContext(opts)
	if err != nil {
		return err
	}

	rawConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	).RawConfig()
	if err != nil {
		return err
	}

	rawConfig.Clusters[contextName] = cluster
	rawConfig.AuthInfos[contextName] = authInfo

	kubeContext := clientcmdapi.NewContext()
	kubeContext.Cluster = contextName
	kubeContext.AuthInfo = contextName
	kubeContext.Namespace = opts.CurrentNamespace
	rawConfig.Contexts[contextName] = kubeContext

	if opts.SetActive {
		if !strings.HasPrefix(rawConfig.CurrentContext, "vcluster-platform") {
			cfg.PreviousContext = rawConfig.CurrentContext
			if err := cfg.Save(); err != nil {
				return fmt.Errorf("save config: %w", err)
			}
		}
		rawConfig.CurrentContext = contextName
	}

	return clientcmd.ModifyConfig(clientcmd.NewDefaultClientConfigLoadingRules(), rawConfig, false)
}

// MergeContext merges a pre-built cluster and authInfo into the local kubeconfig file.
// This is used by connect flows that already have a *clientcmdapi.Config and need to
// merge a single context without going through ContextOptions.
func MergeContext(contextName string, cluster *clientcmdapi.Cluster, authInfo *clientcmdapi.AuthInfo, namespace string, setActive bool) error {
	rawConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	).RawConfig()
	if err != nil {
		return err
	}

	if rawConfig.Clusters == nil {
		rawConfig.Clusters = map[string]*clientcmdapi.Cluster{}
	}
	if rawConfig.AuthInfos == nil {
		rawConfig.AuthInfos = map[string]*clientcmdapi.AuthInfo{}
	}
	if rawConfig.Contexts == nil {
		rawConfig.Contexts = map[string]*clientcmdapi.Context{}
	}

	rawConfig.Clusters[contextName] = cluster
	rawConfig.AuthInfos[contextName] = authInfo

	kubeContext := clientcmdapi.NewContext()
	kubeContext.Cluster = contextName
	kubeContext.AuthInfo = contextName
	kubeContext.Namespace = namespace
	rawConfig.Contexts[contextName] = kubeContext

	if setActive {
		rawConfig.CurrentContext = contextName
	}

	return clientcmd.ModifyConfig(clientcmd.NewDefaultClientConfigLoadingRules(), rawConfig, false)
}

// PrintKubeConfigTo writes a kubeconfig built from opts to w.
func PrintKubeConfigTo(opts ContextOptions, w io.Writer) error {
	contextName, cluster, authInfo, err := buildContext(opts)
	if err != nil {
		return err
	}

	cfg := clientcmdapi.NewConfig()
	cfg.Clusters[contextName] = cluster
	cfg.AuthInfos[contextName] = authInfo

	kubeContext := clientcmdapi.NewContext()
	kubeContext.Cluster = contextName
	kubeContext.AuthInfo = contextName
	kubeContext.Namespace = opts.CurrentNamespace
	cfg.Contexts[contextName] = kubeContext
	cfg.CurrentContext = contextName
	cfg.APIVersion = "v1"
	cfg.Kind = "Config"

	out, err := clientcmd.Write(*cfg)
	if err != nil {
		return err
	}
	_, err = w.Write(out)
	return err
}

// WriteToFile serialises cfg and writes it to the given file path.
func WriteToFile(cfg *clientcmdapi.Config, path string) error {
	out, err := clientcmd.Write(*cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0o666)
}

// DeleteContext removes the named context (and its cluster/authInfo) from the local kubeconfig.
func DeleteContext(contextName string) error {
	rawConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	).RawConfig()
	if err != nil {
		return err
	}

	delete(rawConfig.Contexts, contextName)
	delete(rawConfig.Clusters, contextName)
	delete(rawConfig.AuthInfos, contextName)

	if rawConfig.CurrentContext == contextName {
		rawConfig.CurrentContext = ""
		for name := range rawConfig.Contexts {
			rawConfig.CurrentContext = name
			break
		}
	}

	return clientcmd.ModifyConfig(clientcmd.NewDefaultClientConfigLoadingRules(), rawConfig, false)
}

// SwitchContext sets the current context in the local kubeconfig.
func SwitchContext(kubeConfig *clientcmdapi.Config, otherContext string) error {
	if kubeConfig == nil {
		return fmt.Errorf("nil kubeconfig")
	}
	kubeConfig.CurrentContext = otherContext
	return clientcmd.ModifyConfig(clientcmd.NewDefaultClientConfigLoadingRules(), *kubeConfig, false)
}

// CurrentContext returns the current context name and the full raw kubeconfig.
func CurrentContext() (string, *clientcmdapi.Config, error) {
	rawConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	).RawConfig()
	if err != nil {
		return "", nil, err
	}
	return rawConfig.CurrentContext, &rawConfig, nil
}

// buildContext creates the cluster and authInfo objects from ContextOptions.
func buildContext(opts ContextOptions) (string, *clientcmdapi.Cluster, *clientcmdapi.AuthInfo, error) {
	cluster := clientcmdapi.NewCluster()
	cluster.Server = opts.Server
	cluster.CertificateAuthorityData = opts.CaData
	cluster.InsecureSkipTLSVerify = opts.InsecureSkipTLSVerify

	authInfo := clientcmdapi.NewAuthInfo()
	if opts.Token != "" {
		authInfo.Token = opts.Token
	} else if opts.ConfigPath != "" {
		command, err := os.Executable()
		if err != nil {
			return "", nil, nil, err
		}
		absConfigPath, err := filepath.Abs(opts.ConfigPath)
		if err != nil {
			return "", nil, nil, err
		}
		authInfo.Exec = &clientcmdapi.ExecConfig{
			APIVersion: v1beta1.SchemeGroupVersion.String(),
			Command:    command,
			Args:       []string{"platform", "token", "--silent", "--config", absConfigPath},
		}
	}

	return opts.Name, cluster, authInfo, nil
}
