package labels

import (
	. "github.com/onsi/ginkgo/v2"
)

var (
	// Run on every PR
	PR = Label("pr")
	// Test Groups (legacy?)
	Core = Label("core")

	Test = Label("test")
)
