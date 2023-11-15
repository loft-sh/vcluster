package start

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/loft-sh/admin-apis/pkg/licenseapi"
	"github.com/loft-sh/api/v3/pkg/product"
	"github.com/loft-sh/loftctl/v3/pkg/client"
	"github.com/loft-sh/loftctl/v3/pkg/clihelper"
	"github.com/loft-sh/loftctl/v3/pkg/config"
	"github.com/loft-sh/loftctl/v3/pkg/printhelper"
	"github.com/loft-sh/log/survey"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

func (l *LoftStarter) success(ctx context.Context) error {
	if l.NoWait {
		return nil
	}

	// wait until Loft is ready
	loftPod, err := l.waitForLoft(ctx)
	if err != nil {
		return err
	}

	if l.NoPortForwarding {
		return nil
	}

	// check if Loft was installed locally
	isLocal := clihelper.IsLoftInstalledLocally(ctx, l.KubeClient, l.Namespace)
	if isLocal {
		// check if loft domain secret is there
		if !l.NoTunnel {
			loftRouterDomain, err := l.pingLoftRouter(ctx, loftPod)
			if err != nil {
				l.Log.Errorf("Error retrieving loft router domain: %v", err)
				l.Log.Info("Fallback to use port-forwarding")
			} else if loftRouterDomain != "" {
				return l.successLoftRouter(loftRouterDomain)
			}
		}

		// start port-forwarding
		err = l.startPortForwarding(ctx, loftPod)
		if err != nil {
			return err
		}

		return l.successLocal()
	}

	// get login link
	l.Log.Info("Checking Loft status...")
	host, err := clihelper.GetLoftIngressHost(ctx, l.KubeClient, l.Namespace)
	if err != nil {
		return err
	}

	// check if loft is reachable
	reachable, err := clihelper.IsLoftReachable(ctx, host)
	if !reachable || err != nil {
		const (
			YesOption = "Yes"
			NoOption  = "No, please re-run the DNS check"
		)

		answer, err := l.Log.Question(&survey.QuestionOptions{
			Question:     "Unable to reach Loft at https://" + host + ". Do you want to start port-forwarding instead?",
			DefaultValue: YesOption,
			Options: []string{
				YesOption,
				NoOption,
			},
		})
		if err != nil {
			return err
		}

		if answer == YesOption {
			err = l.startPortForwarding(ctx, loftPod)
			if err != nil {
				return err
			}

			return l.successLocal()
		}
	}

	return l.successRemote(ctx, host)
}

func (l *LoftStarter) pingLoftRouter(ctx context.Context, loftPod *corev1.Pod) (string, error) {
	loftRouterSecret, err := l.KubeClient.CoreV1().Secrets(loftPod.Namespace).Get(ctx, clihelper.LoftRouterDomainSecret, metav1.GetOptions{})
	if err != nil {
		if kerrors.IsNotFound(err) {
			return "", nil
		}

		return "", fmt.Errorf("find loft router domain secret: %w", err)
	} else if loftRouterSecret.Data == nil || len(loftRouterSecret.Data["domain"]) == 0 {
		return "", nil
	}

	// get the domain from secret
	loftRouterDomain := string(loftRouterSecret.Data["domain"])

	// wait until loft is reachable at the given url
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
	l.Log.Infof(product.Replace("Waiting until loft is reachable at https://%s"), loftRouterDomain)
	err = wait.PollUntilContextTimeout(ctx, time.Second*3, time.Minute*5, true, func(ctx context.Context) (bool, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://"+loftRouterDomain+"/version", nil)
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
		return "", err
	}

	return loftRouterDomain, nil
}

func (l *LoftStarter) successLoftRouter(url string) error {
	if !l.NoLogin {
		err := l.login(url)
		if err != nil {
			return err
		}
	}

	printhelper.PrintSuccessMessageLoftRouterInstall(url, l.Password, l.Log)
	l.printVClusterProGettingStarted(url)
	return nil
}

func (l *LoftStarter) successLocal() error {
	url := "https://localhost:" + l.LocalPort

	if !l.NoLogin {
		err := l.login(url)
		if err != nil {
			return err
		}
	}

	printhelper.PrintSuccessMessageLocalInstall(l.Password, url, l.Log)
	l.printVClusterProGettingStarted(url)

	blockChan := make(chan bool)
	<-blockChan
	return nil
}

func (l *LoftStarter) isLoggedIn(url string) bool {
	url = strings.TrimPrefix(url, "https://")

	c, err := client.NewClientFromPath(l.Config)
	return err == nil && strings.TrimPrefix(strings.TrimSuffix(c.Config().Host, "/"), "https://") == strings.TrimSuffix(url, "/")
}

func (l *LoftStarter) successRemote(ctx context.Context, host string) error {
	ready, err := clihelper.IsLoftReachable(ctx, host)
	if err != nil {
		return err
	} else if ready {
		printhelper.PrintSuccessMessageRemoteInstall(host, l.Password, l.Log)
		return nil
	}

	// Print DNS Configuration
	printhelper.PrintDNSConfiguration(host, l.Log)

	l.Log.Info("Waiting for you to configure DNS, so loft can be reached on https://" + host)
	err = wait.PollUntilContextTimeout(ctx, 5*time.Second, config.Timeout(), true, func(ctx context.Context) (done bool, err error) {
		return clihelper.IsLoftReachable(ctx, host)
	})
	if err != nil {
		return err
	}

	l.Log.Done(product.Replace("Loft is reachable at https://") + host)
	printhelper.PrintSuccessMessageRemoteInstall(host, l.Password, l.Log)
	return nil
}

func (l *LoftStarter) printVClusterProGettingStarted(url string) {
	if product.Name() != licenseapi.VClusterPro {
		return
	}

	if l.isLoggedIn(url) {
		l.Log.Donef("You are successfully logged into vCluster.Pro!")
		l.Log.WriteString(logrus.InfoLevel, "- Use `vcluster create` to create a new pro vCluster\n")
		l.Log.WriteString(logrus.InfoLevel, "- Use `vcluster create --disable-pro` to create a new oss vCluster\n")
		l.Log.WriteString(logrus.InfoLevel, "- Use `vcluster import` to import and upgrade an existing oss vCluster\n")
	} else {
		l.Log.Warnf("You are not logged into vCluster.Pro yet, please run the below command to log into the vCluster.Pro instance")
		l.Log.WriteString(logrus.InfoLevel, "- Use `vcluster login "+url+"` to log into the vCluster.Pro instance\n")
	}
}

func (l *LoftStarter) waitForLoft(ctx context.Context) (*corev1.Pod, error) {
	// wait for loft pod to start
	l.Log.Info(product.Replace("Waiting for Loft pod to be running..."))
	loftPod, err := clihelper.WaitForReadyLoftPod(ctx, l.KubeClient, l.Namespace, l.Log)
	l.Log.Donef(product.Replace("Loft pod successfully started"))
	if err != nil {
		return nil, err
	}

	// ensure user admin secret is there
	isNewPassword, err := clihelper.EnsureAdminPassword(ctx, l.KubeClient, l.RestConfig, l.Password, l.Log)
	if err != nil {
		return nil, err
	}

	// If password is different than expected
	if isNewPassword {
		l.Password = ""
	}

	return loftPod, nil
}
