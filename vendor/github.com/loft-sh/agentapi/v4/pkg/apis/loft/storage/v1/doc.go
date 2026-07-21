// Api versions allow the api contract for a resource to be changed while keeping
// backward compatibility by support multiple concurrent versions
// of the same resource

// +k8s:openapi-gen=true
// +k8s:openapi-model-package=com.github.loft-sh.agentapi.v4.pkg.apis.loft.storage.v1
// +k8s:deepcopy-gen=package
// +k8s:defaulter-gen=TypeMeta
// +groupName=storage.loft.sh
package v1 // import "github.com/loft-sh/agentapi/v4/apis/loft/storage/v1"
