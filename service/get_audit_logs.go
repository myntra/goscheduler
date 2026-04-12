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
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gocql/gocql"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/myntra/goscheduler/constants"
	er "github.com/myntra/goscheduler/error"
	sch "github.com/myntra/goscheduler/store"
)

// AuditLogsResponse represents the response structure for audit logs
type AuditLogsResponse struct {
	Status Status           `json:"status"`
	Data   AuditLogsData    `json:"data"`
}

// AuditLogsData contains the list of audit logs
type AuditLogsData struct {
	AuditLogs []sch.AuditLog `json:"auditLogs"`
	AppId     string         `json:"appId"`
	ScheduleId string        `json:"scheduleId"`
}

// GetAuditLogs retrieves audit logs for a specific schedule
func (s *Service) GetAuditLogs(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	appId := vars["appId"]
	scheduleIdStr := vars["scheduleId"]

	// Parse schedule ID
	scheduleId, err := gocql.ParseUUID(scheduleIdStr)
	if err != nil {
		glog.Errorf("GetAuditLogs: Invalid schedule ID %s: %v", scheduleIdStr, err)
		s.recordRequestStatus(constants.GetAuditLogs, constants.Fail)
		er.Handle(w, r, er.NewError(er.InvalidDataCode, err))
		return
	}

	// Get limit from query parameter, default to 100
	limit := 100
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	// Fetch audit logs
	auditLogs, err := s.ScheduleDao.GetAuditLogs(appId, scheduleId, limit)
	if err != nil {
		glog.Errorf("GetAuditLogs: Error fetching audit logs: %v", err)
		s.recordRequestStatus(constants.GetAuditLogs, constants.Fail)
		er.Handle(w, r, er.NewError(er.DataFetchFailure, err))
		return
	}

	s.recordRequestStatus(constants.GetAuditLogs, constants.Success)

	status := Status{
		StatusCode:    constants.SuccessCode200,
		StatusMessage: constants.Success,
		StatusType:    constants.Success,
		TotalCount:    len(auditLogs),
	}

	data := AuditLogsData{
		AuditLogs:  auditLogs,
		AppId:      appId,
		ScheduleId: scheduleIdStr,
	}

	_ = json.NewEncoder(w).Encode(AuditLogsResponse{
		Status: status,
		Data:   data,
	})
}
