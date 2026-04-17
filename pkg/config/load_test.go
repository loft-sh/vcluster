package config

import (
	"os"
	"path/filepath"
	"testing"

	vclusterconfig "github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/constants"
)

func TestLoadNormalizedStandaloneConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	err := os.WriteFile(path, []byte("controlPlane: {}\n"), 0o600)
	if err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadStandaloneRuntimeConfig("test", path, nil)
	if err != nil {
		t.Fatalf("LoadStandaloneRuntimeConfig() error = %v", err)
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

func TestResolveStandaloneConfigPath_DefaultConfigLocationFallsBackToStandaloneDefault(t *testing.T) {
	path, err := ResolveStandaloneConfigPath(constants.DefaultVClusterConfigLocation)
	if err != nil {
		t.Fatalf("ResolveStandaloneConfigPath() error = %v", err)
	}

	expectedPath := constants.VClusterStandaloneDefaultConfigPath
	if info, err := os.Stat(constants.VClusterStandaloneDefaultConfigPath); os.IsNotExist(err) {
		if info, err := os.Stat(constants.DefaultVClusterConfigLocation); err == nil && !info.IsDir() {
			expectedPath = constants.DefaultVClusterConfigLocation
		}
	} else if err != nil {
		t.Fatalf("stat standalone default config path: %v", err)
	} else if info.IsDir() {
		t.Fatalf("expected standalone default config path %q to be a file", constants.VClusterStandaloneDefaultConfigPath)
	}

	if path != expectedPath {
		t.Fatalf("expected resolved config path %q, got %q", expectedPath, path)
	}
}

func TestLoadStandaloneRuntimeConfig_MissingCustomPath(t *testing.T) {
	_, err := LoadStandaloneRuntimeConfig("test", filepath.Join(t.TempDir(), "missing.yaml"), nil)
	if err == nil {
		t.Fatal("expected missing custom config path to fail")
	}
	if !os.IsNotExist(err) {
		t.Fatalf("expected os.ErrNotExist, got %v", err)
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

func TestLoadNormalizedStandaloneConfig_DoesNotValidate(t *testing.T) {
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

	cfg, err := LoadStandaloneRuntimeConfig("test", path, nil)
	if err != nil {
		t.Fatalf("LoadStandaloneRuntimeConfig() error = %v", err)
	}
	if !cfg.ControlPlane.Standalone.Enabled {
		t.Fatalf("expected standalone to be enabled")
	}
}

func TestLoadStandaloneRuntimeConfig_MergesDefaultsAndPreservesExplicitFalse(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	err := os.WriteFile(path, []byte(`
deploy:
  cni:
    flannel:
      enabled: false
`), 0o600)
	if err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadStandaloneRuntimeConfig("test", path, nil)
	if err != nil {
		t.Fatalf("LoadStandaloneRuntimeConfig() error = %v", err)
	}

	defaultConfig, err := vclusterconfig.NewDefaultConfig()
	if err != nil {
		t.Fatalf("NewDefaultConfig() error = %v", err)
	}

	if cfg.ControlPlane.StatefulSet.Image.Repository != defaultConfig.ControlPlane.StatefulSet.Image.Repository {
		t.Fatalf("expected repository %q after merging defaults, got %q", defaultConfig.ControlPlane.StatefulSet.Image.Repository, cfg.ControlPlane.StatefulSet.Image.Repository)
	}
	if cfg.Deploy.CNI.Flannel.Enabled {
		t.Fatal("expected explicit flannel.enabled=false to be preserved")
	}
}

func TestLoadNormalizedStandaloneConfig_DefaultPathMissingUsesInMemoryDefaults(t *testing.T) {
	if _, err := os.Stat(constants.VClusterStandaloneDefaultConfigPath); err == nil {
		t.Skipf("default standalone config path %q exists on this host", constants.VClusterStandaloneDefaultConfigPath)
	}

	cfg, err := LoadStandaloneRuntimeConfig("test", constants.VClusterStandaloneDefaultConfigPath, nil)
	if err != nil {
		t.Fatalf("LoadStandaloneRuntimeConfig() error = %v", err)
	}
	if cfg.Path != constants.VClusterStandaloneDefaultConfigPath {
		t.Fatalf("expected config path %q, got %q", constants.VClusterStandaloneDefaultConfigPath, cfg.Path)
	}
	if !cfg.ControlPlane.Standalone.Enabled {
		t.Fatalf("expected standalone to be enabled")
	}
	if !cfg.PrivateNodes.Enabled {
		t.Fatalf("expected private nodes to be enabled")
	}
	if cfg.ControlPlane.Standalone.DataDir != constants.VClusterStandaloneDefaultDataDir {
		t.Fatalf("expected default data dir %q, got %q", constants.VClusterStandaloneDefaultDataDir, cfg.ControlPlane.Standalone.DataDir)
	}
}

func TestLoadNormalizedStandaloneConfig_CustomMissingPathErrors(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.yaml")

	_, err := LoadStandaloneRuntimeConfig("test", path, nil)
	if err == nil {
		t.Fatal("expected error for missing custom standalone config path")
	}
}

func TestMergeConfigBytesWithDefaults(t *testing.T) {
	mergedConfig, err := mergeConfigBytesWithDefaults([]byte(`
deploy:
  cni:
    flannel:
      enabled: false
`))
	if err != nil {
		t.Fatalf("mergeConfigBytesWithDefaults() error = %v", err)
	}

	cfg, err := ParseConfigBytes(mergedConfig, "test", nil)
	if err != nil {
		t.Fatalf("ParseConfigBytes() error = %v", err)
	}

	defaultConfig, err := vclusterconfig.NewDefaultConfig()
	if err != nil {
		t.Fatalf("NewDefaultConfig() error = %v", err)
	}

	if cfg.ControlPlane.StatefulSet.Image.Repository != defaultConfig.ControlPlane.StatefulSet.Image.Repository {
		t.Fatalf("expected repository %q after merging defaults, got %q", defaultConfig.ControlPlane.StatefulSet.Image.Repository, cfg.ControlPlane.StatefulSet.Image.Repository)
	}
	if cfg.Deploy.CNI.Flannel.Enabled {
		t.Fatal("expected explicit flannel.enabled=false to be preserved")
	}
}
