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

package store

import (
	"github.com/myntra/goscheduler/conf"
)

type Task struct {
	Conf *conf.Configuration
}

var (
	OldHttpTaskQueue chan ScheduleWrapper
	HttpTaskQueue    chan ScheduleWrapper
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
	OldHttpTaskQueue = make(chan ScheduleWrapper)
	HttpTaskQueue = make(chan ScheduleWrapper)
	CronTaskQueue = make(chan CreateScheduleTask)
	//making the channel buffered in order to regulate the flow in a better way
	AggregationTaskQueue = make(chan ScheduleWrapper, t.Conf.AggregateSchedulesConfig.BufferSize)
	StatusTaskQueue = make(chan StatusTask)
	//making the channel buffered in order to regulate the flow in a better way
	BulkActionQueue = make(chan BulkActionTask, t.Conf.BulkActionConfig.BufferSize)
}
