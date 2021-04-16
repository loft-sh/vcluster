package metrics

import (
	"bytes"
	"context"
	"fmt"
	"github.com/loft-sh/vcluster/pkg/constants"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sort"
	"strings"
)

func Decode(data []byte) ([]*dto.MetricFamily, error) {
	var parser expfmt.TextParser
	metricFamilies, err := parser.TextToMetricFamilies(strings.NewReader(string(data)))
	if err != nil {
		return nil, fmt.Errorf("reading text format failed: %v", err)
	}

	// sort metrics alphabetically
	metricFamiliesArr := []*dto.MetricFamily{}
	for k, fam := range metricFamilies {
		name := k
		if fam.Name == nil {
			fam.Name = &name
		}

		metricFamiliesArr = append(metricFamiliesArr, fam)
	}
	sort.Slice(metricFamiliesArr, func(i int, j int) bool {
		return *metricFamiliesArr[i].Name < *metricFamiliesArr[j].Name
	})

	return metricFamiliesArr, nil
}

func Encode(metricsFamilies []*dto.MetricFamily, format expfmt.Format) ([]byte, error) {
	buffer := &bytes.Buffer{}
	encoder := expfmt.NewEncoder(buffer, format)
	for _, fam := range metricsFamilies {
		if len(fam.Metric) > 0 {
			err := encoder.Encode(fam)
			if err != nil {
				return nil, err
			}
		}
	}

	return buffer.Bytes(), nil
}

// Merge merges the metrics in a into b
func Merge(metricsFamiliesA []*dto.MetricFamily, metricsFamiliesB *[]*dto.MetricFamily) {
	for _, mA := range metricsFamiliesA {
		found := false
		for _, mB := range *metricsFamiliesB {
			if mB.Name != nil && mA.Name != nil && *mA.Name == *mB.Name {
				for _, m := range mA.Metric {
					mB.Metric = append(mB.Metric, m)
				}

				found = true
				break
			}
		}

		if !found {
			*metricsFamiliesB = append(*metricsFamiliesB, mA)
		}
	}
}

func AddLabels(metricsFamilies []*dto.MetricFamily, labels []*dto.LabelPair) {
	for _, fam := range metricsFamilies {
		for _, m := range fam.Metric {
			for _, newLabel := range labels {
				m.Label = append(m.Label, newLabel)
			}
		}
	}
}

func Rewrite(ctx context.Context, metricsFamilies []*dto.MetricFamily, targetNamespace string, vClient client.Client) ([]*dto.MetricFamily, error) {
	resultMetricsFamily := []*dto.MetricFamily{}

	// rewrite metrics
	for _, fam := range metricsFamilies {
		newMetrics := []*dto.Metric{}
		for _, m := range fam.Metric {
			var (
				pod       string
				namespace string
			)
			for _, l := range m.Label {
				if l.GetName() == "pod" {
					pod = l.GetValue()
				} else if l.GetName() == "namespace" {
					namespace = l.GetValue()
				}
			}

			// Add metrics that are pod and namespace independent
			if pod == "" && namespace == "" {
				newMetrics = append(newMetrics, m)
				continue
			}

			// skip the metric if it is not within the virtual cluster
			if namespace != targetNamespace {
				continue
			}

			// search if we can find the pod by name in the virtual cluster
			podList := &corev1.PodList{}
			err := vClient.List(ctx, podList, client.MatchingFields{constants.IndexByVName: pod})
			if err != nil {
				return nil, err
			}

			// skip the metric if the pod couldn't be found in the virtual cluster
			if len(podList.Items) == 0 {
				continue
			}

			pod = podList.Items[0].Name
			namespace = podList.Items[0].Namespace

			// exchange label values
			for _, l := range m.Label {
				if l.GetName() == "pod" {
					l.Value = &pod
				}
				if l.GetName() == "namespace" {
					l.Value = &namespace
				}
			}

			// add the rewritten metric
			newMetrics = append(newMetrics, m)
		}

		fam.Metric = newMetrics
		if len(fam.Metric) > 0 {
			resultMetricsFamily = append(resultMetricsFamily, fam)
		}
	}

	return resultMetricsFamily, nil
}
