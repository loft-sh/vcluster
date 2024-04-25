package main

import (
	"context"
	"os"

	"github.com/go-logr/logr"
	loftlogr "github.com/loft-sh/log/logr"
	"github.com/loft-sh/vcluster/cmd/vcluster/cmd"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/klog/v2"
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
		_ = os.Setenv("LOFT_LOG_LEVEL", "debug")
	} else {
		_ = os.Setenv("LOFT_LOG_LEVEL", "info")
	}

	// set global logger
	logger, err := loftlogr.NewLoggerWithOptions(
		loftlogr.WithOptionsFromEnv(),
		loftlogr.WithComponentName("vcluster"),
		loftlogr.WithGlobalZap(true),
		loftlogr.WithGlobalKlog(true),
	)
	if err != nil {
		klog.Fatal(err)
	}
	ctrl.SetLogger(logger)
	ctx := logr.NewContext(context.Background(), logger)

	// create a new command and execute
	err = cmd.BuildRoot().ExecuteContext(ctx)
	if err != nil {
		klog.Fatal(err)
	}
}
