package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VirtualClusterStandalone holds kube config request and response data for virtual clusters
// +subresource-request
type VirtualClusterStandalone struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VirtualClusterStandaloneSpec   `json:"spec,omitempty"`
	Status VirtualClusterStandaloneStatus `json:"status,omitempty"`
}

type VirtualClusterStandaloneSpec struct {
	// CurrentPeer is the current peer that calls this API. The API will make sure this peer is added to the etcd peers list.
	// If this is the first peer, it will be the coordinator.
	CurrentPeer StandaloneEtcdPeer `json:"currentPeer"`

	// CurrentPKI contains certs bundle for vCluster
	CurrentPKI StandalonePKI `json:"currentPKI"`
}

type VirtualClusterStandaloneStatus struct {
	// ETCDPeers holds the comma separated list of etcd peers addresses.
	// It is used as a peer cache for vCluster Standalone deployed in HA mode via NodeProvider.
	ETCDPeers []StandaloneEtcdPeerCoordinator `json:"etcdPeers"`

	// PKI returns certs bundle for vCluster
	PKI StandalonePKI `json:"currentPKI"`
}

type StandaloneEtcdPeerCoordinator struct {
	StandaloneEtcdPeer `json:",inline"`

	// IsCoordinator is true if the peer is the coordinator.
	IsCoordinator bool `json:"isCoordinator"`
}

type StandaloneEtcdPeer struct {
	// Name is the name of the peer.
	Name string `json:"name"`

	// NodeClaim is the name of the node claim.
	NodeClaim string `json:"nodeClaim,omitempty"`

	// Address is the address of the peer.
	Address string `json:"address"`
}

// StandalonePKI is a map of certificates filenames and certs
type StandalonePKI struct {
	Certificates map[string][]byte `json:"certificates"`
}
