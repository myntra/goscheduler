package poller

import (
	"github.com/golang/glog"
	"github.com/myntra/goscheduler/conf"
	"github.com/myntra/goscheduler/constants"
	p "github.com/myntra/goscheduler/monitoring"
	r "github.com/myntra/goscheduler/retrieveriface"
	"strconv"
	"time"
)

type Poller struct {
	AppName               string
	PartitionId           int
	scheduleRetrievalImpl r.Retriever
	ticker                *time.Ticker
	config                conf.PollerConfig
	monitoring            *p.Monitoring
}

func (p *Poller) recordPollerLifeCycle(poller *Poller, lifeCycleMethod string) {
	if p.monitoring != nil && p.monitoring.StatsDClient != nil {
		bucket := lifeCycleMethod + constants.DOT + poller.AppName + constants.DOT + strconv.Itoa(poller.PartitionId)
		p.monitoring.StatsDClient.Increment(bucket)
	}
}

func (p *Poller) Init() error {
	if p.ticker != nil {
		p.ticker.Stop()
	}
	p.ticker = time.NewTicker(time.Duration(p.config.Interval) * time.Second)

	return nil
}

func (p *Poller) Start() {
	p.recordPollerLifeCycle(p, constants.Start)
	for currentTime := range p.ticker.C {
		p.recordPollerLifeCycle(p, constants.Running)
		timeBucket := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), currentTime.Hour(), currentTime.Minute(), 0, 0, currentTime.Location())
		go p.scheduleRetrievalImpl.GetSchedules(p.AppName, p.PartitionId, timeBucket)
	}
}

func (p *Poller) Stop() {
	p.recordPollerLifeCycle(p, constants.Stop)
	glog.Infof("Stopping poller for %s.%d", p.AppName, p.PartitionId)
	p.ticker.Stop()
}
