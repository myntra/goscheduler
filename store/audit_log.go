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
	"encoding/json"
	"time"

	"github.com/gocql/gocql"
)

// AuditAction represents the type of action performed on a schedule
type AuditAction string

const (
	AuditActionCreate AuditAction = "CREATE"
	AuditActionDelete AuditAction = "DELETE"
	AuditActionUpdate AuditAction = "UPDATE"
)

// AuditLog represents an audit log entry for schedule operations
type AuditLog struct {
	ScheduleId gocql.UUID `json:"scheduleId"`
	AppId      string     `json:"appId"`
	Action     AuditAction `json:"action"`
	Actor      string     `json:"actor"`
	Timestamp  time.Time  `json:"timestamp"`
	Details    string     `json:"details"`
}

// NewAuditLog creates a new audit log entry
func NewAuditLog(scheduleId gocql.UUID, appId string, action AuditAction, actor string, details string) AuditLog {
	return AuditLog{
		ScheduleId: scheduleId,
		AppId:      appId,
		Action:     action,
		Actor:      actor,
		Timestamp:  time.Now(),
		Details:    details,
	}
}

// NewAuditLogFromSchedule creates an audit log entry from a schedule
func NewAuditLogFromSchedule(schedule Schedule, action AuditAction, actor string) AuditLog {
	details := map[string]interface{}{
		"payload":          schedule.Payload,
		"partitionId":      schedule.PartitionId,
		"cronExpression":   schedule.CronExpression,
		"scheduleTime":     schedule.ScheduleTime,
		"parentScheduleId": schedule.ParentScheduleId,
	}

	// Safely get callback type
	if schedule.Callback != nil {
		details["callbackType"] = schedule.GetCallBackType()
	} else {
		details["callbackType"] = ""
	}

	detailsJSON, _ := json.Marshal(details)

	return AuditLog{
		ScheduleId: schedule.ScheduleId,
		AppId:      schedule.AppId,
		Action:     action,
		Actor:      actor,
		Timestamp:  time.Now(),
		Details:    string(detailsJSON),
	}
}

// CreateAuditLogFromCassandraMap creates an AuditLog from a Cassandra map
func (a *AuditLog) CreateAuditLogFromCassandraMap(m map[string]interface{}) error {
	if len(m) == 0 {
		return nil
	}

	a.AppId = m["app_id"].(string)
	a.ScheduleId = m["schedule_id"].(gocql.UUID)
	a.Action = AuditAction(m["action"].(string))
	a.Actor = m["actor"].(string)
	a.Timestamp = m["timestamp"].(time.Time)
	a.Details = m["details"].(string)

	return nil
}
