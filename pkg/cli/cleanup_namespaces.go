package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	vClusterMetadataPrefix = "vcluster.loft.sh/"
)

// CleanupSyncedNamespaces handles deletion of namespaces for the 'synced' policy.
// It deletes namespaces from the host cluster that were created as a result of syncing process from vCluster,
func CleanupSyncedNamespaces(
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

// isImportedNamespace checks if a namespace was imported
func isImportedNamespace(ns *corev1.Namespace) bool {
	return ns.Annotations != nil && ns.Annotations[translate.ImportedMarkerAnnotation] == "true"
}
