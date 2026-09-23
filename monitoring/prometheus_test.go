package monitoring

import (
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func gatheredMetric(t *testing.T, registry *prometheus.Registry, name string) *dto.Metric {
	t.Helper()
	families, err := registry.Gather()
	if err != nil {
		t.Fatal(err)
	}
	for _, family := range families {
		if family.GetName() == name && len(family.Metric) == 1 {
			return family.Metric[0]
		}
	}
	t.Fatalf("expected exactly one metric for %s", name)
	return nil
}

// TestConcurrentCollectorInitialization asserts that the double-checked lock
// used to lazily register counter/histogram/gauge collectors is safe under
// simultaneous first use. Before the fix, a caller that entered the write-lock
// branch second could receive a nil vector and panic on With(labels).
func TestConcurrentCollectorInitialization(t *testing.T) {
	const callers = 64
	tests := []struct {
		name  string
		write func(*PrometheusMonitor, map[string]string)
		value func(*dto.Metric) float64
		want  float64
	}{
		{
			name:  "counter",
			write: func(p *PrometheusMonitor, labels map[string]string) { p.IncCounter("concurrent_counter", labels, 1) },
			value: func(m *dto.Metric) float64 { return m.GetCounter().GetValue() },
			want:  callers,
		},
		{
			name: "histogram",
			write: func(p *PrometheusMonitor, labels map[string]string) {
				p.RecordTiming("concurrent_histogram", labels, time.Second)
			},
			value: func(m *dto.Metric) float64 { return float64(m.GetHistogram().GetSampleCount()) },
			want:  callers,
		},
		{
			name:  "add_gauge",
			write: func(p *PrometheusMonitor, labels map[string]string) { p.AddGauge("concurrent_add_gauge", labels, 1) },
			value: func(m *dto.Metric) float64 { return m.GetGauge().GetValue() },
			want:  callers,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for round := 0; round < 25; round++ {
				registry := prometheus.NewRegistry()
				monitor := NewPrometheusMonitorWithRegisterer(registry)
				labels := map[string]string{"app_id": "example", "partition_id": "0", "node": "vm-a"}
				start := make(chan struct{})
				var wg sync.WaitGroup
				for i := 0; i < callers; i++ {
					wg.Add(1)
					go func() {
						defer wg.Done()
						<-start
						test.write(monitor, labels)
					}()
				}
				close(start)
				wg.Wait()
				if got := test.value(gatheredMetric(t, registry, "concurrent_"+test.name)); got != test.want {
					t.Fatalf("round %d: got %v, want %v", round, got, test.want)
				}
			}
		})
	}
}

// TestSeparateProcessesKeepTheirOwnGauge documents that each PrometheusMonitor
// exports only its own local samples. A process cannot revoke another process's
// exported series by touching its own registry.
func TestSeparateProcessesKeepTheirOwnGauge(t *testing.T) {
	registryA, registryB := prometheus.NewRegistry(), prometheus.NewRegistry()
	a := NewPrometheusMonitorWithRegisterer(registryA)
	b := NewPrometheusMonitorWithRegisterer(registryB)
	labelsA := map[string]string{"app_id": "example", "partition_id": "0", "node": "vm-a"}
	labelsB := map[string]string{"app_id": "example", "partition_id": "0", "node": "vm-b"}
	a.AddGauge("poller_distribution", labelsA, 1)
	b.AddGauge("poller_distribution", labelsB, 1)
	if got := gatheredMetric(t, registryA, "poller_distribution").GetGauge().GetValue(); got != 1 {
		t.Fatalf("node A value = %v, want 1", got)
	}
	if got := gatheredMetric(t, registryB, "poller_distribution").GetGauge().GetValue(); got != 1 {
		t.Fatalf("node B value = %v, want 1", got)
	}
	b.AddGauge("poller_distribution", labelsB, -1)
	if got := gatheredMetric(t, registryB, "poller_distribution").GetGauge().GetValue(); got != 0 {
		t.Fatalf("stopped node B value = %v, want 0", got)
	}
}

// legacyMonitor verifies that a Monitor implementation without gauge support
// still satisfies the interface. The scheduler treats gauges as an optional
// GaugeMonitor extension, not a required capability.
type legacyMonitor struct{}

func (legacyMonitor) IncCounter(string, map[string]string, int)             {}
func (legacyMonitor) RecordTiming(string, map[string]string, time.Duration) {}

var _ Monitor = legacyMonitor{}
