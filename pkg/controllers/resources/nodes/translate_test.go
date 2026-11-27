package nodes

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/loft-sh/vcluster/pkg/util/translate"
	"gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type translateBackwardsTest struct {
	name string

	vNode *corev1.Node
	pNode *corev1.Node

	expectedAnnotations map[string]string
	expectedLabels      map[string]string
	expectedTaints      []corev1.Taint
}

func TestTranslateBackwards(t *testing.T) {
	testCases := []translateBackwardsTest{
		{
			name: "simple",

			vNode: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
					Labels: map[string]string{
						"test": "test",
					},
				},
				Spec: corev1.NodeSpec{
					Taints: []corev1.Taint{
						{
							Key:    "test",
							Value:  "true",
							Effect: corev1.TaintEffectNoSchedule,
						},
					},
				},
			},
			pNode: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"test": "test",
					},
					Labels: map[string]string{
						"test": "test",
					},
				},
				Spec: corev1.NodeSpec{
					Taints: nil,
				},
			},

			expectedAnnotations: map[string]string{
				"test":                                 "test",
				translate.ManagedAnnotationsAnnotation: "test",
				translate.ManagedLabelsAnnotation:      "test",
			},

			expectedLabels: map[string]string{
				"test": "test",
			},

			expectedTaints: []corev1.Taint{
				{
					Key:    "test",
					Value:  "true",
					Effect: corev1.TaintEffectNoSchedule,
				},
			},
		},
		{
			name: "not-ready-1",

			vNode: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{},
				Spec:       corev1.NodeSpec{},
			},
			pNode: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{},
				Spec: corev1.NodeSpec{
					Taints: []corev1.Taint{
						{
							Key:    "node.kubernetes.io/not-ready",
							Value:  "true",
							Effect: corev1.TaintEffectNoSchedule,
						},
					},
				},
			},

			expectedLabels: map[string]string{},
			expectedAnnotations: map[string]string{
				TaintsAnnotation: mustMarshal([]string{
					mustMarshal(
						&corev1.Taint{
							Key:    "node.kubernetes.io/not-ready",
							Value:  "true",
							Effect: corev1.TaintEffectNoSchedule,
						},
					),
				}),
			},
			expectedTaints: []corev1.Taint{
				{
					Key:    "node.kubernetes.io/not-ready",
					Value:  "true",
					Effect: corev1.TaintEffectNoSchedule,
				},
			},
		},
		{
			name: "not-ready-2",

			vNode: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						TaintsAnnotation: mustMarshal([]string{
							mustMarshal(
								&corev1.Taint{
									Key:    "node.kubernetes.io/not-ready",
									Value:  "true",
									Effect: corev1.TaintEffectNoSchedule,
								},
							),
						}),
					},
				},
				Spec: corev1.NodeSpec{
					Taints: []corev1.Taint{
						{
							Key:    "node.kubernetes.io/not-ready",
							Value:  "true",
							Effect: corev1.TaintEffectNoSchedule,
						},
					},
				},
			},
			pNode: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{},
				Spec:       corev1.NodeSpec{},
			},

			expectedAnnotations: map[string]string{},
			expectedLabels:      map[string]string{},
			expectedTaints:      []corev1.Taint{},
		},
		{
			name: "not-ready-3",

			vNode: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{},
				Spec: corev1.NodeSpec{
					Taints: []corev1.Taint{
						{
							Key:    "node.kubernetes.io/not-ready",
							Value:  "true",
							Effect: corev1.TaintEffectNoSchedule,
						},
					},
				},
			},
			pNode: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{},
				Spec:       corev1.NodeSpec{},
			},
			expectedAnnotations: map[string]string{},
			expectedLabels:      map[string]string{},

			expectedTaints: []corev1.Taint{},
		},
		{
			name: "custom-taint-1",

			vNode: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
				Spec: corev1.NodeSpec{},
			},
			pNode: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{},
				Spec: corev1.NodeSpec{
					Taints: []corev1.Taint{
						{
							Key:    "custom-taint",
							Value:  "true",
							Effect: corev1.TaintEffectNoExecute,
						},
					},
				},
			},

			expectedAnnotations: map[string]string{
				TaintsAnnotation: mustMarshal([]string{
					mustMarshal(
						&corev1.Taint{
							Key:    "custom-taint",
							Value:  "true",
							Effect: corev1.TaintEffectNoExecute,
						},
					),
				}),
			},
			expectedTaints: []corev1.Taint{
				{
					Key:    "custom-taint",
					Value:  "true",
					Effect: corev1.TaintEffectNoExecute,
				},
			},
			expectedLabels: map[string]string{},
		},
		{
			name: "custom-taint-2",

			vNode: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						TaintsAnnotation: mustMarshal([]string{
							mustMarshal(
								&corev1.Taint{
									Key:    "custom-taint",
									Value:  "true",
									Effect: corev1.TaintEffectNoExecute,
								},
							),
						}),
					},
				},
				Spec: corev1.NodeSpec{
					Taints: []corev1.Taint{
						{
							Key:    "custom-taint",
							Value:  "true",
							Effect: corev1.TaintEffectNoExecute,
						},
					},
				},
			},
			pNode: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{},
				Spec: corev1.NodeSpec{
					Taints: []corev1.Taint{},
				},
			},

			expectedAnnotations: map[string]string{},
			expectedLabels:      map[string]string{},
			expectedTaints:      []corev1.Taint{},
		},
		{
			name: "custom-taint-3",

			vNode: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{},
				Spec: corev1.NodeSpec{
					Taints: []corev1.Taint{
						{
							Key:    "custom-taint",
							Value:  "true",
							Effect: corev1.TaintEffectNoExecute,
						},
					},
				},
			},
			pNode: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{},
				Spec: corev1.NodeSpec{
					Taints: []corev1.Taint{},
				},
			},

			expectedAnnotations: map[string]string{},
			expectedLabels:      map[string]string{},
			expectedTaints: []corev1.Taint{
				{
					Key:    "custom-taint",
					Value:  "true",
					Effect: corev1.TaintEffectNoExecute,
				},
			},
		},
		{
			name: "unreachable-different-timeAdded",

			vNode: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{},
				Spec: corev1.NodeSpec{
					Taints: []corev1.Taint{
						{
							Key:       "node.kubernetes.io/unreachable",
							Effect:    corev1.TaintEffectNoSchedule,
							TimeAdded: mustTime("2026-07-31T20:39:08Z"),
						},
						{
							Key:       "node.kubernetes.io/unreachable",
							Effect:    corev1.TaintEffectNoExecute,
							TimeAdded: mustTime("2026-07-31T20:39:18Z"),
						},
					},
				},
			},
			pNode: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{},
				Spec: corev1.NodeSpec{
					Taints: []corev1.Taint{
						{
							Key:       "node.kubernetes.io/unreachable",
							Effect:    corev1.TaintEffectNoSchedule,
							TimeAdded: mustTime("2026-07-31T20:30:00Z"),
						},
						{
							Key:       "node.kubernetes.io/unreachable",
							Effect:    corev1.TaintEffectNoExecute,
							TimeAdded: mustTime("2026-07-31T20:30:10Z"),
						},
					},
				},
			},

			expectedAnnotations: map[string]string{
				TaintsAnnotation: mustMarshal([]string{
					mustMarshal(&corev1.Taint{
						Key:       "node.kubernetes.io/unreachable",
						Effect:    corev1.TaintEffectNoSchedule,
						TimeAdded: mustTime("2026-07-31T20:30:00Z"),
					}),
					mustMarshal(&corev1.Taint{
						Key:       "node.kubernetes.io/unreachable",
						Effect:    corev1.TaintEffectNoExecute,
						TimeAdded: mustTime("2026-07-31T20:30:10Z"),
					}),
				}),
			},
			expectedLabels: map[string]string{},
			expectedTaints: []corev1.Taint{
				{
					Key:       "node.kubernetes.io/unreachable",
					Effect:    corev1.TaintEffectNoSchedule,
					TimeAdded: mustTime("2026-07-31T20:30:00Z"),
				},
				{
					Key:       "node.kubernetes.io/unreachable",
					Effect:    corev1.TaintEffectNoExecute,
					TimeAdded: mustTime("2026-07-31T20:30:10Z"),
				},
			},
		},
		{
			name: "not-ready-different-timeAdded",

			vNode: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{},
				Spec: corev1.NodeSpec{
					Taints: []corev1.Taint{
						{
							Key:       "node.kubernetes.io/not-ready",
							Effect:    corev1.TaintEffectNoSchedule,
							TimeAdded: mustTime("2026-07-31T20:39:08Z"),
						},
						{
							Key:       "node.kubernetes.io/not-ready",
							Effect:    corev1.TaintEffectNoExecute,
							TimeAdded: mustTime("2026-07-31T20:39:18Z"),
						},
					},
				},
			},
			pNode: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{},
				Spec: corev1.NodeSpec{
					Taints: []corev1.Taint{
						{
							Key:       "node.kubernetes.io/not-ready",
							Effect:    corev1.TaintEffectNoSchedule,
							TimeAdded: mustTime("2026-07-31T20:30:00Z"),
						},
						{
							Key:       "node.kubernetes.io/not-ready",
							Effect:    corev1.TaintEffectNoExecute,
							TimeAdded: mustTime("2026-07-31T20:30:10Z"),
						},
					},
				},
			},

			expectedAnnotations: map[string]string{
				TaintsAnnotation: mustMarshal([]string{
					mustMarshal(&corev1.Taint{
						Key:       "node.kubernetes.io/not-ready",
						Effect:    corev1.TaintEffectNoSchedule,
						TimeAdded: mustTime("2026-07-31T20:30:00Z"),
					}),
					mustMarshal(&corev1.Taint{
						Key:       "node.kubernetes.io/not-ready",
						Effect:    corev1.TaintEffectNoExecute,
						TimeAdded: mustTime("2026-07-31T20:30:10Z"),
					}),
				}),
			},
			expectedLabels: map[string]string{},
			expectedTaints: []corev1.Taint{
				{
					Key:       "node.kubernetes.io/not-ready",
					Effect:    corev1.TaintEffectNoSchedule,
					TimeAdded: mustTime("2026-07-31T20:30:00Z"),
				},
				{
					Key:       "node.kubernetes.io/not-ready",
					Effect:    corev1.TaintEffectNoExecute,
					TimeAdded: mustTime("2026-07-31T20:30:10Z"),
				},
			},
		},
		{
			name: "unreachable-virtual-only",

			vNode: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{},
				Spec: corev1.NodeSpec{
					Taints: []corev1.Taint{
						{
							Key:       "node.kubernetes.io/unreachable",
							Effect:    corev1.TaintEffectNoExecute,
							TimeAdded: mustTime("2026-07-31T20:39:18Z"),
						},
					},
				},
			},
			pNode: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{},
				Spec:       corev1.NodeSpec{},
			},

			expectedAnnotations: map[string]string{},
			expectedLabels:      map[string]string{},
			expectedTaints:      []corev1.Taint{},
		},
		{
			name: "unreachable-removed-different-timeAdded",

			vNode: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						TaintsAnnotation: mustMarshal([]string{
							mustMarshal(&corev1.Taint{
								Key:       "node.kubernetes.io/unreachable",
								Effect:    corev1.TaintEffectNoExecute,
								TimeAdded: mustTime("2026-07-31T20:30:10Z"),
							}),
						}),
					},
				},
				Spec: corev1.NodeSpec{
					Taints: []corev1.Taint{
						{
							Key:       "node.kubernetes.io/unreachable",
							Effect:    corev1.TaintEffectNoExecute,
							TimeAdded: mustTime("2026-07-31T20:39:18Z"),
						},
					},
				},
			},
			pNode: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{},
				Spec:       corev1.NodeSpec{},
			},

			expectedAnnotations: map[string]string{},
			expectedLabels:      map[string]string{},
			expectedTaints:      []corev1.Taint{},
		},
		{
			name: "custom-taint-virtual-only",

			vNode: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{},
				Spec: corev1.NodeSpec{
					Taints: []corev1.Taint{
						{
							Key:    "custom-taint",
							Value:  "true",
							Effect: corev1.TaintEffectNoExecute,
						},
					},
				},
			},
			pNode: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{},
				Spec: corev1.NodeSpec{
					Taints: []corev1.Taint{},
				},
			},

			expectedAnnotations: map[string]string{},
			expectedLabels:      map[string]string{},
			expectedTaints: []corev1.Taint{
				{
					Key:    "custom-taint",
					Value:  "true",
					Effect: corev1.TaintEffectNoExecute,
				},
			},
		},
		{
			name: "host-value-wins",

			vNode: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{},
				Spec: corev1.NodeSpec{
					Taints: []corev1.Taint{
						{
							Key:    "custom-taint",
							Value:  "old",
							Effect: corev1.TaintEffectNoExecute,
						},
					},
				},
			},
			pNode: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{},
				Spec: corev1.NodeSpec{
					Taints: []corev1.Taint{
						{
							Key:    "custom-taint",
							Value:  "new",
							Effect: corev1.TaintEffectNoExecute,
						},
					},
				},
			},

			expectedAnnotations: map[string]string{
				TaintsAnnotation: mustMarshal([]string{
					mustMarshal(&corev1.Taint{
						Key:    "custom-taint",
						Value:  "new",
						Effect: corev1.TaintEffectNoExecute,
					}),
				}),
			},
			expectedLabels: map[string]string{},
			expectedTaints: []corev1.Taint{
				{
					Key:    "custom-taint",
					Value:  "new",
					Effect: corev1.TaintEffectNoExecute,
				},
			},
		},
	}

	// test cases
	s := &nodeSyncer{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := testCase.vNode.DeepCopy()
			s.translateUpdateBackwards(testCase.pNode, result)
			if result == nil {
				result = testCase.vNode
			}
			assert.DeepEqual(t, result.Annotations, testCase.expectedAnnotations)
			assert.DeepEqual(t, result.Labels, testCase.expectedLabels)
			assert.DeepEqual(t, result.Spec.Taints, testCase.expectedTaints)
		})
	}
}

func mustMarshal(i interface{}) string {
	out, err := json.Marshal(i)
	if err != nil {
		panic(err)
	}

	return string(out)
}

func mustTime(s string) *metav1.Time {
	parsed, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	mt := metav1.NewTime(parsed)
	return &mt
}
