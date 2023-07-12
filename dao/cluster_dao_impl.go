package dao

import (
	"errors"
	"fmt"
	"github.com/golang/glog"
	"github.com/myntra/goscheduler/cassandra"
	e "github.com/myntra/goscheduler/cluster_entity"
	"github.com/myntra/goscheduler/conf"
	"github.com/myntra/goscheduler/constants"
	"github.com/myntra/goscheduler/db_wrapper"
	"github.com/myntra/goscheduler/store"
	"strconv"
	"sync"
)

type AppMap struct {
	lock sync.RWMutex
	m    map[string]store.App
}

type ClusterDaoImplCassandra struct {
	Session db_wrapper.SessionInterface
	AppMap  AppMap
	//TODO: Can we merge this?
	ClusterConfig   *conf.ClusterConfig
	ClusterDBConfig *conf.ClusterDBConfig
}

var (
	KeyEntityTable = "entity"
	KeyNodeTable   = "nodes"
	KeyAppTable    = "apps"

	KeyEntitiesOfNode    = "SELECT id, status FROM " + KeyNodeTable + " WHERE nodename='%s';"
	KeyGetAllEntities    = "SELECT id, nodename, status, history FROM " + KeyEntityTable + ";"
	KeyGetEntity         = "SELECT id, nodename, status, history FROM " + KeyEntityTable + " WHERE id='%s';"
	KeyUpdateEntityInfo  = "UPDATE " + KeyEntityTable + " SET nodename='%s', status=%d, history='%s' WHERE id='%s';"
	QueryInsertEntity    = "INSERT INTO " + KeyEntityTable + " (id, nodename, status) VALUES (?, ?, ?)"
	QueryInsertApp       = "INSERT INTO " + KeyAppTable + " (id, partitions, active) VALUES (?, ?, ?)"
	KeyAppById           = "SELECT id, partitions, active FROM " + KeyAppTable + " WHERE id='%s';"
	KeyGelAllApps        = "SELECT id, partitions, active FROM " + KeyAppTable + ";"
	QueryUpdateAppStatus = "UPDATE " + KeyAppTable + " set active = %s where id='%s'"
)

// TODO: Should we make it singleton?
func GetClusterDaoImpl(clusterConfig *conf.ClusterConfig, clusterDBConfig *conf.ClusterDBConfig) *ClusterDaoImplCassandra {
	session, err := cassandra.GetSessionInterface(clusterDBConfig.DBConfig, clusterDBConfig.ClusterKeySpace)
	if err != nil {
		err = errors.New(fmt.Sprintf("Cassandra initialisation failed for configuration: %+v with error %s", clusterDBConfig.DBConfig, err.Error()))
		panic(err)
	}
	return &ClusterDaoImplCassandra{
		Session:         session,
		ClusterConfig:   clusterConfig,
		ClusterDBConfig: clusterDBConfig,
		AppMap: AppMap{
			lock: sync.RWMutex{},
			m:    make(map[string]store.App),
		},
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
	iter := c.Session.Query(query).Consistency(c.ClusterDBConfig.DBConfig.Consistency).PageSize(c.ClusterConfig.PageSize).Iter()

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
		Consistency(c.ClusterDBConfig.DBConfig.Consistency).
		PageSize(c.ClusterConfig.PageSize).
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

	if err := c.Session.Query(query).Consistency(c.ClusterDBConfig.DBConfig.Consistency).Scan(&id, &nodeName, &status, &history); err != nil {
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
	return c.Session.Query(QueryInsertApp, app.AppId, app.Partitions, app.Active).Exec()
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

	query := fmt.Sprintf(KeyAppById, appName)
	if err := c.Session.Query(query).Consistency(c.ClusterDBConfig.DBConfig.Consistency).Scan(&id, &partitions, &active); err != nil {
		glog.Errorf("Error %s while querying %s", err.Error(), query)
		return store.App{}, err
	}

	return c.cache(
		store.App{
			AppId:      id,
			Partitions: partitions,
			Active:     active}), nil
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
	if historyLen > c.ClusterDBConfig.EntityHistorySize {
		history = history[historyLen-c.ClusterDBConfig.EntityHistorySize : historyLen]
	}
	query := fmt.Sprintf(KeyUpdateEntityInfo, nodename, status, history, id)
	glog.Info(query)
	return c.Session.Query(query).Exec()
}

// CreateEntity creates a new entity in the Cassandra database in a disabled state.
func (c *ClusterDaoImplCassandra) CreateEntity(entityInfo e.EntityInfo) error {
	return c.Session.Query(QueryInsertEntity,
		entityInfo.Id,
		c.ClusterConfig.Address,
		0).Exec()
}

// GetApps retrieves application data from the Cassandra database.
// If an appId is provided, it returns data for the specific app. Otherwise, it returns data for all apps.
func (c *ClusterDaoImplCassandra) GetApps(appId string) ([]store.App, error) {
	var id string
	var active bool
	var partitions uint32
	var apps []store.App

	if appId != "" {
		app, err := c.getApp(appId)
		if err != nil {
			return apps, err
		}
		apps = append(apps, app)
		return apps, nil
	}

	iter := c.Session.Query(KeyGelAllApps).Consistency(c.ClusterDBConfig.DBConfig.Consistency).PageSize(c.ClusterConfig.PageSize).Iter()
	for iter.Scan(&id, &partitions, &active) {
		apps = append(apps, store.App{AppId: id, Partitions: partitions, Active: active})
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
