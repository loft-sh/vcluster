package standalone

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/loft-sh/vcluster/pkg/constants"
	"gotest.tools/v3/assert"
)

func TestLoadRuntimeMetadata(t *testing.T) {
	dataDir := t.TempDir()
	err := os.WriteFile(filepath.Join(dataDir, constants.StandaloneRuntimeMetadataFileName), []byte(`{"version":"v0.30.0","ip":"10.0.0.12","nodeClaim":"node-claim-1","startTime":"2026-04-21T10:00:00Z"}`), 0644)
	assert.NilError(t, err)

	metadata, err := LoadRuntimeMetadata(dataDir)
	assert.NilError(t, err)
	assert.Equal(t, metadata.Version, "v0.30.0")
	assert.Equal(t, metadata.IPAddress, "10.0.0.12")
	assert.Equal(t, metadata.NodeClaim, "node-claim-1")
	assert.Equal(t, metadata.StartTime, "2026-04-21T10:00:00Z")
}

func TestResolveStandaloneIPAddress(t *testing.T) {
	t.Run("metadata first", func(t *testing.T) {
		t.Setenv(constants.VClusterStandaloneIPAddressEnvVar, "10.0.0.99")
		dataDir := t.TempDir()
		err := os.WriteFile(filepath.Join(dataDir, constants.StandaloneRuntimeMetadataFileName), []byte(`{"ip":"10.0.0.12"}`), 0644)
		assert.NilError(t, err)

		ipAddress, err := ResolveStandaloneIPAddress(dataDir)
		assert.NilError(t, err)
		assert.Equal(t, ipAddress, "10.0.0.12")
	})

	t.Run("env fallback when metadata file missing", func(t *testing.T) {
		t.Setenv(constants.VClusterStandaloneIPAddressEnvVar, "10.0.0.99")

		ipAddress, err := ResolveStandaloneIPAddress(t.TempDir())
		assert.NilError(t, err)
		assert.Equal(t, ipAddress, "10.0.0.99")
	})

	t.Run("error when metadata and env missing", func(t *testing.T) {
		_, err := ResolveStandaloneIPAddress(t.TempDir())
		assert.ErrorContains(t, err, "could not determine the IP address")
	})

	t.Run("error when metadata exists without ip", func(t *testing.T) {
		dataDir := t.TempDir()
		err := os.WriteFile(filepath.Join(dataDir, constants.StandaloneRuntimeMetadataFileName), []byte(`{"version":"v0.30.0"}`), 0644)
		assert.NilError(t, err)

		_, err = ResolveStandaloneIPAddress(dataDir)
		assert.ErrorContains(t, err, "runtime metadata does not contain standalone IP address")
	})
}
