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
	e "github.com/myntra/goscheduler/cluster_entity"
	"github.com/myntra/goscheduler/store"
)

type ClusterDao interface {
	GetAllEntitiesInfoOfNode(nodeName string) []e.EntityInfo
	GetAllEntitiesInfo() []e.EntityInfo
	GetAllEntitiesForApp(appId string) []e.EntityInfo
	GetEntityInfo(id string) e.EntityInfo
	UpdateEntityStatus(id string, nodeName string, status int) error
	GetApp(appName string) (store.App, error)
	InvalidateSingleAppCache(appName string)
	InsertApp(app store.App) error
	CreateEntity(info e.EntityInfo) error
	UpdateAppActiveStatus(appName string, activeStatus bool) error
	GetApps(appId string) ([]store.App, error)
	GetDCAwareApp(appName string) (store.App, error)
	CreateConfigurations(appId string, configuration store.Configuration) (store.Configuration, error)
	GetConfiguration(appId string) (store.Configuration, error)
	UpdateConfiguration(appId string, configuration store.Configuration) (store.Configuration, error)
	DeleteConfiguration(appId string) (store.Configuration, error)
}
