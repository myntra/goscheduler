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
	"github.com/myntra/goscheduler/dao"
	er "github.com/myntra/goscheduler/error"
	sch "github.com/myntra/goscheduler/store"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	dateTimeLayout  string = "2006-01-02 15:04:05"
	defaultDays     int    = 30
	defaultDuration        = time.Hour * time.Duration(1)
)

// parse the request for size and status and time range query params
// return error if size, time range cannot be parsed
func parse(r *http.Request) (int64, sch.Status, dao.Range, []byte, time.Time, error) {
	var size int64
	var status sch.Status
	var err error
	var timeRange dao.Range
	var _time time.Time
	var pageState []byte = nil
	var continuationStartTime int64

	query := r.URL.Query()

	continuationToken := query.Get("continuation_token")
	sizeParam := query.Get("size")
	startTime, endTime := query.Get("start_time"), query.Get("end_time")
	status = sch.Status(query.Get("status"))
	continuationStartTime, _ = strconv.ParseInt(query.Get("continuation_start_time"), 10, 64)

	glog.Infof("continuationStartTime value: %+v", continuationStartTime)

	if continuationToken != "" {
		glog.Infof("pageState received: %s", continuationToken)
		pageState, err = hex.DecodeString(continuationToken)
		if err != nil {
			return size, status, timeRange, nil, time.Unix(continuationStartTime, 0), errors.New(fmt.Sprintf("Invalid page token: %s", continuationToken))
		}
		glog.Infof("pageState decoded: %+v", pageState)
	}

	if len(sizeParam) == 0 {
		size = 15
	} else {
		if size, err = strconv.ParseInt(sizeParam, 10, 64); err != nil {
			return size, status, timeRange, pageState, time.Unix(continuationStartTime, 0), err
		}
	}

	now := time.Now()

	switch {
	case len(startTime) == 0 && len(endTime) == 0:
		return size, status, dao.Range{
			StartTime: now.Add(-1 * defaultDuration),
			EndTime:   now,
		}, pageState, time.Unix(continuationStartTime, 0), nil

	case len(startTime) > 0 && len(endTime) == 0:
		if _time, err = parseDate(startTime); err != nil {
			return size, status, timeRange, pageState, time.Unix(continuationStartTime, 0), err
		}
		return size, status, dao.Range{
			StartTime: _time,
			EndTime:   _time.Add(defaultDuration),
		}, pageState, time.Unix(continuationStartTime, 0), nil

	case len(startTime) == 0 && len(endTime) > 0:
		if _time, err = parseDate(endTime); err != nil {
			return size, status, timeRange, pageState, time.Unix(continuationStartTime, 0), err
		}
		return size, status, dao.Range{
			StartTime: _time.Add(-1 * defaultDuration),
			EndTime:   _time,
		}, pageState, time.Unix(continuationStartTime, 0), nil
	}

	timeRange, err = parseDates(startTime, endTime)
	return size, status, timeRange, pageState, time.Unix(continuationStartTime, 0), err
}

// parse the date for datetime format for local timezone by removing seconds from it
// return error if the date cannot be parsed in datetime format
func parseDate(date string) (time.Time, error) {
	var parsedDate time.Time
	var err error
	var location *time.Location

	//parse for datetime format
	location, err = time.LoadLocation("Asia/Kolkata")
	if err != nil {
		return parsedDate, err
	}
	if parsedDate, err = time.ParseInLocation(dateTimeLayout, date, location); err != nil {
		return parsedDate, err
	}

	glog.Infof("Date parsed:- %s", parsedDate)
	temp := time.Unix((parsedDate.Unix()/60)*60, 0)
	return temp, nil
}

// parse start datetime and end datetime params
// return error if the any one of the date cannot be parsed
func parseDates(startTime string, endTime string) (dao.Range, error) {
	var timeRange dao.Range

	start, startTimeErr := parseDate(startTime)
	if startTimeErr != nil {
		return dao.Range{}, errors.New(fmt.Sprintf("Incorrect start_time query parameter format %s (expected format %s)", startTimeErr.Error(), dateTimeLayout))
	}

	//set startTime
	timeRange.StartTime = start

	end, endTimeErr := parseDate(endTime)
	if endTimeErr != nil {
		return dao.Range{}, errors.New(fmt.Sprintf("Incorrect end_time query parameter format %s (expected format %s)", endTimeErr.Error(), dateTimeLayout))
	}

	//set endTime
	timeRange.EndTime = end

	return timeRange, nil
}

// Record get app schedules success in StatsD
func (s *Service) recordGetAppSchedulesSuccess(schedules []sch.Schedule) {
	if s.Monitoring != nil && s.Monitoring.StatsDClient != nil {
		schedule := sch.Schedule{}
		if len(schedules) > 0 {
			schedule = schedules[0]
		}

		bucket := prefix(schedule, GetAppSchedule) + Success
		s.Monitoring.StatsDClient.Increment(bucket)
	}
}

// Record get app schedules failure in StatsD
func (s *Service) recordGetAppSchedulesFail(schedules []sch.Schedule) {
	if s.Monitoring != nil && s.Monitoring.StatsDClient != nil {
		schedule := sch.Schedule{}
		if len(schedules) > 0 {
			schedule = schedules[0]
		}

		bucket := prefix(schedule, GetAppSchedule) + Fail
		s.Monitoring.StatsDClient.Increment(bucket)
	}
}

// get all the schedules of an app based on time range and status
func (s *Service) GetAppSchedules(w http.ResponseWriter, r *http.Request) {
	var errs []string

	vars := mux.Vars(r)
	appId := vars["appId"]

	if size, status, timeRange, pageState, continuationStartTime, err := parse(r); err != nil {
		errs = append(errs, err.Error())
		er.Handle(w, r, er.NewError(er.InvalidDataCode, errors.New(strings.Join(errs, ","))))
	} else if size < 0 {
		er.Handle(w, r, er.NewError(er.InvalidDataCode,
			errors.New(fmt.Sprintf("Size provided(%d) should not be less than 0", size))))
	} else if timeRange.EndTime.Before(timeRange.StartTime) {
		er.Handle(w, r, er.NewError(er.InvalidDataCode,
			errors.New(fmt.Sprintf("End time: %s cannot be before start time: %s", timeRange.EndTime, timeRange.StartTime))))
	} else if timeRange.EndTime.Sub(timeRange.StartTime).Seconds() > float64(defaultDays*24*60*60) {
		er.Handle(w, r, er.NewError(er.InvalidDataCode,
			errors.New(fmt.Sprintf("Time range of more than %d days is not allowed", defaultDays))))
	} else {
		schedules, pageState, continuationStartTime, err := s.FetchAppSchedules(appId, timeRange, size, status, pageState, continuationStartTime)
		if err != nil {
			s.recordGetAppSchedulesFail(schedules)
			er.Handle(w, r, err.(er.AppError))
			return
		}

		s.recordGetAppSchedulesSuccess(schedules)
		status := Status{
			StatusCode:    constants.SuccessCode200,
			StatusMessage: constants.Success,
			StatusType:    constants.Success,
			TotalCount:    len(schedules),
		}
		data := GetPaginatedAppSchedulesData{
			Schedules: schedules,
			ContinuationToken: func() string {
				glog.Infof("pageState string: %+v", pageState)
				return hex.EncodeToString(pageState)
			}(),
			ContinuationStartTime: continuationStartTime.Unix(),
		}

		_ = json.NewEncoder(w).Encode(
			GetPaginatedAppSchedulesResponse{
				Status: status,
				Data:   data,
			})
	}
}

func (s *Service) FetchAppSchedules(appId string, timeRange dao.Range, size int64, status sch.Status, pageState []byte, continuationStartTime time.Time) ([]sch.Schedule, []byte, time.Time, error) {
	app, err := s.getActiveOrInactiveApp(appId)
	if err != nil {
		return []sch.Schedule{}, nil, time.Now(), err
	}

	schedules, pageState, continuationStartTime, err := s.ScheduleDao.GetPaginatedSchedules(appId, int(app.Partitions), timeRange, size, status, pageState, continuationStartTime)
	if err != nil {
		return []sch.Schedule{}, nil, time.Now(), er.NewError(er.DataFetchFailure, err)
	}

	return schedules, pageState, continuationStartTime, nil
}

func (s *Service) getActiveOrInactiveApp(appId string) (sch.App, error) {
	app, err := s.ClusterDao.GetApp(appId)
	switch {
	case err == gocql.ErrNotFound:
		return sch.App{}, er.NewError(er.InvalidAppId, errors.New(fmt.Sprintf("app id %s is not registered", appId)))
	case err != nil:
		return sch.App{}, er.NewError(er.DataFetchFailure, err)
	case len(app.AppId) == 0:
		return sch.App{}, er.NewError(er.InvalidAppId, errors.New(fmt.Sprintf("app Id %s is not registered", appId)))
	default:
		return app, nil
	}
}
