package specialservices

import (
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"k8s.io/apimachinery/pkg/types"
)

var Default = DefaultNameserverFinder()

const (
	DefaultKubeDNSServiceName      = "kube-dns"
	DefaultKubeDNSServiceNamespace = "kube-system"
)

type SpecialServiceSyncer func(
	ctx *synccontext.SyncContext,
	svcNamespace,
	svcName string,
	vSvcToSync types.NamespacedName,
	servicePortTranslator ServicePortTranslator,
) error

type Interface interface {
	SpecialServicesToSync() map[types.NamespacedName]SpecialServiceSyncer
}

type NameserverFinder struct {
	SpecialServices map[types.NamespacedName]SpecialServiceSyncer
}

func (f *NameserverFinder) SpecialServicesToSync() map[types.NamespacedName]SpecialServiceSyncer {
	return f.SpecialServices
}

func DefaultNameserverFinder() Interface {
	return &NameserverFinder{
		SpecialServices: map[types.NamespacedName]SpecialServiceSyncer{
			{
				Name:      DefaultKubernetesSVCName,
				Namespace: DefaultKubernetesSVCNamespace,
			}: SyncKubernetesService,
		},
	}
}
