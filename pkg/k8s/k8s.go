package k8s

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/ghodss/yaml"
	"github.com/loft-sh/log/scanner"
	"github.com/loft-sh/vcluster/pkg/etcd"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"golang.org/x/sync/errgroup"
	"k8s.io/klog/v2"
)

const apiserverCmd = "APISERVER_COMMAND"
const schedulerCmd = "SCHEDULER_COMMAND"
const controllerCmd = "CONTROLLER_COMMAND"

type command struct {
	Command []string `json:"command,omitempty"`
}

func StartK8S(ctx context.Context, apiUp chan struct{}, releaseName string) error {
	// we need to retry the functions because etcd is started after the syncer, so
	// the apiservers always fail when we start until etcd is up and running
	// the controller needs the apiserver service to respond to start successfully
	// and the service can only be reachable once the syncer can reach the api servers

	eg := &errgroup.Group{}

	apiCommand := &command{}
	apiEnv, ok := os.LookupEnv(apiserverCmd)
	if ok {
		err := yaml.Unmarshal([]byte(apiEnv), apiCommand)
		if err != nil {
			return fmt.Errorf("parsing apiserver command %s: %w", apiEnv, err)
		}
		eg.Go(func() error {
			_, err := etcd.WaitForEtcdClient(ctx, "/pki", "https://"+releaseName+"-etcd:2379")
			if err != nil {
				return err
			}
			return RunCommand(ctx, *apiCommand, "apiserver")
		})
	}

	isUp := waitForAPI(ctx)
	if !isUp {
		return errors.New("waited until timeout for the api to be up, but it never did")
	}
	close(apiUp)

	controllerCommand := &command{}
	controllerEnv, ok := os.LookupEnv(controllerCmd)
	if ok {
		err := yaml.Unmarshal([]byte(controllerEnv), controllerCommand)
		if err != nil {
			return fmt.Errorf("parsing controller command %s: %w", controllerEnv, err)
		}
		eg.Go(func() error {
			return RunCommand(ctx, *controllerCommand, "controller")
		})
	}

	schedulerCommand := &command{}
	schedulerEnv, ok := os.LookupEnv(schedulerCmd)
	if ok {
		err := yaml.Unmarshal([]byte(schedulerEnv), schedulerCommand)
		if err != nil {
			return fmt.Errorf("parsing scheduler command %s: %w", schedulerEnv, err)
		}
		eg.Go(func() error {
			return RunCommand(ctx, *schedulerCommand, "scheduler")
		})
	}

	err := eg.Wait()

	// regular stop case, will return as soon as a component returns an error.
	// we don't expect the components to stop by themselves since they're supposed
	// to run until killed or until they fail
	if err == nil || err.Error() == "signal: killed" {
		return nil
	}
	return err
}

func RunCommand(ctx context.Context, command command, component string) error {
	reader, writer := io.Pipe()

	done := make(chan struct{})
	// start func
	go func() {
		defer close(done)
		// make sure we scan the output correctly
		scan := scanner.NewScanner(reader)
		for scan.Scan() {
			line := scan.Text()
			if len(line) == 0 {
				continue
			}

			// print to our logs
			args := []interface{}{"component", component}
			loghelper.PrintKlogLine(line, args)
		}
	}()

	// start the command
	klog.InfoS("Starting "+component, "args", strings.Join(command.Command, " "))
	cmd := exec.CommandContext(ctx, command.Command[0], command.Command[1:]...)
	cmd.Stdout = writer
	cmd.Stderr = writer

	err := cmd.Run()
	errPipe := writer.Close()
	if errPipe != nil {
		klog.Errorf("could not close the pipe %s", err.Error())
	}

	// make sure we wait for scanner to be done
	<-done
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
