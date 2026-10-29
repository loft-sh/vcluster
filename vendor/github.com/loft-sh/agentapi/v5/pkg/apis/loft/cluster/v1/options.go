package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// +k8s:conversion-gen:explicit-from=net/url.Values
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Metal3BMCConsoleOptions struct {
	metav1.TypeMeta `json:",inline"`

	// Method is the BMC console transport: "ssh" or "ipmi" (Serial-over-LAN).
	Method string `json:"method,omitempty"`

	// Port on the BMC. Defaults to 22 for ssh and 623 for ipmi.
	// +optional
	Port int `json:"port,omitempty"`

	// Encryption activates the IPMI Serial-over-LAN payload with encryption and
	// authentication, negotiating an AES cipher suite. Only valid for ipmi.
	// +optional
	Encryption bool `json:"encryption,omitempty"`
}

func InstallOptions(scheme *runtime.Scheme) error {
	return addKnownOptionsTypes(scheme)
}

func addKnownOptionsTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(
		SchemeGroupVersion,
		&Metal3BMCConsoleOptions{},
	)
	return nil
}
