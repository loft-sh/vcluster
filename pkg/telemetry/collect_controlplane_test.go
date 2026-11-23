package telemetry

import (
	"context"
	"encoding/json"
	"fmt"
	goruntime "runtime"
	"runtime/debug"
	"strings"
	"testing"
	"time"

	"github.com/loft-sh/analytics-client/client"
	vclusterconfig "github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/strvals"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/version"
	fakediscovery "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/yaml"
)

// Plain alphanumeric on purpose, see commandLineValue.
const configValue = "c0nf1gvalue"

// userValues is what a user passes to helm install, with configValue in free
// text of every kind the config has: secrets, hostnames, map keys, manifests
// and env values.
const userValues = `
controlPlane:
  backingStore:
    database:
      external:
        enabled: true
        dataSource: mysql://admin:c0nf1gvalue@tcp(db.internal:3306)/vcluster
  statefulSet:
    highAvailability:
      replicas: 3
    labels:
      team-c0nf1gvalue: payments
    env:
      - name: HTTPS_PROXY
        value: http://proxy:c0nf1gvalue@proxy.internal:3128
  ingress:
    enabled: true
    host: c0nf1gvalue.example.com
sync:
  toHost:
    ingresses:
      enabled: true
    customResources:
      certificates.c0nf1gvalue.io:
        enabled: true
  fromHost:
    nodes:
      enabled: true
      selector:
        labels:
          pool: c0nf1gvalue
privateNodes:
  joinNode:
    containerd:
      registry:
        auth:
          registry.c0nf1gvalue.internal:
            username: puller
            password: c0nf1gvalue
            identityToken: c0nf1gvalue
policies:
  podSecurityStandard: baseline
  resourceQuota:
    enabled: true
logging:
  encoding: c0nf1gvalue
experimental:
  deploy:
    vcluster:
      manifests: |
        apiVersion: v1
        kind: Secret
        metadata:
          name: registry-creds
        stringData:
          password: c0nf1gvalue
      helm:
        - chart:
            name: internal-app
            repo: https://charts.internal
            username: deploy
            password: c0nf1gvalue
telemetry:
  instanceCreator: vclusterctl
`

// parseChartConfig returns the config the control plane loads after a helm
// install with values: the chart writes its defaults merged with the values
// to the config secret, and the control plane parses that.
func parseChartConfig(t *testing.T, values string) *config.VirtualClusterConfig {
	t.Helper()

	chartValues, overrides := map[string]interface{}{}, map[string]interface{}{}
	assert.NilError(t, yaml.Unmarshal([]byte(vclusterconfig.Values), &chartValues))
	assert.NilError(t, yaml.Unmarshal([]byte(values), &overrides))

	rawConfig, err := yaml.Marshal(strvals.MergeMaps(chartValues, overrides))
	assert.NilError(t, err)

	vConfig, err := config.ParseConfigBytes(rawConfig, "my-vcluster", nil)
	assert.NilError(t, err)
	vConfig.HostNamespace = "namespace-" + configValue
	vConfig.Path = "/" + configValue + "/config.yaml"

	return vConfig
}

// lookup returns the value at a dotted path of nested maps, nil if there is
// none.
func lookup(values map[string]interface{}, path string) interface{} {
	var current interface{} = values
	for _, key := range strings.Split(path, ".") {
		object, _ := current.(map[string]interface{})
		current = object[key]
	}

	return current
}

func toJSONObject(t *testing.T, value interface{}) map[string]interface{} {
	t.Helper()

	raw, err := json.Marshal(value)
	assert.NilError(t, err)
	object := map[string]interface{}{}
	assert.NilError(t, json.Unmarshal(raw, &object))

	return object
}

func TestKubernetesVersion(t *testing.T) {
	testCases := []struct {
		name     string
		reported version.Info
		expected KubernetesVersion
	}{
		{name: "upstream", reported: version.Info{Major: "1", Minor: "36", GitVersion: "v1.36.6"}, expected: KubernetesVersion{Major: "1", Minor: "36", GitVersion: "v1.36.6"}},
		{name: "release candidate", reported: version.Info{Major: "1", Minor: "37", GitVersion: "v1.37.0-rc.1"}, expected: KubernetesVersion{Major: "1", Minor: "37", GitVersion: "v1.37.0-rc.1"}},
		{name: "eks", reported: version.Info{Major: "1", Minor: "33+", GitVersion: "v1.33.5-eks-113cf36"}, expected: KubernetesVersion{Major: "1", Minor: "33+", GitVersion: "v1.33.5-eks"}},
		{name: "gke", reported: version.Info{Major: "1", Minor: "33", GitVersion: "v1.33.5-gke.1162000"}, expected: KubernetesVersion{Major: "1", Minor: "33", GitVersion: "v1.33.5-gke"}},
		{name: "k3s", reported: version.Info{Major: "1", Minor: "33", GitVersion: "v1.33.5+k3s1"}, expected: KubernetesVersion{Major: "1", Minor: "33", GitVersion: "v1.33.5+k3s"}},
		{name: "build suffix", reported: version.Info{Major: "1", Minor: "32", GitVersion: "v1.32.9+7ea2b1c"}, expected: KubernetesVersion{Major: "1", Minor: "32", GitVersion: "v1.32.9"}},
		{
			name:     "version set by the operator of the server",
			reported: version.Info{Major: "1", Minor: "36", GitVersion: "v1.36.6+" + configValue},
			expected: KubernetesVersion{Major: "1", Minor: "36", GitVersion: "v1.36.6"},
		},
		{
			name:     "text after a listed distribution",
			reported: version.Info{Major: "1", Minor: "36", GitVersion: "v1.36.6-eks" + configValue},
			expected: KubernetesVersion{Major: "1", Minor: "36", GitVersion: "v1.36.6-eks"},
		},
		{
			name:     "text in every field",
			reported: version.Info{Major: configValue, Minor: "36" + configValue, GitVersion: configValue + "v1.36.6"},
			expected: KubernetesVersion{},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.DeepEqual(t, *toKubernetesVersion(&testCase.reported), testCase.expected)
		})
	}
}

// versionClient is a client for a server that reports gitVersion with text
// after it, as a server started with its own version does.
func versionClient(gitVersion string, objects ...runtime.Object) *fake.Clientset {
	kubeClient := fake.NewSimpleClientset(objects...)
	kubeClient.Discovery().(*fakediscovery.FakeDiscovery).FakedServerVersion = &version.Info{
		Major: "1", Minor: "36", GitVersion: gitVersion + "+" + configValue,
	}

	return kubeClient
}

// panicWithConfigValue panics the way a failed assertion deep in the control
// plane does, with text in the panic value.
func panicWithConfigValue() {
	panic(fmt.Errorf("unexpected object %s", configValue))
}

func TestControlPlaneEventsHaveNoConfigOrErrorText(t *testing.T) {
	type properties = map[string]interface{}

	wrappedErr := fmt.Errorf("initialize: connect to mysql://admin:%s@tcp(db.internal:3306)/vcluster: %w", configValue,
		kerrors.NewForbidden(schema.GroupResource{Resource: "secrets"}, configValue, fmt.Errorf("%s", configValue)))
	recordStart := func(collector *controlPlaneCollector, vConfig *config.VirtualClusterConfig) {
		collector.RecordStart(context.Background(), vConfig)
	}

	testCases := []struct {
		name            string
		instanceCreator string
		record          func(collector *controlPlaneCollector, vConfig *config.VirtualClusterConfig)
		eventType       string
		expected        properties
		hasConfig       bool
		hasStack        bool
	}{
		{name: "start", instanceCreator: "vclusterctl", record: recordStart, eventType: "vcluster_start", expected: properties{"creation_method": "vclusterctl"}, hasConfig: true},
		{name: "start with the instance creator of the platform", instanceCreator: "loft", record: recordStart, eventType: "vcluster_start", expected: properties{"creation_method": "loft"}, hasConfig: true},
		{name: "start without an instance creator", record: recordStart, eventType: "vcluster_start", expected: properties{"creation_method": ""}, hasConfig: true},
		{name: "start with an instance creator nothing writes", instanceCreator: configValue, record: recordStart, eventType: "vcluster_start", expected: properties{"creation_method": "other"}, hasConfig: true},
		{
			name: "fatal error",
			record: func(collector *controlPlaneCollector, vConfig *config.VirtualClusterConfig) {
				collector.RecordError(context.Background(), vConfig, FatalSeverity, wrappedErr)
			},
			eventType: "vcluster_error",
			expected:  properties{"severity": "fatal", "message": "kubernetes_forbidden"},
			hasConfig: true,
		},
		{
			name: "lost leader election",
			record: func(collector *controlPlaneCollector, vConfig *config.VirtualClusterConfig) {
				collector.RecordError(context.Background(), vConfig, WarningSeverity, ErrLeaderElectionLost)
			},
			eventType: "vcluster_error",
			expected:  properties{"severity": "warning", "message": "leader_election_lost"},
		},
		{
			name: "panic",
			// recovered and recorded the way the start command does
			record: func(collector *controlPlaneCollector, vConfig *config.VirtualClusterConfig) {
				defer func() {
					if r := recover(); r != nil {
						collector.RecordError(context.Background(), vConfig, PanicSeverity, fmt.Errorf("panic: %v %s", r, string(debug.Stack())))
					}
				}()
				panicWithConfigValue()
			},
			eventType: "vcluster_error",
			expected:  properties{"severity": "panic", "message": "other"},
			hasConfig: true,
			hasStack:  true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			vConfig := parseChartConfig(t, userValues)
			vConfig.Telemetry.InstanceCreator = testCase.instanceCreator
			// The identifiers the CLI and the platform write and the name of
			// the instance are sent as they are, so they are the places the
			// text is not put in.
			vConfig.Telemetry.MachineID = "machine"
			vConfig.Telemetry.PlatformUserID = "user"
			vConfig.Telemetry.PlatformInstanceID = "instance"

			// set from the name when the control plane starts
			previousName := translate.VClusterName
			translate.VClusterName = vConfig.Name
			t.Cleanup(func() { translate.VClusterName = previousName })

			created := metav1.NewTime(time.Unix(1700000000, 0))
			analyticsClient := &recordingAnalyticsClient{}
			collector := &controlPlaneCollector{
				analyticsClient: analyticsClient,
				log:             loghelper.New("telemetry"),
				hostClient: versionClient("v1.33.5-eks-113cf36", &corev1.Service{ObjectMeta: metav1.ObjectMeta{
					Name: vConfig.Name, Namespace: vConfig.HostNamespace, UID: "service-uid", CreationTimestamp: created,
				}}),
				hostNamespace: vConfig.HostNamespace,
				hostService:   vConfig.Name,
				virtualClient: versionClient("v1.36.6"),
			}
			testCase.record(collector, vConfig)
			assert.Equal(t, len(analyticsClient.events), 1)

			payload, err := json.Marshal(client.Request{Data: analyticsClient.events})
			assert.NilError(t, err)
			assert.Assert(t, !strings.Contains(string(payload), configValue), "payload carries a config value or error text: %s", payload)

			sentEvent := toJSONObject(t, analyticsClient.events[0])
			rawProperties, _ := lookup(sentEvent, "event.properties").(string)
			expectedEvent := properties{"event": properties{
				"type": testCase.eventType, "vcluster_id": "service-uid",
				"machine_id": "machine", "platform_user_id": "user", "platform_instance_id": "instance",
				"timestamp": lookup(sentEvent, "event.timestamp"),
			}}
			if testCase.eventType == "vcluster_start" {
				rawProperties, _ = lookup(sentEvent, "vcluster_instance.properties").(string)
				expectedEvent["vcluster_instance"] = properties{
					"vcluster_id": "service-uid", "platform_instance_id": "instance",
					"properties": rawProperties, "timestamp": lookup(sentEvent, "vcluster_instance.timestamp"),
				}

				testCase.expected["vcluster_version"] = SyncerVersion
				testCase.expected["vcluster_k8s_distro"] = "my-vcluster"
				testCase.expected["vcluster_k8s_distro_version"] = properties{"major": "1", "minor": "36", "gitVersion": "v1.36.6"}
				testCase.expected["host_cluster_k8s_version"] = properties{"major": "1", "minor": "36", "gitVersion": "v1.33.5-eks"}
				testCase.expected["os_arch"] = goruntime.GOOS + "/" + goruntime.GOARCH
				testCase.expected["creation_timestamp"] = float64(created.Unix())
			} else {
				lookup(expectedEvent, "event").(properties)["properties"] = rawProperties
				testCase.expected["vcluster_version"] = SyncerVersion
			}
			assert.DeepEqual(t, sentEvent, expectedEvent)

			// what the summary holds is in TestConfigSummaryOfChartConfig
			if testCase.hasConfig {
				testCase.expected["helm_values"] = toJSONObject(t, configSummary(&vConfig.Config))
			}
			sent := properties{}
			assert.NilError(t, json.Unmarshal([]byte(rawProperties), &sent))
			if testCase.hasStack {
				// where it panicked, through the recovering function
				stack := fmt.Sprint(sent["stack"])
				assert.Assert(t, strings.Contains(stack, "telemetry.panicWithConfigValue:"), "stack is %s", stack)
				assert.Assert(t, strings.Contains(stack, "runtime.gopanic:"), "stack is %s", stack)
				delete(sent, "stack")
			}
			assert.DeepEqual(t, sent, testCase.expected)
		})
	}
}
