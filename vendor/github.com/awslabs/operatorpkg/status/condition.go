// Inspired by https://github.com/knative/pkg/tree/97c7258e3a98b81459936bc7a29dc6a9540fa357/apis,
// but we chose to diverge due to the unacceptably large dependency closure of knative/pkg.
package status

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Object interface {
	client.Object
	GetConditions() []Condition
	SetConditions([]Condition)
	StatusConditions() ConditionSet
}

// ConditionType is a upper-camel-cased condition type.
type ConditionType string

const (
	// ConditionReady specifies that the resource is ready.
	// For long-running resources.
	ConditionReady = "Ready"
	// ConditionSucceeded specifies that the resource has finished.
	// For resource which run to completion.
	ConditionSucceeded = "Succeeded"
)

// Condition aliases the upstream type and adds additional helper methods
type Condition metav1.Condition

func (c *Condition) IsTrue() bool {
	if c == nil {
		return false
	}
	return c.Status == metav1.ConditionTrue
}

func (c *Condition) IsFalse() bool {
	if c == nil {
		return false
	}
	return c.Status == metav1.ConditionFalse
}

func (c *Condition) IsUnknown() bool {
	if c == nil {
		return true
	}
	return c.Status == metav1.ConditionUnknown
}

func (c *Condition) GetStatus() metav1.ConditionStatus {
	if c == nil {
		return metav1.ConditionUnknown
	}
	return c.Status
}
