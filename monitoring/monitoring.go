package monitoring

import (
	"github.com/golang/glog"
	"github.com/myntra/goscheduler/conf"
	newrelic "github.com/newrelic/go-agent"
	"gopkg.in/alexcesaro/statsd.v2"
)

type Monitoring struct {
	StatsDClient *statsd.Client
	NewrelicApp  *newrelic.Application
}

func NewMonitoring(monitoring *conf.MonitoringConfig) *Monitoring {
	statsdClient, err := initStatsd(monitoring.Statsd)
	if err != nil {
		glog.Fatalf("Error while creating statsd client %+v", err)
		return nil
	}
	return &Monitoring{
		StatsDClient: statsdClient,
	}
}
