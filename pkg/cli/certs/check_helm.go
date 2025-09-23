package certs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/blang/semver"
	"github.com/loft-sh/log"
	"github.com/loft-sh/log/table"
	"github.com/loft-sh/vcluster/pkg/certs"
	"github.com/loft-sh/vcluster/pkg/cli/find"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/util/podhelper"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
)

// Check executes the check command in the backend by piping its output to stdout.
func Check(ctx context.Context, vClusterName string, globalFlags *flags.GlobalFlags, output string, log log.Logger) error {
	vCluster, err := find.GetVCluster(ctx, globalFlags.Context, vClusterName, globalFlags.Namespace, log)
	if err != nil {
		return err
	}

	// check if check command is supported
	version, err := semver.Parse(strings.TrimPrefix(vCluster.Version, "v"))
	if err == nil {
		// only check if version matches if vCluster actually has a parsable version
		if version.LT(semver.MustParse(minVersion)) {
			return fmt.Errorf("cert check is not supported in vCluster version %s", vCluster.Version)
		}
	}

	// abort in case the virtual cluster has a non-running status.
	if vCluster.Status != find.StatusRunning {
		return fmt.Errorf("aborting operation because virtual cluster %q has status %q", vCluster.Name, vCluster.Status)
	}

	var targetPod *corev1.Pod
	for _, pod := range vCluster.Pods {
		if vCluster.StatefulSet != nil && strings.HasSuffix(pod.Name, "-0") {
			targetPod = &pod
			break
		} else if vCluster.Deployment != nil {
			targetPod = &pod
			break
		}
	}
	if targetPod == nil {
		return fmt.Errorf("couldn't find a running pod for vCluster %s", vCluster.Name)
	}

	kubeConfig, err := vCluster.ClientFactory.ClientConfig()
	if err != nil {
		return err
	}

	reader, writer := io.Pipe()
	go func() {
		defer func() {
			if err := writer.Close(); err != nil {
				fmt.Printf("closing writer: %v\n", err)
			}
		}()

		err := podhelper.ExecStream(ctx, kubeConfig, &podhelper.ExecStreamOptions{
			Pod:       targetPod.Name,
			Namespace: vCluster.Namespace,
			Container: "syncer",
			Command:   []string{"sh", "-c", "/vcluster certs check"},
			Stdout:    writer,
			Stderr:    os.Stdout,
		})
		if err != nil {
			fmt.Printf("executing %q in syncer pod: %v\n", "certs check", err)
			return
		}
	}()

	var certificateInfos []certs.Info
	decoder := json.NewDecoder(reader)
	for {
		if err := decoder.Decode(&certificateInfos); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("decoding: %w", err)
		}
	}

	if output == "json" {
		bytes, err := json.MarshalIndent(certificateInfos, "", "    ")
		if err != nil {
			return fmt.Errorf("json marshal vClusters: %w", err)
		}

		log.WriteString(logrus.InfoLevel, string(bytes)+"\n")
	} else {
		header := []string{"FILENAME", "SUBJECT", "ISSUER", "EXPIRES ON", "STATUS"}
		var values [][]string
		for _, certInfo := range certificateInfos {
			values = append(values, []string{certInfo.Filename, certInfo.Subject, certInfo.Issuer, certInfo.ExpiryTime.Format("Jan 02, 2006 15:04 MST"), certInfo.Status})
		}
		table.PrintTable(log, header, values)
	}

	return nil
}
