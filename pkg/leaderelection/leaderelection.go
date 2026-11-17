package leaderelection

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/telemetry"
	"github.com/loft-sh/vcluster/pkg/util/osutil"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	clientv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
)

const (
	// EventReasonReplicaNotServing marks a control plane pod held out of the Service
	// because its backing store isn't shared with the leader.
	EventReasonReplicaNotServing = "ReplicaNotServing"

	releaseTimeout = 5 * time.Second
)

var leading atomic.Bool

// IsLeading reports whether this pod currently holds the controller lease.
func IsLeading() bool {
	return leading.Load()
}

// StartLeaderElection blocks until shutdown and runs `run` once this pod holds the lease.
// On SIGTERM the lease is released so a replacement pod doesn't wait for it to expire.
func StartLeaderElection(ctx *synccontext.ControllerContext, scheme *runtime.Scheme, run func() error) error {
	localConfig := ctx.HostManager.GetConfig()

	// create the event recorder
	recorderClient, err := kubernetes.NewForConfig(localConfig)
	if err != nil {
		return errors.Wrap(err, "create kubernetes client")
	}
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(func(format string, args ...interface{}) { klog.Infof(format, args...) })
	eventBroadcaster.StartRecordingToSink(&clientv1.EventSinkImpl{Interface: recorderClient.CoreV1().Events(ctx.Config.HostNamespace)})
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

	// The data lives on pod-0's volume, so no other StatefulSet pod may take over.
	if ctx.Config.OnlyLeaderServes() && statefulSetOrdinal(id, translate.VClusterName) > 0 {
		reportNotServing(ctx, leaderElectionClient, recorder, ctx.Config.HostNamespace, translate.VClusterName, translate.VClusterName+"-0", ctx.Config.BackingStoreType())
		<-ctx.Done()
		return nil
	}

	leaseName := translate.SafeConcatName("vcluster", translate.VClusterName, "controller")
	if ctx.Config.ControlPlane.StatefulSet.HighAvailability.Replicas <= 1 && leasesForbidden(ctx, leaderElectionClient, ctx.Config.HostNamespace, leaseName) {
		klog.Warning("service account may not access leases: running without leader election, upgrade the chart to match the vcluster image")
		leading.Store(true)
		return run()
	}

	// Lock required for leader election
	rl, err := resourcelock.New(
		resourcelock.LeasesResourceLock,
		ctx.Config.HostNamespace,
		leaseName,
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

	electionCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	released := make(chan struct{})
	osutil.RegisterPreInterruptHandler(func() {
		cancel()
		select {
		case <-released:
		case <-time.After(releaseTimeout):
		}
	})

	// try and become the leader and start controller manager loops
	leaderelection.RunOrDie(electionCtx, leaderelection.LeaderElectionConfig{
		Lock:            rl,
		ReleaseOnCancel: true,
		LeaseDuration:   time.Duration(ctx.Config.ControlPlane.StatefulSet.HighAvailability.LeaseDuration) * time.Second,
		RenewDeadline:   time.Duration(ctx.Config.ControlPlane.StatefulSet.HighAvailability.RenewDeadline) * time.Second,
		RetryPeriod:     time.Duration(ctx.Config.ControlPlane.StatefulSet.HighAvailability.RetryPeriod) * time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(_ context.Context) {
				klog.Info("Acquired leadership and run vcluster in leader mode")
				leading.Store(true)

				// start vcluster in leader mode
				err = run()
				if err != nil {
					klog.Error(err)
					osutil.Exit(1)
				}
			},
			OnStoppedLeading: func() {
				leading.Store(false)
				// Runs after the lease release, and osutil.Exit below waits for it.
				close(released)
				// Shutting down: the lease was released on purpose.
				if electionCtx.Err() != nil {
					return
				}
				klog.Info("leader election lost")

				// vcluster_error
				telemetry.CollectorControlPlane.RecordError(ctx, ctx.Config, telemetry.WarningSeverity, fmt.Errorf("leader election lost"))
				telemetry.CollectorControlPlane.Flush()

				osutil.Exit(1)
			},
			OnNewLeader: func(identity string) {
				if !shouldReportNotServing(identity, rl.Identity(), ctx.Config.OnlyLeaderServes()) {
					return
				}
				reportNotServing(ctx, leaderElectionClient, recorder, ctx.Config.HostNamespace, translate.VClusterName, identity, ctx.Config.BackingStoreType())
			},
		},
	})

	return nil
}

// leasesForbidden reports whether the Role denies reading the controller lease,
// which charts before leader election was unconditional do for a single replica.
func leasesForbidden(ctx context.Context, client kubernetes.Interface, namespace, name string) bool {
	_, err := client.CoordinationV1().Leases(namespace).Get(ctx, name, metav1.GetOptions{})
	return kerrors.IsForbidden(err)
}

// statefulSetOrdinal returns n for a StatefulSet pod named <name>-<n>, or -1 for any other pod.
func statefulSetOrdinal(pod, name string) int {
	suffix, ok := strings.CutPrefix(pod, name+"-")
	if !ok {
		return -1
	}
	ordinal, err := strconv.Atoi(suffix)
	if err != nil {
		return -1
	}
	return ordinal
}

// shouldReportNotServing is true when another pod leads and this pod is held out of the Service.
func shouldReportNotServing(leader, self string, onlyLeaderServes bool) bool {
	return leader != self && onlyLeaderServes
}

// reportNotServing warns that this pod stays out of the Service because another pod
// leads and the backing store can't be shared between them.
func reportNotServing(ctx context.Context, client kubernetes.Interface, recorder record.EventRecorder, namespace, name, leader string, store config.StoreType) {
	msg := fmt.Sprintf("control plane pod is not serving: %s leads and the %s backing store isn't shared between pods. "+
		"Scale the control plane through controlPlane.statefulSet.highAvailability.replicas with a shared backing store",
		leader, store)
	klog.Warning(msg)

	workload, err := controlPlaneWorkload(ctx, client, namespace, name)
	if err != nil {
		klog.Errorf("find control plane workload for %s event: %v", EventReasonReplicaNotServing, err)
		return
	}
	recorder.Event(workload, corev1.EventTypeWarning, EventReasonReplicaNotServing, msg)
}

// controlPlaneWorkload returns the StatefulSet or Deployment the chart renders for the control plane.
func controlPlaneWorkload(ctx context.Context, client kubernetes.Interface, namespace, name string) (runtime.Object, error) {
	sts, err := client.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err == nil {
		return sts, nil
	}
	if !kerrors.IsNotFound(err) {
		return nil, err
	}

	deploy, err := client.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return deploy, nil
}
