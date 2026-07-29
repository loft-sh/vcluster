package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"text/tabwriter"
	"time"

	"github.com/ghodss/yaml"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubectl/pkg/describe"
)

// DockerDescribeOutput holds information about a docker-based vCluster
type DockerDescribeOutput struct {
	Name           string      `json:"name,omitempty"`
	ContainerID    string      `json:"containerId,omitempty"`
	Status         string      `json:"status,omitempty"`
	Created        metav1.Time `json:"created,omitempty"`
	Image          string      `json:"image,omitempty"`
	Ports          string      `json:"ports,omitempty"`
	UserConfigYaml *string     `json:"userConfigYaml,omitempty"`
}

func (do *DockerDescribeOutput) String() string {
	out := &tabwriter.Writer{}
	buf := &bytes.Buffer{}
	out.Init(buf, 0, 8, 2, ' ', 0)

	w := describe.NewPrefixWriter(out)
	w.Write(describe.LEVEL_0, "Name:\t%s\n", do.Name)
	w.Write(describe.LEVEL_0, "Container ID:\t%s\n", do.ContainerID)
	w.Write(describe.LEVEL_0, "Status:\t%s\n", do.Status)
	if !do.Created.IsZero() {
		w.Write(describe.LEVEL_0, "Created:\t%s\n", do.Created.Time.Format(time.RFC1123Z))
	}
	w.Write(describe.LEVEL_0, "Image:\t%s\n", do.Image)
	if do.Ports != "" {
		w.Write(describe.LEVEL_0, "Ports:\t%s\n", do.Ports)
	}

	if do.UserConfigYaml != nil {
		userConfigYaml, isTruncated := truncateString(*do.UserConfigYaml, "\n", 50)
		w.Write(describe.LEVEL_0, "\n------------------- vcluster.yaml -------------------\n")
		w.Write(describe.LEVEL_0, "%s\n", userConfigYaml)
		if isTruncated {
			w.Write(describe.LEVEL_0, "... (truncated)\n")
		}
		w.Write(describe.LEVEL_0, "-----------------------------------------------------\n")
		if isTruncated {
			w.Write(describe.LEVEL_0, "Use --config-only to retrieve the full vcluster.yaml only\n")
		} else {
			w.Write(describe.LEVEL_0, "Use --config-only to retrieve just the vcluster.yaml\n")
		}
	}

	out.Flush()
	return buf.String()
}

// dockerInspectDetails represents detailed docker inspect output
type dockerInspectDetails struct {
	ID      string `json:"Id,omitempty"`
	Name    string `json:"Name,omitempty"`
	Created string `json:"Created,omitempty"`
	State   struct {
		Status  string `json:"Status,omitempty"`
		Running bool   `json:"Running,omitempty"`
	} `json:"State,omitempty"`
	Config struct {
		Image string `json:"Image,omitempty"`
	} `json:"Config,omitempty"`
	NetworkSettings struct {
		Ports map[string][]struct {
			HostIP   string `json:"HostIp,omitempty"`
			HostPort string `json:"HostPort,omitempty"`
		} `json:"Ports,omitempty"`
	} `json:"NetworkSettings,omitempty"`
}

func DescribeDocker(ctx context.Context, flags *flags.GlobalFlags, output io.Writer, l log.Logger, name string, configOnly bool, format string) error {
	containerName := getControlPlaneContainerName(name)

	// inspect the container
	details, err := inspectDockerContainerDetails(ctx, containerName)
	if err != nil {
		return fmt.Errorf("failed to find vcluster container %s: %w", containerName, err)
	}

	// get vcluster.yaml from the container if it's running
	var userConfigYaml *string
	if details.State.Running {
		configBytes, err := getVClusterConfigFromContainer(ctx, containerName)
		if err != nil {
			l.Debugf("Failed to get vcluster config: %v", err)
		} else if len(configBytes) > 0 {
			configStr := string(configBytes)
			userConfigYaml = &configStr
		}
	}

	// Return only the user supplied vcluster.yaml, if configOnly is set
	if configOnly {
		if userConfigYaml == nil {
			return fmt.Errorf("failed to load vcluster config (container may not be running)")
		}

		if _, err := output.Write([]byte(*userConfigYaml)); err != nil {
			return err
		}

		return nil
	}

	// parse created time
	created := metav1.Time{}
	if details.Created != "" {
		t, err := time.Parse(time.RFC3339, details.Created)
		if err == nil {
			created = metav1.Time{Time: t}
		}
	}

	// format ports
	ports := formatPorts(details)

	describeOutput := &DockerDescribeOutput{
		Name:           name,
		ContainerID:    details.ID,
		Status:         details.State.Status,
		Created:        created,
		Image:          details.Config.Image,
		Ports:          ports,
		UserConfigYaml: userConfigYaml,
	}

	return writeDockerDescribeWithFormat(output, format, describeOutput)
}

func inspectDockerContainerDetails(ctx context.Context, containerName string) (*dockerInspectDetails, error) {
	args := []string{"inspect", "--type", "container", containerName}
	out, err := exec.CommandContext(ctx, "docker", args...).Output()
	if err != nil {
		return nil, fmt.Errorf("docker inspect failed: %w", err)
	}

	var results []dockerInspectDetails
	err = json.Unmarshal(out, &results)
	if err != nil {
		return nil, fmt.Errorf("failed to parse docker inspect output: %w", err)
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("container %s not found", containerName)
	}

	return &results[0], nil
}

func getVClusterConfigFromContainer(ctx context.Context, containerName string) ([]byte, error) {
	// The vcluster.yaml is written to /etc/vcluster/vcluster.yaml in create_docker.go
	args := []string{"exec", containerName, "cat", "/etc/vcluster/vcluster.yaml"}
	out, err := exec.CommandContext(ctx, "docker", args...).Output()
	if err != nil {
		return nil, fmt.Errorf("failed to read vcluster config: %w", err)
	}
	return out, nil
}

func formatPorts(details *dockerInspectDetails) string {
	if details.NetworkSettings.Ports == nil {
		return ""
	}

	var ports []string
	for containerPort, hostPorts := range details.NetworkSettings.Ports {
		for _, hp := range hostPorts {
			if hp.HostPort != "" {
				ports = append(ports, fmt.Sprintf("%s:%s->%s", hp.HostIP, hp.HostPort, containerPort))
			}
		}
	}

	if len(ports) == 0 {
		return ""
	}

	result := ""
	for i, p := range ports {
		if i > 0 {
			result += ", "
		}
		result += p
	}
	return result
}

func writeDockerDescribeWithFormat(writer io.Writer, format string, o *DockerDescribeOutput) error {
	var outputBytes []byte
	var err error

	switch format {
	case "json":
		outputBytes, err = json.MarshalIndent(o, "", "  ")
	case "yaml":
		outputBytes, err = yaml.Marshal(o)
	case "":
		outputBytes = []byte(o.String())
	default:
		return fmt.Errorf("unknown format %s", format)
	}

	if err != nil {
		return err
	}

	_, err = writer.Write(outputBytes)
	return err
}
