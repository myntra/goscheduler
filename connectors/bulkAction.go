package connectors

import (
	"fmt"
	"github.com/golang/glog"
	"github.com/myntra/goscheduler/store"
)

// bulkAction processes BulkActionTasks from the provided channel and performs
// the specified action on the schedule.
func (c *Connector) bulkAction(buf chan store.BulkActionTask) {
	for q := range buf {
		glog.Infof("consumed %+v", q)
		_ = c.ScheduleDao.BulkAction(q.App, q.PartitionId, q.ScheduleTimeGroup, []store.Status{q.Status}, q.ActionType)
	}
}

// createBulkActionPool initializes a pool of workers for processing
// BulkActionTasks.
func (c *Connector) createBulkActionPool(buf chan store.BulkActionTask) {
	noOfWorkers := c.Config.BulkActionConfig.Routines
	for i := 0; i < noOfWorkers; i++ {
		fmt.Printf("\nInitializing worker for BulkAction connector %d", i)
		go c.bulkAction(buf)
	}
}

// initBulkActionWorkers starts a pool of workers for processing
// BulkActionTasks by calling createBulkActionPool with the task.BulkActionQueue.
func (c *Connector) initBulkActionWorkers() {
	go c.createBulkActionPool(store.BulkActionQueue)
}
