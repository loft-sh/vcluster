package clihelper

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/find"
	"github.com/loft-sh/vcluster/pkg/cli/podprinter"
	utilkubeconfig "github.com/loft-sh/vcluster/pkg/util/kubeconfig"
	"github.com/pkg/errors"
	"golang.org/x/mod/semver"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

const MinHelmVersion = "v3.10.0"

// CriticalStatus container status
var CriticalStatus = map[string]bool{
	"Error":                      true,
	"Unknown":                    true,
	"ImagePullBackOff":           true,
	"CrashLoopBackOff":           true,
	"RunContainerError":          true,
	"ErrImagePull":               true,
	"CreateContainerConfigError": true,
	"InvalidImageName":           true,
}

var SortPodsByNewest = func(pods []corev1.Pod, i, j int) bool {
	return pods[i].CreationTimestamp.Unix() > pods[j].CreationTimestamp.Unix()
}

// GetKubeConfig attempts to read the kubeconfig from the default Secret and
// falls back to reading from filesystem if the Secret is not read successfully.
// Reading from filesystem is implemented for the backward compatibility and
// can be eventually removed in the future.
//
// This is retried until the kube config is successfully retrieve, or until 10 minute timeout is reached.
func GetKubeConfig(ctx context.Context, kubeClient *kubernetes.Clientset, vclusterName string, namespace string, log log.Logger) (*clientcmdapi.Config, error) {
	var kubeConfig *clientcmdapi.Config

	printedWaiting := false
	podInfoPrinter := podprinter.PodInfoPrinter{LastWarning: time.Now().Add(time.Second * 6)}
	err := wait.PollUntilContextTimeout(ctx, time.Second, time.Minute*10, true, func(ctx context.Context) (done bool, err error) {
		podList, err := kubeClient.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
			LabelSelector: "app=vcluster,release=" + vclusterName,
		})
		if err != nil {
			return false, err
		} else if len(podList.Items) > 0 {
			sort.Slice(podList.Items, func(i, j int) bool {
				return SortPodsByNewest(podList.Items, i, j)
			})
			if HasPodProblem(&podList.Items[0]) {
				if !printedWaiting {
					log.Infof("Waiting for vcluster to come up...")
					printedWaiting = true
				}

				podInfoPrinter.PrintPodWarning(ctx, kubeClient, &podList.Items[0], log)
				return false, nil
			} else if !allContainersReady(&podList.Items[0]) {
				if !printedWaiting {
					log.Infof("Waiting for vcluster to come up...")
					printedWaiting = true
				}

				if find.GetPodStatus(&podList.Items[0]) != "Running" {
					podInfoPrinter.PrintPodInfo(&podList.Items[0], log)
				}
				return false, nil
			}
		}

		kubeConfig, err = utilkubeconfig.ReadKubeConfig(ctx, kubeClient, vclusterName, namespace)
		if err != nil {
			return false, nil
		}
		log.Done("vCluster is up and running")
		return true, nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "wait for vcluster")
	}

	return kubeConfig, nil
}

func allContainersReady(pod *corev1.Pod) bool {
	for _, cs := range pod.Status.ContainerStatuses {
		if !cs.Ready || cs.State.Running == nil {
			return false
		}
	}
	return true
}

func HasPodProblem(pod *corev1.Pod) bool {
	status := find.GetPodStatus(pod)
	status = strings.TrimPrefix(status, "Init:")
	return CriticalStatus[status]
}

func CheckHelmVersion(output string) error {
	if semver.Compare(output, MinHelmVersion) == -1 {
		return fmt.Errorf("please ensure that the \"helm\" binary in your PATH is valid. Currently only Helm >= %s is supported", MinHelmVersion)
	}

	return nil
}

func UpdateKubeConfig(contextName string, cluster *clientcmdapi.Cluster, authInfo *clientcmdapi.AuthInfo, setActive bool) error {
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{}).RawConfig()
	if err != nil {
		return err
	}

	config.Clusters[contextName] = cluster
	config.AuthInfos[contextName] = authInfo

	// Update kube context
	newContext := clientcmdapi.NewContext()
	newContext.Cluster = contextName
	newContext.AuthInfo = contextName

	config.Contexts[contextName] = newContext
	if setActive {
		config.CurrentContext = contextName
	}

	// Save the config
	return clientcmd.ModifyConfig(clientcmd.NewDefaultClientConfigLoadingRules(), config, false)
}

func RandomPort() int {
	for i := 0; i < 10; i++ {
		port := 10000 + rand.Intn(3000)
		s, err := checkPort(port)
		if s && err == nil {
			return port
		}
	}

	// just try another port
	return 10000 + rand.Intn(3000)
}

func checkPort(port int) (status bool, err error) {
	// Concatenate a colon and the port
	host := "localhost:" + strconv.Itoa(port)

	// Try to create a server with the port
	server, err := net.Listen("tcp", host)

	// if it fails then the port is likely taken
	if err != nil {
		return false, err
	}

	// close the server
	_ = server.Close()

	// we successfully used and closed the port
	// so it's now available to be used again
	return true, nil
}
