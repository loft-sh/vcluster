package controllers

import (
	"fmt"
	"strings"

	"github.com/loft-sh/vcluster/cmd/vcluster/context"
	"github.com/loft-sh/vcluster/pkg/controllers/coredns"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/configmaps"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/endpoints"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/events"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/ingresses"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/networkpolicies"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/nodes"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/persistentvolumeclaims"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/persistentvolumes"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/pods"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/priorityclasses"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/secrets"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/services"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/storageclasses"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/volumesnapshots"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"
)

var ResourceControllers = map[string]func(*synccontext.RegisterContext) (syncer.Object, error){
	"services":               services.New,
	"configmaps":             configmaps.New,
	"secrets":                secrets.New,
	"endpoints":              endpoints.New,
	"pods":                   pods.New,
	"events":                 events.New,
	"persistentvolumeclaims": persistentvolumeclaims.New,
	"ingresses":              ingresses.New,
	"storageclasses":         storageclasses.New,
	"priorityclasses":        priorityclasses.New,
	"nodes,fake-nodes":       nodes.New,
	"persistentvolumes,fake-persistentvolumes": persistentvolumes.New,
	"networkpolicies":                          networkpolicies.New,
	"volumesnapshots":                          volumesnapshots.New,
}

func Create(ctx *context.ControllerContext) ([]syncer.Object, error) {
	registerContext := ToRegisterContext(ctx)

	// register controllers for resource synchronization
	syncers := []syncer.Object{}
	for k, v := range ResourceControllers {
		controllers := strings.Split(k, ",")
		for _, controller := range controllers {
			if ctx.Controllers[controller] {
				loghelper.Infof("Start %s sync controller", controller)
				ctrl, err := v(registerContext)
				if err != nil {
					return nil, errors.Wrapf(err, "register %s controller", controller)
				}

				syncers = append(syncers, ctrl)
				break
			}
		}
	}

	return syncers, nil
}

func RegisterIndices(ctx *context.ControllerContext, syncers []syncer.Object) error {
	registerContext := ToRegisterContext(ctx)
	for _, s := range syncers {
		indexRegisterer, ok := s.(syncer.IndicesRegisterer)
		if ok {
			err := indexRegisterer.RegisterIndices(registerContext)
			if err != nil {
				return errors.Wrapf(err, "register indices for %s syncer", s.Name())
			}
		}
	}

	return nil
}

func RegisterControllers(ctx *context.ControllerContext, syncers []syncer.Object) error {
	registerContext := ToRegisterContext(ctx)
	ctx.EventBroadcaster.StartRecordingToSink(&v1core.EventSinkImpl{Interface: kubernetes.NewForConfigOrDie(ctx.VirtualManager.GetConfig()).CoreV1().Events("")})

	// register controller that keeps CoreDNS NodeHosts config up to date
	err := registerCoreDNSController(ctx)
	if err != nil {
		return err
	}

	// register controllers for resource synchronization
	for _, v := range syncers {
		// fake syncer?
		fakeSyncer, ok := v.(syncer.FakeSyncer)
		if ok {
			err = syncer.RegisterFakeSyncer(registerContext, fakeSyncer)
			if err != nil {
				return errors.Wrapf(err, "start %s syncer", v.Name())
			}
		} else {
			// real syncer?
			realSyncer, ok := v.(syncer.Syncer)
			if ok {
				err = syncer.RegisterSyncer(registerContext, realSyncer)
				if err != nil {
					return errors.Wrapf(err, "start %s syncer", v.Name())
				}
			} else {
				return fmt.Errorf("syncer %s does not implement fake syncer or syncer interface", v.Name())
			}
		}
	}

	return nil
}

func registerCoreDNSController(ctx *context.ControllerContext) error {
	err := (&coredns.CoreDNSNodeHostsReconciler{
		Client: ctx.VirtualManager.GetClient(),
		Log:    loghelper.New("corednsnodehosts-controller"),
	}).SetupWithManager(ctx.VirtualManager)
	if err != nil {
		return fmt.Errorf("unable to setup CoreDNS NodeHosts controller: %v", err)
	}
	return nil
}

func ToRegisterContext(ctx *context.ControllerContext) *synccontext.RegisterContext {
	return &synccontext.RegisterContext{
		Context:          ctx.Context,
		EventBroadcaster: ctx.EventBroadcaster,

		Options:             ctx.Options,
		NodeServiceProvider: ctx.NodeServiceProvider,
		Controllers:         ctx.Controllers,
		LockFactory:         ctx.LockFactory,

		TargetNamespace:        ctx.Options.TargetNamespace,
		CurrentNamespace:       ctx.CurrentNamespace,
		CurrentNamespaceClient: ctx.CurrentNamespaceClient,

		VirtualManager:  ctx.VirtualManager,
		PhysicalManager: ctx.LocalManager,
	}
}
