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
	"bytes"
	"github.com/gorilla/mux"
	"github.com/myntra/goscheduler/cluster"
	"github.com/myntra/goscheduler/conf"
	"github.com/myntra/goscheduler/constants"
	"github.com/myntra/goscheduler/dao"
	"github.com/myntra/goscheduler/store"
	"net/http"
	"net/http/httptest"
	"testing"
)

func setupMocks() *Service {
	store.Registry[constants.DefaultCallback] = func() store.Callback { return &store.HttpCallback{} }
	return &Service{
		Config: &conf.Configuration{
			Cluster: conf.ClusterConfig{
				Address: "127.0.0.1:9091",
			},
			CronConfig: conf.CronConfig{
				App: "Athena",
			},
		},
		Supervisor:  new(cluster.DummySupervisor),
		ClusterDao:  new(dao.DummyClusterDaoImpl),
		ScheduleDao: new(dao.DummyScheduleDaoImpl),
	}
}

func TestService_Register(t *testing.T) {
	service := setupMocks()
	for _, test := range []struct {
		Byte   []byte
		Status int
	}{
		{
			[]byte(`{
			   "appId": "test",
			   "partitions": 1,
			   "active": true
			}`),
			http.StatusOK,
		},
		{
			[]byte(`{
			   "appId": "testInsertError",
			   "partitions": 1,
			   "active": true
			}`),
			http.StatusInternalServerError,
		},
		{
			[]byte(`{
			   "appId": "testCreateEntityError",
			   "partitions": 1,
			   "active": true
			}`),
			http.StatusInternalServerError,
		},
		{
			[]byte(`{
			   "appId": "",
			   "partitions": 1,
			   "active": true
			}`),
			http.StatusBadRequest,
		},
		{
			[]byte(nil),
			http.StatusBadRequest,
		},
	} {
		req, err := http.NewRequest("GET", "/goscheduler/apps", bytes.NewBuffer(test.Byte))
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(service.Register)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != test.Status {
			t.Errorf("handler returned wrong status code: got %v want %v", status, test.Status)
		}
	}
}

func TestService_DeactivateApp(t *testing.T) {
	service := setupMocks()

	req, err := http.NewRequest("GET", "/goscheduler/apps/:appId/deactivate", nil)
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		App    string
		Status int
	}{
		{
			"",
			http.StatusBadRequest,
		},
		{
			"testDeactivated",
			http.StatusBadRequest,
		},
		{
			"testGetAppError",
			http.StatusBadRequest,
		},
		{
			"testUpdateAppActiveStatusError",
			http.StatusInternalServerError,
		},
		{
			"testActivated",
			http.StatusOK,
		},
	} {
		vars := map[string]string{
			"appId": test.App,
		}

		req = mux.SetURLVars(req, vars)

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(service.Deactivate)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != test.Status {
			t.Errorf("handler returned wrong status code: got %v want %v", status, test.Status)
		}
	}
}

func TestService_ActivateApp(t *testing.T) {
	service := setupMocks()

	req, err := http.NewRequest("GET", "/goscheduler/apps/:app_id/activate", nil)
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		App    string
		Status int
	}{
		{
			"",
			http.StatusBadRequest,
		},
		{
			"testActivated",
			http.StatusBadRequest,
		},
		{
			"testGetAppError",
			http.StatusBadRequest,
		},
		{
			"testDeactivatedUpdateAppActiveStatus",
			http.StatusInternalServerError,
		},
		{
			"testDeactivated",
			http.StatusOK,
		},
	} {
		vars := map[string]string{
			"appId": test.App,
		}

		req = mux.SetURLVars(req, vars)

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(service.Activate)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != test.Status {
			t.Errorf("handler returned wrong status code: got %v want %v", status, test.Status)
		}
	}
}
