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
			vClusterConfig, err := config.ParseConfig(startOptions.Config, os.Getenv("VCLUSTER_NAME"), startOptions.SetValues)
			if err != nil {
				return err
			}

			// execute command
			return ExecuteStart(cobraCmd.Context(), vClusterConfig)
		},
	}

	cmd.Flags().StringVar(&startOptions.Config, "config", "/var/vcluster/config.yaml", "The path where to find the vCluster config to load")
	cmd.Flags().StringArrayVar(&startOptions.SetValues, "set", []string{}, "Set values for the config. E.g. --set 'exportKubeConfig.secret.name=my-name'")
	return cmd
}

func ExecuteStart(ctx context.Context, vConfig *config.VirtualClusterConfig) error {
	// set global vCluster name
	translate.VClusterName = vConfig.Name

	// set service name
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

	// set target namespace
	vConfig.TargetNamespace = workloadNamespace
	if vConfig.Experimental.SyncSettings.TargetNamespace != "" {
		vConfig.TargetNamespace = vConfig.Experimental.SyncSettings.TargetNamespace
	}

	// init telemetry
	telemetry.Collector.Init(controlPlaneConfig, controlPlaneNamespace, vConfig)

	// initialize feature gate from environment
	err = pro.LicenseInit(ctx, controlPlaneConfig, controlPlaneNamespace, vConfig.Platform.APIKey.Value, vConfig.Platform.APIKey.SecretRef.Namespace, vConfig.Platform.APIKey.SecretRef.Name)
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
