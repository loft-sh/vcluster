/*
Copyright The Kubernetes Authors.

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

<<<<<<<< HEAD:vendor/k8s.io/apimachinery/pkg/sharding/hash.go
package sharding

import (
	"fmt"
	"hash/fnv"
)

// HashField computes a hash of value and returns it
// as a 16-character lowercase hex string (no "0x" prefix).
func HashField(value string) string {
	h := fnv.New64a()
	h.Write([]byte(value))
	return fmt.Sprintf("%016x", h.Sum64())
========
package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "sigs.k8s.io/gateway-api/apis/v1"
)

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:skipversion
// +kubebuilder:deprecatedversion:warning="The v1alpha2 version of GRPCRoute has been deprecated and will be removed in a future release of the API. Please upgrade to v1."
type GRPCRoute v1.GRPCRoute

// +kubebuilder:object:root=true
type GRPCRouteList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GRPCRoute `json:"items"`
>>>>>>>> f68923b19 (Adds Gateway syncer and unit tests):vendor/sigs.k8s.io/gateway-api/apis/v1alpha2/grpcroute_types.go
}
