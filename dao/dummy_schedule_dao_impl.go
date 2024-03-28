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
	"errors"
	"github.com/gocql/gocql"
	"github.com/myntra/goscheduler/db_wrapper"
	s "github.com/myntra/goscheduler/store"
	"time"
)

type DummyScheduleDaoImpl struct{}

func (d *DummyScheduleDaoImpl) CreateSchedule(schedule s.Schedule, app s.App) (s.Schedule, error) {
	switch schedule.AppId {
	case "createScheduleFailureApp":
		return schedule, errors.New("error")
	}
	return schedule, nil
}

func (d *DummyScheduleDaoImpl) GetRecurringScheduleByPartition(partitionId int) ([]s.Schedule, []error) {
	return []s.Schedule{}, nil
}

func (d *DummyScheduleDaoImpl) GetSchedule(uuid gocql.UUID) (s.Schedule, error) {
	return s.Schedule{}, nil
}

func (d *DummyScheduleDaoImpl) GetEnrichedSchedule(uuid gocql.UUID) (s.Schedule, error) {
	switch uuid.String() {
	case "00000000-0000-0000-0000":
		return s.Schedule{}, errors.New("error")
	case "00000000-0000-0000-0000-000000000000":
		return s.Schedule{}, gocql.ErrNotFound
	case "84d0d5b8-d953-11ed-a827-aa665a372253":
		return s.Schedule{}, errors.New("something went wrong")
	default:
		return s.Schedule{}, nil
	}
}

func (d *DummyScheduleDaoImpl) EnrichSchedule(schedule *s.Schedule) error {
	return nil
}

func (d *DummyScheduleDaoImpl) DeleteSchedule(uuid gocql.UUID) (s.Schedule, error) {
	switch uuid.String() {
	case "00000000-0000-0000-0000":
		return s.Schedule{}, errors.New("error")
	case "00000000-0000-0000-0000-000000000000":
		return s.Schedule{}, gocql.ErrNotFound
	case "84d0d5b8-d953-11ed-a827-aa665a372253":
		return s.Schedule{}, errors.New("something went wrong")
	default:
		return s.Schedule{}, nil
	}
}

func (d *DummyScheduleDaoImpl) GetScheduleRuns(uuid gocql.UUID, size int64, when string, pageState []byte) ([]s.Schedule, []byte, error) {
	switch when {
	case "past":
	case "future":
	default:
		switch uuid.String() {
		case "00000000-0000-0000-0000-000000000000":
			return []s.Schedule{}, nil, errors.New("error")
		case "00000000-0000-0000-0000-000000000001":
			return []s.Schedule{}, nil, nil
		default:
			return []s.Schedule{
				{
					ScheduleId:    gocql.TimeUUID(),
					Payload:       "dummy payload",
					AppId:         "dummy app id",
					ScheduleTime:  time.Now().Unix(),
					PartitionId:   1,
					ScheduleGroup: 1,
					Callback: &s.HttpCallback{
						Type: "exampleType",
						Details: struct {
							Url     string            `json:"url"`
							Method  string            `json:"method"`
							Headers map[string]string `json:"headers"`
						}{
							Url:    "http://example.com/callback",
							Method: "TEST",
							Headers: map[string]string{
								"Content-Type":  "application/json",
								"Authorization": "Bearer sample-token",
							},
						},
					},
					CronExpression:   "",
					Status:           s.Scheduled,
					ErrorMessage:     "",
					ParentScheduleId: gocql.UUID{},
				},
			}, nil, nil
		}
	}

	return []s.Schedule{}, nil, nil
}

func (d *DummyScheduleDaoImpl) CreateRun(schedule s.Schedule, app s.App) (s.Schedule, error) {
	return schedule, nil
}

func (d *DummyScheduleDaoImpl) UpdateStatus(schedules []s.Schedule, app s.App) error {
	return nil
}

func (d *DummyScheduleDaoImpl) GetPaginatedSchedules(appId string, partitions int, timeRange Range, size int64, status s.Status, pageState []byte, continuationStartTime time.Time) ([]s.Schedule, []byte, time.Time, error) {
	return []s.Schedule{}, nil, time.Time{}, nil
}

func (d *DummyScheduleDaoImpl) GetSchedulesForEntity(appId string, partitionId int, timeBucket time.Time, pageState []byte) db_wrapper.IterInterface {
	return nil
}

func (d *DummyScheduleDaoImpl) OptimizedEnrichSchedule(schedules []s.Schedule) ([]s.Schedule, error) {
	return schedules, nil
}

func (d *DummyScheduleDaoImpl) GetCronSchedulesByApp(appId string, status s.Status) ([]s.Schedule, []string) {
	switch appId {
	case "testGetAppError":
		return []s.Schedule{
			{
				ScheduleId:    gocql.TimeUUID(),
				Payload:       "dummy payload",
				AppId:         "dummy app id",
				ScheduleTime:  time.Now().Unix(),
				PartitionId:   1,
				ScheduleGroup: 1,
				Callback: &s.HttpCallback{
					Type: "exampleType",
					Details: struct {
						Url     string            `json:"url"`
						Method  string            `json:"method"`
						Headers map[string]string `json:"headers"`
					}{
						Url:    "http://example.com/callback",
						Method: "TEST",
						Headers: map[string]string{
							"Content-Type":  "application/json",
							"Authorization": "Bearer sample-token",
						},
					},
				},
				CronExpression:   "",
				Status:           s.Scheduled,
				ErrorMessage:     "",
				ParentScheduleId: gocql.UUID{},
			},
		}, []string{"error"}
	case "testGetCronSchedulesError":
		return []s.Schedule{}, nil
	default:
		return []s.Schedule{
			{
				ScheduleId:    gocql.TimeUUID(),
				Payload:       "dummy payload",
				AppId:         "dummy app id",
				ScheduleTime:  time.Now().Unix(),
				PartitionId:   1,
				ScheduleGroup: 1,
				Callback: &s.HttpCallback{
					Type: "exampleType",
					Details: struct {
						Url     string            `json:"url"`
						Method  string            `json:"method"`
						Headers map[string]string `json:"headers"`
					}{
						Url:    "http://example.com/callback",
						Method: "TEST",
						Headers: map[string]string{
							"Content-Type":  "application/json",
							"Authorization": "Bearer sample-token",
						},
					},
				},
				CronExpression:   "",
				Status:           s.Scheduled,
				ErrorMessage:     "",
				ParentScheduleId: gocql.UUID{},
			},
		}, nil
	}
}

func (d *DummyScheduleDaoImpl) BulkAction(app s.App, partitionId int, scheduleTimeGroup time.Time, status []s.Status, actionType s.ActionType) error {
	return nil
}
