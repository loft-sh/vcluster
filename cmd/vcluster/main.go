package main

import (
	"os"

	"github.com/loft-sh/vcluster/cmd/vcluster/cmd"
	"github.com/loft-sh/vcluster/pkg/util/log"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"

	// "go.uber.org/zap/zapcore"
	// zappkg "go.uber.org/zap"

	// +kubebuilder:scaffold:imports

	// Make sure dep tools picks up these dependencies
	_ "github.com/go-openapi/loads"
	_ "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth" // Enable cloud provider auth
)

func main() {
	// set global logger
	if os.Getenv("DEBUG") == "true" {
		ctrl.SetLogger(log.NewLog(0))
	} else {
		ctrl.SetLogger(log.NewLog(2))
	}

	// create a new command and execute
	err := cmd.BuildRoot().Execute()
	if err != nil {
		klog.Fatal(err)
	}
}
