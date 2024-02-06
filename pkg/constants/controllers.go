package constants

import "k8s.io/apimachinery/pkg/util/sets"

var ExistingControllers = sets.New(
	"services",
	"configmaps",
	"secrets",
	"endpoints",
	"pods",
	"events",
	"fake-nodes",
	"fake-persistentvolumes",
	"persistentvolumeclaims",
	"ingresses",
	"ingressclasses",
	"nodes",
	"persistentvolumes",
	"storageclasses",
	"hoststorageclasses",
	"priorityclasses",
	"networkpolicies",
	"volumesnapshots",
	"poddisruptionbudgets",
	"serviceaccounts",
	"csinodes",
	"csidrivers",
	"csistoragecapacities",
	"namespaces",
)

var DefaultEnabledControllers = sets.New(
	// helm charts need to be updated when changing this!
	// values.yaml and template/_helpers.tpl reference these
	"services",
	"configmaps",
	"secrets",
	"endpoints",
	"pods",
	"events",
	"persistentvolumeclaims",
	"fake-nodes",
	"fake-persistentvolumes",
)

var SchedulerRequiredControllers = sets.New(
	"csinodes",
	"csidrivers",
	"csistoragecapacities",
)
