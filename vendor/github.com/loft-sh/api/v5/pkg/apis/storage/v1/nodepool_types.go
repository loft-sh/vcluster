package v1

import (
	agentstoragev1 "github.com/loft-sh/agentapi/v5/pkg/apis/loft/storage/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// NodePoolLabel, on a NodeClaim, names the NodePool in the claim's namespace the claim
	// belongs to. The platform sets it on the claims it creates for a static pool, and the
	// cluster sets it on the claims Karpenter creates for a dynamic one.
	NodePoolLabel = "vcluster.com/node-pool"

	// NodePoolTemplateHashLabel, on a NodeClaim of a static pool, records the hash of the pool
	// template the claim was created from. A claim whose hash differs from the pool's current
	// one is outdated and gets replaced.
	NodePoolTemplateHashLabel = "vcluster.com/node-pool-template-hash"

	// NodePoolFinalizer holds a NodePool until the claims it is responsible for are gone.
	NodePoolFinalizer = "vcluster.com/node-pool"
)

const (
	// NodePoolConditionTypeSupported is True when the pool may run: the license allows it, its
	// cluster exists and, for a dynamic pool, runs a vCluster version that reads node pools
	// from the platform.
	NodePoolConditionTypeSupported agentstoragev1.ConditionType = "Supported"
	// NodePoolConditionTypeNetworkReady is True when the network environment the pool places
	// its nodes in is available. It is absent while the pool places its nodes in none.
	NodePoolConditionTypeNetworkReady agentstoragev1.ConditionType = "NetworkReady"
	// NodePoolConditionTypeScaled is True when a static pool has every replica created from
	// the current template and available.
	NodePoolConditionTypeScaled agentstoragev1.ConditionType = "Scaled"
	// NodePoolConditionTypeAutoscalerReady is True when the cluster has applied the current
	// generation of a dynamic pool and reports its autoscaler ready.
	NodePoolConditionTypeAutoscalerReady agentstoragev1.ConditionType = "AutoscalerReady"
)

const (
	// NodePoolReasonFeatureNotAllowed marks a pool the license does not allow.
	NodePoolReasonFeatureNotAllowed = "FeatureNotAllowed"
	// NodePoolReasonVClusterNotFound marks a pool whose cluster does not exist.
	NodePoolReasonVClusterNotFound = "VClusterNotFound"
	// NodePoolReasonVClusterDeleting marks a pool whose cluster is being deleted.
	NodePoolReasonVClusterDeleting = "VClusterDeleting"
	// NodePoolReasonVClusterVersionUnknown marks a dynamic pool whose cluster has not reported
	// a version yet.
	NodePoolReasonVClusterVersionUnknown = "VClusterVersionUnknown"
	// NodePoolReasonVClusterVersionUnsupported marks a dynamic pool whose cluster runs a
	// vCluster version that does not read node pools from the platform.
	NodePoolReasonVClusterVersionUnsupported = "VClusterVersionUnsupported"
	// NodePoolReasonNodeProviderNotFound marks a pool whose node provider does not exist.
	NodePoolReasonNodeProviderNotFound = "NodeProviderNotFound"
	// NodePoolReasonNetworkEnvironmentNotAllowed marks a pool whose network environment the
	// project does not allow.
	NodePoolReasonNetworkEnvironmentNotAllowed = "NetworkEnvironmentNotAllowed"
	// NodePoolReasonNetworkEnvironmentNotFound marks a pool whose network environment does not
	// exist or is not available to the project's tenant.
	NodePoolReasonNetworkEnvironmentNotFound = "NetworkEnvironmentNotFound"
	// NodePoolReasonNetworkEnvironmentNotAvailable marks a pool whose network environment is
	// still being provisioned, has failed or is being deleted.
	NodePoolReasonNetworkEnvironmentNotAvailable = "NetworkEnvironmentNotAvailable"
)

// NodePoolPhase defines the phase of the NodePool
type NodePoolPhase string

const (
	// NodePoolPhasePending means the pool is scaling, rolling out or waiting for its cluster.
	NodePoolPhasePending NodePoolPhase = "Pending"
	// NodePoolPhaseReady means the pool has converged.
	NodePoolPhaseReady NodePoolPhase = "Ready"
	// NodePoolPhaseFailed means the pool cannot run, see its conditions.
	NodePoolPhaseFailed NodePoolPhase = "Failed"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="VCluster",type="string",JSONPath=".spec.vClusterRef"
// +kubebuilder:printcolumn:name="Replicas",type="integer",JSONPath=".status.replicas"
// +kubebuilder:printcolumn:name="Ready",type="integer",JSONPath=".status.readyReplicas"
// +kubebuilder:printcolumn:name="Network",type="string",JSONPath=".status.networkEnvironment"
// +kubebuilder:subresource:status

// NodePool keeps a set of NodeClaims for a cluster that are all built from one template.
// +k8s:openapi-gen=true
type NodePool struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NodePoolSpec   `json:"spec,omitempty"`
	Status NodePoolStatus `json:"status,omitempty"`
}

func (a *NodePool) GetConditions() agentstoragev1.Conditions {
	return a.Status.Conditions
}

func (a *NodePool) SetConditions(conditions agentstoragev1.Conditions) {
	a.Status.Conditions = conditions
}

func (a *NodePool) GetOwner() *UserOrTeam {
	return a.Spec.Owner
}

func (a *NodePool) SetOwner(userOrTeam *UserOrTeam) {
	a.Spec.Owner = userOrTeam
}

func (a *NodePool) GetAccess() []Access {
	return a.Spec.Access
}

func (a *NodePool) SetAccess(access []Access) {
	a.Spec.Access = access
}

// NodePoolSpec defines spec of node pool. A pool is either static, a fixed number of nodes the
// platform creates, or dynamic, scaled by Karpenter inside the cluster. Exactly one of the two is
// set, and a pool cannot switch between them.
// +kubebuilder:validation:XValidation:rule="has(self.static) != has(self.dynamic)",message="exactly one of static or dynamic must be set"
// +kubebuilder:validation:XValidation:rule="has(self.static) == has(oldSelf.static)",message="a node pool cannot switch between static and dynamic"
type NodePoolSpec struct {
	// DisplayName is the name of the NodePool that is displayed in the UI.
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// Owner holds the owner of this object
	// +optional
	Owner *UserOrTeam `json:"owner,omitempty"`

	// Access holds the access rights for users and teams
	// +optional
	Access []Access `json:"access,omitempty"`

	// VClusterRef is the name of the cluster in this namespace the pool's nodes join.
	// It cannot be changed.
	VClusterRef string `json:"vClusterRef"`

	// Static keeps a fixed number of nodes. The platform creates the pool's NodeClaims and,
	// when the template changes, replaces them one at a time. Mutually exclusive with Dynamic.
	// +optional
	Static *NodePoolStatic `json:"static,omitempty"`

	// Dynamic hands the pool to Karpenter inside the cluster, which creates nodes for the pods
	// that need them, removes the ones it can consolidate and replaces drifted ones within the
	// disruption budgets. The cluster has to run a vCluster version that reads node pools from
	// the platform. Mutually exclusive with Static.
	// +optional
	Dynamic *NodePoolDynamic `json:"dynamic,omitempty"`

	// Template is what every node of the pool is built from.
	Template NodePoolTemplate `json:"template"`
}

// NodePoolStatic configures a static pool.
type NodePoolStatic struct {
	// Replicas is the number of nodes the platform keeps for the pool.
	// +kubebuilder:validation:Minimum=0
	Replicas int32 `json:"replicas"`
}

// NodePoolTemplate describes the nodes of a pool.
type NodePoolTemplate struct {
	// ProviderRef is the name of the NodeProvider the pool's nodes are provisioned with.
	ProviderRef string `json:"providerRef"`

	// Requirements select the NodeTypes the pool's nodes may be scheduled onto.
	// +optional
	Requirements []corev1.NodeSelectorRequirement `json:"requirements,omitempty"`

	// Properties are extra properties for the pool's NodeClaims. The
	// vcluster.com/network-environment property places the pool's nodes in that network
	// environment; without it, the pool uses its provider's default, see
	// status.networkEnvironment.
	// +optional
	Properties map[string]string `json:"properties,omitempty"`

	// Taints are applied to the pool's nodes.
	// +optional
	Taints []corev1.Taint `json:"taints,omitempty"`

	// NodeLabels are applied to the pool's nodes. Labels in the kubernetes.io and k8s.io
	// namespaces are refused, except below node.kubernetes.io, since the kubelet may not set them.
	// +optional
	NodeLabels map[string]string `json:"nodeLabels,omitempty"`

	// ProfileRef is the name of a NodeProfile to apply to the pool's nodes. It must be
	// permitted by the project's allowedNodeProfiles.
	// +optional
	ProfileRef string `json:"profileRef,omitempty"`
}

// NodePoolDynamic configures the Karpenter NodePool the cluster creates for a dynamic pool.
type NodePoolDynamic struct {
	// Limits cap the total resources of the pool's nodes, for example cpu: "64".
	// +optional
	Limits corev1.ResourceList `json:"limits,omitempty"`

	// Disruption configures how Karpenter consolidates the pool's nodes.
	// +optional
	Disruption NodePoolDisruption `json:"disruption,omitempty"`

	// ExpireAfter is how long a node lives before Karpenter replaces it, for example 720h.
	// "Never" or empty keeps nodes forever.
	// +optional
	ExpireAfter string `json:"expireAfter,omitempty"`

	// TerminationGracePeriod is how long Karpenter waits for pods to drain before it forcibly
	// removes a node. "Never" waits forever. Defaults to 30s.
	// +optional
	TerminationGracePeriod string `json:"terminationGracePeriod,omitempty"`

	// Weight orders this pool against the cluster's other dynamic pools; higher weights are
	// tried first.
	// +optional
	Weight *int32 `json:"weight,omitempty"`
}

// NodePoolDisruption configures Karpenter consolidation.
type NodePoolDisruption struct {
	// ConsolidationPolicy is WhenEmpty or WhenEmptyOrUnderutilized (the default).
	// +optional
	ConsolidationPolicy string `json:"consolidationPolicy,omitempty"`

	// ConsolidateAfter is how long a node has to be consolidatable before Karpenter acts.
	// Defaults to 0s.
	// +optional
	ConsolidateAfter string `json:"consolidateAfter,omitempty"`

	// Budgets limit how many nodes Karpenter may disrupt at once. Defaults to 10%.
	// +optional
	Budgets []NodePoolDisruptionBudget `json:"budgets,omitempty"`
}

// NodePoolDisruptionBudget limits disruption, optionally during a schedule.
type NodePoolDisruptionBudget struct {
	// Nodes is a count or percentage of the pool's nodes that may be disrupted at once.
	// +optional
	Nodes string `json:"nodes,omitempty"`

	// Schedule is a cron schedule the budget is active from.
	// +optional
	Schedule string `json:"schedule,omitempty"`

	// Duration is how long the budget is active after each schedule hit.
	// +optional
	Duration string `json:"duration,omitempty"`
}

type NodePoolStatus struct {
	// Phase is the current lifecycle phase of the NodePool.
	// +optional
	Phase NodePoolPhase `json:"phase,omitempty"`

	// Reason describes the reason in machine-readable form
	// +optional
	Reason string `json:"reason,omitempty"`

	// Message describes the reason in human-readable form
	// +optional
	Message string `json:"message,omitempty"`

	// ObservedGeneration is the pool generation the platform last reconciled.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Replicas is the number of NodeClaims of the pool that are not being deleted.
	// +optional
	Replicas int32 `json:"replicas,omitempty"`

	// ReadyReplicas is the number of the pool's NodeClaims that are available.
	// +optional
	ReadyReplicas int32 `json:"readyReplicas,omitempty"`

	// UpdatedReplicas is the number of the pool's NodeClaims built from the current template.
	// Only static pools track it.
	// +optional
	UpdatedReplicas int32 `json:"updatedReplicas,omitempty"`

	// TemplateHash is the hash of the current template, as set on the NodeClaims of a static
	// pool.
	// +optional
	TemplateHash string `json:"templateHash,omitempty"`

	// NetworkEnvironment is the network environment the pool places its nodes in: the one the
	// template or the node provider names, else the provider's default when the pool first
	// found one. A pool keeps it when the default changes, so all its nodes share one network.
	// +optional
	NetworkEnvironment string `json:"networkEnvironment,omitempty"`

	// Conditions describe the current state of the NodePool.
	// +optional
	Conditions agentstoragev1.Conditions `json:"conditions,omitempty"`

	// Dynamic is reported by the cluster for a dynamic pool. The platform never writes it.
	// +optional
	Dynamic *NodePoolDynamicStatus `json:"dynamic,omitempty"`
}

// NodePoolDynamicStatus is the state of the Karpenter NodePool the cluster created for a
// dynamic pool.
type NodePoolDynamicStatus struct {
	// ObservedGeneration is the pool generation the cluster last applied.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Resources are the resources of the pool's nodes, to compare against the limits.
	// +optional
	Resources corev1.ResourceList `json:"resources,omitempty"`

	// Ready is true when Karpenter reports the pool ready.
	// +optional
	Ready bool `json:"ready,omitempty"`

	// Reason describes why the pool is not ready in machine-readable form.
	// +optional
	Reason string `json:"reason,omitempty"`

	// Message describes why the pool is not ready in human-readable form.
	// +optional
	Message string `json:"message,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodePoolList contains a list of NodePool
type NodePoolList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NodePool `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NodePool{}, &NodePoolList{})
}
