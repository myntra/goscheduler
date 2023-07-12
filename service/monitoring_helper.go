package service

import (
	"github.com/golang/glog"
	"github.com/myntra/goscheduler/constants"
	s "github.com/myntra/goscheduler/store"
)

type action string

const (
	Create          action = "CREATE"
	Get             action = "GET"
	GetRun          action = "GET_RUNS"
	Delete          action = "DELETE"
	GetAppSchedule  action = "GET_APP_SCHEDULES"
	Reconcile       action = "RECONCILE"
	GetCronSchedule action = "GET_CRON_SCHEDULE"
)

const (
	Success = constants.DOT + constants.Success
	Fail    = constants.DOT + constants.Fail
)

// Prefix the StatsD bucket based on action type and app id.
func prefix(schedule s.Schedule, _type action) string {

	appId := schedule.AppId
	if len(appId) == 0 {
		appId = "0"
	}

	switch _type {
	case Create:
		scheduleType := constants.CreateSchedule
		if schedule.IsRecurring() {
			scheduleType = constants.CreateRecurringSchedule
		}
		return scheduleType + constants.DOT + appId
	case Get:
		return appId + constants.DOT + constants.GetSchedule
	case GetRun:
		return appId + constants.DOT + constants.GetScheduleRuns
	case GetAppSchedule:
		return appId + constants.DOT + constants.GetAppSchedule
	case Delete:
		return appId + constants.DOT + constants.DeleteSchedule
	case GetCronSchedule:
		return appId + constants.DOT + constants.GetCronSchedule

	default:
		glog.Errorf("Unknown action %s", _type)
		return ""
	}
}
