package standalone

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/loft-sh/vcluster/pkg/constants"
)

func TestRenderSystemdPlatformConfFile(t *testing.T) {
	options := &AddToPlatformOptions{
		AccessKey:    "abcd",
		Host:         "test.vcluster.platform",
		InstanceName: "test-instance",
		ProjectName:  "test-project",
	}

	want := `
[Service]
EnvironmentFile=-/etc/vcluster/secrets/platform.env
Environment=LOFT_PLATFORM_HOST="test.vcluster.platform"
Environment=LOFT_PLATFORM_INSECURE="false"
Environment=LOFT_PLATFORM_INSTANCE_NAME="test-instance"
Environment=LOFT_PLATFORM_PROJECT_NAME="test-project"
Environment=LOFT_PLATFORM_SKIP_CONFIG_SYNC="false"
`

	got, err := renderSystemdPlatformConfFile(constants.VClusterStandalonePlatformEnvFile, options)
	if err != nil {
		t.Errorf("renderSystemdServiceFile() error = %v", err)
		return
	}

	gotString := string(got)
	if gotString != want {
		t.Errorf("renderSystemdServiceFile() diff(want, got) = %s", cmp.Diff(want, gotString))
	}

	// the drop-in is world-readable in effect: systemd exposes Environment= values
	// over D-Bus to any local user, so the access key must not appear in it
	if strings.Contains(gotString, options.AccessKey) {
		t.Errorf("renderSystemdServiceFile() leaked the access key into the drop-in: %s", gotString)
	}

	// the template spells the variable names out, so a renamed constant would silently
	// stop matching what the drop-in actually sets
	for _, name := range []string{
		constants.PlatformHostEnv,
		constants.PlatformInsecureEnv,
		constants.PlatformInstanceNameEnv,
		constants.PlatformProjectNameEnv,
		constants.PlatformSkipConfigSyncEnv,
	} {
		if !strings.Contains(gotString, "Environment="+name+"=") {
			t.Errorf("renderSystemdServiceFile() does not set %s: %s", name, gotString)
		}
	}
}

func TestWritePlatformEnvFile(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "secrets")
	envPath := filepath.Join(dir, "platform.env")

	// pre-create the file world-readable: an install from a release that predates
	// the env file may have left one behind, and writing over it must still end up
	// root-only
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(envPath, []byte("LOFT_PLATFORM_ACCESS_KEY=stale\n"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := writePlatformEnvFile(envPath, &AddToPlatformOptions{AccessKey: "abcd"}); err != nil {
		t.Fatalf("writePlatformEnvFile() error = %v", err)
	}

	fileInfo, err := os.Stat(envPath)
	if err != nil {
		t.Fatalf("Stat(%s) error = %v", envPath, err)
	}
	if got := fileInfo.Mode().Perm(); got != 0600 {
		t.Errorf("%s mode = %#o, want 0600", envPath, got)
	}

	// the secrets directory ends up root-only even when it was found world-readable
	dirInfo, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("Stat(%s) error = %v", dir, err)
	}
	if got := dirInfo.Mode().Perm(); got != 0700 {
		t.Errorf("%s mode = %#o, want 0700", dir, got)
	}

	content, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", envPath, err)
	}
	if want := "LOFT_PLATFORM_ACCESS_KEY=abcd\n"; string(content) != want {
		t.Errorf("%s content diff(want, got) = %s", envPath, cmp.Diff(want, string(content)))
	}
}

func TestRenderPlatformEnvFile(t *testing.T) {
	options := &AddToPlatformOptions{
		AccessKey: "abcd",
		Host:      "test.vcluster.platform",
	}

	want := "LOFT_PLATFORM_ACCESS_KEY=abcd\n"

	got := string(renderPlatformEnvFile(options))
	if got != want {
		t.Errorf("renderPlatformEnvFile() diff(want, got) = %s", cmp.Diff(want, got))
	}
}
