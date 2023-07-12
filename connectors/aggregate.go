package connectors

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/myntra/goscheduler/constants"
	"github.com/myntra/goscheduler/store"
)

// prefix for aggregate channel
func channelPrefix() string {
	return "connectors" + constants.DOT + "aggregate"
}

// record aggregate channel length
func (c *Connector) recordChannelLength(length int) {
	if c.Monitoring != nil && c.Monitoring.StatsDClient != nil {
		bucket := channelPrefix() + constants.DOT + "length" + constants.DOT + strconv.Itoa(length)
		c.Monitoring.StatsDClient.Increment(bucket)
	}
}

func (c *Connector) recordAndLogIfRequired(length int) {
	if length > 0 {
		c.recordChannelLength(length)
		glog.Infof("Aggregate channel length recorded %d", length)
	}
}

// aggregate schedules based on appId, partitionId
// forward messages to status update channel once batch is full
// schedules are flushed to db periodically as well
func (c *Connector) aggregateSchedules(buf <-chan store.ScheduleWrapper) {
	schedules := make(map[string]map[int][]store.Schedule)
	var lock sync.RWMutex

	go func() {
		var app store.App
		var reuse bool

		for range time.Tick(time.Duration(c.Config.AggregateSchedulesConfig.FlushPeriod) * time.Second) {

			lock.Lock()

			for appId := range schedules {
				reuse = false
				for partitionId := range schedules[appId] {
					if len(schedules[appId][partitionId]) > 0 {
						if !reuse {
							app, _ = c.ClusterDao.GetApp(appId)
							reuse = true
						}

						store.StatusTaskQueue <- store.StatusTask{
							Schedules: schedules[appId][partitionId],
							App:       app,
						}
					}
				}
			}
			//flush the contents of the map
			if len(schedules) > 0 {
				schedules = map[string]map[int][]store.Schedule{}
			}

			lock.Unlock()
		}
	}()

	for sw := range buf {
		result := sw.Schedule
		app := sw.App

		appId := result.AppId
		partitionId := result.PartitionId

		lock.Lock()

		if schedules[appId] == nil {
			schedules[appId] = map[int][]store.Schedule{}
		}
		if schedules[appId][partitionId] == nil {
			schedules[appId][partitionId] = []store.Schedule{}
		}

		schedules[appId][partitionId] = append(schedules[appId][partitionId], result)

		if len(schedules[appId][partitionId]) == c.Config.AggregateSchedulesConfig.BatchSize {
			//forward to status update channel for batch update
			store.StatusTaskQueue <- store.StatusTask{
				Schedules: schedules[appId][partitionId],
				App:       app,
			}

			//delete the key from map
			delete(schedules[appId], partitionId)
		}

		lock.Unlock()
	}
}

// create status update work pool
func (c *Connector) CreateAggregateSchedulesPool(buf chan store.ScheduleWrapper) {
	noOfWorkers := c.Config.AggregateSchedulesConfig.Routines
	for i := 0; i < noOfWorkers; i++ {
		fmt.Printf("Initializing aggregation workers %d \n", i)
		go c.aggregateSchedules(buf)
	}
}

func (c *Connector) initAggregateWorkers() {
	go c.CreateAggregateSchedulesPool(store.AggregationTaskQueue)
}
