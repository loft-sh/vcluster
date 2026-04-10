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

// LoadRuntimeConfig loads the runtime config for the current process.
// When VCLUSTER_STANDALONE is set, it acts as an explicit mode override:
// "true" forces standalone loading and any other value forces the normal
// in-cluster/default config path. Only when the env var is unset we fall
// back to local standalone host detection via the systemd unit marker.
func LoadRuntimeConfig(vClusterName string) (*VirtualClusterConfig, error) {
	// Respect explicit runtime context from the caller before attempting any
	// local host detection. This allows parent processes to force standalone or
	// non-standalone behavior for child vcluster commands.
	if standaloneEnv, ok := os.LookupEnv(constants.VClusterStandaloneEnvVar); ok {
		if standaloneEnv == "true" {
			vConfig, err := LoadStandaloneConfig(vClusterName, "", nil)
			if err != nil {
				return nil, fmt.Errorf("loading standalone vCluster config: %w", err)
			}
			return vConfig, nil
		}

		vConfig, err := LoadConfig(vClusterName, "", nil)
		if err != nil {
			return nil, fmt.Errorf("loading vCluster config: %w", err)
		}
		return vConfig, nil
	}

	vConfig, err := LoadStandaloneConfigFromSystemd(vClusterName, nil)
	if err == nil {
		return vConfig, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("loading standalone vCluster config from systemd: %w", err)
	}

	vConfig, err = LoadConfig(vClusterName, "", nil)
	if err != nil {
		return nil, fmt.Errorf("loading vCluster config: %w", err)
	}
	return vConfig, nil
}

// LoadConfig loads a vCluster config from an explicit path or from the default
// in-cluster config location when the path is empty.
func LoadConfig(name, path string, setValues []string) (*VirtualClusterConfig, error) {
	if path == "" {
		path = constants.DefaultVClusterConfigLocation
	}

	return ParseConfig(path, name, setValues)
}

// LoadStandaloneConfig loads a standalone vCluster config from an explicit path
// or from shared default standalone locations if the path is empty.
func LoadStandaloneConfig(name, path string, setValues []string) (*VirtualClusterConfig, error) {
	var err error

	if path == "" {
		path, err = resolveStandaloneConfigFromDefaults()
		if err != nil {
			return nil, err
		}
	}
	return loadNormalizedStandaloneConfig(name, path, setValues)
}

// LoadStandaloneConfigFromSystemd loads a standalone vCluster config for
// host-side CLI flows such as vclusterctl. This is useful when the caller runs
// on the standalone host and needs to discover the installed config from the
// local systemd unit, rather than loading config for an already-running runtime
// process.
func LoadStandaloneConfigFromSystemd(name string, setValues []string) (*VirtualClusterConfig, error) {
	resolvedPath, err := resolveStandaloneConfigFromSystemd()
	if err != nil {
		return nil, err
	}

	return loadNormalizedStandaloneConfig(name, resolvedPath, setValues)
}

func loadNormalizedStandaloneConfig(name, path string, setValues []string) (*VirtualClusterConfig, error) {
	cfg, err := LoadConfig(name, path, setValues)
	if err != nil {
		return nil, err
	}

	err = normalizeStandaloneConfig(cfg)
	if err != nil {
		return nil, err
	}

	err = ValidateConfigAndSetDefaults(cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func normalizeStandaloneConfig(cfg *VirtualClusterConfig) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}

	cfg.ControlPlane.Standalone.Enabled = true
	cfg.PrivateNodes.Enabled = true
	if cfg.ControlPlane.Standalone.DataDir == "" {
		cfg.ControlPlane.Standalone.DataDir = constants.StandaloneDefaultDataDir
	}

	return nil
}

// resolveStandaloneConfigFromSystemd resolves a standalone config path from the
// local systemd unit. This is intended for host-side CLI discovery, not for
// in-process standalone runtime startup.
func resolveStandaloneConfigFromSystemd() (string, error) {
	unitData, found, err := standaloneutil.DetectStandaloneHost()
	if err != nil {
		return "", err
	}
	if !found {
		return "", os.ErrNotExist
	}

	return resolveStandaloneConfigFromSystemdUnit(unitData)
}

// resolveStandaloneConfigFromDefaults resolves the config path from shared
// standalone default locations only. This intentionally does not inspect systemd.
// The order mirrors the previous pro standalone parser behavior:
// standalone default first, then container default.
func resolveStandaloneConfigFromDefaults() (string, error) {
	return resolveStandaloneConfigFromCandidates([]string{
		constants.StandaloneDefaultConfigPath,
		constants.DefaultVClusterConfigLocation,
	})
}

func resolveStandaloneConfigFromSystemdUnit(unitData []byte) (string, error) {
	if configPath, ok := resolveConfigPathFromExecStart(unitData); ok {
		return configPath, nil
	}

	return resolveStandaloneConfigFromDefaults()
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
