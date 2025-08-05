/*
Copyright 2023 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
// +kubebuilder:object:generate=true
package v1beta1

import (
	core_v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	snapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v8/apis/volumesnapshot/v1"
)

// VolumeGroupSnapshotSpec defines the desired state of a volume group snapshot.
type VolumeGroupSnapshotSpec struct {
	// Source specifies where a group snapshot will be created from.
	// This field is immutable after creation.
	// Required.
	Source VolumeGroupSnapshotSource `json:"source" protobuf:"bytes,1,opt,name=source"`

	// VolumeGroupSnapshotClassName is the name of the VolumeGroupSnapshotClass
	// requested by the VolumeGroupSnapshot.
	// VolumeGroupSnapshotClassName may be left nil to indicate that the default
	// class will be used.
	// Empty string is not allowed for this field.
	// +optional
	// +kubebuilder:validation:XValidation:rule="size(self) > 0",message="volumeGroupSnapshotClassName must not be the empty string when set"
	VolumeGroupSnapshotClassName *string `json:"volumeGroupSnapshotClassName,omitempty" protobuf:"bytes,2,opt,name=volumeGroupSnapshotClassName"`
}

// VolumeGroupSnapshotSource specifies whether the underlying group snapshot should be
// dynamically taken upon creation or if a pre-existing VolumeGroupSnapshotContent
// object should be used.
// Exactly one of its members must be set.
// Members in VolumeGroupSnapshotSource are immutable.
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.selector) || has(self.selector)", message="selector is required once set"
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.volumeGroupSnapshotContentName) || has(self.volumeGroupSnapshotContentName)", message="volumeGroupSnapshotContentName is required once set"
// +kubebuilder:validation:XValidation:rule="(has(self.selector) && !has(self.volumeGroupSnapshotContentName)) || (!has(self.selector) && has(self.volumeGroupSnapshotContentName))", message="exactly one of selector and volumeGroupSnapshotContentName must be set"
type VolumeGroupSnapshotSource struct {
	// Selector is a label query over persistent volume claims that are to be
	// grouped together for snapshotting.
	// This labelSelector will be used to match the label added to a PVC.
	// If the label is added or removed to a volume after a group snapshot
	// is created, the existing group snapshots won't be modified.
	// Once a VolumeGroupSnapshotContent is created and the sidecar starts to process
	// it, the volume list will not change with retries.
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="selector is immutable"
	Selector *metav1.LabelSelector `json:"selector,omitempty" protobuf:"bytes,1,opt,name=selector"`

	// VolumeGroupSnapshotContentName specifies the name of a pre-existing VolumeGroupSnapshotContent
	// object representing an existing volume group snapshot.
	// This field should be set if the volume group snapshot already exists and
	// only needs a representation in Kubernetes.
	// This field is immutable.
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="volumeGroupSnapshotContentName is immutable"
	VolumeGroupSnapshotContentName *string `json:"volumeGroupSnapshotContentName,omitempty" protobuf:"bytes,2,opt,name=volumeGroupSnapshotContentName"`
}

// VolumeGroupSnapshotStatus defines the observed state of volume group snapshot.
type VolumeGroupSnapshotStatus struct {
	// BoundVolumeGroupSnapshotContentName is the name of the VolumeGroupSnapshotContent
	// object to which this VolumeGroupSnapshot object intends to bind to.
	// If not specified, it indicates that the VolumeGroupSnapshot object has not
	// been successfully bound to a VolumeGroupSnapshotContent object yet.
	// NOTE: To avoid possible security issues, consumers must verify binding between
	// VolumeGroupSnapshot and VolumeGroupSnapshotContent objects is successful
	// (by validating that both VolumeGroupSnapshot and VolumeGroupSnapshotContent
	// point at each other) before using this object.
	// +optional
	BoundVolumeGroupSnapshotContentName *string `json:"boundVolumeGroupSnapshotContentName,omitempty" protobuf:"bytes,1,opt,name=boundVolumeGroupSnapshotContentName"`

	// CreationTime is the timestamp when the point-in-time group snapshot is taken
	// by the underlying storage system.
	// If not specified, it may indicate that the creation time of the group snapshot
	// is unknown.
	// The format of this field is a Unix nanoseconds time encoded as an int64.
	// On Unix, the command date +%s%N returns the current time in nanoseconds
	// since 1970-01-01 00:00:00 UTC.
	// This field is updated based on the CreationTime field in VolumeGroupSnapshotContentStatus
	// +optional
	CreationTime *metav1.Time `json:"creationTime,omitempty" protobuf:"bytes,2,opt,name=creationTime"`

	// ReadyToUse indicates if all the individual snapshots in the group are ready
	// to be used to restore a group of volumes.
	// ReadyToUse becomes true when ReadyToUse of all individual snapshots become true.
	// If not specified, it means the readiness of a group snapshot is unknown.
	// +optional
	ReadyToUse *bool `json:"readyToUse,omitempty" protobuf:"varint,3,opt,name=readyToUse"`

	// Error is the last observed error during group snapshot creation, if any.
	// This field could be helpful to upper level controllers (i.e., application
	// controller) to decide whether they should continue on waiting for the group
	// snapshot to be created based on the type of error reported.
	// The snapshot controller will keep retrying when an error occurs during the
	// group snapshot creation. Upon success, this error field will be cleared.
	// +optional
	Error *snapshotv1.VolumeSnapshotError `json:"error,omitempty" protobuf:"bytes,4,opt,name=error,casttype=VolumeSnapshotError"`
}

//+genclient
//+k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VolumeGroupSnapshot is a user's request for creating either a point-in-time
// group snapshot or binding to a pre-existing group snapshot.
// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Namespaced,shortName=vgs
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="ReadyToUse",type=boolean,JSONPath=`.status.readyToUse`,description="Indicates if all the individual snapshots in the group are ready to be used to restore a group of volumes."
// +kubebuilder:printcolumn:name="VolumeGroupSnapshotClass",type=string,JSONPath=`.spec.volumeGroupSnapshotClassName`,description="The name of the VolumeGroupSnapshotClass requested by the VolumeGroupSnapshot."
// +kubebuilder:printcolumn:name="VolumeGroupSnapshotContent",type=string,JSONPath=`.status.boundVolumeGroupSnapshotContentName`,description="Name of the VolumeGroupSnapshotContent object to which the VolumeGroupSnapshot object intends to bind to. Please note that verification of binding actually requires checking both VolumeGroupSnapshot and VolumeGroupSnapshotContent to ensure both are pointing at each other. Binding MUST be verified prior to usage of this object."
// +kubebuilder:printcolumn:name="CreationTime",type=date,JSONPath=`.status.creationTime`,description="Timestamp when the point-in-time group snapshot was taken by the underlying storage system."
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type VolumeGroupSnapshot struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Spec defines the desired characteristics of a group snapshot requested by a user.
	// Required.
	Spec VolumeGroupSnapshotSpec `json:"spec" protobuf:"bytes,2,opt,name=spec"`
	// Status represents the current information of a group snapshot.
	// Consumers must verify binding between VolumeGroupSnapshot and
	// VolumeGroupSnapshotContent objects is successful (by validating that both
	// VolumeGroupSnapshot and VolumeGroupSnapshotContent point to each other) before
	// using this object.
	// +optional
	Status *VolumeGroupSnapshotStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// VolumeGroupSnapshotList contains a list of VolumeGroupSnapshot objects.
type VolumeGroupSnapshotList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	// Items is the list of VolumeGroupSnapshots.
	Items []VolumeGroupSnapshot `json:"items" protobuf:"bytes,2,rep,name=items"`
}

//+genclient
//+genclient:nonNamespaced
//+k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VolumeGroupSnapshotClass specifies parameters that a underlying storage system
// uses when creating a volume group snapshot. A specific VolumeGroupSnapshotClass
// is used by specifying its name in a VolumeGroupSnapshot object.
// VolumeGroupSnapshotClasses are non-namespaced.
// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster,shortName=vgsclass;vgsclasses
// +kubebuilder:printcolumn:name="Driver",type=string,JSONPath=`.driver`
// +kubebuilder:printcolumn:name="DeletionPolicy",type=string,JSONPath=`.deletionPolicy`,description="Determines whether a VolumeGroupSnapshotContent created through the VolumeGroupSnapshotClass should be deleted when its bound VolumeGroupSnapshot is deleted."
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type VolumeGroupSnapshotClass struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Driver is the name of the storage driver expected to handle this VolumeGroupSnapshotClass.
	// Required.
	Driver string `json:"driver" protobuf:"bytes,2,opt,name=driver"`

	// Parameters is a key-value map with storage driver specific parameters for
	// creating group snapshots.
	// These values are opaque to Kubernetes and are passed directly to the driver.
	// +optional
	Parameters map[string]string `json:"parameters,omitempty" protobuf:"bytes,3,rep,name=parameters"`

	// DeletionPolicy determines whether a VolumeGroupSnapshotContent created
	// through the VolumeGroupSnapshotClass should be deleted when its bound
	// VolumeGroupSnapshot is deleted.
	// Supported values are "Retain" and "Delete".
	// "Retain" means that the VolumeGroupSnapshotContent and its physical group
	// snapshot on underlying storage system are kept.
	// "Delete" means that the VolumeGroupSnapshotContent and its physical group
	// snapshot on underlying storage system are deleted.
	// Required.
	DeletionPolicy snapshotv1.DeletionPolicy `json:"deletionPolicy" protobuf:"bytes,4,opt,name=deletionPolicy"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VolumeGroupSnapshotClassList is a collection of VolumeGroupSnapshotClasses.
// +kubebuilder:object:root=true
type VolumeGroupSnapshotClassList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	// +optional
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Items is the list of VolumeGroupSnapshotClasses.
	Items []VolumeGroupSnapshotClass `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VolumeGroupSnapshotContent represents the actual "on-disk" group snapshot object
// in the underlying storage system
// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster,shortName=vgsc;vgscs
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="ReadyToUse",type=boolean,JSONPath=`.status.readyToUse`,description="Indicates if all the individual snapshots in the group are ready to be used to restore a group of volumes."
// +kubebuilder:printcolumn:name="DeletionPolicy",type=string,JSONPath=`.spec.deletionPolicy`,description="Determines whether this VolumeGroupSnapshotContent and its physical group snapshot on the underlying storage system should be deleted when its bound VolumeGroupSnapshot is deleted."
// +kubebuilder:printcolumn:name="Driver",type=string,JSONPath=`.spec.driver`,description="Name of the CSI driver used to create the physical group snapshot on the underlying storage system."
// +kubebuilder:printcolumn:name="VolumeGroupSnapshotClass",type=string,JSONPath=`.spec.volumeGroupSnapshotClassName`,description="Name of the VolumeGroupSnapshotClass from which this group snapshot was (or will be) created."
// +kubebuilder:printcolumn:name="VolumeGroupSnapshotNamespace",type=string,JSONPath=`.spec.volumeGroupSnapshotRef.namespace`,description="Namespace of the VolumeGroupSnapshot object to which this VolumeGroupSnapshotContent object is bound."
// +kubebuilder:printcolumn:name="VolumeGroupSnapshot",type=string,JSONPath=`.spec.volumeGroupSnapshotRef.name`,description="Name of the VolumeGroupSnapshot object to which this VolumeGroupSnapshotContent object is bound."
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type VolumeGroupSnapshotContent struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Spec defines properties of a VolumeGroupSnapshotContent created by the underlying storage system.
	// Required.
	Spec VolumeGroupSnapshotContentSpec `json:"spec" protobuf:"bytes,2,opt,name=spec"`
	// status represents the current information of a group snapshot.
	// +optional
	Status *VolumeGroupSnapshotContentStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VolumeGroupSnapshotContentList is a list of VolumeGroupSnapshotContent objects
// +kubebuilder:object:root=true
type VolumeGroupSnapshotContentList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	// +optional
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Items is the list of VolumeGroupSnapshotContents.
	Items []VolumeGroupSnapshotContent `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// VolumeGroupSnapshotContentSpec describes the common attributes of a group snapshot content
type VolumeGroupSnapshotContentSpec struct {
	// VolumeGroupSnapshotRef specifies the VolumeGroupSnapshot object to which this
	// VolumeGroupSnapshotContent object is bound.
	// VolumeGroupSnapshot.Spec.VolumeGroupSnapshotContentName field must reference to
	// this VolumeGroupSnapshotContent's name for the bidirectional binding to be valid.
	// For a pre-existing VolumeGroupSnapshotContent object, name and namespace of the
	// VolumeGroupSnapshot object MUST be provided for binding to happen.
	// This field is immutable after creation.
	// Required.
	// +kubebuilder:validation:XValidation:rule="has(self.name) && has(self.__namespace__)",message="both volumeGroupSnapshotRef.name and volumeGroupSnapshotRef.namespace must be set"
	VolumeGroupSnapshotRef core_v1.ObjectReference `json:"volumeGroupSnapshotRef" protobuf:"bytes,1,opt,name=volumeGroupSnapshotRef"`

	// DeletionPolicy determines whether this VolumeGroupSnapshotContent and the
	// physical group snapshot on the underlying storage system should be deleted
	// when the bound VolumeGroupSnapshot is deleted.
	// Supported values are "Retain" and "Delete".
	// "Retain" means that the VolumeGroupSnapshotContent and its physical group
	// snapshot on underlying storage system are kept.
	// "Delete" means that the VolumeGroupSnapshotContent and its physical group
	// snapshot on underlying storage system are deleted.
	// For dynamically provisioned group snapshots, this field will automatically
	// be filled in by the CSI snapshotter sidecar with the "DeletionPolicy" field
	// defined in the corresponding VolumeGroupSnapshotClass.
	// For pre-existing snapshots, users MUST specify this field when creating the
	// VolumeGroupSnapshotContent object.
	// Required.
	DeletionPolicy snapshotv1.DeletionPolicy `json:"deletionPolicy" protobuf:"bytes,2,opt,name=deletionPolicy"`

	// Driver is the name of the CSI driver used to create the physical group snapshot on
	// the underlying storage system.
	// This MUST be the same as the name returned by the CSI GetPluginName() call for
	// that driver.
	// Required.
	Driver string `json:"driver" protobuf:"bytes,3,opt,name=driver"`

	// VolumeGroupSnapshotClassName is the name of the VolumeGroupSnapshotClass from
	// which this group snapshot was (or will be) created.
	// Note that after provisioning, the VolumeGroupSnapshotClass may be deleted or
	// recreated with different set of values, and as such, should not be referenced
	// post-snapshot creation.
	// For dynamic provisioning, this field must be set.
	// This field may be unset for pre-provisioned snapshots.
	// +optional
	VolumeGroupSnapshotClassName *string `json:"volumeGroupSnapshotClassName,omitempty" protobuf:"bytes,4,opt,name=volumeGroupSnapshotClassName"`

	// Source specifies whether the snapshot is (or should be) dynamically provisioned
	// or already exists, and just requires a Kubernetes object representation.
	// This field is immutable after creation.
	// Required.
	Source VolumeGroupSnapshotContentSource `json:"source" protobuf:"bytes,5,opt,name=source"`
}

// VolumeGroupSnapshotContentStatus defines the observed state of VolumeGroupSnapshotContent.
type VolumeGroupSnapshotContentStatus struct {
	// VolumeGroupSnapshotHandle is a unique id returned by the CSI driver
	// to identify the VolumeGroupSnapshot on the storage system.
	// If a storage system does not provide such an id, the
	// CSI driver can choose to return the VolumeGroupSnapshot name.
	// +optional
	VolumeGroupSnapshotHandle *string `json:"volumeGroupSnapshotHandle,omitempty" protobuf:"bytes,1,opt,name=volumeGroupSnapshotHandle"`

	// CreationTime is the timestamp when the point-in-time group snapshot is taken
	// by the underlying storage system.
	// If not specified, it indicates the creation time is unknown.
	// If not specified, it means the readiness of a group snapshot is unknown.
	// The format of this field is a Unix nanoseconds time encoded as an int64.
	// On Unix, the command date +%s%N returns the current time in nanoseconds
	// since 1970-01-01 00:00:00 UTC.
	// This field is the source for the CreationTime field in VolumeGroupSnapshotStatus
	// +optional
	CreationTime *metav1.Time `json:"creationTime,omitempty" protobuf:"bytes,2,opt,name=creationTime"`

	// ReadyToUse indicates if all the individual snapshots in the group are ready to be
	// used to restore a group of volumes.
	// ReadyToUse becomes true when ReadyToUse of all individual snapshots become true.
	// +optional
	ReadyToUse *bool `json:"readyToUse,omitempty" protobuf:"varint,3,opt,name=readyToUse"`

	// Error is the last observed error during group snapshot creation, if any.
	// Upon success after retry, this error field will be cleared.
	// +optional
	Error *snapshotv1.VolumeSnapshotError `json:"error,omitempty" protobuf:"bytes,4,opt,name=error,casttype=VolumeSnapshotError"`

	// VolumeSnapshotHandlePairList is a list of CSI "volume_id" and "snapshot_id"
	// pair returned by the CSI driver to identify snapshots and their source volumes
	// on the storage system.
	// +optional
	VolumeSnapshotHandlePairList []VolumeSnapshotHandlePair `json:"volumeSnapshotHandlePairList,omitempty" protobuf:"bytes,6,opt,name=volumeSnapshotHandlePairList"`
}

// VolumeGroupSnapshotContentSource represents the CSI source of a group snapshot.
// Exactly one of its members must be set.
// Members in VolumeGroupSnapshotContentSource are immutable.
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.volumeHandles) || has(self.volumeHandles)", message="volumeHandles is required once set"
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.groupSnapshotHandles) || has(self.groupSnapshotHandles)", message="groupSnapshotHandles is required once set"
// +kubebuilder:validation:XValidation:rule="(has(self.volumeHandles) && !has(self.groupSnapshotHandles)) || (!has(self.volumeHandles) && has(self.groupSnapshotHandles))", message="exactly one of volumeHandles and groupSnapshotHandles must be set"
type VolumeGroupSnapshotContentSource struct {
	// VolumeHandles is a list of volume handles on the backend to be snapshotted
	// together. It is specified for dynamic provisioning of the VolumeGroupSnapshot.
	// This field is immutable.
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="volumeHandles is immutable"
	VolumeHandles []string `json:"volumeHandles,omitempty" protobuf:"bytes,1,opt,name=volumeHandles"`

	// GroupSnapshotHandles specifies the CSI "group_snapshot_id" of a pre-existing
	// group snapshot and a list of CSI "snapshot_id" of pre-existing snapshots
	// on the underlying storage system for which a Kubernetes object
	// representation was (or should be) created.
	// This field is immutable.
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="groupSnapshotHandles is immutable"
	GroupSnapshotHandles *GroupSnapshotHandles `json:"groupSnapshotHandles,omitempty" protobuf:"bytes,2,opt,name=groupSnapshotHandles"`
}

type GroupSnapshotHandles struct {
	// VolumeGroupSnapshotHandle specifies the CSI "group_snapshot_id" of a pre-existing
	// group snapshot on the underlying storage system for which a Kubernetes object
	// representation was (or should be) created.
	// This field is immutable.
	// Required.
	VolumeGroupSnapshotHandle string `json:"volumeGroupSnapshotHandle" protobuf:"bytes,1,opt,name=volumeGroupSnapshotHandle"`

	// VolumeSnapshotHandles is a list of CSI "snapshot_id" of pre-existing
	// snapshots on the underlying storage system for which Kubernetes objects
	// representation were (or should be) created.
	// This field is immutable.
	// Required.
	VolumeSnapshotHandles []string `json:"volumeSnapshotHandles" protobuf:"bytes,2,opt,name=volumeSnapshotHandles"`
}

// VolumeSnapshotHandlePair defines a pair of a source volume handle and a snapshot handle
type VolumeSnapshotHandlePair struct {
	// VolumeHandle is a unique id returned by the CSI driver to identify a volume
	// on the storage system
	// Required.
	VolumeHandle string `json:"volumeHandle" protobuf:"bytes,1,opt,name=volumeHandle"`

	// SnapshotHandle is a unique id returned by the CSI driver to identify a volume
	// snapshot on the storage system
	// Required.
	SnapshotHandle string `json:"snapshotHandle" protobuf:"bytes,2,opt,name=snapshotHandle"`
}
