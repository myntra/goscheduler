package service

import (
	"bytes"
	"github.com/gorilla/mux"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateConfiguration(t *testing.T) {
	service := setupMocks()
	for _, test := range []struct {
		App    string
		Byte   []byte
		Status int
	}{
		{
			"test",
			[]byte(nil),
			http.StatusBadRequest,
		},
		{
			"testGetAppErrorNotFound",
			[]byte(`{
			   "configuration": {
				   "futureScheduleCreationPeriod": 30,
				   "firedScheduleRetentionPeriod": 7,
				   "payloadSize": 204800,
				   "httpRetries": 3,
				   "httpTimeout": 1000
			   }
			}`),
			http.StatusBadRequest,
		},
		{
			"testGetAppError",
			[]byte(`{
			   "configuration": {
				   "futureScheduleCreationPeriod": 30,
				   "firedScheduleRetentionPeriod": 7,
				   "payloadSize": 204800,
				   "httpRetries": 3,
				   "httpTimeout": 1000
			   }
			}`),
			http.StatusInternalServerError,
		},
		{
			"testCreateConfigurationsError",
			[]byte(`{
			   "configuration": {
				   "futureScheduleCreationPeriod": 30,
				   "firedScheduleRetentionPeriod": 7,
				   "payloadSize": 204800,
				   "httpRetries": 3,
				   "httpTimeout": 1000
			   }
			}`),
			http.StatusInternalServerError,
		},
		{
			"test",
			[]byte(`{
			   "configuration": {
				   "futureScheduleCreationPeriod": 30,
				   "firedScheduleRetentionPeriod": 7,
				   "payloadSize": 204800,
				   "httpRetries": 3,
				   "httpTimeout": 1000
			   }
			}`),
			http.StatusOK,
		},
	} {
		req, err := http.NewRequest("GET", "/myss/app/:app_id/configuration", bytes.NewBuffer(test.Byte))
		if err != nil {
			t.Fatal(err)
		}

		vars := map[string]string{
			"app_id": test.App,
		}

		req = mux.SetURLVars(req, vars)

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(service.CreateConfiguration)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != test.Status {
			t.Errorf("handler returned wrong status code: got %v want %v", status, test.Status)
		}
	}
}
