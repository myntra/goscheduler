package dao

import (
	e "github.com/myntra/goscheduler/cluster_entity"
	"github.com/myntra/goscheduler/store"
)

type ClusterDao interface {
	GetAllEntitiesInfoOfNode(nodeName string) []e.EntityInfo
	GetAllEntitiesInfo() []e.EntityInfo
	GetEntityInfo(id string) e.EntityInfo
	UpdateEntityStatus(id string, nodeName string, status int) error
	GetApp(appName string) (store.App, error)
	InvalidateSingleAppCache(appName string)
	InsertApp(app store.App) error
	CreateEntity(info e.EntityInfo) error
	UpdateAppActiveStatus(appName string, activeStatus bool) error
	GetApps(appId string) ([]store.App, error)
}
