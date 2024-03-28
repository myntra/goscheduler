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

func (s *Service) CreateConfiguration(w http.ResponseWriter, r *http.Request) {
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
		s.recordRequestStatus(constants.CreateConfiguration, constants.Fail)
		return
	}

	app, err = s.ClusterDao.GetApp(appId)

	switch {
	case err == gocql.ErrNotFound:
		er.Handle(w, r, er.NewError(er.InvalidAppId, errors.New(fmt.Sprintf("app id %s is not registered", appId))))
		s.recordRequestStatus(constants.CreateConfiguration, constants.Fail)

	case err != nil:
		er.Handle(w, r, er.NewError(er.DataFetchFailure, err))
		s.recordRequestStatus(constants.CreateConfiguration, constants.Fail)

	default:
		if config, err = s.ClusterDao.CreateConfigurations(app.AppId, input); err != nil {
			er.Handle(w, r, er.NewError(er.DataPersistenceFailure, err))
			s.recordRequestStatus(constants.CreateConfiguration, constants.Fail)
			return
		}

		s.recordRequestStatus(constants.CreateConfiguration, constants.Success)
		status := Status{
			StatusCode:    constants.SuccessCode201,
			StatusMessage: constants.Success,
			StatusType:    constants.Success,
			TotalCount:    1,
		}
		_ = json.NewEncoder(w).Encode(CreateConfigurationResponse{
			Status: status,
			Data: CreateConfigurationData{
				AppId:         app.AppId,
				Configuration: config,
			},
		})
	}
}
