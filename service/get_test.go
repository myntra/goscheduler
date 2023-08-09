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
	"github.com/gorilla/mux"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestService_GetSchedule(t *testing.T) {
	service := setupMocks()

	for _, test := range []struct {
		UUID   string
		Status int
	}{
		{
			"00000000-0000-0000-0000",
			http.StatusBadRequest,
		},
		{
			"00000000-0000-0000-0000-000000000000",
			http.StatusNotFound,
		},
		{
			"84d0d5b8-d953-11ed-a827-aa665a372253",
			http.StatusInternalServerError,
		},
		{
			"589bb372-d4b3-11ed-92b5-acde48001122",
			http.StatusOK,
		},
	} {

		req, err := http.NewRequest("GET", "/goscheduler/schedules/:scheduleId", nil)
		if err != nil {
			t.Fatal(err)
		}

		vars := map[string]string{
			"scheduleId": test.UUID,
		}

		req = mux.SetURLVars(req, vars)

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(service.Get)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != test.Status {
			t.Errorf("handler returned wrong status code: got %v want %v", status, test.Status)
		}
	}
}
