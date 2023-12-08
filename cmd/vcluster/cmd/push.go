package cmd

import (
	"context"
	"fmt"
	"runtime/debug"

	"github.com/loft-sh/vcluster/pkg/oci"
	"github.com/loft-sh/vcluster/pkg/telemetry"
	"github.com/loft-sh/vcluster/pkg/util/clienthelper"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
)

type PushCommand struct {
	Username string
	Password string

	Destination string
}

func NewPushCommand() *cobra.Command {
	pushCommand := &PushCommand{}
	cmd := &cobra.Command{
		Use:   "push",
		Short: "Push the vcluster to a OCI registry",
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
			return pushCommand.Execute(cobraCmd.Context())
		},
	}

	cmd.Flags().StringVar(&pushCommand.Destination, "destination", "", "The destination where to push the vCluster to")
	cmd.Flags().StringVar(&pushCommand.Username, "username", "", "The username to use to push to the destination")
	cmd.Flags().StringVar(&pushCommand.Password, "password", "", "The password to use to push to the destination")
	return cmd
}

func (cmd *PushCommand) Execute(ctx context.Context) error {
	// check flags
	if cmd.Destination == "" {
		return fmt.Errorf("please specify a destination for the vCluster, e.g. ghcr.io/my-user/my-repo")
	}

	// make sure global vCluster name is correct
	translate.ReadSuffix()

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

	// push vCluster to registry
	err = oci.Push(ctx, inClusterClient, currentNamespace, cmd.Destination, cmd.Username, cmd.Password, scheme)
	if err != nil {
		return err
	}

	return nil
}
