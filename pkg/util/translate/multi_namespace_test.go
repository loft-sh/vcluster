package translate

import (
	"testing"

	"github.com/loft-sh/vcluster/config"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testNamespace    = "multi"
	testPrefix       = "prefix"
	testSuffix       = "suffix"
	testSuffixSha256 = "8a237015" // sha256("multixsuffix")
	testBase         = "foo-bar-baz"
	testBaseSha256   = "269dce1a"
)

func TestMultiNamespaceTranslator_IsManaged(t *testing.T) {
	testCases := []struct {
		name       string
		nameFormat config.ExperimentalMultiNamespaceNameFormat
		obj        client.Object
		wantRes    bool
	}{
		{
			name: "managed - default namespace name format",
			nameFormat: config.ExperimentalMultiNamespaceNameFormat{
				Prefix: testPrefix,
			},
			obj: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod-name",
					Namespace: testPrefix + "-base-" + testSuffixSha256,
					Annotations: map[string]string{
						NameAnnotation: "foobar",
					},
				},
			},
			wantRes: true,
		},
		{
			name: "unmanaged - default namespace name format (different prefix)",
			nameFormat: config.ExperimentalMultiNamespaceNameFormat{
				Prefix: testPrefix,
			},
			obj: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod-name",
					Namespace: "other-base-" + testSuffixSha256,
					Annotations: map[string]string{
						NameAnnotation: "foobar",
					},
				},
			},
			wantRes: false,
		},
		{
			name: "unmanaged - default namespace name format (different suffix)",
			nameFormat: config.ExperimentalMultiNamespaceNameFormat{
				Prefix: testPrefix,
			},
			obj: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod-name",
					Namespace: testPrefix + "-base-aaaaaaaa",
					Annotations: map[string]string{
						NameAnnotation: "foobar",
					},
				},
			},
			wantRes: false,
		},
		{
			name: "managed - custom namespace name format (raw suffix)",
			nameFormat: config.ExperimentalMultiNamespaceNameFormat{
				Prefix:    testPrefix,
				RawSuffix: true,
			},
			obj: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod-name",
					Namespace: testPrefix + "-base-" + testSuffix,
					Annotations: map[string]string{
						NameAnnotation: "foobar",
					},
				},
			},
			wantRes: true,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			translator := &multiNamespace{
				currentNamespace: testNamespace,
				nameFormat:       tt.nameFormat,
			}
			res := translator.IsManaged(nil, tt.obj)
			if tt.wantRes != res {
				t.Errorf("wanted result to be %v but got %v", tt.wantRes, res)
			}
		})
	}
}

func TestMultiNamespaceTranslator_HostNamespace(t *testing.T) {
	testCases := []struct {
		name       string
		nameFormat config.ExperimentalMultiNamespaceNameFormat
		namespace  string
		wantRes    string
	}{
		{
			name: "default namespace name format",
			nameFormat: config.ExperimentalMultiNamespaceNameFormat{
				Prefix: testPrefix,
			},
			namespace: testBase,
			wantRes:   testPrefix + "-" + testBaseSha256 + "-" + testSuffixSha256,
		},
		{
			name: "custom namespace name format (raw base)",
			nameFormat: config.ExperimentalMultiNamespaceNameFormat{
				Prefix:  testPrefix,
				RawBase: true,
			},
			namespace: testBase,
			wantRes:   testPrefix + "-" + testBase + "-" + testSuffixSha256,
		},
		{
			name: "custom namespace name format (raw suffix)",
			nameFormat: config.ExperimentalMultiNamespaceNameFormat{
				Prefix:    testPrefix,
				RawSuffix: true,
			},
			namespace: testBase,
			wantRes:   testPrefix + "-" + testBaseSha256 + "-" + testSuffix,
		},
		{
			name: "custom namespace name format (raw base and suffix)",
			nameFormat: config.ExperimentalMultiNamespaceNameFormat{
				Prefix:    testPrefix,
				RawBase:   true,
				RawSuffix: true,
			},
			namespace: testBase,
			wantRes:   testPrefix + "-" + testBase + "-" + testSuffix,
		},
		{
			name: "custom namespace name format without redundancy",
			nameFormat: config.ExperimentalMultiNamespaceNameFormat{
				Prefix:                   testPrefix,
				RawBase:                  true,
				RawSuffix:                true,
				AvoidRedundantFormatting: true,
			},
			namespace: testPrefix + "-" + testBase + "-" + testSuffix,
			wantRes:   testPrefix + "-" + testBase + "-" + testSuffix,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			translator := &multiNamespace{
				currentNamespace: testNamespace,
				nameFormat:       tt.nameFormat,
			}
			res := translator.HostNamespace(tt.namespace)
			if tt.wantRes != res {
				t.Errorf("wanted result to be %q but got %q", tt.wantRes, res)
			}
		})
	}
}
