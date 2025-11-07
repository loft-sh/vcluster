package labels

import (
	. "github.com/onsi/ginkgo/v2"
)

var (
	// Run on every PR
	PR = Label("pr")
	// Test Groups (legacy?)
	Core        = Label("core")
	Sync        = Label("sync")
	Integration = Label("integration")
	Deploy      = Label("deploy")
	Storage     = Label("storage")
	Security    = Label("security")
	Ha          = Label("ha")
)
