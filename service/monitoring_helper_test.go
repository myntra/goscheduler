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
	"github.com/myntra/goscheduler/constants"
	s "github.com/myntra/goscheduler/store"
	"testing"
)

func TestService_prefix(t *testing.T) {
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
