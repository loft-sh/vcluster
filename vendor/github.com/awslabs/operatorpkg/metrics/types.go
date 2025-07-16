package metrics

const (
	Namespace   = "operator"
	LabelGroup  = "group"
	LabelKind   = "kind"
	LabelType   = "type"
	LabelReason = "reason"
)

type ObservationMetric interface {
	Observe(v float64, labels map[string]string)
	Delete(labels map[string]string)
	DeletePartialMatch(labels map[string]string)
	Reset()
}

type CounterMetric interface {
	Add(v float64, labels map[string]string)
	Inc(labels map[string]string)
	Delete(labels map[string]string)
	DeletePartialMatch(labels map[string]string)
	Reset()
}

type GaugeMetric interface {
	Set(v float64, labels map[string]string)
	Delete(labels map[string]string)
	DeletePartialMatch(labels map[string]string)
	Reset()
}
