package monitoring

import "time"

type Monitor interface {
	IncCounter(name string, labels map[string]string, value int)
	RecordTiming(name string, labels map[string]string, duration time.Duration)
	SetGauge(name string, labels map[string]string, value float64)
	DeleteGaugeLabels(name string, labels map[string]string)
}
