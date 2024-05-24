package create

import (
	"testing"

	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	"gotest.tools/v3/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestUpdateLabels(t *testing.T) {
	tests := []struct {
		name      string
		obj       metav1.Object
		labelList []string

		expectedLabels map[string]string
		shouldChange   bool
		shouldErr      bool
	}{
		{
			name: "simple add",
			obj: &managementv1.SpaceInstance{
				ObjectMeta: metav1.ObjectMeta{},
			},
			labelList: []string{
				"a=b",
				"foo=bar",
				"keyonly",
				"", // empty entry should be ignored
			},
			expectedLabels: map[string]string{
				"a":       "b",
				"foo":     "bar",
				"keyonly": "",
			},
			shouldChange: true,
		},
		{
			name: "parse error (extra equals)",
			obj: &managementv1.SpaceInstance{
				ObjectMeta: metav1.ObjectMeta{},
			},
			labelList: []string{
				"a=b=",
			},
			shouldErr: true,
		},
		{
			name: "parse error (extra equals in the middle)",
			obj: &managementv1.SpaceInstance{
				ObjectMeta: metav1.ObjectMeta{},
			},
			labelList: []string{
				"as=dfa=b",
			},
			shouldErr: true,
		},
		{
			name: "update values",
			obj: &managementv1.SpaceInstance{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"a":           "c",
						"foo":         "baz",
						"keyonly":     "",
						"unspecified": "example",
					},
				},
			},
			labelList: []string{
				"a=z",
				"foo=foo",
				"keyonly",
			},
			expectedLabels: map[string]string{
				"a":           "z",
				"foo":         "foo",
				"keyonly":     "",
				"unspecified": "example",
			},
			shouldChange: true,
		},
		{
			name: "no change",
			obj: &managementv1.SpaceInstance{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"a":       "b",
						"foo":     "bar",
						"keyonly": "",
					},
				},
			},
			labelList: []string{
				"a=b",
				"foo=bar",
				"keyonly",
			},
			expectedLabels: map[string]string{
				"a":       "b",
				"foo":     "bar",
				"keyonly": "",
			},
			shouldChange: false,
		},
	}
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("test #%d: %q", i, tt.name)
			got, err := UpdateLabels(tt.obj, tt.labelList)
			if (err != nil) != tt.shouldErr {
				t.Errorf("UpdateLabels() error = %v, wantErr %v", err, tt.shouldErr)
				return
			}

			if !tt.shouldErr {
				if got != tt.shouldChange {
					t.Errorf("UpdateLabels() = %v, want %v", got, tt.shouldChange)
				}

				assert.DeepEqual(t, tt.obj.GetLabels(), tt.expectedLabels)
			}
		})
	}
}

func TestUpdateAnnotations(t *testing.T) {
	tests := []struct {
		name           string
		obj            metav1.Object
		annotationList []string

		expectedAnnotations map[string]string
		shouldChange        bool
		shouldErr           bool
	}{
		{
			name: "simple add",
			obj: &managementv1.SpaceInstance{
				ObjectMeta: metav1.ObjectMeta{},
			},
			annotationList: []string{
				"a=b",
				"foo=bar",
				"keyonly",
				"", // empty entry should be ignored
			},
			expectedAnnotations: map[string]string{
				"a":       "b",
				"foo":     "bar",
				"keyonly": "",
			},
			shouldChange: true,
		},
		{
			name: "parse error (extra equals)",
			obj: &managementv1.SpaceInstance{
				ObjectMeta: metav1.ObjectMeta{},
			},
			annotationList: []string{
				"a=b=",
			},
			shouldErr: true,
		},
		{
			name: "parse error (extra equals in the middle)",
			obj: &managementv1.SpaceInstance{
				ObjectMeta: metav1.ObjectMeta{},
			},
			annotationList: []string{
				"as=dfa=b",
			},
			shouldErr: true,
		},
		{
			name: "update values",
			obj: &managementv1.SpaceInstance{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"a":           "c",
						"foo":         "baz",
						"keyonly":     "",
						"unspecified": "example",
					},
				},
			},
			annotationList: []string{
				"a=z",
				"foo=foo",
				"keyonly",
			},
			expectedAnnotations: map[string]string{
				"a":           "z",
				"foo":         "foo",
				"keyonly":     "",
				"unspecified": "example",
			},
			shouldChange: true,
		},
		{
			name: "no change",
			obj: &managementv1.SpaceInstance{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"a":       "b",
						"foo":     "bar",
						"keyonly": "",
					},
				},
			},
			annotationList: []string{
				"a=b",
				"foo=bar",
				"keyonly",
			},
			expectedAnnotations: map[string]string{
				"a":       "b",
				"foo":     "bar",
				"keyonly": "",
			},
			shouldChange: false,
		},
	}
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("test #%d: %q", i, tt.name)
			got, err := UpdateAnnotations(tt.obj, tt.annotationList)
			if (err != nil) != tt.shouldErr {
				t.Errorf("UpdateAnnotations() error = %v, wantErr %v", err, tt.shouldErr)
				return
			}

			if !tt.shouldErr {
				if got != tt.shouldChange {
					t.Errorf("UpdateAnnotations() = %v, want %v", got, tt.shouldChange)
				}

				assert.DeepEqual(t, tt.obj.GetAnnotations(), tt.expectedAnnotations)
			}
		})
	}
}
