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
