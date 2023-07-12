package service

import (
	"encoding/json"
	"github.com/myntra/goscheduler/constants"
	"net/http"
)

type HealthCheckResponse struct {
	Status string `json:"status"`
	Code   int    `json:"code"`
}

func HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	jsonResponse, _ := json.Marshal(HealthCheckResponse{Status: constants.Success, Code: 200})
	w.Write(jsonResponse)
}
