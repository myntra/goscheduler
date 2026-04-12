package service

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gocql/gocql"
	"github.com/gorilla/mux"
	"github.com/myntra/goscheduler/dao"
	"github.com/myntra/goscheduler/store"
)

// MockScheduleDaoForAuditLogs provides custom behavior for audit log tests
type MockScheduleDaoForAuditLogs struct {
	dao.DummyScheduleDaoImpl
	auditLogs []store.AuditLog
}

var getAuditLogsCallCount int
var lastGetAuditLogsAppId string
var lastGetAuditLogsScheduleId gocql.UUID
var lastGetAuditLogsLimit int

func (m *MockScheduleDaoForAuditLogs) GetAuditLogs(appId string, scheduleId gocql.UUID, limit int) ([]store.AuditLog, error) {
	getAuditLogsCallCount++
	lastGetAuditLogsAppId = appId
	lastGetAuditLogsScheduleId = scheduleId
	lastGetAuditLogsLimit = limit
	return m.auditLogs, nil
}

func setupMocksForAuditLogs() *Service {
	sh := setupMocks()
	sh.ScheduleDao = &MockScheduleDaoForAuditLogs{
		auditLogs: []store.AuditLog{
			{
				ScheduleId: gocql.MustRandomUUID(),
				AppId:      "testApp",
				Action:     store.AuditActionCreate,
				Actor:      "user1",
				Timestamp:  time.Now(),
				Details:    `{"payload":"test"}`,
			},
			{
				ScheduleId: gocql.MustRandomUUID(),
				AppId:      "testApp",
				Action:     store.AuditActionDelete,
				Actor:      "user2",
				Timestamp:  time.Now().Add(-1 * time.Hour),
				Details:    `{"payload":"test"}`,
			},
		},
	}
	getAuditLogsCallCount = 0
	return sh
}

func TestService_GetAuditLogs(t *testing.T) {
	service := setupMocksForAuditLogs()

	tests := []struct {
		name           string
		appId          string
		scheduleId     string
		limit          string
		wantStatus     int
		wantCallCount  int
		description    string
	}{
		{
			name:          "AL-01_HappyPath",
			appId:         "testApp",
			scheduleId:    "55555555-5555-5555-5555-555555555555",
			limit:         "",
			wantStatus:    http.StatusOK,
			wantCallCount: 1,
			description:   "Valid request to get audit logs",
		},
		{
			name:          "AL-02_InvalidScheduleId",
			appId:         "testApp",
			scheduleId:    "invalid-uuid",
			limit:         "",
			wantStatus:    http.StatusBadRequest,
			wantCallCount: 0,
			description:   "Invalid UUID format for schedule ID",
		},
		{
			name:          "AL-03_WithLimit",
			appId:         "testApp",
			scheduleId:    "55555555-5555-5555-5555-555555555555",
			limit:         "5",
			wantStatus:    http.StatusOK,
			wantCallCount: 1,
			description:   "Get audit logs with custom limit",
		},
		{
			name:          "AL-04_InvalidLimit",
			appId:         "testApp",
			scheduleId:    "55555555-5555-5555-5555-555555555555",
			limit:         "invalid",
			wantStatus:    http.StatusOK,
			wantCallCount: 1,
			description:   "Invalid limit should default to 100",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Reset counter for each test
			getAuditLogsCallCount = 0

			url := "/goscheduler/apps/" + tc.appId + "/schedules/" + tc.scheduleId + "/audit-logs"
			if tc.limit != "" {
				url += "?limit=" + tc.limit
			}

			req, err := http.NewRequest("GET", url, bytes.NewBuffer(nil))
			if err != nil {
				t.Fatalf("could not create request: %v", err)
			}
			req = mux.SetURLVars(req, map[string]string{
				"appId":      tc.appId,
				"scheduleId": tc.scheduleId,
			})

			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(service.GetAuditLogs)
			handler.ServeHTTP(rr, req)

			if status := rr.Code; status != tc.wantStatus {
				t.Errorf("%s: unexpected status code: got %v, want %v. Description: %s",
					tc.name, status, tc.wantStatus, tc.description)
				t.Logf("Response body: %s", rr.Body.String())
			}

			if tc.wantCallCount != getAuditLogsCallCount {
				t.Errorf("%s: Expected GetAuditLogs to be called %d times, but was called %d times",
					tc.name, tc.wantCallCount, getAuditLogsCallCount)
			}

			// Verify response structure for successful requests
			if tc.wantStatus == http.StatusOK {
				var response AuditLogsResponse
				if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
					t.Errorf("%s: Failed to parse response: %v", tc.name, err)
				}
				if response.Status.StatusCode != http.StatusOK {
					t.Errorf("%s: Expected status code in response to be %d, got %d",
						tc.name, http.StatusOK, response.Status.StatusCode)
				}
				if response.Data.AppId != tc.appId {
					t.Errorf("%s: Expected appId in response to be %s, got %s",
						tc.name, tc.appId, response.Data.AppId)
				}
			}
		})
	}
}
