package cassandra

import (
	"errors"
	"github.com/gocql/gocql"
	"github.com/golang/glog"
	"github.com/myntra/goscheduler/conf"
	"github.com/myntra/goscheduler/db_wrapper"
	"io/ioutil"
	"strings"
	"time"
)

// getCassandraHosts converts a comma-separated list of Cassandra hosts into a
// slice of strings representing the individual host names or IP addresses. This
// function is used to parse the Cassandra hosts specified in the configuration
// file and create a list of hosts to connect to.
func getCassandraHosts(cassandraHosts string) []string {
	return strings.Split(cassandraHosts, ",")
}

//skip this for production envs
func CassandraInit(cassandraConfig conf.CassandraConfig, cassandraInitialisationFile string) {
	createSession, err := GetSessionInterface(cassandraConfig, "")

	if err != nil {
		panic(errors.New("GetSession failed with error " + err.Error()))
	}

	var bytes []byte
	bytes, err = ioutil.ReadFile(cassandraInitialisationFile)
	if err != nil {
		panic(errors.New("ReadFile " + cassandraInitialisationFile + " failed with error " + err.Error()))
	}

	cqlStmts := strings.SplitAfter(string(bytes), ";")

	for _, cqlStmt := range cqlStmts {
		cqlStmt = strings.Trim(cqlStmt, "\r\n")
		if cqlStmt == "" {
			continue
		}
		glog.Info("Here " + cqlStmt)
		err = createSession.Query(cqlStmt).Exec()
		if err != nil {
			panic(errors.New("CQL execution failed with  " + err.Error()))
		}
	}
	createSession.Close()
}

// withPool sets the host selection policy for the given cluster configuration
// to a DC-aware round-robin policy if the data center is specified in the
// Cassandra configuration. This can improve query performance by selecting
// nodes that are closer to the data center where the queries are being run.
func withPool(cluster *gocql.ClusterConfig, config conf.CassandraConfig) *gocql.ClusterConfig {
	if len(config.DataCenter) != 0 {
		cluster.PoolConfig = gocql.PoolConfig{
			HostSelectionPolicy: gocql.DCAwareRoundRobinPolicy(config.DataCenter)}
	}

	return cluster
}

// withAuthenticator sets the password authenticator for the given cluster
// configuration to use the username and password specified in the Vault
// configuration, if Vault is enabled. This allows the Cassandra client to
// authenticate with the Cassandra cluster using credentials stored in Vault.
func withAuthenticator(cluster *gocql.ClusterConfig, config conf.CassandraConfig) *gocql.ClusterConfig {
	if config.VaultConfig.Enabled {
		cluster.Authenticator = gocql.PasswordAuthenticator{
			Username: config.VaultConfig.Username,
			Password: config.VaultConfig.Password,
		}
	}
	return cluster
}

// Deprecated: This function is used to get a Cassandra gocql.Session.
// In order to get the ability to mock methods we are using GetSessionInterface which provides wrapper over gocql.Session
func GetSession(cassandraConfig conf.CassandraConfig, keyspace string) (*gocql.Session, error) {
	hosts := getCassandraHosts(cassandraConfig.Hosts)
	glog.Infof("Cassandra hosts to connnect %s ", hosts)

	cluster := gocql.NewCluster(hosts...)
	cluster.Keyspace = keyspace
	cluster.Consistency = cassandraConfig.Consistency
	cluster.Timeout = time.Duration(cassandraConfig.ConnectionPool.InitialConnectTimeout) * time.Millisecond
	cluster.ConnectTimeout = time.Duration(cassandraConfig.ConnectionPool.ConnectTimeout) * time.Millisecond
	cluster.NumConns = cassandraConfig.ConnectionPool.MaxNumConnections

	withPool(cluster, cassandraConfig)
	withAuthenticator(cluster, cassandraConfig)

	session, err := cluster.CreateSession()

	if err != nil {
		glog.Error("ERROR CONNECTING TO CASSANDRA", err)
		return nil, err
	}
	return session, nil
}

// GetSessionInterface returns a new Cassandra session interface for the specified
// keyspace using the given Cassandra configuration. The function creates a new
// cluster configuration with the specified hosts and keyspace, and sets various
// parameters such as page size, consistency level, and connection timeouts. The
// function also applies optional modifications to the cluster configuration using
// the withPool and withAuthenticator functions. Finally, the function creates a
// new session and returns a wrapper object that implements the db_wrapper.SessionInterface
func GetSessionInterface(cassandraConfig conf.CassandraConfig, keyspace string) (db_wrapper.SessionInterface, error) {
	hosts := getCassandraHosts(cassandraConfig.Hosts)
	glog.Infof("Cassandra hosts to connect to %s for keyspace %s", hosts, keyspace)

	cluster := gocql.NewCluster(hosts...)
	cluster.Keyspace = keyspace
	cluster.Consistency = cassandraConfig.Consistency
	cluster.Timeout = time.Duration(cassandraConfig.ConnectionPool.InitialConnectTimeout) * time.Millisecond
	cluster.ConnectTimeout = time.Duration(cassandraConfig.ConnectionPool.ConnectTimeout) * time.Millisecond
	cluster.NumConns = cassandraConfig.ConnectionPool.MaxNumConnections
	withPool(cluster, cassandraConfig)
	withAuthenticator(cluster, cassandraConfig)

	session, err := cluster.CreateSession()
	if err != nil {
		glog.Error("ERROR CONNECTING TO CASSANDRA", err)
		return nil, err
	}
	return db_wrapper.NewSession(session), nil
}
