package telemetry

import (
	"context"
	"encoding/json"
	"os"
	"runtime"
	"time"

	"github.com/loft-sh/analytics-client/client"
	managementv1 "github.com/loft-sh/api/v3/pkg/apis/management/v1"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/options"
	"github.com/loft-sh/vcluster/pkg/upgrade"
	"github.com/loft-sh/vcluster/pkg/util/cliconfig"
	"github.com/loft-sh/vcluster/pkg/util/clihelper"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"

	"github.com/loft-sh/vcluster/pkg/util/loghelper"
)

type ErrorSeverityType string

const (
	ConfigEnvVar = "VCLUSTER_TELEMETRY_CONFIG"

	WarningSeverity ErrorSeverityType = "warning"
	ErrorSeverity   ErrorSeverityType = "error"
	FatalSeverity   ErrorSeverityType = "fatal"
	PanicSeverity   ErrorSeverityType = "panic"
)

var Collector EventCollector = &noopCollector{}

type EventCollector interface {
	RecordStart(ctx context.Context)
	RecordError(ctx context.Context, severity ErrorSeverityType, err error)
	RecordCLI(self *managementv1.Self, err error)

	// Flush makes sure all events are sent to the backend
	Flush()

	Init(currentNamespaceConfig *rest.Config, currentNamespace string, options *options.VirtualClusterOptions)
	SetVirtualClient(virtualClient *kubernetes.Clientset)
}

// Start starts collecting events and sending them to the backend
func Start(isCli bool) {
	var c Config

	if os.Getenv(ConfigEnvVar) != "" {
		err := json.Unmarshal([]byte(os.Getenv(ConfigEnvVar)), &c)
		if err != nil {
			loghelper.New("telemetry").Infof("failed to parse telemetry config from the %s environment variable: %v", ConfigEnvVar, err)
		}
	} else if isCli {
		cliConfig := cliconfig.GetConfig(log.Discard)
		if cliConfig.TelemetryDisabled {
			c.Disabled = "true"
		}
	}

	// if disabled, we return noop collector
	if c.Disabled == "true" {
		return
	} else if (!isCli && SyncerVersion == "dev") || (isCli && upgrade.GetVersion() == upgrade.DevelopmentVersion) {
		// client.Dry = true
		return
	}

	// create a new default collector
	collector, err := NewDefaultCollector(c, isCli)
	if err != nil {
		// Log the problem but don't fail - use disabled Collector instead
		loghelper.New("telemetry").Infof("%s", err.Error())
	} else {
		Collector = collector
	}
}

func NewDefaultCollector(config Config, isCli bool) (*DefaultCollector, error) {
	defaultCollector := &DefaultCollector{
		analyticsClient: client.NewClient(),

		config: config,
		log:    loghelper.New("telemetry"),
	}

	if !isCli {
		go defaultCollector.startReportStatus(context.Background())
	}
	return defaultCollector, nil
}

type DefaultCollector struct {
	analyticsClient client.Client

	config Config
	log    loghelper.Logger

	vClusterID            cachedValue[string]
	hostClusterVersion    cachedValue[*KubernetesVersion]
	virtualClusterVersion cachedValue[*KubernetesVersion]
	chartInfo             cachedValue[*ChartInfo]

	// everything below will be set during runtime
	virtualClient *kubernetes.Clientset
	options       *options.VirtualClusterOptions
	hostClient    *kubernetes.Clientset
	hostNamespace string
}

func (d *DefaultCollector) startReportStatus(ctx context.Context) {
	time.Sleep(time.Second * 30)

	wait.Until(func() {
		d.RecordStatus(ctx)
	}, time.Minute*5, ctx.Done())
}

func (d *DefaultCollector) Init(currentNamespaceConfig *rest.Config, currentNamespace string, options *options.VirtualClusterOptions) {
	hostClient, err := kubernetes.NewForConfig(currentNamespaceConfig)
	if err != nil {
		klog.V(1).ErrorS(err, "create host client")
	}

	d.hostClient = hostClient
	d.hostNamespace = currentNamespace
	d.options = options
}

func (d *DefaultCollector) SetVirtualClient(virtualClient *kubernetes.Clientset) {
	d.virtualClient = virtualClient
}

func (d *DefaultCollector) Flush() {
	d.analyticsClient.Flush()
}

func (d *DefaultCollector) RecordCLI(self *managementv1.Self, err error) {
	timezone, _ := time.Now().Zone()
	eventProperties := map[string]interface{}{
		"command": os.Args,
		"version": upgrade.GetVersion(),
	}
	userProperties := map[string]interface{}{
		"os_name":  runtime.GOOS,
		"os_arch":  runtime.GOARCH,
		"timezone": timezone,
	}
	if err != nil {
		eventProperties["error"] = err.Error()
	}

	// build the event and record
	eventPropertiesRaw, _ := json.Marshal(eventProperties)
	userPropertiesRaw, _ := json.Marshal(userProperties)
	d.analyticsClient.RecordEvent(client.Event{
		"event": {
			"type":                 "vcluster_cli",
			"platform_user_id":     GetPlatformUserID(self),
			"platform_instance_id": GetPlatformInstanceID(self),
			"machine_id":           GetMachineID(log.Discard),
			"properties":           string(eventPropertiesRaw),
			"timestamp":            time.Now().Unix(),
		},
		"user": {
			"platform_user_id":     GetPlatformUserID(self),
			"platform_instance_id": GetPlatformInstanceID(self),
			"machine_id":           GetMachineID(log.Discard),
			"properties":           string(userPropertiesRaw),
			"timestamp":            time.Now().Unix(),
		},
	})
}

func (d *DefaultCollector) RecordStatus(ctx context.Context) {
	properties := d.getMetrics(ctx)

	// build the event and record
	propertiesRaw, _ := json.Marshal(properties)
	d.analyticsClient.RecordEvent(client.Event{
		"event": {
			"type":                 "vcluster_status",
			"vcluster_id":          d.getVClusterID(ctx),
			"platform_user_id":     d.config.PlatformUserID,
			"platform_instance_id": d.config.PlatformInstanceID,
			"machine_id":           d.config.MachineID,
			"properties":           string(propertiesRaw),
			"timestamp":            time.Now().Unix(),
		},
	})
}

func (d *DefaultCollector) RecordStart(ctx context.Context) {
	chartInfo := d.getChartInfo(ctx)
	properties := map[string]interface{}{
		"vcluster_version":            SyncerVersion,
		"vcluster_k8s_distro_version": d.getVirtualClusterVersion(),
		"host_cluster_k8s_version":    d.getHostClusterVersion(),
		"os_arch":                     runtime.GOOS + "/" + runtime.GOARCH,
		"creation_method":             d.config.InstanceCreator,
	}
	if chartInfo != nil {
		properties["vcluster_k8s_distro"] = chartInfo.Name
		properties["helm_values"] = chartInfo.Values
	}

	// build the event and record
	propertiesRaw, _ := json.Marshal(properties)
	d.analyticsClient.RecordEvent(client.Event{
		"event": {
			"type":                 "vcluster_start",
			"vcluster_id":          d.getVClusterID(ctx),
			"platform_user_id":     d.config.PlatformUserID,
			"platform_instance_id": d.config.PlatformInstanceID,
			"machine_id":           d.config.MachineID,
			"timestamp":            time.Now().Unix(),
		},
		"vcluster_instance": {
			"vcluster_id":          d.getVClusterID(ctx),
			"platform_instance_id": d.config.PlatformInstanceID,
			"properties":           string(propertiesRaw),
			"timestamp":            time.Now().Unix(),
		},
	})
}

func (d *DefaultCollector) RecordError(ctx context.Context, severity ErrorSeverityType, err error) {
	properties := map[string]interface{}{
		"severity": string(severity),
		"message":  err.Error(),
	}

	// if panic or fatal we add the helm values
	if severity == PanicSeverity || severity == FatalSeverity {
		chartInfo := d.getChartInfo(ctx)
		if chartInfo != nil {
			properties["helm_values"] = chartInfo.Values
		}
	}

	// build the event and record
	propertiesRaw, _ := json.Marshal(properties)
	d.analyticsClient.RecordEvent(client.Event{
		"event": {
			"type":                 "vcluster_error",
			"vcluster_id":          d.getVClusterID(ctx),
			"platform_user_id":     d.config.PlatformUserID,
			"platform_instance_id": d.config.PlatformInstanceID,
			"machine_id":           d.config.MachineID,
			"properties":           string(propertiesRaw),
			"timestamp":            time.Now().Unix(),
		},
	})
}

func (d *DefaultCollector) getVirtualClusterVersion() *KubernetesVersion {
	virtualVersion, err := d.virtualClusterVersion.Get(func() (*KubernetesVersion, error) {
		return getKubernetesVersion(d.virtualClient)
	})
	if err != nil {
		klog.V(1).ErrorS(err, "Error retrieving virtual cluster version")
	}

	return virtualVersion
}

func (d *DefaultCollector) getHostClusterVersion() *KubernetesVersion {
	hostVersion, err := d.hostClusterVersion.Get(func() (*KubernetesVersion, error) {
		return getKubernetesVersion(d.hostClient)
	})
	if err != nil {
		klog.V(1).ErrorS(err, "Error retrieving host cluster version")
	}

	return hostVersion
}

func (d *DefaultCollector) getChartInfo(ctx context.Context) *ChartInfo {
	chartInfo, err := d.chartInfo.Get(func() (*ChartInfo, error) {
		return getChartInfo(ctx, d.hostClient, d.hostNamespace)
	})
	if err != nil {
		klog.V(1).ErrorS(err, "Error retrieving chart info")
	}

	return chartInfo
}

func (d *DefaultCollector) getVClusterID(ctx context.Context) string {
	vClusterID, err := d.vClusterID.Get(func() (string, error) {
		return getVClusterID(ctx, d.hostClient, d.hostNamespace, d.options)
	})
	if err != nil {
		klog.V(1).ErrorS(err, "Error retrieving vClusterID")
	}

	return vClusterID
}

func (d *DefaultCollector) getMetrics(ctx context.Context) map[string]interface{} {
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
