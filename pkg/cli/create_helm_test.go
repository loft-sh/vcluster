package cli

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"os"
	"testing"

	snapshotapi "github.com/loft-sh/api/v4/pkg/snapshot"
	"github.com/loft-sh/log"
	"gotest.tools/v3/assert"
)

// snapshotEntry is a single tar entry for makeContainerSnapshot.
// An ordered slice is used instead of a map so tar entry order is deterministic;
// getVClusterConfigFromSnapshot reads only the first entry, so ordering matters.
type snapshotEntry struct {
	name string
	data []byte
}

// makeContainerSnapshot writes a minimal .tar.gz to a temp file and returns its path.
func makeContainerSnapshot(t *testing.T, entries []snapshotEntry) string {
	t.Helper()

	f, err := os.CreateTemp(t.TempDir(), "snapshot-*.tar.gz")
	assert.NilError(t, err)
	defer f.Close()

	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)

	for _, e := range entries {
		hdr := &tar.Header{Name: e.name, Size: int64(len(e.data)), Mode: 0644}
		assert.NilError(t, tw.WriteHeader(hdr))
		_, err := tw.Write(e.data)
		assert.NilError(t, err)
	}

	assert.NilError(t, tw.Close())
	assert.NilError(t, gw.Close())
	return f.Name()
}

// TestBuildExtraValuesErrorsOnInaccessibleSnapshot verifies that when --restore is
// set and the snapshot cannot be fetched, buildExtraValues returns an error instead
// of silently continuing.
func TestBuildExtraValuesErrorsOnInaccessibleSnapshot(t *testing.T) {
	cmd := &CreateOptions{
		Restore: "container:///nonexistent/snapshot.tar.gz",
	}

	_, err := buildExtraValues(context.Background(), cmd, log.Discard)
	assert.ErrorContains(t, err, "get vCluster config from snapshot")
}

// TestBuildExtraValuesSkipsSnapshotWhenValuesProvided verifies that when the caller
// already provides --values or --set, buildExtraValues skips the snapshot fetch
// entirely (so an inaccessible snapshot does not cause an error).
func TestBuildExtraValuesSkipsSnapshotWhenValuesProvided(t *testing.T) {
	// Use a string with characters that are invalid base64 so it is passed
	// through as a raw path, and that is clearly not a real filesystem path
	// to avoid any accidental base64 decode of an existing file.
	const sentinel = "not!base64/values"

	cmd := &CreateOptions{
		Restore: "container:///nonexistent/snapshot.tar.gz",
		Values:  []string{sentinel},
	}

	filesToRemove, err := buildExtraValues(context.Background(), cmd, log.Discard)
	assert.NilError(t, err)
	assert.Equal(t, 0, len(filesToRemove))
	assert.Equal(t, 1, len(cmd.Values))
	assert.Equal(t, sentinel, cmd.Values[0])
}

// TestBuildExtraValuesSkipsSnapshotWhenSetValuesProvided mirrors the Values case
// for --set, covering the other half of the `len == 0 && len == 0` gate.
func TestBuildExtraValuesSkipsSnapshotWhenSetValuesProvided(t *testing.T) {
	cmd := &CreateOptions{
		Restore:   "container:///nonexistent/snapshot.tar.gz",
		SetValues: []string{"key=value"},
	}

	_, err := buildExtraValues(context.Background(), cmd, log.Discard)
	assert.NilError(t, err)
	// SetValues must not be modified: buildExtraValues only reads it.
	assert.Equal(t, 1, len(cmd.SetValues))
	assert.Equal(t, "key=value", cmd.SetValues[0])
}

// TestBuildExtraValuesExtractsConfigFromSnapshot verifies that when --restore points
// to an accessible snapshot that contains a Helm release entry, buildExtraValues
// extracts the values and writes them to a temp file.
func TestBuildExtraValuesExtractsConfigFromSnapshot(t *testing.T) {
	release := snapshotapi.HelmRelease{
		ChartVersion: "v0.33.0",
		Values:       []byte("controlPlane:\n  distro:\n    k8s: {}\n"),
	}
	releaseJSON, err := json.Marshal(release)
	assert.NilError(t, err)

	snapshotPath := makeContainerSnapshot(t, []snapshotEntry{
		{name: snapshotapi.SnapshotReleaseKey, data: releaseJSON},
	})

	cmd := &CreateOptions{
		Restore: "container://" + snapshotPath,
	}

	filesToRemove, err := buildExtraValues(context.Background(), cmd, log.Discard)
	assert.NilError(t, err)
	assert.Equal(t, 1, len(filesToRemove))
	t.Cleanup(func() {
		for _, f := range filesToRemove {
			os.Remove(f)
		}
	})

	data, err := os.ReadFile(filesToRemove[0])
	assert.NilError(t, err)
	assert.Equal(t, string(release.Values), string(data))
	assert.Equal(t, "v0.33.0", cmd.ChartVersion)
}

// TestBuildExtraValuesNoRestoreIsNoop verifies that without --restore, the function
// is a no-op and returns no files to clean up.
func TestBuildExtraValuesNoRestoreIsNoop(t *testing.T) {
	cmd := &CreateOptions{}

	filesToRemove, err := buildExtraValues(context.Background(), cmd, log.Discard)
	assert.NilError(t, err)
	assert.Equal(t, 0, len(filesToRemove))
}
