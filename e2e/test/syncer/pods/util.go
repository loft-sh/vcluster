package pods

import (
	"time"

	"github.com/loft-sh/vcluster/e2e/framework"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

func WaitForPodRunning(f *framework.Framework, podName string, ns string) error {
	return wait.PollImmediate(time.Second, framework.PollTimeout, func() (bool, error) {
		pod, err := f.HostClient.CoreV1().Pods(f.VclusterNamespace).Get(f.Context, podName+"-x-"+ns+"-x-"+f.Suffix, metav1.GetOptions{})
		if err != nil {
			if kerrors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		if pod.Status.Phase != corev1.PodRunning {
			return false, nil
		}
		vpod, err := f.VclusterClient.CoreV1().Pods(ns).Get(f.Context, podName, metav1.GetOptions{})
		if err != nil {
			if kerrors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		if vpod.Status.Phase != corev1.PodRunning {
			return false, nil
		}
		return true, nil
	})
}

func WaitForServiceAccount(f *framework.Framework, saName string, ns string) error {
	return wait.PollImmediate(time.Second, framework.PollTimeout, func() (bool, error) {
		_, err := f.VclusterClient.CoreV1().ServiceAccounts(ns).Get(f.Context, saName, metav1.GetOptions{})
		if err != nil {
			if kerrors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		return true, nil
	})
}
