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

package scheduler

import (
	"os"

	"github.com/gorilla/mux"
	"github.com/myntra/goscheduler/cassandra"
	"github.com/myntra/goscheduler/cluster"
	c "github.com/myntra/goscheduler/conf"
	conn "github.com/myntra/goscheduler/connectors"
	"github.com/myntra/goscheduler/dao"
	m "github.com/myntra/goscheduler/monitoring"
	"github.com/myntra/goscheduler/poller"
	r "github.com/myntra/goscheduler/retrievers"
	"github.com/myntra/goscheduler/server"
	s "github.com/myntra/goscheduler/service"
	st "github.com/myntra/goscheduler/store"
)

// Scheduler is a struct that holds pointers to various components of the scheduler.
type Scheduler struct {
	Config     *c.Configuration
	Router     *mux.Router
	Supervisor *cluster.Supervisor
	Service    *s.Service
	Connectors *conn.Connector
	Retrievers r.Retrievers
	Monitor    m.Monitor
}

// initCassandra initializes the Cassandra database with the given configuration and schema.
func initCassandra(conf *c.Configuration, createSchema bool) {
	if createSchema {
		cassandra.CassandraInit(conf.ClusterDB.DBConfig, os.Getenv("GOPATH")+conf.SchemaPath)
	}
}

// initDAOs creates and returns the implementation objects for the Cluster and Schedule data access objects.
func initDAOs(conf *c.Configuration, monitor m.Monitor) (dao.ClusterDao, dao.ScheduleDao) {
	clusterDao := dao.GetClusterDaoImpl(conf, monitor)
	scheduleDao := dao.GetScheduleDaoImpl(conf, monitor)
	return clusterDao, scheduleDao
}

// initRetrievers initializes the retrievers for the schedules and clusters using the configuration provided.
func initRetrievers(conf *c.Configuration, clusterDao dao.ClusterDao, scheduleDao dao.ScheduleDao, monitor m.Monitor) r.Retrievers {
	return r.InitRetrievers(conf, clusterDao, scheduleDao, monitor)
}

// initSupervisor creates a new Supervisor object that manages the cluster of nodes running the scheduler.
func initSupervisor(conf *c.Configuration, retrievers r.Retrievers, clusterDao dao.ClusterDao, monitor m.Monitor) *cluster.Supervisor {
	supervisor := cluster.NewSupervisor(
		poller.NewPollerFactory(retrievers, conf.Poller, monitor),
		clusterDao,
		monitor,
		cluster.WithClusterName(conf.Cluster.ClusterName),
		cluster.WithAddress(conf.Cluster.Address),
		cluster.WithBootStrapServers(conf.Cluster.BootStrapServers),
		cluster.WithJoinSize(conf.Cluster.JoinSize),
		cluster.WithLogEnabled(conf.Cluster.Log.Enable),
		cluster.WithStatsD(m.GetRingPopStatsD(conf.MonitoringConfig.Statsd)),
		cluster.WithReplicaPoints(conf.Cluster.ReplicaPoints),
		cluster.WithReconciliationEnabled(conf.NodeCrashReconcile.NeedsReconcile),
		cluster.WithReconciliationOffset(conf.NodeCrashReconcile.ReconcileOffset),
	)
	supervisor.InitRingPop()
	supervisor.Boot()
	return supervisor
}

// initConnectors creates the connector object used to communicate with the cluster nodes.
func initConnectors(conf *c.Configuration, clusterDao dao.ClusterDao, scheduleDao dao.ScheduleDao, monitor m.Monitor, callbackWorkers bool) *conn.Connector {
	t := &st.Task{Conf: conf}
	t.InitTaskQueues()
	connector := conn.NewConnector(conf, clusterDao, scheduleDao, monitor)
	connector.InitConnectors(callbackWorkers)
	return connector
}

// initService creates a new Service object that handles the scheduling logic and communication with the cluster nodes.
func initService(conf *c.Configuration, supervisor cluster.SupervisorHandler, clusterDao dao.ClusterDao, scheduleDao dao.ScheduleDao, monitor m.Monitor) *s.Service {
	return s.NewService(conf, supervisor, clusterDao, scheduleDao, monitor)
}

// initServer starts an HTTP server using the provided configuration and Service object to handle requests.
func initServer(conf *c.Configuration, router *mux.Router, service *s.Service) *server.Server {
	return server.NewHTTPServer(conf.HttpPort, router, service)
}

// initMonitoring initializes the monitoring component with the given configuration.
func initMonitoring() m.Monitor {
	return m.NewPrometheusMonitor()
}

func initCallbackRegistry(registry map[string]st.Factory) {
	st.InitializeCallbackRegistry(registry)
}

// New creates a new Scheduler instance with a given configuration and callback factories.
// This is a base constructor that uses configuration and callback factory objects directly.
func New(conf *c.Configuration, callbackFactories map[string]st.Factory) *Scheduler {
	initCassandra(conf, true)
	initCallbackRegistry(callbackFactories)
	monitor := initMonitoring()
	clusterDao, schedulerDao := initDAOs(conf, monitor)
	retrievers := initRetrievers(conf, clusterDao, schedulerDao, monitor)
	supervisor := initSupervisor(conf, retrievers, clusterDao, monitor)
	connectors := initConnectors(conf, clusterDao, schedulerDao, monitor, true)
	service := initService(conf, supervisor, clusterDao, schedulerDao, monitor)
	router := mux.NewRouter().StrictSlash(true)
	svr := initServer(conf, router, service)
	go svr.StartServer()
	return &Scheduler{
		Config:     conf,
		Router:     router,
		Supervisor: supervisor,
		Service:    service,
		Connectors: connectors,
		Retrievers: retrievers,
		Monitor:    monitor,
	}
}

// TODO: Reformat this constructor
// NewScheduler creates a new Scheduler instance with a given params.
func NewScheduler(conf *c.Configuration, callbackFactories map[string]st.Factory, clusterDao dao.ClusterDao, scheduleDao dao.ScheduleDao, monitor m.Monitor, createSchema bool, callbackWorkers bool) *Scheduler {
	initCassandra(conf, createSchema)
	initCallbackRegistry(callbackFactories)
	retrievers := initRetrievers(conf, clusterDao, scheduleDao, monitor)
	supervisor := initSupervisor(conf, retrievers, clusterDao, monitor)
	connectors := initConnectors(conf, clusterDao, scheduleDao, monitor, callbackWorkers)
	service := initService(conf, supervisor, clusterDao, scheduleDao, monitor)
	router := mux.NewRouter().StrictSlash(true)
	initServer(conf, router, service)
	return &Scheduler{
		Config:     conf,
		Router:     router,
		Supervisor: supervisor,
		Service:    service,
		Connectors: connectors,
		Retrievers: retrievers,
		Monitor:    monitor,
	}
}

// FromFile creates a new Scheduler instance using a configuration loaded from a file.
// This constructor is useful when configuration is stored in a file.
func FromConfFile(confFile string) *Scheduler {
	scheduler := New(c.LoadConfig(confFile), map[string]st.Factory{})
	go scheduler.Supervisor.WaitForTermination()
	return scheduler
}

// WithOptions creates a new Scheduler instance using a configuration constructed from the given options.
// This constructor is useful when you want to construct a Scheduler using a set of options.
func WithOptions(opts ...c.Option) *Scheduler {
	return New(c.NewConfig(opts...), map[string]st.Factory{})
}

// FromFileWithCallbacks creates a new Scheduler instance using a configuration loaded from a file and
// a map of callback factories. This constructor is useful when both configuration and callback
// customizations are needed and configuration is stored in a file.
func FromFileWithCallbacks(callbackFactories map[string]st.Factory, confFile string) *Scheduler {
	scheduler := New(c.LoadConfig(confFile), callbackFactories)
	go scheduler.Supervisor.WaitForTermination()
	return scheduler
}

// WithOptionsWithCallbacks creates a new Scheduler instance using a configuration constructed from
// the given options and a map of callback factories. This constructor is useful when both configuration
// and callback customizations are needed, and you want to construct a Scheduler using a set of options.
func WithOptionsWithCallbacks(callbackFactories map[string]st.Factory, opts ...c.Option) *Scheduler {
	scheduler := New(c.NewConfig(opts...), callbackFactories)
	go scheduler.Supervisor.WaitForTermination()
	return scheduler
}
