package cli

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/snapshot"
)

// DockerSnapshotMetadata stores the metadata needed to restore a Docker-based vCluster.
type DockerSnapshotMetadata struct {
	// Name is the original vCluster name.
	Name string `json:"name"`
	// CreatedAt is the timestamp when the snapshot was created.
	CreatedAt time.Time `json:"createdAt"`
	// VClusterVersion is the vCluster version used to create the cluster.
	VClusterVersion string `json:"vclusterVersion,omitempty"`
	// Volumes lists the Docker volumes that were captured, keyed by their logical name.
	Volumes []SnapshotVolume `json:"volumes"`
	// ConfigDir is the relative path inside the archive to the config directory contents.
	ConfigDir string `json:"configDir"`
	// Nodes lists worker node names that were part of this cluster.
	Nodes []string `json:"nodes,omitempty"`
	// VClusterYAML is the vcluster.yaml config content, preserved so multi-node
	// configurations can be restored with the correct node definitions.
	VClusterYAML string `json:"vclusterYAML,omitempty"`
}

// SnapshotVolume describes a single Docker volume in the snapshot.
type SnapshotVolume struct {
	// LogicalName identifies the volume role (e.g. "cp.var", "node.worker-0.var").
	LogicalName string `json:"logicalName"`
	// ArchivePath is the relative path of this volume's tarball inside the snapshot archive.
	ArchivePath string `json:"archivePath"`
}

// SnapshotDocker creates a snapshot of a Docker-based vCluster by exporting its Docker volumes
// and configuration directory into a single gzipped tar archive.
func SnapshotDocker(ctx context.Context, globalFlags *flags.GlobalFlags, vClusterName, outputPath string, log log.Logger) error {
	cpContainer := getControlPlaneContainerName(vClusterName)

	// Verify the vCluster exists.
	exists, running, err := checkDockerContainerState(ctx, cpContainer)
	if err != nil {
		return fmt.Errorf("failed to check container state: %w", err)
	}
	if !exists {
		return fmt.Errorf("vCluster %q not found as a Docker-based cluster (no container %s). Docker snapshots only work with the Docker driver. For helm-based vClusters, use 'vcluster snapshot create' without --driver docker", vClusterName, cpContainer)
	}

	// Pause the vCluster to get a consistent snapshot.
	wasPaused := !running
	if !wasPaused {
		log.Infof("Pausing vCluster %s for consistent snapshot...", vClusterName)
		if err := PauseDocker(ctx, globalFlags, vClusterName, log); err != nil {
			return fmt.Errorf("failed to pause vCluster: %w", err)
		}
		defer func() {
			log.Infof("Resuming vCluster %s...", vClusterName)
			if err := ResumeDocker(ctx, globalFlags, vClusterName, log); err != nil {
				log.Warnf("Failed to resume vCluster after snapshot: %v", err)
			}
		}()
	}

	// Discover worker nodes.
	nodes, err := findDockerContainer(ctx, constants.DockerNodePrefix+vClusterName+".")
	if err != nil {
		return fmt.Errorf("failed to find worker nodes: %w", err)
	}

	// Build list of volumes to export.
	var volumes []SnapshotVolume
	for volName := range containerVolumes {
		volumes = append(volumes, SnapshotVolume{
			LogicalName: "cp." + volName,
			ArchivePath: "volumes/cp." + volName + ".tar",
		})
	}
	var nodeNames []string
	for _, node := range nodes {
		nodeNames = append(nodeNames, node.Name)
		for volName := range containerVolumes {
			logicalName := "node." + node.Name + "." + volName
			volumes = append(volumes, SnapshotVolume{
				LogicalName: logicalName,
				ArchivePath: "volumes/" + logicalName + ".tar",
			})
		}
	}

	// Read the version recorded when this cluster was created. If the .version
	// file is missing (clusters created before it existed), inspect the container's
	// bind mount for the vcluster binary to derive the version from the cache path.
	configDir := filepath.Join(filepath.Dir(globalFlags.Config), "docker", "vclusters", vClusterName)
	chartVersion := ""
	if data, err := os.ReadFile(filepath.Join(configDir, ".version")); err == nil && len(data) > 0 {
		chartVersion = string(data)
	} else {
		chartVersion = detectVersionFromContainer(ctx, cpContainer)
	}

	// Read the vcluster.yaml to preserve node configuration for multi-node restore.
	vclusterYAMLPath := filepath.Join(configDir, "vcluster.yaml")
	vclusterYAML, _ := os.ReadFile(vclusterYAMLPath)

	metadata := DockerSnapshotMetadata{
		Name:            vClusterName,
		CreatedAt:       time.Now().UTC(),
		VClusterVersion: chartVersion,
		Volumes:         volumes,
		ConfigDir:       "config",
		Nodes:           nodeNames,
		VClusterYAML:    string(vclusterYAML),
	}

	// For remote URLs (oci://, s3://), write to a temp file first, then push.
	isRemote := isRemoteSnapshotURL(outputPath)
	localPath := outputPath
	if isRemote {
		tmpFile, err := os.CreateTemp("", "vcluster-docker-snapshot-*.tar.gz")
		if err != nil {
			return fmt.Errorf("failed to create temp file: %w", err)
		}
		localPath = tmpFile.Name()
		tmpFile.Close()
		defer os.Remove(localPath)
	}

	// Create the output file.
	outFile, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("failed to create snapshot file: %w", err)
	}
	defer outFile.Close()

	gw := gzip.NewWriter(outFile)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()

	// Write metadata.
	metadataBytes, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}
	if err := addBytesToTar(tw, "metadata.json", metadataBytes); err != nil {
		return fmt.Errorf("failed to write metadata to archive: %w", err)
	}

	// Export each Docker volume. Volumes are streamed through a temp file to avoid
	// holding multi-GB volume data in memory.
	for _, vol := range volumes {
		dockerVolumeName := resolveDockerVolumeName(vClusterName, vol.LogicalName)
		log.Infof("Exporting volume %s...", vol.LogicalName)
		if err := exportDockerVolumeToTar(ctx, tw, vol.ArchivePath, dockerVolumeName); err != nil {
			return fmt.Errorf("failed to export volume %s: %w", vol.LogicalName, err)
		}
	}

	// Export the config directory.
	if _, err := os.Stat(configDir); err == nil {
		log.Infof("Exporting config directory...")
		if err := addDirToTar(tw, configDir, "config"); err != nil {
			return fmt.Errorf("failed to write config to archive: %w", err)
		}
	}

	// Close tar and gzip writers explicitly to ensure the file is complete
	// before any remote push. Deferred Close() calls are no-ops after this.
	if err := tw.Close(); err != nil {
		return fmt.Errorf("failed to finalize tar archive: %w", err)
	}
	if err := gw.Close(); err != nil {
		return fmt.Errorf("failed to finalize gzip compression: %w", err)
	}
	if err := outFile.Close(); err != nil {
		return fmt.Errorf("failed to close snapshot file: %w", err)
	}

	if isRemote {
		log.Infof("Pushing snapshot to %s...", outputPath)
		if err := pushDockerSnapshot(ctx, outputPath, localPath); err != nil {
			return fmt.Errorf("failed to push snapshot to %s: %w", outputPath, err)
		}
		log.Donef("Snapshot pushed to %s", outputPath)
	} else {
		log.Donef("Snapshot saved to %s", outputPath)
	}
	return nil
}

// detectVersionFromContainer inspects the container's bind mounts to find the vcluster
// binary source path, which is of the form <cacheDir>/<version>/vcluster. The parent
// directory name is the chart version.
func detectVersionFromContainer(ctx context.Context, containerName string) string {
	// Use docker inspect with a Go template to extract the source path of the
	// /var/lib/vcluster/bin/vcluster bind mount.
	out, err := exec.CommandContext(ctx, "docker", "inspect",
		"--format", `{{range .Mounts}}{{if eq .Destination "/var/lib/vcluster/bin/vcluster"}}{{.Source}}{{end}}{{end}}`,
		containerName,
	).Output()
	if err != nil {
		return ""
	}
	// source looks like: /Users/x/.vcluster/docker/vcluster/0.33.0/vcluster
	// The version is the parent directory name.
	source := strings.TrimSpace(string(out))
	if source == "" {
		return ""
	}
	return filepath.Base(filepath.Dir(source))
}

// resolveDockerVolumeName maps a logical volume name (e.g. "cp.var" or "node.worker-0.var")
// to the actual Docker volume name. For node volumes, the volume type (var, etc, bin, cni-bin)
// is parsed from the last dot-separated segment to handle node names that may contain dots.
func resolveDockerVolumeName(vClusterName, logicalName string) string {
	// logicalName format: "cp.<volName>" or "node.<nodeName>.<volName>"
	if strings.HasPrefix(logicalName, "cp.") {
		volName := strings.TrimPrefix(logicalName, "cp.")
		return getControlPlaneVolumeName(vClusterName, volName)
	}
	if strings.HasPrefix(logicalName, "node.") {
		// Split from the right: last segment is the volume name, middle is the node name.
		// This handles node names with dots (e.g. "node.my.worker.var" -> node="my.worker", vol="var").
		withoutPrefix := strings.TrimPrefix(logicalName, "node.")
		lastDot := strings.LastIndex(withoutPrefix, ".")
		if lastDot > 0 {
			nodeName := withoutPrefix[:lastDot]
			volName := withoutPrefix[lastDot+1:]
			return getWorkerVolumeName(vClusterName, nodeName, volName)
		}
	}
	return logicalName
}

// exportDockerVolumeToTar exports a Docker volume directly into a tar writer by streaming
// through a temporary file. This avoids holding multi-GB volume data in memory.
func exportDockerVolumeToTar(ctx context.Context, tw *tar.Writer, archivePath, volumeName string) error {
	// Write docker output to a temp file (uses disk, not RAM).
	tmpFile, err := os.CreateTemp("", "vcluster-vol-export-*.tar")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	args := []string{
		"run", "--rm",
		"-v", volumeName + ":/data:ro",
		"alpine",
		"tar", "cf", "-", "-C", "/data", ".",
	}
	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stdout = tmpFile
	if err := cmd.Run(); err != nil {
		tmpFile.Close()
		return fmt.Errorf("docker run tar failed for volume %s: %w", volumeName, err)
	}
	tmpFile.Close()

	// Get file size for the tar header.
	info, err := os.Stat(tmpFile.Name())
	if err != nil {
		return fmt.Errorf("stat temp file: %w", err)
	}

	// Write tar header and stream file contents into the archive.
	header := &tar.Header{
		Name:    archivePath,
		Size:    info.Size(),
		Mode:    0644,
		ModTime: time.Now(),
	}
	if err := tw.WriteHeader(header); err != nil {
		return fmt.Errorf("write tar header: %w", err)
	}

	f, err := os.Open(tmpFile.Name())
	if err != nil {
		return fmt.Errorf("open temp file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(tw, f); err != nil {
		return fmt.Errorf("stream volume to tar: %w", err)
	}

	return nil
}

// addBytesToTar writes a single file entry to a tar writer.
func addBytesToTar(tw *tar.Writer, name string, data []byte) error {
	header := &tar.Header{
		Name:    name,
		Size:    int64(len(data)),
		Mode:    0644,
		ModTime: time.Now(),
	}
	if err := tw.WriteHeader(header); err != nil {
		return err
	}
	_, err := tw.Write(data)
	return err
}

// addDirToTar recursively adds a directory to a tar writer under the given prefix.
func addDirToTar(tw *tar.Writer, srcDir, prefix string) error {
	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		archivePath := filepath.Join(prefix, relPath)

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = archivePath

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = io.Copy(tw, f)
		return err
	})
}

// writeToFile writes data from a reader to a file path using a buffered writer.
func writeToFile(reader io.Reader, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}

	// Use a 1MB buffer for large snapshot files to avoid streaming issues.
	buf := make([]byte, 1024*1024)
	if _, err := io.CopyBuffer(f, reader, buf); err != nil {
		f.Close()
		return fmt.Errorf("write file: %w", err)
	}

	if err := f.Sync(); err != nil {
		f.Close()
		return fmt.Errorf("sync file: %w", err)
	}
	return f.Close()
}

// isRemoteSnapshotURL returns true if the path is an OCI or S3 URL.
func isRemoteSnapshotURL(path string) bool {
	return strings.HasPrefix(path, "oci://") || strings.HasPrefix(path, "s3://") || strings.HasPrefix(path, "https://")
}

// pushDockerSnapshot pushes a local snapshot file to a remote storage backend (OCI, S3, Azure).
func pushDockerSnapshot(ctx context.Context, remoteURL, localPath string) error {
	opts := &snapshot.Options{}
	if err := opts.SetURLAndFillCredentials(ctx, remoteURL, false); err != nil {
		return fmt.Errorf("parse remote URL: %w", err)
	}

	store, err := snapshot.CreateStore(ctx, opts)
	if err != nil {
		return fmt.Errorf("create storage backend: %w", err)
	}

	f, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("open snapshot file: %w", err)
	}
	defer f.Close()

	return store.PutObject(ctx, f)
}

// pullDockerSnapshot pulls a snapshot from a remote storage backend to a local file.
func pullDockerSnapshot(ctx context.Context, remoteURL, localPath string) error {
	opts := &snapshot.Options{}
	if err := opts.SetURLAndFillCredentials(ctx, remoteURL, false); err != nil {
		return fmt.Errorf("parse remote URL: %w", err)
	}

	store, err := snapshot.CreateStore(ctx, opts)
	if err != nil {
		return fmt.Errorf("create storage backend: %w", err)
	}

	reader, err := store.GetObject(ctx)
	if err != nil {
		return fmt.Errorf("download snapshot: %w", err)
	}
	defer reader.Close()

	return writeToFile(reader, localPath)
}
