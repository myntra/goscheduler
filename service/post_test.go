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
	"fmt"
	"github.com/gocql/gocql"
	"github.com/gorilla/mux"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestPost(t *testing.T) {
	service := setupMocks()

	for _, test := range []struct {
		uuid   string
		body   []byte
		Status int
	}{

		{
			gocql.TimeUUID().String(),
			[]byte(fmt.Sprintf(`{"AppId": "testAppNotFound", "callback": {"type": "http", "details": {"url": "https://dummy.url", "method": "POST", "headers": {"header": "value"}}}, "ScheduleTime":%d, "Payload":"{}"}`, time.Now().Add(90000000000).Unix())),
			http.StatusBadRequest,
		},
		{
			gocql.TimeUUID().String(),
			[]byte(fmt.Sprintf(`{"AppId": "testAppNotActive", "callback": {"type": "http", "details": {"url": "https://dummy.url", "method": "POST", "headers": {"header": "value"}}}, "ScheduleTime":%d, "Payload":"{}"}`, time.Now().Add(90000000000).Unix())),
			http.StatusBadRequest,
		},
		{
			gocql.TimeUUID().String(),
			[]byte(fmt.Sprintf(`{"AppId": "test", "callback": {"type": "http", "details": {"url": "https://dummy.url", "method": "POST", "headers": {"header": "value"}}}, "ScheduleTime":%d, "Payload":"{}"}`, time.Now().Add(90000000000).Unix())),
			http.StatusOK,
		},
		{
			gocql.TimeUUID().String(),
			[]byte(fmt.Sprintf(`{"AppId": "test", "callback": {"type": "http", "details": {"url": "https://dummy.url", "method": "POST", "headers": {"header": "value"}}}, "CronExpression": "%s", "Payload":"{}"}`, "*/1 * * * *")),
			http.StatusOK,
		},
		{
			gocql.TimeUUID().String(),
			[]byte(fmt.Sprintf(`{"AppId": "test", "callback": {"type": "http", "details": {"url": "https://dummy.url", "method": "POST", "headers": {"header": "value"}}}, "CronExpression": "%s", "Payload":"{}"}`, "*/1 * 1 * * 12")),
			http.StatusBadRequest,
		},
		{
			gocql.TimeUUID().String(),
			[]byte(fmt.Sprintf(`{"AppId": "createScheduleFailureApp", "callback": {"type": "http", "details": {"url": "https://dummy.url", "method": "POST", "headers": {"header": "value"}}}, "ScheduleTime":%d, "Payload":"{}"}`, time.Now().Add(90000000000).Unix())),
			http.StatusInternalServerError,
		},
	} {

		req, err := http.NewRequest("POST", "/goscheduler/schedules", bytes.NewBuffer(test.body))
		if err != nil {
			t.Fatal(err)
		}

		vars := map[string]string{
			"scheduleId": test.uuid,
		}

		req.Header.Add("x-myntra-client-id", "test")
		req = mux.SetURLVars(req, vars)

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(service.Post)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != test.Status {
			t.Errorf("handler returned wrong status code: got %v want %v", status, test.Status)
		}
	}
}
