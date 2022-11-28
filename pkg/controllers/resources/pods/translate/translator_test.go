package translate

import (
	"testing"

	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"gotest.tools/assert"
	"gotest.tools/assert/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
)

func TestPodAffinityTermsTranslation(t *testing.T) {
	longKey := "pretty-loooooooooooong-test-key"
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-little-pody",
			Namespace: "equestria",
		},
	}
	basicSelector := &metav1.LabelSelector{
		MatchLabels: map[string]string{"random-key": "value"},
	}
	basicSelectorTranslatedWithMarker := &metav1.LabelSelector{MatchLabels: map[string]string{}}
	for k, v := range basicSelector.MatchLabels {
		basicSelectorTranslatedWithMarker.MatchLabels[translate.ConvertLabelKey(k)] = v
	}
	basicSelectorTranslatedWithMarker.MatchLabels[translate.MarkerLabel] = translate.Suffix

	testCases := []translatePodAffinityTermTestCase{
		{
			name: "no selector",
			term: corev1.PodAffinityTerm{
				Namespaces: []string{"blabla"},
			},
			expectedTerm: corev1.PodAffinityTerm{},
		},
		{
			name: "empty namespaces array and null namespaceSelector",
			term: corev1.PodAffinityTerm{
				LabelSelector: basicSelector,
				Namespaces:    []string{},
			},
			expectedTerm: corev1.PodAffinityTerm{
				LabelSelector: appendToMatchLabels(basicSelectorTranslatedWithMarker, translate.NamespaceLabel, pod.GetNamespace()),
			},
		},
		{
			name: "null namespaceSelector and defined namespaces array",
			term: corev1.PodAffinityTerm{
				LabelSelector: basicSelector,
				Namespaces:    []string{pod.GetNamespace(), "dummy namespace"},
			},
			expectedTerm: corev1.PodAffinityTerm{
				LabelSelector: appendNamespacesToMatchExpressions(basicSelectorTranslatedWithMarker, pod.GetNamespace(), "dummy namespace"),
			},
		},
		{
			name: "empty namespaceSelector and defined namespaces array",
			term: corev1.PodAffinityTerm{
				LabelSelector:     basicSelector,
				Namespaces:        []string{pod.GetNamespace()},
				NamespaceSelector: &metav1.LabelSelector{},
			},
			expectedTerm: corev1.PodAffinityTerm{
				LabelSelector: basicSelectorTranslatedWithMarker,
			},
		},
		{ // TODO: remove once we implement the support for using .namespaces and .namespaceSelector together
			name: "namespaces array specified together with the namespaceSelector",
			term: corev1.PodAffinityTerm{
				LabelSelector:     basicSelector,
				Namespaces:        []string{pod.GetNamespace()},
				NamespaceSelector: basicSelector,
			},
			expectedTerm: corev1.PodAffinityTerm{
				LabelSelector: appendNamespacesToMatchExpressions(basicSelectorTranslatedWithMarker, pod.GetNamespace()),
			},
			expectedEvents: []string{"Warning SyncWarning Inter-pod affinity rule(s) that use both .namespaces and .namespaceSelector fields in the same term are not supported by vcluster yet. The .namespaceSelector fields of the unsupported affinity entries will be ignored."},
		},
		{
			name: "defined namespaceSelector and no namespaces",
			term: corev1.PodAffinityTerm{
				LabelSelector: basicSelector,
				NamespaceSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						longKey: "good-value",
					},
				},
			},
			expectedTerm: corev1.PodAffinityTerm{
				LabelSelector: appendToMatchLabels(&metav1.LabelSelector{
					MatchLabels: map[string]string{
						translate.ConvertLabelKeyWithPrefix(NamespaceLabelPrefix, longKey): "good-value",
					},
				}, translate.MarkerLabel, translate.Suffix),
			},
		},
		{
			name: "namespaceSelector with MatchExpressions",
			term: corev1.PodAffinityTerm{
				LabelSelector: basicSelector,
				NamespaceSelector: &metav1.LabelSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{
							Key:      longKey,
							Operator: metav1.LabelSelectorOpNotIn,
							Values:   []string{"bad-value"},
						},
					},
				},
			},
			expectedTerm: corev1.PodAffinityTerm{
				LabelSelector: appendToMatchLabels(&metav1.LabelSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{
							Key:      translate.ConvertLabelKeyWithPrefix(NamespaceLabelPrefix, longKey),
							Operator: metav1.LabelSelectorOpNotIn,
							Values:   []string{"bad-value"},
						},
					},
				}, translate.MarkerLabel, translate.Suffix),
			},
		},
	}

	for _, testCase := range testCases {
		fakeRecorder := record.NewFakeRecorder(10)
		tr := &translator{
			eventRecorder: fakeRecorder,
			log:           loghelper.New("pods-syncer-translator-test"),
		}

		result := tr.translatePodAffinityTerm(pod, testCase.term)
		assert.Assert(t, cmp.DeepEqual(result, testCase.expectedTerm), "Unexpected translation of the PodAffinityTerm in the '%s' test case", testCase.name)

		// read the events from the recorder mock
		close(fakeRecorder.Events)
		events := make([]string, 0)
		for v := range fakeRecorder.Events {
			events = append(events, v)
		}
		if len(events) == 0 {
			events = nil
		}

		assert.Assert(t, cmp.DeepEqual(events, testCase.expectedEvents), "Unexpected Event in the '%s' test case", testCase.name)
	}
}

type translatePodAffinityTermTestCase struct {
	name           string
	term           corev1.PodAffinityTerm
	expectedTerm   corev1.PodAffinityTerm
	expectedEvents []string
}

func TestVolumeTranslation(t *testing.T) {
	testCases := []translatePodVolumesTestCase{
		{
			name: "ephemeral volume",
			vPod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod-name",
					Namespace: "test-ns",
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "eph-vol",
							VolumeSource: corev1.VolumeSource{
								Ephemeral: &corev1.EphemeralVolumeSource{
									VolumeClaimTemplate: &corev1.PersistentVolumeClaimTemplate{},
								},
							},
						},
					},
				},
			},
			expectedVolumes: []corev1.Volume{
				{
					Name: "eph-vol",
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: translate.Default.PhysicalName("pod-name-eph-vol", "test-ns"),
						},
						Ephemeral: nil,
					},
				},
			},
		},
	}

	for _, testCase := range testCases {
		fakeRecorder := record.NewFakeRecorder(10)
		tr := &translator{
			eventRecorder: fakeRecorder,
			log:           loghelper.New("pods-syncer-translator-test"),
		}

		pPod := testCase.vPod.DeepCopy()
		err := tr.translateVolumes(pPod, &testCase.vPod)
		assert.NilError(t, err)
		assert.Assert(t, cmp.DeepEqual(pPod.Spec.Volumes, testCase.expectedVolumes), "Unexpected translation of the Volumes in the '%s' test case", testCase.name)
	}
}

type translatePodVolumesTestCase struct {
	name            string
	vPod            corev1.Pod
	expectedVolumes []corev1.Volume
}

func appendToMatchLabels(source *metav1.LabelSelector, k, v string) *metav1.LabelSelector {
	ls := source.DeepCopy()
	if ls.MatchLabels == nil {
		ls.MatchLabels = map[string]string{}
	}
	ls.MatchLabels[k] = v
	return ls
}

func appendNamespacesToMatchExpressions(source *metav1.LabelSelector, namespaces ...string) *metav1.LabelSelector {
	ls := source.DeepCopy()
	ls.MatchExpressions = append(ls.MatchExpressions, metav1.LabelSelectorRequirement{
		Key:      translate.NamespaceLabel,
		Operator: metav1.LabelSelectorOpIn,
		Values:   append([]string{}, namespaces...),
	})
	return ls
}
