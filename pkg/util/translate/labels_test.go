package translate

import (
	"testing"

	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"gotest.tools/v3/assert"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
)

func TestAnnotationsSync(t *testing.T) {
	// exclude the default class
	vAnnotations, pAnnotations := AnnotationsBidirectionalUpdate(synccontext.NewSyncEventWithOld(
		&storagev1.StorageClass{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"storageclass.kubernetes.io/is-default-class": "my-default",
				},
			},
		},
		&storagev1.StorageClass{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"storageclass.kubernetes.io/is-default-class": "my-other",
				},
			},
		},
		&storagev1.StorageClass{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"storageclass.kubernetes.io/is-default-class": "my-default",
				},
			},
		},
		&storagev1.StorageClass{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"storageclass.kubernetes.io/is-default-class": "my-default",
				},
			},
		},
	), "storageclass.kubernetes.io/is-default-class")
	assert.DeepEqual(t, vAnnotations, map[string]string{
		"storageclass.kubernetes.io/is-default-class": "my-default",
	})
	assert.DeepEqual(t, pAnnotations, map[string]string{
		"storageclass.kubernetes.io/is-default-class": "my-other",
		NameAnnotation:     "",
		UIDAnnotation:      "",
		KindAnnotation:     storagev1.SchemeGroupVersion.WithKind("StorageClass").String(),
		HostNameAnnotation: "",
	})

	// not exclude the default class
	vAnnotations, pAnnotations = AnnotationsBidirectionalUpdate(synccontext.NewSyncEventWithOld(
		&storagev1.StorageClass{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"storageclass.kubernetes.io/is-default-class": "my-default",
				},
			},
		},
		&storagev1.StorageClass{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"storageclass.kubernetes.io/is-default-class": "my-other",
				},
			},
		},
		&storagev1.StorageClass{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"storageclass.kubernetes.io/is-default-class": "my-default",
				},
			},
		},
		&storagev1.StorageClass{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"storageclass.kubernetes.io/is-default-class": "my-default",
				},
			},
		},
	))
	assert.DeepEqual(t, vAnnotations, map[string]string{
		"storageclass.kubernetes.io/is-default-class": "my-other",
	})
	assert.DeepEqual(t, pAnnotations, map[string]string{
		"storageclass.kubernetes.io/is-default-class": "my-other",
		NameAnnotation:     "",
		UIDAnnotation:      "",
		KindAnnotation:     storagev1.SchemeGroupVersion.WithKind("StorageClass").String(),
		HostNameAnnotation: "",
	})

	// check on creation with exclude host -> virtual
	pObj := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"storageclass.kubernetes.io/is-default-class": "my-default",
				"not-excluded": "true",
			},
		},
	}
	vObj := CopyObjectWithName(pObj, types.NamespacedName{}, true, "storageclass.kubernetes.io/is-default-class")
	vAnnotations = VirtualAnnotations(pObj, vObj, "storageclass.kubernetes.io/is-default-class")
	assert.DeepEqual(t, vAnnotations, map[string]string{
		"not-excluded": "true",
	})

	// check on creation with exclude virtual -> host
	vObj = &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"storageclass.kubernetes.io/is-default-class": "my-default",
				"not-excluded": "true",
			},
		},
	}
	pObj = CopyObjectWithName(vObj, types.NamespacedName{}, true, "storageclass.kubernetes.io/is-default-class")
	pAnnotations = HostAnnotations(vObj, pObj, "storageclass.kubernetes.io/is-default-class")
	assert.DeepEqual(t, pAnnotations, map[string]string{
		"not-excluded":               "true",
		ManagedAnnotationsAnnotation: "not-excluded",
		NameAnnotation:               "",
		UIDAnnotation:                "",
		KindAnnotation:               storagev1.SchemeGroupVersion.WithKind("StorageClass").String(),
		HostNameAnnotation:           "",
	})
}
func TestVirtualLabels(t *testing.T) {
	testCases := []struct {
		name                  string
		hostLabels            map[string]string
		virtualLabels         map[string]string
		expectedVirtualLabels map[string]string
	}{
		{
			name: "Host object labels are copied to the virtual resource",
			hostLabels: map[string]string{
				"example.com/hello": "world",
			},
			virtualLabels: map[string]string{},
			expectedVirtualLabels: map[string]string{
				"example.com/hello": "world",
				SyncDirectionLabel:  string(synccontext.SyncHostToVirtual),
			},
		},
		{
			name:          "Sync direction label is set on the virtual resource",
			hostLabels:    map[string]string{},
			virtualLabels: map[string]string{},
			expectedVirtualLabels: map[string]string{
				SyncDirectionLabel: string(synccontext.SyncHostToVirtual),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hostResource := &unstructured.Unstructured{}
			hostResource.SetLabels(tc.hostLabels)
			virtualResource := &unstructured.Unstructured{}
			virtualResource.SetLabels(tc.virtualLabels)

			virtualLabels := VirtualLabels(hostResource, virtualResource)
			assert.DeepEqual(t, tc.expectedVirtualLabels, virtualLabels)
		})
	}
}
