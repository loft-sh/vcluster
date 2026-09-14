package standalone

import (
	"os"
	"testing"
)

func TestDetectStandaloneHost_NotFound(t *testing.T) {
	path := t.TempDir() + "/missing.service"

	unitData, found, err := detectStandaloneHostFromPath(path)
	if err != nil {
		t.Fatalf("detectStandaloneHostFromPath() error = %v", err)
	}
	if found {
		t.Fatal("expected standalone host to be undetected")
	}
	if unitData != nil {
		t.Fatalf("expected nil unit data, got %q", string(unitData))
	}
}

func TestDetectStandaloneHost_Found(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/vcluster.service"

	expected := []byte("[Service]\nExecStart=/bin/true\n")
	if err := os.WriteFile(path, expected, 0o600); err != nil {
		t.Fatalf("write unit file: %v", err)
	}

	unitData, found, err := detectStandaloneHostFromPath(path)
	if err != nil {
		t.Fatalf("detectStandaloneHostFromPath() error = %v", err)
	}
	if !found {
		t.Fatal("expected standalone host to be detected")
	}
	if string(unitData) != string(expected) {
		t.Fatalf("expected unit data %q, got %q", string(expected), string(unitData))
	}
}

func TestParseEnvFromSystemdUnit(t *testing.T) {
	unitData := []byte(`
[Service]
Environment="VCLUSTER_VERSION=v0.1.2"
`)

	value := ParseEnvFromSystemdUnit(unitData, "VCLUSTER_VERSION")
	if value != "v0.1.2" {
		t.Fatalf("expected parsed env value %q, got %q", "v0.1.2", value)
	}
}
