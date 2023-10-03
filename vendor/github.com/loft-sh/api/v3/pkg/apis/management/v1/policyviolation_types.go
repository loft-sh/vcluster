package v1

import (
	clusterv1 "github.com/loft-sh/agentapi/v3/pkg/apis/loft/cluster/v1"
	policyv1beta1 "github.com/loft-sh/jspolicy/pkg/apis/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PolicyViolation
// +k8s:openapi-gen=true
// +resource:path=policyviolations,rest=PolicyViolationREST
type PolicyViolation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PolicyViolationSpec   `json:"spec,omitempty"`
	Status PolicyViolationStatus `json:"status,omitempty"`
}

type PolicyViolationSpec struct {
}

type PolicyViolationStatus struct {
	// Policy is the name of the policy where the violation occurred
	// +optional
	Policy string `json:"policy,omitempty"`

	// Cluster is the cluster where the violation occurred in
	// +optional
	Cluster string `json:"cluster,omitempty"`

	// The Loft user that caused the violation
	// +optional
	User *clusterv1.EntityInfo `json:"user,omitempty"`

	// Violation contains information about the violation
	// +optional
	Violation policyv1beta1.PolicyViolation `json:"violation,omitempty"`
}
