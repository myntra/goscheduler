// Copyright (c) 2023 Myntra Designs Private Limited.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package service

import (
	"encoding/json"
	"errors"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	e "github.com/myntra/goscheduler/cluster_entity"
	"github.com/myntra/goscheduler/constants"
	er "github.com/myntra/goscheduler/error"
	"github.com/myntra/goscheduler/store"
	"io/ioutil"
	"net/http"
	"strconv"
)

func validateApp(input store.App) error {
	return validateAppId(input.AppId)
}

func validateAppId(appId string) error {
	if len(appId) == 0 {
		return er.NewError(er.InvalidDataCode, errors.New("AppId cannot be empty"))
	}

	return nil
}

func (s *Service) Register(w http.ResponseWriter, r *http.Request) {
	var input store.App

	b, _ := ioutil.ReadAll(r.Body)
	err := json.Unmarshal(b, &input)
	if err != nil {
		s.recordRequestStatus(constants.RegisterApp, constants.Fail)
		er.Handle(w, r, er.NewError(er.UnmarshalErrorCode, err))
		return
	}

	input, err = s.RegisterApp(input)
	if err != nil {
		s.recordRequestStatus(constants.RegisterApp, constants.Fail)
		er.Handle(w, r, err.(er.AppError))
		return
	}

	s.recordRequestStatus(constants.RegisterApp, constants.Success)
	status := Status{StatusCode: constants.SuccessCode201, StatusMessage: constants.Success, StatusType: constants.Success, TotalCount: 1}
	_ = json.NewEncoder(w).Encode(CreateAppResponse{Status: status, Data: CreateAppData{AppId: input.AppId, Partitions: input.Partitions, Active: input.Active, Configuration: input.Configuration}})
}

func (s *Service) RegisterApp(input store.App) (store.App, error) {
	err := validateApp(input)
	if err != nil {
		return store.App{}, err
	}

	if input.Partitions == 0 {
		input.Partitions = s.Config.Poller.DefaultCount
	}

	err = s.ClusterDao.InsertApp(input)
	if err != nil {
		return store.App{}, er.NewError(er.DataPersistenceFailure, err)
	}

	glog.Infof("Creating entities for app %s", input.AppId)
	err = s.createEntities(input)
	if err != nil {
		return store.App{}, err
	}

	return input, nil
}

func (s *Service) createEntities(input store.App) error {
	for partition := uint32(0); partition < input.Partitions; partition++ {
		entity := e.EntityInfo{
			Id:      input.AppId + constants.PollerKeySep + strconv.Itoa(int(partition)),
			Node:    s.Config.Cluster.Address,
			Status:  0,
			History: "",
		}

		err := s.ClusterDao.CreateEntity(entity)
		if err != nil {
			return er.NewError(er.DataPersistenceFailure, err)
		}

		// We are calling boot entity with forward true. This is forward the request to the correct node
		// if the current node is not the node to start the entity.
		// TODO: Handle the error
		err = s.Supervisor.BootEntity(entity, true)
		if err != nil {
			return er.NewError(er.EntityBootFailed, err)
		}
	}

	return nil
}

func (s *Service) Deactivate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	appId := vars["appId"]

	err := s.DeactivateApp(appId)
	if err != nil {
		s.recordRequestStatus(constants.DeactivateApp, constants.Fail)
		er.Handle(w, r, err.(er.AppError))
		return
	}

	s.recordRequestStatus(constants.DeactivateApp, constants.Success)
	status := Status{StatusCode: constants.SuccessCode201, StatusMessage: constants.Success, StatusType: constants.Success}
	_ = json.NewEncoder(w).Encode(UpdateAppActiveStatusResponse{Status: status, Data: UpdateAppActiveStatusData{AppId: appId, Active: false}})
}

func (s *Service) DeactivateApp(appId string) error {
	err := validateAppId(appId)
	if err != nil {
		return err
	}

	app, err := s.ClusterDao.GetApp(appId)
	if err != nil {
		return er.NewError(er.InvalidAppId, errors.New("unregistered App"))
	}

	if !app.Active {
		return er.NewError(er.DeactivatedApp, errors.New("app is already deactivated"))
	}

	err = s.ClusterDao.UpdateAppActiveStatus(appId, false)
	if err != nil {
		return er.NewError(er.DataPersistenceFailure, err)
	}

	s.Supervisor.DeactivateApp(app)
	return nil
}

func (s *Service) Activate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	appId := vars["appId"]

	err := s.ActivateApp(appId)
	if err != nil {
		s.recordRequestStatus(constants.ActivateApp, constants.Fail)
		er.Handle(w, r, err.(er.AppError))
		return
	}

	s.recordRequestStatus(constants.ActivateApp, constants.Success)
	status := Status{StatusCode: constants.SuccessCode201, StatusMessage: constants.Success, StatusType: constants.Success}
	_ = json.NewEncoder(w).Encode(UpdateAppActiveStatusResponse{Status: status, Data: UpdateAppActiveStatusData{AppId: appId, Active: true}})
}

func (s *Service) ActivateApp(appId string) error {
	err := validateAppId(appId)
	if err != nil {
		return err
	}

	app, err := s.ClusterDao.GetApp(appId)
	if err != nil {
		return er.NewError(er.InvalidAppId, errors.New("unregistered App"))
	}

	if app.Active {
		return er.NewError(er.ActivatedApp, errors.New("app is already activated"))
	}

	err = s.ClusterDao.UpdateAppActiveStatus(appId, true)
	if err != nil {
		return er.NewError(er.DataPersistenceFailure, err)
	}

	s.Supervisor.ActivateApp(app)
	return nil
}
