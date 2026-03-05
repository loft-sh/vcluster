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

	// Test that the enforced toleration is not duplicated on the physical pod when the virtual
	// pod already includes it in its own spec.
	ginkgo.It("Physical pod should not have duplicate enforced tolerations when virtual pod already has it", func() {
		podName := "enforced-tol-dedup"
		_, err := f.VClusterClient.CoreV1().Pods(ns).Create(f.Context, &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: podName},
			Spec: corev1.PodSpec{
				// Explicitly include the enforced toleration in the virtual pod spec.
				Tolerations: []corev1.Toleration{
					{
						Key:      enforcedTolerationKey,
						Operator: corev1.TolerationOpExists,
						Effect:   enforcedTolerationEffect,
					},
				},
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

		count := countTolerationsByKey(pPod.Spec.Tolerations, enforcedTolerationKey)
		framework.ExpectEqual(
			count,
			1,
			"Enforced toleration key %q must appear exactly once on the physical pod, got %d",
			enforcedTolerationKey, count,
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

	// Test that a virtual pod carrying the enforced taint target with TolerationSeconds set
	// does not produce a duplicate on the physical pod.
	ginkgo.It("Physical pod has no duplicate when virtual pod carries enforced taint with TolerationSeconds", func() {
		podName := "enforced-tol-seconds-dedup"
		tolerationSeconds := int64(300)

		_, err := f.VClusterClient.CoreV1().Pods(ns).Create(f.Context, &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: podName},
			Spec: corev1.PodSpec{
				// The virtual pod explicitly sets the enforced taint with a TolerationSeconds.
				// The enforced config entry has the same Key+Effect but no TolerationSeconds.
				Tolerations: []corev1.Toleration{
					{
						Key:               enforcedTolerationKey,
						Operator:          corev1.TolerationOpExists,
						Effect:            enforcedTolerationEffect,
						TolerationSeconds: &tolerationSeconds,
					},
				},
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

		count := countTolerationsByKey(pPod.Spec.Tolerations, enforcedTolerationKey)
		framework.ExpectEqual(
			count, 1,
			"Enforced toleration key %q must appear exactly once; got %d (duplicate from enforced config entry expected to be suppressed)",
			enforcedTolerationKey, count,
		)
	})

	// Test that updating TolerationSeconds on a NoExecute toleration in the virtual cluster
	// propagates the new value to the physical pod without accumulating a stale duplicate.
	ginkgo.It("TolerationSeconds update is propagated to host pod without duplicating the stale variant", func() {
		podName := "tol-seconds-update"
		const tolKey = "e2e-noexec-seconds"
		initialSeconds := int64(60)
		updatedSeconds := int64(120)

		_, err := f.VClusterClient.CoreV1().Pods(ns).Create(f.Context, &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: podName},
			Spec: corev1.PodSpec{
				Tolerations: []corev1.Toleration{
					{
						Key:               tolKey,
						Operator:          corev1.TolerationOpExists,
						Effect:            corev1.TaintEffectNoExecute,
						TolerationSeconds: &initialSeconds,
					},
				},
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

		// Sanity check: physical pod should have exactly one instance of the toleration.
		pPodName := translate.Default.HostName(nil, podName, ns)
		pPod, err := f.HostClient.CoreV1().Pods(pPodName.Namespace).Get(f.Context, pPodName.Name, metav1.GetOptions{})
		framework.ExpectNoError(err)
		framework.ExpectEqual(
			countTolerationsByKey(pPod.Spec.Tolerations, tolKey), 1,
			"Expected exactly one toleration with key %q on the physical pod at creation", tolKey,
		)

		// Update the virtual pod, increasing TolerationSeconds from 60 to 120.
		// Kubernetes allows this update (additive-only rules permit increasing TolerationSeconds
		// on NoExecute tolerations because a longer value is more lenient for a running pod).
		err = wait.PollUntilContextTimeout(f.Context, time.Second, framework.PollTimeout, true, func(ctx context.Context) (bool, error) {
			vpod, err := f.VClusterClient.CoreV1().Pods(ns).Get(ctx, podName, metav1.GetOptions{})
			if err != nil {
				return false, err
			}
			for i, tol := range vpod.Spec.Tolerations {
				if tol.Key == tolKey {
					vpod.Spec.Tolerations[i].TolerationSeconds = &updatedSeconds
					break
				}
			}
			_, err = f.VClusterClient.CoreV1().Pods(ns).Update(ctx, vpod, metav1.UpdateOptions{})
			if kerrors.IsConflict(err) {
				return false, nil
			}
			return err == nil, err
		})
		framework.ExpectNoError(err, "Failed to update TolerationSeconds on the virtual pod")

		// Wait until the physical pod reflects the updated value.
		err = wait.PollUntilContextTimeout(f.Context, time.Second, framework.PollTimeout, true, func(ctx context.Context) (bool, error) {
			pPod, err := f.HostClient.CoreV1().Pods(pPodName.Namespace).Get(ctx, pPodName.Name, metav1.GetOptions{})
			if err != nil {
				return false, err
			}
			return getTolerationSeconds(pPod.Spec.Tolerations, tolKey) == updatedSeconds, nil
		})
		framework.ExpectNoError(err, "Physical pod should reflect the updated TolerationSeconds=%d for key %q", updatedSeconds, tolKey)

		// The stale variant (TolerationSeconds=60) must not be present alongside the updated one.
		pPodAfter, err := f.HostClient.CoreV1().Pods(pPodName.Namespace).Get(f.Context, pPodName.Name, metav1.GetOptions{})
		framework.ExpectNoError(err)
		framework.ExpectEqual(
			countTolerationsByKey(pPodAfter.Spec.Tolerations, tolKey), 1,
			"Toleration with key %q must appear exactly once; stale TolerationSeconds=%d variant must not be duplicated",
			tolKey, initialSeconds,
		)
	})

	// Test that a toleration added directly on the physical (host) pod
	// is not dropped when the virtual pod's tolerations are subsequently changed.
	ginkgo.It("Host-side toleration added directly on the physical pod is not dropped when virtual tolerations change", func() {
		podName := "enforced-tol-host-side"
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

		// Add a toleration directly on the physical pod
		hostSideTolerationKey := "e2e-host-injected"
		pPodName := translate.Default.HostName(nil, podName, ns)
		err = wait.PollUntilContextTimeout(f.Context, time.Second, framework.PollTimeout, true, func(ctx context.Context) (bool, error) {
			pPod, err := f.HostClient.CoreV1().Pods(pPodName.Namespace).Get(ctx, pPodName.Name, metav1.GetOptions{})
			if err != nil {
				return false, err
			}
			pPod.Spec.Tolerations = append(pPod.Spec.Tolerations, corev1.Toleration{
				Key:      hostSideTolerationKey,
				Operator: corev1.TolerationOpExists,
				Effect:   corev1.TaintEffectNoSchedule,
			})
			_, err = f.HostClient.CoreV1().Pods(pPodName.Namespace).Update(ctx, pPod, metav1.UpdateOptions{})
			if kerrors.IsConflict(err) {
				return false, nil
			}
			return err == nil, err
		})
		framework.ExpectNoError(err, "Failed to add host-side toleration directly on the physical pod")

		// Update the virtual pod to add a new toleration, triggering the Diff() path in the syncer.
		// The syncer must carry the host-side toleration forward rather than dropping it.
		virtualTolerationKey := "e2e-virtual-added"
		err = wait.PollUntilContextTimeout(f.Context, time.Second, framework.PollTimeout, true, func(ctx context.Context) (bool, error) {
			vpod, err := f.VClusterClient.CoreV1().Pods(ns).Get(ctx, podName, metav1.GetOptions{})
			if err != nil {
				return false, err
			}
			vpod.Spec.Tolerations = append(vpod.Spec.Tolerations, corev1.Toleration{
				Key:      virtualTolerationKey,
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

		err = wait.PollUntilContextTimeout(f.Context, time.Second, framework.PollTimeout, true, func(ctx context.Context) (bool, error) {
			pPod, err := f.HostClient.CoreV1().Pods(pPodName.Namespace).Get(ctx, pPodName.Name, metav1.GetOptions{})
			if err != nil {
				return false, err
			}
			return hasTolerationWithKey(pPod.Spec.Tolerations, virtualTolerationKey), nil
		})
		framework.ExpectNoError(err, "Virtual toleration should appear on physical pod after virtual pod update")

		// The host-side toleration must still be present — the syncer must not have dropped it.
		pPodAfter, err := f.HostClient.CoreV1().Pods(pPodName.Namespace).Get(f.Context, pPodName.Name, metav1.GetOptions{})
		framework.ExpectNoError(err)

		framework.ExpectEqual(
			hasTolerationWithKey(pPodAfter.Spec.Tolerations, hostSideTolerationKey),
			true,
			"Host-side toleration %q was dropped from the physical pod after the virtual toleration update",
			hostSideTolerationKey,
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

// countTolerationsByKey returns the number of tolerations with the given key.
func countTolerationsByKey(tolerations []corev1.Toleration, key string) int {
	count := 0
	for _, tol := range tolerations {
		if tol.Key == key {
			count++
		}
	}
	return count
}

// getTolerationSeconds returns the TolerationSeconds value for the first toleration
// matching key, or 0 if not found or TolerationSeconds is nil.
func getTolerationSeconds(tolerations []corev1.Toleration, key string) int64 {
	for _, tol := range tolerations {
		if tol.Key == key && tol.TolerationSeconds != nil {
			return *tol.TolerationSeconds
		}
	}
	return 0
}
