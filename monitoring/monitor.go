package monitoring

import "time"

type Monitor interface {
	IncCounter(name string, labels map[string]string, value int)
	RecordTiming(name string, labels map[string]string, duration time.Duration)
}

// GaugeMonitor is an optional extension that lets a Monitor implementation
// publish additive gauge activity. Existing Monitor implementations do not
// need to implement this to remain compatible with the scheduler; components
// that publish gauges check for the extension with a type assertion.
type GaugeMonitor interface {
	AddGauge(name string, labels map[string]string, delta float64)
}
