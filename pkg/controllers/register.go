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

var ResourceControllers = map[string]func(*synccontext.RegisterContext) syncer.Object{
	"services":               services.New,
	"configmaps":             configmaps.New,
	"secrets":                secrets.Register,
	"endpoints":              endpoints.Register,
	"pods":                   pods.Register,
	"events":                 events.Register,
	"persistentvolumeclaims": persistentvolumeclaims.Register,
	"ingresses":              ingresses.Register,
	"storageclasses":         storageclasses.Register,
	"priorityclasses":        priorityclasses.Register,
	"nodes,fake-nodes":       nodes.Register,
	"persistentvolumes,fake-persistentvolumes": persistentvolumes.Register,
	"networkpolicies":                          networkpolicies.Register,
	"volumesnapshots":                          volumesnapshots.Register,
}

var ResourcePrerequisites = map[string]func(*context.ControllerContext) error{
	"volumesnapshots": volumesnapshots.EnsurePrerequisites,
}

func EnsurePrerequisites(ctx *context.ControllerContext) error {
	// call the EnsurePrerequisites for the resources that require it
	for k, v := range ResourcePrerequisites {
		controllers := strings.Split(k, ",")
		for _, controller := range controllers {
			if ctx.Controllers[controller] {
				err := v(ctx)
				if err != nil {
					return errors.Wrapf(err, "ensure %s prerequisites", controller)
				}
				break
			}
		}
	}

	return nil
}

func RegisterControllers(ctx *context.ControllerContext) error {
	ctx.EventBroadcaster.StartRecordingToSink(&v1core.EventSinkImpl{Interface: kubernetes.NewForConfigOrDie(ctx.VirtualManager.GetConfig()).CoreV1().Events("")})

	// register controller that keeps CoreDNS NodeHosts config up to date
	err := registerCoreDNSController(ctx)
	if err != nil {
		return err
	}

	// register controllers for resource synchronization
	for k, v := range ResourceControllers {
		controllers := strings.Split(k, ",")
		for _, controller := range controllers {
			if ctx.Controllers[controller] {
				loghelper.Infof("Start %s sync controller", controller)
				err := v(ctx, ctx.EventBroadcaster)
				if err != nil {
					return errors.Wrapf(err, "register %s controller", controller)
				}
				break
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
