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
