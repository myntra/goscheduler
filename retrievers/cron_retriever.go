package retrievers

import (
	"github.com/golang/glog"
	"github.com/myntra/goscheduler/conf"
	"github.com/myntra/goscheduler/constants"
	"github.com/myntra/goscheduler/dao"
	p "github.com/myntra/goscheduler/monitoring"
	s "github.com/myntra/goscheduler/store"
	"strconv"
	"time"
)

type CronRetriever struct {
	scheduleDao dao.ScheduleDao
	cronConfig  *conf.CronConfig
	monitoring  *p.Monitoring
}

// Records any failure in fetching the recurring schedules to StatsD
func (r CronRetriever) recordFailure(partitionId int) {
	if r.monitoring != nil && r.monitoring.StatsDClient != nil {
		bucket := "cronretriever" + constants.DOT + strconv.Itoa(partitionId) + constants.DOT + constants.Fail
		r.monitoring.StatsDClient.Increment(bucket)
	}
}

// Get recurring schedules with partition id and pushes them on the channel for creating one time schedules for them.
// The one time schedules are created from _time till end of the configured window duration.
func (r CronRetriever) GetSchedules(app string, partitionId int, _time time.Time) error {
	var window = r.cronConfig.Window * time.Minute

	var schedules []s.Schedule
	var errs []error
	if schedules, errs = r.scheduleDao.GetRecurringScheduleByPartition(partitionId); len(errs) != 0 {
		glog.Errorf("%d errors occurred in retrieving for %s %d", len(errs), app, partitionId)
		glog.Errorf("%v", errs)
		r.recordFailure(partitionId)
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

// Implement BulkAction for Cron if required
func (r CronRetriever) BulkAction(app s.App, partitionId int, timeBucket time.Time, status []s.Status, actionType s.ActionType) error {
	return nil
}
