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
	"github.com/myntra/goscheduler/conf"
	"github.com/myntra/goscheduler/dao"
	p "github.com/myntra/goscheduler/monitoring"
	s "github.com/myntra/goscheduler/store"
	"time"
)

type CronRetriever struct {
	scheduleDao dao.ScheduleDao
	cronConfig  *conf.CronConfig
	monitor     p.Monitor
}

// GetSchedules Get recurring schedules with partition id and pushes them on the channel for creating one time schedules for them.
// The one time schedules are created from _time till end of the configured window duration.
func (r CronRetriever) GetSchedules(app string, partitionId int, _time time.Time) error {
	var window = r.cronConfig.Window * time.Minute

	var schedules []s.Schedule
	var errs []error
	if schedules, errs = r.scheduleDao.GetRecurringScheduleByPartition(partitionId); len(errs) != 0 {
		glog.Errorf("%d errors occurred in retrieving for %s %d", len(errs), app, partitionId)
		glog.Errorf("%v", errs)
	}

	for _, schedule := range schedules {

		if schedule.Status == s.Scheduled {
			task := s.CreateScheduleTask{
				Cron:     schedule,
				From:     _time,
				Duration: window,
			}
			s.CronTaskQueue <- task
		}
	}

	return nil
}

// BulkAction Implement BulkAction for Cron if required
func (r CronRetriever) BulkAction(app s.App, partitionId int, timeBucket time.Time, status []s.Status, actionType s.ActionType) error {
	return nil
}
