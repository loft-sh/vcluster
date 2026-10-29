package v1

import (
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NetworkPeer holds the network peer for vCluster.
// +k8s:openapi-gen=true
// +resource:path=networkpeers,rest=NetworkPeerREST
// +subresource:request=NetworkPeerDebug,path=debug,kind=NetworkPeerDebug,rest=NetworkPeerDebugREST
type NetworkPeer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NetworkPeerSpec   `json:"spec,omitempty"`
	Status NetworkPeerStatus `json:"status,omitempty"`
}

// NetworkPeerSpec defines spec of network peer.
type NetworkPeerSpec struct {
	storagev1.NetworkPeerSpec `json:",inline"`
}

type NetworkPeerStatus struct {
	storagev1.NetworkPeerStatus `json:",inline"`
}
