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
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gocql/gocql"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/myntra/goscheduler/constants"
	er "github.com/myntra/goscheduler/error"
	sch "github.com/myntra/goscheduler/store"
	"net/http"
	"strconv"
)

func parseQueryParams(r *http.Request) (int64, string, []byte, error) {
	var size int64
	var when string
	var continuationToken string
	var err error
	var pageState []byte = nil

	sizeParam := r.URL.Query().Get("size")
	if len(sizeParam) == 0 {
		size = 15
	} else {
		if size, err = strconv.ParseInt(sizeParam, 10, 64); err != nil {
			glog.Errorf("Cannot parse size %s to int", sizeParam)
		}
	}

	continuationToken = r.URL.Query().Get("continuation_token")
	when = r.URL.Query().Get("when")
	if continuationToken != "" {
		glog.Infof("pageState received: %s", continuationToken)
		pageState, err = hex.DecodeString(continuationToken)
		if err != nil {
			return size, when, nil, errors.New(fmt.Sprintf("Invalid page token: %s", continuationToken))
		}
		glog.Infof("pageState decoded: %+v", pageState)
	}
	return size, when, pageState, err
}

// Record get runs success in StatsD
func (s *Service) recordGetRunsSuccess(schedules []sch.Schedule) {
	schedule := sch.Schedule{}
	if len(schedules) > 0 {
		schedule = schedules[0]
	}
	s.recordRequestAppStatus(constants.GetScheduleRuns, getAppId(schedule), constants.Success)
}

func (s *Service) GetRuns(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scheduleId := vars["scheduleId"]

	size, when, pageState, err := parseQueryParams(r)
	if err != nil {
		s.recordRequestStatus(constants.GetScheduleRuns, constants.Fail)
		er.Handle(w, r, er.NewError(er.InvalidDataCode, err))
		return
	}

	schedules, pageState, err := s.FetchCronRuns(scheduleId, size, when, pageState)
	if err != nil {
		s.recordRequestStatus(constants.GetScheduleRuns, constants.Fail)
		er.Handle(w, r, err.(er.AppError))
		return
	}

	s.recordGetRunsSuccess(schedules)

	status := Status{
		StatusCode:    constants.SuccessCode200,
		StatusMessage: constants.Success,
		StatusType:    constants.Success,
		TotalCount:    len(schedules),
	}
	data := GetPaginatedRunSchedulesData{
		Schedules: schedules,
		ContinuationToken: func() string {
			glog.Infof("pageState string: %+v", pageState)
			return hex.EncodeToString(pageState)
		}(),
	}

	_ = json.NewEncoder(w).Encode(
		GetPaginatedRunSchedulesResponse{
			Status: status,
			Data:   data,
		})
}

func (s *Service) FetchCronRuns(uuid string, size int64, when string, pageState []byte) ([]sch.Schedule, []byte, error) {
	scheduleId, err := gocql.ParseUUID(uuid)
	if err != nil {
		return []sch.Schedule{}, nil, er.NewError(er.InvalidDataCode, err)
	}

	switch schedules, pageState, err := (s.ScheduleDao).GetScheduleRuns(scheduleId, size, when, pageState); {
	case err != nil:
		return []sch.Schedule{}, nil, er.NewError(er.DataFetchFailure, err)
	case len(schedules) == 0:
		return []sch.Schedule{}, nil, er.NewError(er.DataNotFound, errors.New(fmt.Sprint("No cron runs found")))
	default:
		return schedules, pageState, nil
	}
}
