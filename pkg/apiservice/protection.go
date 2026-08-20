package apiservice

import (
	"context"
	"fmt"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	admissionregistrationv1ac "k8s.io/client-go/applyconfigurations/admissionregistration/v1"
	metav1ac "k8s.io/client-go/applyconfigurations/meta/v1"
	apiregistrationv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// protectionLabelKey identifies APIService backends that should be defended
	// against deletion by the shared ValidatingAdmissionPolicy. The label value
	// names the feature owning the resource, so a per-feature binding can scope
	// its deny rule via objectSelector.
	protectionLabelKey = "vcluster.loft.sh/protected"

	// protectionPolicyName is the name of the shared generic policy. Bindings
	// per feature reference it.
	protectionPolicyName = "vcluster-protected-apiservices"

	protectionBindingNamePrefix = "vcluster-protected-"

	protectionFieldOwner = "vcluster-apiservice-protection"
)

// enableDeletionProtection installs the shared protection policy and a
// per-feature binding scoped to tag, then returns the labels the caller must
// stamp onto each resource to bring it under that protection.
func enableDeletionProtection(ctx context.Context, c client.Client, tag string) (map[string]string, error) {
	if err := ensureProtectionPolicy(ctx, c); err != nil {
		return nil, fmt.Errorf("ensure protection policy: %w", err)
	}
	if err := ensureProtectionBinding(ctx, c, tag); err != nil {
		return nil, fmt.Errorf("ensure protection binding for %q: %w", tag, err)
	}
	return map[string]string{protectionLabelKey: tag}, nil
}

// disableDeletionProtection removes the protection label from the APIService
// for the given group/version and from its backend Service so the protection
// policy stops blocking deletes.
func disableDeletionProtection(ctx context.Context, c client.Client, groupVersion schema.GroupVersion) error {
	apiService := &apiregistrationv1.APIService{}
	if err := c.Get(ctx, types.NamespacedName{Name: groupVersion.Version + "." + groupVersion.Group}, apiService); err != nil {
		return client.IgnoreNotFound(err)
	}
	if err := removeLabel(ctx, c, apiService, protectionLabelKey); err != nil {
		return err
	}

	ref := apiService.Spec.Service
	if ref == nil || ref.Name == "" {
		return nil
	}
	service := &corev1.Service{}
	if err := c.Get(ctx, types.NamespacedName{Namespace: ref.Namespace, Name: ref.Name}, service); err != nil {
		return client.IgnoreNotFound(err)
	}
	return removeLabel(ctx, c, service, protectionLabelKey)
}

func ensureProtectionPolicy(ctx context.Context, c client.Client) error {
	policy := admissionregistrationv1ac.ValidatingAdmissionPolicy(protectionPolicyName).
		WithSpec(admissionregistrationv1ac.ValidatingAdmissionPolicySpec().
			WithFailurePolicy(admissionregistrationv1.Fail).
			WithMatchConstraints(admissionregistrationv1ac.MatchResources().
				WithObjectSelector(metav1ac.LabelSelector().
					WithMatchExpressions(metav1ac.LabelSelectorRequirement().
						WithKey(protectionLabelKey).
						WithOperator(metav1.LabelSelectorOpExists))).
				WithResourceRules(
					protectedDeleteRule("", "v1", "services"),
					protectedDeleteRule("apiregistration.k8s.io", "v1", "apiservices"),
				)).
			WithValidations(admissionregistrationv1ac.Validation().
				WithExpression(`false`).
				WithMessage("deletion of vCluster-protected resources is denied; remove the corresponding entry from your vCluster configuration")))

	if err := applyProtectionObject(ctx, c, policy); err != nil {
		return fmt.Errorf("apply %s: %w", protectionPolicyName, err)
	}
	return nil
}

func ensureProtectionBinding(ctx context.Context, c client.Client, tag string) error {
	bindingName := protectionBindingNamePrefix + tag
	binding := admissionregistrationv1ac.ValidatingAdmissionPolicyBinding(bindingName).
		WithSpec(admissionregistrationv1ac.ValidatingAdmissionPolicyBindingSpec().
			WithPolicyName(protectionPolicyName).
			WithValidationActions(admissionregistrationv1.Deny).
			WithMatchResources(admissionregistrationv1ac.MatchResources().
				WithObjectSelector(metav1ac.LabelSelector().
					WithMatchLabels(map[string]string{protectionLabelKey: tag}))))

	if err := applyProtectionObject(ctx, c, binding); err != nil {
		return fmt.Errorf("apply %s: %w", bindingName, err)
	}
	return nil
}

func protectedDeleteRule(group, version, resource string) *admissionregistrationv1ac.NamedRuleWithOperationsApplyConfiguration {
	return admissionregistrationv1ac.NamedRuleWithOperations().
		WithOperations(admissionregistrationv1.Delete).
		WithAPIGroups(group).
		WithAPIVersions(version).
		WithResources(resource)
}

func applyProtectionObject(ctx context.Context, c client.Client, obj runtime.ApplyConfiguration) error {
	return c.Apply(ctx, obj, client.FieldOwner(protectionFieldOwner), client.ForceOwnership)
}

func removeLabel(ctx context.Context, c client.Client, obj client.Object, key string) error {
	labels := obj.GetLabels()
	if _, ok := labels[key]; !ok {
		return nil
	}
	delete(labels, key)
	return client.IgnoreNotFound(c.Update(ctx, obj))
}
