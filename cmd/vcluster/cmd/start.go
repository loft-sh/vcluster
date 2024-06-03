package cmd

import (
	"context"
	"fmt"
	"os"
	"runtime/debug"

	"github.com/loft-sh/vcluster/pkg/leaderelection"
	"github.com/loft-sh/vcluster/pkg/options"
	"github.com/loft-sh/vcluster/pkg/plugin"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/scheme"
	"github.com/loft-sh/vcluster/pkg/setup"
	"github.com/loft-sh/vcluster/pkg/telemetry"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

func NewStartCommand() *cobra.Command {
	vClusterOptions := &options.VirtualClusterOptions{}
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

			// execute command
			return ExecuteStart(cobraCmd.Context(), vClusterOptions)
		},
	}

	options.AddFlags(cmd.Flags(), vClusterOptions)
	pro.AddProFlags(cmd.Flags(), vClusterOptions)
	return cmd
}

func ExecuteStart(ctx context.Context, options *options.VirtualClusterOptions) error {
	err := pro.ValidateProOptions(options)
	if err != nil {
		return err
	}

	// set suffix
	translate.VClusterName = options.Name
	if translate.VClusterName == "" {
		translate.VClusterName = options.DeprecatedSuffix
	}
	if translate.VClusterName == "" {
		translate.VClusterName = "vcluster"
	}

	// set service name
	if options.ServiceName == "" {
		options.ServiceName = translate.VClusterName
	}

	// get current namespace
	controlPlaneConfig, controlPlaneNamespace, controlPlaneService, workloadConfig, workloadNamespace, workloadService, err := pro.GetRemoteClient(options)
	if err != nil {
		return err
	}
	options.ServiceName = workloadService
	err = os.Setenv("NAMESPACE", workloadNamespace)
	if err != nil {
		return fmt.Errorf("set NAMESPACE env var: %w", err)
	}

	// init telemetry
	telemetry.Collector.Init(controlPlaneConfig, controlPlaneNamespace, options)

	// initialize feature gate from environment
	err = pro.LicenseInit(ctx, controlPlaneConfig, controlPlaneNamespace, options.ProOptions.ProLicenseSecret)
	if err != nil {
		return fmt.Errorf("init license: %w", err)
	}

	// set features for plugins to recognize
	plugin.DefaultManager.SetProFeatures(pro.LicenseFeatures())

	// get host cluster config and tweak rate-limiting configuration
	workloadClient, err := kubernetes.NewForConfig(workloadConfig)
	if err != nil {
		return err
	}
	controlPlaneClient, err := kubernetes.NewForConfig(controlPlaneConfig)
	if err != nil {
		return err
	}

	// // set global owner for use in owner references
	err = SetGlobalOwner(
		ctx,
		workloadClient,
		options.MultiNamespaceMode,
		controlPlaneNamespace,
		options.TargetNamespace,
		options.SetOwner,
		options.ServiceName,
	)
	if err != nil {
		return errors.Wrap(err, "finding vcluster pod owner")
	}

	// check if we should create certs
	err = setup.Initialize(
		ctx,
		workloadClient,
		controlPlaneClient,
		workloadNamespace,
		controlPlaneNamespace,
		translate.VClusterName,
		options,
	)
	if err != nil {
		return fmt.Errorf("initialize: %w", err)
	}

	// build controller context
	controllerCtx, err := setup.NewControllerContext(
		ctx,
		options,
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
	if controllerCtx.Options.ProOptions.IntegratedCoredns {
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

// SetGlobalOwner fetches the owning service and populates in translate.Owner if: the vcluster is configured to setOwner is,
// and if the currentNamespace == targetNamespace (because cross namespace owner refs don't work).
func SetGlobalOwner(ctx context.Context, currentNamespaceClient kubernetes.Interface, multins bool, currentNamespace, targetNamespace string, setOwner bool, serviceName string) error {
	if !setOwner {
		return nil
	}

	if multins {
		klog.Warningf("Skip setting owner, because multi namespace mode is enabled")

		return nil
	}

	// this might be called before target namespace is defaulted to current namespace
	if targetNamespace != "" && currentNamespace != targetNamespace {
		klog.Warningf("Skip setting owner, because current namespace %s != target namespace %s", currentNamespace, targetNamespace)

		return nil
	}

	service, err := currentNamespaceClient.CoreV1().Services(currentNamespace).Get(ctx, serviceName, metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "get vcluster service")
	}
	// client doesn't populate typemeta
	service.TypeMeta.APIVersion = "v1"
	service.TypeMeta.Kind = "Service"
	translate.Owner = service
	return nil
}
