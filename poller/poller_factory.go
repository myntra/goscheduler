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
