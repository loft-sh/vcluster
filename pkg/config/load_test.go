package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/loft-sh/vcluster/pkg/constants"
)

func TestNormalizeStandaloneConfig(t *testing.T) {
	cfg := &VirtualClusterConfig{}

	err := normalizeStandaloneConfig(cfg)
	if err != nil {
		t.Fatalf("normalizeStandaloneConfig() error = %v", err)
	}

	if !cfg.ControlPlane.Standalone.Enabled {
		t.Fatalf("expected standalone to be enabled")
	}
	if !cfg.PrivateNodes.Enabled {
		t.Fatalf("expected private nodes to be enabled")
	}
	if cfg.ControlPlane.Standalone.DataDir != constants.StandaloneDefaultDataDir {
		t.Fatalf("expected default data dir to be set, got %q", cfg.ControlPlane.Standalone.DataDir)
	}
}

func TestResolveStandaloneConfigFromSystemdUnit_ExecStartConfig(t *testing.T) {
	unit := []byte(`
[Service]
ExecStart=/var/lib/vcluster/bin/vcluster start --config /etc/vcluster/custom.yaml
`)

	path, err := resolveStandaloneConfigFromSystemdUnit(unit)
	if err != nil {
		t.Fatalf("resolveStandaloneConfigFromSystemdUnit() error = %v", err)
	}
	if path != "/etc/vcluster/custom.yaml" {
		t.Fatalf("expected config path from ExecStart, got %q", path)
	}
}

func TestResolveStandaloneConfigFromSystemdUnit_InstallerUnit(t *testing.T) {
	unit := []byte(`
[Service]
EnvironmentFile=-/etc/default/%N
EnvironmentFile=-/etc/sysconfig/%N
ExecStart=/var/lib/vcluster/bin/vcluster start --config /etc/vcluster/vcluster.yaml
`)

	path, err := resolveStandaloneConfigFromSystemdUnit(unit)
	if err != nil {
		t.Fatalf("resolveStandaloneConfigFromSystemdUnit() error = %v", err)
	}
	if path != "/etc/vcluster/vcluster.yaml" {
		t.Fatalf("expected installer config path, got %q", path)
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
	cfg := &VirtualClusterConfig{
		Name: "test",
	}
	cfg.Integrations.MetricsServer.Enabled = true

	err := normalizeStandaloneConfig(cfg)
	if err != nil {
		t.Fatalf("normalizeStandaloneConfig() error = %v", err)
	}

	err = ValidateConfigAndSetDefaults(cfg)
	if err == nil {
		t.Fatal("expected standalone validation error")
	}
	if !strings.Contains(err.Error(), "metrics-server integration is not supported in private nodes mode") {
		t.Fatalf("expected private nodes validation error, got %v", err)
	}
}
