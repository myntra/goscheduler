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

package connectors

import (
	"github.com/gocql/gocql"
	"github.com/golang/glog"
	"github.com/myntra/goscheduler/constants"
	"github.com/myntra/goscheduler/cron"
	s "github.com/myntra/goscheduler/store"
	"strconv"
	"time"
)

func prefix(schedule s.Schedule) string {
	return "connectors" + constants.DOT + "cron" + constants.DOT +
		schedule.AppId + constants.DOT + strconv.Itoa(schedule.PartitionId)
}

// TODO: Can we rename these functions?
// Record a one time schedule creation success in StatsD
func (c *Connector) recordSuccess(schedule s.Schedule) {
	if c.Monitoring != nil && c.Monitoring.StatsDClient != nil {
		bucket := prefix(schedule) + constants.DOT + constants.Success
		c.Monitoring.StatsDClient.Increment(bucket)
	}
}

// Record a one time schedule creation failure in StatsD
func (c *Connector) recordFailure(schedule s.Schedule) {
	if c.Monitoring != nil && c.Monitoring.StatsDClient != nil {
		bucket := prefix(schedule) + constants.DOT + constants.Fail
		c.Monitoring.StatsDClient.Increment(bucket)
	}
}

// Creates one time schedules for a recurring schedule.
// Listens for a create task event on the channel. And creates the schedule if the cron expression matches
// any time within the duration window.
// If the time doesn't match or if a schedule already exists at time then the creation will be skipped.
// The method records any errors occurred during execution and recovers.
func (c *Connector) createSchedules(tasks <-chan s.CreateScheduleTask) {
	for task := range tasks {
		var err error
		var errs []string

		parent := task.Cron

		if !parent.IsRecurring() {
			glog.Errorf("Schedule %v is not a cron", parent)
			c.recordFailure(parent)

			continue
		}

		var _cron cron.Expression
		if _cron, errs = cron.Parse(parent.CronExpression); len(errs) != 0 {
			glog.Errorf("Parsing cron expression for schedule %s failed with errors %v", parent.ScheduleId, errs)
			c.recordFailure(parent)

			continue
		}

		var app s.App
		if app, err = c.ClusterDao.GetApp(parent.AppId); err != nil || !app.Active {
			glog.Errorf("App %s is not active", parent.AppId)
			c.recordFailure(parent)

			continue
		}

		existing := map[time.Time]bool{}
		switch runs, _, err := c.ScheduleDao.GetScheduleRuns(parent.ScheduleId, int64(task.Duration/time.Minute), "future", nil); {
		case err == nil, err == gocql.ErrNotFound:
			for _, run := range runs {
				existing[time.Unix(run.ScheduleGroup, 0)] = true
			}
		default:
			glog.Errorf("Error getting future runs for %s", parent.ScheduleId)
			c.recordFailure(parent)

			continue
		}

		for _time := task.From.Add(task.Duration); _time.After(task.From); _time = _time.Add(time.Minute * -1) {

			if _, found := existing[_time]; !found && _cron.Match(_time) {

				clone := parent.CloneAsOneTime(_time)
				clone.SetFields(app)
				if errs := clone.ValidateSchedule(app, c.Config.AppLevelConfiguration); len(errs) != 0 {
					glog.Errorf(
						"Validation failed for one time schedule %v of cron %s with errors %v",
						clone, parent.ScheduleId, errs)
					c.recordFailure(parent)

					continue
				}

				if clone, err = c.ScheduleDao.CreateRun(clone, app); err != nil {
					glog.Errorf(
						"Creation failed for one time schedule %v of cron %s with errors %s",
						clone, parent.ScheduleId, err.Error())
					c.recordFailure(parent)

					continue
				}

				c.recordSuccess(parent)
			}
		}
	}
}

// Start count number of go routines to listen on the task channel
// The routines on receiving the messages will create one time schedules.
func (c *Connector) StartScheduleCreateWorkers(tasks <-chan s.CreateScheduleTask) {
	count := c.Config.CronConfig.Routines
	for i := 0; i < count; i++ {
		go c.createSchedules(tasks)
	}
}

func (c *Connector) initCronRetriever() {
	go c.StartScheduleCreateWorkers(s.CronTaskQueue)
}
