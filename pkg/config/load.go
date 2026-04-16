package config

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/kballard/go-shellquote"
	"github.com/loft-sh/vcluster/pkg/constants"
	standaloneutil "github.com/loft-sh/vcluster/pkg/util/standalone"
)

// LoadAutoDetectedRuntimeConfig loads vCluster config for commands that run in
// an existing vCluster runtime and need to work in both standalone and
// in-cluster mode. It may inspect local host-side standalone signals such as
// systemd before falling back to the in-cluster config path. It is not intended
// for standalone runtime startup, which should use LoadStandaloneRuntimeConfig
// directly.
//
// When VCLUSTER_STANDALONE is set, it acts as an explicit mode override:
// "true" forces standalone loading and any other value forces the normal
// in-cluster/default config path. Only when the env var is unset we fall back
// to local standalone host detection via the systemd unit marker.
func LoadAutoDetectedRuntimeConfig(vClusterName string) (*VirtualClusterConfig, error) {
	// Respect explicit runtime context from the caller before attempting any
	// local host detection. This allows parent processes to force standalone or
	// non-standalone behavior for child vcluster commands.
	if standaloneEnv, ok := os.LookupEnv(constants.VClusterStandaloneEnvVar); ok {
		if standaloneEnv == "true" {
			vConfig, err := LoadLocalStandaloneConfig(vClusterName, nil)
			if err != nil {
				return nil, fmt.Errorf("loading standalone vCluster config: %w", err)
			}
			return vConfig, nil
		}

		vConfig, err := LoadInClusterRuntimeConfig(vClusterName, "", nil)
		if err != nil {
			return nil, fmt.Errorf("loading vCluster config: %w", err)
		}
		return vConfig, nil
	}

	vConfig, err := LoadLocalStandaloneConfig(vClusterName, nil)
	if err == nil {
		return vConfig, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("loading standalone vCluster config from systemd: %w", err)
	}

	vConfig, err = LoadInClusterRuntimeConfig(vClusterName, "", nil)
	if err != nil {
		return nil, fmt.Errorf("loading vCluster config: %w", err)
	}
	return vConfig, nil
}

// LoadInClusterRuntimeConfig loads config for a vCluster running as a regular
// in-cluster control plane. When path is empty, it falls back to the default
// pod/container config location. It does not perform standalone detection or
// standalone normalization.
func LoadInClusterRuntimeConfig(name, path string, setValues []string) (*VirtualClusterConfig, error) {
	if path == "" {
		path = constants.DefaultVClusterConfigLocation
	}

	return ParseConfig(path, name, setValues)
}

// LoadStandaloneRuntimeConfig loads config for standalone runtime startup. It is
// intended for standalone runtime paths that already know they are operating in
// standalone mode, not for host-side command autodetection.
//
// For historical compatibility, when path is empty or points at the legacy
// in-cluster config location, it resolves from shared standalone default
// locations. Regardless of where the file was loaded from, the resulting config
// is normalized as standalone.
func LoadStandaloneRuntimeConfig(name, path string, setValues []string) (*VirtualClusterConfig, error) {
	var err error

	if path == "" || path == constants.DefaultVClusterConfigLocation {
		path, err = resolveStandaloneConfigFromCandidates([]string{
			constants.VClusterStandaloneDefaultConfigPath,
			constants.DefaultVClusterConfigLocation,
		})
		if err != nil {
			return nil, err
		}
	}
	return loadNormalizedStandaloneConfig(name, path, setValues)
}

// LoadLocalStandaloneConfig loads config for a standalone vCluster installed on
// this host. It discovers the config path from the local systemd unit, uses the
// runtime vCluster name from that unit when name is empty, and normalizes the
// result as standalone. This is for host-side CLI/API flows such as
// vclusterctl, not for standalone runtime startup.
func LoadLocalStandaloneConfig(name string, setValues []string) (*VirtualClusterConfig, error) {
	resolvedPath, unitData, err := resolveStandaloneConfigFromSystemd()
	if err != nil {
		return nil, err
	}
	if name == "" {
		name = resolveStandaloneRuntimeName(unitData)
	}

	return loadNormalizedStandaloneConfig(name, resolvedPath, setValues)
}

func loadNormalizedStandaloneConfig(name, path string, setValues []string) (*VirtualClusterConfig, error) {
	cfg, err := LoadInClusterRuntimeConfig(name, path, setValues)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}

	cfg.ControlPlane.Standalone.Enabled = true
	cfg.PrivateNodes.Enabled = true
	if cfg.ControlPlane.Standalone.DataDir == "" {
		cfg.ControlPlane.Standalone.DataDir = constants.VClusterStandaloneDefaultDataDir
	}

	err = ValidateConfigAndSetDefaults(cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

// resolveStandaloneConfigFromSystemd resolves a standalone config path from the
// local systemd unit, falling back to shared standalone default locations when
// the unit does not provide an explicit config flag. This is intended for
// host-side CLI discovery, not for in-process standalone runtime startup.
func resolveStandaloneConfigFromSystemd() (string, []byte, error) {
	unitData, found, err := standaloneutil.DetectStandaloneHost()
	if err != nil {
		return "", nil, err
	}
	if !found {
		return "", nil, os.ErrNotExist
	}

	if configPath, ok := resolveConfigPathFromExecStart(unitData); ok {
		return configPath, unitData, nil
	}

	configPath, err := resolveStandaloneConfigFromCandidates([]string{
		constants.VClusterStandaloneDefaultConfigPath,
		constants.DefaultVClusterConfigLocation,
	})
	if err != nil {
		return "", nil, err
	}
	return configPath, unitData, nil
}

func resolveStandaloneRuntimeName(unitData []byte) string {
	name := standaloneutil.ParseEnvFromSystemdUnit(unitData, "VCLUSTER_NAME")
	if name == "" {
		return constants.VClusterStandaloneDefaultName
	}
	return name
}

func resolveStandaloneConfigFromCandidates(candidates []string) (string, error) {
	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err == nil && !info.IsDir() {
			return candidate, nil
		}
		if err != nil && !os.IsNotExist(err) {
			return "", err
		}
	}

	return "", os.ErrNotExist
}

func resolveConfigPathFromExecStart(unitData []byte) (string, bool) {
	scanner := bufio.NewScanner(bytes.NewReader(unitData))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "ExecStart=") {
			continue
		}

		command := strings.TrimPrefix(line, "ExecStart=")
		args, err := shellquote.Split(command)
		if err != nil {
			continue
		}

		for i, arg := range args {
			if arg == "--config" && i+1 < len(args) {
				return args[i+1], true
			}
			if value, ok := strings.CutPrefix(arg, "--config="); ok && value != "" {
				return value, true
			}
		}
	}

	return "", false
}
