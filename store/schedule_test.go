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

package store

import (
	"errors"
	"github.com/gocql/gocql"
	"github.com/golang/mock/gomock"
	"testing"
	"time"
)

type MockCallback struct {
	Field string
}

func (m *MockCallback) GetType() string {
	//TODO implement me
	panic("implement me")
}

func (m *MockCallback) GetDetails() (string, error) {
	//TODO implement me
	panic("implement me")
}

func (m *MockCallback) Invoke(wrapper ScheduleWrapper) error {
	//TODO implement me
	panic("implement me")
}

func (m *MockCallback) Validate() error {
	switch m.Field {
	case "success":
		return nil
	default:
		return errors.New("error in validation")
	}
}

func (m *MockCallback) UnmarshalJSON(bytes []byte) error {
	//TODO implement me
	panic("implement me")
}

func (m *MockCallback) Marshal(mm map[string]interface{}) error {
	m.Field = "test"
	return nil
}

func TestCreateScheduleFromCassandraMap(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("test empty map", func(t *testing.T) {
		s := &Schedule{}
		err := s.CreateScheduleFromCassandraMap(map[string]interface{}{})
		if err != nil {
			t.Fatal("error should be nil for empty map")
		}
	})

	t.Run("test wrong callback type", func(t *testing.T) {
		s := &Schedule{}
		err := s.CreateScheduleFromCassandraMap(map[string]interface{}{
			"app_id":              "test-app-id",
			"partition_id":        1,
			"callback_type":       "nonexistent",
			"payload":             "test-payload",
			"schedule_time_group": time.Now(),
			"schedule_time":       time.Now(),
			"schedule_id":         gocql.TimeUUID(),
		})
		if err == nil || err.Error() != "wrong callback type" {
			t.Fatal("error should be 'wrong callback type' for wrong callback type")
		}
	})

	t.Run("test successful map parsing", func(t *testing.T) {
		mockCallback := &MockCallback{
			Field: "test",
		}
		Registry["mock"] = func() Callback {
			return mockCallback
		}

		m := map[string]interface{}{
			"app_id":              "test-app-id",
			"partition_id":        1,
			"callback_type":       "mock",
			"payload":             "test-payload",
			"schedule_time_group": time.Now(),
			"schedule_time":       time.Now(),
			"schedule_id":         gocql.TimeUUID(),
		}

		s := &Schedule{}
		err := s.CreateScheduleFromCassandraMap(m)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestValidateSchedule(t *testing.T) {
	t.Run("valid non-recurring schedule", func(t *testing.T) {
		s := &Schedule{
			AppId:          "test-app-id",
			Payload:        "test-payload",
			Callback:       &MockCallback{Field: "success"},
			ScheduleTime:   time.Now().Unix(),
			CronExpression: "*/5 * * * *",
		}

		errs := s.ValidateSchedule()
		if len(errs) != 0 {
			t.Fatalf("expected no errors, got %v", errs)
		}
	})

	t.Run("invalid app in schedule", func(t *testing.T) {
		s := &Schedule{
			AppId:          "",
			Payload:        "test-payload",
			Callback:       &MockCallback{Field: "success"},
			ScheduleTime:   time.Now().Unix(),
			CronExpression: "*/5 * * * *",
		}

		errs := s.ValidateSchedule()
		if len(errs) == 0 {
			t.Fatal("expected errors, got none")
		}
	})

	t.Run("invalid payload in schedule", func(t *testing.T) {
		s := &Schedule{
			AppId:          "appId",
			Payload:        "",
			Callback:       &MockCallback{Field: "success"},
			ScheduleTime:   time.Now().Unix(),
			CronExpression: "*/5 * * * *",
		}

		errs := s.ValidateSchedule()
		if len(errs) == 0 {
			t.Fatal("expected errors, got none")
		}
	})

	t.Run("invalid schedule time in schedule", func(t *testing.T) {
		s := &Schedule{
			AppId:        "appId",
			Payload:      "{}",
			Callback:     &MockCallback{Field: "success"},
			ScheduleTime: time.Now().Unix() - 100,
		}

		errs := s.ValidateSchedule()
		if len(errs) == 0 {
			t.Fatal("expected errors, got none")
		}
	})

	t.Run("invalid cron expression in schedule", func(t *testing.T) {
		s := &Schedule{
			AppId:          "appId",
			Payload:        "{}",
			Callback:       &MockCallback{Field: "success"},
			ScheduleTime:   time.Now().Unix(),
			CronExpression: "*/5 * * * * *",
		}

		errs := s.ValidateSchedule()
		if len(errs) == 0 {
			t.Fatal("expected errors, got none")
		}
	})

	t.Run("invalid callback in schedule", func(t *testing.T) {
		s := &Schedule{
			AppId:          "appId",
			Payload:        "{}",
			Callback:       &MockCallback{Field: "fail"},
			ScheduleTime:   time.Now().Unix(),
			CronExpression: "*/5 * * * *",
		}

		errs := s.ValidateSchedule()
		if len(errs) == 0 {
			t.Fatal("expected errors, got none")
		}
	})

	t.Run("valid recurring schedule", func(t *testing.T) {
		s := &Schedule{
			AppId:          "test-app-id",
			Payload:        "test-payload",
			Callback:       &MockCallback{Field: "success"},
			CronExpression: "* * * * *",
		}

		errs := s.ValidateSchedule()
		if len(errs) != 0 {
			t.Fatalf("expected no errors, got %v", errs)
		}
	})

	t.Run("valid one time schedule", func(t *testing.T) {
		s := &Schedule{
			AppId:        "test-app-id",
			Payload:      "test-payload",
			Callback:     &MockCallback{Field: "success"},
			ScheduleTime: time.Now().Unix() + 100,
		}

		errs := s.ValidateSchedule()
		if len(errs) != 0 {
			t.Fatalf("expected no errors, got %v", errs)
		}
	})
}

// Test for SetFields function
func TestSetFields(t *testing.T) {
	schedule := &Schedule{}
	app := App{
		AppId:      "test",
		Partitions: 10,
		Active:     false,
	}

	schedule.SetFields(app)

	if schedule.PartitionId < 0 || schedule.PartitionId >= int(app.Partitions) {
		t.Errorf("Expected PartitionId in range [0, %d) but got %d", app.Partitions, schedule.PartitionId)
	}
}

func TestCloneAsOneTime(t *testing.T) {
	now := time.Now()

	// creating initial schedule for test
	initialSchedule := Schedule{
		AppId:        "testAppId",
		ScheduleTime: now.Unix(),
		Callback:     &MockCallback{Field: "success"},
		Payload:      "testPayload",
	}

	// cloning the initial schedule
	clonedSchedule := initialSchedule.CloneAsOneTime(now)

	if clonedSchedule.ScheduleGroup != now.Unix() {
		t.Errorf("Expected ScheduleGroup %v, but got %v", now.Unix(), clonedSchedule.ScheduleGroup)
	}

	if clonedSchedule.ScheduleTime != now.Unix() {
		t.Errorf("Expected ScheduleTime %v, but got %v", now.Unix(), clonedSchedule.ScheduleTime)
	}

	if clonedSchedule.AppId != initialSchedule.AppId {
		t.Errorf("Expected AppId %s, but got %s", initialSchedule.AppId, clonedSchedule.AppId)
	}

	if clonedSchedule.Callback != initialSchedule.Callback {
		t.Errorf("Expected Callback %s, but got %s", initialSchedule.Callback, clonedSchedule.Callback)
	}

	if clonedSchedule.Payload != initialSchedule.Payload {
		t.Errorf("Expected Payload %s, but got %s", initialSchedule.Payload, clonedSchedule.Payload)
	}

	if clonedSchedule.ParentScheduleId != initialSchedule.ScheduleId {
		t.Errorf("Expected ParentScheduleId %s, but got %s", initialSchedule.ScheduleId, clonedSchedule.ParentScheduleId)
	}
}

func TestSetUnknownStatus(t *testing.T) {
	now := time.Now()

	// Scenario: The callback time is yet to come
	scheduleFuture := &Schedule{
		ScheduleGroup: now.Add(2 * time.Minute).Unix(),
	}

	scheduleFuture.SetUnknownStatus(30)

	if scheduleFuture.Status != Scheduled {
		t.Errorf("Expected status %s, got %s", Scheduled, scheduleFuture.Status)
	}

	// Scenario: The callback time has already expired
	schedulePast := &Schedule{
		ScheduleGroup: now.Add(-2 * time.Minute).Unix(),
	}

	schedulePast.SetUnknownStatus(30)

	if schedulePast.Status != Miss {
		t.Errorf("Expected status %s, got %s", Miss, schedulePast.Status)
	}
}

func TestUpdateReconciliationHistory(t *testing.T) {
	s := &Schedule{
		ReconciliationHistory: []ReconciliationHistory{
			{Status: Success, ErrorMessage: "msg1", CallbackOn: "time1"},
			{Status: Miss, ErrorMessage: "msg2", CallbackOn: "time2"},
			{Status: Scheduled, ErrorMessage: "msg3", CallbackOn: "time3"},
			{Status: Failure, ErrorMessage: "msg4", CallbackOn: "time4"},
			{Status: Failure, ErrorMessage: "msg5", CallbackOn: "time5"},
		},
	}

	s.UpdateReconciliationHistory(Success, "newMsg")

	if len(s.ReconciliationHistory) != maxHistorySize {
		t.Fatalf("Expected length %d, got %d", maxHistorySize, len(s.ReconciliationHistory))
	}

	if s.ReconciliationHistory[0].Status != Miss || s.ReconciliationHistory[0].ErrorMessage != "msg2" {
		t.Errorf("Expected first item to have status %s and message 'msg2', got status %s and message '%s'", Miss, s.ReconciliationHistory[0].Status, s.ReconciliationHistory[0].ErrorMessage)
	}

	lastItem := s.ReconciliationHistory[len(s.ReconciliationHistory)-1]
	if lastItem.Status != Success || lastItem.ErrorMessage != "newMsg" {
		t.Errorf("Expected last item to have status %s and message 'newMsg', got status %s and message '%s'", Success, lastItem.Status, lastItem.ErrorMessage)
	}
}

func TestUnmarshalJSON(t *testing.T) {
	// In your initialization code
	Registry["http"] = func() Callback {
		return &HttpCallback{}
	}

	// Create a Schedule as JSON
	jsonData := []byte(`{
		"appId": "TestApp",
		"scheduleId": "550e8400-e29b-41d4-a716-446655440000",
		"payload": "Sample Payload",
		"scheduleTime": 1623949803,
		"partitionId": 0,
		"scheduleGroup": 1623949800,
		"callback": {
			"type": "http",
			"details": {
				"url": "http://localhost:8080/test",
				"method": "POST",
				"headers": {
					"Content-Type": "application/json"
				}
			}
		}
	}`)

	s := new(Schedule)
	err := s.UnmarshalJSON(jsonData)
	if err != nil {
		t.Fatalf("UnmarshalJSON returned error: %v", err)
	}

	// Check fields of the Schedule struct
	if s.AppId != "TestApp" {
		t.Errorf("Expected AppId 'TestApp', got '%s'", s.AppId)
	}

	if s.ScheduleId.String() != "550e8400-e29b-41d4-a716-446655440000" {
		t.Errorf("Expected ScheduleId '550e8400-e29b-41d4-a716-446655440000', got '%s'", s.ScheduleId)
	}

	if s.Payload != "Sample Payload" {
		t.Errorf("Expected Payload 'Sample Payload', got '%s'", s.Payload)
	}

	if s.ScheduleTime != 1623949803 {
		t.Errorf("Expected ScheduleTime 1623949803, got '%d'", s.ScheduleTime)
	}

	httpCallback, ok := s.Callback.(*HttpCallback)
	if !ok {
		t.Fatalf("Expected Callback of type *HttpCallback, got %T", s.Callback)
	}

	if httpCallback.Details.Url != "http://localhost:8080/test" {
		t.Errorf("Expected Callback URL 'http://localhost:8080/test', got '%s'", httpCallback.Details.Url)
	}

	if httpCallback.Details.Method != "POST" {
		t.Errorf("Expected Callback Method 'POST', got '%s'", httpCallback.Details.Method)
	}

	contentType, ok := httpCallback.Details.Headers["Content-Type"]
	if !ok {
		t.Errorf("Expected 'Content-Type' header in Callback")
	}

	if contentType != "application/json" {
		t.Errorf("Expected 'Content-Type' header to be 'application/json', got '%s'", contentType)
	}
}

func TestSetStatus(t *testing.T) {
	// Prepare the map input
	m := map[string]interface{}{
		"schedule_status": "Scheduled",
		"error_msg":       "Test Error",
		"reconciliation_history": `[{
			"status": "Scheduled",
			"errorMessage": "Test History Error",
			"callbackOn": "2023-06-12T14:00:00Z"
		}]`,
	}

	s := new(Schedule)
	err := s.SetStatus(m)
	if err != nil {
		t.Fatalf("SetStatus returned error: %v", err)
	}

	// Check fields of the Schedule struct
	if s.Status != "Scheduled" {
		t.Errorf("Expected Status 'Scheduled', got '%s'", s.Status)
	}

	if s.ErrorMessage != "Test Error" {
		t.Errorf("Expected ErrorMessage 'Test Error', got '%s'", s.ErrorMessage)
	}

	if len(s.ReconciliationHistory) != 1 {
		t.Fatalf("Expected ReconciliationHistory of length 1, got %d", len(s.ReconciliationHistory))
	}

	history := s.ReconciliationHistory[0]
	if history.Status != "Scheduled" {
		t.Errorf("Expected ReconciliationHistory[0].Status 'Scheduled', got '%s'", history.Status)
	}

	if history.ErrorMessage != "Test History Error" {
		t.Errorf("Expected ReconciliationHistory[0].ErrorMessage 'Test History Error', got '%s'", history.ErrorMessage)
	}

	layout := "2023-06-12T14:00:00Z"
	if history.CallbackOn != layout {
		t.Errorf("Expected ReconciliationHistory[0].CallbackOn '2023-06-12T14:00:00Z', got '%v'", history.CallbackOn)
	}
}
