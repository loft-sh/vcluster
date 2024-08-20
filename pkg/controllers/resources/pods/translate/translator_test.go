package translate

import (
	"testing"

	"github.com/loft-sh/vcluster/pkg/scheme"
	generictesting "github.com/loft-sh/vcluster/pkg/syncer/testing"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
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
		basicSelectorTranslatedWithMarker.MatchLabels[translate.HostLabel(k)] = v
	}
	basicSelectorTranslatedWithMarker.MatchLabels[translate.MarkerLabel] = translate.VClusterName

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
				LabelSelector: translate.MergeLabelSelectors(
					basicSelectorTranslatedWithMarker,
					&metav1.LabelSelector{MatchLabels: map[string]string{translate.NamespaceLabel: pod.GetNamespace()}},
				),
			},
		},
		{
			name: "null namespaceSelector and defined namespaces array",
			term: corev1.PodAffinityTerm{
				LabelSelector: basicSelector,
				Namespaces:    []string{pod.GetNamespace(), "dummy namespace"},
			},
			expectedTerm: corev1.PodAffinityTerm{
				LabelSelector: appendNamespacesToMatchExpressions(
					basicSelectorTranslatedWithMarker,
					pod.GetNamespace(), "dummy namespace"),
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
				LabelSelector: appendNamespacesToMatchExpressions(
					basicSelectorTranslatedWithMarker,
					pod.GetNamespace(),
				),
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
				LabelSelector: translate.MergeLabelSelectors(
					basicSelectorTranslatedWithMarker,
					&metav1.LabelSelector{MatchLabels: map[string]string{
						translate.HostLabelNamespace(longKey): "good-value",
					}},
				),
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
				LabelSelector: translate.MergeLabelSelectors(
					basicSelectorTranslatedWithMarker,
					&metav1.LabelSelector{
						MatchExpressions: []metav1.LabelSelectorRequirement{
							{
								Key:      translate.HostLabelNamespace(longKey),
								Operator: metav1.LabelSelectorOpNotIn,
								Values:   []string{"bad-value"},
							},
						},
					},
				),
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
							ClaimName: translate.Default.HostName(nil, "pod-name-eph-vol", "test-ns").Name,
						},
						Ephemeral: nil,
					},
				},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			fakeRecorder := record.NewFakeRecorder(10)
			pClient := testingutil.NewFakeClient(scheme.Scheme)
			vClient := testingutil.NewFakeClient(scheme.Scheme)
			registerCtx := generictesting.NewFakeRegisterContext(testingutil.NewFakeConfig(), pClient, vClient)
			tr := &translator{
				eventRecorder: fakeRecorder,
				log:           loghelper.New("pods-syncer-translator-test"),
				pClient:       pClient,
			}

			pPod := testCase.vPod.DeepCopy()
			err := tr.translateVolumes(registerCtx.ToSyncContext("pods-syncer-translator-test"), pPod, &testCase.vPod)
			assert.NilError(t, err)
			assert.Assert(t, cmp.DeepEqual(pPod.Spec.Volumes, testCase.expectedVolumes), "Unexpected translation of the Volumes in the '%s' test case", testCase.name)
		})
	}
}

type translatePodVolumesTestCase struct {
	name            string
	vPod            corev1.Pod
	expectedVolumes []corev1.Volume
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
