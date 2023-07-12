package dao

import (
	"errors"
	"fmt"
	"github.com/gocql/gocql"
	e "github.com/myntra/goscheduler/cluster_entity"
	"github.com/myntra/goscheduler/store"
)

type DummyClusterDaoImpl struct {
}

func (d DummyClusterDaoImpl) GetAllEntitiesInfoOfNode(nodeName string) []e.EntityInfo {
	return []e.EntityInfo{}
}

func (d DummyClusterDaoImpl) GetAllEntitiesInfo() []e.EntityInfo {
	return []e.EntityInfo{
		{
			Id:      "Tony.0",
			Node:    "",
			Status:  0,
			History: "",
		},
		{
			Id:      "Steve.0",
			Node:    "",
			Status:  0,
			History: "",
		},
		{
			Id:      "Thor.0",
			Node:    "",
			Status:  0,
			History: "",
		},
	}
}

func (d DummyClusterDaoImpl) GetEntityInfo(id string) e.EntityInfo {
	return e.EntityInfo{}
}

func (d DummyClusterDaoImpl) UpdateEntityStatus(id string, nodeName string, status int) error {
	return nil
}

func (d DummyClusterDaoImpl) GetApp(appName string) (store.App, error) {
	switch appName {
	case "testDeactivatedUpdateAppActiveStatus", "testDeactivated":
		return store.App{
			AppId:      appName,
			Partitions: 1,
			Active:     false,
		}, nil
	case "testGetAppError":
		return store.App{}, errors.New(fmt.Sprintf("Error while getting app %s", appName))
	case "testGetAppErrorNotFound":
		return store.App{}, gocql.ErrNotFound
	case "testAppNotFound":
		return store.App{}, nil
	case "testAppNotActive":
		return store.App{Active: false}, nil
	default:
		return store.App{
			AppId:      appName,
			Partitions: 1,
			Active:     true,
		}, nil
	}
}

func (d DummyClusterDaoImpl) InvalidateSingleAppCache(appName string) {
}

func (d DummyClusterDaoImpl) InsertApp(app store.App) error {
	switch app.AppId {
	case "testInsertError":
		return errors.New(fmt.Sprintf("Error while inserting %s", app.AppId))
	default:
		return nil
	}
}

func (d DummyClusterDaoImpl) CreateEntity(info e.EntityInfo) error {
	switch info.GetAppName() {
	case "testCreateEntityError":
		return errors.New(fmt.Sprintf("Error while creating entity %+v", info))
	default:
		return nil
	}
}

func (d DummyClusterDaoImpl) UpdateAppActiveStatus(appName string, activeStatus bool) error {
	switch appName {
	case "testUpdateAppActiveStatusError", "testDeactivatedUpdateAppActiveStatus":
		return errors.New(fmt.Sprintf("Error while updating status for app %s", appName))
	default:
		return nil
	}
}

//func (d DummyClusterDaoImpl) CreateConfigurations(appId string, configuration schedule.Configuration) (schedule.Configuration, error) {
//	switch appId {
//	case "testCreateConfigurationsError":
//		return schedule.Configuration{}, errors.New(fmt.Sprintf("Error creating configurations for app %s", appId))
//	default:
//		return schedule.Configuration{}, nil
//	}
//}
//
//func (d DummyClusterDaoImpl) GetConfiguration(appId string) (schedule.Configuration, error) {
//	switch appId {
//	case "testGetConfigurationError":
//		return schedule.Configuration{}, errors.New(fmt.Sprintf("Error getting configurations for app %s", appId))
//	default:
//		return schedule.Configuration{}, nil
//	}
//}
//
//func (d DummyClusterDaoImpl) UpdateConfiguration(appId string, configuration schedule.Configuration) (schedule.Configuration, error) {
//	switch appId {
//	case "testUpdateConfigurationError":
//		return schedule.Configuration{}, errors.New(fmt.Sprintf("Error updating configurations for app %s", appId))
//	default:
//		return schedule.Configuration{}, nil
//	}
//}
//
//func (d DummyClusterDaoImpl) DeleteConfiguration(appId string) (schedule.Configuration, error) {
//	switch appId {
//	case "testDeleteConfigurationError":
//		return schedule.Configuration{}, errors.New(fmt.Sprintf("Error deleting configurations for app %s", appId))
//	default:
//		return schedule.Configuration{}, nil
//	}
//}

func (d DummyClusterDaoImpl) GetApps(appId string) ([]store.App, error) {
	switch appId {
	case "testGetAppsError":
		return []store.App{}, errors.New(fmt.Sprintf("Error getting apps for app: %s", appId))
	case "testEmptyData":
		return []store.App{}, nil
	case "test":
		return []store.App{
			{
				AppId:      appId,
				Partitions: 1,
				Active:     true,
			},
		}, nil
	default:
		return []store.App{
			{
				AppId:      "test1",
				Partitions: 1,
				Active:     true,
			},
			{
				AppId:      "test2",
				Partitions: 1,
				Active:     false,
			},
		}, nil
	}
}

func (d DummyClusterDaoImpl) GetDCAwareApp(appName string) (store.App, error) {
	return store.App{}, nil
}
