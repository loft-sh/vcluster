package main

import (
	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd"
	"os"

	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

func main() {
	cmd.Execute()
	os.Exit(0)
}
