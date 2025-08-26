package service

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gocql/gocql"
	"github.com/gorilla/mux"
	"github.com/myntra/goscheduler/dao"
	"github.com/myntra/goscheduler/store"
)

// MockScheduleDaoForUpdate provides custom behaviour for update-recurring-schedule tests.
type MockScheduleDaoForUpdate struct {
	dao.DummyScheduleDaoImpl
}

var updateRecurringScheduleCallCount int
var lastUpdateRecurringScheduleInput store.Schedule

func (m *MockScheduleDaoForUpdate) GetSchedule(uuid gocql.UUID) (store.Schedule, error) {
	switch uuid.String() {
	case "00000000-0000-0000-0000-000000000000":
		// U-05: Schedule not found
		return store.Schedule{}, gocql.ErrNotFound
	case "11111111-1111-1111-1111-111111111111":
		// Non-recurring schedule (no cron expression)
		return store.Schedule{
			ScheduleId:     uuid,
			AppId:          "testApp",
			CronExpression: "", // Empty = non-recurring
			Status:         store.Scheduled,
			PartitionId:    0,
		}, nil
	case "22222222-2222-2222-2222-222222222222":
		// U-05: Database error when fetching schedule
		return store.Schedule{}, errors.New("database connection error")
	case "33333333-3333-3333-3333-333333333333":
		// U-04: Schedule exists but app not found
		return store.Schedule{
			ScheduleId:     uuid,
			AppId:          "nonExistentApp",
			CronExpression: "0 0 * * *",
			Status:         store.Scheduled,
			PartitionId:    0,
		}, nil
	case "44444444-4444-4444-4444-444444444444":
		// U-09: For testing Cassandra timeout in UpdateRecurringSchedule
		return store.Schedule{
			ScheduleId:     uuid,
			AppId:          "timeoutApp",
			CronExpression: "0 0 * * *",
			Payload:        `{"test": "data"}`,
			PartitionId:    0,
			Callback: &store.HttpCallback{
				Type: "http",
				Details: store.Details{
					Url:    "http://example.com",
					Method: "POST",
				},
			},
			Status: store.Scheduled,
		}, nil
	default:
		// U-01: Valid recurring schedule for happy path
		return store.Schedule{
			ScheduleId:     uuid,
			AppId:          "testApp",
			Payload:        `{"foo":"bar"}`,
			CronExpression: "0 0 * * *",
			PartitionId:    0,
			Callback: &store.HttpCallback{
				Type: "http",
				Details: store.Details{
					Url:    "http://example.com",
					Method: "GET",
				},
			},
			Status: store.Scheduled,
		}, nil
	}
}

func (m *MockScheduleDaoForUpdate) UpdateRecurringSchedule(schedule store.Schedule) (store.Schedule, error) {
	updateRecurringScheduleCallCount++
	lastUpdateRecurringScheduleInput = schedule

	// U-09: Simulate Cassandra timeout
	if schedule.AppId == "timeoutApp" {
		return store.Schedule{}, gocql.ErrTimeoutNoResponse
	}

	return schedule, nil
}

// MockClusterDaoForUpdate to handle app-related test cases
type MockClusterDaoForUpdate struct {
	dao.DummyClusterDaoImpl
}

func (m *MockClusterDaoForUpdate) GetApp(appId string) (store.App, error) {
	switch appId {
	case "nonExistentApp":
		// U-04: App not found
		return store.App{}, gocql.ErrNotFound
	case "testApp", "timeoutApp":
		return store.App{
			AppId:      appId,
			Active:     true,
			Partitions: 1,
		}, nil
	default:
		return store.App{}, errors.New("unexpected app id")
	}
}

func setupMocksForUpdateRecurringSchedule() *Service {
	sh := setupMocks()
	sh.ScheduleDao = &MockScheduleDaoForUpdate{}
	sh.ClusterDao = &MockClusterDaoForUpdate{}
	updateRecurringScheduleCallCount = 0
	return sh
}

func TestService_UpdateRecurringSchedule(t *testing.T) {
	service := setupMocksForUpdateRecurringSchedule()

	tests := []struct {
		name        string
		testID      string
		scheduleID  string
		body        []byte
		wantStatus  int
		description string
	}{
		{
			name:        "U-01_HappyPath",
			testID:      "U-01",
			scheduleID:  "55555555-5555-5555-5555-555555555555",
			body:        []byte(`{"cronExpression":"*/10 * * * *","payload":"{\"updated\":\"value\"}"}`),
			wantStatus:  http.StatusOK,
			description: "Valid JSON; schedule exists & is recurring",
		},
		{
			name:        "U-02_MalformedJSON",
			testID:      "U-02",
			scheduleID:  "55555555-5555-5555-5555-555555555555",
			body:        []byte(`{bad json`),
			wantStatus:  http.StatusBadRequest,
			description: "Malformed JSON body",
		},
		{
			name:        "U-05_ScheduleNotFound",
			testID:      "U-05",
			scheduleID:  "00000000-0000-0000-0000-000000000000",
			body:        []byte(`{}`),
			wantStatus:  http.StatusNotFound,
			description: "Schedule does not exist",
		},
		{
			name:        "NonRecurringSchedule",
			testID:      "Related",
			scheduleID:  "11111111-1111-1111-1111-111111111111",
			body:        []byte(`{"cronExpression":"*/10 * * * *"}`),
			wantStatus:  http.StatusUnprocessableEntity,
			description: "Schedule is not recurring",
		},
		{
			name:        "U-05_DatabaseError",
			testID:      "U-05",
			scheduleID:  "22222222-2222-2222-2222-222222222222",
			body:        []byte(`{}`),
			wantStatus:  http.StatusInternalServerError,
			description: "DAO failure when reading schedule",
		},
		{
			name:        "U-04_AppNotFound",
			testID:      "U-04",
			scheduleID:  "33333333-3333-3333-3333-333333333333",
			body:        []byte(`{}`),
			wantStatus:  http.StatusBadRequest,
			description: "App not found",
		},
		{
			name:        "U-03_AppIdMismatch",
			testID:      "U-03",
			scheduleID:  "55555555-5555-5555-5555-555555555555",
			body:        []byte(`{"appId":"differentApp","cronExpression":"*/10 * * * *"}`),
			wantStatus:  http.StatusBadRequest,
			description: "AppId in body doesn't match schedule's appId",
		},
		{
			name:        "U-06_InvalidCronExpression",
			testID:      "U-06",
			scheduleID:  "55555555-5555-5555-5555-555555555555",
			body:        []byte(`{"cronExpression":"* * *"}`),
			wantStatus:  http.StatusUnprocessableEntity,
			description: "Invalid cron expression",
		},
		{
			name:        "U-09_CassandraTimeout",
			testID:      "U-09",
			scheduleID:  "44444444-4444-4444-4444-444444444444",
			body:        []byte(`{"cronExpression":"*/10 * * * *"}`),
			wantStatus:  http.StatusInternalServerError,
			description: "Cassandra timeout during update",
		},
		{
			name:        "InvalidScheduleUUID",
			testID:      "Related",
			scheduleID:  "not-a-valid-uuid",
			body:        []byte(`{}`),
			wantStatus:  http.StatusBadRequest,
			description: "Invalid UUID format",
		},
		{
			name:        "UpdateCallbackType",
			testID:      "Related",
			scheduleID:  "55555555-5555-5555-5555-555555555555",
			body:        []byte(`{"callback":{"type":"http","details":{"url":"http://newurl.com","method":"POST","headers":{"X-Custom":"header"}}}}`),
			wantStatus:  http.StatusOK,
			description: "Update callback_type and details",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Reset counter for each test
			updateRecurringScheduleCallCount = 0

			req, err := http.NewRequest("PUT", "/goscheduler/schedules/{scheduleId}/updateRecurringSchedule", bytes.NewBuffer(tc.body))
			if err != nil {
				t.Fatalf("could not create request: %v", err)
			}
			req = mux.SetURLVars(req, map[string]string{"scheduleId": tc.scheduleID})

			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(service.UpdateRecurringSchedule)
			handler.ServeHTTP(rr, req)

			if status := rr.Code; status != tc.wantStatus {
				t.Errorf("%s: unexpected status code: got %v, want %v. Description: %s",
					tc.testID, status, tc.wantStatus, tc.description)
				t.Logf("Response body: %s", rr.Body.String())
			}

			// Additional validations for specific test cases
			if tc.testID == "U-01" && updateRecurringScheduleCallCount != 1 {
				t.Errorf("U-01: Expected UpdateRecurringSchedule to be called once, but was called %d times",
					updateRecurringScheduleCallCount)
			}

			if tc.name == "UpdateCallbackType" && updateRecurringScheduleCallCount == 1 {
				// Verify callback was properly updated
				if lastUpdateRecurringScheduleInput.GetCallBackType() != "http" {
					t.Errorf("Expected callback type to be 'http', got '%s'",
						lastUpdateRecurringScheduleInput.GetCallBackType())
				}
			}
		})
	}
}
