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
