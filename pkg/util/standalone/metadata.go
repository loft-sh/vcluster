package standalone

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/loft-sh/vcluster/pkg/constants"
)

type RuntimeMetadata struct {
	Version   string `json:"version,omitempty"`
	IPAddress string `json:"ip,omitempty"`
	NodeClaim string `json:"nodeClaim,omitempty"`
	StartTime string `json:"startTime,omitempty"`
}

func LoadRuntimeMetadata(dataDir string) (*RuntimeMetadata, error) {
	raw, err := os.ReadFile(filepath.Join(dataDir, constants.StandaloneRuntimeMetadataFileName))
	if err != nil {
		return nil, err
	}

	metadata := &RuntimeMetadata{}
	if err := json.Unmarshal(raw, metadata); err != nil {
		return nil, fmt.Errorf("unmarshal runtime metadata: %w", err)
	}

	return metadata, nil
}

func ResolveStandaloneIPAddress(dataDir string) (string, error) {
	metadata, err := LoadRuntimeMetadata(dataDir)
	if err == nil {
		ipAddress := strings.TrimSpace(metadata.IPAddress)
		if ipAddress == "" {
			return "", fmt.Errorf("runtime metadata does not contain standalone IP address")
		}
		return ipAddress, nil
	}
	if !os.IsNotExist(err) {
		return "", err
	}

	ipAddress := strings.TrimSpace(os.Getenv(constants.VClusterStandaloneIPAddressEnvVar))
	if ipAddress == "" {
		return "", fmt.Errorf("could not determine the IP address for the embedded etcd peer")
	}

	return ipAddress, nil
}
