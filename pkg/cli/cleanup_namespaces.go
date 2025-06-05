package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/util/namespaces"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	vClusterMetadataPrefix = "vcluster.loft.sh/"
)

// NamespaceCleanupHandler defines the function signature for namespace cleanup operations.
type NamespaceCleanupHandler func(
	ctx context.Context,
	mainPhysicalNamespace string,
	vClusterName string,
	nsSyncConfig config.SyncToHostNamespaces,
	k8sClient *kubernetes.Clientset,
	logger log.Logger,
) error

// GetNamespaceCleanupHandler returns a NamespaceCleanupHandler function based on the provided policy.
func GetNamespaceCleanupHandler(policy config.HostDeletionPolicy) (NamespaceCleanupHandler, error) {
	switch policy {
	case config.HostDeletionPolicyAll:
		return cleanupAllNamespaces, nil
	case config.HostDeletionPolicySynced:
		return cleanupSyncedNamespaces, nil
	case config.HostDeletionPolicyNone:
		return cleanupNoneNamespaces, nil
	default:
		return nil, fmt.Errorf("unsupported host namespace cleanup policy: %s", policy)
	}
}

// cleanupNoneNamespaces is a no-op handler for the 'none' policy.
func cleanupNoneNamespaces(
	_ context.Context,
	_ string, _ string,
	_ config.SyncToHostNamespaces,
	_ *kubernetes.Clientset,
	_ log.Logger,
) error {
	return nil
}

// cleanupSyncedNamespaces handles deletion of namespaces for the 'synced' policy.
// It deletes namespaces from the host cluster that were created as a result of syncing process from vCluster,
func cleanupSyncedNamespaces(
	ctx context.Context,
	mainPhysicalNamespace string,
	vClusterName string,
	_ config.SyncToHostNamespaces,
	k8sClient *kubernetes.Clientset,
	logger log.Logger,
) error {
	logger.Infof("Starting cleanup of vCluster '%s' namespaces.", vClusterName)

	if mainPhysicalNamespace == "" || vClusterName == "" {
		return fmt.Errorf("main physical namespace or vCluster name is empty")
	}

	labelSelector := translate.MarkerLabel + "=" + translate.SafeConcatName(mainPhysicalNamespace, "x", vClusterName)
	nsList, err := k8sClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return fmt.Errorf("list namespaces: %w", err)
	}

	if nsList == nil || len(nsList.Items) == 0 {
		logger.Infof("No additional managed namespaces found with label selector '%s'.", labelSelector)
		return nil
	}

	var errs []error
	for _, ns := range nsList.Items {
		// Check if namespace was imported, if yes we skip deletion for 'synced' policy.
		if ns.Annotations != nil && ns.Annotations[translate.ImportedMarkerAnnotation] == "true" {
			logger.Infof("Namespace %s was imported, cleaning up import.", ns.Name)
			if err := cleanupNamespaceMetadata(ctx, k8sClient, &ns); err != nil {
				errs = append(errs, err)
			}
			continue
		}

		logger.Infof("Attempting to delete virtual cluster namespace %s.", ns.Name)
		err := k8sClient.CoreV1().Namespaces().Delete(ctx, ns.Name, metav1.DeleteOptions{})
		if err != nil {
			errs = append(errs, fmt.Errorf("namespace %s: %w", ns.Name, err))
		} else {
			logger.Donef("Successfully deleted virtual cluster namespace %s.", ns.Name)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("cleanup of vCluster '%s' namespaces finished with errors: %w", vClusterName, errors.Join(errs...))
	}

	logger.Infof("Cleanup of vCluster '%s' namespaces finished.", vClusterName)
	return nil
}

// cleanupAllNamespaces handles deletion of namespaces for the 'all' policy.
// It deletes all namespaces matching target patterns in mappings, regardless of whether they were imported, created through syncing or not.
func cleanupAllNamespaces(
	ctx context.Context,
	_ string,
	vClusterName string,
	nsSyncConfig config.SyncToHostNamespaces,
	k8sClient *kubernetes.Clientset,
	logger log.Logger,
) error {
	logger.Infof("Starting cleanup of vCluster '%s' namespaces.", vClusterName)
	mappingsConfig := nsSyncConfig.Mappings

	if len(mappingsConfig.ByName) == 0 {
		logger.Infof("No namespace mappings defined.")
		logger.Infof("Cleanup of vCluster '%s' namespaces finished.", vClusterName)
		return nil
	}
	logger.Debugf("Processing %d namespace mappings for potential deletion.", len(mappingsConfig.ByName))

	hostNamespaces, err := k8sClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("list all host namespaces for mapping check: %w", err)
	}

	if hostNamespaces == nil || len(hostNamespaces.Items) == 0 {
		logger.Infof("No host namespaces found to check against mappings.")
		logger.Infof("Cleanup of vCluster '%s' namespaces finished - no namespaces found.", vClusterName)
		return nil
	}

	var errs []error
	for _, hostNs := range hostNamespaces.Items {
		// Check if this hostNs matches any mapping rule target
		for _, hostTargetPatternRaw := range mappingsConfig.ByName {
			processedHostTargetPattern := namespaces.ProcessNamespaceName(hostTargetPatternRaw, vClusterName)

			var currentRuleMatches bool
			if namespaces.IsPattern(processedHostTargetPattern) {
				_, currentRuleMatches = namespaces.MatchAndExtractWildcard(hostNs.Name, processedHostTargetPattern)
			} else {
				currentRuleMatches = (hostNs.Name == processedHostTargetPattern)
			}

			if currentRuleMatches {
				logger.Infof("Attempting to delete virtual cluster namespace %s.", hostNs.Name)
				err := k8sClient.CoreV1().Namespaces().Delete(ctx, hostNs.Name, metav1.DeleteOptions{})
				if err != nil {
					errs = append(errs, fmt.Errorf("delete virtual cluster namespace %s: %w", hostNs.Name, err))
				} else {
					logger.Donef("Successfully deleted virtual cluster namespace %s.", hostNs.Name)
				}
				// This namespace has been handled. Skip other mappings and move to the next one.
				break
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("cleanup of vCluster '%s' namespaces finished with errors: %w", vClusterName, errors.Join(errs...))
	}

	logger.Infof("Cleanup of vCluster '%s' namespaces finished.", vClusterName)
	return nil
}

func cleanupNamespaceMetadata(
	ctx context.Context,
	k8sClient *kubernetes.Clientset,
	ns *corev1.Namespace,
) error {
	for k := range ns.GetAnnotations() {
		if strings.HasPrefix(k, vClusterMetadataPrefix) {
			delete(ns.Annotations, k)
		}
	}

	for k := range ns.GetLabels() {
		if strings.HasPrefix(k, vClusterMetadataPrefix) {
			delete(ns.Labels, k)
		}
	}

	_, err := k8sClient.CoreV1().Namespaces().Update(ctx, ns, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("updating namespace %s after cleaning metadata: %w", ns.Name, err)
	}

	return nil
}
