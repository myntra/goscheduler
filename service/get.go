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
	"github.com/gocql/gocql"
	"github.com/gorilla/mux"
	"github.com/myntra/goscheduler/constants"
	er "github.com/myntra/goscheduler/error"
	sch "github.com/myntra/goscheduler/store"
	"net/http"
)

// Record get schedule success in StatsD
func (s *Service) recordGetSuccess(schedule sch.Schedule) {
	if s.Monitoring != nil && s.Monitoring.StatsDClient != nil {
		bucket := prefix(schedule, Get) + Success
		s.Monitoring.StatsDClient.Increment(bucket)
	}
}

// Record get schedule failure in StatsD
func (s *Service) recordGetFail() {
	if s.Monitoring != nil && s.Monitoring.StatsDClient != nil {
		bucket := constants.GetSchedule + constants.DOT + Fail
		s.Monitoring.StatsDClient.Increment(bucket)
	}
}

func (s *Service) Get(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	uuid := vars["scheduleId"]

	schedule, err := s.GetSchedule(uuid)
	if err != nil {
		s.recordGetFail()
		er.Handle(w, r, err.(er.AppError))
		return
	}

	s.recordGetSuccess(schedule)

	status := Status{
		StatusCode:    constants.SuccessCode200,
		StatusMessage: constants.Success,
		StatusType:    constants.Success,
		TotalCount:    1,
	}
	data := GetScheduleData{
		Schedule: schedule,
	}
	_ = json.NewEncoder(w).Encode(
		GetScheduleResponse{
			Status: status,
			Data:   data,
		})
}

func (s *Service) GetSchedule(uuid string) (sch.Schedule, error) {
	scheduleId, err := gocql.ParseUUID(uuid)
	if err != nil {
		return sch.Schedule{}, er.NewError(er.InvalidDataCode, err)
	}

	switch schedule, err := s.scheduleDao.GetEnrichedSchedule(scheduleId); err {
	case gocql.ErrNotFound:
		return sch.Schedule{}, er.NewError(er.DataNotFound, err)
	case nil:
		return schedule, nil
	default:
		return sch.Schedule{}, er.NewError(er.DataFetchFailure, err)
	}
}
