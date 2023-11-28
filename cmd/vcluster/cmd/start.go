package cmd

import (
	"context"
	"fmt"
	"runtime/debug"

	volumesnapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	"github.com/loft-sh/vcluster/pkg/apis"
	"github.com/loft-sh/vcluster/pkg/leaderelection"
	"github.com/loft-sh/vcluster/pkg/setup"
	"github.com/loft-sh/vcluster/pkg/setup/options"
	"github.com/loft-sh/vcluster/pkg/telemetry"
	"github.com/loft-sh/vcluster/pkg/util/clienthelper"
	"github.com/loft-sh/vcluster/pkg/util/translate"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	apiregistrationv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)
	// API extensions are not in the above scheme set,
	// and must thus be added separately.
	_ = apiextensionsv1beta1.AddToScheme(scheme)
	_ = apiextensionsv1.AddToScheme(scheme)
	_ = apiregistrationv1.AddToScheme(scheme)

	// Register the fake conversions
	_ = apis.RegisterConversions(scheme)

	// Register VolumeSnapshot CRDs
	_ = volumesnapshotv1.AddToScheme(scheme)
}

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
	translate.Suffix = options.Name
	if translate.Suffix == "" {
		translate.Suffix = options.DeprecatedSuffix
	}
	if translate.Suffix == "" {
		translate.Suffix = "vcluster"
	}

	// set service name
	if options.ServiceName == "" {
		options.ServiceName = translate.Suffix
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
		translate.Suffix,
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
		scheme,
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
		err = leaderelection.StartLeaderElection(ctx, scheme, func() error {
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
