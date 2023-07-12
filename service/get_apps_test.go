package service

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestService_GetApps(t *testing.T) {
	service := setupMocks()

	for _, test := range []struct {
		App    string
		Status int
	}{
		{
			"testGetAppsError",
			http.StatusInternalServerError,
		},
		{
			"testEmptyData",
			http.StatusNotFound,
		},
		{
			"test",
			http.StatusOK,
		},
		{
			"",
			http.StatusOK,
		},
	} {
		req, err := http.NewRequest("GET", "/goscheduler/apps", nil)
		if err != nil {
			t.Fatal(err)
		}

		q := req.URL.Query()
		q.Add("app_id", test.App)
		req.URL.RawQuery = q.Encode()

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(service.GetApps)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != test.Status {
			t.Errorf("handler returned wrong status code: got %v want %v", status, test.Status)
		}
	}
}
