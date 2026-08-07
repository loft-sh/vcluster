package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/loft-sh/log"
	"github.com/loft-sh/log/scanner"
	"github.com/loft-sh/log/table"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/duration"
	"k8s.io/client-go/tools/clientcmd"
)

// dockerVCluster holds information about a docker-based vCluster
type dockerVCluster struct {
	Name      string
	Status    string
	Created   time.Time
	Connected bool
}

func ListDocker(ctx context.Context, options *ListOptions, globalFlags *flags.GlobalFlags, log log.Logger) error {
	// find all vcluster containers
	vClusters, err := findDockerContainer(ctx, constants.DockerControlPlanePrefix)
	if err != nil {
		return fmt.Errorf("failed to list docker vclusters: %w", err)
	}

	// get current context to check if connected
	rawConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	).RawConfig()
	if err != nil {
		log.Debugf("Failed to load kubeconfig: %v", err)
	}
	currentContext := rawConfig.CurrentContext

	// mark connected vclusters
	for i := range vClusters {
		expectedContext := "vcluster-docker_" + vClusters[i].Name
		vClusters[i].Connected = currentContext == expectedContext
	}

	// print output
	if options.Output == "json" {
		// convert to ListVCluster format for consistent JSON output
		output := make([]ListVCluster, len(vClusters))
		for i, vc := range vClusters {
			output[i] = ListVCluster{
				Name:       vc.Name,
				Namespace:  "docker", // use "docker" as namespace placeholder
				Status:     vc.Status,
				Created:    vc.Created,
				AgeSeconds: int(time.Since(vc.Created).Round(time.Second).Seconds()),
				Connected:  vc.Connected,
			}
		}

		bytes, err := json.MarshalIndent(output, "", "    ")
		if err != nil {
			return fmt.Errorf("json marshal vClusters: %w", err)
		}
		log.WriteString(logrus.InfoLevel, string(bytes)+"\n")
	} else {
		header := []string{"NAME", "STATUS", "CONNECTED", "AGE"}
		values := dockerVClustersToValues(vClusters)
		table.PrintTable(log, header, values)
	}

	return nil
}

func findDockerContainer(ctx context.Context, prefix string) ([]dockerVCluster, error) {
	// list all containers with name starting with the prefix
	args := []string{"ps", "-a", "--filter", "name=^" + prefix, "--format", "{{.ID}}"}
	out, err := exec.CommandContext(ctx, "docker", args...).Output()
	if err != nil {
		return nil, fmt.Errorf("docker ps failed: %w", err)
	}

	// parse container IDs
	var containerIDs []string
	scan := scanner.NewScanner(bytes.NewReader(out))
	for scan.Scan() {
		id := strings.TrimSpace(scan.Text())
		if id != "" {
			containerIDs = append(containerIDs, id)
		}
	}

	if len(containerIDs) == 0 {
		return nil, nil
	}

	// inspect each container to get details
	var vClusters []dockerVCluster
	for _, containerID := range containerIDs {
		details, err := inspectDockerContainerForList(ctx, containerID)
		if err != nil {
			continue // skip containers we can't inspect
		}

		// extract name from container name (remove prefix)
		name := strings.TrimPrefix(details.Name, "/"+prefix)
		if name == details.Name {
			// doesn't have the prefix, skip
			continue
		}

		// parse created time
		created, err := time.Parse(time.RFC3339, details.Created)
		if err != nil {
			created = time.Time{}
		}

		vClusters = append(vClusters, dockerVCluster{
			Name:    name,
			Status:  details.State.Status,
			Created: created,
		})
	}

	return vClusters, nil
}

// dockerInspectResult represents the result of docker inspect
type dockerInspectResult struct {
	Name    string               `json:"Name,omitempty"`
	Created string               `json:"Created,omitempty"`
	State   dockerContainerState `json:"State,omitempty"`
}

func inspectDockerContainerForList(ctx context.Context, containerID string) (*dockerInspectResult, error) {
	args := []string{"inspect", "--type", "container", containerID}
	out, err := exec.CommandContext(ctx, "docker", args...).Output()
	if err != nil {
		return nil, fmt.Errorf("docker inspect failed: %w", err)
	}

	var results []dockerInspectResult
	err = json.Unmarshal(out, &results)
	if err != nil {
		return nil, fmt.Errorf("failed to parse docker inspect output: %w", err)
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("container %s not found", containerID)
	}

	return &results[0], nil
}

func dockerVClustersToValues(vClusters []dockerVCluster) [][]string {
	var values [][]string
	for _, vc := range vClusters {
		isConnected := ""
		if vc.Connected {
			isConnected = "True"
		}

		age := ""
		if !vc.Created.IsZero() {
			age = duration.HumanDuration(time.Since(vc.Created))
		}

		values = append(values, []string{
			vc.Name,
			vc.Status,
			isConnected,
			age,
		})
	}
	return values
}
