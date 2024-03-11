package cmd

import (
	"context"
	"fmt"
	"os"
	"runtime/debug"

	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/leaderelection"
	"github.com/loft-sh/vcluster/pkg/plugin"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/scheme"
	"github.com/loft-sh/vcluster/pkg/setup"
	"github.com/loft-sh/vcluster/pkg/telemetry"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
)

type StartOptions struct {
	Config string
}

func NewStartCommand() *cobra.Command {
	startOptions := &StartOptions{}
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Execute the vcluster",
		Args:  cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, _ []string) (err error) {
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

			// parse vCluster config
			vClusterConfig, err := config.ParseConfig(startOptions.Config, os.Getenv("VCLUSTER_NAME"))
			if err != nil {
				return err
			}

			// execute command
			return ExecuteStart(cobraCmd.Context(), vClusterConfig)
		},
	}

	cmd.Flags().StringVar(&startOptions.Config, "config", "", "The path where to find the vCluster config to load")
	return cmd
}

func ExecuteStart(ctx context.Context, vConfig *config.VirtualClusterConfig) error {
	// set global vCluster name
	if translate.VClusterName == "" {
		translate.VClusterName = vConfig.Name
	}

	// set service name
	if vConfig.ServiceName == "" {
		vConfig.ServiceName = translate.VClusterName
	}
	if vConfig.ControlPlane.Advanced.WorkloadServiceAccount.Name == "" {
		vConfig.ControlPlane.Advanced.WorkloadServiceAccount.Name = "vc-workload-" + vConfig.Name
	}

	// get current namespace
	controlPlaneConfig, controlPlaneNamespace, controlPlaneService, workloadConfig, workloadNamespace, workloadService, err := pro.GetRemoteClient(vConfig)
	if err != nil {
		return err
	}
	vConfig.ServiceName = workloadService
	err = os.Setenv("NAMESPACE", workloadNamespace)
	if err != nil {
		return fmt.Errorf("set NAMESPACE env var: %w", err)
	}

	// init telemetry
	telemetry.Collector.Init(controlPlaneConfig, controlPlaneNamespace, vConfig)

	// initialize feature gate from environment
	err = pro.LicenseInit(ctx, controlPlaneConfig, controlPlaneNamespace, vConfig.Platform.ApiKey.Value, vConfig.Platform.ApiKey.SecretRef.Namespace, vConfig.Platform.ApiKey.SecretRef.Name)
	if err != nil {
		return fmt.Errorf("init license: %w", err)
	}

	// set features for plugins to recognize
	plugin.DefaultManager.SetProFeatures(pro.LicenseFeatures())

	// get host cluster config and tweak rate-limiting configuration
	controlPlaneClient, err := kubernetes.NewForConfig(controlPlaneConfig)
	if err != nil {
		return err
	}

	// check if we should create certs
	err = setup.Initialize(
		ctx,
		controlPlaneClient,
		controlPlaneNamespace,
		translate.VClusterName,
		vConfig,
	)
	if err != nil {
		return fmt.Errorf("initialize: %w", err)
	}

	// build controller context
	controllerCtx, err := setup.NewControllerContext(
		ctx,
		vConfig,
		workloadNamespace,
		workloadConfig,
		scheme.Scheme,
	)
	if err != nil {
		return fmt.Errorf("create controller context: %w", err)
	}

	// start proxy
	err = setup.StartProxy(
		controllerCtx,
		controlPlaneNamespace,
		controlPlaneService,
		controlPlaneClient,
	)
	if err != nil {
		return fmt.Errorf("start proxy: %w", err)
	}

	// start integrated coredns
	if vConfig.ControlPlane.CoreDNS.Embedded {
		err = pro.StartIntegratedCoreDNS(controllerCtx)
		if err != nil {
			return fmt.Errorf("start integrated core dns: %w", err)
		}
	}

	// start leader election + controllers
	err = StartLeaderElection(controllerCtx, func() error {
		return setup.StartControllers(controllerCtx, controlPlaneNamespace, controlPlaneService, controlPlaneConfig)
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
