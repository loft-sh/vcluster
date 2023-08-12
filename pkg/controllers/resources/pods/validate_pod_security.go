package pods

import (
	"context"
	"fmt"

	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/pod-security-admission/admission"
	admissionapi "k8s.io/pod-security-admission/admission/api"
	"k8s.io/pod-security-admission/api"
	"k8s.io/pod-security-admission/metrics"
	"k8s.io/pod-security-admission/policy"
)

func (s *podSyncer) isPodSecurityStandardsValid(ctx context.Context, pod *corev1.Pod, log loghelper.Logger) (bool, error) {
	result, err := s.validatePodSecurityStandards(ctx, pod)
	if err != nil {
		log.Errorf(err.Error())
	} else if result != nil {
		if !result.Allowed {
			log.Errorf("%s pod creation not allowed: %s", pod.Name, result.Result.Message)
			s.EventRecorder().Eventf(pod, "Warning", "SyncError", `Pod %s is forbidden: %s`, pod.Name, result.Result.Message)
		}
		return result.Allowed, nil
	}
	return false, err
}

func (s *podSyncer) validatePodSecurityStandards(ctx context.Context, pod *corev1.Pod) (*admissionv1.AdmissionResponse, error) {
	version := api.LatestVersion()
	podSecurityDefaults := admissionapi.PodSecurityDefaults{
		Enforce:        s.podSecurityStandard,
		EnforceVersion: version.String(),
		Audit:          s.podSecurityStandard,
		AuditVersion:   version.String(),
		Warn:           s.podSecurityStandard,
		WarnVersion:    version.String(),
	}

	evaluator, err := policy.NewEvaluator(policy.DefaultChecks())
	if err != nil {
		return nil, fmt.Errorf("could not create PodSecurityRegistry: %w", err)
	}

	adm := &admission.Admission{
		Evaluator:     evaluator,
		Metrics:       &FakeMetricsRecorder{},
		Configuration: &admissionapi.PodSecurityConfiguration{},
	}

	nsPolicy, nsError := admissionapi.ToPolicy(podSecurityDefaults)
	return adm.EvaluatePod(ctx, nsPolicy, nsError, &pod.ObjectMeta, &pod.Spec, &PodAttributes{pod}, true), nil
}

type PodAttributes struct {
	*corev1.Pod
}

var _ api.Attributes = &PodAttributes{}

func (a *PodAttributes) GetKind() schema.GroupVersionKind {
	return corev1.SchemeGroupVersion.WithKind("pod")
}

func (a *PodAttributes) GetObject() (runtime.Object, error) {
	return a, nil
}

func (a *PodAttributes) GetOldObject() (runtime.Object, error) {
	return a, nil
}

func (a *PodAttributes) GetOperation() admissionv1.Operation {
	return admissionv1.Create
}

func (a *PodAttributes) GetResource() schema.GroupVersionResource {
	return corev1.SchemeGroupVersion.WithResource("pods")
}

func (a *PodAttributes) GetSubresource() string {
	return ""
}

func (a *PodAttributes) GetUserName() string {
	return ""
}

type FakeMetricsRecorder struct{}

var _ metrics.Recorder = &FakeMetricsRecorder{}

func (n *FakeMetricsRecorder) RecordEvaluation(_ metrics.Decision, _ api.LevelVersion, _ metrics.Mode, _ api.Attributes) {
}
func (n *FakeMetricsRecorder) RecordExemption(api.Attributes)       {}
func (n *FakeMetricsRecorder) RecordError(_ bool, _ api.Attributes) {}
