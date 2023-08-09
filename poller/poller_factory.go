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

package poller

import (
	"errors"
	"fmt"
	"github.com/golang/glog"
	"github.com/myntra/goscheduler/cluster_entity"
	"github.com/myntra/goscheduler/conf"
	"github.com/myntra/goscheduler/constants"
	p "github.com/myntra/goscheduler/monitoring"
	riface "github.com/myntra/goscheduler/retrieveriface"
	r "github.com/myntra/goscheduler/retrievers"
	"strconv"
	"strings"
)

type PollerFactory struct {
	Retrievers r.Retrievers
	Config     conf.PollerConfig
	Monitoring *p.Monitoring
}

func (p PollerFactory) CreateEntity(pollerId string) cluster_entity.Entity {
	glog.Infof(" *** Starting poller for entity  ***  %s ", pollerId)
	seq := strings.Split(pollerId, constants.PollerKeySep)
	appName := seq[0]
	partitionId := seq[1]

	scheduleRetrievalImpl := p.GetEntityRetriever(appName)
	glog.Infof("Got schedule retriever %v for app %s", scheduleRetrievalImpl, appName)

	id, err := strconv.Atoi(partitionId)
	if err != nil {
		panic(errors.New(fmt.Sprintf("Cannot create poller for %s", pollerId)))

	}
	return &Poller{
		AppName:               appName,
		PartitionId:           id,
		scheduleRetrievalImpl: scheduleRetrievalImpl,
		config:                p.Config,
		monitoring:            p.Monitoring,
	}
}

func (p PollerFactory) GetEntityRetriever(appName string) riface.Retriever {
	return p.Retrievers.Get(appName)
}

func NewPollerFactory(retriever r.Retrievers, config conf.PollerConfig, monitoring *p.Monitoring) PollerFactory {
	return PollerFactory{
		Retrievers: retriever,
		Config:     config,
		Monitoring: monitoring,
	}
}
