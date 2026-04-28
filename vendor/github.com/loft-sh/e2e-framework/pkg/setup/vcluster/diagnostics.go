package vcluster

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	. "github.com/onsi/ginkgo/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/ptr"
)

const syncerLogTailLines = 200

// DumpDiagnostics attaches vcluster debug info (config, pods, events,
// syncer logs) to the current spec report. Each section becomes its own
// report entry with FailureOrVerbose visibility. Uses the host K8s client,
// no kubectl required.
func DumpDiagnostics(ctx context.Context, hostClusterName, vclusterName, configFile string) {
	ns := "vcluster-" + vclusterName

	entry("vcluster-name", vclusterName)
	entry("namespace-name", ns)
	if configFile != "" {
		if data, err := os.ReadFile(configFile); err == nil {
			entry("vcluster-config", string(data))
		} else {
			entry("vcluster-config", "read failed: "+err.Error())
		}
	}

	host := cluster.From(ctx, hostClusterName)
	hostClient := cluster.KubeClientFrom(ctx, hostClusterName)
	if host == nil || hostClient == nil {
		return
	}
	entry("pods-in-namespace", dumpPods(ctx, hostClient, ns))
	entry("events-in-namespace", dumpEvents(ctx, hostClient, ns))
	entry("syncer-logs", dumpSyncerLogs(ctx, hostClient, ns))
}

func entry(name, body string) {
	AddReportEntry(name, ReportEntryVisibilityFailureOrVerbose, body)
}

func dumpPods(ctx context.Context, c kubernetes.Interface, ns string) string {
	pods, err := c.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return "list pods failed: " + err.Error()
	}
	var buf strings.Builder
	for _, p := range pods.Items {
		ready := 0
		for _, cs := range p.Status.ContainerStatuses {
			if cs.Ready {
				ready++
			}
		}
		fmt.Fprintf(&buf, "%s  %s  %d/%d  node=%s  age=%s\n",
			p.Name, p.Status.Phase, ready, len(p.Status.ContainerStatuses),
			p.Spec.NodeName, time.Since(p.CreationTimestamp.Time).Truncate(time.Second))
	}
	return buf.String()
}

func dumpEvents(ctx context.Context, c kubernetes.Interface, ns string) string {
	events, err := c.CoreV1().Events(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return "list events failed: " + err.Error()
	}
	sort.Slice(events.Items, func(i, j int) bool {
		return events.Items[i].LastTimestamp.Before(&events.Items[j].LastTimestamp)
	})
	var buf strings.Builder
	for _, e := range events.Items {
		fmt.Fprintf(&buf, "%s  %s  %s/%s  %s: %s\n",
			e.LastTimestamp.Format(time.RFC3339), e.Type,
			e.InvolvedObject.Kind, e.InvolvedObject.Name,
			e.Reason, e.Message)
	}
	return buf.String()
}

func dumpSyncerLogs(ctx context.Context, c kubernetes.Interface, ns string) string {
	pods, err := c.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{LabelSelector: "app=vcluster"})
	if err != nil {
		return "list vcluster pods failed: " + err.Error()
	}
	var buf strings.Builder
	for _, p := range pods.Items {
		fmt.Fprintf(&buf, "--- pod/%s ---\n", p.Name)
		stream, err := c.CoreV1().Pods(ns).GetLogs(p.Name, &corev1.PodLogOptions{
			Container: "syncer",
			TailLines: ptr.To(int64(syncerLogTailLines)),
		}).Stream(ctx)
		if err != nil {
			fmt.Fprintf(&buf, "stream logs failed: %v\n", err)
			continue
		}
		copyLines(&buf, stream)
		stream.Close() //nolint:errcheck // diagnostic path
	}
	return buf.String()
}

func copyLines(w io.Writer, r io.Reader) {
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		fmt.Fprintln(w, sc.Text())
	}
}
