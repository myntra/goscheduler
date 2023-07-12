package retrievers

import (
	"github.com/myntra/goscheduler/store"
	"time"
)

type DummyRetriever struct {
}

func (d DummyRetriever) GetSchedules(appName string, partitionId int, timeBucket time.Time) error {
	return nil
}

func (d DummyRetriever) BulkAction(app store.App, partitionId int, scheduleTimeGroup time.Time, status []store.Status, actionType store.ActionType) error {
	return nil
}
