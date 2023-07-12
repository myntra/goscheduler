package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/myntra/goscheduler/constants"
	er "github.com/myntra/goscheduler/error"
	"github.com/myntra/goscheduler/store"
	"net/http"
)

// Record get apps success in StatsD
func (s *Service) recordGetAppsSuccess() {
	if s.Monitoring != nil && s.Monitoring.StatsDClient != nil {
		bucket := constants.GetApps + constants.DOT + Success
		s.Monitoring.StatsDClient.Increment(bucket)
	}
}

// Record get apps failure in StatsD
func (s *Service) recordGetAppsFail() {
	if s.Monitoring != nil && s.Monitoring.StatsDClient != nil {
		bucket := constants.GetApps + constants.DOT + Fail
		s.Monitoring.StatsDClient.Increment(bucket)
	}
}

func parseAppQueryParams(r *http.Request) string {
	query := r.URL.Query()
	return query.Get("app_id")
}

func (s *Service) GetApps(w http.ResponseWriter, r *http.Request) {
	appId := parseAppQueryParams(r)

	apps, err := s.FetchApps(appId)
	if err != nil {
		s.recordGetAppsFail()
		er.Handle(w, r, err.(er.AppError))
		return
	}

	s.recordGetAppsSuccess()

	status := Status{
		StatusCode:    constants.SuccessCode200,
		StatusMessage: constants.Success,
		StatusType:    constants.Success,
		TotalCount:    len(apps),
	}

	data := GetAppsData{
		Apps: apps,
	}

	_ =  json.NewEncoder(w).Encode(
		GetAppsResponse{
			Status: status,
			Data:   data,
		})
}

func (s *Service) FetchApps(appId string) ([]store.App, error) {
	switch apps, err := s.clusterDao.GetApps(appId); {
	case err != nil:
		return []store.App{}, er.NewError(er.DataFetchFailure, err)
	case len(apps) == 0:
		return []store.App{}, er.NewError(er.DataNotFound, errors.New(fmt.Sprint("No app found")))
	default:
		return apps, nil
	}
}
