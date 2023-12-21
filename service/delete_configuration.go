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
		s.recordRequestStatus(constants.DeleteConfiguration, constants.Fail)

	case err != nil:
		er.Handle(w, r, er.NewError(er.DataFetchFailure, err))
		s.recordRequestStatus(constants.DeleteConfiguration, constants.Fail)

	default:
		if config, err = s.ClusterDao.DeleteConfiguration(app.AppId); err != nil {
			er.Handle(w, r, er.NewError(er.DataPersistenceFailure, err))
			s.recordRequestStatus(constants.DeleteConfiguration, constants.Fail)
			return
		}

		s.recordRequestStatus(constants.DeleteConfiguration, constants.Success)
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
