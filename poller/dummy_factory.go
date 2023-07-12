package poller

import (
	"github.com/myntra/goscheduler/cluster_entity"
	"github.com/myntra/goscheduler/constants"
	"github.com/myntra/goscheduler/retrieveriface"
	r "github.com/myntra/goscheduler/retrievers"
	"strconv"
	"strings"
)

type DummyFactory struct {
}

func (d DummyFactory) CreateEntity(pollerId string) cluster_entity.Entity {
	seq := strings.Split(pollerId, constants.PollerKeySep)
	id, _ := strconv.Atoi(seq[1])
	return Dummy{
		AppName:     seq[0],
		PartitionId: id,
	}
}

func (d DummyFactory) GetEntityRetriever(appId string) retrieveriface.Retriever {
	return r.DummyRetriever{}
}
