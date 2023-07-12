package service

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestService_GetCronSchedules(t *testing.T) {
	service := setupMocks()

	for _, test := range []struct {
		App    string
		Status int
	}{
		{
			"testGetAppError",
			http.StatusInternalServerError,
		},
		{
			"testGetCronSchedulesError",
			http.StatusNotFound,
		},
		{
			"test",
			http.StatusOK,
		},
	} {

		req, err := http.NewRequest("GET", "/goscheduler/crons/schedules", nil)
		if err != nil {
			t.Fatal(err)
		}

		q := req.URL.Query()
		q.Add("app_id", test.App)
		req.URL.RawQuery = q.Encode()

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(service.GetCronSchedules)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != test.Status {
			t.Errorf("handler returned wrong status code: got %v want %v", status, test.Status)
		}
	}
}
