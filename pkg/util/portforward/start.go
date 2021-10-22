package portforward

import (
	"fmt"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/log"
	"github.com/loft-sh/vcluster/pkg/util/podhelper"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/transport/spdy"
	"net/http"
	"os"
)

func StartPortForwardingWithRestart(config *rest.Config, address, pod, namespace string, localPort, remotePort string, log log.Logger) error {
	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	// restart port forwarding
	stopChan, err := StartPortForwarding(config, kubeClient, address, pod, namespace, localPort, remotePort, log)
	if err != nil {
		return fmt.Errorf("error starting port forwarding: %v", err)
	}

	for {
		<-stopChan
		log.Info("Restart port forwarding")

		// wait for loft pod to start
		_, err := podhelper.GetVClusterConfig(config, pod, namespace, log)
		if err != nil {
			return fmt.Errorf("error waiting for ready vcluster pod: %v", err)
		}

		// restart port forwarding
		stopChan, err = StartPortForwarding(config, kubeClient, address, pod, namespace, localPort, remotePort, log)
		if err != nil {
			return fmt.Errorf("error starting port forwarding: %v", err)
		}

		log.Donef("Successfully restarted port forwarding")
	}
}

func StartPortForwarding(config *rest.Config, client kubernetes.Interface, address, pod, namespace, localPort, remotePort string, log log.Logger) (chan struct{}, error) {
	log.Info("Starting port-forwarding at " + localPort + ":" + remotePort)
	execRequest := client.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(pod).
		Namespace(namespace).
		SubResource("portforward")

	t, upgrader, err := spdy.RoundTripperFor(config)
	if err != nil {
		return nil, err
	}

	if address == "" {
		address = "localhost"
	}

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: t}, "POST", execRequest.URL())
	errChan := make(chan error)
	readyChan := make(chan struct{})
	stopChan := make(chan struct{})
	forwarder, err := NewOnAddresses(dialer, []string{address}, []string{localPort + ":" + remotePort}, stopChan, readyChan, errChan, os.Stdout, os.Stderr)
	if err != nil {
		return nil, err
	}

	go func() {
		err := forwarder.ForwardPorts()
		if err != nil {
			errChan <- err
		}
	}()

	// wait till ready
	select {
	case err = <-errChan:
		return nil, err
	case <-readyChan:
	case <-stopChan:
		return nil, fmt.Errorf("stopped before ready")
	}

	// start watcher
	go func() {
		for {
			select {
			case <-stopChan:
				return
			case err = <-errChan:
				log.Infof("error during port forwarder: %v", err)
				close(stopChan)
				return
			}
		}
	}()

	return stopChan, nil
}
