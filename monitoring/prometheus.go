package monitoring

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type PrometheusMonitor struct {
	Counters   map[string]*prometheus.CounterVec
	Histograms map[string]*prometheus.HistogramVec
	Gauges     map[string]*prometheus.GaugeVec
	Mu         sync.RWMutex
	registerer prometheus.Registerer
}

func NewPrometheusMonitor() *PrometheusMonitor {
	return NewPrometheusMonitorWithRegisterer(prometheus.DefaultRegisterer)
}

// NewPrometheusMonitorWithRegisterer allows callers to supply an independent
// registry. This is used by tests that need to observe the metrics exported by
// a single monitor without interfering with the global default registerer.
func NewPrometheusMonitorWithRegisterer(registerer prometheus.Registerer) *PrometheusMonitor {
	return &PrometheusMonitor{
		Counters:   make(map[string]*prometheus.CounterVec),
		Histograms: make(map[string]*prometheus.HistogramVec),
		Gauges:     make(map[string]*prometheus.GaugeVec),
		registerer: registerer,
	}
}

func (p *PrometheusMonitor) IncCounter(name string, labels map[string]string, value int) {
	p.Mu.RLock()
	counter, ok := p.Counters[name]
	p.Mu.RUnlock()

	if !ok {
		p.Mu.Lock()
		// Re-read under the write lock: a concurrent caller may have created the
		// collector between our RUnlock and Lock. Assigning back into `counter`
		// ensures every caller returns with a non-nil vector.
		counter, ok = p.Counters[name]
		if !ok {
			counter = promauto.With(p.registerer).NewCounterVec(
				prometheus.CounterOpts{Name: name},
				getLabelNames(labels),
			)
			p.Counters[name] = counter
		}
		p.Mu.Unlock()
	}

	counter.With(labels).Add(float64(value))
}

func (p *PrometheusMonitor) RecordTiming(name string, labels map[string]string, duration time.Duration) {
	p.Mu.RLock()
	histogram, ok := p.Histograms[name]
	p.Mu.RUnlock()

	if !ok {
		p.Mu.Lock()
		histogram, ok = p.Histograms[name]
		if !ok {
			histogram = promauto.With(p.registerer).NewHistogramVec(
				prometheus.HistogramOpts{Name: name, Buckets: prometheus.DefBuckets},
				getLabelNames(labels),
			)
			p.Histograms[name] = histogram
		}
		p.Mu.Unlock()
	}

	histogram.With(labels).Observe(duration.Seconds())
}

// AddGauge changes a gauge value by delta. Additive updates let callers count
// actual concurrent activity (for example, two polling loops for the same
// app/partition on the same node) instead of overwriting the value with Set(1),
// which would hide duplicates.
func (p *PrometheusMonitor) AddGauge(name string, labels map[string]string, delta float64) {
	p.getGauge(name, labels).With(labels).Add(delta)
}

func (p *PrometheusMonitor) getGauge(name string, labels map[string]string) *prometheus.GaugeVec {
	p.Mu.RLock()
	gauge, ok := p.Gauges[name]
	p.Mu.RUnlock()

	if !ok {
		p.Mu.Lock()
		gauge, ok = p.Gauges[name]
		if !ok {
			gauge = promauto.With(p.registerer).NewGaugeVec(
				prometheus.GaugeOpts{Name: name},
				getLabelNames(labels),
			)
			p.Gauges[name] = gauge
		}
		p.Mu.Unlock()
	}

	return gauge
}

func getLabelNames(labels map[string]string) []string {
	var names []string
	for name := range labels {
		names = append(names, name)
	}
	return names
}
