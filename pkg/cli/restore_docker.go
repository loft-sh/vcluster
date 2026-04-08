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

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/upgrade"
)

// RestoreDocker restores a Docker-based vCluster from a snapshot archive created by SnapshotDocker.
// If targetName is empty, the original vCluster name from the snapshot is used.
//
// The archive is processed in a single streaming pass to avoid loading multi-GB snapshots
// into memory. Tar entries are handled as they arrive: metadata is parsed first, volume
// entries are piped directly to Docker, and config files are written directly to disk.
// callerOpts may be nil (standalone restore) or the user's CreateOptions (create --restore).
func RestoreDocker(ctx context.Context, globalFlags *flags.GlobalFlags, snapshotPath, targetName string, callerOpts *CreateOptions, log log.Logger) error {
	// If the snapshot path is a remote URL, pull it to a temp file first.
	localPath := snapshotPath
	if isRemoteSnapshotURL(snapshotPath) {
		log.Infof("Pulling snapshot from %s...", snapshotPath)
		tmpFile, err := os.CreateTemp("", "vcluster-docker-restore-*.tar.gz")
		if err != nil {
			return fmt.Errorf("failed to create temp file: %w", err)
		}
		localPath = tmpFile.Name()
		tmpFile.Close()
		defer os.Remove(localPath)

		if err := pullDockerSnapshot(ctx, snapshotPath, localPath); err != nil {
			return fmt.Errorf("failed to pull snapshot from %s: %w", snapshotPath, err)
		}
		log.Donef("Snapshot downloaded to temp file")
	}

	// Open and decompress the snapshot archive.
	f, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open snapshot file: %w", err)
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)

	// First entry must be metadata.json.
	header, err := tr.Next()
	if err != nil {
		return fmt.Errorf("failed to read first tar entry: %w", err)
	}
	if header.Name != "metadata.json" {
		return fmt.Errorf("expected metadata.json as first entry, got %s", header.Name)
	}
	metadataBytes, err := io.ReadAll(tr)
	if err != nil {
		return fmt.Errorf("failed to read metadata: %w", err)
	}

	var metadata DockerSnapshotMetadata
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		return fmt.Errorf("failed to parse snapshot metadata: %w", err)
	}

	vClusterName := targetName
	if vClusterName == "" {
		vClusterName = metadata.Name
	}

	// Check that a vCluster with this name doesn't already exist.
	cpContainer := getControlPlaneContainerName(vClusterName)
	exists, _, err := checkDockerContainerState(ctx, cpContainer)
	if err != nil {
		return fmt.Errorf("failed to check container state: %w", err)
	}
	if exists {
		return fmt.Errorf("vCluster %q already exists; delete it first or use a different name", vClusterName)
	}

	log.Infof("Restoring Docker-based vCluster from snapshot (note: Docker snapshots can only be restored as Docker-based vClusters, not helm or platform-based)")

	// Prepare config directory.
	configDir := filepath.Join(filepath.Dir(globalFlags.Config), "docker", "vclusters", vClusterName)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Stream remaining entries: volumes are piped directly to Docker,
	// config files are written directly to disk. No buffering.
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar entry: %w", err)
		}

		switch {
		case strings.HasPrefix(header.Name, "volumes/"):
			// Resolve the Docker volume name from the archive path.
			logicalName := strings.TrimPrefix(header.Name, "volumes/")
			logicalName = strings.TrimSuffix(logicalName, ".tar")
			dockerVolumeName := resolveDockerVolumeName(vClusterName, logicalName)

			log.Infof("Restoring volume %s...", logicalName)
			if err := importDockerVolumeFromReader(ctx, dockerVolumeName, tr); err != nil {
				return fmt.Errorf("failed to restore volume %s: %w", logicalName, err)
			}

		case strings.HasPrefix(header.Name, "config/"):
			relPath := strings.TrimPrefix(header.Name, "config/")
			if relPath == "" || relPath == "." {
				continue
			}
			destPath := filepath.Join(configDir, relPath)

			// Prevent path traversal attacks (e.g. config/../../.ssh/authorized_keys).
			if !strings.HasPrefix(filepath.Clean(destPath), filepath.Clean(configDir)+string(os.PathSeparator)) {
				return fmt.Errorf("snapshot contains path traversal entry: %s", header.Name)
			}

			if header.Typeflag == tar.TypeDir {
				if err := os.MkdirAll(destPath, 0755); err != nil {
					return fmt.Errorf("failed to create directory %s: %w", destPath, err)
				}
				continue
			}

			if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory for %s: %w", destPath, err)
			}
			outFile, err := os.Create(destPath)
			if err != nil {
				return fmt.Errorf("failed to create config file %s: %w", destPath, err)
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return fmt.Errorf("failed to write config file %s: %w", destPath, err)
			}
			outFile.Close()
		}
	}

	// Start the cluster using the restored volumes.
	log.Infof("Starting vCluster %s from restored snapshot...", vClusterName)
	chartVersion := metadata.VClusterVersion
	if chartVersion == "" {
		return fmt.Errorf("snapshot does not contain a vCluster version; please create the cluster manually: vcluster create %s --chart-version <version>", vClusterName)
	}

	// Use caller's CreateOptions if provided (from vcluster create --restore),
	// otherwise build minimal options from snapshot metadata.
	createOpts := callerOpts
	if createOpts == nil {
		createOpts = &CreateOptions{
			Connect:       true,
			UpdateCurrent: true,
		}
	}
	if createOpts.ChartVersion == "" || createOpts.ChartVersion == upgrade.DevelopmentVersion {
		createOpts.ChartVersion = chartVersion
	}

	// If restoring to a different name, write a marker file with the original
	// hostname so CreateDocker uses it instead of the new cluster name. This
	// ensures the kubelet re-registers with the node name that matches etcd data.
	if metadata.Name != "" && metadata.Name != vClusterName {
		hostnameFile := filepath.Join(configDir, ".restore-hostname")
		_ = os.WriteFile(hostnameFile, []byte(metadata.Name), 0644)
	}

	// If the snapshot contains a vcluster.yaml, pass it as a values file so that
	// any custom configuration (including multi-node definitions) is preserved.
	if metadata.VClusterYAML != "" {
		tmpFile, err := os.CreateTemp("", "vcluster-restore-*.yaml")
		if err != nil {
			return fmt.Errorf("failed to create temp values file: %w", err)
		}
		defer os.Remove(tmpFile.Name())
		if _, err := tmpFile.WriteString(metadata.VClusterYAML); err != nil {
			return fmt.Errorf("failed to write temp values file: %w", err)
		}
		tmpFile.Close()
		createOpts.Values = append(createOpts.Values, tmpFile.Name())
	}

	if err := CreateDocker(ctx, createOpts, globalFlags, vClusterName, log); err != nil {
		return fmt.Errorf("failed to create vCluster from restored snapshot: %w", err)
	}

	log.Donef("Successfully restored vCluster %s from snapshot", vClusterName)
	return nil
}

// importDockerVolumeFromReader creates a Docker volume and populates it by piping
// data from a reader directly to a Docker container. No intermediate buffering.
func importDockerVolumeFromReader(ctx context.Context, volumeName string, reader io.Reader) error {
	createArgs := []string{"volume", "create", volumeName}
	if out, err := exec.CommandContext(ctx, "docker", createArgs...).CombinedOutput(); err != nil {
		return fmt.Errorf("docker volume create failed: %w, output: %s", err, string(out))
	}

	importArgs := []string{
		"run", "--rm", "-i",
		"-v", volumeName + ":/data",
		"alpine",
		"tar", "xf", "-", "-C", "/data",
	}
	cmd := exec.CommandContext(ctx, "docker", importArgs...)
	cmd.Stdin = reader
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("docker volume import failed: %w, output: %s", err, string(out))
	}

	return nil
}
