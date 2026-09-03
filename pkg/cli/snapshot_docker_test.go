package cli

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gotest.tools/v3/assert"
)

func TestResolveDockerVolumeName(t *testing.T) {
	tests := []struct {
		name         string
		vClusterName string
		logicalName  string
		expected     string
	}{
		{
			name:         "control plane var volume",
			vClusterName: "my-cluster",
			logicalName:  "cp.var",
			expected:     "vcluster.cp.my-cluster.var",
		},
		{
			name:         "control plane etc volume",
			vClusterName: "my-cluster",
			logicalName:  "cp.etc",
			expected:     "vcluster.cp.my-cluster.etc",
		},
		{
			name:         "worker node var volume",
			vClusterName: "my-cluster",
			logicalName:  "node.worker-0.var",
			expected:     "vcluster.node.my-cluster.worker-0.var",
		},
		{
			name:         "worker node etc volume",
			vClusterName: "test",
			logicalName:  "node.my-node.etc",
			expected:     "vcluster.node.test.my-node.etc",
		},
		{
			name:         "worker node name with dots",
			vClusterName: "test",
			logicalName:  "node.my.dotted.node.var",
			expected:     "vcluster.node.test.my.dotted.node.var",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolveDockerVolumeName(tt.vClusterName, tt.logicalName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDockerSnapshotMetadataRoundTrip(t *testing.T) {
	original := DockerSnapshotMetadata{
		Name:            "test-cluster",
		CreatedAt:       time.Date(2026, 4, 4, 12, 0, 0, 0, time.UTC),
		VClusterVersion: "v0.33.0",
		Volumes: []SnapshotVolume{
			{LogicalName: "cp.var", ArchivePath: "volumes/cp.var.tar"},
			{LogicalName: "cp.etc", ArchivePath: "volumes/cp.etc.tar"},
			{LogicalName: "node.worker-0.var", ArchivePath: "volumes/node.worker-0.var.tar"},
		},
		ConfigDir: "config",
		Nodes:     []string{"worker-0"},
	}

	data, err := json.Marshal(original)
	assert.NilError(t, err)

	var restored DockerSnapshotMetadata
	err = json.Unmarshal(data, &restored)
	assert.NilError(t, err)

	assert.Equal(t, original.Name, restored.Name)
	assert.Equal(t, original.VClusterVersion, restored.VClusterVersion)
	assert.Equal(t, original.ConfigDir, restored.ConfigDir)
	assert.Equal(t, len(original.Volumes), len(restored.Volumes))
	assert.Equal(t, len(original.Nodes), len(restored.Nodes))

	for i, vol := range original.Volumes {
		assert.Equal(t, vol.LogicalName, restored.Volumes[i].LogicalName)
		assert.Equal(t, vol.ArchivePath, restored.Volumes[i].ArchivePath)
	}
}

func TestAddBytesToTar(t *testing.T) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	testData := []byte(`{"name": "test"}`)
	err := addBytesToTar(tw, "metadata.json", testData)
	assert.NilError(t, err)
	assert.NilError(t, tw.Close())

	// Read back and verify.
	tr := tar.NewReader(&buf)
	header, err := tr.Next()
	assert.NilError(t, err)
	assert.Equal(t, "metadata.json", header.Name)
	assert.Equal(t, int64(len(testData)), header.Size)

	readBack, err := io.ReadAll(tr)
	assert.NilError(t, err)
	assert.Equal(t, string(testData), string(readBack))
}

func TestAddDirToTar(t *testing.T) {
	// Create a temp directory with some files.
	tmpDir := t.TempDir()
	assert.NilError(t, os.WriteFile(filepath.Join(tmpDir, "vcluster.yaml"), []byte("test: true"), 0644))
	assert.NilError(t, os.MkdirAll(filepath.Join(tmpDir, "subdir"), 0755))
	assert.NilError(t, os.WriteFile(filepath.Join(tmpDir, "subdir", "nested.txt"), []byte("nested"), 0644))

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	err := addDirToTar(tw, tmpDir, "config")
	assert.NilError(t, err)
	assert.NilError(t, tw.Close())

	// Verify archive contents.
	tr := tar.NewReader(&buf)
	files := make(map[string]string)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		assert.NilError(t, err)
		if !header.FileInfo().IsDir() {
			data, err := io.ReadAll(tr)
			assert.NilError(t, err)
			files[header.Name] = string(data)
		}
	}

	assert.Equal(t, "test: true", files["config/vcluster.yaml"])
	assert.Equal(t, "nested", files["config/subdir/nested.txt"])
}

func TestRestoreDockerMetadataParsing(t *testing.T) {
	// Build a minimal snapshot archive in memory.
	metadata := DockerSnapshotMetadata{
		Name:      "test-cluster",
		CreatedAt: time.Now().UTC(),
		Volumes:   []SnapshotVolume{},
		ConfigDir: "config",
	}
	metaBytes, err := json.MarshalIndent(metadata, "", "  ")
	assert.NilError(t, err)

	var archiveBuf bytes.Buffer
	gw := gzip.NewWriter(&archiveBuf)
	tw := tar.NewWriter(gw)

	err = addBytesToTar(tw, "metadata.json", metaBytes)
	assert.NilError(t, err)
	assert.NilError(t, tw.Close())
	assert.NilError(t, gw.Close())

	// Parse it back.
	gr, err := gzip.NewReader(&archiveBuf)
	assert.NilError(t, err)

	tr := tar.NewReader(gr)
	header, err := tr.Next()
	assert.NilError(t, err)
	assert.Equal(t, "metadata.json", header.Name)

	data, err := io.ReadAll(tr)
	assert.NilError(t, err)

	var parsed DockerSnapshotMetadata
	err = json.Unmarshal(data, &parsed)
	assert.NilError(t, err)
	assert.Equal(t, "test-cluster", parsed.Name)
}
