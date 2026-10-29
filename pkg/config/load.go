package config

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/kballard/go-shellquote"
	vclusterconfig "github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/constants"
	standaloneutil "github.com/loft-sh/vcluster/pkg/util/standalone"
	"sigs.k8s.io/yaml"
)

// LoadConfig loads config for commands that should use the current
// runtime context. It first respects the explicit VCLUSTER_STANDALONE
// environment override, then falls back to local standalone host detection, and
// finally falls back to the regular in-cluster config path.
//
// When VCLUSTER_STANDALONE is set, it acts as an explicit mode override:
// "true" forces standalone loading and any other value forces the normal
// in-cluster/default config path. Only when the env var is unset we fall back
// to local standalone host detection via the systemd unit marker.
func LoadConfig(vClusterName string) (*VirtualClusterConfig, error) {
	// Respect explicit runtime context from the caller before attempting any
	// local host detection. This allows parent processes to force standalone or
	// non-standalone behavior for child vcluster commands.
	if standaloneEnv, ok := os.LookupEnv(constants.VClusterStandaloneEnvVar); ok {
		if standaloneEnv == "true" {
			vConfig, err := LoadStandaloneConfig(vClusterName, nil)
			if err != nil {
				return nil, fmt.Errorf("loading standalone vCluster config: %w", err)
			}
			return vConfig, nil
		}

		vConfig, err := LoadInClusterConfig(vClusterName, "", nil)
		if err != nil {
			return nil, fmt.Errorf("loading vCluster config: %w", err)
		}
		return vConfig, nil
	}

	vConfig, err := LoadStandaloneConfig(vClusterName, nil)
	if err == nil {
		return vConfig, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("loading standalone vCluster config from systemd: %w", err)
	}

	vConfig, err = LoadInClusterConfig(vClusterName, "", nil)
	if err != nil {
		return nil, fmt.Errorf("loading vCluster config: %w", err)
	}
	return vConfig, nil
}

// LoadInClusterConfig loads config for a vCluster running as a regular
// in-cluster control plane. When path is empty, it falls back to the default
// pod/container config location. It does not perform standalone detection or
// standalone normalization.
func LoadInClusterConfig(name, path string, setValues []string) (*VirtualClusterConfig, error) {
	if path == "" {
		path = constants.DefaultVClusterConfigLocation
	}

	return ParseConfig(path, name, setValues)
}

// LoadStandaloneConfig loads config for a standalone vCluster installed on
// this host. It discovers the config path from the local systemd unit, uses the
// runtime vCluster name from that unit when name is empty, and then delegates
// to LoadStandaloneRuntimeConfig. This is for host-side CLI/API flows such as
// vclusterctl, not for standalone runtime startup.
func LoadStandaloneConfig(name string, setValues []string) (*VirtualClusterConfig, error) {
	resolvedPath, unitData, err := resolveStandaloneConfigFromSystemd()
	if err != nil {
		return nil, err
	}
	if name == "" {
		name = resolveStandaloneRuntimeName(unitData)
	}

	return LoadStandaloneRuntimeConfig(name, resolvedPath, setValues)
}

// LoadStandaloneRuntimeConfig loads a standalone config from the given path,
// applies set values, overlays the chart default config values, and normalizes
// it as standalone. When the path is one of the default standalone config
// locations and the file does not exist, it falls back to an in-memory default
// config before merging chart defaults. Missing custom config paths still
// return an error.
func LoadStandaloneRuntimeConfig(name, path string, setValues []string) (*VirtualClusterConfig, error) {
	if name == "" {
		return nil, fmt.Errorf("empty vCluster name")
	}

	rawConfig, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) || !isDefaultStandaloneConfigPath(path) {
			return nil, err
		}
		rawConfig = nil
	}

	if err := validateStrictStandaloneRawConfig(rawConfig); err != nil {
		return nil, err
	}

	rawConfig, err = applySetValues(rawConfig, setValues)
	if err != nil {
		return nil, err
	}

	mergedConfig, err := mergeConfigBytesWithDefaults(rawConfig)
	if err != nil {
		return nil, err
	}

	cfg, err := ParseStandaloneConfigBytes(mergedConfig, name, nil)
	if err != nil {
		return nil, err
	}
	cfg.Path = path

	return cfg, nil
}

// ResolveStandaloneConfigPath resolves the standalone config path from an
// explicit path or the default standalone locations. When path is empty or
// points at the shared in-cluster default location, it prefers the standalone
// default path and falls back to the in-cluster default path if that file
// exists. If neither file exists, it returns the standalone default path.
func ResolveStandaloneConfigPath(path string) (string, error) {
	if path != "" && path != constants.DefaultVClusterConfigLocation {
		return path, nil
	}

	configPath, err := resolveStandaloneConfigFromCandidates([]string{
		constants.VClusterStandaloneDefaultConfigPath,
		constants.DefaultVClusterConfigLocation,
	})
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		return constants.VClusterStandaloneDefaultConfigPath, nil
	}

	return configPath, nil
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

	configPath, err := ResolveStandaloneConfigPath("")
	if err != nil {
		return "", nil, err
	}
	return configPath, unitData, nil
}

func isDefaultStandaloneConfigPath(path string) bool {
	return path == constants.VClusterStandaloneDefaultConfigPath || path == constants.DefaultVClusterConfigLocation
}

func validateStrictStandaloneRawConfig(rawConfig []byte) error {
	if len(rawConfig) == 0 {
		return nil
	}

	return yaml.UnmarshalStrict(rawConfig, &vclusterconfig.Config{})
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
