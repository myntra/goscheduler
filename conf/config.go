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

package conf

import (
	"flag"
	"os"
	"time"

	"github.com/myntra/goscheduler/constants"

	"github.com/gocql/gocql"
	"github.com/golang/glog"
	"github.com/jinzhu/configor"
)

// ClusterConfig represents the configuration of a ringpop cluster, including its name,
// address, TChannel port, bootstrap servers, and other settings.
type ClusterConfig struct {
	ClusterName      string   // Name of the cluster
	Address          string   // Address of the cluster
	TChannelPort     string   // TChannel port for inter-node communication
	BootStrapServers []string // List of bootstrap servers for cluster formation
	JoinSize         int      // Number of nodes required to form a cluster
	Log              Log      // Logging configuration

	//TODO: move it to DB config
	PageSize      int // Page size for pagination
	NumRetry      int // Number of retries for failed operations
	ReplicaPoints int // Number of replica points for data replication
}

// Log represents the logging configuration, including whether logging is enabled.
type Log struct {
	Enable bool // Indicates if logging is enabled
}

// CassandraConfig represents the configuration for connecting to a Cassandra
// cluster, including hosts, consistency level, data center, and connection pool settings.
type CassandraConfig struct {
	Hosts          string            // Comma-separated list of Cassandra hosts
	Consistency    gocql.Consistency // Consistency level for Cassandra operations
	DataCenter     string            // Name of the data center to connect to
	ConnectionPool ConnectionPool    // Connection pool configuration
}

// ClusterDBConfig represents the configuration for a cluster database, including
// keyspace, database configuration, and entity history size.
type ClusterDBConfig struct {
	ClusterKeySpace   string          // Keyspace for the cluster
	DBConfig          CassandraConfig // Cassandra configuration for the cluster
	EntityHistorySize int             // Size of the entity history
}

// ScheduleDBConfig represents the configuration for a schedule database, including
// keyspace, table name, database configuration, and TTL settings.
type ScheduleDBConfig struct {
	ScheduleKeySpace  string          // Keyspace for the schedule
	ScheduleTableName string          // Table name for the schedule
	DBConfig          CassandraConfig // Cassandra configuration for the schedule
}

// PollerConfig represents the configuration for a poller, including interval,
// buffer size, and default count.
type PollerConfig struct {
	Interval     int    // Polling interval in seconds
	DefaultCount uint32 // Default number of items to be polled
}

// ConnectionPool represents the configuration for a connection pool, including
// initial connect timeout, connect timeout, and maximum number of connections.
type ConnectionPool struct {
	InitialConnectTimeout int // Initial connection timeout in milliseconds
	ConnectTimeout        int // Connection timeout in milliseconds
	MaxNumConnections     int // Maximum number of connections in the pool
}

// RingPopConfig represents the configuration for a RingPop instance, including
// the host and port information.
type RingPopConfig struct {
	Host string // Hostname or IP address of the RingPop instance
	Port int    // Port number on which the RingPop instance is listening
}

// StatsdConfig represents the configuration for a StatsD client, including
// whether it is enabled, its address, and an optional prefix for metrics.
type StatsdConfig struct {
	Enabled bool   // Indicates if the StatsD client is enabled
	Address string // Address of the StatsD server, in the format "host:port"
	Prefix  string // Optional prefix for metric names
}

// HttpConnectorConfig represents the configuration for an HTTP connector,
// including the number of routines, maximum retries, and timeout settings.
type HttpConnectorConfig struct {
	Routines      int           // Number of concurrent routines for processing
	MaxRetry      int           // Maximum number of retries for failed requests
	TimeoutMillis time.Duration // Timeout for HTTP requests in milliseconds
}

// EventListener represents the configuration for an event listener, including
// the application name, event name, number of concurrent listeners, and consumer count.
type EventListener struct {
	AppName             string // Name of the application associated with the listener
	EventName           string // Name of the event the listener is listening for
	ConcurrentListeners int    // Number of concurrent listeners
	ConsumerCount       int    // Number of consumers for each listener
}

// NewrelicConfig represents the configuration for a New Relic monitoring agent,
// including the license key, application name, and whether it is enabled.
type NewrelicConfig struct {
	LicenseKey string // License key for the New Relic agent
	AppName    string // Name of the application being monitored
	Enable     bool   // Indicates if the New Relic agent is enabled
}

// Configurations for cron conversions
type CronConfig struct {
	App string // Name of the special app which converts the recurring schedules to one time schedules.
	// A special schedule retriever will be used by this app pollers.
	Window   time.Duration // The time window within which new future one time schedules for the recurring schedules will be created.
	Routines int           // Number of worker routines converting the schedules to one time.
}

// AggregateSchedulesConfig represents the configuration options for schedule aggregation.
type AggregateSchedulesConfig struct {
	BufferSize  int // Channel buffer size
	Routines    int // Number of workers aggregating schedules
	BatchSize   int // Batchsize of schedules for bulk update
	FlushPeriod int // Flush period in seconds after which schedule batches are pushed to db (even if they are not full)
}

// StatusUpdateConfig represents the configuration options for status updates of schedules.
type StatusUpdateConfig struct {
	Routines int // Number of workers updating status of schedules
}

// NodeCrashReconcile represents the configuration options for reconciling node crashes.
type NodeCrashReconcile struct {
	NeedsReconcile  bool
	ReconcileOffset int
}

// TODO: Need to take care of maintaining history for delete action
// BulkActionConfig represents the configuration options for bulk actions.
type BulkActionConfig struct {
	BufferSize int // Channel buffer size
	Routines   int // Number of workers aggregating schedules
}

// MonitoringConfig represents the configuration options for monitoring, including
// Statsd configuration.
type MonitoringConfig struct {
	Statsd *StatsdConfig // Configuration options for Statsd
}

type AppLevelConfiguration struct {
	// Requests are rejected if the schedule time is beyond specified FutureScheduleCreationPeriod (in days) from current time
	FutureScheduleCreationPeriod int

	// Period in days for which schedules are kept in DB after the schedules are fired
	FiredScheduleRetentionPeriod int

	// Maximum Payload size in bytes allowed
	PayloadSize int

	// Http Retries for requests
	HttpRetries int

	// HTTP Timeout in milliseconds for requests
	HttpTimeout int
}

type DCConfig struct {
	// used to prefix appIds
	Prefix string

	// location of the DC
	Location string
}

// GetAddress concatenates host and port strings and returns an address in the
// format of "host:port".
func GetAddress(host string, port string) string {
	return host + ":" + port
}

// Configuration represents the main configuration structure, including fields
// for various components and settings.
type Configuration struct {
	HttpPort                 string                   // Port for the HTTP server
	ConfigFile               string                   `json:"-"` // Configuration file path
	SchemaPath               string                   // Configuration options for Cassandra Schema path
	Cluster                  ClusterConfig            // Configuration options for the cluster
	ClusterDB                ClusterDBConfig          // Configuration options for the cluster database
	ScheduleDB               ScheduleDBConfig         // Configuration options for the schedule database
	Poller                   PollerConfig             // Configuration options for the poller
	MonitoringConfig         MonitoringConfig         // Configuration options for monitoring
	HttpConnector            HttpConnectorConfig      // Configuration options for the HTTP connector
	CronConfig               CronConfig               // Configuration options for the cron scheduler
	StatusUpdateConfig       StatusUpdateConfig       // Configuration options for status updates
	AggregateSchedulesConfig AggregateSchedulesConfig // Configuration options for schedule aggregation
	NodeCrashReconcile       NodeCrashReconcile       // Configuration options for node crash reconciliation
	BulkActionConfig         BulkActionConfig         // Configuration options for bulk actions
	AppLevelConfiguration    AppLevelConfiguration    // Configuration options for app level configuration
	DCConfig                 DCConfig                 // Configuration options for DC configuration
}

var defaultConfig = Configuration{
	HttpPort:   "8080",
	ConfigFile: "./conf/conf.json",
	SchemaPath: "/src/goscheduler/cassandra/cassandra.cql",
	Cluster: ClusterConfig{
		ClusterName:      "goscheduler",
		Address:          "127.0.0.1:9091",
		TChannelPort:     "9091",
		BootStrapServers: []string{"127.0.0.1:9091"},
		JoinSize:         1,
		PageSize:         1000,
		NumRetry:         3,
		ReplicaPoints:    2,
	},
	ClusterDB: ClusterDBConfig{
		ClusterKeySpace: "cluster",
		DBConfig: CassandraConfig{
			Hosts:       "127.0.0.1",
			Consistency: gocql.One,
			DataCenter:  "",
			ConnectionPool: ConnectionPool{
				InitialConnectTimeout: 1000,
				ConnectTimeout:        5000,
				MaxNumConnections:     4,
			},
		},
		EntityHistorySize: 5,
	},
	ScheduleDB: ScheduleDBConfig{
		ScheduleKeySpace: "schedule_management",
		DBConfig: CassandraConfig{
			Hosts:       "127.0.0.1",
			Consistency: gocql.One,
			DataCenter:  "",
			ConnectionPool: ConnectionPool{
				InitialConnectTimeout: 1000,
				ConnectTimeout:        5000,
				MaxNumConnections:     4,
			},
		},
	},
	Poller: PollerConfig{
		Interval:     60,
		DefaultCount: 5,
	},
	MonitoringConfig: MonitoringConfig{Statsd: nil},
	HttpConnector: HttpConnectorConfig{
		Routines:      10,
		MaxRetry:      3,
		TimeoutMillis: 1000,
	},
	CronConfig: CronConfig{
		App:      "Athena",
		Window:   5,
		Routines: 10,
	},
	StatusUpdateConfig: StatusUpdateConfig{Routines: 5},
	AggregateSchedulesConfig: AggregateSchedulesConfig{
		BufferSize:  1000,
		Routines:    5,
		BatchSize:   10,
		FlushPeriod: 60,
	},
	NodeCrashReconcile: NodeCrashReconcile{
		NeedsReconcile:  true,
		ReconcileOffset: 3,
	},
	BulkActionConfig: BulkActionConfig{
		BufferSize: 1000,
		Routines:   10,
	},
	AppLevelConfiguration: AppLevelConfiguration{
		FutureScheduleCreationPeriod: 7,
		FiredScheduleRetentionPeriod: 1,
		PayloadSize:                  1024,
		HttpRetries:                  1,
		HttpTimeout:                  1000,
	},
	DCConfig: DCConfig{
		Prefix:   "",
		Location: "Local",
	},
}

type Option func(*Configuration)

func WithHTTPPort(port string) Option {
	return func(c *Configuration) {
		c.HttpPort = port
	}
}

func WithConfigFile(configFile string) Option {
	return func(c *Configuration) {
		c.ConfigFile = configFile
	}
}

func WithSchemaPath(schemaPath string) Option {
	return func(c *Configuration) {
		c.SchemaPath = schemaPath
	}
}

func WithClusterConfig(clusterConfig ClusterConfig) Option {
	return func(c *Configuration) {
		c.Cluster = clusterConfig
	}
}

func WithClusterDB(clusterDB ClusterDBConfig) Option {
	return func(c *Configuration) {
		c.ClusterDB = clusterDB
	}
}

func WithScheduleDB(scheduleDB ScheduleDBConfig) Option {
	return func(c *Configuration) {
		c.ScheduleDB = scheduleDB
	}
}

func WithPoller(poller PollerConfig) Option {
	return func(c *Configuration) {
		c.Poller = poller
	}
}

func WithMonitoringConfig(monitoringConfig MonitoringConfig) Option {
	return func(c *Configuration) {
		c.MonitoringConfig = monitoringConfig
	}
}

func WithHttpConnectorConfig(httpConnectorConfig HttpConnectorConfig) Option {
	return func(c *Configuration) {
		c.HttpConnector = httpConnectorConfig
	}
}

func WithCronConfig(cronConfig CronConfig) Option {
	return func(c *Configuration) {
		c.CronConfig = cronConfig
	}
}

func WithStatusUpdateConfig(statusUpdateConfig StatusUpdateConfig) Option {
	return func(c *Configuration) {
		c.StatusUpdateConfig = statusUpdateConfig
	}
}

func WithUpdateStatusConfig(statusUpdateConfig StatusUpdateConfig) Option {
	return func(c *Configuration) {
		c.StatusUpdateConfig = statusUpdateConfig
	}
}

func WithAggregateSchedulesConfig(aggregateSchedulesConfig AggregateSchedulesConfig) Option {
	return func(c *Configuration) {
		c.AggregateSchedulesConfig = aggregateSchedulesConfig
	}
}

func WithNodeCrashReconcileConfig(reconcile NodeCrashReconcile) Option {
	return func(c *Configuration) {
		c.NodeCrashReconcile = reconcile
	}
}

func WithBulkActionConfig(bulkActionConfig BulkActionConfig) Option {
	return func(c *Configuration) {
		c.BulkActionConfig = bulkActionConfig
	}
}

func WithAppLevelConfiguration(appLevelConfiguration AppLevelConfiguration) Option {
	return func(c *Configuration) {
		c.AppLevelConfiguration = appLevelConfiguration
	}
}

func WithDCConfiguration(dcConfig DCConfig) Option {
	return func(c *Configuration) {
		c.DCConfig = dcConfig
	}
}

func NewConfig(opts ...Option) *Configuration {
	config := defaultConfig
	for _, opt := range opts {
		opt(&config)
	}
	return &config
}

// ParseFlags parses command line flags and returns the values.
func ParseFlags() (string, string, string) {
	var port string
	var host string
	var confFile string

	defaultConfFile := os.Getenv("GOPATH") + "/src/goscheduler/conf/conf.json"
	flag.StringVar(&confFile, "conf", defaultConfFile, "goscheduler config file")
	flag.StringVar(&port, "p", "9091", "goscheduler ringpop port")
	flag.StringVar(&host, "h", "127.0.0.1", "goscheduler server host")
	flag.Parse()

	return confFile, port, host + ":" + port
}

// getHttpPortFromEnv retrieves the HTTP port from the environment variable.
func getHttpPortFromEnv() string {
	httpPort := os.Getenv("PORT")
	if httpPort == "" {
		httpPort = "8080"
	}
	return httpPort
}

// InitConfig initializes the configuration by setting default values,
// parsing command line flags, and loading the configuration from the
// specified file. It returns a pointer to a Configuration struct.
func InitConfig(confFile, port, address string) *Configuration {
	httpPort := getHttpPortFromEnv()
	config := LoadConfig(confFile)

	//override values if provided as arguments
	config.HttpPort = httpPort
	config.Cluster.TChannelPort = port
	config.Cluster.Address = address

	return config
}

// LoadConfig loads the configuration from a JSON file and populates a
// Configuration struct. It returns a pointer to the populated struct.
// If there's an error while loading the configuration file, the function
// will log a fatal error and exit the application.
func LoadConfig(confFile string) *Configuration {
	var GlobalConfig Configuration
	if err := configor.Load(&GlobalConfig, confFile); err != nil {
		glog.Fatalln(constants.ErrorConfig, err)
	}
	glog.Infoln("Config:", GlobalConfig)
	return &GlobalConfig
}
