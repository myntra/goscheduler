package monitoring

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"sync"
	"time"
)

type PrometheusMonitor struct {
	Counters   map[string]*prometheus.CounterVec
	Histograms map[string]*prometheus.HistogramVec
	Mu         sync.RWMutex
}

func NewPrometheusMonitor() *PrometheusMonitor {
	return &PrometheusMonitor{
		Counters:   make(map[string]*prometheus.CounterVec),
		Histograms: make(map[string]*prometheus.HistogramVec),
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

func getLabelNames(labels map[string]string) []string {
	var names []string
	for name := range labels {
		names = append(names, name)
	}
	return names
}
