package v1

import (
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodeProvider holds the information of a node provider config.
// This resource defines various ways a node can be provisioned or configured.
// +k8s:openapi-gen=true
// +resource:path=nodeproviders,rest=NodeProviderREST,statusRest=NodeProviderStatusREST
// +subresource:request=NodeProviderExec,path=exec,kind=NodeProviderExec,rest=NodeProviderExecREST
type NodeProvider struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NodeProviderSpec   `json:"spec,omitempty"`
	Status NodeProviderStatus `json:"status,omitempty"`
}

// NodeProviderSpec defines the desired state of NodeProvider.
// Only one of the provider types (Pods, BCM, Kubevirt) should be specified at a time.
type NodeProviderSpec struct {
	storagev1.NodeProviderSpec `json:",inline"`
}

// NodeProviderStatus defines the observed state of NodeProvider.
type NodeProviderStatus struct {
	storagev1.NodeProviderStatus `json:",inline"`
}
