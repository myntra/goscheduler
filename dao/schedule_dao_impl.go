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

package dao

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gocql/gocql"
	"github.com/golang/glog"
	"github.com/myntra/goscheduler/cassandra"
	"github.com/myntra/goscheduler/conf"
	"github.com/myntra/goscheduler/constants"
	"github.com/myntra/goscheduler/db_wrapper"
	p "github.com/myntra/goscheduler/monitoring"
	"github.com/myntra/goscheduler/store"
	"gopkg.in/alexcesaro/statsd.v2"
	"runtime/debug"
	"strconv"
	"time"
)

const BatchSize = 50

type ScheduleDaoImpl struct {
	Session    db_wrapper.SessionInterface
	Monitoring *p.Monitoring
	//TODO: Can we merge this?
	ClusterConfig            *conf.ClusterConfig
	AggregateSchedulesConfig *conf.AggregateSchedulesConfig
}

// Profile execution of do
func (s *ScheduleDaoImpl) profile(do func() (store.Schedule, error), bucket string) (store.Schedule, error) {
	var timing statsd.Timing
	if s.Monitoring != nil && s.Monitoring.StatsDClient != nil {
		timing = s.Monitoring.StatsDClient.NewTiming()
	}

	result, err := do()

	if s.Monitoring != nil && s.Monitoring.StatsDClient != nil {
		timing.Send(bucket)
		s.Monitoring.StatsDClient.Increment(bucket)
	}

	return result, err
}

// prefix for getBulkStatus update
func (s *ScheduleDaoImpl) getEnrichSchedulePrefix(appId string, partitionId int) string {
	return "scheduleRetriever" + constants.DOT + "enrichSchedule" + constants.DOT + appId + constants.DOT + strconv.Itoa(partitionId)
}

// Record enrich schedule in bulk success
func (s *ScheduleDaoImpl) recordEnrichSchedulesSuccess(prefix string) {
	if s.Monitoring != nil && s.Monitoring.StatsDClient != nil {
		bucket := prefix + constants.DOT + constants.Success
		s.Monitoring.StatsDClient.Increment(bucket)
	}
}

// Record enrich schedule in bulk failure
func (s *ScheduleDaoImpl) recordEnrichScheduleFailure(prefix string) {
	if s.Monitoring != nil && s.Monitoring.StatsDClient != nil {
		bucket := prefix + constants.DOT + constants.Fail
		s.Monitoring.StatsDClient.Increment(bucket)
	}
}

// Record statsD metrics for the execution of do()
// log error messages in case of failures
func (s *ScheduleDaoImpl) recordAndLog(do func() ([]store.Schedule, error), bucket string) ([]store.Schedule, error) {
	var timing statsd.Timing

	if s.Monitoring != nil && s.Monitoring.StatsDClient != nil {
		timing = s.Monitoring.StatsDClient.NewTiming()
	}

	schedules, err := do()

	if s.Monitoring != nil && s.Monitoring.StatsDClient != nil {
		timing.Send(bucket)
		s.Monitoring.StatsDClient.Increment(bucket)
	}

	if err != nil {
		s.recordEnrichScheduleFailure(bucket)
		glog.Errorf("Schedule enrichment failed for bucket: %s with error %s", bucket, err.Error())
	} else {
		s.recordEnrichSchedulesSuccess(bucket)
	}
	return schedules, err
}

type Range struct {
	//start time of the schedules for filter
	StartTime time.Time
	//end time of the schedules for filter
	EndTime time.Time
}

func GetScheduleDaoImpl(clusterDConfig *conf.ClusterConfig, scheduleDBConfig *conf.ScheduleDBConfig, aggregateSchedulesConfig *conf.AggregateSchedulesConfig, monitoring *p.Monitoring) *ScheduleDaoImpl {
	session, err := cassandra.GetSessionInterface(scheduleDBConfig.DBConfig, scheduleDBConfig.ScheduleKeySpace)
	if err != nil {
		err = errors.New(fmt.Sprintf("Cassandra initialisation failed for configuration: %+v with error %s", scheduleDBConfig.DBConfig, err.Error()))
		panic(err)
	}
	return &ScheduleDaoImpl{
		Session:                  session,
		ClusterConfig:            clusterDConfig,
		AggregateSchedulesConfig: aggregateSchedulesConfig,
		Monitoring:               monitoring,
	}
}

// Persist a cron schedule in Cassandra.
// The data is denormalized across two different tables.
// The schedules are created with status as Scheduled.
// Throws error in writing data to the schedule fails.
func (s *ScheduleDaoImpl) createRecurringSchedule(schedule store.Schedule) (store.Schedule, error) {
	batch := gocql.NewBatch(gocql.LoggedBatch)

	for _, query := range []string{
		"INSERT INTO recurring_schedules_by_id (" +
			"app_id," +
			"partition_id," +
			"schedule_id," +
			"payload," +
			"callback_type," +
			"callback_details," +
			"cron_expression, " +
			"status) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",

		"INSERT INTO recurring_schedules_by_partition (" +
			"app_id," +
			"partition_id," +
			"schedule_id," +
			"payload," +
			"callback_type," +
			"callback_details," +
			"cron_expression, " +
			"status) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
	} {
		batch.Query(
			query,
			schedule.AppId,
			schedule.PartitionId,
			schedule.ScheduleId,
			schedule.Payload,
			schedule.GetCallBackType(),
			schedule.GetCallbackDetails(),
			schedule.CronExpression,
			store.Scheduled)
	}

	err := s.Session.ExecuteBatch(batch)

	schedule.Status = store.Scheduled
	return schedule, err
}

// Persist a one time schedule in Cassandra.
// Throws error if writing data to schedule fails.
func (s *ScheduleDaoImpl) createOneTimeSchedule(schedule store.Schedule) (store.Schedule, error) {
	query := "INSERT INTO schedules (" +
		"app_id," +
		"partition_id," +
		"schedule_time_group," +
		"schedule_id," +
		"schedule_time," +
		"payload," +
		"callback_type," +
		"callback_details) VALUES (?, ?, ?, ?, ?, ?, ?, ?)"

	err := s.Session.Query(
		query,
		schedule.AppId,
		schedule.PartitionId,
		schedule.ScheduleGroup*constants.SecondsToMillis,
		schedule.ScheduleId,
		schedule.ScheduleTime*constants.SecondsToMillis,
		schedule.Payload,
		schedule.GetCallBackType(),
		schedule.GetCallbackDetails()).Exec()

	return schedule, err
}

// Persist the schedule details in cassandra.
// The tables to which the schedule is written to is determined based on it being a recurring schedule or not.
// Throws error if the writing to the schedule fails.
func (s *ScheduleDaoImpl) CreateSchedule(schedule store.Schedule) (store.Schedule, error) {
	if schedule.IsRecurring() {
		bucket := constants.CassandraInsert + constants.DOT + constants.CreateRecurringSchedule
		return s.profile(func() (store.Schedule, error) {
			return s.createRecurringSchedule(schedule)
		}, bucket)
	} else {
		bucket := constants.CassandraInsert + constants.DOT + constants.CreateSchedule
		return s.profile(func() (store.Schedule, error) {
			return s.createOneTimeSchedule(schedule)
		}, bucket)
	}
}

// Get all recurring schedules with partition id
// Returns a list of schedules and non nill error in case fetching the details fail.
func (s *ScheduleDaoImpl) GetRecurringScheduleByPartition(partitionId int) ([]store.Schedule, []error) {
	query := "SELECT " +
		"schedule_id," +
		"payload," +
		"callback_type," +
		"callback_details," +
		"app_id," +
		"partition_id, " +
		"cron_expression, " +
		"status " +
		"FROM recurring_schedules_by_partition " +
		"WHERE partition_id = ?"

	var schedules []store.Schedule
	var errs []error

	_map := make(map[string]interface{})
	iter := s.Session.Query(query, partitionId).
		RetryPolicy(&gocql.SimpleRetryPolicy{NumRetries: s.ClusterConfig.NumRetry}).
		Iter()

	for iter.MapScan(_map) {
		var schedule store.Schedule
		if err := schedule.CreateScheduleFromCassandraMap(_map); err != nil {
			errs = append(errs, err)
		} else {
			schedules = append(schedules, schedule)
		}

		_map = make(map[string]interface{})
	}

	if err := iter.Close(); err != nil {
		errs = append(errs, err)
	}

	return schedules, errs
}

// Find a recurring schedule with the supplied id.
// Returns a non nil error if fetching the details failed or if no row with the id is found.
func (s *ScheduleDaoImpl) getRecurringSchedule(uuid gocql.UUID) (store.Schedule, error) {
	query := "SELECT " +
		"schedule_id," +
		"payload," +
		"callback_type," +
		"callback_details," +
		"app_id," +
		"partition_id, " +
		"cron_expression, " +
		"status " +
		"FROM recurring_schedules_by_id " +
		"WHERE schedule_id= ? LIMIT 1"

	_map := make(map[string]interface{})
	err := s.Session.Query(query, uuid).
		RetryPolicy(&gocql.SimpleRetryPolicy{NumRetries: s.ClusterConfig.NumRetry}).
		MapScan(_map)
	if err != nil {
		return store.Schedule{}, err
	}

	var schedule store.Schedule
	err = schedule.CreateScheduleFromCassandraMap(_map)
	return schedule, err
}

// Find a one time schedule with the supplied id.
// Returns a non nil error if fetching the details failed or if no row with the id is found.
func (s *ScheduleDaoImpl) getOneTimeSchedule(uuid gocql.UUID) (store.Schedule, error) {
	query := "SELECT " +
		"schedule_id," +
		"payload," +
		"schedule_time_group," +
		"schedule_time," +
		"callback_type," +
		"callback_details," +
		"app_id," +
		"partition_id " +
		"FROM view_schedules " +
		"WHERE schedule_id= ? LIMIT 1"

	_map := make(map[string]interface{})
	err := s.Session.Query(query, uuid).
		RetryPolicy(&gocql.SimpleRetryPolicy{NumRetries: s.ClusterConfig.NumRetry}).
		MapScan(_map)
	if err != nil {
		return store.Schedule{}, err
	}

	var schedule store.Schedule
	err = schedule.CreateScheduleFromCassandraMap(_map)
	return schedule, err
}

// Get a single schedule with status data enriched
// returns a non nil error if setting of status fails
func (s *ScheduleDaoImpl) getEnrichedSchedule(uuid gocql.UUID) (store.Schedule, error) {
	schedule, err := s.getOneTimeSchedule(uuid)
	if err != nil {
		return schedule, err
	}

	if err = s.setStatus(&schedule); err != nil {
		return schedule, err
	}

	return schedule, nil
}

// Get a single schedule with the supplied id.
// Attempts to find a one time schedule and falls back to recurring schedules if not found.
// Returns a non nil error if fetching the details of the schedule fails.
func (s *ScheduleDaoImpl) GetSchedule(uuid gocql.UUID) (store.Schedule, error) {
	schedule, err := s.getOneTimeSchedule(uuid)
	if err == gocql.ErrNotFound {
		return s.getRecurringSchedule(uuid)
	}

	return schedule, err
}

// Get a single schedule enriched with status data
// Returns a non nil error if fetching the details of the schedule fails.
func (s *ScheduleDaoImpl) GetEnrichedSchedule(uuid gocql.UUID) (store.Schedule, error) {
	schedule, err := s.getEnrichedSchedule(uuid)
	if err == gocql.ErrNotFound {
		return s.getRecurringSchedule(uuid)
	}

	return schedule, err
}

// Enrich a non-recurring schedule with status data
// Returns a non nil error if enriching the schedule fails.
func (s *ScheduleDaoImpl) EnrichSchedule(schedule *store.Schedule) error {
	var err error

	if err = s.setStatus(schedule); err != nil {
		return err
	}
	return nil
}

// Delete a schedule from the recurring schedule tables.
// These are soft deletes marked with status as Deleted
// All one time future runs generated from this schedule will also be deleted.
// Returns a non nil error in case deleting the rows fails.
func (s *ScheduleDaoImpl) deleteRecurringSchedule(schedule store.Schedule) (store.Schedule, error) {
	batch := gocql.NewBatch(gocql.LoggedBatch)

	deleteById := "UPDATE recurring_schedules_by_id " +
		"SET status = ? " +
		"WHERE schedule_id = ?"
	batch.Query(deleteById, store.Deleted, schedule.ScheduleId)

	deleteByPartition := "UPDATE recurring_schedules_by_partition " +
		"SET status = ? " +
		"WHERE partition_id = ? " +
		"AND schedule_id = ? " +
		"AND app_id = ?"
	batch.Query(deleteByPartition, store.Deleted, schedule.PartitionId, schedule.ScheduleId, schedule.AppId)

	runs, _, err := s.getFutureRuns(schedule.ScheduleId, -1, nil)
	if err != nil {
		return schedule, err
	}

	for _, run := range runs {
		batch.Query(
			deleteFromSchedule,
			run.AppId,
			run.PartitionId,
			run.ScheduleGroup*constants.SecondsToMillis,
			run.ScheduleId)
	}

	err = s.Session.ExecuteBatch(batch)
	schedule.Status = store.Deleted

	return schedule, err
}

const deleteFromSchedule string = "DELETE from schedules " +
	"WHERE app_id = ? " +
	"AND partition_id = ? " +
	"AND schedule_time_group = ? " +
	"AND schedule_id = ?"

// Deletes a given schedule from the schedule table.
// The rows are removed from the Cassandra.
// Return a non nil error in case the delete fails.
func (s *ScheduleDaoImpl) deleteOneTimeSchedule(schedule store.Schedule) (store.Schedule, error) {
	err := s.Session.Query(
		deleteFromSchedule,
		schedule.AppId,
		schedule.PartitionId,
		schedule.ScheduleGroup*constants.SecondsToMillis,
		schedule.ScheduleId).Exec()

	return schedule, err
}

// Delete schedule with the given id
// For one time schedules, the rows are removed from the schedule table
// where as for the recurring schedules, the status is marked accordingly.
// Returns a non nil error in case deleting the row from Cassandra fails.
func (s *ScheduleDaoImpl) DeleteSchedule(uuid gocql.UUID) (store.Schedule, error) {
	switch schedule, err := s.GetSchedule(uuid); {
	case err != nil:
		return store.Schedule{}, err
	case schedule.IsRecurring():
		return s.deleteRecurringSchedule(schedule)
	default:
		return s.deleteOneTimeSchedule(schedule)
	}
}

// Get runs belonging to a parent schedule id.
// The page state restores the fetching from the last known partition.
// At max size number or rows are fetched.
// Returns an iterator to the rows fetched.
func (s *ScheduleDaoImpl) getRuns(uuid gocql.UUID, pageState []byte, size int64) db_wrapper.IterInterface {
	query := "SELECT app_id, " +
		"partition_id, " +
		"schedule_time_group, " +
		"schedule_id, " +
		"callback_type, " +
		"callback_details, " +
		"payload, " +
		"schedule_time " +
		"FROM recurring_schedule_runs " +
		"WHERE parent_schedule_id = ? "

	return s.Session.Query(query, uuid).
		PageState(pageState).
		PageSize(int(size)).
		RetryPolicy(&gocql.SimpleRetryPolicy{NumRetries: s.ClusterConfig.NumRetry}).
		Iter()
}

type scheduleWriter struct {
	// List of schedules fetched.
	schedules *[]store.Schedule

	// Number of filtered schedules written to the writer from the last fetched page.
	lastPageState *int

	// Error in fetching the details from Cassandra
	err *error

	// Function determines when to stop further writes to this writer.
	terminate func() bool

	// Determines whether the provided schedule can be written to the writer or not.
	filter func(schedule store.Schedule) bool

	// Page state that can be used to resume fetching schedules
	pageState *[]byte

	// Resume start time
	continuationStartTime *time.Time
}

type paginatedScheduleWriter struct {
	// List of schedules fetched.
	schedules *[]store.Schedule

	// Number of filtered schedules written to the writer from the last fetched page.
	lastPageState *int

	//pageState used to resume fetching schedules
	pageState *[]byte

	// Error in fetching the details from Cassandra
	err *error

	// Function determines when to stop further writes to this writer.
	terminate func() bool

	// Determines whether the provided schedule can be written to the writer or not.
	filter func(schedule store.Schedule) bool
}

// Fetch runs corresponding to the parent uuid and write them to the supplied witter.
// This function iteratively fetches rows from cassandra by pages. It continues fetching until there are no more rows to
// fetch from source, or the writer is terminated.
// For every row that is fetched, the item is written to the writer only if is successfully filtered by the filter function.
// Any error occurred while fetching these rows will be written to the error field in the writer.
// The number of items that were successfully filtered and written to the writer from the last page is set to the
// last page state filed in writer.
func (s *ScheduleDaoImpl) getFilteredRuns(uuid gocql.UUID, writer paginatedScheduleWriter, size int64) {
	_map := make(map[string]interface{})
	pageState := *writer.pageState
	glog.V(constants.INFO).Infof("Initial pageState %+v", pageState)
	for {
		schedulesLeft := int64(int(size) - len(*writer.schedules))
		iter := s.getRuns(uuid, pageState, schedulesLeft)
		glog.V(constants.INFO).Infof("uuid: %+v, PageState: %+v, schedulesLeft: %d", uuid, iter.PageState(), schedulesLeft)
		var items int
		for !writer.terminate() && iter.MapScan(_map) {
			var schedule store.Schedule
			if err := schedule.CreateScheduleFromCassandraMap(_map); err != nil {
				writer.err = &err
				return
			}

			glog.V(constants.INFO).Infof("Got schedule here %s", schedule.ScheduleId.String())

			//enrich schedule with status data
			if err := s.setStatus(&schedule); err != nil {
				//log and suppress
				//We will not stop the process due to failed enrichment
				glog.Errorf("Error occurred while enriching with status %s", err.Error())
			}

			if writer.filter(schedule) {
				*writer.schedules = append(*writer.schedules, schedule)
				items++
			}
			_map = make(map[string]interface{})
		}
		*writer.lastPageState = items
		*writer.pageState = iter.PageState()

		if err := iter.Close(); err != nil {
			writer.err = &err
			return
		}

		pageState = iter.PageState()
		if len(pageState) == 0 || writer.terminate() {
			glog.V(constants.INFO).Infof("Reached end %+v", iter.PageState())
			return
		}
	}
}

// Get size number of runs for a given schedule id.
func (s *ScheduleDaoImpl) getAllRuns(uuid gocql.UUID, size int64, pageState []byte) ([]store.Schedule, []byte, error) {
	var schedules []store.Schedule
	var lastPageState int
	var err error

	writer := paginatedScheduleWriter{
		schedules:     &schedules,
		lastPageState: &lastPageState,
		pageState:     &pageState,
		err:           &err,
		terminate: func() bool {
			return len(schedules) == int(size)
		},
		filter: func(schedule store.Schedule) bool {
			return true
		},
	}

	s.getFilteredRuns(uuid, writer, size)
	return *writer.schedules, *writer.pageState, *writer.err
}

// Find size number of past runs of a given schedule id.
// The result list is sorted in reverse order of schedule time.
// The functions assumes that at most window number of schedules might have
// created by the poller in the future.
func (s *ScheduleDaoImpl) getPastRuns(uuid gocql.UUID, size int64, pageState []byte) ([]store.Schedule, []byte, error) {
	var schedules []store.Schedule
	var lastPageState int
	var err error
	now := time.Now()

	writer := paginatedScheduleWriter{
		schedules:     &schedules,
		lastPageState: &lastPageState,
		pageState:     &pageState,
		err:           &err,
		terminate: func() bool {
			return len(schedules) == int(size)
		},
		filter: func(schedule store.Schedule) bool {
			return time.Unix(schedule.ScheduleGroup, 0).Before(now)
		},
	}

	s.getFilteredRuns(uuid, writer, size)
	return *writer.schedules, *writer.pageState, *writer.err
}

// Find size number of runs for a given schedule id scheduled at a future time.
// The result list is sorted from immediate schedule to further in future.
// The functions assumes that at most window number of schedules might have
// created by the poller in the future.
func (s *ScheduleDaoImpl) getFutureRuns(uuid gocql.UUID, size int64, pageState []byte) ([]store.Schedule, []byte, error) {
	var schedules []store.Schedule
	lastPageState := -1
	var err error
	now := time.Now()

	writer := paginatedScheduleWriter{
		schedules:     &schedules,
		lastPageState: &lastPageState,
		pageState:     &pageState,
		err:           &err,
		terminate: func() bool {
			return len(schedules) == int(size)
		},
		filter: func(schedule store.Schedule) bool {
			return time.Unix(schedule.ScheduleGroup, 0).After(now)
		},
	}

	s.getFilteredRuns(uuid, writer, size)

	var output []store.Schedule
	for i := 0; i < len(*writer.schedules) && len(output) != int(size); i++ {
		output = append(output, (*writer.schedules)[len(*writer.schedules)-i-1])
	}

	return output, *writer.pageState, *writer.err
}

// Get size number of top runs for a given schedule id.
func (s *ScheduleDaoImpl) GetScheduleRuns(uuid gocql.UUID, size int64, when string, pageState []byte) ([]store.Schedule, []byte, error) {
	switch when {
	case "past":
		return s.getPastRuns(uuid, size, pageState)
	case "future":
		return s.getFutureRuns(uuid, size, pageState)
	default:
		return s.getAllRuns(uuid, size, pageState)
	}
}

// Create a one time schedule for a recurring schedule.
// The schedule will be persisted in schedule and runs tables.
// Returns a non nil error in case persisting the data fails.
func (s *ScheduleDaoImpl) CreateRun(schedule store.Schedule) (store.Schedule, error) {

	batch := gocql.NewBatch(gocql.LoggedBatch)

	for _, query := range []string{"INSERT INTO schedules (" +
		"app_id," +
		"partition_id," +
		"schedule_time_group," +
		"schedule_id," +
		"schedule_time," +
		"payload," +
		"callback_type," +
		"callback_details," +
		"parent_schedule_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",

		"INSERT INTO recurring_schedule_runs (" +
			"app_id," +
			"partition_id," +
			"schedule_time_group," +
			"schedule_id," +
			"schedule_time," +
			"payload," +
			"callback_type," +
			"callback_details," +
			"parent_schedule_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
	} {
		batch.
			RetryPolicy(&gocql.SimpleRetryPolicy{NumRetries: s.ClusterConfig.NumRetry}).
			Query(
				query,
				schedule.AppId,
				schedule.PartitionId,
				schedule.ScheduleGroup*constants.SecondsToMillis,
				schedule.ScheduleId,
				schedule.ScheduleTime*constants.SecondsToMillis,
				schedule.Payload,
				schedule.GetCallBackType(),
				schedule.GetCallbackDetails(),
				schedule.ParentScheduleId)
	}

	return schedule, s.Session.ExecuteBatch(batch)
}

// updates status in batches
// set ttl same as buffer ttl as data is added to this table after callback is fired
func (s *ScheduleDaoImpl) UpdateStatus(schedules []store.Schedule) error {
	//log schedule status update
	for _, schedule := range schedules {
		glog.Infof("update status for schedule: %s", schedule.ScheduleId.String())
	}

	const insertStatusQuery string = "INSERT INTO status (" +
		"app_id," +
		"partition_id," +
		"schedule_time_group," +
		"schedule_id," +
		"schedule_status," +
		"error_msg," +
		"reconciliation_history) VALUES (?, ?, ?, ?, ?, ?, ?)"

	batch := gocql.NewBatch(gocql.UnloggedBatch)

	for _, query := range schedules {
		reconciliationHistory, _ := json.Marshal(query.ReconciliationHistory)

		batch.
			RetryPolicy(&gocql.SimpleRetryPolicy{NumRetries: s.ClusterConfig.NumRetry}).
			Query(
				insertStatusQuery,
				query.AppId,
				query.PartitionId,
				query.ScheduleGroup*constants.SecondsToMillis,
				query.ScheduleId,
				query.Status,
				query.ErrorMessage,
				reconciliationHistory)
	}

	return s.Session.ExecuteBatch(batch)
}

//get schedules filtered by status

// get paginated schedules by status
func (s *ScheduleDaoImpl) getPaginatedSchedulesByStatus(appId string, partitions int, timeRange Range, size int64, status store.Status, pageState []byte, continuationStartTime time.Time) ([]store.Schedule, []byte, time.Time, error) {
	var schedules []store.Schedule = nil
	var lastPageState int
	var err error
	var filter func(schedule store.Schedule) bool

	if status == "" {
		filter = func(schedule store.Schedule) bool { return true }
	} else {
		filter = func(schedule store.Schedule) bool { return schedule.Status == status }
	}

	writer := scheduleWriter{
		schedules:     &schedules,
		lastPageState: &lastPageState,
		pageState:     &pageState,
		err:           &err,
		terminate: func() bool {
			return len(schedules) == int(size)
		},
		filter:                filter,
		continuationStartTime: &continuationStartTime,
	}

	s.getFilteredSchedules(appId, partitions, timeRange, size, writer)
	return *writer.schedules, *writer.pageState, *writer.continuationStartTime, *writer.err
}

func (s *ScheduleDaoImpl) GetPaginatedSchedules(appId string, partitions int, timeRange Range, size int64, status store.Status, pageState []byte, continuationStartTime time.Time) ([]store.Schedule, []byte, time.Time, error) {
	switch status {
	case store.Success, store.Failure, store.Miss, store.Scheduled:
		return s.getPaginatedSchedulesByStatus(appId, partitions, timeRange, size, status, pageState, continuationStartTime)
	default:
		return s.getPaginatedSchedulesByStatus(appId, partitions, timeRange, size, "", pageState, continuationStartTime)
	}
}

// get filtered schedules based on appId, partitions and time range [startTime, endTime)
func (s *ScheduleDaoImpl) getFilteredSchedules(appId string, partitions int, timeRange Range, size int64, writer scheduleWriter) {
	_map := make(map[string]interface{})
	pageState := *writer.pageState

	var partitionList []int
	for i := 0; i < partitions; i++ {
		partitionList = append(partitionList, i)
	}

	startTime := timeRange.StartTime
	endTime := timeRange.EndTime

	glog.V(constants.INFO).Infof("continuationStartTime: %+v", writer.continuationStartTime)

	// override the start time if resume start time is present
	if writer.continuationStartTime.Unix() != 0 {
		startTime = *writer.continuationStartTime
	}

	// finds the next endTime given the start time
	next := func(startTime time.Time) time.Time {
		if endTime.Sub(startTime) < time.Hour*1 {
			return endTime
		}
		return startTime.Add(time.Hour * 1)
	}

	//set initial interval
	var interval = Range{StartTime: startTime, EndTime: next(startTime)}

	for {
		glog.V(constants.INFO).Infof("startTime: %+v, endTime: %+v", interval.StartTime, interval.EndTime)
		glog.V(constants.INFO).Infof("pageState: %+v", pageState)
		//return if the start time equal end time
		if interval.StartTime == interval.EndTime {
			return
		}

		// required schedules gives the schedules which are required at any given time during pagination
		requiredSchedules := int64(int(size) - len(*writer.schedules))

		iter := s.getSchedules(appId, partitionList, interval, pageState, requiredSchedules)

		var items int
		for !writer.terminate() && iter.MapScan(_map) {
			var schedule store.Schedule
			if err := schedule.CreateScheduleFromCassandraMap(_map); err != nil {
				writer.err = &err
				return
			}

			glog.V(constants.INFO).Infof("Found scheduleId: %s", schedule.ScheduleId)
			//enrich schedule with status data
			if err := s.setStatus(&schedule); err != nil {
				//log and suppress
				//We will not stop the process due to failed enrichment
				glog.Errorf("Error occurred while enriching with status %s", err.Error())
			}

			if writer.filter(schedule) {
				glog.V(constants.INFO).Infof("Accepted scheduleId: %s", schedule.ScheduleId)
				*writer.schedules = append(*writer.schedules, schedule)
				items++
			}
			_map = make(map[string]interface{})
		}
		*writer.lastPageState = items
		*writer.pageState = iter.PageState()
		*writer.continuationStartTime = interval.StartTime

		if err := iter.Close(); err != nil {
			writer.err = &err
			return
		}

		if writer.terminate() {
			return
		}

		pageState = iter.PageState()
		if len(pageState) == 0 {
			glog.V(constants.INFO).Info("Reached end of page")
			newStartTime := interval.EndTime

			//set start and end of interval
			interval.StartTime = newStartTime
			interval.EndTime = next(newStartTime)
		}
	}
}

// fetch schedules from the db based on appId, time range in paginated way
func (s *ScheduleDaoImpl) getSchedules(appId string, partitions []int, timeRange Range, pageState []byte, size int64) db_wrapper.IterInterface {

	timeStamps := func(startTime time.Time, endTime time.Time) []int64 {
		var timeStamps []int64
		for _time := startTime; _time.Before(endTime); _time = _time.Add(time.Minute * 1) {
			timeStamps = append(timeStamps, _time.Unix()*constants.SecondsToMillis)
		}
		return timeStamps
	}

	query := "SELECT " +
		"schedule_id," +
		"payload," +
		"schedule_time_group," +
		"schedule_time," +
		"callback_type," +
		"callback_details," +
		"app_id," +
		"partition_id " +
		"FROM schedules " +
		"WHERE app_id = ? " +
		"AND partition_id IN ? " +
		"AND schedule_time_group IN ?"

	iter := s.Session.Query(
		query,
		appId,
		partitions,
		timeStamps(timeRange.StartTime, timeRange.EndTime)).
		PageState(pageState).
		PageSize(int(size)).
		RetryPolicy(&gocql.SimpleRetryPolicy{NumRetries: s.ClusterConfig.NumRetry}).
		Iter()

	return iter
}

// fetch schedules from the db based on appId, time range in paginated way
func (s *ScheduleDaoImpl) GetSchedulesForEntity(appId string, partitionId int, timeBucket time.Time, pageState []byte) db_wrapper.IterInterface {
	query := "SELECT " +
		"app_id," +
		"partition_id," +
		"schedule_time_group," +
		"schedule_id," +
		"callback_type," +
		"callback_details," +
		"payload," +
		"schedule_time," +
		"parent_schedule_id " +
		"FROM schedules " +
		"WHERE app_id = ? " +
		"AND partition_id = ? " +
		"AND schedule_time_group = ?"

	iter := s.Session.Query(
		query,
		appId,
		partitionId,
		timeBucket).
		PageState(pageState).
		PageSize(s.ClusterConfig.PageSize).
		RetryPolicy(&gocql.SimpleRetryPolicy{NumRetries: s.ClusterConfig.NumRetry}).
		Iter()

	return iter
}

// fetch schedule status and error from status table
// case 1) schedule is not found in status table
//
//	sub-case 1) current time is already past the schedule time group of the schedule with difference greater than 1 minute plus flush period
//	         -> set the status as ERROR with default message
//	sub-case 2) current time is yet to to come
//	         -> set status as SCHEDULED
//
// case 2) schedule is found in status table
//
//	-> set status of the schedule from the fetched row
//
// return error if there is any other error while fetching data from db
func (s *ScheduleDaoImpl) setStatus(schedule *store.Schedule) error {
	query := "SELECT " +
		"schedule_status," +
		"error_msg," +
		"reconciliation_history " +
		"FROM status " +
		"WHERE app_id= ? " +
		"AND partition_id= ? " +
		"AND schedule_id= ? LIMIT 1"

	_map := make(map[string]interface{})

	err := s.Session.Query(
		query,
		schedule.AppId,
		schedule.PartitionId,
		schedule.ScheduleId).
		RetryPolicy(&gocql.SimpleRetryPolicy{NumRetries: s.ClusterConfig.NumRetry}).
		MapScan(_map)

	if err != nil {
		if err == gocql.ErrNotFound {
			schedule.SetUnknownStatus(s.AggregateSchedulesConfig.FlushPeriod)
			return nil
		}
		return err
	}

	if err = schedule.SetStatus(_map); err != nil {
		return err
	}

	return nil
}

// Fetch status data in bulk by appId, partitionId, scheduleIds
func (s *ScheduleDaoImpl) getBulkStatus(schedules []store.Schedule) db_wrapper.IterInterface {
	var uuids []gocql.UUID

	if len(schedules) == 0 {
		return nil
	}

	for _, schedule := range schedules {
		uuids = append(uuids, schedule.ScheduleId)
	}

	query := "SELECT " +
		"schedule_id," +
		"schedule_status," +
		"error_msg," +
		"reconciliation_history " +
		"FROM status " +
		"WHERE app_id= ? " +
		"AND partition_id= ? " +
		"AND schedule_id IN ?"

	iter := s.Session.Query(
		query,
		schedules[0].AppId,
		schedules[0].PartitionId,
		uuids).
		RetryPolicy(&gocql.SimpleRetryPolicy{NumRetries: s.ClusterConfig.NumRetry}).
		Iter()

	return iter
}

// Fetch status data in bulk by appId, partitionId, scheduleIds
// Set status for schedules present in Status table
// Set status for schedules non present in status table
// Return error if there is any error while fetching data
func (s *ScheduleDaoImpl) OptimizedEnrichSchedule(schedules []store.Schedule) ([]store.Schedule, error) {
	var enrichedSchedules []store.Schedule
	idToSchedule := make(map[gocql.UUID]store.Schedule)
	found := make(map[gocql.UUID]bool)
	_map := make(map[string]interface{})

	if len(schedules) == 0 {
		return enrichedSchedules, nil
	}

	for _, schedule := range schedules {
		idToSchedule[schedule.ScheduleId] = schedule
	}

	glog.Infof("idToSchedule: %+v", idToSchedule)

	// Set status of schedules which are present in status table
	iter := s.getBulkStatus(schedules)
	for iter.MapScan(_map) {
		uuid := _map["schedule_id"].(gocql.UUID)
		schedule := idToSchedule[uuid]
		found[uuid] = true

		if err := schedule.SetStatus(_map); err != nil {
			return enrichedSchedules, err
		}

		glog.V(constants.INFO).Infof("Enriched Schedule: %+v", schedule)

		enrichedSchedules = append(enrichedSchedules, schedule)
		_map = make(map[string]interface{})
	}
	if err := iter.Close(); err != nil {
		glog.Errorf("Error: %s while calling status query for schedules: %+v", err.Error(), schedules)
		return enrichedSchedules, err
	}

	// Set status of schedules which are not present in status table
	for uuid, schedule := range idToSchedule {
		if _, ok := found[uuid]; !ok {
			schedule.SetUnknownStatus(s.AggregateSchedulesConfig.FlushPeriod)

			glog.V(constants.INFO).Infof("Enriched Schedule: %+v", schedule)
			enrichedSchedules = append(enrichedSchedules, schedule)
		}
	}

	return enrichedSchedules, nil
}

func (s *ScheduleDaoImpl) GetCronSchedulesByApp(appId string, status store.Status) ([]store.Schedule, []string) {
	query := "SELECT " +
		"schedule_id," +
		"payload," +
		"callback_type," +
		"callback_details," +
		"app_id," +
		"partition_id, " +
		"cron_expression, " +
		"status " +
		"FROM recurring_schedules_by_id"

	var schedules []store.Schedule
	var errs []string

	_map := make(map[string]interface{})
	iter := s.Session.Query(query).
		RetryPolicy(&gocql.SimpleRetryPolicy{NumRetries: s.ClusterConfig.NumRetry}).
		Iter()

	for iter.MapScan(_map) {
		var schedule store.Schedule
		if err := schedule.CreateScheduleFromCassandraMap(_map); err != nil {
			errs = append(errs, err.Error())
		} else if schedule.AppId == appId || appId == "" {
			if schedule.Status == status || (status != store.Scheduled && status != store.Deleted) {
				schedules = append(schedules, schedule)
			}
		}
		_map = make(map[string]interface{})
	}

	if err := iter.Close(); err != nil {
		errs = append(errs, err.Error())
	}

	return schedules, errs
}

func (s *ScheduleDaoImpl) BulkAction(app store.App, partitionId int, scheduleTimeGroup time.Time, status []store.Status, actionType store.ActionType) error {
	defer func() {
		if r := recover(); r != nil {
			s.Monitoring.StatsDClient.Increment(constants.Panic + constants.DOT + string(actionType))
			glog.Errorf("Recovered in %s from error %+v with stacktrace %s", string(actionType), r, string(debug.Stack()))
		}
	}()

	var pageState []byte = nil
	var batch []store.Schedule
	var err error
	counter := 0
	_sch := store.Schedule{}
	_map := make(map[string]interface{})

	iter := s.GetSchedulesForEntity(app.AppId, partitionId, scheduleTimeGroup, pageState)
	for iter.MapScan(_map) {
		if err := _sch.CreateScheduleFromCassandraMap(_map); err != nil {
			glog.Infof("Error while forming schedule from cassandra map: %+v, error: %s", _map, err.Error())
			return err
		}

		glog.V(constants.INFO).Infof("Got schedule: %+v, pageState: %+v", _sch, iter.PageState())

		batch = append(batch, _sch)
		counter++

		if counter == BatchSize {
			if err := s.actionIfRequired(app, batch, status, actionType); err != nil {
				return err
			}

			counter = 0
			batch = nil
		}
		_map = make(map[string]interface{})
		_sch = store.Schedule{}
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
func contains(status []store.Status, _sch store.Schedule) bool {
	for _, v := range status {
		switch v {
		case store.Success, store.Failure, store.Miss, store.Scheduled:
			if v == _sch.Status {
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
func (s *ScheduleDaoImpl) actionIfRequired(app store.App, schedules []store.Schedule, status []store.Status, actionType store.ActionType) error {
	if len(schedules) == 0 {
		return nil
	}

	enrichedSchedules, err := s.recordAndLog(
		func() ([]store.Schedule, error) { return s.OptimizedEnrichSchedule(schedules) },
		s.getEnrichSchedulePrefix(schedules[0].AppId, schedules[0].PartitionId))

	// we already logged the error, so no need to log it
	if err != nil {
		return err
	}

	glog.V(constants.INFO).Infof("Enriched schedules: %+v", enrichedSchedules)

	for _, _sch := range enrichedSchedules {
		if contains(status, _sch) {
			switch actionType {
			case store.Reconcile:
				wrapper := store.ScheduleWrapper{Schedule: _sch, App: app, IsReconciliation: true}
				err := _sch.Callback.Invoke(wrapper)
				if err != nil {
					return err
				}
			case store.Delete:
				_, _ = s.DeleteSchedule(_sch.ScheduleId)
			}

		}
	}

	return nil
}
