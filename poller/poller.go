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
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/myntra/goscheduler/conf"
	"github.com/myntra/goscheduler/constants"
	"github.com/myntra/goscheduler/monitoring"
	r "github.com/myntra/goscheduler/retrieveriface"
)

type Poller struct {
	AppName               string
	PartitionId           int
	scheduleRetrievalImpl r.Retriever
	config                conf.PollerConfig
	monitor               monitoring.Monitor
	mu                    sync.Mutex
	node                  string
	run                   *pollerRun
}

// pollerRun holds the state of a single polling generation: its ticker and its
// stop signal. Each Init creates a fresh run; reinitialization signals the
// previous run to exit rather than leaving it blocked on a stopped ticker,
// because time.Ticker.Stop does not close the underlying channel.
type pollerRun struct {
	ticker   *time.Ticker
	stop     chan struct{}
	stopOnce sync.Once
}

func (r *pollerRun) cancel() {
	r.stopOnce.Do(func() {
		r.ticker.Stop()
		close(r.stop)
	})
}

// SetNodeAddress records the node label used when publishing this poller's
// gauge activity. The supervisor calls this before Start; callers running a
// Poller directly should call it themselves.
func (p *Poller) SetNodeAddress(address string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.node = address
}

func (p *Poller) recordPollerLifeCycle(lifeCycleMethod string) {
	if p.monitor != nil {
		p.monitor.IncCounter(constants.PollerLifeCycle, map[string]string{"lifeCycleMethod": lifeCycleMethod, "appId": p.AppName, "partitionId": strconv.Itoa(p.PartitionId)}, 1)
	}
}

func (p *Poller) Init() error {
	if p.config.Interval <= 0 {
		return fmt.Errorf("poller interval must be positive, got %d", p.config.Interval)
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.run != nil {
		p.run.cancel()
	}
	p.run = &pollerRun{
		ticker: time.NewTicker(time.Duration(p.config.Interval) * time.Second),
		stop:   make(chan struct{}),
	}
	return nil
}

func (p *Poller) Start() {
	p.mu.Lock()
	run := p.run
	labels := map[string]string{
		"app_id":       p.AppName,
		"partition_id": strconv.Itoa(p.PartitionId),
		"node":         p.node,
	}
	p.mu.Unlock()
	if run == nil {
		return
	}
	// Honor a Stop that was requested before Start had a chance to run.
	select {
	case <-run.stop:
		return
	default:
	}

	// Measure real loop entry/exit, not the caller's start/stop requests.
	// Additive updates keep duplicate loops on the same node visible instead of
	// being hidden by Set(1). The deferred decrement runs even on panic.
	addGauge := func(delta float64) {
		if monitor, ok := p.monitor.(monitoring.GaugeMonitor); ok {
			monitor.AddGauge(constants.PollerDistribution, labels, delta)
		}
	}
	addGauge(1)
	defer addGauge(-1)
	defer run.cancel()

	p.recordPollerLifeCycle(constants.Start)
	for {
		select {
		case <-run.stop:
			return
		case currentTime := <-run.ticker.C:
			// Re-check stop in case Stop raced with the tick; do not dispatch
			// a retrieval that a cancel already asked us to skip.
			select {
			case <-run.stop:
				return
			default:
			}
			p.recordPollerLifeCycle(constants.Running)
			timeBucket := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), currentTime.Hour(), currentTime.Minute(), 0, 0, currentTime.Location())
			go p.scheduleRetrievalImpl.GetSchedules(p.AppName, p.PartitionId, timeBucket)
		}
	}
}

func (p *Poller) Stop() {
	p.recordPollerLifeCycle(constants.Stop)
	glog.Infof("Stopping poller for %s.%d", p.AppName, p.PartitionId)
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.run != nil {
		p.run.cancel()
	}
}
