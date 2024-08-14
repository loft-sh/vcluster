package platform

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	"github.com/loft-sh/api/v4/pkg/auth"
	"github.com/loft-sh/api/v4/pkg/product"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/config"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/platform/kube"
	"github.com/loft-sh/vcluster/pkg/projectutil"
	"github.com/loft-sh/vcluster/pkg/upgrade"
	perrors "github.com/pkg/errors"
	"github.com/skratchdot/open-golang/open"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
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
	Logout(ctx context.Context) error

	Self() *managementv1.Self
	RefreshSelf(ctx context.Context) error

	Management() (kube.Interface, error)
	Cluster(cluster string) (kube.Interface, error)
	VirtualCluster(cluster, namespace, virtualCluster string) (kube.Interface, error)

	ManagementConfig() (*rest.Config, error)

	Config() *config.CLI
	Save() error

	Version() (*auth.Version, error)
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
	return &client{
		config: config,
	}
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

func (c *client) VirtualClusterConfig(cluster, namespace, virtualCluster string) (*rest.Config, error) {
	return c.restConfig("/kubernetes/virtualcluster/" + cluster + "/" + namespace + "/" + virtualCluster)
}

func (c *client) VirtualCluster(cluster, namespace, virtualCluster string) (kube.Interface, error) {
	restConfig, err := c.VirtualClusterConfig(cluster, namespace, virtualCluster)
	if err != nil {
		return nil, err
	}

	return kube.NewForConfig(restConfig)
}

func (c *client) Config() *config.CLI {
	return c.config
}

func verifyHost(host string) error {
	if !strings.HasPrefix(host, "https") {
		return fmt.Errorf("cannot log into a non https loft instance '%s', please make sure you have TLS enabled", host)
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

type keyStruct struct {
	Key string
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
	config, err := getRestConfig(c.config.Platform.Host+hostSuffix, c.config.Platform.AccessKey, c.config.Platform.CertificateAuthorityData, c.config.Platform.Insecure)
	if err != nil {
		return nil, err
	}

	return config, err
}

func getRestConfig(host, token string, certificateAuthorityData []byte, insecure bool) (*rest.Config, error) {
	config, err := getKubeConfig(host, token, "", certificateAuthorityData, insecure).ClientConfig()
	if err != nil {
		return nil, err
	}
	config.UserAgent = constants.GetVclusterUserAgent() + upgrade.GetVersion()

	return config, nil
}

func getKubeConfig(host, token, namespace string, certificateAuthorityData []byte, insecure bool) clientcmd.ClientConfig {
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
	if len(certificateAuthorityData) > 0 {
		kubeConfig.Clusters[contextName].CertificateAuthorityData = certificateAuthorityData
		kubeConfig.Clusters[contextName].InsecureSkipTLSVerify = false
	}
	kubeConfig.AuthInfos = map[string]*clientcmdapi.AuthInfo{
		contextName: {
			Token: token,
		},
	}
	kubeConfig.CurrentContext = contextName
	return clientcmd.NewDefaultClientConfig(*kubeConfig, &clientcmd.ConfigOverrides{})
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
