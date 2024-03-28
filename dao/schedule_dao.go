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
	"github.com/gocql/gocql"
	"github.com/myntra/goscheduler/db_wrapper"
	s "github.com/myntra/goscheduler/store"
	"time"
)

type ScheduleDao interface {
	CreateSchedule(schedule s.Schedule, app s.App) (s.Schedule, error)
	GetRecurringScheduleByPartition(partitionId int) ([]s.Schedule, []error)
	GetSchedule(uuid gocql.UUID) (s.Schedule, error)
	GetEnrichedSchedule(uuid gocql.UUID) (s.Schedule, error)
	EnrichSchedule(schedule *s.Schedule) error
	DeleteSchedule(uuid gocql.UUID) (s.Schedule, error)
	GetScheduleRuns(uuid gocql.UUID, size int64, when string, pageState []byte) ([]s.Schedule, []byte, error)
	CreateRun(schedule s.Schedule, app s.App) (s.Schedule, error)
	UpdateStatus(schedules []s.Schedule, app s.App) error
	GetPaginatedSchedules(appId string, partitions int, timeRange Range, size int64, status s.Status, pageState []byte, continuationStartTime time.Time) ([]s.Schedule, []byte, time.Time, error)
	GetSchedulesForEntity(appId string, partitionId int, timeBucket time.Time, pageState []byte) db_wrapper.IterInterface
	OptimizedEnrichSchedule(schedules []s.Schedule) ([]s.Schedule, error)
	GetCronSchedulesByApp(appId string, status s.Status) ([]s.Schedule, []string)
	BulkAction(app s.App, partitionId int, scheduleTimeGroup time.Time, status []s.Status, actionType s.ActionType) error
}
