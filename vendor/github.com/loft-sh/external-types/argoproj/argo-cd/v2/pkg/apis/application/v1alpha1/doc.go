// Api versions allow the api contract for a resource to be changed while keeping
// backward compatibility by support multiple concurrent versions
// of the same resource

// +k8s:deepcopy-gen=package,register
// +k8s:defaulter-gen=TypeMeta
// +k8s:openapi-gen=true
// +k8s:openapi-model-package=com.github.loft-sh.external-types.argoproj.argo-cd.v2.pkg.apis.application.v1alpha1
// +groupName=application.argoproj.io
package v1alpha1 // import "github.com/loft-sh/external-types/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
