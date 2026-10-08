package v1

import (
	agentstoragev1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// MachineConditionTypeSynced indicates the machine state was confirmed
	// against the provider's inventory on the last sync attempt.
	MachineConditionTypeSynced agentstoragev1.ConditionType = "Synced"

	// MachineNodeClaimAnnotation names the NodeClaim ("<namespace>/<name>")
	// currently consuming this machine. Set by the inventory sync when the
	// provider-side consumer matches a platform NodeClaim; absent when the
	// machine is free or consumed outside the platform.
	MachineNodeClaimAnnotation = "machines.vcluster.com/node-claim"

	// MachineSourceLabel names the inventory a machine was recorded from before
	// the provider knew about it. A machine without it is a plain mirror of the
	// provider inventory and disappears with it; a machine carrying it is owned
	// by that source's sync, which is the only thing allowed to delete it, and
	// the provider sync only fills in its status once the provider lists it.
	MachineSourceLabel = "machines.vcluster.com/source"

	// MachineSourceNetBox is the MachineSourceLabel value for machines imported
	// from a NetBox inventory through NodeProviderMetal3NetBox.
	MachineSourceNetBox = "netbox"

	// MachineConditionTypeRegistered is set on sourced machines: True once the
	// provider holds a record for the machine (metal3: the BareMetalHost
	// exists), False with the reason while it cannot be created.
	MachineConditionTypeRegistered agentstoragev1.ConditionType = "Registered"

	// MachineConditionTypeSourceAttached is set on sourced machines: True while
	// the source still selects the machine, False with reason Orphaned once it
	// no longer does (the NetBox device lost its tag or was deleted) but the
	// machine was kept because it is provisioned or held by a NodeClaim.
	// Releasing it is then an operator's decision, not the sync's.
	MachineConditionTypeSourceAttached agentstoragev1.ConditionType = "SourceAttached"
)

// MachineConditions lists the conditions summarized into the Ready condition.
var MachineConditions = []agentstoragev1.ConditionType{
	MachineConditionTypeSynced,
	MachineConditionTypeRegistered,
}

// MachinePhase is the machine's state as the provider sees it, normalized
// across providers. It says nothing about platform NodeClaims: which claim
// holds a machine is answered by the node-claim annotation.
type MachinePhase string

const (
	// MachinePhasePending means the machine is recorded in the platform from an
	// inventory source (see MachineSourceLabel) but the provider does not list
	// it yet: the BareMetalHost is still to be created, or was rejected. Reason
	// and Message say which.
	MachinePhasePending MachinePhase = "Pending"
	// MachinePhaseAvailable means the provider reports the machine as
	// inventoried and free.
	MachinePhaseAvailable MachinePhase = "Available"
	// MachinePhaseProvisioned means the provider handed the machine to a
	// consumer: metal3 wrote an image or a consumerRef onto the host, NICo
	// assigned it to an instance. Whether that consumer is one of this
	// platform's NodeClaims is answered by the node-claim annotation.
	MachinePhaseProvisioned MachinePhase = "Provisioned"
	// MachinePhaseProvisioning means the provider is working on the machine:
	// registering, inspecting, cleaning, imaging, tearing down. The provider's
	// own word for the step is in Reason, which for metal3 can be its literal
	// "provisioning" as well as "deprovisioning" or "inspecting".
	MachinePhaseProvisioning MachinePhase = "Provisioning"
	// MachinePhaseMaintenance means the machine was deliberately taken out of
	// service (metal3 detached, NICo Maintenance or Decommissioned).
	MachinePhaseMaintenance MachinePhase = "Maintenance"
	// MachinePhaseError means the provider reports a fault. Reason and Message
	// carry the provider's own words for it.
	MachinePhaseError MachinePhase = "Error"
	// MachinePhaseUnknown means the provider could not be reached or reports no
	// state at all, so the mirrored state may be stale.
	MachinePhaseUnknown MachinePhase = "Unknown"
)

// MachinePowerState is the normalized power state of a machine. Empty means
// the provider does not report power at all.
type MachinePowerState string

const (
	MachinePowerStateOn  MachinePowerState = "On"
	MachinePowerStateOff MachinePowerState = "Off"
	// MachinePowerStateUnknown means the provider reports a state that maps
	// to neither On nor Off.
	MachinePowerStateUnknown MachinePowerState = "Unknown"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Provider",type="string",JSONPath=".spec.providerRef"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Power",type="string",JSONPath=".status.powerState"
// +kubebuilder:printcolumn:name="Product",type="string",JSONPath=".status.hardware.product"
// +kubebuilder:printcolumn:name="Last Sync",type="date",JSONPath=".status.lastSyncTime"

// Machine is a read-only mirror of one physical machine in a bare metal
// provider's inventory. It is created, updated, and deleted exclusively by the
// platform's inventory sync; users cannot write it. Its name is
// "<providerRef>.<provider-side machine id>" (the NodeType naming convention):
// metal3 uses the BareMetalHost name, NICo the machine id.
// +k8s:openapi-gen=true
type Machine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="machine spec is read-only"
	Spec   MachineSpec   `json:"spec,omitempty"`
	Status MachineStatus `json:"status,omitempty"`
}

func (a *Machine) GetConditions() agentstoragev1.Conditions {
	return a.Status.Conditions
}

func (a *Machine) SetConditions(conditions agentstoragev1.Conditions) {
	a.Status.Conditions = conditions
}

// MachineSpec holds the mirror's identity. The machine id lives in the
// object name; the spec only duplicates the provider reference for indexing.
type MachineSpec struct {
	// DisplayName is shown in the UI; providers fill it from their native
	// naming (metal3: the BareMetalHost name, NICo: the machine hostname).
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// ProviderRef is the NodeProvider whose inventory this machine mirrors.
	ProviderRef string `json:"providerRef,omitempty"`
}

// MachineStatus is the observed provider-side state of the machine.
type MachineStatus struct {
	// Phase is the machine's state at the provider, normalized across providers.
	// It says nothing about platform NodeClaims: the node-claim annotation
	// answers which claim, if any, holds the machine.
	// +optional
	Phase MachinePhase `json:"phase,omitempty"`

	// Reason is the provider's own name for the state behind the phase, kept
	// verbatim: a metal3 provisioning state or errorType, a NICo machine status
	// or health alert id.
	// +optional
	Reason string `json:"reason,omitempty"`

	// Message is what the provider says about it, when it says anything: a
	// metal3 errorMessage, a NICo alert or maintenance message.
	// +optional
	Message string `json:"message,omitempty"`

	// Conditions holds the sync conditions of the machine.
	// +optional
	Conditions agentstoragev1.Conditions `json:"conditions,omitempty"`

	// Hardware is the machine's discovered hardware inventory.
	// +optional
	Hardware *MachineHardware `json:"hardware,omitempty"`

	// System is the machine's observed runtime state (assigned addressing and
	// similar), as opposed to the physical inventory in Hardware.
	// +optional
	System *MachineSystem `json:"system,omitempty"`

	// PowerState is the machine's normalized power state: On, Off, or Unknown
	// when the provider reports a state that maps to neither (e.g. an error).
	// Empty when the provider does not report power at all.
	// +optional
	PowerState MachinePowerState `json:"powerState,omitempty"`

	// LastSyncTime is when this state was last confirmed against the provider.
	// +optional
	LastSyncTime metav1.Time `json:"lastSyncTime,omitempty"`
}

// MachineSystem is the machine's observed runtime state.
type MachineSystem struct {
	// Network is the machine's network runtime state.
	// +optional
	Network *MachineNetwork `json:"network,omitempty"`
}

// MachineNetwork is the machine's network runtime state.
type MachineNetwork struct {
	// Interfaces lists the machine's network interfaces and their current
	// addressing.
	// +optional
	Interfaces []MachineNetworkInterface `json:"interfaces,omitempty"`
}

// MachineNetworkInterface is one network interface of a machine.
type MachineNetworkInterface struct {
	// Name is the interface name when the provider reports it (e.g. enp1s0).
	// +optional
	Name string `json:"name,omitempty"`

	// MAC is the interface's hardware address.
	// +optional
	MAC string `json:"mac,omitempty"`

	// Primary marks the interface the machine's traffic goes through (NICo:
	// the primary interface; metal3: the PXE/provisioning interface).
	// +optional
	Primary bool `json:"primary,omitempty"`

	// IPs are the addresses currently assigned to the interface; empty for a
	// machine waiting unprovisioned in the pool.
	// +optional
	IPs []string `json:"ips,omitempty"`
}

// MachineHardware is the normalized hardware inventory of a machine.
type MachineHardware struct {
	// CPU aggregates all CPU capabilities of the machine (sockets, cores and
	// threads summed across every CPU entry in Capabilities).
	// +optional
	CPU *CPUResource `json:"cpu,omitempty"`

	// Memory aggregates all Memory capabilities of the machine (capacity
	// summed across every Memory entry in Capabilities).
	// +optional
	Memory *resource.Quantity `json:"memory,omitempty"`

	// Storage aggregates all Storage capabilities of the machine (capacity
	// summed across every Storage entry in Capabilities).
	// +optional
	Storage *resource.Quantity `json:"storage,omitempty"`

	// Capabilities lists the machine's discovered devices, one entry per
	// device model.
	// +optional
	Capabilities []MachineCapability `json:"capabilities,omitempty"`

	// Vendor is the machine's hardware vendor when the provider reports it.
	// +optional
	Vendor string `json:"vendor,omitempty"`

	// Product is the machine's product name when the provider reports it.
	// +optional
	Product string `json:"product,omitempty"`

	// Serial is the machine's serial number when the provider reports it.
	// +optional
	Serial string `json:"serial,omitempty"`
}

// MachineCapabilityType is the kind of device a MachineCapability describes.
type MachineCapabilityType string

const (
	MachineCapabilityCPU        MachineCapabilityType = "CPU"
	MachineCapabilityMemory     MachineCapabilityType = "Memory"
	MachineCapabilityGPU        MachineCapabilityType = "GPU"
	MachineCapabilityDPU        MachineCapabilityType = "DPU"
	MachineCapabilityNetwork    MachineCapabilityType = "Network"
	MachineCapabilityStorage    MachineCapabilityType = "Storage"
	MachineCapabilityInfiniBand MachineCapabilityType = "InfiniBand"
)

// MachineCapability describes one device model present in a machine.
type MachineCapability struct {
	// Type is the device kind.
	Type MachineCapabilityType `json:"type"`

	// Name is the device model name.
	// +optional
	Name string `json:"name,omitempty"`

	// Vendor is the device vendor.
	// +optional
	Vendor string `json:"vendor,omitempty"`

	// Count is how many devices of this model the machine has.
	// +optional
	Count int64 `json:"count,omitempty"`

	// Resources holds the device resources this capability provides, per
	// device kind (CPU topology for CPU entries, capacity for the rest).
	// +optional
	Resources *Resources `json:"resources,omitempty"`
}

// Resources describes what one capability entry provides. Which fields are
// set depends on the capability type.
type Resources struct {
	// Capacity is the per-device capacity when the provider reports one
	// (Memory size, GPU memory, disk size).
	// +optional
	Capacity *resource.Quantity `json:"capacity,omitempty"`

	// CPU topology fields, inlined; only set on CPU capabilities.
	CPUResource `json:",inline"`
}

// CPUResource describes CPU topology. On a CPU capability entry it covers all
// packages of that entry's model together (sockets is the package count,
// cores/threads are totals across them); MachineHardware.CPU is the sum over
// all CPU entries. Providers fill what they report (metal3: architecture and
// logical CPU count as threads; NICo: packages with per-package cores, and
// threads when scout provides them); zero means "not reported".
type CPUResource struct {
	// Architecture is the CPU architecture (e.g. x86_64, aarch64).
	// +optional
	Architecture string `json:"architecture,omitempty"`

	// Sockets is the number of CPU packages.
	// +optional
	Sockets int64 `json:"sockets,omitempty"`

	// Cores is the number of physical cores.
	// +optional
	Cores int64 `json:"cores,omitempty"`

	// Threads is the number of logical CPUs.
	// +optional
	Threads int64 `json:"threads,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MachineList contains a list of Machine
type MachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Machine `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Machine{}, &MachineList{})
}
