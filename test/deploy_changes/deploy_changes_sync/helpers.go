package deploysyncchanges

import (
	"os/exec"

	"github.com/loft-sh/vcluster/pkg/util/translate"
	"github.com/loft-sh/vcluster/test/framework"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func createVClusterTestNamespace(testNamespace string, f *framework.Framework) {
	_, err := f.VClusterClient.CoreV1().Namespaces().Create(f.Context, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   testNamespace,
			Labels: map[string]string{initialNsLabelKey: initialNsLabelValue},
		},
	}, metav1.CreateOptions{})
	framework.ExpectNoError(err, "Failed to create test namespace")
}

func createServiceAccount(testNamespace string, saName string, f *framework.Framework) {
	_, err := f.VClusterClient.CoreV1().ServiceAccounts(testNamespace).Create(f.Context, &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name: saName,
		},
	}, metav1.CreateOptions{})
	framework.ExpectNoError(err, "Failed to create service account")

	err = f.WaitForServiceAccount(saName, testNamespace)
	framework.ExpectNoError(err)
}

func createPod(testNamespace string, podName string, f *framework.Framework) {

	_, err := f.VClusterClient.CoreV1().Pods(testNamespace).Create(f.Context, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: podName},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:            testingContainerName,
					Image:           testingContainerImage,
					ImagePullPolicy: corev1.PullIfNotPresent,
					SecurityContext: f.GetDefaultSecurityContext(),
				},
			},
		},
	}, metav1.CreateOptions{})
	framework.ExpectNoError(err)

	f.WaitForPodRunning(podName, testNamespace)
}

func verifyPodSyncToHostStatus(ns, podName string, f *framework.Framework, sync bool) {
	vpod, err := f.VClusterClient.CoreV1().Pods(ns).Get(f.Context, podName, metav1.GetOptions{})
	framework.ExpectNoError(err)

	pPodName := translate.Default.HostName(nil, podName, ns)
	pod, err := f.HostClient.CoreV1().Pods(pPodName.Namespace).Get(f.Context, pPodName.Name, metav1.GetOptions{})

	if sync {
		framework.ExpectNoError(err)
		framework.ExpectEqual(vpod.Status, pod.Status, "Pod status should be synced between vCluster and Host")
	} else {
		framework.ExpectError(err)
		framework.ExpectNotEqual(vpod.Status, pod.Status, "Pod status should not be synced between vCluster and Host")
	}

}

func verifyServiceAccountSyncToHostStatus(ns, saName string, f *framework.Framework, sync bool) {
	hostSaName := translate.Default.HostName(nil, saName, ns)
	_, err := f.HostClient.CoreV1().ServiceAccounts(hostSaName.Namespace).Get(f.Context, hostSaName.Name, metav1.GetOptions{})
	if sync {
		framework.ExpectNoError(err, "ServiceAccount is not synced to Host as expected")
	} else {
		framework.ExpectError(err, "ServiceAccount is synced to Host which is not expected")
	}
}

func updateVClusterSyncSettings(syncSettings []string) {
	for _, expr := range syncSettings {
		cmdExec := exec.Command("yq", "e", "-i", expr, filePath)
		err := cmdExec.Run()
		framework.ExpectNoError(err, "Failure to edit file")
	}
}
