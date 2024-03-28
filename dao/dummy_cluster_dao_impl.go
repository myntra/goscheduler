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
			AppId:         appName,
			Partitions:    1,
			Active:        false,
			Configuration: store.Configuration{},
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
			AppId:         appName,
			Partitions:    1,
			Active:        true,
			Configuration: store.Configuration{FutureScheduleCreationPeriod: 1000},
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

func (d DummyClusterDaoImpl) CreateConfigurations(appId string, configuration store.Configuration) (store.Configuration, error) {
	switch appId {
	case "testCreateConfigurationsError":
		return store.Configuration{}, errors.New(fmt.Sprintf("Error creating configurations for app %s", appId))
	default:
		return store.Configuration{}, nil
	}
}

func (d DummyClusterDaoImpl) GetConfiguration(appId string) (store.Configuration, error) {
	switch appId {
	case "testGetConfigurationError":
		return store.Configuration{}, errors.New(fmt.Sprintf("Error getting configurations for app %s", appId))
	default:
		return store.Configuration{}, nil
	}
}

func (d DummyClusterDaoImpl) UpdateConfiguration(appId string, configuration store.Configuration) (store.Configuration, error) {
	switch appId {
	case "testUpdateConfigurationError":
		return store.Configuration{}, errors.New(fmt.Sprintf("Error updating configurations for app %s", appId))
	default:
		return store.Configuration{}, nil
	}
}

func (d DummyClusterDaoImpl) DeleteConfiguration(appId string) (store.Configuration, error) {
	switch appId {
	case "testDeleteConfigurationError":
		return store.Configuration{}, errors.New(fmt.Sprintf("Error deleting configurations for app %s", appId))
	default:
		return store.Configuration{}, nil
	}
}

func (d DummyClusterDaoImpl) GetApps(appId string) ([]store.App, error) {
	switch appId {
	case "testGetAppsError":
		return []store.App{}, errors.New(fmt.Sprintf("Error getting apps for app: %s", appId))
	case "testEmptyData":
		return []store.App{}, nil
	case "test":
		return []store.App{
			{
				AppId:         appId,
				Partitions:    1,
				Active:        true,
				Configuration: store.Configuration{},
			},
		}, nil
	default:
		return []store.App{
			{
				AppId:         "test1",
				Partitions:    1,
				Active:        true,
				Configuration: store.Configuration{},
			},
			{
				AppId:         "test2",
				Partitions:    1,
				Active:        false,
				Configuration: store.Configuration{},
			},
		}, nil
	}
}

func (d DummyClusterDaoImpl) GetDCAwareApp(appName string) (store.App, error) {
	return store.App{}, nil
}
