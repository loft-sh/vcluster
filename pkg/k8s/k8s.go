package k8s

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/ghodss/yaml"
	"github.com/loft-sh/vcluster/pkg/etcd"
	"github.com/loft-sh/vcluster/pkg/util/commandwriter"
	"golang.org/x/sync/errgroup"
	"k8s.io/klog/v2"
)

const apiServerCmd = "APISERVER_COMMAND"
const schedulerCmd = "SCHEDULER_COMMAND"
const controllerCmd = "CONTROLLER_COMMAND"

type command struct {
	Command []string `json:"command,omitempty"`
}

func StartK8S(ctx context.Context, serviceCIDR string) error {
	serviceCIDRArg := fmt.Sprintf("--service-cluster-ip-range=%s", serviceCIDR)
	eg := &errgroup.Group{}

	// start api server first
	apiEnv, ok := os.LookupEnv(apiServerCmd)
	if ok {
		apiCommand := &command{}
		err := yaml.Unmarshal([]byte(apiEnv), apiCommand)
		if err != nil {
			return fmt.Errorf("parsing apiserver command %s: %w", apiEnv, err)
		}

		apiCommand.Command = append(apiCommand.Command, serviceCIDRArg)
		eg.Go(func() error {
			// get etcd endpoints and certificates from flags
			endpoints, certificates, err := etcd.EndpointsAndCertificatesFromFlags(apiCommand.Command)
			if err != nil {
				return fmt.Errorf("get etcd certificates and endpoint: %w", err)
			}

			// wait until etcd is up and running
			_, err = etcd.WaitForEtcdClient(ctx, certificates, endpoints...)
			if err != nil {
				return err
			}

			// now start the api server
			return RunCommand(ctx, apiCommand, "apiserver")
		})
	}

	// wait for api server to be up as otherwise controller and scheduler might fail
	isUp := waitForAPI(ctx)
	if !isUp {
		return errors.New("waited until timeout for the api to be up, but it never did")
	}

	// start controller command
	controllerEnv, ok := os.LookupEnv(controllerCmd)
	if ok {
		controllerCommand := &command{}
		err := yaml.Unmarshal([]byte(controllerEnv), controllerCommand)
		if err != nil {
			return fmt.Errorf("parsing controller command %s: %w", controllerEnv, err)
		}

		controllerCommand.Command = append(controllerCommand.Command, serviceCIDRArg)
		eg.Go(func() error {
			return RunCommand(ctx, controllerCommand, "controller")
		})
	}

	// start scheduler command
	schedulerEnv, ok := os.LookupEnv(schedulerCmd)
	if ok {
		schedulerCommand := &command{}
		err := yaml.Unmarshal([]byte(schedulerEnv), schedulerCommand)
		if err != nil {
			return fmt.Errorf("parsing scheduler command %s: %w", schedulerEnv, err)
		}

		eg.Go(func() error {
			return RunCommand(ctx, schedulerCommand, "scheduler")
		})
	}

	// regular stop case, will return as soon as a component returns an error.
	// we don't expect the components to stop by themselves since they're supposed
	// to run until killed or until they fail
	err := eg.Wait()
	if err == nil || err.Error() == "signal: killed" {
		return nil
	}
	return err
}

func RunCommand(ctx context.Context, command *command, component string) error {
	writer, err := commandwriter.NewCommandWriter(component)
	if err != nil {
		return err
	}
	defer writer.Writer()

	// start the command
	klog.InfoS("Starting "+component, "args", strings.Join(command.Command, " "))
	cmd := exec.CommandContext(ctx, command.Command[0], command.Command[1:]...)
	cmd.Stdout = writer.Writer()
	cmd.Stderr = writer.Writer()
	err = cmd.Run()

	// make sure we wait for scanner to be done
	writer.CloseAndWait(ctx, err)
	return err
}

// waits for the api to be up, ignoring certs and calling it
// localhost
func waitForAPI(ctx context.Context) bool {
	client := &http.Client{
		Timeout: 2 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	// sometimes the etcd pod takes a very long time to be ready,
	// we might want to fine tune how long we wait later
	for i := 0; i < 60; i++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://127.0.0.1:6443/version", nil)
		if err != nil {
			klog.Errorf("could not create the request to wait for the api: %s", err.Error())
		}
		_, err = client.Do(req)
		switch {
		case errors.Is(err, nil):
			return true
		case errors.Is(err, context.Canceled):
			return false
		default:
			klog.Info("error while targeting the api on localhost, this is expected during the vcluster creation, will retry after 2 seconds:", err)
			time.Sleep(time.Second * 2)
		}
	}
	return false
}
