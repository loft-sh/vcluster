package helm

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/loft-sh/log"
	"github.com/pkg/errors"
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
	SetValues       []string
	SetStringValues []string

	CreateNamespace bool

	Username string
	Password string
	WorkDir  string

	Insecure bool
	Atomic   bool
	Force    bool
	Debug    bool
}

const (
	errorExecutingHelm = "error executing helm %s: %s"
	errorTimeout       = "error executing helm %s: %s operation timedout"
)

// Client defines the interface how to interact with helm
type Client interface {
	Install(ctx context.Context, name, namespace string, options UpgradeOptions) error
	Upgrade(ctx context.Context, name, namespace string, options UpgradeOptions) error
	Pull(ctx context.Context, name string, options UpgradeOptions) error
	Delete(name, namespace string) error
	Exists(name, namespace string) (bool, error)
	Rollback(ctx context.Context, name, namespace string) error
	Status(ctx context.Context, name, namespace string) ([]byte, error)
}

type client struct {
	config   *clientcmdapi.Config
	log      log.Logger
	helmPath string
}

// NewClient creates a new helm client from the given config
func NewClient(config *clientcmdapi.Config, log log.Logger, commandPath string) Client {
	return &client{
		config:   config,
		log:      log,
		helmPath: commandPath,
	}
}

func (c *client) Install(ctx context.Context, name, namespace string, options UpgradeOptions) error {
	return c.run(ctx, name, namespace, options, "install", []string{"--repository-config=''"})
}

func (c *client) Upgrade(ctx context.Context, name, namespace string, options UpgradeOptions) error {
	return c.run(ctx, name, namespace, options, "upgrade", []string{"--install", "--repository-config=''"})
}

func (c *client) Pull(ctx context.Context, name string, options UpgradeOptions) error {
	return c.pull(ctx, name, options)
}

func (c *client) Rollback(ctx context.Context, name, namespace string) error {
	return c.run(ctx, name, namespace, UpgradeOptions{}, "rollback", []string{})
}

func (c *client) run(ctx context.Context, name, namespace string, options UpgradeOptions, command string, extraArgs []string) error {
	kubeConfig, err := WriteKubeConfig(c.config)
	if err != nil {
		return err
	}
	defer os.Remove(kubeConfig)

	args := []string{command, name}
	if options.Path != "" {
		args = append(args, options.Path)
	} else {
		if options.Chart != "" {
			args = append(args, options.Chart)
		}
		if options.Repo != "" {
			args = append(args, "--repo", options.Repo)
		}
		if options.Version != "" {
			args = append(args, "--version", options.Version)
		}
	}

	if options.CreateNamespace {
		args = append(args, "--create-namespace")
	}
	if options.Insecure {
		args = append(args, "--insecure-skip-tls-verify")
	}

	args = append(args, "--kubeconfig", kubeConfig, "--namespace", namespace)
	args = append(args, extraArgs...)

	// Values
	if options.Values != "" {
		// Create temp file
		tempFile, err := os.CreateTemp("", "")
		if err != nil {
			return errors.Wrap(err, "create temp file")
		}

		// Write to temp file
		_, err = tempFile.Write([]byte(options.Values))
		if err != nil {
			os.Remove(tempFile.Name())
			return errors.Wrap(err, "write temp file")
		}

		// Close temp file
		tempFile.Close()
		defer os.Remove(tempFile.Name())

		// Wait quickly so helm will find the file
		time.Sleep(time.Millisecond)

		args = append(args, "--values", tempFile.Name())
	}

	// Values files
	if len(options.ValuesFiles) > 0 {
		for _, file := range options.ValuesFiles {
			args = append(args, "--values", file)
		}
	}

	// Set values
	for _, value := range options.SetValues {
		args = append(args, "--set", value)
	}

	// Set string values
	for _, value := range options.SetStringValues {
		args = append(args, "--set-string", value)
	}

	// force
	if options.Force {
		args = append(args, "--force")
	}
	if options.Atomic {
		args = append(args, "--atomic")
	}
	if options.Debug {
		args = append(args, "--debug")
	}

	return c.execute(ctx, args, command, options.WorkDir)
}

func (c *client) pull(ctx context.Context, name string, options UpgradeOptions) error {
	kubeConfig, err := WriteKubeConfig(c.config)
	if err != nil {
		return err
	}
	defer os.Remove(kubeConfig)

	if options.Repo == "" {
		return fmt.Errorf("cannot deploy chart without repo")
	}

	if options.Username != "" && options.Password != "" {
		// login
		err = c.login(ctx, options)
		if err != nil {
			return fmt.Errorf("error login to registry: %w", err)
		}
	}
	defer c.logout(ctx, options)

	args := []string{"pull"}

	if strings.HasPrefix(options.Repo, "oci://") {
		fullChart := options.Repo + "/" + options.Chart
		args = append(args, fullChart)
	} else {
		args = append(args, name, options.Chart)
		args = append(args, "--repo", options.Repo)
	}

	if options.Version != "" {
		args = append(args, "--version", options.Version)
	}

	if options.Insecure {
		args = append(args, "--insecure-skip-tls-verify")
	}

	return c.execute(ctx, args, "pull", options.WorkDir)
}

func (c *client) login(ctx context.Context, options UpgradeOptions) error {
	url, err := url.Parse(options.Repo)
	if err != nil {
		return fmt.Errorf("error login in, repo is not a valid URL: %s", options.Repo)
	}
	host := url.Hostname()
	loginArgs := []string{"registry", "login", "--username", options.Username, "--password", options.Password, host}
	if options.Insecure {
		loginArgs = append(loginArgs, "--insecure")
	}
	return c.execute(ctx, loginArgs, "login", options.WorkDir)
}

func (c *client) logout(ctx context.Context, options UpgradeOptions) {
	url, err := url.Parse(options.Repo)
	if err != nil {
		return
	}
	host := url.Hostname()
	logoutArgs := []string{"registry", "logout", host}
	if options.Insecure {
		logoutArgs = append(logoutArgs, "--insecure")
	}
	_ = c.execute(ctx, logoutArgs, "login", "")
}

func (c *client) execute(ctx context.Context, args []string, operation string, workdir string) error {
	c.log.Info("execute command: helm " + strings.Join(args, " "))
	cmd := exec.CommandContext(ctx, c.helmPath, args...)

	if workdir != "" {
		cmd.Dir = workdir
	}

	output, err := cmd.CombinedOutput()

	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return fmt.Errorf(errorTimeout, string(output), operation)
	}
	if err != nil {
		return fmt.Errorf(errorExecutingHelm, strings.Join(args, " "), string(output))
	}
	return nil
}

func (c *client) Delete(name, namespace string) error {
	kubeConfig, err := WriteKubeConfig(c.config)
	if err != nil {
		return err
	}
	defer os.Remove(kubeConfig)

	args := []string{"delete", name, "--namespace", namespace, "--kubeconfig", kubeConfig, "--repository-config=''"}

	c.log.Debug("Delete helm chart with helm " + strings.Join(args, " "))
	output, err := exec.Command(c.helmPath, args...).CombinedOutput()
	if err != nil {
		if strings.Contains(string(output), "release: not found") {
			return fmt.Errorf("release '%s' was not found in namespace '%s'", name, namespace)
		}

		return fmt.Errorf("error executing helm delete: %s", string(output))
	}

	return nil
}

func (c *client) Exists(name, namespace string) (bool, error) {
	kubeConfig, err := WriteKubeConfig(c.config)
	if err != nil {
		return false, err
	}
	defer os.Remove(kubeConfig)

	args := []string{"status", name, "--namespace", namespace, "--kubeconfig", kubeConfig}
	output, err := exec.Command(c.helmPath, args...).CombinedOutput()
	if err != nil {
		if strings.Contains(string(output), "release: not found") {
			return false, nil
		}

		return false, fmt.Errorf("error executing helm status: %s", string(output))
	}

	return true, nil
}

func (c *client) Status(ctx context.Context, name, namespace string) ([]byte, error) {
	kubeConfig, err := WriteKubeConfig(c.config)
	if err != nil {
		return nil, err
	}
	defer os.Remove(kubeConfig)

	args := []string{"status", name, "--namespace", namespace, "--kubeconfig", kubeConfig}
	return exec.CommandContext(ctx, c.helmPath, args...).CombinedOutput()
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
		os.Remove(tempFile.Name())
		return "", errors.Wrap(err, "write temp file")
	}

	// Close temp file
	tempFile.Close()

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

			os.Remove(tempFile.Name())
			return "", err
		}

		break
	}

	return tempFile.Name(), nil
}
