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

type NamespaceCleanupHandler func(
	ctx context.Context,
	mainPhysicalNamespace string,
	vClusterName string,
	nsSyncConfig config.SyncToHostNamespaces,
	k8sClient *kubernetes.Clientset,
	logger log.Logger,
) error

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

// cleanupNoneNamespaces is a no-op handler for the 'none' policy. It only removes namespace metadata added by vCluster.
func cleanupNoneNamespaces(
	ctx context.Context,
	mainPhysicalNamespace string,
	vClusterName string,
	_ config.SyncToHostNamespaces,
	k8sClient *kubernetes.Clientset,
	logger log.Logger,
) error {
	logger.Infof("Starting metadata cleanup for vCluster '%s' namespaces.", vClusterName)
	managedNamespaces, err := getManagedNamespaces(ctx, k8sClient, mainPhysicalNamespace, vClusterName, logger)
	if err != nil {
		return err
	}

	var errs []error
	for _, ns := range managedNamespaces {
		if err := cleanupNamespaceMetadata(ctx, k8sClient, &ns); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("metadata cleanup for vCluster '%s' namespaces finished with errors: %w", vClusterName, errors.Join(errs...))
	}

	logger.Infof("Metadata cleanup for vCluster '%s' namespaces finished.", vClusterName)
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

	managedNamespaces, err := getManagedNamespaces(ctx, k8sClient, mainPhysicalNamespace, vClusterName, logger)
	if err != nil {
		return err
	}

	var errs []error
	for _, ns := range managedNamespaces {
		if isImportedNamespace(&ns) {
			logger.Infof("Namespace %s was imported, cleaning up import.", ns.Name)
			if err := cleanupNamespaceMetadata(ctx, k8sClient, &ns); err != nil {
				errs = append(errs, err)
			}
			continue
		}

		err := deleteAndLogNamespace(ctx, k8sClient, ns.Name, logger)
		if err != nil {
			errs = append(errs, err)
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

	hostNamespaces, err := getNamespaces(ctx, k8sClient, metav1.ListOptions{})
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
				var err error
				if isProtectedNamespace(hostNs.Name) {
					logger.Infof("Namespace %s is protected, cleaning up its metadata.", hostNs.Name)
					err = cleanupNamespaceMetadata(ctx, k8sClient, &hostNs)
				} else {
					err = deleteAndLogNamespace(ctx, k8sClient, hostNs.Name, logger)
				}

				if err != nil {
					errs = append(errs, err)
				}
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

func getNamespaces(ctx context.Context, k8sClient *kubernetes.Clientset, listOptions metav1.ListOptions) (*corev1.NamespaceList, error) {
	nsList, err := k8sClient.CoreV1().Namespaces().List(ctx, listOptions)
	if err != nil {
		return nil, fmt.Errorf("listing namespaces with options %+v: %w", listOptions, err)
	}
	return nsList, nil
}

func getManagedNamespaces(ctx context.Context, k8sClient *kubernetes.Clientset, mainPhysicalNamespace, vClusterName string, logger log.Logger) ([]corev1.Namespace, error) {
	if mainPhysicalNamespace == "" || vClusterName == "" {
		return nil, fmt.Errorf("main physical namespace or vCluster name is empty")
	}

	labelSelector := translate.MarkerLabel + "=" + translate.SafeConcatName(mainPhysicalNamespace, "x", vClusterName)
	listOptions := metav1.ListOptions{LabelSelector: labelSelector}

	nsList, err := getNamespaces(ctx, k8sClient, listOptions)
	if err != nil {
		return nil, err
	}

	if nsList == nil || len(nsList.Items) == 0 {
		logger.Infof("No additional managed namespaces found with label selector '%s'.", labelSelector)
		return nil, nil
	}

	return nsList.Items, nil
}

func deleteAndLogNamespace(ctx context.Context, k8sClient *kubernetes.Clientset, nsName string, logger log.Logger) error {
	logger.Infof("Attempting to delete virtual cluster namespace %s.", nsName)
	err := k8sClient.CoreV1().Namespaces().Delete(ctx, nsName, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("namespace %s: %w", nsName, err)
	}
	logger.Donef("Successfully deleted virtual cluster namespace %s.", nsName)
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

// isProtectedNamespace checks if a namespace should be protected from deletion.
// Protected namespaces include 'default' and any namespace prefixed with 'kube-'.
func isProtectedNamespace(name string) bool {
	return name == "default" || strings.HasPrefix(name, "kube-")
}

// isImportedNamespace checks if a namespace was imported
func isImportedNamespace(ns *corev1.Namespace) bool {
	return ns.Annotations != nil && ns.Annotations[translate.ImportedMarkerAnnotation] == "true"
}
