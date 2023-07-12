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
