package service

import (
	"github.com/gorilla/mux"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestService_GetConfiguration(t *testing.T) {
	service := setupMocks()

	for _, test := range []struct {
		App    string
		Status int
	}{
		{
			"testGetAppErrorNotFound",
			http.StatusBadRequest,
		},
		{
			"testGetAppError",
			http.StatusInternalServerError,
		},
		{
			"testGetConfigurationError",
			http.StatusNotFound,
		},
		{
			"test",
			http.StatusOK,
		},
	} {
		req, err := http.NewRequest("GET", "/myss/app/:app_id/configuration", nil)
		if err != nil {
			t.Fatal(err)
		}

		vars := map[string]string{
			"app_id": test.App,
		}

		req = mux.SetURLVars(req, vars)

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(service.GetConfiguration)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != test.Status {
			t.Errorf("handler returned wrong status code: got %v want %v", status, test.Status)
		}
	}
}
