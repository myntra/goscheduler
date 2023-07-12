package store

import (
	"github.com/myntra/goscheduler/conf"
)

type Task struct {
	Conf *conf.Configuration
}

var (
	HttpTaskQueue chan ScheduleWrapper
	// Channel sends the tasks to convert a recurring schedule to one time schedules
	CronTaskQueue chan CreateScheduleTask
	// Channel aggregates the schedules and forward to status update
	AggregationTaskQueue chan ScheduleWrapper
	// Channel updates the status of schedules after callback is fired
	StatusTaskQueue chan StatusTask
	// Channel
	BulkActionQueue chan BulkActionTask
)

func (t *Task) InitTaskQueues() {
	HttpTaskQueue = make(chan ScheduleWrapper)
	CronTaskQueue = make(chan CreateScheduleTask)
	//making the channel buffered in order to regulate the flow in a better way
	AggregationTaskQueue = make(chan ScheduleWrapper, t.Conf.AggregateSchedulesConfig.BufferSize)
	StatusTaskQueue = make(chan StatusTask)
	//making the channel buffered in order to regulate the flow in a better way
	BulkActionQueue = make(chan BulkActionTask, t.Conf.BulkActionConfig.BufferSize)
}
