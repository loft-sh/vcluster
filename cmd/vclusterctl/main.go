package main

import (
	"os"

	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd"
	"github.com/loft-sh/vcluster/pkg/upgrade"

	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

var version string = "0.0.1"

func main() {
	upgrade.SetVersion(version)

	cmd.Execute()
	os.Exit(0)
}
