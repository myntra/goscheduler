package poller

import (
	"sync"
	"testing"
	"time"

	"github.com/myntra/goscheduler/conf"
	"github.com/myntra/goscheduler/constants"
)

// gaugeRecordingMonitor records poller_distribution deltas for tests without
// pulling Prometheus registry wiring into the poller package.
type gaugeRecordingMonitor struct {
	mu                 sync.Mutex
	pollerDistribution float64
}

func (m *gaugeRecordingMonitor) AddGauge(name string, _ map[string]string, delta float64) {
	if name != constants.PollerDistribution {
		return
	}
	m.mu.Lock()
	m.pollerDistribution += delta
	m.mu.Unlock()
}

func (m *gaugeRecordingMonitor) IncCounter(string, map[string]string, int)             {}
func (m *gaugeRecordingMonitor) RecordTiming(string, map[string]string, time.Duration) {}

func (m *gaugeRecordingMonitor) pollerDistributionValue() float64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.pollerDistribution
}

func newTestPoller(t *testing.T) (*Poller, *gaugeRecordingMonitor) {
	t.Helper()
	monitor := &gaugeRecordingMonitor{}
	p := &Poller{
		AppName:     "example",
		PartitionId: 0,
		config:      conf.PollerConfig{Interval: 3600},
		monitor:     monitor,
	}
	p.SetNodeAddress("vm-a")
	if err := p.Init(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(p.Stop)
	return p, monitor
}

func awaitDistributionValue(t *testing.T, monitor *gaugeRecordingMonitor, want float64) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if got := monitor.pollerDistributionValue(); got == want {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("poller_distribution = %v, want %v", monitor.pollerDistributionValue(), want)
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

func TestGaugeCountsActualLoops(t *testing.T) {
	p, monitor := newTestPoller(t)
	if monitor.pollerDistributionValue() != 0 {
		t.Fatal("Init reported a running loop before Start")
	}
	first := startLoop(p)
	awaitDistributionValue(t, monitor, 1)
	second := startLoop(p)
	awaitDistributionValue(t, monitor, 2)
	p.Stop()
	p.Stop()
	awaitExit(t, first)
	awaitExit(t, second)
	awaitDistributionValue(t, monitor, 0)
}

func TestStopBeforeStartDoesNotReportRunning(t *testing.T) {
	p, monitor := newTestPoller(t)
	p.Stop()
	awaitExit(t, startLoop(p))
	if monitor.pollerDistributionValue() != 0 {
		t.Fatal("a canceled run reported activity")
	}
}

func TestReinitializationExitsPreviousLoop(t *testing.T) {
	p, monitor := newTestPoller(t)
	previous := startLoop(p)
	awaitDistributionValue(t, monitor, 1)
	if err := p.Init(); err != nil {
		t.Fatal(err)
	}
	awaitExit(t, previous)
	awaitDistributionValue(t, monitor, 0)
	next := startLoop(p)
	awaitDistributionValue(t, monitor, 1)
	p.Stop()
	awaitExit(t, next)
	awaitDistributionValue(t, monitor, 0)
}

func TestInvalidInitializationDoesNotReportRunning(t *testing.T) {
	monitor := &gaugeRecordingMonitor{}
	p := &Poller{monitor: monitor}
	if err := p.Init(); err == nil {
		t.Fatal("expected invalid interval error")
	}
	awaitExit(t, startLoop(p))
	if monitor.pollerDistributionValue() != 0 {
		t.Fatal("uninitialized poller reported activity")
	}
}

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

type panicOnStartMonitor struct {
	*gaugeRecordingMonitor
}

func (panicOnStartMonitor) IncCounter(string, map[string]string, int) {
	panic("test loop failure")
}

func TestLoopPanicDecrementsGauge(t *testing.T) {
	p, monitor := newTestPoller(t)
	p.monitor = &panicOnStartMonitor{gaugeRecordingMonitor: monitor}
	func() {
		defer func() {
			if recover() == nil {
				t.Error("expected injected loop panic")
			}
		}()
		p.Start()
	}()
	awaitDistributionValue(t, monitor, 0)
	p.monitor = nil
}
