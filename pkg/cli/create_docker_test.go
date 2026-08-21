package cli

import (
	"os"
	"path/filepath"
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
