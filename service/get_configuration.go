package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gocql/gocql"
	"github.com/gorilla/mux"
	"github.com/myntra/goscheduler/constants"
	er "github.com/myntra/goscheduler/error"
	sch "github.com/myntra/goscheduler/store"
	"net/http"
)

func (s *Service) recordGetConfigurationSuccess() {
	if s.Monitoring != nil && s.Monitoring.StatsDClient != nil {
		key := constants.GetConfiguration + constants.DOT + constants.Success
		s.Monitoring.StatsDClient.Increment(key)
	}
}

func (s *Service) recordGetConfigurationFail() {
	if s.Monitoring != nil && s.Monitoring.StatsDClient != nil {
		key := constants.GetConfiguration + constants.DOT + constants.Fail
		s.Monitoring.StatsDClient.Increment(key)
	}
}

func (s *Service) GetConfiguration(w http.ResponseWriter, r *http.Request) {
	var configuration sch.Configuration

	vars := mux.Vars(r)
	appId := vars["app_id"]

	app, err := s.ClusterDao.GetApp(appId)

	switch {
	case err == gocql.ErrNotFound:
		er.Handle(w, r, er.NewError(er.InvalidAppId, errors.New(fmt.Sprintf("app id %s is not registered", appId))))
		s.recordGetConfigurationFail()

	case err != nil:
		er.Handle(w, r, er.NewError(er.DataFetchFailure, err))
		s.recordGetConfigurationFail()

	default:
		if configuration, err = s.ClusterDao.GetConfiguration(app.AppId); err != nil {
			er.Handle(w, r, er.NewError(er.DataNotFound, err))
			s.recordGetConfigurationFail()
			return
		}

		s.recordGetConfigurationSuccess()
		status := Status{
			StatusCode:    constants.SuccessCode201,
			StatusMessage: constants.Success,
			StatusType:    constants.Success,
			TotalCount:    1,
		}
		_ = json.NewEncoder(w).Encode(GetConfigurationResponse{
			Status: status,
			Data: GetConfigurationData{
				AppId:         app.AppId,
				Configuration: configuration,
			},
		})
	}

}
