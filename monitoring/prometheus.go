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
}

func NewPrometheusMonitor() *PrometheusMonitor {
	return &PrometheusMonitor{
		Counters:   make(map[string]*prometheus.CounterVec),
		Histograms: make(map[string]*prometheus.HistogramVec),
		Gauges:     make(map[string]*prometheus.GaugeVec),
	}
}

func (p *PrometheusMonitor) IncCounter(name string, labels map[string]string, value int) {
	p.Mu.RLock()
	counter, ok := p.Counters[name]
	p.Mu.RUnlock()

	if !ok {
		p.Mu.Lock()
		if _, ok := p.Counters[name]; !ok {
			counter = promauto.NewCounterVec(
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
		if _, ok := p.Histograms[name]; !ok {
			histogram = promauto.NewHistogramVec(
				prometheus.HistogramOpts{Name: name, Buckets: prometheus.DefBuckets},
				getLabelNames(labels),
			)
			p.Histograms[name] = histogram
		}
		p.Mu.Unlock()
	}

	histogram.With(labels).Observe(duration.Seconds())
}

// SetGauge sets a gauge value with the given name, labels, and value
func (p *PrometheusMonitor) SetGauge(name string, labels map[string]string, value float64) {
	p.Mu.RLock()
	gauge, ok := p.Gauges[name]
	p.Mu.RUnlock()

	if !ok {
		p.Mu.Lock()
		if _, ok := p.Gauges[name]; !ok {
			gauge = promauto.NewGaugeVec(
				prometheus.GaugeOpts{Name: name},
				getLabelNames(labels),
			)
			p.Gauges[name] = gauge
		}
		p.Mu.Unlock()
	}

	gauge.With(labels).Set(value)
}

// DeleteGaugeLabels removes a specific label combination from a gauge
func (p *PrometheusMonitor) DeleteGaugeLabels(name string, labels map[string]string) {
	p.Mu.RLock()
	gauge, ok := p.Gauges[name]
	p.Mu.RUnlock()

	if ok {
		gauge.Delete(labels)
	}
}

func getLabelNames(labels map[string]string) []string {
	var names []string
	for name := range labels {
		names = append(names, name)
	}
	return names
}
