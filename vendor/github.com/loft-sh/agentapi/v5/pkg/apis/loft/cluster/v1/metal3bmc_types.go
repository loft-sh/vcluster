package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Metal3BMC addresses the BMC of the metal3 BareMetalHost with the same name and
// namespace. It has no verbs of its own and only carries the redfish and console
// subresources, which the agent serves from inside the cluster that can reach the BMC.
// +k8s:openapi-gen=true
// +resource:path=metal3bmcs,rest=Metal3BMCREST
// +subresource:request=Metal3BMCRedfish,path=redfish,kind=Metal3BMCRedfish,rest=Metal3BMCRedfishREST
// +subresource:request=Metal3BMCConsole,path=console,kind=Metal3BMCConsole,rest=Metal3BMCConsoleREST
type Metal3BMC struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   Metal3BMCSpec   `json:"spec,omitempty"`
	Status Metal3BMCStatus `json:"status,omitempty"`
}

type Metal3BMCSpec struct {
}

type Metal3BMCStatus struct {
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Metal3BMCRedfish sends a single Redfish request to the BareMetalHost's BMC and
// returns the BMC's response. A BMC HTTP status is carried in Status.StatusCode;
// only resolution or transport failures surface as an API error.
// +subresource-request
type Metal3BMCRedfish struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   Metal3BMCRedfishSpec   `json:"spec,omitempty"`
	Status Metal3BMCRedfishStatus `json:"status,omitempty"`
}

// Metal3BMCRedfishSpec is the Redfish request to send to the BMC.
type Metal3BMCRedfishSpec struct {
	Method string `json:"method,omitempty"`
	Path   string `json:"path,omitempty"`
	Body   []byte `json:"body,omitempty"`
}

// Metal3BMCRedfishStatus is the BMC's response.
type Metal3BMCRedfishStatus struct {
	StatusCode int                 `json:"statusCode,omitempty"`
	Header     map[string][]string `json:"header,omitempty"`
	Body       []byte              `json:"body,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Metal3BMCConsole attaches a WebSocket (v4.channel.k8s.io) to the BareMetalHost's
// BMC serial console over SSH or IPMI Serial-over-LAN. It is connect-only; options
// are passed as Metal3BMCConsoleOptions query parameters.
// +subresource-request
type Metal3BMCConsole struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
}
