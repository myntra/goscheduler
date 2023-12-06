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

package dao

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/golang/glog"
	"github.com/imdario/mergo"
	"github.com/myntra/goscheduler/cassandra"
	e "github.com/myntra/goscheduler/cluster_entity"
	"github.com/myntra/goscheduler/conf"
	"github.com/myntra/goscheduler/constants"
	"github.com/myntra/goscheduler/db_wrapper"
	p "github.com/myntra/goscheduler/monitoring"
	"github.com/myntra/goscheduler/store"
	"strconv"
	"sync"
)

type AppMap struct {
	lock sync.RWMutex
	m    map[string]store.App
}

type ClusterDaoImplCassandra struct {
	Session    db_wrapper.SessionInterface
	AppMap     AppMap
	Conf       *conf.Configuration
	Monitoring *p.Monitoring
}

var (
	KeyEntityTable = "entity"
	KeyNodeTable   = "nodes"
	KeyAppTable    = "apps"
	MaxConfigApp   = "maxConfig"

	KeyEntitiesOfNode    = "SELECT id, status FROM " + KeyNodeTable + " WHERE nodename='%s';"
	KeyGetAllEntities    = "SELECT id, nodename, status, history FROM " + KeyEntityTable + ";"
	KeyGetEntity         = "SELECT id, nodename, status, history FROM " + KeyEntityTable + " WHERE id='%s';"
	KeyUpdateEntityInfo  = "UPDATE " + KeyEntityTable + " SET nodename='%s', status=%d, history='%s' WHERE id='%s';"
	QueryInsertEntity    = "INSERT INTO " + KeyEntityTable + " (id, nodename, status) VALUES (?, ?, ?)"
	QueryInsertApp       = "INSERT INTO " + KeyAppTable + " (id, partitions, active, configuration) VALUES (?, ?, ?, ?)"
	KeyAppById           = "SELECT id, partitions, active, configuration FROM " + KeyAppTable + " WHERE id='%s';"
	KeyAppByIds          = "SELECT id, partitions, active, configuration FROM " + KeyAppTable + " WHERE id in (?, ?);"
	KeyGelAllApps        = "SELECT id, partitions, active, configuration FROM " + KeyAppTable + ";"
	QueryUpdateAppStatus = "UPDATE " + KeyAppTable + " set active = %s where id='%s'"
	QueryGetConfig       = "SELECT configuration FROM " + KeyAppTable + " WHERE id='%s';"
	QueryUpdateConfig    = "UPDATE " + KeyAppTable + " SET configuration='%s' WHERE id='%s';"
)

// TODO: Should we make it singleton?
func GetClusterDaoImpl(conf *conf.Configuration, monitoring *p.Monitoring) *ClusterDaoImplCassandra {
	session, err := cassandra.GetSessionInterface(conf.ClusterDB.DBConfig, conf.ClusterDB.ClusterKeySpace)
	if err != nil {
		err = errors.New(fmt.Sprintf("Cassandra initialisation failed for configuration: %+v with error %s", conf.ClusterDB.DBConfig, err.Error()))
		panic(err)
	}
	return &ClusterDaoImplCassandra{
		Session: session,
		Conf:    conf,
		AppMap: AppMap{
			lock: sync.RWMutex{},
			m:    make(map[string]store.App),
		},
		Monitoring: monitoring,
	}
}

func (c *ClusterDaoImplCassandra) getDefaultApps() (map[string]bool, error) {
	var appId string
	var active bool
	var config string
	var partitions uint32
	apps := make(map[string]bool)

	iter := c.Session.
		Query(KeyAppByIds, MaxConfigApp, c.Conf.CronConfig.App).
		Consistency(c.Conf.ClusterDB.DBConfig.Consistency).
		Iter()

	for iter.Scan(&appId, &partitions, &active, &config) {
		apps[appId] = true
	}

	if err := iter.Close(); err != nil {
		glog.Errorf("Error occurred while getting default apps: %s", err.Error())
		return apps, err
	}

	return apps, nil
}

func (c *ClusterDaoImplCassandra) createDefaultAppsIfRequired() {
	apps, err := c.getDefaultApps()

	if err != nil {
		glog.Fatal(err)
	}

	if _, ok := apps[MaxConfigApp]; !ok {
		configuration := store.Configuration{
			FutureScheduleCreationPeriod: c.Conf.AppLevelConfiguration.FutureScheduleCreationPeriod,
			FiredScheduleRetentionPeriod: c.Conf.AppLevelConfiguration.FiredScheduleRetentionPeriod,
			PayloadSize:                  c.Conf.AppLevelConfiguration.PayloadSize,
			HttpRetries:                  c.Conf.AppLevelConfiguration.HttpRetries,
			HttpTimeout:                  c.Conf.AppLevelConfiguration.HttpTimeout,
		}

		maxConfigApp := store.App{
			AppId:         MaxConfigApp,
			Partitions:    0,
			Active:        false,
			Configuration: configuration,
		}

		if err := c.InsertApp(maxConfigApp); err != nil {
			glog.Fatal(err)
		}

		glog.Info("maxConfig app created!")
	}

	if _, ok := apps[c.Conf.CronConfig.App]; !ok {
		cronApp := store.App{
			AppId:         c.Conf.CronConfig.App,
			Partitions:    c.Conf.Poller.DefaultCount,
			Active:        true,
			Configuration: store.Configuration{},
		}

		if err := c.InsertApp(cronApp); err != nil {
			glog.Fatal(err)
		}

		var partition uint32 = 0
		for ; partition < cronApp.Partitions; partition++ {
			entity := e.EntityInfo{
				Id:      cronApp.AppId + constants.PollerKeySep + strconv.Itoa(int(partition)),
				Node:    c.Conf.Cluster.Address,
				Status:  0,
				History: "",
			}

			if err := c.CreateEntity(entity); err != nil {
				glog.Fatal(err)
			}
		}
		glog.Info("Cron app created!")
	}
}

// Get all the entities assigned to the node
// Raise a fatal exception in case there is an exception getting all entities
// TODO: Return error
func (c *ClusterDaoImplCassandra) GetAllEntitiesInfoOfNode(nodeName string) []e.EntityInfo {
	var id string
	var status int
	var entities []e.EntityInfo

	query := fmt.Sprintf(KeyEntitiesOfNode, nodeName)
	iter := c.Session.
		Query(query).
		Consistency(c.Conf.ClusterDB.DBConfig.Consistency).
		PageSize(c.Conf.Cluster.PageSize).Iter()

	for iter.Scan(&id, &status) {
		entities = append(entities, e.EntityInfo{Id: id, Node: nodeName, Status: status})
	}

	if err := iter.Close(); err != nil {
		glog.Fatal(err)
	}

	glog.Infof("GetAllEntitiesInfoOfNode result is : %+v", entities)
	return entities
}

// Get all the entities
// Raise a fatal exception in case there is an exception getting all entities
// TODO: Return error
func (c *ClusterDaoImplCassandra) GetAllEntitiesInfo() []e.EntityInfo {
	var id string
	var nodeName string
	var status int
	var history string

	var entities []e.EntityInfo
	iter := c.Session.
		Query(KeyGetAllEntities).
		Consistency(c.Conf.ClusterDB.DBConfig.Consistency).
		PageSize(c.Conf.Cluster.PageSize).
		Iter()

	for iter.Scan(&id, &nodeName, &status, &history) {
		entities = append(entities, e.EntityInfo{Id: id, Node: nodeName, Status: status, History: history})
	}

	if err := iter.Close(); err != nil {
		glog.Fatal(err)
	}

	glog.Infof("GetAllEntitiesInfo result is : %+v", entities)
	return entities
}

// Get entity by id
// TODO: Return error
func (c *ClusterDaoImplCassandra) GetEntityInfo(id string) e.EntityInfo {
	var nodeName string
	var status int
	var history string

	query := fmt.Sprintf(KeyGetEntity, id)
	glog.Info(query)

	if err := c.Session.Query(query).Consistency(c.Conf.ClusterDB.DBConfig.Consistency).Scan(&id, &nodeName, &status, &history); err != nil {
		glog.Errorf("Error %s while querying %s", err.Error(), query)
		return e.EntityInfo{}
	}

	entity := e.EntityInfo{
		Id:      id,
		Node:    nodeName,
		Status:  status,
		History: history,
	}

	glog.Infof("GetEntityInfo result for %s is : %+v", id, entity)
	return entity
}

// By default if Active parameter is not set it will create the app in deactivated state
func (c *ClusterDaoImplCassandra) InsertApp(app store.App) error {
	var err error
	var config []byte

	// skip validations if the appName is same as maxConfigApp or
	// if the configurations are empty
	if app.AppId != MaxConfigApp {
		if err = c.ValidateConfigurations(app.Configuration); err != nil {
			return err
		}
	}

	if config, err = json.Marshal(app.Configuration); err != nil {
		return err
	}

	return c.Session.Query(QueryInsertApp, app.AppId, app.Partitions, app.Active, string(config)).Exec()
}

func (c *ClusterDaoImplCassandra) GetApp(appName string) (store.App, error) {
	if app, found := c.check(appName); found {
		return app, nil
	}
	return c.getApp(appName)
}

// getApp gets specified app from DB and puts into in memory cache
func (c *ClusterDaoImplCassandra) getApp(appName string) (store.App, error) {
	var id string
	var active bool
	var partitions uint32
	var config string
	var configuration store.Configuration

	query := fmt.Sprintf(KeyAppById, appName)
	if err := c.Session.Query(query).Consistency(c.Conf.ClusterDB.DBConfig.Consistency).Scan(&id, &partitions, &active, &config); err != nil {
		glog.Errorf("Error %s while querying %s", err.Error(), query)
		return store.App{}, err
	}

	if err := json.Unmarshal([]byte(config), &configuration); err != nil {
		glog.Errorf("Error: %s while unmarshalling config: %+v for app: %s", err.Error(), config, id)
		configuration = store.Configuration{}
	}

	return c.cache(
		store.App{
			AppId:         id,
			Partitions:    partitions,
			Active:        active,
			Configuration: configuration,
		}), nil
}

// Checks whether app is found in in-memory cache or not
// Return app and the flag (found/ not found)
func (c *ClusterDaoImplCassandra) check(appName string) (store.App, bool) {
	c.AppMap.lock.RLock()
	app, found := c.AppMap.m[appName]
	c.AppMap.lock.RUnlock()

	return app, found
}

// cache adds the app in in-memory cache
func (c *ClusterDaoImplCassandra) cache(app store.App) store.App {
	c.AppMap.lock.Lock()
	c.AppMap.m[app.AppId] = app
	c.AppMap.lock.Unlock()
	return app
}

// UpdateEntityStatus updates the status of a specific entity in the Cassandra database.
// It also updates the history of the entity status changes.
func (c *ClusterDaoImplCassandra) UpdateEntityStatus(id string, nodename string, status int) error {
	entity := c.GetEntityInfo(id)

	history := ""

	if entity.History != "" || entity.Node != "" {
		toAppend := "{" + entity.Node + ":" + strconv.Itoa(entity.Status) + "}"
		history = entity.History + "->" + toAppend
	}

	historyLen := len(history)
	if historyLen > c.Conf.ClusterDB.EntityHistorySize {
		history = history[historyLen-c.Conf.ClusterDB.EntityHistorySize : historyLen]
	}
	query := fmt.Sprintf(KeyUpdateEntityInfo, nodename, status, history, id)
	glog.Info(query)
	return c.Session.Query(query).Exec()
}

// CreateEntity creates a new entity in the Cassandra database in a disabled state.
func (c *ClusterDaoImplCassandra) CreateEntity(entityInfo e.EntityInfo) error {
	return c.Session.Query(QueryInsertEntity,
		entityInfo.Id,
		c.Conf.Cluster.Address,
		0).Exec()
}

// GetApps retrieves application data from the Cassandra database.
// If an appId is provided, it returns data for the specific app. Otherwise, it returns data for all apps.
func (c *ClusterDaoImplCassandra) GetApps(appId string) ([]store.App, error) {
	var id string
	var active bool
	var config string
	var partitions uint32
	var configuration store.Configuration
	var apps []store.App

	if appId != "" {
		app, err := c.getApp(appId)
		if err != nil {
			return apps, err
		}
		apps = append(apps, app)
		return apps, nil
	}

	iter := c.Session.Query(KeyGelAllApps).Consistency(c.Conf.ClusterDB.DBConfig.Consistency).PageSize(c.Conf.Cluster.PageSize).Iter()
	for iter.Scan(&id, &partitions, &active, &config) {
		configuration = store.Configuration{}
		if err := json.Unmarshal([]byte(config), &configuration); err != nil {
			glog.Errorf("Error: %s while unmarshalling config: %+v for app: %s", err.Error(), config, id)
		}
		apps = append(apps, store.App{AppId: id, Partitions: partitions, Active: active, Configuration: configuration})
	}

	if err := iter.Close(); err != nil {
		return apps, err
	}

	glog.V(constants.INFO).Infof("Get all apps result is : +%v", apps)
	return apps, nil
}

// UpdateAppActiveStatus updates the active status of an application in the Cassandra database.
func (c *ClusterDaoImplCassandra) UpdateAppActiveStatus(appName string, activeStatus bool) error {
	var query string
	if activeStatus {
		query = fmt.Sprintf(QueryUpdateAppStatus, "True", appName)
	} else {
		query = fmt.Sprintf(QueryUpdateAppStatus, "False", appName)
	}

	glog.Info(query)
	return c.Session.Query(query).Exec()
}

// InvalidateSingleAppCache removes a specific app from the AppMap cache.
func (c *ClusterDaoImplCassandra) InvalidateSingleAppCache(appName string) {
	c.AppMap.lock.Lock()
	_, ok := c.AppMap.m[appName]
	if ok {
		delete(c.AppMap.m, appName)
	}
	c.AppMap.lock.Unlock()
}

func (c *ClusterDaoImplCassandra) GetDCAwareApp(appName string) (store.App, error) {
	// check in memory cache for app
	c.AppMap.lock.RLock()
	dcPrefixedApp, dcPrefixedFound := c.AppMap.m[c.Conf.DCConfig.Prefix+appName]
	fallbackApp, fallbackFound := c.AppMap.m[appName]
	c.AppMap.lock.RUnlock()

	if dcPrefixedFound {
		return dcPrefixedApp, nil
	} else if fallbackFound {
		return fallbackApp, nil
	}

	return c.getDCAwareApp(appName)
}

// get DC aware app from DB
func (c *ClusterDaoImplCassandra) getDCAwareApp(appName string) (store.App, error) {
	var appId string
	var partitions uint32
	var active bool
	var config string
	var configuration store.Configuration

	// add dc prefix to app name
	dcPrefixedAppName := c.Conf.DCConfig.Prefix + constants.DCPrefix + appName

	appIdToApp := make(map[string]store.App)
	iter := c.Session.Query(KeyAppByIds, dcPrefixedAppName, appName).Consistency(c.Conf.ClusterDB.DBConfig.Consistency).Iter()

	for iter.Scan(&appId, &partitions, &active, &config) {
		configuration = store.Configuration{}
		if err := json.Unmarshal([]byte(config), &configuration); err != nil {
			glog.Errorf("Error: %s while unmarshalling config: %+v for app: %s", err.Error(), config, appId)
		}
		appIdToApp[appId] = store.App{
			AppId:         appId,
			Partitions:    partitions,
			Active:        active,
			Configuration: configuration,
		}
	}

	if err := iter.Close(); err != nil {
		glog.Infof("Error occurred while fetching apps: %+v", err)
		return store.App{}, err
	}

	if app, ok := appIdToApp[dcPrefixedAppName]; ok {
		c.cache(app)
		return app, nil
	} else if fallbackApp, ok := appIdToApp[appName]; ok {
		c.cache(fallbackApp)
		return fallbackApp, nil
	} else {
		return store.App{}, nil
	}

}

// Create configuration for a given appId and configuration
func (c *ClusterDaoImplCassandra) CreateConfigurations(appId string, configuration store.Configuration) (store.Configuration, error) {
	var err error
	var query string
	var config []byte

	if appId != MaxConfigApp {
		if err = c.ValidateConfigurations(configuration); err != nil {
			return store.Configuration{}, err
		}
	}

	if config, err = json.Marshal(configuration); err != nil {
		return store.Configuration{}, err
	}

	query = fmt.Sprintf(QueryUpdateConfig, string(config), appId)
	glog.Info(query)

	return configuration, c.Session.Query(query).Exec()
}

// Get app configurations for a given appId
func (c *ClusterDaoImplCassandra) GetConfiguration(appId string) (store.Configuration, error) {
	var query string
	var config string
	var configuration store.Configuration

	query = fmt.Sprintf(QueryGetConfig, appId)
	glog.Info(query)

	if err := c.Session.Query(query).Consistency(c.Conf.ClusterDB.DBConfig.Consistency).Scan(&config); err != nil {
		return configuration, err
	}

	if err := json.Unmarshal([]byte(config), &configuration); err != nil {
		return configuration, err
	}

	return configuration, nil
}

// Update the configurations for given appId and configurations
func (c *ClusterDaoImplCassandra) UpdateConfiguration(appId string, configuration store.Configuration) (store.Configuration, error) {
	var err error
	var query string
	var config []byte
	var existingConfig store.Configuration

	if appId != MaxConfigApp {
		if err = c.ValidateConfigurations(configuration); err != nil {
			return store.Configuration{}, err
		}
	}

	if existingConfig, err = c.GetConfiguration(appId); err != nil {
		return store.Configuration{}, err
	}

	if err = mergo.Merge(&configuration, existingConfig); err != nil {
		return store.Configuration{}, err
	}

	if config, err = json.Marshal(configuration); err != nil {
		return configuration, nil
	}

	query = fmt.Sprintf(QueryUpdateConfig, string(config), appId)
	glog.Info(query)

	return configuration, c.Session.Query(query).Exec()
}

// Delete the configurations for a given appId
func (c *ClusterDaoImplCassandra) DeleteConfiguration(appId string) (store.Configuration, error) {
	var query string

	// empty config
	config, _ := json.Marshal(store.Configuration{})

	query = fmt.Sprintf(QueryUpdateConfig, string(config), appId)
	glog.Info(query)

	return store.Configuration{}, c.Session.Query(query).Exec()
}

// App configurations are validated against max configs
func (c *ClusterDaoImplCassandra) ValidateConfigurations(config store.Configuration) error {
	var app store.App
	var err error

	if config == (store.Configuration{}) {
		return nil
	}

	if app, err = c.GetApp(MaxConfigApp); err != nil {
		return err
	}

	if config.PayloadSize > app.Configuration.PayloadSize {
		return errors.New(fmt.Sprintf("provided payload size: %d, max payload size: %d", config.PayloadSize, app.Configuration.PayloadSize))
	} else if config.HttpRetries > app.Configuration.HttpRetries {
		return errors.New(fmt.Sprintf("provided http retries: %d, max http retries: %d", config.HttpRetries, app.Configuration.HttpRetries))
	} else if config.HttpTimeout > app.Configuration.HttpTimeout {
		return errors.New(fmt.Sprintf("provided http timeout: %d, max http timeout: %d", config.HttpTimeout, app.Configuration.HttpTimeout))
	} else if config.FiredScheduleRetentionPeriod > app.Configuration.FiredScheduleRetentionPeriod {
		return errors.New(fmt.Sprintf("provided fired schedule retention period: %d, max fired schedule retention period: %d", config.FiredScheduleRetentionPeriod, app.Configuration.FiredScheduleRetentionPeriod))
	} else if config.FutureScheduleCreationPeriod > app.Configuration.FutureScheduleCreationPeriod {
		return errors.New(fmt.Sprintf("provided schedule retention period: %d, max future schedule creation period: %d", config.FutureScheduleCreationPeriod, app.Configuration.FutureScheduleCreationPeriod))
	}

	return nil
}
