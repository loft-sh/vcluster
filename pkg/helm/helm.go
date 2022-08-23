package helm

import (
	"context"
	"fmt"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli/values"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/release"
	"os"
	"strings"
	"time"

	vclusterlog "github.com/loft-sh/vcluster/cmd/vclusterctl/log"
	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// UpgradeOptions holds all the options for upgrading / installing a chart
type UpgradeOptions struct {
	Chart string
	Path  string

	Repo            string
	Version         string
	Values          string
	ValuesFiles     []string
	SetValues       map[string]string
	SetStringValues map[string]string

	CreateNamespace bool

	Username string
	Password string
	WorkDir  string

	Insecure bool
	Atomic   bool
	Force    bool
}

// Client defines the interface how to interact with helm
type Client interface {
	Install(ctx context.Context, name, namespace string, options UpgradeOptions) error
	Upgrade(ctx context.Context, name, namespace string, options UpgradeOptions) error
	Pull(ctx context.Context, name string, options UpgradeOptions) error
	Delete(name, namespace string) error
	Exists(name, namespace string) (bool, error)
	Rollback(ctx context.Context, name, namespace string) error
	Status(ctx context.Context, name, namespace string) (*release.Release, error)
}

type client struct {
	config   *clientcmdapi.Config
	log      vclusterlog.Logger
	settings *cli.EnvSettings
}

// NewClient creates a new helm client from the given config
func NewClient(config *clientcmdapi.Config, log vclusterlog.Logger) Client {
	return &client{
		config: config,
		log:    log,
	}
}

func (c *client) Upgrade(ctx context.Context, name, namespace string, options UpgradeOptions) error {
	return c.run(ctx, name, namespace, options, "upgrade", []string{"--install", "--repository-config=''"})
}

func (c *client) Pull(ctx context.Context, name string, options UpgradeOptions) error {
	return c.run(ctx, name, "", options, "pull", []string{})
}

func (c *client) Rollback(ctx context.Context, name, namespace string) error {
	return c.run(ctx, name, namespace, UpgradeOptions{}, "rollback", []string{})
}

func isChartInstallable(ch *chart.Chart) (bool, error) {
	switch ch.Metadata.Type {
	case "", "application":
		return true, nil
	}
	return false, errors.Errorf("%s charts are not installable", ch.Metadata.Type)
}

func (c *client) Install(ctx context.Context, name, namespace string, options UpgradeOptions) error {
	err := c.addSettings(namespace)
	if err != nil {
		return err
	}
	actionConfig, err := c.getActionConfig()
	if err != nil {
		return err
	}

	newInstallClient := action.NewInstall(actionConfig)
	if newInstallClient.Version == "" && newInstallClient.Devel {
		newInstallClient.Version = ">0.0.0-0"
	}

	newInstallClient.ReleaseName = name
	var cp string
	if options.Path != "" {
		newInstallClient.RepoURL = options.Path
	} else if options.Chart != "" {
		if options.Repo == "" {
			return fmt.Errorf("cannot deploy chart without repo")
		}
		if options.Version != "" {
			newInstallClient.Version = options.Version
		}
		if options.Username != "" {
			newInstallClient.Username = options.Username
		}
		if options.Password != "" {
			newInstallClient.Password = options.Password
		}
		cp, err := newInstallClient.ChartPathOptions.LocateChart(fmt.Sprintf("%s/%s", options.Repo, options.Chart), c.settings)
		if err != nil {
			return err
		}
		c.debug("CHART PATH: %s\n", cp)
	}

	cp, err = newInstallClient.ChartPathOptions.LocateChart(fmt.Sprintf("%s/%s", options.Repo, options.Chart), c.settings)
	if err != nil {
		return err
	}

	c.debug("CHART PATH: %s\n", cp)

	p := getter.All(c.settings)

	setStringValues := ""
	for key, value := range options.SetStringValues {
		if setStringValues != "" {
			setStringValues += ","
		}

		setStringValues += key + "=" + value
	}

	setValues := ""
	for key, value := range options.SetValues {
		if setValues != "" {
			setValues += ","
		}

		setValues += key + "=" + value
	}

	valueOpts := &values.Options{
		ValueFiles:   options.ValuesFiles,
		StringValues: []string{setStringValues},
		Values:       []string{options.Values, setValues},
	}

	vals, err := valueOpts.MergeValues(p)
	if err != nil {
		return err
	}

	// Check chart dependencies to make sure all are present in /charts
	chartRequested, err := loader.Load(cp)
	if err != nil {
		return err
	}

	validInstallableChart, err := isChartInstallable(chartRequested)
	if !validInstallableChart {
		return err
	}

	if req := chartRequested.Metadata.Dependencies; req != nil {
		// If CheckDependencies returns an error, we have unfulfilled dependencies.
		// As of Helm 2.4.0, this is treated as a stopping condition:
		// https://github.com/helm/helm/issues/2209
		if err := action.CheckDependencies(chartRequested, req); err != nil {
			if newInstallClient.DependencyUpdate {
				man := &downloader.Manager{
					Out:              os.Stdout,
					ChartPath:        cp,
					Keyring:          newInstallClient.ChartPathOptions.Keyring,
					SkipUpdate:       false,
					Getters:          p,
					RepositoryConfig: c.settings.RepositoryConfig,
					RepositoryCache:  c.settings.RepositoryCache,
				}
				if err := man.Update(); err != nil {
					return err
				}
			} else {
				return err
			}
		}
	}

	newInstallClient.Namespace = c.settings.Namespace()
	release, err := newInstallClient.Run(chartRequested, vals)
	if err != nil {
		return err
	}
	fmt.Println(release.Manifest)

	return nil
}

func (c *client) run(ctx context.Context, name, namespace string, options UpgradeOptions, command string, extraArgs []string) error {
	return nil
}

func (c *client) Delete(name, namespace string) error {
	c.log.Debugf("Delete release '%s' in namespace '%s'", name, namespace)

	err := c.addSettings(namespace)
	if err != nil {
		return err
	}

	actionConfig, err := c.getActionConfig()
	if err != nil {
		return err
	}

	_, err = action.NewUninstall(actionConfig).Run(name)

	if err != nil {
		if strings.Contains(err.Error(), "release: not found") {
			return fmt.Errorf("release '%s' was not found in namespace '%s'", name, namespace)
		}
		return fmt.Errorf("error executing delete: %s", err)
	}

	return nil
}

func (c *client) addSettings(namespace string) error {
	_ = os.Setenv("HELM_NAMESPACE", namespace)
	settings := cli.New()

	kubeConfig, err := WriteKubeConfig(c.config)
	if err != nil {
		return err
	}
	defer func(name string) {
		_ = os.Remove(name)
	}(kubeConfig)

	settings.KubeConfig = kubeConfig
	c.settings = settings
	return nil
}

func (c *client) debug(format string, v ...interface{}) {
	c.log.Debug(2, fmt.Sprintf(format, v...))
}

func (c *client) getActionConfig() (*action.Configuration, error) {
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(c.settings.RESTClientGetter(), c.settings.Namespace(), os.Getenv("HELM_DRIVER"), c.debug); err != nil {
		return nil, err
	}
	return actionConfig, nil
}

func (c *client) Exists(name, namespace string) (bool, error) {
	_, err := c.retrieveRelease(name, namespace)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (c *client) retrieveRelease(name string, namespace string) (*release.Release, error) {
	err := c.addSettings(namespace)
	if err != nil {
		return nil, err
	}

	actionConfig, err := c.getActionConfig()
	if err != nil {
		return nil, err
	}

	releaseDetails, err := action.NewStatus(actionConfig).Run(name)
	if err != nil {
		if strings.Contains(err.Error(), "release: not found") {
			return nil, err
		}
		return nil, fmt.Errorf("error executing release status: %s", err.Error())
	}

	return releaseDetails, nil
}

func (c *client) Status(ctx context.Context, name, namespace string) (*release.Release, error) {
	err := c.addSettings(namespace)
	if err != nil {
		return nil, err
	}
	return c.retrieveRelease(name, namespace)
}

// WriteKubeConfig writes the kubeconfig to a file and returns the filename
func WriteKubeConfig(configRaw *clientcmdapi.Config) (string, error) {
	data, err := clientcmd.Write(*configRaw)
	if err != nil {
		return "", err
	}

	// Create temp file
	tempFile, err := os.CreateTemp("", "")
	if err != nil {
		return "", errors.Wrap(err, "create temp file")
	}

	// Write to temp file
	_, err = tempFile.Write(data)
	if err != nil {
		_ = os.Remove(tempFile.Name())
		return "", errors.Wrap(err, "write temp file")
	}

	// Close temp file
	_ = tempFile.Close()

	// Okay sometimes the file is written so quickly that helm somehow
	// cannot read it immediately which causes errors
	// so we wait here till the file is ready
	now := time.Now()
	for time.Since(now) < time.Minute {
		_, err = os.Stat(tempFile.Name())
		if err != nil {
			if os.IsNotExist(err) {
				time.Sleep(time.Millisecond * 50)
				continue
			}

			_ = os.Remove(tempFile.Name())
			return "", err
		}

		break
	}

	return tempFile.Name(), nil
}
