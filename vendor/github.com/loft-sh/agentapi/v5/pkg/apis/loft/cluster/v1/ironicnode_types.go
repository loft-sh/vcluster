package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// IronicNode is the Ironic node behind a metal3 BareMetalHost, read live from
// the Ironic API of this cluster on every request. It carries the same name
// and namespace as the BareMetalHost.
// +k8s:openapi-gen=true
// +resource:path=ironicnodes,rest=IronicNodeREST
// +subresource:request=IronicNodeAbort,path=abort,kind=IronicNodeAbort,rest=IronicNodeAbortREST
type IronicNode struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IronicNodeSpec   `json:"spec,omitempty"`
	Status IronicNodeStatus `json:"status,omitempty"`
}

// IronicNodeSpec is empty: an IronicNode is read-only.
type IronicNodeSpec struct {
}

// IronicNodeStatus is what Ironic reports about the node. Only the fields
// below are copied from Ironic, and driver info without its credentials, so
// BMC credentials, agent tokens and instance data stay in Ironic.
type IronicNodeStatus struct {
	// UUID is the Ironic node uuid, the BareMetalHost's status.provisioning.ID.
	UUID string `json:"uuid,omitempty"`

	// ProvisionState is the Ironic provision state, such as "wait call-back"
	// or "clean wait".
	ProvisionState string `json:"provisionState,omitempty"`

	// TargetProvisionState is the state Ironic is moving the node to.
	// +optional
	TargetProvisionState string `json:"targetProvisionState,omitempty"`

	// ProvisionUpdatedAt is when ProvisionState last changed.
	// +optional
	ProvisionUpdatedAt *metav1.Time `json:"provisionUpdatedAt,omitempty"`

	// PowerState is the power state Ironic last read from the BMC.
	// +optional
	PowerState string `json:"powerState,omitempty"`

	// TargetPowerState is the power state Ironic is moving the node to.
	// +optional
	TargetPowerState string `json:"targetPowerState,omitempty"`

	// LastError is the last error Ironic recorded for the node.
	// +optional
	LastError string `json:"lastError,omitempty"`

	// Maintenance is set when the node is in maintenance mode, by an operator
	// or by Ironic itself after a fault.
	// +optional
	Maintenance bool `json:"maintenance,omitempty"`

	// MaintenanceReason says why the node is in maintenance mode.
	// +optional
	MaintenanceReason string `json:"maintenanceReason,omitempty"`

	// Fault is the fault that made Ironic put the node in maintenance mode,
	// such as "power failure" or "clean failure".
	// +optional
	Fault string `json:"fault,omitempty"`

	// Conductor is the Ironic conductor that manages the node, and so the one
	// whose logs cover it.
	// +optional
	Conductor string `json:"conductor,omitempty"`

	// Reservation is the conductor that currently holds the node's lock.
	// +optional
	Reservation string `json:"reservation,omitempty"`

	// DeployStep is the deploy step the node is running or waiting in.
	// +optional
	DeployStep *IronicNodeStep `json:"deployStep,omitempty"`

	// CleanStep is the clean step the node is running or waiting in.
	// +optional
	CleanStep *IronicNodeStep `json:"cleanStep,omitempty"`

	// ServiceStep is the service step the node is running or waiting in.
	// +optional
	ServiceStep *IronicNodeStep `json:"serviceStep,omitempty"`

	// Agent is what Ironic knows about the ironic-python-agent ramdisk.
	// +optional
	Agent IronicNodeAgent `json:"agent,omitempty"`

	// Driver is the hardware type Ironic manages the node with, such as
	// "redfish" or "ipmi".
	// +optional
	Driver string `json:"driver,omitempty"`

	// Interfaces are the driver interfaces the node uses.
	// +optional
	Interfaces IronicNodeInterfaces `json:"interfaces,omitempty"`

	// DriverInfo holds the node's driver settings, such as the BMC address,
	// the deploy ramdisk or force_persistent_boot_device. A value that is not
	// a string is JSON. Credentials are left out: whatever Ironic masks, and
	// every username, password, token, key, secret or certificate.
	// +optional
	DriverInfo map[string]string `json:"driverInfo,omitempty"`

	// BootMode is the boot mode Ironic last read from the BMC, "uefi" or
	// "bios".
	// +optional
	BootMode string `json:"bootMode,omitempty"`

	// SecureBoot is whether UEFI secure boot is on, as Ironic last read it
	// from the BMC. Unset when the BMC does not report it.
	// +optional
	SecureBoot *bool `json:"secureBoot,omitempty"`

	// Abortable is true when the node waits in a state that the abort
	// subresource accepts.
	// +optional
	Abortable bool `json:"abortable,omitempty"`

	// History holds the node's most recent Ironic history events, newest first.
	// Only a get fills it: a list leaves it empty.
	// +optional
	History []IronicNodeHistoryEvent `json:"history,omitempty"`
}

// IronicNodeInterfaces are the driver interfaces of an Ironic node, each the
// implementation Ironic uses for it, such as "redfish-virtual-media" for boot.
type IronicNodeInterfaces struct {
	// +optional
	Boot string `json:"boot,omitempty"`
	// +optional
	Deploy string `json:"deploy,omitempty"`
	// +optional
	Power string `json:"power,omitempty"`
	// +optional
	Management string `json:"management,omitempty"`
	// +optional
	Inspect string `json:"inspect,omitempty"`
	// +optional
	Console string `json:"console,omitempty"`
	// +optional
	Network string `json:"network,omitempty"`
	// +optional
	Storage string `json:"storage,omitempty"`
	// +optional
	RAID string `json:"raid,omitempty"`
	// +optional
	BIOS string `json:"bios,omitempty"`
	// +optional
	Firmware string `json:"firmware,omitempty"`
	// +optional
	Rescue string `json:"rescue,omitempty"`
	// +optional
	Vendor string `json:"vendor,omitempty"`
}

// IronicNodeStep is a deploy, clean or service step of an Ironic node.
type IronicNodeStep struct {
	// DriverInterface is the driver interface the step belongs to, such as
	// "deploy" or "raid".
	DriverInterface string `json:"driverInterface,omitempty"`

	// Step is the step name, such as "write_image".
	Step string `json:"step,omitempty"`

	// Abortable is true when Ironic can abort the step while it runs. An abort
	// requested during a step that is not abortable takes effect once the
	// step is done.
	// +optional
	Abortable bool `json:"abortable,omitempty"`
}

// IronicNodeAgent is what Ironic knows about the ramdisk agent of a node.
type IronicNodeAgent struct {
	// Started is true once the agent has looked up its node and been given a
	// token, so the ramdisk booted and reached Ironic.
	// +optional
	Started bool `json:"started,omitempty"`

	// LastHeartbeat is when the agent last called back to Ironic.
	// +optional
	LastHeartbeat *metav1.Time `json:"lastHeartbeat,omitempty"`

	// Version is the agent version.
	// +optional
	Version string `json:"version,omitempty"`
}

// IronicNodeHistoryEvent is one entry of an Ironic node's history.
type IronicNodeHistoryEvent struct {
	// CreatedAt is when Ironic recorded the event.
	CreatedAt metav1.Time `json:"createdAt"`

	// Severity is the event severity, such as "ERROR" or "INFO".
	// +optional
	Severity string `json:"severity,omitempty"`

	// EventType is the operation the event belongs to, such as "deployment"
	// or "cleaning".
	// +optional
	EventType string `json:"eventType,omitempty"`

	// Event is the event message.
	// +optional
	Event string `json:"event,omitempty"`

	// Conductor is the conductor that recorded the event.
	// +optional
	Conductor string `json:"conductor,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// IronicNodeAbort aborts the operation an Ironic node is waiting in. Ironic
// then fails the operation and the bare metal operator retries it, the same
// way it does when Ironic's own callback timeout expires.
// +subresource-request
type IronicNodeAbort struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Status is the node as Ironic reports it right after the abort request.
	// +optional
	Status IronicNodeStatus `json:"status,omitempty"`
}
