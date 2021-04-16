package controllers

import (
	"github.com/loft-sh/vcluster/cmd/vcluster/context"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/configmaps"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/endpoints"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/events"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/ingresses"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/nodes"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/persistentvolumeclaims"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/persistentvolumes"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/pods"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/secrets"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/services"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/storageclasses"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"github.com/pkg/errors"
	"strings"
)

var Resources = map[string]func(*context.ControllerContext) error{
	"services":               services.Register,
	"configmaps":             configmaps.Register,
	"secrets":                secrets.Register,
	"endpoints":              endpoints.Register,
	"pods":                   pods.Register,
	"events":                 events.Register,
	"persistentvolumeclaims": persistentvolumeclaims.Register,
	"ingresses":              ingresses.Register,
	"nodes":                  nodes.Register,
	"persistentvolumes":      persistentvolumes.Register,
	"storageclasses":         storageclasses.Register,
}

func Register(ctx *context.ControllerContext) error {
	disabled := parseDisabled(ctx.Options.DisableSyncResources)
	for k, v := range Resources {
		if disabled[k] {
			continue
		}

		loghelper.Infof("Start %s sync controller", k)
		err := v(ctx)
		if err != nil {
			return errors.Wrapf(err, "register %s controller", k)
		}
	}

	return nil
}

func parseDisabled(str string) map[string]bool {
	splitted := strings.Split(str, ",")
	ret := map[string]bool{}
	for _, s := range splitted {
		ret[strings.TrimSpace(strings.ToLower(s))] = true
	}
	return ret
}
