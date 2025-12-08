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
	setupconfig "github.com/loft-sh/vcluster/pkg/setup/config"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/telemetry"
	"github.com/loft-sh/vcluster/pkg/util/osutil"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
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
	logger := log.GetInstance()
	logger.Infof("vCluster version: %s", telemetry.SyncerVersion)
	if os.Getenv("POD_NAME") == "" && os.Getenv("POD_NAMESPACE") == "" {
		return pro.StartStandalone(ctx, &pro.StandaloneOptions{
			Config: options.Config,
		})
	}

	return StartInCluster(ctx, options)
}

// StartInCluster is invoked when running in a container
func StartInCluster(ctx context.Context, options *StartOptions) error {
	vClusterName := os.Getenv("VCLUSTER_NAME")
	// parse vCluster config
	vConfig, err := config.ParseConfig(options.Config, vClusterName, options.SetValues)
	if err != nil {
		return err
	}

	// get current namespace
	vConfig.HostConfig, vConfig.HostNamespace, err = setupconfig.InitClientConfig()
	if err != nil {
		return err
	}

	// init config
	err = setupconfig.InitAndValidateConfig(ctx, vConfig)
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
			klog.Errorf("panic: %v %s", r, string(debug.Stack()))
			osutil.Exit(1)
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
	vClusters, err := find.ListVClusters(ctx, "", "", vConfig.HostNamespace, logger)
	if err != nil {
		return err
	}

	// from v0.25 onwards, creation of multiple vClusters inside the same ns is not allowed
	for _, v := range vClusters {
		if v.Namespace == vConfig.HostNamespace && v.Name != vClusterName {
			return fmt.Errorf("there is already a virtual cluster in namespace %s; "+
				"creating multiple virtual clusters inside the same namespace is not supported", vConfig.HostNamespace)
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

	// start konnectivity server
	err = pro.StartKonnectivity(controllerCtx)
	if err != nil {
		return fmt.Errorf("start konnectivity: %w", err)
	}

	// should start embedded coredns?
	if vConfig.ControlPlane.CoreDNS.Embedded {
		// write vCluster kubeconfig to /data/vcluster/admin.conf
		err = clientcmd.WriteToFile(*controllerCtx.VirtualRawConfig, constants.EmbeddedCoreDNSAdminConf)
		if err != nil {
			return fmt.Errorf("write vCluster kube config for embedded coredns: %w", err)
		}

		// start embedded coredns
		err = pro.StartIntegratedCoreDNS(controllerCtx)
		if err != nil {
			return fmt.Errorf("start integrated core dns: %w", err)
		}
	}

	// start embedded kube-vip
	if vConfig.ControlPlane.Advanced.KubeVip.Enabled {
		if err := pro.StartEmbeddedKubeVip(controllerCtx, vConfig); err != nil {
			return fmt.Errorf("start embedded kube-vip: %w", err)
		}
	}

	// Check if any proxy resources are enabled
	if len(vConfig.Proxy.Resources) > 0 {
		if err := pro.StartResourceProxy(controllerCtx, vConfig); err != nil {
			return fmt.Errorf("start resource proxy: %w", err)
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
