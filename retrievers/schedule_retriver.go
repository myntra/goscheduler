package retrievers

import (
	"github.com/golang/glog"
	"github.com/myntra/goscheduler/constants"
	"github.com/myntra/goscheduler/dao"
	p "github.com/myntra/goscheduler/monitoring"
	"github.com/myntra/goscheduler/store"
	"gopkg.in/alexcesaro/statsd.v2"
	"runtime/debug"
	"strconv"
	"time"
)

const BatchSize = 50

type ScheduleRetriever struct {
	clusterDao  dao.ClusterDao
	scheduleDao dao.ScheduleDao
	monitoring  *p.Monitoring
}

// prefix for getBulkStatus update
func (s ScheduleRetriever) getEnrichSchedulePrefix(appId string, partitionId int) string {
	return "scheduleRetriever" + constants.DOT + "enrichSchedule" + constants.DOT + appId + constants.DOT + strconv.Itoa(partitionId)
}

// Record enrich schedule in bulk success
func (s ScheduleRetriever) recordEnrichSchedulesSuccess(prefix string) {
	if s.monitoring != nil && s.monitoring.StatsDClient != nil {
		bucket := prefix + constants.DOT + constants.Success
		s.monitoring.StatsDClient.Increment(bucket)
	}
}

// Record enrich schedule in bulk failure
func (s ScheduleRetriever) recordEnrichScheduleFailure(prefix string) {
	if s.monitoring != nil && s.monitoring.StatsDClient != nil {
		bucket := prefix + constants.DOT + constants.Fail
		s.monitoring.StatsDClient.Increment(bucket)
	}
}

// Record statsD metrics for the execution of do()
// log error messages in case of failures
func (s ScheduleRetriever) recordAndLog(do func() ([]store.Schedule, error), bucket string) ([]store.Schedule, error) {
	var timing statsd.Timing

	if s.monitoring != nil && s.monitoring.StatsDClient != nil {
		timing = s.monitoring.StatsDClient.NewTiming()
	}

	schedules, err := do()

	if s.monitoring != nil && s.monitoring.StatsDClient != nil {
		timing.Send(bucket)
		s.monitoring.StatsDClient.Increment(bucket)
	}

	if err != nil {
		s.recordEnrichScheduleFailure(bucket)
		glog.Errorf("Schedule enrichment failed for bucket: %s with error %s", bucket, err.Error())
	} else {
		s.recordEnrichSchedulesSuccess(bucket)
	}
	return schedules, err
}

func (s ScheduleRetriever) GetSchedules(appName string, partitionId int, timeBucket time.Time) (err error) {
	defer func() {
		if r := recover(); r != nil {
			s.monitoring.StatsDClient.Increment(constants.Panic + constants.DOT + "ScheduleRetrieverImplCassandra")
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
			s.monitoring.StatsDClient.Increment(constants.Panic + constants.DOT + string(actionType))
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

	enrichedSchedules, err := s.recordAndLog(
		func() ([]store.Schedule, error) { return s.scheduleDao.OptimizedEnrichSchedule(schedules) },
		s.getEnrichSchedulePrefix(schedules[0].AppId, schedules[0].PartitionId))

	// we already logged the error, so no need to log it
	if err != nil {
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
