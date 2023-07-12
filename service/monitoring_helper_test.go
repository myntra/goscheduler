package service

import (
	"github.com/myntra/goscheduler/constants"
	s "github.com/myntra/goscheduler/store"
	"testing"
)

func TestPrefix(t *testing.T) {
	tests := []struct {
		schedule   s.Schedule
		actionType action
		want       string
	}{
		{s.Schedule{AppId: "app1"}, Create, constants.CreateSchedule + constants.DOT + "app1"},
		{s.Schedule{AppId: "app1", CronExpression: "*****"}, Create, constants.CreateRecurringSchedule + constants.DOT + "app1"},
		{s.Schedule{AppId: "app2"}, Get, "app2" + constants.DOT + constants.GetSchedule},
		{s.Schedule{AppId: "app2"}, GetRun, "app2" + constants.DOT + constants.GetScheduleRuns},
		{s.Schedule{AppId: "app3"}, GetAppSchedule, "app3" + constants.DOT + constants.GetAppSchedule},
		{s.Schedule{AppId: "app4"}, Delete, "app4" + constants.DOT + constants.DeleteSchedule},
		{s.Schedule{AppId: "app5"}, GetCronSchedule, "app5" + constants.DOT + constants.GetCronSchedule},
		{s.Schedule{AppId: ""}, Create, constants.CreateSchedule + constants.DOT + "0"},
		{s.Schedule{}, Get, "0" + constants.DOT + constants.GetSchedule},
		{s.Schedule{}, Reconcile, ""},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			got := prefix(test.schedule, test.actionType)
			if got != test.want {
				t.Errorf("prefix(%v, %v) = %v, want %v", test.schedule, test.actionType, got, test.want)
			}
		})
	}
}
