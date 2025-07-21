package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

type PrometheusCounter struct {
	*prometheus.CounterVec
}

func NewPrometheusCounter(registry prometheus.Registerer, opts prometheus.CounterOpts, labelNames []string) CounterMetric {
	c := prometheus.NewCounterVec(opts, labelNames)
	registry.MustRegister(c)
	return &PrometheusCounter{CounterVec: c}
}

func (pc *PrometheusCounter) Inc(labels map[string]string) {
	pc.CounterVec.With(labels).Inc()
}

func (pc *PrometheusCounter) Add(v float64, labels map[string]string) {
	pc.CounterVec.With(labels).Add(v)
}

func (pc *PrometheusCounter) Delete(labels map[string]string) {
	pc.CounterVec.Delete(labels)
}

func (pc *PrometheusCounter) DeletePartialMatch(labels map[string]string) {
	pc.CounterVec.DeletePartialMatch(labels)
}

func (pc *PrometheusCounter) Reset() {
	pc.CounterVec.Reset()
}

type PrometheusGauge struct {
	*prometheus.GaugeVec
}

func NewPrometheusGauge(registry prometheus.Registerer, opts prometheus.GaugeOpts, labelNames []string) GaugeMetric {
	g := prometheus.NewGaugeVec(opts, labelNames)
	registry.MustRegister(g)
	return &PrometheusGauge{GaugeVec: g}
}

func (pg *PrometheusGauge) Set(v float64, labels map[string]string) {
	pg.GaugeVec.With(labels).Set(v)
}

func (pg *PrometheusGauge) Delete(labels map[string]string) {
	pg.GaugeVec.Delete(labels)
}

func (pg *PrometheusGauge) DeletePartialMatch(labels map[string]string) {
	pg.GaugeVec.DeletePartialMatch(labels)
}

func (pg *PrometheusGauge) Reset() {
	pg.GaugeVec.Reset()
}

type PrometheusHistogram struct {
	*prometheus.HistogramVec
}

func NewPrometheusHistogram(registry prometheus.Registerer, opts prometheus.HistogramOpts, labelNames []string) ObservationMetric {
	h := prometheus.NewHistogramVec(opts, labelNames)
	registry.MustRegister(h)
	return &PrometheusHistogram{HistogramVec: h}
}

func (ph *PrometheusHistogram) Observe(v float64, labels map[string]string) {
	ph.HistogramVec.With(labels).Observe(v)
}

func (ph *PrometheusHistogram) Delete(labels map[string]string) {
	ph.HistogramVec.Delete(labels)
}

func (ph *PrometheusHistogram) DeletePartialMatch(labels map[string]string) {
	ph.HistogramVec.DeletePartialMatch(labels)
}

func (ph *PrometheusHistogram) Reset() {
	ph.HistogramVec.Reset()
}

type PrometheusSummary struct {
	*prometheus.SummaryVec
}

func NewPrometheusSummary(registry prometheus.Registerer, opts prometheus.SummaryOpts, labelNames []string) ObservationMetric {
	s := prometheus.NewSummaryVec(opts, labelNames)
	registry.MustRegister(s)
	return &PrometheusSummary{SummaryVec: s}
}

func (ps *PrometheusSummary) Observe(v float64, labels map[string]string) {
	ps.SummaryVec.With(labels).Observe(v)
}

func (ps *PrometheusSummary) Delete(labels map[string]string) {
	ps.SummaryVec.Delete(labels)
}

func (ps *PrometheusSummary) DeletePartialMatch(labels map[string]string) {
	ps.SummaryVec.DeletePartialMatch(labels)
}

func (ps *PrometheusSummary) Reset() {
	ps.SummaryVec.Reset()
}
