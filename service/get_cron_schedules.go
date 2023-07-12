package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/myntra/goscheduler/constants"
	er "github.com/myntra/goscheduler/error"
	sch "github.com/myntra/goscheduler/store"
	"net/http"
	"strings"
)

func (s *Service) recordGetCronAppSchedulesSuccess(schedules []sch.Schedule) {
	if s.Monitoring != nil && s.Monitoring.StatsDClient != nil {
		cronSchedule := sch.Schedule{}
		if len(schedules) > 0 {
			cronSchedule = schedules[0]
		}

		bucket := prefix(cronSchedule, GetCronSchedule) + Success
		s.Monitoring.StatsDClient.Increment(bucket)
	}
}

func (s *Service) recordGetCronAppSchedulesFail() {
	if s.Monitoring != nil && s.Monitoring.StatsDClient != nil {
		bucket := constants.GetCronSchedule + constants.DOT + Fail
		s.Monitoring.StatsDClient.Increment(bucket)
	}
}

func parseCron(r *http.Request) (string, sch.Status, error) {
	var appId string
	var status sch.Status

	query := r.URL.Query()
	appId = query.Get("app_id")
	status = sch.Status(query.Get("status"))

	return appId, status, nil
}

func (s *Service) GetCronSchedules(w http.ResponseWriter, r *http.Request) {
	appId, status, _ := parseCron(r)

	cronSchedules, err := s.FetchCronSchedules(appId, status)
	if err != nil {
		s.recordGetCronAppSchedulesFail()
		er.Handle(w, r, err.(er.AppError))
		return
	}

	s.recordGetCronAppSchedulesSuccess(cronSchedules)
	_ = json.NewEncoder(w).Encode(
		GetCronSchedulesResponse{
			Status: Status{
				StatusCode:    constants.SuccessCode200,
				StatusMessage: constants.Success,
				StatusType:    constants.Success,
			},
			Data:   cronSchedules,
		})
}

func (s *Service) FetchCronSchedules(appId string, status sch.Status) ([]sch.Schedule, error) {
	switch cronSchedules, errs := (s.scheduleDao).GetCronSchedulesByApp(appId, status); {
	case len(errs) != 0:
		return []sch.Schedule{}, er.NewError(er.DataFetchFailure, errors.New(strings.Join(errs, ",")))
	case len(cronSchedules) == 0:
		return []sch.Schedule{}, er.NewError(er.DataNotFound, errors.New(fmt.Sprint("No cron schedules found")))
	default:
		return cronSchedules, nil
	}
}
