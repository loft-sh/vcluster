package main

import (
	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd"
	"github.com/loft-sh/vcluster/pkg/upgrade"
	"os"

	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

var version string = "v0.0.0"

func main() {
	upgrade.SetVersion(version)

	cmd.Execute()
	os.Exit(0)
}
