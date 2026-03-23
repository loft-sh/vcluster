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
	NetworkPolicies = Label("networkpolicies")
	Pods            = Label("pods")
	PVCs            = Label("pvcs")
	Events          = Label("events")
	CoreDNS         = Label("coredns")
	Webhooks        = Label("webhooks")
	Snapshots       = Label("snapshots")

	// NonDefault marks tests that require special infrastructure not available
	// in the standard Kind cluster (e.g. a CNI with NetworkPolicy enforcement).
	// These tests are excluded from the default label filter ("!non-default").
	NonDefault = Label("non-default")
)
