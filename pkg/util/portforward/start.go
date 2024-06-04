package portforward

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/loft-sh/log"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/transport/spdy"
)

func StartPortForwardingWithRestart(ctx context.Context, config *rest.Config, address, pod, namespace string, localPort, remotePort string, interrupt chan struct{}, stdout io.Writer, stderr io.Writer, log log.Logger) error {
	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	// restart port forwarding
	stopChan, err := StartPortForwarding(ctx, config, kubeClient, address, pod, namespace, localPort, remotePort, stdout, stderr, log)
	if err != nil {
		return fmt.Errorf("error starting port forwarding: %w", err)
	}

	for {
		select {
		case <-interrupt:
			close(stopChan)
			return nil
		case <-stopChan:
			log.Info("Restarting port forwarding")

			// wait for loft pod to start
			err := wait.PollUntilContextTimeout(ctx, time.Second, time.Minute*10, true, func(ctx context.Context) (done bool, err error) {
				pod, err := kubeClient.CoreV1().Pods(namespace).Get(ctx, pod, metav1.GetOptions{})
				if err != nil {
					return false, nil
				}
				for _, c := range pod.Status.Conditions {
					if c.Type == corev1.PodReady && c.Status == corev1.ConditionTrue {
						return true, nil
					}
				}
				return false, nil
			})
			if err != nil {
				log.Warnf("error waiting for ready vcluster pod: %v", err)
				continue
			}

			// restart port forwarding
			stopChan, err = StartPortForwarding(ctx, config, kubeClient, address, pod, namespace, localPort, remotePort, stdout, stderr, log)
			if err != nil {
				log.Warnf("error starting port forwarding: %v", err)
				continue
			}

			log.Donef("Successfully restarted port forwarding")
		}
	}
}

func StartPortForwarding(ctx context.Context, config *rest.Config, client kubernetes.Interface, address, pod, namespace, localPort, remotePort string, stdout io.Writer, stderr io.Writer, log log.Logger) (chan struct{}, error) {
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
	forwarder, err := NewOnAddresses(dialer, []string{address}, []string{localPort + ":" + remotePort}, stopChan, readyChan, errChan, stdout, stderr)
	if err != nil {
		return nil, err
	}

	go func() {
		err := forwarder.ForwardPorts(ctx)
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
