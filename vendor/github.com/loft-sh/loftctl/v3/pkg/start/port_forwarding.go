package start

import (
	"context"
	"crypto/tls"
	"net/http"
	"time"

	"github.com/loft-sh/api/v3/pkg/product"
	"github.com/loft-sh/loftctl/v3/pkg/clihelper"
	"github.com/loft-sh/loftctl/v3/pkg/config"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

func (l *LoftStarter) startPortForwarding(ctx context.Context, loftPod *corev1.Pod) error {
	stopChan, err := clihelper.StartPortForwarding(ctx, l.RestConfig, l.KubeClient, loftPod, l.LocalPort, l.Log)
	if err != nil {
		return err
	}
	go l.restartPortForwarding(ctx, stopChan)

	// wait until loft is reachable at the given url
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
	l.Log.Infof(product.Replace("Waiting until loft is reachable at https://localhost:%s"), l.LocalPort)
	err = wait.PollUntilContextTimeout(ctx, time.Second, config.Timeout(), true, func(ctx context.Context) (bool, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://localhost:"+l.LocalPort+"/version", nil)
		if err != nil {
			return false, nil
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			return false, nil
		}

		return resp.StatusCode == http.StatusOK, nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (l *LoftStarter) restartPortForwarding(ctx context.Context, stopChan chan struct{}) {
	for {
		<-stopChan
		l.Log.Info("Restart port forwarding")

		// wait for loft pod to start
		l.Log.Info(product.Replace("Waiting until loft pod has been started..."))
		loftPod, err := clihelper.WaitForReadyLoftPod(ctx, l.KubeClient, l.Namespace, l.Log)
		if err != nil {
			l.Log.Fatalf(product.Replace("Error waiting for ready loft pod: %v"), err)
		}

		// restart port forwarding
		stopChan, err = clihelper.StartPortForwarding(ctx, l.RestConfig, l.KubeClient, loftPod, l.LocalPort, l.Log)
		if err != nil {
			l.Log.Fatalf("Error starting port forwarding: %v", err)
		}

		l.Log.Donef("Successfully restarted port forwarding")
	}
}
