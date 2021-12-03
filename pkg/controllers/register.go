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
	"github.com/loft-sh/vcluster/pkg/controllers/resources/nodes"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/persistentvolumeclaims"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/persistentvolumes"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/pods"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/priorityclasses"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/secrets"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/services"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/storageclasses"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
)

var ResourceControllers = map[string]func(*context.ControllerContext, record.EventBroadcaster) error{
	"services":               services.Register,
	"configmaps":             configmaps.Register,
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
}

var ResourceIndices = map[string]func(*context.ControllerContext) error{
	"services":               services.RegisterIndices,
	"configmaps":             configmaps.RegisterIndices,
	"secrets":                secrets.RegisterIndices,
	"endpoints":              endpoints.RegisterIndices,
	"pods":                   pods.RegisterIndices,
	"events":                 events.RegisterIndices,
	"persistentvolumeclaims": persistentvolumeclaims.RegisterIndices,
	"ingresses":              ingresses.RegisterIndices,
	"storageclasses":         storageclasses.RegisterIndices,
	"priorityclasses":        priorityclasses.RegisterIndices,
	"nodes,fake-nodes":       nodes.RegisterIndices,
	"persistentvolumes,fake-persistentvolumes": persistentvolumes.RegisterIndices,
}

func RegisterIndices(ctx *context.ControllerContext) error {
	// register the resource indices
	for k, v := range ResourceIndices {
		controllers := strings.Split(k, ",")
		for _, controller := range controllers {
			if ctx.Controllers[controller] {
				err := v(ctx)
				if err != nil {
					return errors.Wrapf(err, "register %s indices", controller)
				}
				break
			}
		}
	}

	return nil
}

func RegisterControllers(ctx *context.ControllerContext) error {
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartRecordingToSink(&v1core.EventSinkImpl{Interface: kubernetes.NewForConfigOrDie(ctx.VirtualManager.GetConfig()).CoreV1().Events("")})

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
				err := v(ctx, eventBroadcaster)
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
