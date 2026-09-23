package poller

import (
	"testing"
	"time"

	"github.com/myntra/goscheduler/conf"
	"github.com/myntra/goscheduler/constants"
	"github.com/myntra/goscheduler/monitoring"
	"github.com/prometheus/client_golang/prometheus"
)

// newTestPoller returns a Poller wired to an isolated Prometheus registry so
// its exported samples do not leak into other tests. Interval is set high
// enough that no automatic tick will fire during the test.
func newTestPoller(t *testing.T) (*Poller, *prometheus.Registry) {
	t.Helper()
	registry := prometheus.NewRegistry()
	p := &Poller{
		AppName:     "example",
		PartitionId: 0,
		config:      conf.PollerConfig{Interval: 3600},
		monitor:     monitoring.NewPrometheusMonitorWithRegisterer(registry),
	}
	p.SetNodeAddress("vm-a")
	if err := p.Init(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(p.Stop)
	return p, registry
}

// metricValue reads the current value for the sole label combination we expect
// the test poller to publish. It returns (value, present) so a caller can
// distinguish "never recorded" from "recorded and zero".
func metricValue(t *testing.T, registry *prometheus.Registry, name string) (float64, bool) {
	t.Helper()
	families, err := registry.Gather()
	if err != nil {
		t.Fatal(err)
	}
	for _, family := range families {
		if family.GetName() == name {
			if len(family.Metric) != 1 {
				t.Fatalf("%s: expected exactly one label set, got %d", name, len(family.Metric))
			}
			labels := map[string]string{}
			for _, label := range family.Metric[0].Label {
				labels[label.GetName()] = label.GetValue()
			}
			if labels["app_id"] != "example" || labels["partition_id"] != "0" || labels["node"] != "vm-a" {
				t.Fatalf("unexpected labels: %v", labels)
			}
			return family.Metric[0].GetGauge().GetValue(), true
		}
	}
	return 0, false
}

// awaitValue polls the registry until it sees the wanted value or times out.
// The polling loop is necessary because Start increments the gauge in a
// goroutine and the test cannot synchronize on that entry point directly.
func awaitValue(t *testing.T, registry *prometheus.Registry, name string, want float64) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if got, found := metricValue(t, registry, name); found && got == want {
			return
		}
		time.Sleep(time.Millisecond)
	}
	got, found := metricValue(t, registry, name)
	t.Fatalf("%s = %v (present %v), want %v", name, got, found, want)
}

func startLoop(p *Poller) <-chan struct{} {
	done := make(chan struct{})
	go func() { defer close(done); p.Start() }()
	return done
}

func awaitExit(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("poller loop did not exit")
	}
}

// TestGaugeCountsActualLoops asserts that the gauge reflects the number of
// running Start goroutines. Two overlapping loops on the same node show as 2,
// not 1, so a duplicate-local-start is not hidden. A repeated Stop is safe.
func TestGaugeCountsActualLoops(t *testing.T) {
	p, registry := newTestPoller(t)
	if _, found := metricValue(t, registry, constants.PollerDistribution); found {
		t.Fatal("Init reported a running loop before Start")
	}
	first := startLoop(p)
	awaitValue(t, registry, constants.PollerDistribution, 1)
	second := startLoop(p)
	awaitValue(t, registry, constants.PollerDistribution, 2)
	p.Stop()
	p.Stop() // repeated Stop is safe and cannot double-decrement
	awaitExit(t, first)
	awaitExit(t, second)
	awaitValue(t, registry, constants.PollerDistribution, 0)
}

// TestStopBeforeStartDoesNotReportRunning asserts that when Stop precedes
// Start, the loop exits without ever incrementing the gauge, so the process
// never falsely publishes a running series for that generation.
func TestStopBeforeStartDoesNotReportRunning(t *testing.T) {
	p, registry := newTestPoller(t)
	p.Stop()
	awaitExit(t, startLoop(p))
	if _, found := metricValue(t, registry, constants.PollerDistribution); found {
		t.Fatal("a canceled run reported activity")
	}
}

// TestReinitializationExitsPreviousLoop asserts that Init cancels the previous
// run so a long-running loop does not remain blocked on a stopped ticker.
// Without an explicit stop channel a re-Init would leak the previous goroutine.
func TestReinitializationExitsPreviousLoop(t *testing.T) {
	p, registry := newTestPoller(t)
	previous := startLoop(p)
	awaitValue(t, registry, constants.PollerDistribution, 1)
	if err := p.Init(); err != nil {
		t.Fatal(err)
	}
	awaitExit(t, previous)
	awaitValue(t, registry, constants.PollerDistribution, 0)
	next := startLoop(p)
	awaitValue(t, registry, constants.PollerDistribution, 1)
	p.Stop()
	awaitExit(t, next)
	awaitValue(t, registry, constants.PollerDistribution, 0)
}

// TestInvalidInitializationDoesNotReportRunning asserts that Init rejects a
// non-positive interval and Start on an uninitialized poller does not publish
// any gauge activity.
func TestInvalidInitializationDoesNotReportRunning(t *testing.T) {
	registry := prometheus.NewRegistry()
	p := &Poller{monitor: monitoring.NewPrometheusMonitorWithRegisterer(registry)}
	if err := p.Init(); err == nil {
		t.Fatal("expected invalid interval error")
	}
	awaitExit(t, startLoop(p))
	if _, found := metricValue(t, registry, constants.PollerDistribution); found {
		t.Fatal("uninitialized poller reported activity")
	}
}

// counterOnlyMonitor is a Monitor without gauge support, used to prove that a
// legacy implementation still drives the polling loop even though gauge
// activity is silently dropped.
type counterOnlyMonitor struct{ started chan struct{} }

func (m counterOnlyMonitor) IncCounter(_ string, labels map[string]string, _ int) {
	if labels["lifeCycleMethod"] == constants.Start {
		select {
		case <-m.started:
		default:
			close(m.started)
		}
	}
}

func (counterOnlyMonitor) RecordTiming(string, map[string]string, time.Duration) {}

// TestPollerWithoutGaugeMonitor asserts backward compatibility: a Monitor
// that does not implement GaugeMonitor still lets Start run to completion.
func TestPollerWithoutGaugeMonitor(t *testing.T) {
	p, _ := newTestPoller(t)
	monitor := counterOnlyMonitor{started: make(chan struct{})}
	p.monitor = monitor
	done := startLoop(p)
	select {
	case <-monitor.started:
	case <-time.After(2 * time.Second):
		t.Fatal("Start counter was never recorded")
	}
	p.Stop()
	awaitExit(t, done)
}

// panicOnStartMonitor injects a panic from the very first IncCounter call so
// the deferred gauge cleanup in Start is exercised. Without the deferred
// decrement, a panic during setup would leak a permanent +1 in the gauge.
type panicOnStartMonitor struct{ *monitoring.PrometheusMonitor }

func (panicOnStartMonitor) IncCounter(string, map[string]string, int) {
	panic("test loop failure")
}

// TestLoopPanicDecrementsGauge asserts that a panic during the polling loop
// still runs the deferred AddGauge(-1), so the exported value returns to zero.
func TestLoopPanicDecrementsGauge(t *testing.T) {
	p, registry := newTestPoller(t)
	p.monitor = panicOnStartMonitor{p.monitor.(*monitoring.PrometheusMonitor)}
	func() {
		defer func() {
			if recover() == nil {
				t.Error("expected injected loop panic")
			}
		}()
		p.Start()
	}()
	awaitValue(t, registry, constants.PollerDistribution, 0)
	// Cleanup must not deliberately inject another panic from Stop's counter.
	p.monitor = nil
}
