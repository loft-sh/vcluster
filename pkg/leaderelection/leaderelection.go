package leaderelection

import (
	"context"
	"os"
	"time"

	context2 "github.com/loft-sh/vcluster/cmd/vcluster/context"
	"github.com/loft-sh/vcluster/pkg/telemetry"
	telemetrytypes "github.com/loft-sh/vcluster/pkg/telemetry/types"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	clientv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
)

func StartLeaderElection(ctx *context2.ControllerContext, scheme *runtime.Scheme, run func() error) error {
	localConfig := ctx.LocalManager.GetConfig()

	// create the event recorder
	recorderClient, err := kubernetes.NewForConfig(localConfig)
	if err != nil {
		return errors.Wrap(err, "create kubernetes client")
	}
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(func(format string, args ...interface{}) { klog.Infof(format, args...) })
	eventBroadcaster.StartRecordingToSink(&clientv1.EventSinkImpl{Interface: recorderClient.CoreV1().Events(ctx.CurrentNamespace)})
	recorder := eventBroadcaster.NewRecorder(scheme, corev1.EventSource{Component: "vcluster"})

	// create the leader election client
	leaderElectionClient, err := kubernetes.NewForConfig(rest.AddUserAgent(localConfig, "leader-election"))
	if err != nil {
		return errors.Wrap(err, "create leader election client")
	}

	// Identity used to distinguish between multiple controller manager instances
	id, err := os.Hostname()
	if err != nil {
		return err
	}

	// Lock required for leader election
	rl, err := resourcelock.New(
		resourcelock.ConfigMapsLeasesResourceLock,
		ctx.CurrentNamespace,
		translate.SafeConcatName("vcluster", translate.Suffix, "controller"),
		leaderElectionClient.CoreV1(),
		leaderElectionClient.CoordinationV1(),
		resourcelock.ResourceLockConfig{
			Identity:      id + "-external-vcluster-controller",
			EventRecorder: recorder,
		},
	)
	if err != nil {
		return err
	}

	// try and become the leader and start controller manager loops
	leaderelection.RunOrDie(ctx.Context, leaderelection.LeaderElectionConfig{
		Lock:          rl,
		LeaseDuration: time.Duration(ctx.Options.LeaseDuration) * time.Second,
		RenewDeadline: time.Duration(ctx.Options.RenewDeadline) * time.Second,
		RetryPeriod:   time.Duration(ctx.Options.RetryPeriod) * time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				klog.Info("Acquired leadership and run vcluster in leader mode")
				if telemetry.Collector.IsEnabled() {
					telemetry.Collector.RecordEvent(telemetry.Collector.NewEvent(telemetrytypes.EventLeadershipStarted))
				}

				// start vcluster in leader mode
				err = run()
				if err != nil {
					klog.Fatal(err)
				}
			},
			OnStoppedLeading: func() {
				klog.Info("leader election lost")
				if telemetry.Collector.IsEnabled() {
					telemetry.Collector.RecordEvent(telemetry.Collector.NewEvent(telemetrytypes.EventLeadershipStopped))
				}
				//TODO: force telemetry upload
				os.Exit(1)
			},
		},
	})

	return nil
}
