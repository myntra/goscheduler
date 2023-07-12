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

// Record delete schedule success in StatsD
func (s *Service) recordDeleteSuccess(schedule sch.Schedule) {
	if s.Monitoring != nil && s.Monitoring.StatsDClient != nil {
		bucket := prefix(schedule, Delete) + Success
		s.Monitoring.StatsDClient.Increment(bucket)
	}

}

// Record delete schedule failure in StatsD
func (s *Service) recordDeleteFail() {
	if s.Monitoring != nil && s.Monitoring.StatsDClient != nil {
		bucket := constants.DeleteSchedule + constants.DOT + Fail
		s.Monitoring.StatsDClient.Increment(bucket)
	}
}

func (s *Service) CancelSchedule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	uuid := vars["scheduleId"]

	schedule, err := s.DeleteSchedule(uuid)
	if err != nil {
		s.recordDeleteFail()
		er.Handle(w, r, err.(er.AppError))
		return
	}

	s.recordDeleteSuccess(schedule)

	status := Status{
		StatusCode:    constants.SuccessCode200,
		StatusMessage: constants.Success,
		StatusType:    constants.Success,
		TotalCount:    1,
	}
	data := DeleteScheduleData{
		Schedule: schedule,
	}
	_ = json.NewEncoder(w).Encode(
		DeleteScheduleResponse{
			Status: status,
			Data:   data,
		})
}

func (s *Service) DeleteSchedule(uuid string) (sch.Schedule, error) {
	scheduleId, err := gocql.ParseUUID(uuid)
	if err != nil {
		return sch.Schedule{}, er.NewError(er.InvalidDataCode, err)
	}

	switch schedule, err := s.scheduleDao.DeleteSchedule(scheduleId); err {
	case gocql.ErrNotFound:
		return sch.Schedule{}, er.NewError(er.DataNotFound, err)
	case nil:
		return schedule, nil
	default:
		return sch.Schedule{}, er.NewError(er.DataFetchFailure, err)
	}
}
