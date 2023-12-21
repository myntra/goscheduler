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

package retrievers

import (
	"github.com/golang/glog"
	"github.com/myntra/goscheduler/constants"
	"github.com/myntra/goscheduler/dao"
	p "github.com/myntra/goscheduler/monitoring"
	"github.com/myntra/goscheduler/store"
	"runtime/debug"
	"time"
)

const BatchSize = 50

type ScheduleRetriever struct {
	clusterDao  dao.ClusterDao
	scheduleDao dao.ScheduleDao
	monitor     p.Monitor
}

func (s ScheduleRetriever) GetSchedules(appName string, partitionId int, timeBucket time.Time) (err error) {
	defer func() {
		if r := recover(); r != nil {
			glog.Errorf("Recovered in ScheduleRetrieverImplCassandra from error %s with stacktrace %s", r, string(debug.Stack()))
		}
	}()

	app, err := s.clusterDao.GetApp(appName)

	sch := store.Schedule{}
	_map := make(map[string]interface{})
	iter := s.scheduleDao.GetSchedulesForEntity(appName, partitionId, timeBucket, nil)
	for iter.MapScan(_map) {
		if err := sch.CreateScheduleFromCassandraMap(_map); err != nil {
			glog.Infof("Error while forming schedule from cassandra map: %+v, error: %s", _map, err.Error())
			return err
		}

		glog.V(constants.INFO).Infof("Got schedule: %+v, pageState: %+v", sch, iter.PageState())
		sch.Callback.Invoke(store.ScheduleWrapper{Schedule: sch, App: app, IsReconciliation: false})

		_map = make(map[string]interface{})
		sch = store.Schedule{}
	}

	if err = iter.Close(); err != nil {
		glog.Errorf("Error: %s while fetching schedulers for app: %s, poller: %d, timeStamp: %v", err.Error(), appName, partitionId, timeBucket)
		return err
	}

	return nil
}

// Fetches data from DB for a given appId, partitionId, scheduleTimeGroup in paginated way
// Enriches the data with status and makes the reconciliation if required
// Return error if there is any error while querying DB or enriching them with status
func (s ScheduleRetriever) BulkAction(app store.App, partitionId int, scheduleTimeGroup time.Time, status []store.Status, actionType store.ActionType) error {
	defer func() {
		if r := recover(); r != nil {
			glog.Errorf("Recovered in %s from error %+v with stacktrace %s", string(actionType), r, string(debug.Stack()))
		}
	}()

	var pageState []byte = nil
	var batch []store.Schedule
	var err error
	counter := 0
	sch := store.Schedule{}
	_map := make(map[string]interface{})

	iter := s.scheduleDao.GetSchedulesForEntity(app.AppId, partitionId, scheduleTimeGroup, pageState)
	for iter.MapScan(_map) {
		if err := sch.CreateScheduleFromCassandraMap(_map); err != nil {
			glog.Infof("Error while forming schedule from cassandra map: %+v, error: %s", _map, err.Error())
			return err
		}

		glog.V(constants.INFO).Infof("Got schedule: %+v, pageState: %+v", sch, iter.PageState())

		batch = append(batch, sch)
		counter++

		if counter == BatchSize {
			if err := s.actionIfRequired(app, batch, status, actionType); err != nil {
				return err
			}

			counter = 0
			batch = nil
		}
		_map = make(map[string]interface{})
		sch = store.Schedule{}
	}

	if err := s.actionIfRequired(app, batch, status, actionType); err != nil {
		return err
	}

	if err = iter.Close(); err != nil {
		glog.Errorf("Error: %s while making query for app: %s, partitionId: %d, scheduleTimeGroup: %+v",
			err.Error(),
			app.AppId,
			partitionId,
			scheduleTimeGroup)

		return err
	}

	return nil
}

// contains checks if the status mentioned in the schedule matches with
// any one of the status provided
// Returns true if the status matches with existing status values or if status is any other value
func contains(status []store.Status, sch store.Schedule) bool {
	for _, v := range status {
		switch v {
		case store.Success, store.Failure, store.Miss, store.Scheduled:
			if v == sch.Status {
				return true
			}
		default:
			return true
		}
	}
	return false
}

// Makes the callback for schedules based on the status
// Schedules are enriched with status before making a callback
func (s ScheduleRetriever) actionIfRequired(app store.App, schedules []store.Schedule, status []store.Status, actionType store.ActionType) error {
	if len(schedules) == 0 {
		return nil
	}

	enrichedSchedules, err := s.scheduleDao.OptimizedEnrichSchedule(schedules)
	// we already logged the error, so no need to log it
	if err != nil {
		glog.Errorf("Schedule enrichment failed for appId: %s, partitionId: %d with error %s", schedules[0].AppId, schedules[0].PartitionId, err.Error())
		return err
	}

	glog.V(constants.INFO).Infof("Enriched schedules: %+v", enrichedSchedules)

	for _, sch := range enrichedSchedules {
		if contains(status, sch) {
			switch actionType {
			case store.Reconcile:
				_ = sch.Callback.Invoke(store.ScheduleWrapper{Schedule: sch, App: app, IsReconciliation: true})
			case store.Delete:
				_, _ = s.scheduleDao.DeleteSchedule(sch.ScheduleId)
			}

		}
	}

	return nil
}
