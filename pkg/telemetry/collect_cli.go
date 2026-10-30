package telemetry

import (
	"encoding/json"
	"os"
	"runtime"
	"time"

	"github.com/loft-sh/analytics-client/client"
	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	"github.com/loft-sh/vcluster/pkg/cli/config"
	"github.com/loft-sh/vcluster/pkg/upgrade"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
)

type ErrorSeverityType string

const (
	WarningSeverity ErrorSeverityType = "warning"
	ErrorSeverity   ErrorSeverityType = "error"
	FatalSeverity   ErrorSeverityType = "fatal"
	PanicSeverity   ErrorSeverityType = "panic"
)

var CollectorCLI CLICollector = &noopCollector{}

type CLICollector interface {
	RecordCLI(cliConfig *config.CLI, self *managementv1.Self, err error)

	// Flush makes sure all events are sent to the backend
	Flush()
}

// StartCLI starts collecting events and sending them to the backend from the CLI
func StartCLI(cliConfig *config.CLI) {
	// if disabled, we return noop collector
	if cliConfig.TelemetryDisabled || upgrade.GetVersion() == upgrade.DevelopmentVersion {
		return
	}

	// create a new default collector
	collector, err := newCLICollector()
	if err != nil {
		// Log the problem but don't fail - use disabled Collector instead
		loghelper.New("telemetry").Infof("%s", err.Error())
	} else {
		CollectorCLI = collector
	}
}

func newCLICollector() (*cliCollector, error) {
	defaultCollector := &cliCollector{
		analyticsClient: client.NewClient(),
		log:             loghelper.New("telemetry"),
	}

	return defaultCollector, nil
}

type cliCollector struct {
	analyticsClient client.Client

	log loghelper.Logger
}

func (d *cliCollector) Flush() {
	d.analyticsClient.Flush()
}

func (d *cliCollector) RecordCLI(cliConfig *config.CLI, self *managementv1.Self, err error) {
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
			"platform_user_id":     GetPlatformUserID(cliConfig, self),
			"platform_instance_id": GetPlatformInstanceID(cliConfig, self),
			"machine_id":           GetMachineID(cliConfig),
			"properties":           string(eventPropertiesRaw),
			"timestamp":            time.Now().Unix(),
		},
		"user": {
			"platform_user_id":     GetPlatformUserID(cliConfig, self),
			"platform_instance_id": GetPlatformInstanceID(cliConfig, self),
			"machine_id":           GetMachineID(cliConfig),
			"properties":           string(userPropertiesRaw),
			"timestamp":            time.Now().Unix(),
		},
	})
}
