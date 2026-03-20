package labels

import (
	. "github.com/onsi/ginkgo/v2"
)

var (
	// Run on every PR
	PR = Label("pr")
	// Test Groups
	Core        = Label("core")
	Sync        = Label("sync")
	Integration = Label("integration")
	Deploy      = Label("deploy")
	Storage     = Label("storage")
	Security    = Label("security")

	// Resource-specific labels for fromHost sync tests
	PriorityClasses = Label("priorityclasses")
	RuntimeClasses  = Label("runtimeclasses")
	StorageClasses  = Label("storageclasses")
	IngressClasses  = Label("ingressclasses")
	ConfigMaps      = Label("configmaps")
	Secrets         = Label("secrets")
)
