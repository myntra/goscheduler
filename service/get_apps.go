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
	"fmt"
	"github.com/myntra/goscheduler/constants"
	er "github.com/myntra/goscheduler/error"
	"github.com/myntra/goscheduler/store"
	"net/http"
)

func parseAppQueryParams(r *http.Request) string {
	query := r.URL.Query()
	return query.Get("app_id")
}

func (s *Service) GetApps(w http.ResponseWriter, r *http.Request) {
	appId := parseAppQueryParams(r)

	apps, err := s.FetchApps(appId)
	if err != nil {
		s.recordRequestStatus(constants.GetApps, constants.Fail)
		er.Handle(w, r, err.(er.AppError))
		return
	}

	s.recordRequestStatus(constants.GetApps, constants.Success)

	status := Status{
		StatusCode:    constants.SuccessCode200,
		StatusMessage: constants.Success,
		StatusType:    constants.Success,
		TotalCount:    len(apps),
	}

	data := GetAppsData{
		Apps: apps,
	}

	_ = json.NewEncoder(w).Encode(
		GetAppsResponse{
			Status: status,
			Data:   data,
		})
}

func (s *Service) FetchApps(appId string) ([]store.App, error) {
	switch apps, err := s.ClusterDao.GetApps(appId); {
	case err != nil:
		return []store.App{}, er.NewError(er.DataFetchFailure, err)
	case len(apps) == 0:
		return []store.App{}, er.NewError(er.DataNotFound, errors.New(fmt.Sprint("No app found")))
	default:
		return apps, nil
	}
}
