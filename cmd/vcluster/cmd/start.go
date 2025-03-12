package cmd

import (
	"context"
	"fmt"
	"os"
	"runtime/debug"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/find"
	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/integrations"
	"github.com/loft-sh/vcluster/pkg/leaderelection"
	"github.com/loft-sh/vcluster/pkg/plugin"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/scheme"
	"github.com/loft-sh/vcluster/pkg/setup"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
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

	cmd.Flags().StringVar(&startOptions.Config, "config", constants.DefaultVClusterConfigLocation, "The path where to find the vCluster config to load")

	// Should only be used for development
	cmd.Flags().StringArrayVar(&startOptions.SetValues, "set", []string{}, "Set values for the config. E.g. --set 'exportKubeConfig.secret.name=my-name'")
	_ = cmd.Flags().MarkHidden("set")
	return cmd
}

func ExecuteStart(ctx context.Context, options *StartOptions) error {
	vClusterName := os.Getenv("VCLUSTER_NAME")
	// parse vCluster config
	vConfig, err := config.ParseConfig(options.Config, vClusterName, options.SetValues)
	if err != nil {
		return err
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
	if err := pro.LicenseInit(ctx, vConfig); err != nil {
		return fmt.Errorf("license init: %w", err)
	}

	logger := log.GetInstance()
	// check if there are existing vClusters in the current namespace
	vClusters, err := find.ListVClusters(ctx, "", "", vConfig.ControlPlaneNamespace, logger)
	if err != nil {
		return err
	}
	var vClusterExists bool
	for _, v := range vClusters {
		if v.Namespace == vConfig.ControlPlaneNamespace && v.Name != vClusterName {
			vClusterExists = true
			break
		}
	}
	// add a deprecation warning for multiple vCluster creation scenario
	if vClusterExists {
		logger.Warnf("Please note that creating multiple virtual clusters in the same namespace " +
			"and the 'reuseNamespace' config are deprecated and will be removed soon.")

		// throw an error if reuseNamespace config is not set
		if !vConfig.Experimental.ReuseNamespace {
			return fmt.Errorf("there is already a virtual cluster in namespace %s. To create multiple virtual clusters "+
				"within the same namespace, it is mandatory to set 'reuse-namespace' to true in vCluster config", vConfig.ControlPlaneNamespace)
		}
	}

	err = setup.Initialize(ctx, vConfig)
	if err != nil {
		return fmt.Errorf("initialize: %w", err)
	}

	// set features for plugins to recognize
	plugin.DefaultManager.SetProFeatures(pro.LicenseFeatures())

	// build controller context
	controllerCtx, err := setup.NewControllerContext(ctx, vConfig)
	if err != nil {
		return fmt.Errorf("create controller context: %w", err)
	}

	// start license loader
	err = pro.LicenseStart(controllerCtx)
	if err != nil {
		return fmt.Errorf("start license loader: %w", err)
	}

	err = pro.CheckFeatures(controllerCtx)
	if err != nil {
		return fmt.Errorf("pro features check: %w", err)
	}

	// start integrations
	err = integrations.StartIntegrations(controllerCtx)
	if err != nil {
		return fmt.Errorf("start integrations: %w", err)
	}

	// start managers
	syncers, err := setup.StartManagers(controllerCtx.ToRegisterContext())
	if err != nil {
		return fmt.Errorf("start managers: %w", err)
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

	// start leader election + controllers
	err = StartLeaderElection(controllerCtx, func() error {
		return setup.StartControllers(controllerCtx, syncers)
	})
	if err != nil {
		return fmt.Errorf("start controllers: %w", err)
	}

	<-controllerCtx.StopChan
	return nil
}

func StartLeaderElection(ctx *synccontext.ControllerContext, startLeading func() error) error {
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
