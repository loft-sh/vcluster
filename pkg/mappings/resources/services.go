package resources

import (
	"fmt"

	"github.com/loft-sh/vcluster/pkg/mappings/generic"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CreateServiceMapper(ctx *synccontext.RegisterContext) (synccontext.Mapper, error) {
	mapper, err := generic.NewMapperWithoutRecorder(ctx, &corev1.Service{}, func(ctx *synccontext.SyncContext, vName, vNamespace string, _ client.Object) types.NamespacedName {
		return translate.Default.HostName(ctx, vName, vNamespace)
	})
	if err != nil {
		return nil, err
	}

	return generic.WithRecorder(&servicesMapper{
		Mapper: mapper,
	}), nil
}

type servicesMapper struct {
	synccontext.Mapper
}

func (s *servicesMapper) Migrate(ctx *synccontext.RegisterContext, _ synccontext.Mapper) error {
	vObj := synccontext.Object{
		GroupVersionKind: s.GroupVersionKind(),
		NamespacedName: types.NamespacedName{
			Namespace: "default",
			Name:      "kubernetes",
		},
	}
	expectedHostName := types.NamespacedName{
		Name:      translate.VClusterName,
		Namespace: ctx.CurrentNamespace,
	}

	// check if there is an existing mapping already
	existingHostName, ok := ctx.Mappings.Store().VirtualToHostName(ctx, vObj)
	if ok && existingHostName.String() != expectedHostName.String() {
		klog.FromContext(ctx).Info("Fix default/kubernetes mapping", "before", existingHostName, "now", expectedHostName)

		// delete existing mapping & references
		existingMapping := vObj.WithHostName(existingHostName)
		err := ctx.Mappings.Store().DeleteMapping(ctx, existingMapping)
		if err != nil {
			return err
		}
		for _, reference := range ctx.Mappings.Store().ReferencesTo(ctx, vObj) {
			err = ctx.Mappings.Store().DeleteReferenceAndSave(ctx, existingMapping, reference)
			if err != nil {
				return fmt.Errorf("delete reference: %w", err)
			}
		}

		// add new mapping
		expectedMapping := vObj.WithHostName(expectedHostName)
		err = ctx.Mappings.Store().AddReferenceAndSave(ctx, expectedMapping, expectedMapping)
		if err != nil {
			return fmt.Errorf("add mapping: %w", err)
		}
	}

	return nil
}

func (s *servicesMapper) VirtualToHost(ctx *synccontext.SyncContext, req types.NamespacedName, vObj client.Object) types.NamespacedName {
	if req.Name == "kubernetes" && req.Namespace == "default" {
		return types.NamespacedName{
			Name:      translate.VClusterName,
			Namespace: ctx.CurrentNamespace,
		}
	}

	return s.Mapper.VirtualToHost(ctx, req, vObj)
}

func (s *servicesMapper) HostToVirtual(ctx *synccontext.SyncContext, req types.NamespacedName, pObj client.Object) types.NamespacedName {
	if req.Name == translate.VClusterName && req.Namespace == ctx.CurrentNamespace {
		return types.NamespacedName{
			Name:      "kubernetes",
			Namespace: "default",
		}
	}

	namespaceName := s.Mapper.HostToVirtual(ctx, req, pObj)
	if namespaceName.Name == "kubernetes" && req.Namespace == "default" {
		return types.NamespacedName{}
	}

	return namespaceName
}
