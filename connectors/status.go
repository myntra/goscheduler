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
	s "github.com/myntra/goscheduler/store"
)

// update status in bulk
func (c *Connector) updateStatus(buf <-chan s.StatusTask) {
	for statusTask := range buf {
		batch := statusTask.Schedules
		if len(batch) > 0 {
			err := c.ScheduleDao.UpdateStatus(batch, statusTask.App)
			if err != nil {
				glog.Errorf("status update failed for appId: %s, partitionId: %d with error %s", batch[0].AppId, batch[0].PartitionId, err.Error())
			}
		}
	}
}

// CreateStatusUpdatePool create status update work pool
func (c *Connector) CreateStatusUpdatePool(buf chan s.StatusTask) {
	noOfWorkers := c.Config.StatusUpdateConfig.Routines
	for i := 0; i < noOfWorkers; i++ {
		fmt.Printf("Initializing status update workers %d \n", i)
		go c.updateStatus(buf)
	}
}

func (c *Connector) initStatusUpdatePool() {
	go c.CreateStatusUpdatePool(s.StatusTaskQueue)
}
