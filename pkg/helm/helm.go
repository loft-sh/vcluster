package helm

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gofrs/flock"
	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli/values"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"

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

func isChartInstallable(ch *chart.Chart) (bool, error) {
	switch ch.Metadata.Type {
	case "", "application":
		return true, nil
	}
	return false, errors.Errorf("%s charts are not installable", ch.Metadata.Type)
}

// RepoAdd adds repo to local
func (c *client) RepoAdd(name, url string) error {
	repoFile := c.settings.RepositoryConfig

	//Ensure the file directory exists as it is required for file locking
	err := os.MkdirAll(filepath.Dir(repoFile), os.ModePerm)
	if err != nil && !os.IsExist(err) {
		return err
	}

	// Acquire a file lock for process synchronization
	fileLock := flock.New(strings.Replace(repoFile, filepath.Ext(repoFile), ".lock", 1))
	lockCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	locked, err := fileLock.TryLockContext(lockCtx, time.Second)
	if err == nil && locked {
		defer func(fileLock *flock.Flock) {
			_ = fileLock.Unlock()
		}(fileLock)
	}
	if err != nil {
		return err
	}

	b, err := os.ReadFile(repoFile)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	var f repo.File
	if err := yaml.Unmarshal(b, &f); err != nil {
		return err
	}

	if f.Has(name) {
		c.log.Infof("repository name (%s) already exists\n", name)
		return nil
	}

	repoEntry := repo.Entry{
		Name: name,
		URL:  url,
	}

	r, err := repo.NewChartRepository(&repoEntry, getter.All(c.settings))
	if err != nil {
		return err
	}

	if _, err := r.DownloadIndexFile(); err != nil {
		return errors.Wrapf(err, "looks like %q is not a valid chart repository or cannot be reached", url)
	}

	f.Update(&repoEntry)

	if err := f.WriteFile(repoFile, 0644); err != nil {
		return err
	}
	c.log.Infof("%q has been added to your repositories\n", name)
	return nil
}

// RepoUpdate updates charts for all helm repos
func (c *client) RepoUpdate() error {
	repoFile := c.settings.RepositoryConfig

	f, err := repo.LoadFile(repoFile)
	if os.IsNotExist(errors.Cause(err)) || len(f.Repositories) == 0 {
		c.log.Error("no repositories found. You must add one before updating")
		return err
	}
	var repos []*repo.ChartRepository
	for _, cfg := range f.Repositories {
		r, err := repo.NewChartRepository(cfg, getter.All(c.settings))
		if err != nil {
			return err
		}
		repos = append(repos, r)
	}

	c.log.Debugf("Hang tight while we grab the latest from your chart repositories...\n")
	var wg sync.WaitGroup
	for _, re := range repos {
		wg.Add(1)
		go func(re *repo.ChartRepository) {
			defer wg.Done()
			if _, err := re.DownloadIndexFile(); err != nil {
				c.log.Infof("...Unable to get an update from the %q chart repository (%s):\n\t%s\n", re.Config.Name, re.Config.URL, err)
			} else {
				c.log.Infof("...Successfully got an update from the %q chart repository\n", re.Config.Name)
			}
		}(re)
	}
	wg.Wait()
	c.log.Debugf("Update Complete. ⎈ Happy Helming!⎈\n")
	return nil
}

func (c *client) Pull(ctx context.Context, name string, options UpgradeOptions) error {
	newPullClient := action.NewPull()
	if newPullClient.Version == "" && newPullClient.Devel {
		newPullClient.Version = options.Version
	}

	if options.Repo == "" {
		return fmt.Errorf("cannot deploy chart without repo")
	}

	installedRelease, err := newPullClient.Run(name)
	if err != nil {
		return err
	}
	c.log.Info(installedRelease)

	return nil
}

func (c *client) Rollback(ctx context.Context, name, namespace string) error {
	err := c.addSettings(namespace)
	if err != nil {
		return err
	}

	actionConfig, err := c.getActionConfig()
	if err != nil {
		return err
	}

	newRollbackClient := action.NewRollback(actionConfig)

	err = newRollbackClient.Run(name)
	if err != nil {
		return err
	}

	return nil
}

func (c *client) Upgrade(ctx context.Context, name, namespace string, options UpgradeOptions) error {
	err := c.addSettings(namespace)
	if err != nil {
		return err
	}
	defer func(name string) {
		_ = os.Remove(name)
	}(c.settings.KubeConfig)

	const chartPrefix = "loft-sh"
	if options.Path == "" {
		if options.Repo == "" {
			return fmt.Errorf("cannot deploy chart without repo")
		}

		err = c.RepoAdd(chartPrefix, options.Repo)
		if err != nil {
			return err
		}

		err = c.RepoUpdate()
		if err != nil {
			return err
		}
	}

	actionConfig, err := c.getActionConfig()
	if err != nil {
		return err
	}

	newUpgradeClient := action.NewUpgrade(actionConfig)

	if options.Version != "" {
		newUpgradeClient.Version = options.Version
	}

	var chartName string
	if options.Path != "" {
		chartName = options.Path
	} else {
		chartName = fmt.Sprintf("%s/%s", chartPrefix, options.Chart)
	}

	cp, err := newUpgradeClient.ChartPathOptions.LocateChart(chartName, c.settings)
	if err != nil {
		return err
	}
	c.debug("CHART PATH: %s\n", cp)

	// Check chart dependencies to make sure all are present in /charts
	chartRequested, err := loader.Load(cp)
	if err != nil {
		return err
	}

	validInstallableChart, err := isChartInstallable(chartRequested)
	if !validInstallableChart {
		return err
	}

	newUpgradeClient.Namespace = c.settings.Namespace()

	valueOpts := &values.Options{
		ValueFiles:   []string{},
		StringValues: []string{},
		Values:       []string{},
	}

	if len(options.ValuesFiles) > 0 {
		valueOpts.ValueFiles = append(valueOpts.ValueFiles, options.ValuesFiles...)
	}

	if options.Values != "" {
		valueOpts.Values = append(valueOpts.Values, options.Values)
	}

	setStringValues := ""
	for key, value := range options.SetStringValues {
		if setStringValues != "" {
			setStringValues += ","
		}
		setStringValues += key + "=" + value
	}
	if setStringValues != "" {
		valueOpts.StringValues = append(valueOpts.StringValues, setStringValues)
	}

	setValues := ""
	for key, value := range options.SetValues {
		if setValues != "" {
			setValues += ","
		}
		setValues += key + "=" + value
	}
	if setStringValues != "" {
		valueOpts.Values = append(valueOpts.Values, setValues)
	}

	p := getter.All(c.settings)

	vals, err := valueOpts.MergeValues(p)
	if err != nil {
		return err
	}
	newUpgradeClient.Install = true
	updatedRelease, err := newUpgradeClient.Run(name, chartRequested, vals)
	if err != nil {
		return err
	}
	c.log.Info(updatedRelease.Manifest)

	return nil
}

func (c *client) Install(ctx context.Context, name, namespace string, options UpgradeOptions) error {
	err := c.addSettings(namespace)
	if err != nil {
		return err
	}
	// File gets deleted when the below code is added in addSettings method
	defer func(name string) {
		_ = os.Remove(name)
	}(c.settings.KubeConfig)

	const chartPrefix = "loft-sh"
	if options.Path == "" {
		if options.Repo == "" {
			return fmt.Errorf("cannot deploy chart without repo")
		}

		err = c.RepoAdd(chartPrefix, options.Repo)
		if err != nil {
			return err
		}

		err = c.RepoUpdate()
		if err != nil {
			return err
		}
	}

	actionConfig, err := c.getActionConfig()
	if err != nil {
		return err
	}

	newInstallClient := action.NewInstall(actionConfig)
	if options.Version != "" {
		newInstallClient.Version = options.Version
	}

	newInstallClient.ReleaseName = name
	var chartName string
	if options.Path != "" {
		chartName = options.Path
	} else {
		chartName = fmt.Sprintf("%s/%s", chartPrefix, options.Chart)
	}

	cp, err := newInstallClient.ChartPathOptions.LocateChart(chartName, c.settings)
	if err != nil {
		return err
	}
	c.debug("CHART PATH: %s\n", cp)

	// Check chart dependencies to make sure all are present in /charts
	chartRequested, err := loader.Load(cp)
	if err != nil {
		return err
	}
	validInstallableChart, err := isChartInstallable(chartRequested)
	if !validInstallableChart {
		return err
	}

	p := getter.All(c.settings)

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

	valueOpts := &values.Options{
		ValueFiles:   []string{},
		StringValues: []string{},
		Values:       []string{},
	}

	if len(options.ValuesFiles) > 0 {
		valueOpts.ValueFiles = append(valueOpts.ValueFiles, options.ValuesFiles...)
	}

	if options.Values != "" {
		valueOpts.Values = append(valueOpts.Values, options.Values)
	}

	setStringValues := ""
	for key, value := range options.SetStringValues {
		if setStringValues != "" {
			setStringValues += ","
		}
		setStringValues += key + "=" + value
	}
	if setStringValues != "" {
		valueOpts.StringValues = append(valueOpts.StringValues, setStringValues)
	}

	setValues := ""
	for key, value := range options.SetValues {
		if setValues != "" {
			setValues += ","
		}
		setValues += key + "=" + value
	}
	if setStringValues != "" {
		valueOpts.Values = append(valueOpts.Values, setValues)
	}

	vals, err := valueOpts.MergeValues(p)
	if err != nil {
		return err
	}

	newInstallClient.Namespace = c.settings.Namespace()
	installedRelease, err := newInstallClient.Run(chartRequested, vals)
	if err != nil {
		return err
	}
	c.log.Info(installedRelease.Manifest)

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

func (c *client) Exists(name, namespace string) (bool, error) {
	_, err := c.retrieveRelease(name, namespace)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (c *client) Status(ctx context.Context, name, namespace string) (*release.Release, error) {
	err := c.addSettings(namespace)
	if err != nil {
		return nil, err
	}
	return c.retrieveRelease(name, namespace)
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

func (c *client) addSettings(namespace string) error {
	_ = os.Setenv("HELM_NAMESPACE", namespace)
	settings := cli.New()

	kubeConfig, err := WriteKubeConfig(c.config)
	if err != nil {
		return err
	}

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
