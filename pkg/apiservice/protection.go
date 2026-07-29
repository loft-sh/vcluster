package apiservice

import (
	"context"
	"fmt"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	apiregistrationv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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
	policy := &admissionregistrationv1.ValidatingAdmissionPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name: protectionPolicyName,
		},
	}
	if _, err := controllerutil.CreateOrUpdate(ctx, c, policy, func() error {
		policy.Spec.FailurePolicy = ptr.To(admissionregistrationv1.Fail)
		policy.Spec.MatchConstraints = &admissionregistrationv1.MatchResources{
			ObjectSelector: &metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{
					{
						Key:      protectionLabelKey,
						Operator: metav1.LabelSelectorOpExists,
					},
				},
			},
			ResourceRules: []admissionregistrationv1.NamedRuleWithOperations{
				{
					RuleWithOperations: admissionregistrationv1.RuleWithOperations{
						Operations: []admissionregistrationv1.OperationType{admissionregistrationv1.Delete},
						Rule: admissionregistrationv1.Rule{
							APIGroups:   []string{""},
							APIVersions: []string{"v1"},
							Resources:   []string{"services"},
						},
					},
				},
				{
					RuleWithOperations: admissionregistrationv1.RuleWithOperations{
						Operations: []admissionregistrationv1.OperationType{admissionregistrationv1.Delete},
						Rule: admissionregistrationv1.Rule{
							APIGroups:   []string{"apiregistration.k8s.io"},
							APIVersions: []string{"v1"},
							Resources:   []string{"apiservices"},
						},
					},
				},
			},
		}
		policy.Spec.Validations = []admissionregistrationv1.Validation{
			{
				Expression: `false`,
				Message:    "deletion of vCluster-protected resources is denied; remove the corresponding entry from your vCluster configuration",
			},
		}
		return nil
	}); err != nil {
		return fmt.Errorf("create or update %s: %w", protectionPolicyName, err)
	}
	return nil
}

func ensureProtectionBinding(ctx context.Context, c client.Client, tag string) error {
	bindingName := protectionBindingNamePrefix + tag
	binding := &admissionregistrationv1.ValidatingAdmissionPolicyBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: bindingName,
		},
	}
	if _, err := controllerutil.CreateOrUpdate(ctx, c, binding, func() error {
		binding.Spec.PolicyName = protectionPolicyName
		binding.Spec.ValidationActions = []admissionregistrationv1.ValidationAction{
			admissionregistrationv1.Deny,
		}
		binding.Spec.MatchResources = &admissionregistrationv1.MatchResources{
			ObjectSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					protectionLabelKey: tag,
				},
			},
		}
		return nil
	}); err != nil {
		return fmt.Errorf("create or update %s: %w", bindingName, err)
	}
	return nil
}

func removeLabel(ctx context.Context, c client.Client, obj client.Object, key string) error {
	labels := obj.GetLabels()
	if _, ok := labels[key]; !ok {
		return nil
	}
	delete(labels, key)
	return client.IgnoreNotFound(c.Update(ctx, obj))
}
