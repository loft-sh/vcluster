package specialservices

import (
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var Default Interface

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
	DNSNamespace(ctx *synccontext.SyncContext) (client.Client, string)
}

func NewDefaultServiceSyncer() Interface {
	return &NameserverFinder{
		SpecialServices: map[types.NamespacedName]SpecialServiceSyncer{
			DefaultKubernetesSvcKey: SyncKubernetesService,
		},
	}
}

type NameserverFinder struct {
	SpecialServices map[types.NamespacedName]SpecialServiceSyncer
}

func (f *NameserverFinder) DNSNamespace(ctx *synccontext.SyncContext) (client.Client, string) {
	return ctx.PhysicalClient, translate.Default.PhysicalNamespace(DefaultKubeDNSServiceNamespace)
}

func (f *NameserverFinder) SpecialServicesToSync() map[types.NamespacedName]SpecialServiceSyncer {
	return f.SpecialServices
}
