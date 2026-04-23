// Package labels lists the Ginkgo labels used across the e2e-next suite.
// Use them with --label-filter="..." to pick which suites run.
// Each suite_*_test.go file tags its outer Describe with its primary label.
package labels

import (
	. "github.com/onsi/ginkgo/v2"
)

var (
	// PR is applied to suites that gate every PR.
	PR = Label("pr")

	// Feature-area labels.
	Core        = Label("core")
	Sync        = Label("sync")
	Integration = Label("integration")
	Deploy      = Label("deploy")
	Storage     = Label("storage")
	Security    = Label("security")
	Vind        = Label("vind")

	// Resource-specific labels for targeted filtering inside sync tests.
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
)
