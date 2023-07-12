package service

import (
	"github.com/gocql/gocql"
	"github.com/gorilla/mux"
	"github.com/myntra/goscheduler/cluster"
	"github.com/myntra/goscheduler/dao"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestService_Runs(t *testing.T) {
	service := &Service{
		supervisor:  new(cluster.DummySupervisor),
		clusterDao:  new(dao.DummyClusterDaoImpl),
		scheduleDao: new(dao.DummyScheduleDaoImpl),
	}

	for _, test := range []struct {
		uuid   string
		when   string
		Status int
	}{
		{

			gocql.TimeUUID().String(),
			"past",
			http.StatusOK,
		},
		{

			gocql.TimeUUID().String(),
			"future",
			http.StatusOK,
		},
		{

			gocql.UUID{}.String(),
			"",
			http.StatusInternalServerError,
		},
		{
			"00000000-0000-0000-0000-000000000000",
			"",
			http.StatusInternalServerError,
		},
		{
			"00000000-0000-0000-0000-000000000001",
			"",
			http.StatusNotFound,
		},
		{
			gocql.TimeUUID().String(),
			"",
			http.StatusOK,
		},
	} {

		req, err := http.NewRequest("GET", "/goscheduler/schedules/{scheduleId}/runs", nil)
		if err != nil {
			t.Fatal(err)
		}

		vars := map[string]string{
			"scheduleId": test.uuid,
		}

		req = mux.SetURLVars(req, vars)

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(service.GetRuns)
		handler.ServeHTTP(rr, req)

		q := req.URL.Query()
		q.Add("when", test.when)
		req.URL.RawQuery = q.Encode()

		if status := rr.Code; status != test.Status {
			t.Errorf("handler returned wrong status code: got %v want %v", status, test.Status)
		}
	}
}
