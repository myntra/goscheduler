package service

import (
	"github.com/gorilla/mux"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestService_CancelSchedule(t *testing.T) {
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

		req, err := http.NewRequest("DELETE", "/goscheduler/schedules/:scheduleId", nil)
		if err != nil {
			t.Fatal(err)
		}

		vars := map[string]string{
			"scheduleId": test.UUID,
		}

		req = mux.SetURLVars(req, vars)

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(service.CancelSchedule)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != test.Status {
			t.Errorf("handler returned wrong status code: got %v want %v", status, test.Status)
		}
	}
}
