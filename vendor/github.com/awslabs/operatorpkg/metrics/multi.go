package metrics

type MultiCounter struct {
	counters []CounterMetric
}

func NewMultiCounter(counters ...CounterMetric) CounterMetric {
	return &MultiCounter{counters: counters}
}

func (mc *MultiCounter) Inc(labels map[string]string) {
	for _, c := range mc.counters {
		c.Inc(labels)
	}
}

func (mc *MultiCounter) Add(v float64, labels map[string]string) {
	for _, c := range mc.counters {
		c.Add(v, labels)
	}
}

func (mc *MultiCounter) Delete(labels map[string]string) {
	for _, c := range mc.counters {
		c.Delete(labels)
	}
}

func (mc *MultiCounter) DeletePartialMatch(labels map[string]string) {
	for _, c := range mc.counters {
		c.DeletePartialMatch(labels)
	}
}

func (mc *MultiCounter) Reset() {
	for _, c := range mc.counters {
		c.Reset()
	}
}

type MultiGauge struct {
	gauges []GaugeMetric
}

func NewMultiGauge(gauges ...GaugeMetric) GaugeMetric {
	return &MultiGauge{gauges: gauges}
}

func (mg *MultiGauge) Set(v float64, labels map[string]string) {
	for _, g := range mg.gauges {
		g.Set(v, labels)
	}
}

func (mg *MultiGauge) Delete(labels map[string]string) {
	for _, g := range mg.gauges {
		g.Delete(labels)
	}
}

func (mg *MultiGauge) DeletePartialMatch(labels map[string]string) {
	for _, g := range mg.gauges {
		g.DeletePartialMatch(labels)
	}
}

func (mg *MultiGauge) Reset() {
	for _, g := range mg.gauges {
		g.Reset()
	}
}

type MultiObservation struct {
	observations []ObservationMetric
}

func NewMultiObservation(observations ...ObservationMetric) ObservationMetric {
	return &MultiObservation{observations: observations}
}

func (mo *MultiObservation) Observe(v float64, labels map[string]string) {
	for _, o := range mo.observations {
		o.Observe(v, labels)
	}
}

func (mo *MultiObservation) Delete(labels map[string]string) {
	for _, o := range mo.observations {
		o.Delete(labels)
	}
}

func (mo *MultiObservation) DeletePartialMatch(labels map[string]string) {
	for _, o := range mo.observations {
		o.DeletePartialMatch(labels)
	}
}

func (mo *MultiObservation) Reset() {
	for _, o := range mo.observations {
		o.Reset()
	}
}
