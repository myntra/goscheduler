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
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gocql/gocql"
	"github.com/golang/glog"
	"github.com/myntra/goscheduler/constants"
	er "github.com/myntra/goscheduler/error"
	sch "github.com/myntra/goscheduler/store"
	"io/ioutil"
	"net/http"
	"strings"
)

// Record create schedule success success in StatsD
func (s *Service) recordCreateSuccess(schedule sch.Schedule) {
	if s.Monitoring != nil && s.Monitoring.StatsDClient != nil {
		bucket := prefix(schedule, Create) + Success
		s.Monitoring.StatsDClient.Increment(bucket)
	}
}

// Record create schedule failure in StatsD
func (s *Service) recordCreateFail(schedule sch.Schedule) {
	if s.Monitoring != nil && s.Monitoring.StatsDClient != nil {
		bucket := prefix(schedule, Create) + Fail
		s.Monitoring.StatsDClient.Increment(bucket)
	}
}

func (s *Service) Post(w http.ResponseWriter, r *http.Request) {
	var input sch.Schedule

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		s.recordCreateFail(sch.Schedule{})
		er.Handle(w, r, er.NewError(er.UnmarshalErrorCode, err))
		return
	}

	err = json.Unmarshal(b, &input)
	if err != nil {
		s.recordCreateFail(sch.Schedule{})
		er.Handle(w, r, er.NewError(er.UnmarshalErrorCode, err))
		return
	}

	//To be removed
	glog.Infof("Successfully unmarshalled schedule. Schedule: %+v", input)

	schedule, err := s.CreateSchedule(input)
	if err != nil {
		s.recordCreateFail(schedule)
		er.Handle(w, r, err.(er.AppError))
	} else {
		s.recordCreateSuccess(schedule)
		glog.V(constants.INFO).Infof("Schedule created successfully. Schedule id is :  %s ", schedule.ScheduleId)
		status := Status{StatusCode: constants.SuccessCode201, StatusMessage: constants.Success, StatusType: constants.Success, TotalCount: 1}
		_ = json.NewEncoder(w).Encode(CreateScheduleResponse{Status: status, Data: CreateScheduleData{Schedule: schedule}})
	}

}

// createSchedule creates a new schedule
func (s *Service) CreateSchedule(input sch.Schedule) (sch.Schedule, error) {
	app, err := s.getApp(input.AppId)
	if err != nil {
		return sch.Schedule{}, err
	}

	errs := input.ValidateSchedule(app, s.Config.AppLevelConfiguration)
	if errs != nil && len(errs) > 0 {
		return sch.Schedule{}, er.NewError(er.InvalidDataCode, errors.New(strings.Join(errs, ",")))
	}

	if input.IsRecurring() {
		cronApp, err := s.getApp(s.Config.CronConfig.App)
		if err != nil {
			return sch.Schedule{}, er.NewError(er.DataPersistenceFailure, err)
		}
		app = cronApp
	}

	input.SetFields(app)

	schedule, err := s.scheduleDao.CreateSchedule(input, app)
	if err != nil {
		return sch.Schedule{}, er.NewError(er.DataPersistenceFailure, err)
	}

	return schedule, nil
}

// getApp retrieves the app based on the provided app ID
func (s *Service) getApp(appId string) (sch.App, error) {
	app, err := s.clusterDao.GetApp(appId)
	switch {
	case err == gocql.ErrNotFound:
		return sch.App{}, er.NewError(er.InvalidAppId, errors.New(fmt.Sprintf("app Id %s is not registered", appId)))
	case err != nil:
		return sch.App{}, er.NewError(er.DataFetchFailure, err)
	case len(app.AppId) == 0:
		return sch.App{}, er.NewError(er.InvalidAppId, errors.New(fmt.Sprintf("app Id %s is not registered", appId)))
	case !app.Active:
		return sch.App{}, er.NewError(er.DeactivatedApp, errors.New(fmt.Sprintf("app Id %s is deactivated", app.AppId)))
	default:
		return app, nil
	}
}
