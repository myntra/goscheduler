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
