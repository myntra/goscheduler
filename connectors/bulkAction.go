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
