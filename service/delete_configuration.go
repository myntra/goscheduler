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

func (s *Service) recordDeleteConfigurationSuccess() {
	if s.Monitoring != nil && s.Monitoring.StatsDClient != nil {
		key := constants.DeleteConfiguration + constants.DOT + constants.Success
		s.Monitoring.StatsDClient.Increment(key)
	}
}

func (s *Service) recordDeleteConfigurationFail() {
	if s.Monitoring != nil && s.Monitoring.StatsDClient != nil {
		key := constants.DeleteConfiguration + constants.DOT + constants.Fail
		s.Monitoring.StatsDClient.Increment(key)
	}
}

func (s *Service) DeleteConfiguration(w http.ResponseWriter, r *http.Request) {
	var err error
	var app sch.App
	var config sch.Configuration

	vars := mux.Vars(r)
	appId := vars["app_id"]

	app, err = s.ClusterDao.GetApp(appId)

	switch {
	case err == gocql.ErrNotFound:
		er.Handle(w, r, er.NewError(er.InvalidAppId, errors.New(fmt.Sprintf("app %+v is not registered", app))))
		s.recordDeleteConfigurationFail()

	case err != nil:
		er.Handle(w, r, er.NewError(er.DataFetchFailure, err))
		s.recordDeleteConfigurationFail()

	default:
		if config, err = s.ClusterDao.DeleteConfiguration(app.AppId); err != nil {
			er.Handle(w, r, er.NewError(er.DataPersistenceFailure, err))
			s.recordDeleteConfigurationFail()
			return
		}

		s.recordDeleteConfigurationSuccess()
		status := Status{
			StatusCode:    constants.SuccessCode201,
			StatusMessage: constants.Success,
			StatusType:    constants.Success,
			TotalCount:    1,
		}
		_ = json.NewEncoder(w).Encode(DeleteConfigurationResponse{
			Status: status,
			Data: DeleteConfigurationData{
				AppId:         app.AppId,
				Configuration: config,
			},
		})
	}

}
