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

package connectors

import (
	"github.com/myntra/goscheduler/conf"
	"github.com/myntra/goscheduler/dao"
	"github.com/myntra/goscheduler/monitoring"
	"net/http"
	"time"
)

// Connector represents a component that manages various worker pools for different tasks.
type Connector struct {
	Config      *conf.Configuration
	ClusterDao  dao.ClusterDao
	ScheduleDao dao.ScheduleDao
	HttpClient  *http.Client
	Monitoring  *monitoring.Monitoring
}

// NewConnector creates a new Connector instance with the given configuration, DAOs, and monitoring.
func NewConnector(config *conf.Configuration, clusterDao dao.ClusterDao, scheduleDAO dao.ScheduleDao, monitoring *monitoring.Monitoring) *Connector {
	client := &http.Client{
		Timeout:       config.HttpConnector.TimeoutMillis * time.Millisecond,
	}
	return &Connector{
		Config:      config,
		ClusterDao:  clusterDao,
		ScheduleDao: scheduleDAO,
		HttpClient:  client,
		Monitoring:  monitoring,
	}
}

// InitConnectors initializes all the worker pools managed by the Connector.
func (c *Connector) InitConnectors() {
	c.initHttpWorkers()
	c.initAggregateWorkers()
	c.initStatusUpdatePool()
	c.initCronRetriever()
	c.initBulkActionWorkers()
}
