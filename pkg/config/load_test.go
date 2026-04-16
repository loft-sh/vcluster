package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/loft-sh/vcluster/pkg/constants"
)

func TestLoadNormalizedStandaloneConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	err := os.WriteFile(path, []byte("controlPlane: {}\n"), 0o600)
	if err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := loadNormalizedStandaloneConfig("test", path, nil)
	if err != nil {
		t.Fatalf("loadNormalizedStandaloneConfig() error = %v", err)
	}

	if !cfg.ControlPlane.Standalone.Enabled {
		t.Fatalf("expected standalone to be enabled")
	}
	if !cfg.PrivateNodes.Enabled {
		t.Fatalf("expected private nodes to be enabled")
	}
	if cfg.ControlPlane.Standalone.DataDir != constants.VClusterStandaloneDefaultDataDir {
		t.Fatalf("expected default data dir to be set, got %q", cfg.ControlPlane.Standalone.DataDir)
	}
}

func TestResolveConfigPathFromExecStart_ConfigFlag(t *testing.T) {
	unit := []byte(`
[Service]
ExecStart=/var/lib/vcluster/bin/vcluster start --config /etc/vcluster/custom.yaml
`)

	path, ok := resolveConfigPathFromExecStart(unit)
	if !ok {
		t.Fatal("resolveConfigPathFromExecStart() did not find config path")
	}
	if path != "/etc/vcluster/custom.yaml" {
		t.Fatalf("expected config path from ExecStart, got %q", path)
	}
}

func TestResolveConfigPathFromExecStart_InstallerUnit(t *testing.T) {
	unit := []byte(`
[Service]
EnvironmentFile=-/etc/default/%N
EnvironmentFile=-/etc/sysconfig/%N
ExecStart=/var/lib/vcluster/bin/vcluster start --config /etc/vcluster/vcluster.yaml
`)

	path, ok := resolveConfigPathFromExecStart(unit)
	if !ok {
		t.Fatal("resolveConfigPathFromExecStart() did not find config path")
	}
	if path != "/etc/vcluster/vcluster.yaml" {
		t.Fatalf("expected installer config path, got %q", path)
	}
}

func TestResolveStandaloneRuntimeName(t *testing.T) {
	unit := []byte(`
[Service]
Environment="VCLUSTER_NAME=my-runtime-name"
`)

	name := resolveStandaloneRuntimeName(unit)
	if name != "my-runtime-name" {
		t.Fatalf("expected runtime name from systemd, got %q", name)
	}
}

func TestResolveStandaloneRuntimeName_Default(t *testing.T) {
	name := resolveStandaloneRuntimeName([]byte("[Service]\n"))
	if name != constants.VClusterStandaloneDefaultName {
		t.Fatalf("expected default standalone name, got %q", name)
	}
}

func TestResolveStandaloneConfigFromCandidates_FallbackOrder(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "standalone.yaml")
	second := filepath.Join(dir, "default.yaml")

	err := os.WriteFile(second, []byte("controlPlane: {}\n"), 0o600)
	if err != nil {
		t.Fatalf("write fallback config: %v", err)
	}

	path, err := resolveStandaloneConfigFromCandidates([]string{first, second})
	if err != nil {
		t.Fatalf("resolveStandaloneConfigFromCandidates() error = %v", err)
	}
	if path != second {
		t.Fatalf("expected fallback candidate, got %q", path)
	}
}

func TestValidateNormalizedStandaloneConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	err := os.WriteFile(path, []byte(`
controlPlane: {}
integrations:
  metricsServer:
    enabled: true
`), 0o600)
	if err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err = loadNormalizedStandaloneConfig("test", path, nil)
	if err == nil {
		t.Fatal("expected standalone validation error")
	}
	if !strings.Contains(err.Error(), "metrics-server integration is not supported in private nodes mode") {
		t.Fatalf("expected private nodes validation error, got %v", err)
	}
}
