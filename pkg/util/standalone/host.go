package standalone

import (
	"bufio"
	"bytes"
	"os"
	"strings"

	"github.com/loft-sh/vcluster/pkg/constants"
)

// DetectStandaloneHost returns the local standalone systemd unit contents when
// running on a standalone host. Missing standalone markers are reported as
// found=false without an error.
func DetectStandaloneHost() ([]byte, bool, error) {
	return detectStandaloneHostFromPath(constants.VClusterServiceFile)
}

func detectStandaloneHostFromPath(path string) ([]byte, bool, error) {
	unitData, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}

	return unitData, true, nil
}

// ParseEnvFromSystemdUnit extracts the value of an Environment="KEY=value"
// directive from a systemd unit file. Returns empty string if not found.
func ParseEnvFromSystemdUnit(data []byte, key string) string {
	prefix := "Environment=\"" + key + "="
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, prefix) {
			val := strings.TrimPrefix(line, prefix)
			val = strings.TrimSuffix(val, "\"")
			return val
		}
	}
	return ""
}
