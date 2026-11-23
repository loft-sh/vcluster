package telemetry

import (
	"context"
	"encoding/json"
	"errors"
	"runtime"
	"strconv"
	"time"

	"github.com/loft-sh/analytics-client/client"
	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/util/clihelper"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

var CollectorControlPlane ControlPlaneCollector = &noopCollector{}

// ErrLeaderElectionLost is what RecordError is given when the control plane
// stops because it lost the leader election.
var ErrLeaderElectionLost = errors.New("leader election lost")

// isInstanceCreator accepts what the CLI and the platform write to
// telemetry.instanceCreator.
var isInstanceCreator = oneOf("vclusterctl", "loft")

type ControlPlaneCollector interface {
	RecordStart(ctx context.Context, config *config.VirtualClusterConfig)

	// RecordError records the kind of err, never its text. With PanicSeverity
	// it has to be called from the deferred function that recovered the panic,
	// because it records the functions on the stack of its caller.
	RecordError(ctx context.Context, config *config.VirtualClusterConfig, severity ErrorSeverityType, err error)

	// Flush makes sure all events are sent to the backend
	Flush()

	SetVirtualClient(virtualClient kubernetes.Interface)
}

func StartControlPlane(config *config.VirtualClusterConfig) {
	if !config.Telemetry.Enabled || SyncerVersion == "dev" || config.ControlPlane.Standalone.Enabled {
		return
	}

	// create a new default collector
	collector, err := newControlPlaneCollector(config)
	if err != nil {
		// Log the problem but don't fail - use disabled Collector instead
		loghelper.New("telemetry").Infof("%s", err.Error())
	} else {
		CollectorControlPlane = collector
	}
}

func newControlPlaneCollector(config *config.VirtualClusterConfig) (*controlPlaneCollector, error) {
	collector := &controlPlaneCollector{
		analyticsClient: client.NewClient(),

		log: loghelper.New("telemetry"),

		hostClient:    config.HostClient,
		hostNamespace: config.HostNamespace,
		hostService:   config.Name,
	}

	go collector.startReportStatus(context.Background(), config.Telemetry.PlatformUserID, config.Telemetry.PlatformInstanceID, config.Telemetry.MachineID)
	return collector, nil
}

type controlPlaneCollector struct {
	analyticsClient client.Client

	log loghelper.Logger

	vClusterID                cachedValue[string]
	hostClusterVersion        cachedValue[*KubernetesVersion]
	virtualClusterVersion     cachedValue[*KubernetesVersion]
	vClusterCreationTimestamp cachedValue[int64]

	hostClient    kubernetes.Interface
	hostNamespace string
	hostService   string

	// everything below will be set during runtime
	virtualClient kubernetes.Interface
}

func (d *controlPlaneCollector) startReportStatus(ctx context.Context, platformUserID, platformInstanceID, machineID string) {
	time.Sleep(time.Second * 30)

	wait.Until(func() {
		d.RecordStatus(ctx, platformUserID, platformInstanceID, machineID)
	}, time.Minute*5, ctx.Done())
}

func (d *controlPlaneCollector) SetVirtualClient(virtualClient kubernetes.Interface) {
	d.virtualClient = virtualClient
}

func (d *controlPlaneCollector) Flush() {
	d.analyticsClient.Flush()
}

func (d *controlPlaneCollector) RecordStatus(ctx context.Context, platformUserID, platformInstanceID, machineID string) {
	properties := d.getMetrics(ctx)

	// build the event and record
	propertiesRaw, _ := json.Marshal(properties)
	d.analyticsClient.RecordEvent(client.Event{
		"event": {
			"type":                 "vcluster_status",
			"vcluster_id":          d.getVClusterID(ctx),
			"platform_user_id":     platformUserID,
			"platform_instance_id": platformInstanceID,
			"machine_id":           machineID,
			"properties":           string(propertiesRaw),
			"timestamp":            time.Now().Unix(),
		},
	})
}

func (d *controlPlaneCollector) RecordStart(ctx context.Context, config *config.VirtualClusterConfig) {
	if !config.Telemetry.Enabled || config.ControlPlane.Standalone.Enabled {
		return
	}

	creationMethod := config.Telemetry.InstanceCreator
	if creationMethod != "" && !isInstanceCreator(creationMethod) {
		creationMethod = "other"
	}
	properties := map[string]interface{}{
		"vcluster_version":            SyncerVersion,
		"vcluster_k8s_distro_version": d.getVirtualClusterVersion(),
		"host_cluster_k8s_version":    d.getHostClusterVersion(),
		"os_arch":                     runtime.GOOS + "/" + runtime.GOARCH,
		"creation_method":             creationMethod,
		"creation_timestamp":          d.getVClusterCreationTimestamp(ctx),
		"vcluster_k8s_distro":         translate.VClusterName,
		"helm_values":                 configSummary(&config.Config),
	}

	// build the event and record
	propertiesRaw, _ := json.Marshal(properties)
	d.analyticsClient.RecordEvent(client.Event{
		"event": {
			"type":                 "vcluster_start",
			"vcluster_id":          d.getVClusterID(ctx),
			"platform_user_id":     config.Telemetry.PlatformUserID,
			"platform_instance_id": config.Telemetry.PlatformInstanceID,
			"machine_id":           config.Telemetry.MachineID,
			"timestamp":            time.Now().Unix(),
		},
		"vcluster_instance": {
			"vcluster_id":          d.getVClusterID(ctx),
			"platform_instance_id": config.Telemetry.PlatformInstanceID,
			"properties":           string(propertiesRaw),
			"timestamp":            time.Now().Unix(),
		},
	})
}

func (d *controlPlaneCollector) RecordError(ctx context.Context, config *config.VirtualClusterConfig, severity ErrorSeverityType, err error) {
	if !config.Telemetry.Enabled || config.ControlPlane.Standalone.Enabled {
		return
	}

	properties := map[string]interface{}{
		"vcluster_version": SyncerVersion,
		"severity":         string(severity),
		"message":          errorCategory(err),
	}
	if severity == PanicSeverity || severity == FatalSeverity {
		properties["helm_values"] = configSummary(&config.Config)
	}
	if severity == PanicSeverity {
		properties["stack"] = callerFunctions()
	}

	// build the event and record
	propertiesRaw, _ := json.Marshal(properties)
	d.analyticsClient.RecordEvent(client.Event{
		"event": {
			"type":                 "vcluster_error",
			"vcluster_id":          d.getVClusterID(ctx),
			"platform_user_id":     config.Telemetry.PlatformUserID,
			"platform_instance_id": config.Telemetry.PlatformInstanceID,
			"machine_id":           config.Telemetry.MachineID,
			"properties":           string(propertiesRaw),
			"timestamp":            time.Now().Unix(),
		},
	})
}

func (d *controlPlaneCollector) getVirtualClusterVersion() *KubernetesVersion {
	virtualVersion, err := d.virtualClusterVersion.Get(func() (*KubernetesVersion, error) {
		return getKubernetesVersion(d.virtualClient)
	})
	if err != nil {
		klog.V(1).ErrorS(err, "Error retrieving virtual cluster version")
	}

	return virtualVersion
}

func (d *controlPlaneCollector) getHostClusterVersion() *KubernetesVersion {
	if d.hostClient == nil {
		return nil
	}

	hostVersion, err := d.hostClusterVersion.Get(func() (*KubernetesVersion, error) {
		return getKubernetesVersion(d.hostClient)
	})
	if err != nil {
		klog.V(1).ErrorS(err, "Error retrieving host cluster version")
	}

	return hostVersion
}

func (d *controlPlaneCollector) getVClusterID(ctx context.Context) string {
	if d.hostClient == nil {
		return ""
	}

	vClusterID, err := d.vClusterID.Get(func() (string, error) {
		return getVClusterID(ctx, d.hostClient, d.hostNamespace, d.hostService)
	})
	if err != nil {
		klog.V(1).ErrorS(err, "Error retrieving vClusterID")
	}

	return vClusterID
}

func (d *controlPlaneCollector) getVClusterCreationTimestamp(ctx context.Context) int64 {
	if d.hostClient == nil {
		return 0
	}

	vClusterCreationTimestamp, err := d.vClusterCreationTimestamp.Get(func() (int64, error) {
		return getVClusterCreationTimestamp(ctx, d.hostClient, d.hostNamespace, d.hostService)
	})
	if err != nil {
		klog.V(1).ErrorS(err, "Error retrieving vClusterCreationTimestamp")
	}

	return vClusterCreationTimestamp
}

func (d *controlPlaneCollector) getMetrics(ctx context.Context) map[string]interface{} {
	// maximum 20 seconds
	ctx, cancel := context.WithTimeout(ctx, time.Second*20)
	defer cancel()

	// metrics map
	retMap := map[string]interface{}{}
	if d.virtualClient == nil {
		return retMap
	}

	// list pods
	podList, err := d.virtualClient.CoreV1().Pods(corev1.NamespaceAll).List(ctx, metav1.ListOptions{})
	if err == nil {
		retMap["pods"] = len(podList.Items)

		failingPods := 0
		for _, pod := range podList.Items {
			if clihelper.HasPodProblem(&pod) {
				failingPods++
			}
		}
		retMap["pods_failing"] = failingPods
	}

	// list namespaces
	namespaceList, err := d.virtualClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err == nil {
		retMap["namespaces"] = len(namespaceList.Items)
	}

	return retMap
}

// callerFunctions names the functions on the stack of the goroutine, starting
// at the caller of the function that calls it, each with its line. Unlike a
// stack trace it has no argument values and no panic message.
func callerFunctions() []string {
	programCounters := make([]uintptr, 64)
	frames := runtime.CallersFrames(programCounters[:runtime.Callers(3, programCounters)])

	functions := []string{}
	for {
		frame, more := frames.Next()
		functions = append(functions, frame.Function+":"+strconv.Itoa(frame.Line))
		if !more {
			return functions
		}
	}
}
