package setup

import (
	"errors"
	"fmt"

	"github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// deletePreviouslySyncedResources deletes resources that were synced from host to virtual, but
// should not be synced anymore, because from host syncing has been disabled.
func deletePreviouslySyncedResources(ctx *synccontext.ControllerContext) error {
	err := deletePreviouslyReplicatedServices(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete previously synced services: %w", err)
	}
	return nil
}

// deletePreviouslyReplicatedServices deletes services that were synced from host to virtual, but
// should not be synced anymore, because from host syncing for services has been disabled.
func deletePreviouslyReplicatedServices(ctx *synccontext.ControllerContext) error {
	virtualClient := ctx.VirtualManager.GetClient()
	listOptions := client.ListOptions{
		LabelSelector: labels.SelectorFromSet(labels.Set{
			translate.ControllerLabel: "vcluster",
		}),
	}
	previouslySyncedServices := &corev1.ServiceList{}
	err := virtualClient.List(ctx, previouslySyncedServices, &listOptions)
	if err != nil {
		return fmt.Errorf("failed to list previously synced services: %w", err)
	}
	if len(previouslySyncedServices.Items) == 0 {
		return nil
	}

	logger := ctx.VirtualManager.GetLogger()
	logger.Info("deleting previously synced services")
	var deleteErrors []error
	for _, service := range previouslySyncedServices.Items {
		if replicateServicesFromHostConfigContainsVirtualService(ctx.Config.Networking.ReplicateServices, service) {
			logger.Info("virtual service has replication config, not deleting it", "name", service.Name, "namespace", service.Namespace)
			continue
		}
		logger.Info("deleting previously synced service", "name", service.Name, "namespace", service.Namespace)
		err = virtualClient.Delete(ctx, &service)
		if err != nil {
			deleteErrors = append(deleteErrors, fmt.Errorf("failed to delete previously synced service: %w", err))
			continue
		}
		logger.Info("deleted previously synced service", "name", service.Name, "namespace", service.Namespace)
	}
	if len(deleteErrors) > 0 {
		return fmt.Errorf("failed to delete one or more previously synced services: %w", errors.Join(deleteErrors...))
	}
	logger.Info("finished deleting previously synced services")
	return nil
}

func replicateServicesFromHostConfigContainsVirtualService(replicateServicesConfig config.ReplicateServices, service corev1.Service) bool {
	serviceNamespacedName := types.NamespacedName{
		Namespace: service.Namespace,
		Name:      service.Name,
	}.String()
	for _, serviceMapping := range replicateServicesConfig.FromHost {
		if serviceMapping.To == serviceNamespacedName {
			return true
		}
	}
	return false
}
