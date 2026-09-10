package filters

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/admission"
)

func NewFakeObjectInterfaces(scheme *runtime.Scheme, restMapper meta.RESTMapper) admission.ObjectInterfaces {
	return &fakeObjectInterfaces{
		scheme:     scheme,
		restMapper: restMapper,
	}
}

type fakeObjectInterfaces struct {
	scheme     *runtime.Scheme
	restMapper meta.RESTMapper
}

func (f *fakeObjectInterfaces) EquivalentResourcesFor(resource schema.GroupVersionResource, subresource string) []schema.GroupVersionResource {
	if subresource != "" {
		return []schema.GroupVersionResource{
			{
				Group:    resource.Group,
				Version:  resource.Version,
				Resource: resource.Resource + "/" + subresource,
			},
		}
	}

	return []schema.GroupVersionResource{
		resource,
	}
}

func (f *fakeObjectInterfaces) KindFor(resource schema.GroupVersionResource, subresource string) schema.GroupVersionKind {
	if resource.Resource == "pods" {
		if subresource == "exec" {
			return corev1.SchemeGroupVersion.WithKind("PodExecOptions")
		} else if subresource == "attach" {
			return corev1.SchemeGroupVersion.WithKind("PodAttachOptions")
		} else if subresource == "portforward" {
			return corev1.SchemeGroupVersion.WithKind("PodPortForwardOptions")
		}
	}

	// this is fake, but works most of the time
	k, _ := f.restMapper.KindFor(resource)
	return k
}

func (f *fakeObjectInterfaces) GetEquivalentResourceMapper() runtime.EquivalentResourceMapper {
	return f
}

func (f *fakeObjectInterfaces) GetObjectConvertor() runtime.ObjectConvertor {
	return f.scheme
}

func (f *fakeObjectInterfaces) GetObjectDefaulter() runtime.ObjectDefaulter {
	return f.scheme
}

func (f *fakeObjectInterfaces) GetObjectCreater() runtime.ObjectCreater {
	return f.scheme
}

func (f *fakeObjectInterfaces) GetObjectTyper() runtime.ObjectTyper {
	return f.scheme
}
