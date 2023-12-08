package cmd

import (
	"context"
	"fmt"
	"runtime/debug"

	"github.com/loft-sh/vcluster/pkg/leaderelection"
	"github.com/loft-sh/vcluster/pkg/options"
	"github.com/loft-sh/vcluster/pkg/scheme"
	"github.com/loft-sh/vcluster/pkg/setup"
	"github.com/loft-sh/vcluster/pkg/telemetry"
	"github.com/loft-sh/vcluster/pkg/util/blockingcacheclient"
	"github.com/loft-sh/vcluster/pkg/util/clienthelper"
	"github.com/loft-sh/vcluster/pkg/util/pluginhookclient"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
)

func NewStartCommand() *cobra.Command {
	vClusterOptions := &options.VirtualClusterOptions{}
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Execute the vcluster",
		Args:  cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) (err error) {
			// start telemetry
			telemetry.Start(false)
			defer telemetry.Collector.Flush()

			// capture errors
			defer func() {
				if r := recover(); r != nil {
					telemetry.Collector.RecordError(cobraCmd.Context(), telemetry.PanicSeverity, fmt.Errorf("panic: %v %s", r, string(debug.Stack())))
					panic(r)
				} else if err != nil {
					telemetry.Collector.RecordError(cobraCmd.Context(), telemetry.FatalSeverity, err)
				}
			}()

			// execute command
			return ExecuteStart(cobraCmd.Context(), vClusterOptions)
		},
	}

	options.AddFlags(cmd.Flags(), vClusterOptions)
	return cmd
}

func ExecuteStart(ctx context.Context, options *options.VirtualClusterOptions) error {
	// set suffix
	translate.VClusterName = options.Name
	if translate.VClusterName == "" {
		translate.VClusterName = options.DeprecatedSuffix
	}
	if translate.VClusterName == "" {
		translate.VClusterName = "vcluster"
	}
	translate.SaveSuffix()

	// set service name
	if options.ServiceName == "" {
		options.ServiceName = translate.VClusterName
	}

	// get current namespace
	currentNamespace, err := clienthelper.CurrentNamespace()
	if err != nil {
		return err
	}

	// get host cluster config and tweak rate-limiting configuration
	inClusterConfig := ctrl.GetConfigOrDie()
	inClusterConfig.QPS = 40
	inClusterConfig.Burst = 80
	inClusterConfig.Timeout = 0
	inClusterClient, err := kubernetes.NewForConfig(inClusterConfig)
	if err != nil {
		return err
	}

	// init telemetry
	telemetry.Collector.Init(inClusterConfig, currentNamespace, options)

	// check if we should create certs
	err = setup.Initialize(
		ctx,
		inClusterClient,
		inClusterClient,
		currentNamespace,
		currentNamespace,
		translate.VClusterName,
		options,
	)
	if err != nil {
		return err
	}

	// build controller context
	controllerCtx, err := setup.NewControllerContext(
		ctx,
		options,
		currentNamespace,
		inClusterConfig,
		scheme.Scheme,
		pluginhookclient.NewPhysicalPluginClientFactory(blockingcacheclient.NewCacheClient),
		pluginhookclient.NewVirtualPluginClientFactory(blockingcacheclient.NewCacheClient),
	)
	if err != nil {
		return err
	}

	// start proxy
	err = setup.StartProxy(controllerCtx)
	if err != nil {
		return err
	}

	// start leader election + controllers
	err = StartLeaderElection(controllerCtx, func() error {
		return setup.StartControllers(controllerCtx)
	})
	if err != nil {
		return err
	}

	<-controllerCtx.StopChan
	return nil
}

func StartLeaderElection(ctx *options.ControllerContext, startLeading func() error) error {
	var err error
	if ctx.Options.LeaderElect {
		err = leaderelection.StartLeaderElection(ctx, scheme.Scheme, func() error {
			return startLeading()
		})
	} else {
		err = startLeading()
	}
	if err != nil {
		return errors.Wrap(err, "start controllers")
	}

	return nil
}
