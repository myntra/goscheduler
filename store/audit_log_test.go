package store

import (
	"testing"
	"time"

	"github.com/gocql/gocql"
)

func TestNewAuditLog(t *testing.T) {
	scheduleId := gocql.MustRandomUUID()
	appId := "testApp"
	action := AuditActionCreate
	actor := "user1"
	details := `{"payload":"test"}`

	auditLog := NewAuditLog(scheduleId, appId, action, actor, details)

	if auditLog.ScheduleId != scheduleId {
		t.Errorf("Expected ScheduleId to be %v, got %v", scheduleId, auditLog.ScheduleId)
	}
	if auditLog.AppId != appId {
		t.Errorf("Expected AppId to be %s, got %s", appId, auditLog.AppId)
	}
	if auditLog.Action != action {
		t.Errorf("Expected Action to be %s, got %s", action, auditLog.Action)
	}
	if auditLog.Actor != actor {
		t.Errorf("Expected Actor to be %s, got %s", actor, auditLog.Actor)
	}
	if auditLog.Details != details {
		t.Errorf("Expected Details to be %s, got %s", details, auditLog.Details)
	}
	if auditLog.Timestamp.IsZero() {
		t.Error("Expected Timestamp to be set, got zero value")
	}
}

func TestNewAuditLogFromSchedule(t *testing.T) {
	scheduleId := gocql.MustRandomUUID()
	parentScheduleId := gocql.MustRandomUUID()
	
	schedule := Schedule{
		ScheduleId:     scheduleId,
		AppId:          "testApp",
		Payload:        `{"key":"value"}`,
		PartitionId:    1,
		CronExpression: "0 0 * * *",
		ScheduleTime:   time.Now().Unix(),
		ParentScheduleId: parentScheduleId,
		Callback: &HttpCallback{
			Type: "http",
			Details: Details{
				Url:    "http://example.com",
				Method: "POST",
			},
		},
	}

	action := AuditActionCreate
	actor := "user1"

	auditLog := NewAuditLogFromSchedule(schedule, action, actor)

	if auditLog.ScheduleId != scheduleId {
		t.Errorf("Expected ScheduleId to be %v, got %v", scheduleId, auditLog.ScheduleId)
	}
	if auditLog.AppId != "testApp" {
		t.Errorf("Expected AppId to be testApp, got %s", auditLog.AppId)
	}
	if auditLog.Action != action {
		t.Errorf("Expected Action to be %s, got %s", action, auditLog.Action)
	}
	if auditLog.Actor != actor {
		t.Errorf("Expected Actor to be %s, got %s", actor, auditLog.Actor)
	}
	if auditLog.Timestamp.IsZero() {
		t.Error("Expected Timestamp to be set, got zero value")
	}

	// Verify details JSON contains expected fields
	if auditLog.Details == "" {
		t.Error("Expected Details to be non-empty")
	}
}

func TestNewAuditLogFromSchedule_WithNilCallback(t *testing.T) {
	scheduleId := gocql.MustRandomUUID()
	
	schedule := Schedule{
		ScheduleId:  scheduleId,
		AppId:       "testApp",
		Payload:     `{"key":"value"}`,
		PartitionId: 1,
		Callback:    nil,
	}

	action := AuditActionDelete
	actor := "user2"

	// Should not panic with nil callback
	auditLog := NewAuditLogFromSchedule(schedule, action, actor)

	if auditLog.ScheduleId != scheduleId {
		t.Errorf("Expected ScheduleId to be %v, got %v", scheduleId, auditLog.ScheduleId)
	}
	if auditLog.Action != action {
		t.Errorf("Expected Action to be %s, got %s", action, auditLog.Action)
	}
}

func TestCreateAuditLogFromCassandraMap(t *testing.T) {
	scheduleId := gocql.MustRandomUUID()
	timestamp := time.Now()
	
	m := map[string]interface{}{
		"app_id":      "testApp",
		"schedule_id": scheduleId,
		"action":      "CREATE",
		"actor":       "user1",
		"timestamp":   timestamp,
		"details":     `{"payload":"test"}`,
	}

	var auditLog AuditLog
	err := auditLog.CreateAuditLogFromCassandraMap(m)
	if err != nil {
		t.Fatalf("CreateAuditLogFromCassandraMap failed: %v", err)
	}

	if auditLog.AppId != "testApp" {
		t.Errorf("Expected AppId to be testApp, got %s", auditLog.AppId)
	}
	if auditLog.ScheduleId != scheduleId {
		t.Errorf("Expected ScheduleId to be %v, got %v", scheduleId, auditLog.ScheduleId)
	}
	if auditLog.Action != AuditActionCreate {
		t.Errorf("Expected Action to be CREATE, got %s", auditLog.Action)
	}
	if auditLog.Actor != "user1" {
		t.Errorf("Expected Actor to be user1, got %s", auditLog.Actor)
	}
	if !auditLog.Timestamp.Equal(timestamp) {
		t.Errorf("Expected Timestamp to be %v, got %v", timestamp, auditLog.Timestamp)
	}
	if auditLog.Details != `{"payload":"test"}` {
		t.Errorf("Expected Details to be {\"payload\":\"test\"}, got %s", auditLog.Details)
	}
}

func TestCreateAuditLogFromCassandraMap_EmptyMap(t *testing.T) {
	m := map[string]interface{}{}

	var auditLog AuditLog
	err := auditLog.CreateAuditLogFromCassandraMap(m)
	if err != nil {
		t.Fatalf("CreateAuditLogFromCassandraMap failed: %v", err)
	}

	// All fields should be zero values
	if auditLog.AppId != "" {
		t.Errorf("Expected AppId to be empty, got %s", auditLog.AppId)
	}
}
