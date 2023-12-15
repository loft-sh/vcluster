package start

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/denisbrodbeck/machineid"
	"github.com/loft-sh/api/v3/pkg/product"
	"github.com/loft-sh/loftctl/v3/pkg/clihelper"
	"github.com/loft-sh/log"
	"github.com/loft-sh/log/hash"
	"github.com/loft-sh/log/scanner"
	"github.com/mgutz/ansi"
	"github.com/mitchellh/go-homedir"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
)

var (
	ErrMissingContainer = errors.New("missing container")
	ErrLoftNotReachable = errors.New("product is not reachable")
)

type ContainerDetails struct {
	NetworkSettings ContainerNetworkSettings `json:"NetworkSettings,omitempty"`
	State           ContainerDetailsState    `json:"State,omitempty"`
	ID              string                   `json:"ID,omitempty"`
	Created         string                   `json:"Created,omitempty"`
	Config          ContainerDetailsConfig   `json:"Config,omitempty"`
}

type ContainerNetworkSettings struct {
	Ports map[string][]ContainerPort `json:"ports,omitempty"`
}

type ContainerPort struct {
	HostIP   string `json:"HostIp,omitempty"`
	HostPort string `json:"HostPort,omitempty"`
}

type ContainerDetailsConfig struct {
	Labels map[string]string `json:"Labels,omitempty"`
	Image  string            `json:"Image,omitempty"`
	User   string            `json:"User,omitempty"`
	Env    []string          `json:"Env,omitempty"`
}

type ContainerDetailsState struct {
	Status    string `json:"Status,omitempty"`
	StartedAt string `json:"StartedAt,omitempty"`
}

func (l *LoftStarter) startDocker(ctx context.Context, name string) error {
	l.Log.Infof(product.Replace("Starting loft in Docker..."))

	// prepare installation
	err := l.prepareDocker()
	if err != nil {
		return err
	}

	// try to find loft container
	containerID, err := l.findLoftContainer(ctx, name, true)
	if err != nil {
		return err
	}

	// check if container is there
	if containerID != "" && (l.Reset || l.Upgrade) {
		l.Log.Info(product.Replace("Existing Loft instance found."))
		err = l.uninstallDocker(ctx, containerID)
		if err != nil {
			return err
		}

		containerID = ""
	}

	// Use default password if none is set
	if l.Password == "" {
		l.Password = getMachineUID(l.Log)
	}

	// check if is installed
	if containerID != "" {
		l.Log.Info(product.Replace("Existing Loft instance found. Run with --upgrade to apply new configuration"))
		return l.successDocker(ctx, containerID)
	}

	// Install Loft
	l.Log.Info(product.Replace("Welcome to Loft!"))
	l.Log.Info(product.Replace("This installer will help you configure and deploy Loft."))

	// make sure we are ready for installing
	containerID, err = l.runLoftInDocker(ctx, name)
	if err != nil {
		return err
	} else if containerID == "" {
		return fmt.Errorf("%w: %s", ErrMissingContainer, product.Replace("couldn't find Loft container after starting it"))
	}

	return l.successDocker(ctx, containerID)
}

func (l *LoftStarter) successDocker(ctx context.Context, containerID string) error {
	if l.NoWait {
		return nil
	}

	// wait until Loft is ready
	host, err := l.waitForLoftDocker(ctx, containerID)
	if err != nil {
		return err
	}

	// wait for domain to become reachable
	l.Log.Infof(product.Replace("Wait for Loft to become available at %s..."), host)
	err = wait.PollUntilContextTimeout(ctx, time.Second, time.Minute*10, true, func(ctx context.Context) (bool, error) {
		containerDetails, err := l.inspectContainer(ctx, containerID)
		if err != nil {
			return false, fmt.Errorf("inspect loft container: %w", err)
		} else if strings.ToLower(containerDetails.State.Status) == "exited" || strings.ToLower(containerDetails.State.Status) == "dead" {
			logs, _ := l.logsContainer(ctx, containerID)
			return false, fmt.Errorf("container failed (status: %s):\n %s", containerDetails.State.Status, logs)
		}

		return clihelper.IsLoftReachable(ctx, host)
	})
	if err != nil {
		return fmt.Errorf(product.Replace("error waiting for loft: %v%w"), err)
	}

	// print success message
	PrintSuccessMessageDockerInstall(host, l.Password, l.Log)
	return nil
}

func PrintSuccessMessageDockerInstall(host, password string, log log.Logger) {
	url := "https://" + host
	log.WriteString(logrus.InfoLevel, fmt.Sprintf(product.Replace(`


##########################   LOGIN   ############################

Username: `+ansi.Color("admin", "green+b")+`
Password: `+ansi.Color(password, "green+b")+`

Login via UI:  %s
Login via CLI: %s

#################################################################

Loft was successfully installed and can now be reached at: %s

Thanks for using Loft!
`),
		ansi.Color(url, "green+b"),
		ansi.Color(product.LoginCmd()+" "+url, "green+b"),
		url,
	))
}

func (l *LoftStarter) waitForLoftDocker(ctx context.Context, containerID string) (string, error) {
	l.Log.Info(product.Replace("Wait for Loft to become available..."))

	// check for local port
	containerDetails, err := l.inspectContainer(ctx, containerID)
	if err != nil {
		return "", err
	} else if len(containerDetails.NetworkSettings.Ports) > 0 && len(containerDetails.NetworkSettings.Ports["10443/tcp"]) > 0 {
		return "localhost:" + containerDetails.NetworkSettings.Ports["10443/tcp"][0].HostPort, nil
	}

	// check if no tunnel
	if l.NoTunnel {
		return "", fmt.Errorf("%w: %s", ErrLoftNotReachable, product.Replace("cannot connect to Loft as it has no exposed port and --no-tunnel is enabled"))
	}

	// wait for router
	url := ""
	waitErr := wait.PollUntilContextTimeout(ctx, time.Second, time.Minute*10, true, func(ctx context.Context) (bool, error) {
		url, err = l.findLoftRouter(ctx, containerID)
		if err != nil {
			return false, nil
		}

		return true, nil
	})
	if waitErr != nil {
		return "", fmt.Errorf("error waiting for loft router domain: %w", err)
	}

	return url, nil
}

func (l *LoftStarter) findLoftRouter(ctx context.Context, id string) (string, error) {
	out, err := l.buildDockerCmd(ctx, "exec", id, "cat", "/var/lib/loft/loft-domain.txt").Output()
	if err != nil {
		return "", WrapCommandError(out, err)
	}

	return strings.TrimSpace(string(out)), nil
}

func (l *LoftStarter) prepareDocker() error {
	// test for helm and kubectl
	_, err := exec.LookPath("docker")
	if err != nil {
		return fmt.Errorf("seems like docker is not installed. Docker is required for the installation of loft. Please visit https://docs.docker.com/engine/install/ for install instructions")
	}

	output, err := exec.Command("docker", "ps").CombinedOutput()
	if err != nil {
		return fmt.Errorf("seems like there are issues with your docker cli: \n\n%s", output)
	}

	return nil
}

func (l *LoftStarter) uninstallDocker(ctx context.Context, id string) error {
	l.Log.Infof(product.Replace("Uninstalling loft..."))

	// stop container
	out, err := l.buildDockerCmd(ctx, "stop", id).Output()
	if err != nil {
		return fmt.Errorf("stop container: %w", WrapCommandError(out, err))
	}

	// remove container
	out, err = l.buildDockerCmd(ctx, "rm", id).Output()
	if err != nil {
		return fmt.Errorf("remove container: %w", WrapCommandError(out, err))
	}

	return nil
}

func (l *LoftStarter) runLoftInDocker(ctx context.Context, name string) (string, error) {
	args := []string{"run", "-d", "--name", name}
	if l.NoTunnel {
		args = append(args, "--env", "DISABLE_LOFT_ROUTER=true")
	}
	if l.Password != "" {
		args = append(args, "--env", "ADMIN_PASSWORD_HASH="+hash.String(l.Password))
	}

	// run as root otherwise we get permission errors
	args = append(args, "-u", "root")

	// mount the loft lib
	args = append(args, "-v", "loft-data:/var/lib/loft")

	// set port
	if l.LocalPort != "" {
		args = append(args, "-p", l.LocalPort+":10443")
	}

	// set extra args
	args = append(args, l.DockerArgs...)

	// set image
	if l.DockerImage != "" {
		args = append(args, l.DockerImage)
	} else if l.Version != "" {
		args = append(args, "ghcr.io/loft-sh/loft:"+strings.TrimPrefix(l.Version, "v"))
	} else {
		args = append(args, "ghcr.io/loft-sh/loft:latest")
	}

	l.Log.Infof("Start Loft via 'docker %s'", strings.Join(args, " "))
	runCmd := l.buildDockerCmd(ctx, args...)
	runCmd.Stdout = os.Stdout
	runCmd.Stderr = os.Stderr
	err := runCmd.Run()
	if err != nil {
		return "", err
	}

	return l.findLoftContainer(ctx, name, false)
}

func (l *LoftStarter) logsContainer(ctx context.Context, id string) (string, error) {
	args := []string{"logs", id}
	out, err := l.buildDockerCmd(ctx, args...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("logs container: %w", WrapCommandError(out, err))
	}

	return string(out), nil
}

func (l *LoftStarter) inspectContainer(ctx context.Context, id string) (*ContainerDetails, error) {
	args := []string{"inspect", "--type", "container", id}
	out, err := l.buildDockerCmd(ctx, args...).Output()
	if err != nil {
		return nil, fmt.Errorf("inspect container: %w", WrapCommandError(out, err))
	}

	containerDetails := []*ContainerDetails{}
	err = json.Unmarshal(out, &containerDetails)
	if err != nil {
		return nil, fmt.Errorf("parse inspect output: %w", err)
	} else if len(containerDetails) == 0 {
		return nil, fmt.Errorf("coudln't find container %s", id)
	}

	return containerDetails[0], nil
}

func (l *LoftStarter) removeContainer(ctx context.Context, id string) error {
	args := []string{"rm", id}
	out, err := l.buildDockerCmd(ctx, args...).Output()
	if err != nil {
		return fmt.Errorf("remove container: %w", WrapCommandError(out, err))
	}

	return nil
}

func (l *LoftStarter) findLoftContainer(ctx context.Context, name string, onlyRunning bool) (string, error) {
	args := []string{"ps", "-q", "-a", "-f", "name=^" + name + "$"}
	out, err := l.buildDockerCmd(ctx, args...).Output()
	if err != nil {
		// fallback to manual search
		return "", fmt.Errorf("error finding container: %w", WrapCommandError(out, err))
	}

	arr := []string{}
	scan := scanner.NewScanner(bytes.NewReader(out))
	for scan.Scan() {
		arr = append(arr, strings.TrimSpace(scan.Text()))
	}
	if len(arr) == 0 {
		return "", nil
	}

	// remove the failed / exited containers
	runningContainerID := ""
	for _, containerID := range arr {
		containerState, err := l.inspectContainer(ctx, containerID)
		if err != nil {
			return "", err
		} else if onlyRunning && strings.ToLower(containerState.State.Status) != "running" {
			err = l.removeContainer(ctx, containerID)
			if err != nil {
				return "", err
			}
		} else {
			runningContainerID = containerID
		}
	}

	return runningContainerID, nil
}

func (l *LoftStarter) buildDockerCmd(ctx context.Context, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "docker", args...)
	return cmd
}

func WrapCommandError(stdout []byte, err error) error {
	if err == nil {
		return nil
	}

	return &Error{
		stdout: stdout,
		err:    err,
	}
}

type Error struct {
	err    error
	stdout []byte
}

func (e *Error) Error() string {
	message := ""
	if len(e.stdout) > 0 {
		message += string(e.stdout) + "\n"
	}

	var exitError *exec.ExitError
	if errors.As(e.err, &exitError) && len(exitError.Stderr) > 0 {
		message += string(exitError.Stderr) + "\n"
	}

	return message + e.err.Error()
}

func getMachineUID(log log.Logger) string {
	id, err := machineid.ID()
	if err != nil {
		id = "error"
		if log != nil {
			log.Debugf("Error retrieving machine uid: %v", err)
		}
	}
	// get $HOME to distinguish two users on the same machine
	// will be hashed later together with the ID
	home, err := homedir.Dir()
	if err != nil {
		home = "error"
		if log != nil {
			log.Debugf("Error retrieving machine home: %v", err)
		}
	}
	mac := hmac.New(sha256.New, []byte(id))
	mac.Write([]byte(home))
	return fmt.Sprintf("%x", mac.Sum(nil))
}
