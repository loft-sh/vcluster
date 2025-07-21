// Inspired by https://github.com/knative/pkg/tree/97c7258e3a98b81459936bc7a29dc6a9540fa357/apis,
// but we chose to diverge due to the unacceptably large dependency closure of knative/pkg.
package status

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/samber/lo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ConditionTypes is an abstract collection of the possible ConditionType values
// that a particular resource might expose.  It also holds the "root condition"
// for that resource, which we define to be one of Ready or Succeeded depending
// on whether it is a Living or Batch process respectively.
type ConditionTypes struct {
	root       string
	dependents []string
}

// NewReadyConditions returns a ConditionTypes to hold the conditions for the
// resource. ConditionReady is used as the root condition.
// The set of condition types provided are those of the terminal subconditions.
func NewReadyConditions(d ...string) ConditionTypes {
	return newConditionTypes(ConditionReady, d...)
}

// NewSucceededConditions returns a ConditionTypes to hold the conditions for the
// batch resource. ConditionSucceeded is used as the root condition.
// The set of condition types provided are those of the terminal subconditions.
func NewSucceededConditions(d ...string) ConditionTypes {
	return newConditionTypes(ConditionSucceeded, d...)
}

func newConditionTypes(root string, dependents ...string) ConditionTypes {
	return ConditionTypes{
		root:       root,
		dependents: lo.Reject(lo.Uniq(dependents), func(c string, _ int) bool { return c == root }),
	}
}

// ConditionSet provides methods for evaluating Conditions.
// +k8s:deepcopy-gen=false
type ConditionSet struct {
	ConditionTypes
	object Object
}

// For creates a ConditionSet from an object using the original
// ConditionTypes as a reference. Status must be a pointer to a struct.
func (r ConditionTypes) For(object Object) ConditionSet {
	cs := ConditionSet{object: object, ConditionTypes: r}
	// Set known conditions Unknown if not set.
	// Set the root condition first to get consistent timing for LastTransitionTime
	for _, t := range append([]string{r.root}, r.dependents...) {
		if cs.Get(t) == nil {
			cs.SetUnknown(t)
		}
	}
	return cs
}

// Root returns the root Condition, typically "Ready" or "Succeeded"
func (c ConditionSet) Root() *Condition {
	if c.object == nil {
		return nil
	}
	return c.Get(c.root)
}

func (c ConditionSet) List() []Condition {
	if c.object == nil {
		return nil
	}
	return c.object.GetConditions()
}

// Get finds and returns the Condition that matches the ConditionType
// previously set on Conditions.
func (c ConditionSet) Get(t string) *Condition {
	if c.object == nil {
		return nil
	}
	if condition, found := lo.Find(c.object.GetConditions(), func(c Condition) bool { return c.Type == t }); found {
		return &condition
	}
	return nil
}

// IsTrue returns true if all condition types are true.
func (c ConditionSet) IsTrue(conditionTypes ...string) bool {
	for _, conditionType := range conditionTypes {
		if !c.Get(conditionType).IsTrue() {
			return false
		}
	}
	return true
}

func (c ConditionSet) IsDependentCondition(t string) bool {
	return t == c.root || lo.Contains(c.dependents, t)
}

// Set sets or updates the Condition on Conditions for Condition.Type.
// If there is an update, Conditions are stored back sorted.
func (c ConditionSet) Set(condition Condition) (modified bool) {
	var conditions []Condition
	var foundCondition bool

	condition.ObservedGeneration = c.object.GetGeneration()
	for _, cond := range c.object.GetConditions() {
		if cond.Type != condition.Type {
			// If we are deleting, we just bump all the observed generations
			if !c.object.GetDeletionTimestamp().IsZero() {
				cond.ObservedGeneration = c.object.GetGeneration()
			}
			conditions = append(conditions, cond)
		} else {
			foundCondition = true
			if condition.Status == cond.Status {
				condition.LastTransitionTime = cond.LastTransitionTime
			} else {
				condition.LastTransitionTime = metav1.Now()
			}
			if reflect.DeepEqual(condition, cond) {
				return false
			}
		}
	}
	if !foundCondition {
		// Dependent conditions should always be set, so if it's not found, that means
		// that we are initializing the condition type, and it's last "transition" was object creation
		if c.IsDependentCondition(condition.Type) {
			condition.LastTransitionTime = c.object.GetCreationTimestamp()
		} else {
			condition.LastTransitionTime = metav1.Now()
		}
	}
	conditions = append(conditions, condition)
	// Sorted for convenience of the consumer, i.e. kubectl.
	sort.SliceStable(conditions, func(i, j int) bool {
		// Order the root status condition at the end
		if conditions[i].Type == c.root || conditions[j].Type == c.root {
			return conditions[j].Type == c.root
		}
		return conditions[i].LastTransitionTime.Time.Before(conditions[j].LastTransitionTime.Time)
	})
	c.object.SetConditions(conditions)

	// Recompute the root condition after setting any other condition
	c.recomputeRootCondition(condition.Type)
	return true
}

// Clear removes the independent condition that matches the ConditionType
// Not implemented for dependent conditions
func (c ConditionSet) Clear(t string) error {
	var conditions []Condition

	if c.object == nil {
		return nil
	}
	// Dependent conditions are not handled as they can't be nil
	if c.IsDependentCondition(t) {
		return fmt.Errorf("clearing dependent conditions not implemented")
	}
	cond := c.Get(t)
	if cond == nil {
		return nil
	}
	for _, c := range c.object.GetConditions() {
		if c.Type != t {
			conditions = append(conditions, c)
		}
	}

	// Sorted for convenience of the consumer, i.e. kubectl.
	sort.Slice(conditions, func(i, j int) bool { return conditions[i].Type < conditions[j].Type })
	c.object.SetConditions(conditions)

	return nil
}

// SetTrue sets the status of conditionType to true with the reason, and then marks the root condition to
// true if all other dependents are also true.
func (c ConditionSet) SetTrue(conditionType string) (modified bool) {
	return c.SetTrueWithReason(conditionType, conditionType, "")
}

// SetTrueWithReason sets the status of conditionType to true with the reason, and then marks the root condition to
// true if all other dependents are also true.
func (c ConditionSet) SetTrueWithReason(conditionType string, reason, message string) (modified bool) {
	return c.Set(Condition{
		Type:    conditionType,
		Status:  metav1.ConditionTrue,
		Reason:  reason,
		Message: message,
	})
}

// SetUnknown sets the status of conditionType to Unknown and also sets the root condition
// to Unknown if no other dependent condition is in an error state.
func (c ConditionSet) SetUnknown(conditionType string) (modified bool) {
	return c.SetUnknownWithReason(conditionType, "AwaitingReconciliation", "object is awaiting reconciliation")
}

// SetUnknownWithReason sets the status of conditionType to Unknown with the reason, and also sets the root condition
// to Unknown if no other dependent condition is in an error state.
func (c ConditionSet) SetUnknownWithReason(conditionType string, reason, message string) (modified bool) {
	return c.Set(Condition{
		Type:    conditionType,
		Status:  metav1.ConditionUnknown,
		Reason:  reason,
		Message: message,
	})
}

// SetFalse sets the status of conditionType and the root condition to False.
func (c ConditionSet) SetFalse(conditionType string, reason, message string) (modified bool) {
	return c.Set(Condition{
		Type:    conditionType,
		Status:  metav1.ConditionFalse,
		Reason:  reason,
		Message: message,
	})
}

// recomputeRootCondition marks the root condition to true if all other dependents are also true.
func (c ConditionSet) recomputeRootCondition(conditionType string) {
	if conditionType == c.root {
		return
	}
	if conditions := c.findUnhealthyDependents(); len(conditions) == 0 {
		c.SetTrue(c.root)
	} else {
		// The root condition is no longer unknown as soon as any dependent condition goes false with the latest observedGeneration
		status := lo.Ternary(
			lo.ContainsBy(conditions, func(condition Condition) bool {
				return condition.IsFalse() &&
					condition.ObservedGeneration == c.object.GetGeneration()
			}),
			metav1.ConditionFalse,
			metav1.ConditionUnknown,
		)
		c.Set(Condition{
			Type:   c.root,
			Status: status,
			Reason: lo.Ternary(
				status == metav1.ConditionUnknown,
				"ReconcilingDependents",
				"UnhealthyDependents",
			),
			Message: strings.Join(lo.Map(conditions, func(condition Condition, _ int) string {
				return fmt.Sprintf("%s=%s", condition.Type, condition.Status)
			}), ", "),
		})
	}
}

func (c ConditionSet) findUnhealthyDependents() []Condition {
	if len(c.dependents) == 0 {
		return nil
	}
	// Get dependent conditions
	conditions := c.object.GetConditions()
	conditions = lo.Filter(conditions, func(condition Condition, _ int) bool {
		return lo.Contains(c.dependents, condition.Type)
	})
	conditions = lo.Filter(conditions, func(condition Condition, _ int) bool {
		return condition.IsFalse() || condition.IsUnknown() || condition.ObservedGeneration != c.object.GetGeneration()
	})

	// Sort set conditions by time.
	sort.Slice(conditions, func(i, j int) bool {
		return conditions[i].LastTransitionTime.After(conditions[j].LastTransitionTime.Time)
	})
	return conditions
}
