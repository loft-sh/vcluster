package cmd

import (
	"context"
	"fmt"
	"os"
	"runtime/debug"

	"github.com/go-logr/logr"
	vconfig "github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/leaderelection"
	"github.com/loft-sh/vcluster/pkg/plugin"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/scheme"
	"github.com/loft-sh/vcluster/pkg/setup"
	"github.com/loft-sh/vcluster/pkg/telemetry"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
)

type StartOptions struct {
	Config string

	SetValues []string
}

func NewStartCommand() *cobra.Command {
	startOptions := &StartOptions{}
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Execute the vcluster",
		Args:  cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, _ []string) (err error) {
			// execute command
			return ExecuteStart(cobraCmd.Context(), startOptions)
		},
	}

	cmd.Flags().StringVar(&startOptions.Config, "config", "/var/vcluster/config.yaml", "The path where to find the vCluster config to load")

	// Should only be used for development
	cmd.Flags().StringArrayVar(&startOptions.SetValues, "set", []string{}, "Set values for the config. E.g. --set 'exportKubeConfig.secret.name=my-name'")
	_ = cmd.Flags().MarkHidden("set")
	return cmd
}

func ExecuteStart(ctx context.Context, options *StartOptions) error {
	// parse vCluster config
	vConfig, err := config.ParseConfig(options.Config, os.Getenv("VCLUSTER_NAME"), options.SetValues)
	if err != nil {
		return err
	}

	if vconfig.ShouldCheckForProFeatures() && vConfig.IsProFeatureEnabled() {
		log, err := logr.FromContext(ctx)
		if err != nil {
			return err
		}

		log.Info("In order to use a Pro feature, please contact us at https://www.vcluster.com/pro-demo or downgrade by running `vcluster upgrade --version v0.19.5`")
		os.Exit(1)
	}

	// get current namespace
	vConfig.ControlPlaneConfig, vConfig.ControlPlaneNamespace, vConfig.ControlPlaneService, vConfig.WorkloadConfig, vConfig.WorkloadNamespace, vConfig.WorkloadService, err = pro.GetRemoteClient(vConfig)
	if err != nil {
		return err
	}

	// init config
	err = setup.InitAndValidateConfig(ctx, vConfig)
	if err != nil {
		return err
	}

	// start telemetry
	telemetry.StartControlPlane(vConfig)
	defer telemetry.CollectorControlPlane.Flush()

	// capture errors
	defer func() {
		if r := recover(); r != nil {
			telemetry.CollectorControlPlane.RecordError(ctx, vConfig, telemetry.PanicSeverity, fmt.Errorf("panic: %v %s", r, string(debug.Stack())))
			panic(r)
		} else if err != nil {
			telemetry.CollectorControlPlane.RecordError(ctx, vConfig, telemetry.FatalSeverity, err)
		}
	}()

	// initialize feature gate from environment
	err = pro.LicenseInit(ctx, vConfig)
	if err != nil {
		return fmt.Errorf("init license: %w", err)
	}

	// set features for plugins to recognize
	plugin.DefaultManager.SetProFeatures(pro.LicenseFeatures())

	// check if we should create certs
	err = setup.Initialize(ctx, vConfig)
	if err != nil {
		return fmt.Errorf("initialize: %w", err)
	}

	// build controller context
	controllerCtx, err := setup.NewControllerContext(ctx, vConfig)
	if err != nil {
		return fmt.Errorf("create controller context: %w", err)
	}

	// start proxy
	err = setup.StartProxy(controllerCtx)
	if err != nil {
		return fmt.Errorf("start proxy: %w", err)
	}

	// should start embedded coredns?
	if vConfig.ControlPlane.CoreDNS.Embedded {
		// write vCluster kubeconfig to /data/vcluster/admin.conf
		err = clientcmd.WriteToFile(*controllerCtx.VirtualRawConfig, "/data/vcluster/admin.conf")
		if err != nil {
			return fmt.Errorf("write vCluster kube config for embedded coredns: %w", err)
		}

		// start embedded coredns
		err = pro.StartIntegratedCoreDNS(controllerCtx)
		if err != nil {
			return fmt.Errorf("start integrated core dns: %w", err)
		}
	}

	if err := pro.ConnectToPlatform(
		ctx,
		vConfig,
		controllerCtx.VirtualManager.GetHTTPClient().Transport,
	); err != nil {
		return fmt.Errorf("connect to platform: %w", err)
	}

	// start leader election + controllers
	err = StartLeaderElection(controllerCtx, func() error {
		return setup.StartControllers(controllerCtx)
	})
	if err != nil {
		return fmt.Errorf("start controllers: %w", err)
	}

	<-controllerCtx.StopChan
	return nil
}

func StartLeaderElection(ctx *config.ControllerContext, startLeading func() error) error {
	var err error
	if ctx.Config.ControlPlane.StatefulSet.HighAvailability.Replicas > 1 {
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
