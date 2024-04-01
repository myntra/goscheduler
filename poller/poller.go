// Copyright (c) 2023 Myntra Designs Private Limited.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

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
	monitor               p.Monitor
}

func (p *Poller) recordPollerLifeCycle(lifeCycleMethod string) {
	if p.monitor != nil {
		p.monitor.IncCounter(constants.PollerLifeCycle, map[string]string{"lifeCycleMethod": lifeCycleMethod, "appId": p.AppName, "partitionId": strconv.Itoa(p.PartitionId)}, 1)
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
	p.recordPollerLifeCycle(constants.Start)
	for currentTime := range p.ticker.C {
		p.recordPollerLifeCycle(constants.Running)
		timeBucket := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), currentTime.Hour(), currentTime.Minute(), 0, 0, currentTime.Location())
		go p.scheduleRetrievalImpl.GetSchedules(p.AppName, p.PartitionId, timeBucket)
	}
}

func (p *Poller) Stop() {
	p.recordPollerLifeCycle(constants.Stop)
	glog.Infof("Stopping poller for %s.%d", p.AppName, p.PartitionId)
	p.ticker.Stop()
}
