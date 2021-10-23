package networkpolicies

import (
	context2 "github.com/loft-sh/vcluster/cmd/vcluster/context"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/generic"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/networking/v1"
	"k8s.io/client-go/kubernetes"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
)

func RegisterIndices(ctx *context2.ControllerContext) error {
	err := generic.RegisterSyncerIndices(ctx, &v1.NetworkPolicy{})
	if err != nil {
		return err
	}

	return nil
}

func Register(ctx *context2.ControllerContext) error {
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartRecordingToSink(&v1core.EventSinkImpl{Interface: kubernetes.NewForConfigOrDie(ctx.VirtualManager.GetConfig()).CoreV1().Events("")})
	return generic.RegisterSyncer(ctx, &syncer{
		eventRecoder:    eventBroadcaster.NewRecorder(ctx.VirtualManager.GetScheme(), corev1.EventSource{Component: "networkpolicy-syncer"}),
		enableIngress:   ctx.Options.EnableNetworkPoliciesIngress,
		enableEgress:    ctx.Options.EnableNetworkPoliciesEgress,
		targetNamespace: ctx.Options.TargetNamespace,
		virtualClient:   ctx.VirtualManager.GetClient(),
		localClient:     ctx.LocalManager.GetClient(),
	}, "networkpolicy", generic.RegisterSyncerOptions{})
}
