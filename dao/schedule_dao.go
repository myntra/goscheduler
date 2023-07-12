package dao

import (
	"github.com/gocql/gocql"
	"github.com/myntra/goscheduler/db_wrapper"
	s "github.com/myntra/goscheduler/store"
	"time"
)

type ScheduleDao interface {
	CreateSchedule(schedule s.Schedule) (s.Schedule, error)
	GetRecurringScheduleByPartition(partitionId int) ([]s.Schedule, []error)
	GetSchedule(uuid gocql.UUID) (s.Schedule, error)
	GetEnrichedSchedule(uuid gocql.UUID) (s.Schedule, error)
	EnrichSchedule(schedule *s.Schedule) error
	DeleteSchedule(uuid gocql.UUID) (s.Schedule, error)
	GetScheduleRuns(uuid gocql.UUID, size int64, when string, pageState []byte) ([]s.Schedule, []byte, error)
	CreateRun(schedule s.Schedule) (s.Schedule, error)
	UpdateStatus(schedules []s.Schedule) error
	GetPaginatedSchedules(appId string, partitions int, timeRange Range, size int64, status s.Status, pageState []byte, continuationStartTime time.Time) ([]s.Schedule, []byte, time.Time, error)
	GetSchedulesForEntity(appId string, partitionId int, timeBucket time.Time, pageState []byte) db_wrapper.IterInterface
	OptimizedEnrichSchedule(schedules []s.Schedule) ([]s.Schedule, error)
	GetCronSchedulesByApp(appId string, status s.Status) ([]s.Schedule, []string)
	BulkAction(app s.App, partitionId int, scheduleTimeGroup time.Time, status []s.Status, actionType s.ActionType) error
}
