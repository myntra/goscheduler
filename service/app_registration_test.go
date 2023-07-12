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
			CronConfig:conf.CronConfig{
				App:      "Athena",

			},
		},
		supervisor:  new(cluster.DummySupervisor),
		clusterDao:  new(dao.DummyClusterDaoImpl),
		scheduleDao: new(dao.DummyScheduleDaoImpl),
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
