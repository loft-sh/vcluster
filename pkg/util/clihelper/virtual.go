package clihelper

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strconv"
	"time"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/find"
	"github.com/loft-sh/vcluster/pkg/util/portforward"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type PortForwardingOptions struct {
	StdOut io.Writer
	StdErr io.Writer
}

func GetVClusterKubeConfig(ctx context.Context, kubeConfig *rest.Config, kubeClient *kubernetes.Clientset, vCluster *find.VCluster, log log.Logger, portForwardingOptions PortForwardingOptions) (*rest.Config, error) {
	var err error
	podName := ""
	waitErr := wait.PollUntilContextTimeout(ctx, time.Second, time.Second*30, true, func(ctx context.Context) (bool, error) {
		// get vcluster pod name
		var pods *corev1.PodList
		pods, err = kubeClient.CoreV1().Pods(vCluster.Namespace).List(ctx, metav1.ListOptions{
			LabelSelector: "app=vcluster,release=" + vCluster.Name,
		})
		if err != nil {
			return false, err
		} else if len(pods.Items) == 0 {
			err = fmt.Errorf("can't find a running vcluster pod in namespace %s", vCluster.Namespace)
			log.Debugf("can't find a running vcluster pod in namespace %s", vCluster.Namespace)
			return false, nil
		}

		// sort by newest
		sort.Slice(pods.Items, func(i, j int) bool {
			return pods.Items[i].CreationTimestamp.Unix() > pods.Items[j].CreationTimestamp.Unix()
		})
		if pods.Items[0].DeletionTimestamp != nil {
			err = fmt.Errorf("can't find a running vcluster pod in namespace %s", vCluster.Namespace)
			log.Debugf("can't find a running vcluster pod in namespace %s", vCluster.Namespace)
			return false, nil
		}

		podName = pods.Items[0].Name
		return true, nil
	})
	if waitErr != nil {
		return nil, fmt.Errorf("finding vcluster pod: %w - %w", waitErr, err)
	}

	log.Infof("Start port-forwarding to virtual cluster")
	vKubeConfig, err := GetKubeConfig(ctx, kubeClient, vCluster.Name, vCluster.Namespace, log)
	if err != nil {
		return nil, fmt.Errorf("failed to parse kube config: %w", err)
	}

	localPort := RandomPort()
	errorChan := make(chan error)
	go func() {
		errorChan <- portforward.StartPortForwardingWithRestart(ctx, kubeConfig, "127.0.0.1", podName, vCluster.Namespace, strconv.Itoa(localPort), "8443", make(chan struct{}), portForwardingOptions.StdOut, portForwardingOptions.StdErr, log)
	}()

	for _, cluster := range vKubeConfig.Clusters {
		if cluster == nil {
			continue
		}
		cluster.Server = "https://localhost:" + strconv.Itoa(localPort)
	}

	restConfig, err := clientcmd.NewDefaultClientConfig(*vKubeConfig, &clientcmd.ConfigOverrides{}).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create rest client config: %w", err)
	}

	vKubeClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create vcluster client: %w", err)
	}

	err = wait.PollUntilContextTimeout(ctx, time.Millisecond*200, time.Minute*3, true, func(ctx context.Context) (bool, error) {
		select {
		case err := <-errorChan:
			return false, err
		default:
			// check if service account exists
			_, err = vKubeClient.CoreV1().ServiceAccounts("default").Get(ctx, "default", metav1.GetOptions{})
			return err == nil, nil
		}
	})
	if err != nil {
		return nil, fmt.Errorf("wait for vcluster to become ready: %w", err)
	}

	return restConfig, nil
}
