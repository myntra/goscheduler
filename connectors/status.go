package connectors

import (
	"fmt"
	"github.com/golang/glog"
	"github.com/myntra/goscheduler/constants"
	s "github.com/myntra/goscheduler/store"
	"gopkg.in/alexcesaro/statsd.v2"
	"strconv"
)

//prefix for status update
func statusUpdatePrefix(appId string, partitionId int, size int) string {
	return "connectors" + constants.DOT + "status" + constants.DOT + appId + constants.DOT +
		strconv.Itoa(partitionId) + constants.DOT + strconv.Itoa(size)
}

// record status update success
func (c *Connector) recordStatusUpdateSuccess(prefix string) {
	if c.Monitoring != nil && c.Monitoring.StatsDClient != nil {
		bucket := prefix + constants.DOT + constants.Success
		c.Monitoring.StatsDClient.Increment(bucket)
	}
}

// record status update failure
func (c *Connector) recordStatusUpdateFailure(prefix string) {
	if c.Monitoring != nil && c.Monitoring.StatsDClient != nil {
		bucket := prefix + constants.DOT + constants.Fail
		c.Monitoring.StatsDClient.Increment(bucket)
	}
}

// record statsD metrics for the execution of do()
// log error messages in case of failures
func (c *Connector) recordAndLog(do func() error, bucket string) {
	var timing statsd.Timing
	if c.Monitoring != nil && c.Monitoring.StatsDClient != nil {
		timing = c.Monitoring.StatsDClient.NewTiming()
	}

	err := do()

	if c.Monitoring != nil && c.Monitoring.StatsDClient != nil {
		timing.Send(bucket)
		c.Monitoring.StatsDClient.Increment(bucket)
	}

	if err != nil {
		c.recordStatusUpdateFailure(bucket)
		glog.Errorf("status update failed with error %s", err.Error())
	} else {
		c.recordStatusUpdateSuccess(bucket)
	}
}

// update status in bulk
func (c *Connector) updateStatus(buf <-chan s.StatusTask) {
	for statusTask := range buf {
		batch := statusTask.Schedules
		if len(batch) > 0 {
			c.recordAndLog(
				func() error { return c.ScheduleDao.UpdateStatus(batch) },
				statusUpdatePrefix(batch[0].AppId, batch[0].PartitionId, len(batch)))
		}
	}
}

// create status update work pool
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
