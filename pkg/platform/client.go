package platform

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/blang/semver"
	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	"github.com/loft-sh/api/v4/pkg/auth"
	"github.com/loft-sh/api/v4/pkg/product"
	"github.com/loft-sh/loftctl/v4/pkg/constants"
	"github.com/loft-sh/loftctl/v4/pkg/kube"
	"github.com/loft-sh/loftctl/v4/pkg/parameters"
	"github.com/loft-sh/loftctl/v4/pkg/upgrade"
	"github.com/loft-sh/loftctl/v4/pkg/version"
	"github.com/loft-sh/log"
	"github.com/loft-sh/log/survey"
	"github.com/loft-sh/vcluster/pkg/cli/config"
	"github.com/loft-sh/vcluster/pkg/platform/clihelper"
	"github.com/loft-sh/vcluster/pkg/platform/kubeconfig"
	"github.com/loft-sh/vcluster/pkg/projectutil"
	perrors "github.com/pkg/errors"
	"github.com/skratchdot/open-golang/open"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/utils/ptr"
)

const (
	LoftDirectClusterEndpointCaData = "loft.sh/direct-cluster-endpoint-ca-data"
	VersionPath                     = "%s/version"
	LoginPath                       = "%s/login?cli=true"
	RedirectPath                    = "%s/spaces"
	AccessKeyPath                   = "%s/profile/access-keys"
	ConfigFileName                  = "platform.json"
	RefreshToken                    = time.Minute * 30
	CacheFolder                     = ".vcluster"
)

var (
	Self     *managementv1.Self
	selfOnce sync.Once
)

type Client interface {
	LoginClient
	Management() (kube.Interface, error)
	ManagementConfig() (*rest.Config, error)
	RefreshSelf(ctx context.Context) error
	Self() *managementv1.Self

	SpaceInstance(project, name string) (kube.Interface, error)
	SpaceInstanceConfig(project, name string) (*rest.Config, error)

	VirtualClusterInstance(project, name string) (kube.Interface, error)
	VirtualClusterInstanceConfig(project, name string) (*rest.Config, error)

	Cluster(cluster string) (kube.Interface, error)
	ClusterConfig(cluster string) (*rest.Config, error)

	ListVClusters(ctx context.Context, virtualClusterName, projectName string) ([]*VirtualClusterInstanceProject, error)
	CreateVirtualClusterInstanceOptions(ctx context.Context, config string, projectName string, virtualClusterInstance *managementv1.VirtualClusterInstance, setActive bool) (kubeconfig.ContextOptions, error)
	VirtualClusterAccessPointCertificate(project, virtualCluster string, forceRefresh bool) (string, string, error)
	ResolveTemplate(ctx context.Context, project, template, templateVersion string, setParams []string, fileParams string, log log.Logger) (*managementv1.VirtualClusterTemplate, string, error)

	ApplyPlatformSecret(ctx context.Context, kubeClient kubernetes.Interface, importName, namespace, project string) error

	Logout(ctx context.Context) error

	Version() (*auth.Version, error)
	Config() *config.CLI
	Save() error
}

type LoginClient interface {
	Login(host string, insecure bool, log log.Logger) error
	LoginWithAccessKey(host, accessKey string, insecure bool) error
}

// InitClientFromConfig returns a client with the client identity initialized through the selves api.
// Use this by default, unless performing actions that don't require a log in (like login itself).
func InitClientFromConfig(ctx context.Context, config *config.CLI) (Client, error) {
	c := NewClientFromConfig(config)

	if err := c.RefreshSelf(ctx); err != nil {
		return nil, err
	}

	selfOnce.Do(func() {
		Self = c.Self()
	})

	return c, nil
}

// NewClientFromConfig returns a client without the client identity initialized through the selves api.
// Use this only when performing actions that don't require a log in (like login itself).
func NewClientFromConfig(config *config.CLI) Client {
	c := &client{
		config: config,
	}
	return c
}

func NewLoginClientFromConfig(config *config.CLI) LoginClient {
	return &client{
		config: config,
	}
}

type client struct {
	config *config.CLI

	self *managementv1.Self
}

func (c *client) RefreshSelf(ctx context.Context) error {
	managementClient, err := c.Management()
	if err != nil {
		return fmt.Errorf("create mangement client: %w", err)
	}

	c.self, err = managementClient.Loft().ManagementV1().Selves().Create(ctx, &managementv1.Self{}, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("get self: %w", err)
	}

	projectNamespacePrefix := projectutil.LegacyProjectNamespacePrefix
	if c.self.Status.ProjectNamespacePrefix != nil {
		projectNamespacePrefix = *c.self.Status.ProjectNamespacePrefix
	}

	projectutil.SetProjectNamespacePrefix(projectNamespacePrefix)
	return nil
}

func (c *client) Self() *managementv1.Self {
	return c.self.DeepCopy()
}

// Logout implements Client.
func (c *client) Logout(ctx context.Context) error {
	managementClient, err := c.Management()
	if err != nil {
		return fmt.Errorf("create management client: %w", err)
	}

	self, err := managementClient.Loft().ManagementV1().Selves().Create(ctx, &managementv1.Self{}, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("get self: %w", err)
	}

	if self.Status.AccessKey != "" && self.Status.AccessKeyType == storagev1.AccessKeyTypeLogin {
		err = managementClient.Loft().ManagementV1().OwnedAccessKeys().Delete(ctx, self.Status.AccessKey, metav1.DeleteOptions{})
		if err != nil {
			return fmt.Errorf("delete access key: %w", err)
		}
	}

	return nil
}

func (c *client) VirtualClusterAccessPointCertificate(project, virtualCluster string, forceRefresh bool) (string, string, error) {
	if c.config == nil {
		return "", "", perrors.New("no config loaded")
	}

	contextName := kubeconfig.VirtualClusterInstanceContextName(project, virtualCluster)

	// see if we have stored cert data for this vci
	now := metav1.Now()
	cachedVirtualClusterAccessPointCertificate, ok := c.config.Platform.VirtualClusterAccessPointCertificates[contextName]
	if !forceRefresh && ok && cachedVirtualClusterAccessPointCertificate.LastRequested.Add(RefreshToken).After(now.Time) && cachedVirtualClusterAccessPointCertificate.ExpirationTime.After(now.Time) {
		return cachedVirtualClusterAccessPointCertificate.CertificateData, cachedVirtualClusterAccessPointCertificate.KeyData, nil
	}

	// refresh token
	managementClient, err := c.Management()
	if err != nil {
		return "", "", err
	}

	kubeConfigResponse, err := managementClient.Loft().ManagementV1().VirtualClusterInstances(projectutil.ProjectNamespace(project)).GetKubeConfig(
		context.Background(),
		virtualCluster,
		&managementv1.VirtualClusterInstanceKubeConfig{
			Spec: managementv1.VirtualClusterInstanceKubeConfigSpec{
				CertificateTTL: ptr.To[int32](86_400),
			},
		},
		metav1.CreateOptions{},
	)
	if err != nil {
		return "", "", perrors.Wrap(err, "fetch certificate data")
	}

	certificateData, keyData, err := getCertificateAndKeyDataFromKubeConfig(kubeConfigResponse.Status.KubeConfig)
	if err != nil {
		return "", "", err
	}

	if c.config.Platform.VirtualClusterAccessPointCertificates == nil {
		c.config.Platform.VirtualClusterAccessPointCertificates = make(map[string]config.VirtualClusterCertificatesEntry)
	}
	c.config.Platform.VirtualClusterAccessPointCertificates[contextName] = config.VirtualClusterCertificatesEntry{
		CertificateData: certificateData,
		KeyData:         keyData,
		LastRequested:   now,
		ExpirationTime:  now.Add(86_400 * time.Second),
	}

	err = c.Save()
	if err != nil {
		return "", "", perrors.Wrap(err, "save config")
	}

	return certificateData, keyData, nil
}

func (c *client) ResolveTemplate(
	ctx context.Context,
	project,
	template,
	templateVersion string,
	setParams []string,
	fileParams string,
	log log.Logger,
) (*managementv1.VirtualClusterTemplate, string, error) {
	// determine space template to use
	virtualClusterTemplate, err := c.SelectVirtualClusterTemplate(ctx, project, template, log)
	if err != nil {
		return nil, "", err
	}

	// get parameters
	var templateParameters []storagev1.AppParameter
	if len(virtualClusterTemplate.Spec.Versions) > 0 {
		if templateVersion == "" {
			latestVersion := version.GetLatestVersion(virtualClusterTemplate)
			if latestVersion == nil {
				return nil, "", fmt.Errorf("couldn't find any version in template")
			}

			templateParameters = latestVersion.(*storagev1.VirtualClusterTemplateVersion).Parameters
		} else {
			_, latestMatched, err := version.GetLatestMatchedVersion(virtualClusterTemplate, templateVersion)
			if err != nil {
				return nil, "", err
			} else if latestMatched == nil {
				return nil, "", fmt.Errorf("couldn't find any matching version to %s", templateVersion)
			}

			templateParameters = latestMatched.(*storagev1.VirtualClusterTemplateVersion).Parameters
		}
	} else {
		templateParameters = virtualClusterTemplate.Spec.Parameters
	}

	// resolve space template parameters
	resolvedParameters, err := parameters.ResolveTemplateParameters(setParams, templateParameters, fileParams)
	if err != nil {
		return nil, "", err
	}

	return virtualClusterTemplate, resolvedParameters, nil
}

func (c *client) SelectVirtualClusterTemplate(ctx context.Context, projectName, templateName string, log log.Logger) (*managementv1.VirtualClusterTemplate, error) {
	managementClient, err := c.Management()
	if err != nil {
		return nil, err
	}

	projectTemplates, err := managementClient.Loft().ManagementV1().Projects().ListTemplates(ctx, projectName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	// select default template
	if templateName == "" && projectTemplates.DefaultVirtualClusterTemplate != "" {
		templateName = projectTemplates.DefaultVirtualClusterTemplate
	}

	// try to find template
	if templateName != "" {
		for _, virtualClusterTemplate := range projectTemplates.VirtualClusterTemplates {
			if virtualClusterTemplate.Name == templateName {
				return &virtualClusterTemplate, nil
			}
		}

		return nil, fmt.Errorf("couldn't find template %s as allowed template in project %s", templateName, projectName)
	} else if len(projectTemplates.VirtualClusterTemplates) == 0 {
		return nil, fmt.Errorf("there are no allowed virtual cluster templates in project %s", projectName)
	} else if len(projectTemplates.VirtualClusterTemplates) == 1 {
		return &projectTemplates.VirtualClusterTemplates[0], nil
	}

	templateNames := []string{}
	for _, template := range projectTemplates.VirtualClusterTemplates {
		templateNames = append(templateNames, clihelper.GetDisplayName(template.Name, template.Spec.DisplayName))
	}
	answer, err := log.Question(&survey.QuestionOptions{
		Question:     "Please choose a template to use",
		DefaultValue: templateNames[0],
		Options:      templateNames,
	})
	if err != nil {
		return nil, err
	}
	for _, template := range projectTemplates.VirtualClusterTemplates {
		if answer == clihelper.GetDisplayName(template.Name, template.Spec.DisplayName) {
			return &template, nil
		}
	}

	return nil, fmt.Errorf("answer not found")
}

func getCertificateAndKeyDataFromKubeConfig(config string) (string, string, error) {
	clientCfg, err := clientcmd.NewClientConfigFromBytes([]byte(config))
	if err != nil {
		return "", "", err
	}

	apiCfg, err := clientCfg.RawConfig()
	if err != nil {
		return "", "", err
	}

	return string(apiCfg.AuthInfos["vcluster"].ClientCertificateData), string(apiCfg.AuthInfos["vcluster"].ClientKeyData), nil
}

func (c *client) Save() error {
	return c.config.Save()
}

func (c *client) ManagementConfig() (*rest.Config, error) {
	return c.restConfig("/kubernetes/management")
}

func (c *client) Management() (kube.Interface, error) {
	restConfig, err := c.ManagementConfig()
	if err != nil {
		return nil, err
	}

	return kube.NewForConfig(restConfig)
}

func (c *client) SpaceInstanceConfig(project, name string) (*rest.Config, error) {
	return c.restConfig("/kubernetes/project/" + project + "/space/" + name)
}

func (c *client) SpaceInstance(project, name string) (kube.Interface, error) {
	restConfig, err := c.SpaceInstanceConfig(project, name)
	if err != nil {
		return nil, err
	}

	return kube.NewForConfig(restConfig)
}

func (c *client) VirtualClusterInstanceConfig(project, name string) (*rest.Config, error) {
	return c.restConfig("/kubernetes/project/" + project + "/virtualcluster/" + name)
}

func (c *client) VirtualClusterInstance(project, name string) (kube.Interface, error) {
	restConfig, err := c.VirtualClusterInstanceConfig(project, name)
	if err != nil {
		return nil, err
	}

	return kube.NewForConfig(restConfig)
}

func (c *client) ClusterConfig(cluster string) (*rest.Config, error) {
	return c.restConfig("/kubernetes/cluster/" + cluster)
}

func (c *client) Cluster(cluster string) (kube.Interface, error) {
	restConfig, err := c.ClusterConfig(cluster)
	if err != nil {
		return nil, err
	}

	return kube.NewForConfig(restConfig)
}

func (c *client) Config() *config.CLI {
	return c.config
}

type keyStruct struct {
	Key string
}

// ListVClusters lists all virtual clusters across all projects if virtualClusterName and projectName are empty.
// The list can be narrowed down by the given virtual cluster name and project name.
func (c *client) ListVClusters(ctx context.Context, virtualClusterName, projectName string) ([]*VirtualClusterInstanceProject, error) {
	managementClient, err := c.Management()
	if err != nil {
		return nil, err
	}

	// gather projects and virtual cluster instances to access
	projects := []*managementv1.Project{}
	if projectName != "" {
		project, err := managementClient.Loft().ManagementV1().Projects().Get(ctx, projectName, metav1.GetOptions{})
		if err != nil {
			if kerrors.IsNotFound(err) {
				return nil, fmt.Errorf("couldn't find or access project %s", projectName)
			}

			return nil, err
		}

		projects = append(projects, project)
	} else {
		projectsList, err := managementClient.Loft().ManagementV1().Projects().List(ctx, metav1.ListOptions{})
		if err != nil || len(projectsList.Items) == 0 {
			return nil, err
		}

		for _, p := range projectsList.Items {
			proj := p
			projects = append(projects, &proj)
		}
	}

	// gather virtual cluster instances in those projects
	virtualClusters := []*VirtualClusterInstanceProject{}
	for _, p := range projects {
		if virtualClusterName != "" {
			virtualClusterInstance, err := getProjectVirtualClusterInstance(ctx, managementClient, p, virtualClusterName)
			if err != nil {
				continue
			}

			virtualClusters = append(virtualClusters, virtualClusterInstance)
		} else {
			virtualClusters, err = getProjectVirtualClusterInstances(ctx, managementClient, p)
			if err != nil {
				continue
			}
		}
	}

	return virtualClusters, nil
}

func (c *client) CreateVirtualClusterInstanceOptions(ctx context.Context, config string, projectName string, virtualClusterInstance *managementv1.VirtualClusterInstance, setActive bool) (kubeconfig.ContextOptions, error) {
	var cluster *managementv1.Cluster

	// skip finding cluster if virtual cluster is directly connected
	if !virtualClusterInstance.Spec.NetworkPeer {
		var err error
		cluster, err = c.findProjectCluster(ctx, projectName, virtualClusterInstance.Spec.ClusterRef.Cluster)
		if err != nil {
			return kubeconfig.ContextOptions{}, fmt.Errorf("find space instance cluster: %w", err)
		}
	}

	contextOptions := kubeconfig.ContextOptions{
		Name:       kubeconfig.VirtualClusterInstanceContextName(projectName, virtualClusterInstance.Name),
		ConfigPath: config,
		SetActive:  setActive,
	}
	if virtualClusterInstance.Status.VirtualCluster != nil && virtualClusterInstance.Status.VirtualCluster.AccessPoint.Ingress.Enabled {
		kubeConfig, err := c.getVirtualClusterInstanceAccessConfig(ctx, virtualClusterInstance)
		if err != nil {
			return kubeconfig.ContextOptions{}, fmt.Errorf("retrieve kube config %w", err)
		}

		// get server
		for _, val := range kubeConfig.Clusters {
			contextOptions.Server = val.Server
		}

		contextOptions.InsecureSkipTLSVerify = true
		contextOptions.VirtualClusterAccessPointEnabled = true
	} else {
		contextOptions.Server = c.Config().Platform.Host + "/kubernetes/project/" + projectName + "/virtualcluster/" + virtualClusterInstance.Name
		contextOptions.InsecureSkipTLSVerify = c.Config().Platform.Insecure

		data, err := RetrieveCaData(cluster)
		if err != nil {
			return kubeconfig.ContextOptions{}, err
		}
		contextOptions.CaData = data
	}
	return contextOptions, nil
}

func (c *client) findProjectCluster(ctx context.Context, projectName, clusterName string) (*managementv1.Cluster, error) {
	managementClient, err := c.Management()
	if err != nil {
		return nil, err
	}

	projectClusters, err := managementClient.Loft().ManagementV1().Projects().ListClusters(ctx, projectName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("list project clusters: %w", err)
	}

	for _, cluster := range projectClusters.Clusters {
		if cluster.Name == clusterName {
			return &cluster, nil
		}
	}

	return nil, fmt.Errorf("couldn't find cluster %s in project %s", clusterName, projectName)
}

func (c *client) getVirtualClusterInstanceAccessConfig(ctx context.Context, virtualClusterInstance *managementv1.VirtualClusterInstance) (*clientcmdapi.Config, error) {
	managementClient, err := c.Management()
	if err != nil {
		return nil, err
	}

	kubeConfig, err := managementClient.Loft().ManagementV1().VirtualClusterInstances(virtualClusterInstance.Namespace).GetKubeConfig(
		ctx,
		virtualClusterInstance.Name,
		&managementv1.VirtualClusterInstanceKubeConfig{
			Spec: managementv1.VirtualClusterInstanceKubeConfigSpec{},
		},
		metav1.CreateOptions{},
	)
	if err != nil {
		return nil, err
	}

	// parse kube config string
	clientCfg, err := clientcmd.NewClientConfigFromBytes([]byte(kubeConfig.Status.KubeConfig))
	if err != nil {
		return nil, err
	}

	apiCfg, err := clientCfg.RawConfig()
	if err != nil {
		return nil, err
	}

	return &apiCfg, nil
}

func RetrieveCaData(cluster *managementv1.Cluster) ([]byte, error) {
	if cluster == nil || cluster.Annotations == nil || cluster.Annotations[LoftDirectClusterEndpointCaData] == "" {
		return nil, nil
	}

	data, err := base64.StdEncoding.DecodeString(cluster.Annotations[LoftDirectClusterEndpointCaData])
	if err != nil {
		return nil, fmt.Errorf("error decoding cluster %s annotation: %w", LoftDirectClusterEndpointCaData, err)
	}

	return data, nil
}

func verifyHost(host string) error {
	if !strings.HasPrefix(host, "https") {
		return fmt.Errorf("cannot log into a non https loft instance '%s', please make sure you have TLS enabled", host)
	}

	return nil
}

// VerifyVersion checks if the Loft version is compatible with this CLI version
func VerifyVersion(platformClient Client) error {
	v, err := platformClient.Version()
	if err != nil {
		return err
	} else if v.Version == "v0.0.0" {
		return nil
	}

	backendMajor, err := strconv.Atoi(v.Major)
	if err != nil {
		return perrors.Wrap(err, "parse major version string")
	}

	cliVersionStr := upgrade.GetVersion()
	if cliVersionStr == "" {
		return nil
	}

	cliVersion, err := semver.Parse(cliVersionStr)
	if err != nil {
		return err
	}

	if int(cliVersion.Major) > backendMajor {
		return fmt.Errorf("unsupported %[1]s version %[2]s. Please downgrade your CLI to below v%[3]d.0.0 to support this version, as %[1]s v%[3]d.0.0 and newer versions are incompatible with v%[4]d.x.x", product.DisplayName(), v.Version, cliVersion.Major, backendMajor)
	} else if int(cliVersion.Major) < backendMajor {
		return fmt.Errorf("unsupported %[1]s version %[2]s. Please upgrade your CLI to v%[3]d.0.0 or above to support this version, as %[1]s v%[3]d.0.0 and newer versions are incompatible with v%[4]d.x.x", product.DisplayName(), v.Version, backendMajor, cliVersion.Major)
	}

	return nil
}

func (c *client) Version() (*auth.Version, error) {
	restConfig, err := c.restConfig("")
	if err != nil {
		return nil, err
	}

	restClient, err := kube.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	raw, err := restClient.CoreV1().RESTClient().Get().RequestURI("/version").DoRaw(context.Background())
	if err != nil {
		return nil, perrors.New(fmt.Sprintf("%s\n\nYou may need to login again via `%s login %s --insecure` to allow self-signed certificates\n", err.Error(), os.Args[0], restConfig.Host))
	}

	version := &auth.Version{}
	err = json.Unmarshal(raw, version)
	if err != nil {
		return nil, perrors.Wrap(err, "parse version response")
	}

	return version, nil
}

func (c *client) Login(host string, insecure bool, log log.Logger) error {
	var (
		loginURL   = fmt.Sprintf(LoginPath, host)
		key        keyStruct
		keyChannel = make(chan keyStruct)
	)

	err := verifyHost(host)
	if err != nil {
		return err
	}

	server := startServer(fmt.Sprintf(RedirectPath, host), keyChannel, log)
	err = open.Run(fmt.Sprintf(LoginPath, host))
	if err != nil {
		return fmt.Errorf("couldn't open the login page in a browser: %w. Please use the --access-key flag for the login command. You can generate an access key here: %s", err, fmt.Sprintf(AccessKeyPath, host))
	}

	log.Infof("If the browser does not open automatically, please navigate to %s", loginURL)
	msg := "If you have problems logging in, please navigate to %s/profile/access-keys, click on 'Create Access Key' and then login via '%s %s --access-key ACCESS_KEY"
	if insecure {
		msg += " --insecure"
	}
	msg += "'"
	log.Infof(msg, host, product.LoginCmd(), host)
	log.Infof("Logging into %s...", product.DisplayName())

	key = <-keyChannel

	go func() {
		err = server.Shutdown(context.Background())
		if err != nil {
			log.Debugf("Error shutting down server: %v", err)
		}
	}()

	close(keyChannel)
	return c.LoginWithAccessKey(host, key.Key, insecure)
}

func (c *client) LoginWithAccessKey(host, accessKey string, insecure bool) error {
	err := verifyHost(host)
	if err != nil {
		return err
	}

	platformConfig := c.Config().Platform
	if platformConfig.Host == host && platformConfig.AccessKey == accessKey {
		return nil
	}

	// delete old access key if were logged in before
	if platformConfig.AccessKey != "" {
		managementClient, err := c.Management()
		if err == nil {
			self, err := managementClient.Loft().ManagementV1().Selves().Create(context.TODO(), &managementv1.Self{}, metav1.CreateOptions{})
			if err == nil && self.Status.AccessKey != "" && self.Status.AccessKeyType == storagev1.AccessKeyTypeLogin {
				_ = managementClient.Loft().ManagementV1().OwnedAccessKeys().Delete(context.TODO(), self.Status.AccessKey, metav1.DeleteOptions{})
			}
		}
	}

	platformConfig.Host = host
	platformConfig.Insecure = insecure
	platformConfig.AccessKey = accessKey
	c.Config().Platform = platformConfig

	// verify version
	err = VerifyVersion(c)
	if err != nil {
		return err
	}

	// verify the connection works
	managementClient, err := c.Management()
	if err != nil {
		return perrors.Wrap(err, "create management client")
	}

	// try to get self
	_, err = managementClient.Loft().ManagementV1().Selves().Create(context.TODO(), &managementv1.Self{}, metav1.CreateOptions{})
	if err != nil {
		var urlError *url.Error
		if errors.As(err, &urlError) {
			var err x509.UnknownAuthorityError
			if errors.As(urlError.Err, &err) {
				return fmt.Errorf("unsafe login endpoint '%s', if you wish to login into an insecure loft endpoint run with the '--insecure' flag", c.config.Platform.Host)
			}
		}

		return perrors.Errorf("error logging in: %v", err)
	}

	return c.Save()
}

func (c *client) restConfig(hostSuffix string) (*rest.Config, error) {
	if c.config == nil {
		return nil, perrors.New("no config loaded")
	} else if c.config.Platform.Host == "" || c.config.Platform.AccessKey == "" {
		return nil, perrors.New(fmt.Sprintf("not logged in, please make sure you have run '%s' to create one or '%s [%s]' if one already exists", product.StartCmd(), product.LoginCmd(), product.Url()))
	}

	// build a rest config
	config, err := GetRestConfig(c.config.Platform.Host+hostSuffix, c.config.Platform.AccessKey, c.config.Platform.Insecure)
	if err != nil {
		return nil, err
	}

	return config, err
}

func GetKubeConfig(host, token, namespace string, insecure bool) clientcmd.ClientConfig {
	contextName := "local"
	kubeConfig := clientcmdapi.NewConfig()
	kubeConfig.Contexts = map[string]*clientcmdapi.Context{
		contextName: {
			Cluster:   contextName,
			AuthInfo:  contextName,
			Namespace: namespace,
		},
	}
	kubeConfig.Clusters = map[string]*clientcmdapi.Cluster{
		contextName: {
			Server:                host,
			InsecureSkipTLSVerify: insecure,
		},
	}
	kubeConfig.AuthInfos = map[string]*clientcmdapi.AuthInfo{
		contextName: {
			Token: token,
		},
	}
	kubeConfig.CurrentContext = contextName
	return clientcmd.NewDefaultClientConfig(*kubeConfig, &clientcmd.ConfigOverrides{})
}

func GetRestConfig(host, token string, insecure bool) (*rest.Config, error) {
	config, err := GetKubeConfig(host, token, "", insecure).ClientConfig()
	if err != nil {
		return nil, err
	}
	config.UserAgent = constants.LoftctlUserAgentPrefix + upgrade.GetVersion()

	return config, nil
}

func startServer(redirectURI string, keyChannel chan keyStruct, log log.Logger) *http.Server {
	srv := &http.Server{Addr: ":25843"}

	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		keys, ok := r.URL.Query()["key"]
		if !ok || len(keys[0]) == 0 {
			log.Warn("Login: the key used to login is not valid")
			return
		}

		keyChannel <- keyStruct{
			Key: keys[0],
		}
		http.Redirect(w, r, redirectURI, http.StatusSeeOther)
	})

	go func() {
		// cannot panic, because this probably is an intentional close
		_ = srv.ListenAndServe()
	}()

	// returning reference so caller can call Shutdown()
	return srv
}
