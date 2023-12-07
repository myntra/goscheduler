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
	"io/ioutil"
	"net/http"
)

func (s *Service) recordUpdateConfigurationSuccess() {
	if s.Monitoring != nil && s.Monitoring.StatsDClient != nil {
		key := constants.UpdateConfiguration + constants.DOT + constants.Success
		s.Monitoring.StatsDClient.Increment(key)
	}
}

func (s *Service) recordUpdateConfigurationFail() {
	if s.Monitoring != nil && s.Monitoring.StatsDClient != nil {
		key := constants.UpdateConfiguration + constants.DOT + constants.Fail
		s.Monitoring.StatsDClient.Increment(key)
	}
}

func (s *Service) UpdateConfiguration(w http.ResponseWriter, r *http.Request) {
	var input sch.Configuration
	var config sch.Configuration
	var err error
	var app sch.App

	vars := mux.Vars(r)
	appId := vars["app_id"]
	b, _ := ioutil.ReadAll(r.Body)
	err = json.Unmarshal(b, &input)

	if err != nil {
		er.Handle(w, r, er.NewError(er.UnmarshalErrorCode, err))
		s.recordUpdateConfigurationFail()
		return
	}

	app, err = s.ClusterDao.GetApp(appId)

	switch {
	case err == gocql.ErrNotFound:
		er.Handle(w, r, er.NewError(er.InvalidAppId, errors.New(fmt.Sprintf("app %+v is not registered", app))))
		s.recordUpdateConfigurationFail()

	case err != nil:
		er.Handle(w, r, er.NewError(er.DataFetchFailure, err))
		s.recordUpdateConfigurationFail()

	default:
		if config, err = s.ClusterDao.UpdateConfiguration(app.AppId, input); err != nil {
			er.Handle(w, r, er.NewError(er.DataPersistenceFailure, err))
			s.recordUpdateConfigurationFail()
			return
		}

		s.recordUpdateConfigurationSuccess()
		status := Status{
			StatusCode:    constants.SuccessCode201,
			StatusMessage: constants.Success,
			StatusType:    constants.Success,
			TotalCount:    1,
		}
		_ = json.NewEncoder(w).Encode(UpdateConfigurationResponse{
			Status: status,
			Data: UpdateConfigurationData{
				AppId:         app.AppId,
				Configuration: config,
			},
		})
	}
}
