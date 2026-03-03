package pods

import (
	"context"
	"fmt"
	"time"

	"github.com/loft-sh/vcluster/pkg/util/random"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"github.com/loft-sh/vcluster/test/framework"
	"github.com/onsi/ginkgo/v2"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	testingContainerName  = "nginx"
	testingContainerImage = "nginxinc/nginx-unprivileged:stable-alpine3.20-slim"

	// enforcedTolerationKey is the toleration key configured in this suite's values.yaml:
	//   sync.toHost.pods.enforceTolerations: ["vcluster-e2e-enforced:NoSchedule"]
	enforcedTolerationKey    = "vcluster-e2e-enforced"
	enforcedTolerationEffect = corev1.TaintEffectNoSchedule
)

var _ = ginkgo.Describe("Pod enforce tolerations", func() {
	var (
		f         *framework.Framework
		iteration int
		ns        string
	)

	ginkgo.JustBeforeEach(func() {
		f = framework.DefaultFramework
		iteration++
		ns = fmt.Sprintf("e2e-enforce-tolerations-%d-%s", iteration, random.String(5))

		_, err := f.VClusterClient.CoreV1().Namespaces().Create(f.Context, &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: ns},
		}, metav1.CreateOptions{})
		framework.ExpectNoError(err)
	})

	ginkgo.AfterEach(func() {
		err := f.DeleteTestNamespace(ns, false)
		framework.ExpectNoError(err)
	})

	// Test that the enforced toleration from vcluster config is applied to the physical pod
	// at creation time, even when the virtual pod spec does not include it.
	ginkgo.It("Physical pod should have the configured enforced toleration at creation", func() {
		podName := "enforced-tol-creation"
		_, err := f.VClusterClient.CoreV1().Pods(ns).Create(f.Context, &corev1.Pod{
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

		err = f.WaitForPodRunning(podName, ns)
		framework.ExpectNoError(err, "A pod created in the vcluster is expected to be in the Running phase eventually.")

		pPodName := translate.Default.HostName(nil, podName, ns)
		pPod, err := f.HostClient.CoreV1().Pods(pPodName.Namespace).Get(f.Context, pPodName.Name, metav1.GetOptions{})
		framework.ExpectNoError(err)

		framework.ExpectEqual(
			hasEnforcedToleration(pPod.Spec.Tolerations),
			true,
			"Physical pod should contain the enforced toleration (key=%s, effect=%s)",
			enforcedTolerationKey, enforcedTolerationEffect,
		)
	})

	// Test that the enforced toleration from vcluster config is preserved on the physical pod
	// after the virtual pod's tolerations are updated. This exercises the Diff() path added to
	// fix the PodObservedGenerationTracking conformance test failure.
	ginkgo.It("Physical pod should retain the enforced toleration after virtual pod toleration update", func() {
		podName := "enforced-tol-update"
		_, err := f.VClusterClient.CoreV1().Pods(ns).Create(f.Context, &corev1.Pod{
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

		err = f.WaitForPodRunning(podName, ns)
		framework.ExpectNoError(err, "A pod created in the vcluster is expected to be in the Running phase eventually.")

		// Update the virtual pod to add a new toleration. This triggers the Diff() path in the
		// syncer and must preserve the vcluster-configured enforced toleration on the physical pod.
		newTolerationKey := "e2e-added"
		err = wait.PollUntilContextTimeout(f.Context, time.Second, framework.PollTimeout, true, func(ctx context.Context) (bool, error) {
			vpod, err := f.VClusterClient.CoreV1().Pods(ns).Get(ctx, podName, metav1.GetOptions{})
			if err != nil {
				return false, err
			}
			vpod.Spec.Tolerations = append(vpod.Spec.Tolerations, corev1.Toleration{
				Key:      newTolerationKey,
				Operator: corev1.TolerationOpExists,
				Effect:   corev1.TaintEffectNoSchedule,
			})
			_, err = f.VClusterClient.CoreV1().Pods(ns).Update(ctx, vpod, metav1.UpdateOptions{})
			if kerrors.IsConflict(err) {
				return false, nil
			}
			return err == nil, err
		})
		framework.ExpectNoError(err, "Failed to update virtual pod tolerations")

		// Wait for the new toleration to appear on the physical pod (confirms the syncer ran Diff).
		pPodName := translate.Default.HostName(nil, podName, ns)
		err = wait.PollUntilContextTimeout(f.Context, time.Second, framework.PollTimeout, true, func(ctx context.Context) (bool, error) {
			pPod, err := f.HostClient.CoreV1().Pods(pPodName.Namespace).Get(ctx, pPodName.Name, metav1.GetOptions{})
			if err != nil {
				return false, err
			}
			return hasTolerationWithKey(pPod.Spec.Tolerations, newTolerationKey), nil
		})
		framework.ExpectNoError(err, "New toleration should appear on physical pod after virtual pod update")

		// Now verify the enforced toleration is still present alongside the newly added one.
		pPod, err := f.HostClient.CoreV1().Pods(pPodName.Namespace).Get(f.Context, pPodName.Name, metav1.GetOptions{})
		framework.ExpectNoError(err)

		framework.ExpectEqual(
			hasEnforcedToleration(pPod.Spec.Tolerations),
			true,
			"Enforced toleration must still be present on the physical pod after the virtual pod toleration update",
		)
	})
})

// hasEnforcedToleration returns true if the list contains the toleration configured via
// enforceTolerations in this test suite's values.yaml.
func hasEnforcedToleration(tolerations []corev1.Toleration) bool {
	for _, tol := range tolerations {
		if tol.Key == enforcedTolerationKey && tol.Effect == enforcedTolerationEffect {
			return true
		}
	}
	return false
}

// hasTolerationWithKey returns true if the list contains a toleration with the given key.
func hasTolerationWithKey(tolerations []corev1.Toleration, key string) bool {
	for _, tol := range tolerations {
		if tol.Key == key {
			return true
		}
	}
	return false
}
