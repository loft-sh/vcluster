package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"gotest.tools/v3/assert"
)

func TestGetInstallStandaloneScript(t *testing.T) {
	ctx := t.Context()

	t.Run("environment variable override", func(t *testing.T) {
		scriptPath := filepath.Join(t.TempDir(), "install-standalone.sh")
		assert.NilError(t, os.WriteFile(scriptPath, []byte("#!/bin/sh\necho override"), 0644))
		t.Setenv(installStandaloneScriptEnv, scriptPath)

		globalFlags := &flags.GlobalFlags{Config: filepath.Join(t.TempDir(), "config.json")}
		script, err := getInstallStandaloneScript(ctx, "0.31.0", globalFlags)
		assert.NilError(t, err)
		assert.Equal(t, string(script), "#!/bin/sh\necho override")
	})

	t.Run("environment variable override with missing file", func(t *testing.T) {
		t.Setenv(installStandaloneScriptEnv, filepath.Join(t.TempDir(), "does-not-exist.sh"))

		globalFlags := &flags.GlobalFlags{Config: filepath.Join(t.TempDir(), "config.json")}
		_, err := getInstallStandaloneScript(ctx, "0.31.0", globalFlags)
		assert.ErrorContains(t, err, installStandaloneScriptEnv)
	})

	t.Run("script validation", func(t *testing.T) {
		assert.NilError(t, validateInstallStandaloneScript([]byte("#!/bin/sh\necho ok")))
		assert.ErrorContains(t, validateInstallStandaloneScript([]byte("<html>proxy error page</html>")), "shebang")
		assert.ErrorContains(t, validateInstallStandaloneScript(nil), "shebang")
	})

	t.Run("cached script is used without downloading", func(t *testing.T) {
		configDir := t.TempDir()
		cachePath := filepath.Join(configDir, "docker", "install-standalone", "v0.31.0", "install-standalone.sh")
		assert.NilError(t, os.MkdirAll(filepath.Dir(cachePath), 0755))
		assert.NilError(t, os.WriteFile(cachePath, []byte("#!/bin/sh\necho cached"), 0644))

		globalFlags := &flags.GlobalFlags{Config: filepath.Join(configDir, "config.json")}
		script, err := getInstallStandaloneScript(ctx, "0.31.0", globalFlags)
		assert.NilError(t, err)
		assert.Equal(t, string(script), "#!/bin/sh\necho cached")
	})
}

func TestLoadBalancerUnsupportedReason(t *testing.T) {
	t.Run("reachable network keeps the load balancer enabled everywhere", func(t *testing.T) {
		assert.Equal(t, loadBalancerUnsupportedReason("linux", true, false, false), "")
		assert.Equal(t, loadBalancerUnsupportedReason("darwin", true, false, false), "")
	})

	t.Run("unreachable network disables it outside darwin", func(t *testing.T) {
		reason := loadBalancerUnsupportedReason("linux", false, false, false)
		assert.Assert(t, strings.Contains(reason, "only supported on macOS"), "got: %s", reason)
	})

	t.Run("darwin without privileged ports on podman names the podman limitation", func(t *testing.T) {
		// this is the customer-facing half of loft-sh/vind#5: when the load
		// balancer must be disabled, the warning has to explain the podman
		// limitation instead of offering Docker Desktop advice
		reason := loadBalancerUnsupportedReason("darwin", false, false, true)
		assert.Assert(t, strings.Contains(reason, "Podman"), "got: %s", reason)
		assert.Assert(t, strings.Contains(reason, "containers/podman/issues/28009"), "got: %s", reason)
		assert.Assert(t, !strings.Contains(reason, "Docker Desktop"), "got: %s", reason)
	})

	t.Run("darwin without privileged ports on docker names the docker desktop setting", func(t *testing.T) {
		reason := loadBalancerUnsupportedReason("darwin", false, false, false)
		assert.Assert(t, strings.Contains(reason, "Docker Desktop"), "got: %s", reason)
		assert.Assert(t, !strings.Contains(reason, "Podman"), "got: %s", reason)
		assert.Assert(t, !strings.Contains(reason, "podman/issues/28009"), "got: %s", reason)
	})

	t.Run("darwin with privileged ports keeps it enabled", func(t *testing.T) {
		assert.Equal(t, loadBalancerUnsupportedReason("darwin", false, true, false), "")
	})
}
