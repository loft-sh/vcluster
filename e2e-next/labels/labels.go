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
	Vind        = Label("vind")

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

<<<<<<< HEAD
	// Feature-specific labels for targeted filtering
	Scheduler    = Label("scheduler")
	MetricsProxy = Label("metricsproxy")
	Certs        = Label("certs")
	Rootless     = Label("rootless")
	Isolation    = Label("isolation")
	NodeSync     = Label("nodesync")
	Plugin       = Label("plugin")
	CLI          = Label("cli")
=======
	// Suite-primary labels (one per opt-in suite).
	Scheduler        = Label("scheduler")
	MetricsProxy     = Label("metricsproxy")
	Certs            = Label("certs")
	Rootless         = Label("rootless")
	Isolation        = Label("isolation")
	NodeSync         = Label("nodesync")
	Plugin           = Label("plugin")
	CLI              = Label("cli")
	ExportKubeConfig = Label("exportkubeconfig")
	Migration        = Label("migration")
>>>>>>> 0f8c245a0 (fix: k3s to k8s cert migration (#3952))
)
