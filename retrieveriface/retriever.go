package retrieveriface

import (
	"github.com/myntra/goscheduler/store"
	"time"
)

type Retriever interface {
	GetSchedules(appName string, partitionID int, timeBucket time.Time) error
	BulkAction(app store.App, partitionId int, timeBucket time.Time, status []store.Status, actionType store.ActionType) error
}
