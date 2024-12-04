package translate

import (
	"testing"

	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"gotest.tools/v3/assert"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
