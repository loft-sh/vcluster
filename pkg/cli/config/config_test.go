package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestClearPlatform(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")

	// Build a populated config and write it to disk
	cfg := &CLI{
		path: cfgPath,
		Driver: Driver{
			Type: DockerDriver,
		},
		TelemetryDisabled: true,
		PreviousContext:   "my-context",
		Platform: Platform{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Config",
				APIVersion: "storage.loft.sh/v1",
			},
			Host:               "https://example.loft.host",
			AccessKey:          "secret-key",
			LastInstallContext: "some-context",
			Insecure:           true,
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := cfg.ClearPlatform(); err != nil {
		t.Fatalf("ClearPlatform: %v", err)
	}

	// Non-platform fields must be preserved
	if cfg.Driver.Type != DockerDriver {
		t.Errorf("Driver.Type = %q, want %q", cfg.Driver.Type, DockerDriver)
	}
	if !cfg.TelemetryDisabled {
		t.Error("TelemetryDisabled should still be true")
	}
	if cfg.PreviousContext != "my-context" {
		t.Errorf("PreviousContext = %q, want %q", cfg.PreviousContext, "my-context")
	}

	// Platform fields must be cleared
	if cfg.Platform.Host != "" {
		t.Errorf("Platform.Host = %q, want empty", cfg.Platform.Host)
	}
	if cfg.Platform.AccessKey != "" {
		t.Errorf("Platform.AccessKey = %q, want empty", cfg.Platform.AccessKey)
	}
	if cfg.Platform.LastInstallContext != "" {
		t.Errorf("Platform.LastInstallContext = %q, want empty", cfg.Platform.LastInstallContext)
	}
	if cfg.Platform.Insecure {
		t.Error("Platform.Insecure should be false after clear")
	}

	// TypeMeta defaults must be restored
	if cfg.Platform.Kind != "Config" {
		t.Errorf("Platform.Kind = %q, want %q", cfg.Platform.Kind, "Config")
	}
	if cfg.Platform.APIVersion != "storage.loft.sh/v1" {
		t.Errorf("Platform.APIVersion = %q, want %q", cfg.Platform.APIVersion, "storage.loft.sh/v1")
	}

	// File on disk must reflect the cleared state
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	saved := &CLI{}
	if err := json.Unmarshal(data, saved); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if saved.Driver.Type != DockerDriver {
		t.Errorf("saved Driver.Type = %q, want %q", saved.Driver.Type, DockerDriver)
	}
	if saved.Platform.Host != "" {
		t.Errorf("saved Platform.Host = %q, want empty", saved.Platform.Host)
	}
}

func TestClearPlatform_PlatformDriverResetToHelm(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")

	cfg := &CLI{
		path: cfgPath,
		Driver: Driver{
			Type: PlatformDriver,
		},
		Platform: Platform{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Config",
				APIVersion: "storage.loft.sh/v1",
			},
			Host:      "https://example.loft.host",
			AccessKey: "secret-key",
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := cfg.ClearPlatform(); err != nil {
		t.Fatalf("ClearPlatform: %v", err)
	}

	// platform driver must be reset to helm to avoid "not logged in" errors
	if cfg.Driver.Type != HelmDriver {
		t.Errorf("Driver.Type = %q, want %q (platform driver should reset to helm after destroy)", cfg.Driver.Type, HelmDriver)
	}
	if cfg.Platform.Host != "" {
		t.Errorf("Platform.Host = %q, want empty", cfg.Platform.Host)
	}
}
