package dao

import (
	"encoding/json"
	"github.com/gocql/gocql"
	"github.com/golang/mock/gomock"
	"github.com/myntra/goscheduler/conf"
	"github.com/myntra/goscheduler/constants"
	"github.com/myntra/goscheduler/mocks"
	s "github.com/myntra/goscheduler/store"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func setupMocks(t *testing.T) (*ScheduleDaoImpl, *mocks.MockSessionInterface, *mocks.MockQueryInterface, *mocks.MockIterInterface, *gomock.Controller) {
	s.Registry[constants.DefaultCallback] = func() s.Callback { return &s.HttpCallback{} }
	dao := &ScheduleDaoImpl{
		ClusterConfig: &conf.ClusterConfig{
			PageSize: 10,
			NumRetry: 2,
		},
		AggregateSchedulesConfig: &conf.AggregateSchedulesConfig{
			FlushPeriod: 60,
		},
	}
	ctrl := gomock.NewController(t)
	m := mocks.NewMockSessionInterface(ctrl)
	mq := mocks.NewMockQueryInterface(ctrl)
	mItr := mocks.NewMockIterInterface(ctrl)

	dao.Session = m

	return dao, m, mq, mItr, ctrl
}

func TestScheduleDaoImpl_CreateSchedule(t *testing.T) {
	dao, m, mq, _, ctrl := setupMocks(t)
	defer ctrl.Finish()

	m.EXPECT().ExecuteBatch(gomock.Any()).Return(nil).AnyTimes()
	m.EXPECT().Query(gomock.Any(), gomock.Any()).Return(mq).AnyTimes()
	mq.EXPECT().Exec().Return(nil).AnyTimes()
	mq.EXPECT().Consistency(gomock.Any()).Return(mq).AnyTimes()

	callbackDetails := s.Details{
		Url:     "http://example.com/callback",
		Method:  "POST",
		Headers: map[string]string{"Content-Type": "application/json"},
	}

	callback := &s.HttpCallback{
		Type:    "http",
		Details: callbackDetails,
	}

	callbackRaw, _ := json.Marshal(*callback)

	schedule := s.Schedule{
		ScheduleId:     gocql.TimeUUID(),
		Payload:        "Test Payload",
		ScheduleTime:   time.Now().Unix(),
		CronExpression: "* * * * *",
		CallbackRaw:    callbackRaw,
		Callback:       callback,
	}

	// Test for recurring schedule
	createdSchedule, err := dao.CreateSchedule(schedule)
	assert.NoError(t, err)
	assert.Equal(t, schedule.Payload, createdSchedule.Payload)
	assert.True(t, createdSchedule.IsRecurring())

	// Test for one-time schedule
	schedule.CronExpression = ""
	createdSchedule, err = dao.CreateSchedule(schedule)
	assert.NoError(t, err)
	assert.Equal(t, schedule.Payload, createdSchedule.Payload)
	assert.False(t, createdSchedule.IsRecurring())
}

func TestScheduleDaoImpl_GetRecurringScheduleByPartition(t *testing.T) {
	dao, m, mq, mItr, ctrl := setupMocks(t)
	defer ctrl.Finish()

	m.EXPECT().Query(gomock.Any(), gomock.Any()).Return(mq).AnyTimes()
	mq.EXPECT().RetryPolicy(gomock.Any()).Return(mq).AnyTimes()
	mq.EXPECT().Iter().Return(mItr).AnyTimes()
	mItr.EXPECT().MapScan(gomock.Any()).Return(false).AnyTimes()
	mItr.EXPECT().Close().Return(nil).AnyTimes()
	mItr.EXPECT().Scan(gomock.All()).Return(false).AnyTimes()

	partitionId := 1
	schedules, errs := dao.GetRecurringScheduleByPartition(partitionId)

	if len(errs) != 0 {
		t.Errorf("Expected no errors, got %v", errs)
	}
	for _, schedule := range schedules {
		if schedule.PartitionId != partitionId {
			t.Errorf("Expected schedule's partitionId to be %d, got %d", partitionId, schedule.PartitionId)
		}
	}
}

func TestScheduleDaoImpl_GetSchedule(t *testing.T) {
	dao, m, mq, _, ctrl := setupMocks(t)
	defer ctrl.Finish()

	m.EXPECT().Query(gomock.Any(), gomock.Any()).Return(mq).AnyTimes()
	mq.EXPECT().RetryPolicy(gomock.Any()).Return(mq).AnyTimes()
	mq.EXPECT().MapScan(gomock.Any()).Return(nil).Times(1)

	_, err := dao.GetSchedule(gocql.TimeUUID())
	if err != nil {
		t.Errorf("Expected no error, got %s", err)
	}

	mq.EXPECT().MapScan(gomock.Any()).Return(gocql.ErrNotFound).AnyTimes()

	_, err = dao.GetSchedule(gocql.TimeUUID())
	if err == nil {
		t.Errorf("Expected no error, got %s", err)
	}
}

func TestScheduleDaoImpl_GetEnrichedSchedule(t *testing.T) {
	dao, m, mq, _, ctrl := setupMocks(t)
	defer ctrl.Finish()

	m.EXPECT().Query(gomock.Any(), gomock.Any()).Return(mq).AnyTimes()
	mq.EXPECT().RetryPolicy(gomock.Any()).Return(mq).AnyTimes()
	mq.EXPECT().MapScan(gomock.Any()).Return(nil).Times(2)

	_, err := dao.GetEnrichedSchedule(gocql.TimeUUID())
	if err != nil {
		t.Errorf("Expected no error, got %s", err)
	}

	mq.EXPECT().MapScan(gomock.Any()).Return(gocql.ErrNotFound).AnyTimes()

	_, err = dao.GetEnrichedSchedule(gocql.TimeUUID())
	if err == nil {
		t.Errorf("Expected no error, got %s", err)
	}
}

func TestScheduleDaoImpl_EnrichSchedule(t *testing.T) {
	dao, m, mq, _, ctrl := setupMocks(t)
	defer ctrl.Finish()

	m.EXPECT().Query(gomock.Any(), gomock.Any()).Return(mq).AnyTimes()
	mq.EXPECT().RetryPolicy(gomock.Any()).Return(mq).AnyTimes()
	mq.EXPECT().MapScan(gomock.Any()).Return(nil).Times(1)

	schedule := &s.Schedule{
		ScheduleId:     gocql.TimeUUID(),
		Payload:        "Test Payload",
		ScheduleTime:   time.Now().Unix(),
		ScheduleGroup:  60 * (time.Now().Unix() / 60),
		CronExpression: "* * * * *",
	}

	err := dao.EnrichSchedule(schedule)
	if err != nil {
		t.Errorf("Expected no error, got %s", err)
	}

	mq.EXPECT().MapScan(gomock.Any()).Return(gocql.ErrNotFound).AnyTimes()

	err = dao.EnrichSchedule(schedule)
	if schedule.Status == "" {
		t.Errorf("Expected no error, got %s", err)
	}
}

func TestScheduleDaoImpl_DeleteSchedule(t *testing.T) {
	dao, m, mq, _, ctrl := setupMocks(t)
	defer ctrl.Finish()

	m.EXPECT().Query(gomock.Any(), gomock.Any()).Return(mq).AnyTimes()
	mq.EXPECT().Exec().Return(nil).AnyTimes()
	mq.EXPECT().RetryPolicy(gomock.Any()).Return(mq).AnyTimes()
	mq.EXPECT().MapScan(gomock.Any()).Return(nil).Times(1)

	_, err := dao.DeleteSchedule(gocql.TimeUUID())
	if err != nil {
		t.Errorf("Expected no error, got %s", err)
	}
}

func TestScheduleDaoImpl_GetScheduleRuns(t *testing.T) {
	dao, m, mq, mItr, ctrl := setupMocks(t)
	defer ctrl.Finish()

	m.EXPECT().Query(gomock.Any(), gomock.Any()).Return(mq).AnyTimes()
	mq.EXPECT().RetryPolicy(gomock.Any()).Return(mq).AnyTimes()
	mq.EXPECT().Consistency(gomock.Any()).Return(mq).AnyTimes()
	mq.EXPECT().PageState(gomock.Any()).Return(mq).AnyTimes()
	mq.EXPECT().PageSize(gomock.Any()).Return(mq).AnyTimes()
	mq.EXPECT().Iter().Return(mItr).AnyTimes()
	mItr.EXPECT().PageState().Return(nil).AnyTimes()
	mItr.EXPECT().MapScan(gomock.Any()).Return(false).AnyTimes()
	mItr.EXPECT().Close().Return(nil).AnyTimes()
	mItr.EXPECT().Scan(gomock.All()).Return(false).AnyTimes()

	_, _, err := dao.GetScheduleRuns(gocql.TimeUUID(), 10, "past", nil)
	if err != nil {
		t.Errorf("Expected no error, got %s", err)
	}

	_, _, err = dao.GetScheduleRuns(gocql.TimeUUID(), 10, "future", nil)
	if err != nil {
		t.Errorf("Expected no error, got %s", err)
	}

	_, _, err = dao.GetScheduleRuns(gocql.TimeUUID(), 10, "", nil)
	if err != nil {
		t.Errorf("Expected no error, got %s", err)
	}
}

func TestScheduleDaoImpl_UpdateStatus(t *testing.T) {
	dao, m, _, _, ctrl := setupMocks(t)
	defer ctrl.Finish()

	schedules := []s.Schedule{
		{
			AppId:                 "testApp",
			PartitionId:           1,
			ScheduleGroup:         10,
			ScheduleId:            gocql.TimeUUID(),
			Status:                "Success",
			ErrorMessage:          "",
			ReconciliationHistory: []s.ReconciliationHistory{},
		},
		{
			AppId:                 "testApp",
			PartitionId:           2,
			ScheduleGroup:         20,
			ScheduleId:            gocql.TimeUUID(),
			Status:                "Failed",
			ErrorMessage:          "Error message",
			ReconciliationHistory: []s.ReconciliationHistory{},
		},
	}

	m.EXPECT().ExecuteBatch(gomock.Any()).Return(nil).AnyTimes()

	err := dao.UpdateStatus(schedules)
	if err != nil {
		t.Errorf("Expected no error, got %s", err)
	}
}

func TestScheduleDaoImpl_GetPaginatedSchedules(t *testing.T) {
	dao, m, mq, mItr, ctrl := setupMocks(t)
	defer ctrl.Finish()

	appId := "testApp"
	partitions := 2
	timeRange := Range{StartTime: time.Now().Add(-24 * time.Hour), EndTime: time.Now()}
	size := int64(10)
	status := s.Success
	pageState := []byte{}
	continuationStartTime := time.Now().Add(-12 * time.Hour)

	m.EXPECT().ExecuteBatch(gomock.Any()).Return(nil).AnyTimes()
	m.EXPECT().Query(gomock.Any(), gomock.Any()).Return(mq).AnyTimes()
	mq.EXPECT().RetryPolicy(gomock.Any()).Return(mq).AnyTimes()
	mq.EXPECT().Consistency(gomock.Any()).Return(mq).AnyTimes()
	mq.EXPECT().PageState(gomock.Any()).Return(mq).AnyTimes()
	mq.EXPECT().PageSize(gomock.Any()).Return(mq).AnyTimes()
	mq.EXPECT().Iter().Return(mItr).AnyTimes()
	mItr.EXPECT().PageState().Return(nil).AnyTimes()
	mItr.EXPECT().MapScan(gomock.Any()).Return(false).AnyTimes()
	mItr.EXPECT().Close().Return(nil).AnyTimes()
	mItr.EXPECT().Scan(gomock.All()).Return(false).AnyTimes()

	schedules, _, nextContinuationStartTime, err := dao.GetPaginatedSchedules(appId, partitions, timeRange, size, status, pageState, continuationStartTime)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if nextContinuationStartTime.IsZero() {
		t.Errorf("Expected non-zero next continuation start time")
	}

	for _, schedule := range schedules {
		if schedule.Status != s.Success {
			t.Errorf("Expected all schedules to have status Success, got %s", schedule.Status)
		}
	}

}

func TestScheduleDaoImpl_GetSchedulesForEntity(t *testing.T) {
	dao, m, mq, mItr, ctrl := setupMocks(t)
	defer ctrl.Finish()

	appId := "test-app"
	partitionId := 1
	timeBucket := time.Now()
	pageState := []byte("initial-page-state")

	m.EXPECT().Query(gomock.Any(), gomock.Any()).Return(mq).AnyTimes()
	mq.EXPECT().RetryPolicy(gomock.Any()).Return(mq).AnyTimes()
	mq.EXPECT().Iter().Return(mItr).AnyTimes()
	mq.EXPECT().PageState(gomock.Any()).Return(mq).AnyTimes()
	mq.EXPECT().PageSize(gomock.Any()).Return(mq).AnyTimes()
	mItr.EXPECT().MapScan(gomock.Any()).Return(false).AnyTimes()
	mItr.EXPECT().Close().Return(nil).AnyTimes()
	mItr.EXPECT().Scan(gomock.All()).Return(false).AnyTimes()

	result := dao.GetSchedulesForEntity(appId, partitionId, timeBucket, pageState)

	// Assert that the result is the expected mock IterInterface
	if result == nil {
		t.Error("Expected non-nil iter")
	}
}

func TestScheduleDaoImpl_OptimizedEnrichSchedule(t *testing.T) {
	dao, m, mq, mItr, ctrl := setupMocks(t)
	defer ctrl.Finish()

	m.EXPECT().Query(gomock.Any(), gomock.Any()).Return(mq).AnyTimes()
	mq.EXPECT().RetryPolicy(gomock.Any()).Return(mq).AnyTimes()
	mq.EXPECT().Iter().Return(mItr).AnyTimes()
	mq.EXPECT().PageState(gomock.Any()).Return(mq).AnyTimes()
	mq.EXPECT().PageSize(gomock.Any()).Return(mq).AnyTimes()
	mItr.EXPECT().MapScan(gomock.Any()).Return(false).AnyTimes()
	mItr.EXPECT().Close().Return(nil).AnyTimes()
	mItr.EXPECT().Scan(gomock.All()).Return(false).AnyTimes()

	// Create a slice of schedules
	schedules := []s.Schedule{
		{
			ScheduleId:   gocql.UUID{},
			Payload:      "payload1",
			AppId:        "app1",
			PartitionId:  1,
			ScheduleTime: time.Now().Unix(),
		},
	}

	// Call the OptimizedEnrichSchedule function
	enrichedSchedules, err := dao.OptimizedEnrichSchedule(schedules)
	if err != nil {
		t.Fatalf("Error while calling OptimizedEnrichSchedule: %s", err.Error())
	}

	// Assert that the schedules have the correct status
	if enrichedSchedules[0].Status != "MISS" {
		t.Errorf("Expected schedule 0 to have status SUCCESS, got %s", enrichedSchedules[0].Status)
	}

}

func TestScheduleDaoImpl_GetCronSchedulesByApp(t *testing.T) {
	dao, m, mq, mItr, ctrl := setupMocks(t)
	defer ctrl.Finish()

	m.EXPECT().Query(gomock.Any(), gomock.Any()).Return(mq).Times(1)
	mq.EXPECT().RetryPolicy(gomock.Any()).Return(mq).Times(1)
	mq.EXPECT().Iter().Return(mItr).Times(1)
	mItr.EXPECT().MapScan(gomock.Any()).Return(true).Times(1)
	mItr.EXPECT().MapScan(gomock.Any()).Return(false).Times(1)
	mItr.EXPECT().Close().Return(nil).AnyTimes()

	schedules, errs := dao.GetCronSchedulesByApp("app1", s.Scheduled)

	if len(schedules) != 0 {
		t.Errorf("Expected 0 schedules, got %d", len(schedules))
	}

	if len(errs) != 0 {
		t.Errorf("Expected 0 errors, got %d", len(errs))
	}
}
