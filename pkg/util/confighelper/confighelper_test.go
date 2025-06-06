package confighelper

import (
	"context"
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestUpdateAnnotations(t *testing.T) {
	tests := []struct {
		name             string
		annotations      map[string]string
		distro           string
		backingStoreType string
		expected         map[string]string
		returnedChanged  bool
	}{
		{
			name:             "nil annotations",
			annotations:      nil,
			distro:           "k3s",
			backingStoreType: "etcd",
			expected: map[string]string{
				AnnotationDistro: "k3s",
				AnnotationStore:  "etcd",
			},
			returnedChanged: true,
		},
		{
			name:             "empty annotations",
			annotations:      map[string]string{},
			distro:           "k8s",
			backingStoreType: "etcd",
			expected: map[string]string{
				AnnotationDistro: "k8s",
				AnnotationStore:  "etcd",
			},
			returnedChanged: true,
		},
		{
			name: "existing annotations no change",
			annotations: map[string]string{
				AnnotationDistro: "k3s",
				AnnotationStore:  "etcd",
			},
			distro:           "k3s",
			backingStoreType: "etcd",
			expected: map[string]string{
				AnnotationDistro: "k3s",
				AnnotationStore:  "etcd",
			},
			returnedChanged: false,
		},
		{
			name: "existing annotations with change",
			annotations: map[string]string{
				AnnotationDistro: "k3s",
				AnnotationStore:  "etcd",
				"other":          "value",
			},
			distro:           "k8s",
			backingStoreType: "etcd",
			expected: map[string]string{
				AnnotationDistro: "k8s",
				AnnotationStore:  "etcd",
				"other":          "value",
			},
			returnedChanged: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			annotations := test.annotations
			changed := UpdateAnnotations(&annotations, test.distro, test.backingStoreType)

			if changed != test.returnedChanged {
				t.Errorf("Expected change: %v, got: %v", test.returnedChanged, changed)
			}

			if !reflect.DeepEqual(annotations, test.expected) {
				t.Errorf("Expected annotations: %v, got: %v", test.expected, annotations)
			}
		})
	}
}

func TestGetVClusterConfigResource(t *testing.T) {
	tests := []struct {
		name             string
		setupClientset   func() *fake.Clientset
		expectedError    bool
		expectedContains string
	}{
		{
			name: "secret exists",
			setupClientset: func() *fake.Clientset {
				client := fake.NewSimpleClientset()
				secret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "vc-config-test",
						Namespace: "test-namespace",
					},
					Data: map[string][]byte{
						"config.yaml": []byte("distro: k3s"),
					},
				}
				_, err := client.CoreV1().Secrets("test-namespace").Create(context.Background(), secret, metav1.CreateOptions{})
				if err != nil {
					panic(err)
				}
				return client
			},
			expectedError:    false,
			expectedContains: "distro: k3s",
		},
		{
			name: "configmap exists",
			setupClientset: func() *fake.Clientset {
				client := fake.NewSimpleClientset()
				configMap := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "vc-config-test",
						Namespace: "test-namespace",
					},
					Data: map[string]string{
						"config.yaml": "distro: k3s",
					},
				}
				_, err := client.CoreV1().ConfigMaps("test-namespace").Create(context.Background(), configMap, metav1.CreateOptions{})
				if err != nil {
					panic(err)
				}
				return client
			},
			expectedError:    false,
			expectedContains: "distro: k3s",
		},
		{
			name: "neither secret nor configmap exists",
			setupClientset: func() *fake.Clientset {
				return fake.NewSimpleClientset()
			},
			expectedError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := test.setupClientset()
			result, err := GetVClusterConfigResource(context.Background(), client, "test", "test-namespace")

			if test.expectedError && err == nil {
				t.Error("Expected error but got nil")
			}

			if !test.expectedError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if !test.expectedError && err == nil {
				if string(result) != test.expectedContains {
					t.Errorf("Expected result to contain: %s, got: %s", test.expectedContains, string(result))
				}
			}
		})
	}
}

func TestGetResourceAnnotations(t *testing.T) {
	tests := []struct {
		name                string
		setupClientset      func() *fake.Clientset
		expectedError       bool
		expectedAnnotations map[string]string
	}{
		{
			name: "secret exists with annotations",
			setupClientset: func() *fake.Clientset {
				client := fake.NewSimpleClientset()
				secret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "vc-config-test",
						Namespace: "test-namespace",
						Annotations: map[string]string{
							"vcluster.loft.sh/distro": "k3s",
							"vcluster.loft.sh/store":  "etcd",
						},
					},
				}
				_, err := client.CoreV1().Secrets("test-namespace").Create(context.Background(), secret, metav1.CreateOptions{})
				if err != nil {
					panic(err)
				}
				return client
			},
			expectedError: false,
			expectedAnnotations: map[string]string{
				"vcluster.loft.sh/distro": "k3s",
				"vcluster.loft.sh/store":  "etcd",
			},
		},
		{
			name: "configmap exists with annotations",
			setupClientset: func() *fake.Clientset {
				client := fake.NewSimpleClientset()
				configMap := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "vc-config-test",
						Namespace: "test-namespace",
						Annotations: map[string]string{
							"vcluster.loft.sh/distro": "k8s",
							"vcluster.loft.sh/store":  "etcd",
						},
					},
				}
				_, err := client.CoreV1().ConfigMaps("test-namespace").Create(context.Background(), configMap, metav1.CreateOptions{})
				if err != nil {
					panic(err)
				}
				return client
			},
			expectedError: false,
			expectedAnnotations: map[string]string{
				"vcluster.loft.sh/distro": "k8s",
				"vcluster.loft.sh/store":  "etcd",
			},
		},
		{
			name: "neither secret nor configmap exists",
			setupClientset: func() *fake.Clientset {
				return fake.NewSimpleClientset()
			},
			expectedError:       true,
			expectedAnnotations: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := test.setupClientset()
			annotations, err := GetResourceAnnotations(context.Background(), client, "test", "test-namespace")

			if test.expectedError && err == nil {
				t.Error("Expected error but got nil")
			}

			if !test.expectedError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if !test.expectedError && !reflect.DeepEqual(annotations, test.expectedAnnotations) {
				t.Errorf("Expected annotations: %v, got: %v", test.expectedAnnotations, annotations)
			}
		})
	}
}
