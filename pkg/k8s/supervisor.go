package k8s

import (
	"context"
	"os"
	"strconv"

	"github.com/loft-sh/vcluster/pkg/util/command"
)

// ComponentSupervisorEnvVar is the env var that toggles in-process subprocess
// supervision. It is a temporary local-development knob — the design doc
// proposes promoting it to controlPlane.statefulSet.componentSupervisor.enabled
// in vcluster.yaml before this lands. Default in this build is on, so a local
// build can be validated without setting anything; production builds should
// gate on the chart value before shipping.
const ComponentSupervisorEnvVar = "VCLUSTER_COMPONENT_SUPERVISOR"

// componentSupervisorEnabled returns true when the per-component supervisor
// should wrap each control-plane subprocess. Reading the env var on every
// component start lets tests and operators flip it without restarting the pod
// (when used in conjunction with cgroup-aware test fixtures).
func componentSupervisorEnabled() bool {
	v := os.Getenv(ComponentSupervisorEnvVar)
	if v == "" {
		// Local validation default. Flip to false before this lands on
		// main; the chart value will own the production default.
		return true
	}
	enabled, err := strconv.ParseBool(v)
	if err != nil {
		// Unrecognized value: be conservative, take the legacy path.
		return false
	}
	return enabled
}

// supervisorPolicy is the restart policy used when the supervisor is enabled.
// It is a package-level variable so tests can substitute a faster policy
// without burning real wall time on backoffs.
var supervisorPolicy = command.DefaultRestartPolicy()

// runComponent invokes runner under the supervisor (when enabled) or once
// directly (legacy). Returns nil on graceful completion, ctx.Err() on
// cancellation, or a non-nil error when the supervisor exhausts its restart
// budget. Callers map the non-nil case to osutil.Exit(1) so kubelet's
// CrashLoopBackoff still applies, preserving the contract from PR #2647.
func runComponent(ctx context.Context, name string, runner command.ComponentRunner) error {
	if componentSupervisorEnabled() {
		return command.RunWithRestart(ctx, name, runner, supervisorPolicy, nil)
	}
	return runner(ctx)
}
