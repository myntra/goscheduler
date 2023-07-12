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
	if s.Monitoring != nil && s.Monitoring.StatsDClient != nil {
		schedule := sch.Schedule{}
		if len(schedules) > 0 {
			schedule = schedules[0]
		}

		bucket := prefix(schedule, GetRun) + Success
		s.Monitoring.StatsDClient.Increment(bucket)
	}
}

// Record get runs failure in StatsD
func (s *Service) recordGetRunsFail() {
	if s.Monitoring != nil && s.Monitoring.StatsDClient != nil {
		bucket := constants.GetScheduleRuns + constants.DOT + Fail
		s.Monitoring.StatsDClient.Increment(bucket)
	}
}

func (s *Service) GetRuns(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scheduleId := vars["scheduleId"]

	size, when, pageState, err := parseQueryParams(r)
	if err != nil {
		s.recordGetRunsFail()
		er.Handle(w, r, er.NewError(er.InvalidDataCode, err))
		return
	}

	schedules, pageState, err := s.FetchCronRuns(scheduleId, size, when, pageState)
	if err != nil {
		s.recordGetRunsFail()
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

	switch schedules, pageState, err := (s.scheduleDao).GetScheduleRuns(scheduleId, size, when, pageState); {
	case err != nil:
		return []sch.Schedule{}, nil, er.NewError(er.DataFetchFailure, err)
	case len(schedules) == 0:
		return []sch.Schedule{}, nil, er.NewError(er.DataNotFound, errors.New(fmt.Sprint("No cron runs found")))
	default:
		return schedules, pageState, nil
	}
}
