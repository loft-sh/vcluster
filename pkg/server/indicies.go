package server

import (
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/nodes"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/nodes/nodeservice"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// RegisterIndices adds the server indices to the managers
func RegisterIndices(ctx *synccontext.RegisterContext) error {
	// index services by ip
	if ctx.Config.Networking.Advanced.ProxyKubelets.ByIP {
		err := ctx.HostManager.GetFieldIndexer().IndexField(ctx, &corev1.Service{}, constants.IndexByClusterIP, func(object client.Object) []string {
			svc := object.(*corev1.Service)
			if len(svc.Labels) == 0 || svc.Labels[nodeservice.ServiceClusterLabel] != translate.VClusterName {
				return nil
			}

			return []string{svc.Spec.ClusterIP}
		})
		if err != nil {
			return err
		}
	}

	err := ctx.VirtualManager.GetFieldIndexer().IndexField(ctx, &corev1.Node{}, constants.IndexByHostName, func(rawObj client.Object) []string {
		return []string{nodes.GetNodeHost(rawObj.GetName()), nodes.GetNodeHostLegacy(rawObj.GetName(), ctx.Config.HostNamespace)}
	})
	if err != nil {
		return err
	}

	return nil
}
