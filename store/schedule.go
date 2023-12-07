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
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gocql/gocql"
	"github.com/golang/glog"
	"github.com/myntra/goscheduler/conf"
	"github.com/myntra/goscheduler/constants"
	"github.com/myntra/goscheduler/cron"
	"github.com/myntra/goscheduler/util"
	"time"
)

// Status for schedules in the system
type Status string

type ActionType string

const DefaultTimeLayout = "2006-01-02 15:04:05"
const maxHistorySize = 5
const _60seconds = 60

const (
	Scheduled Status     = "SCHEDULED"
	Deleted   Status     = "DELETED"
	Success   Status     = "SUCCESS"
	Failure   Status     = "FAILURE"
	Miss      Status     = "MISS"
	Error     Status     = "ERROR"
	Reconcile ActionType = "reconcile"
	Delete    ActionType = "delete"
)

type Schedule struct {
	ScheduleId            gocql.UUID              `json:"scheduleId"`
	Payload               string                  `json:"payload"`
	AppId                 string                  `json:"appId"`
	ScheduleTime          int64                   `json:"scheduleTime,omitempty"`
	PartitionId           int                     `json:"partitionId"`
	ScheduleGroup         int64                   `json:"scheduleGroup,omitempty"`
	Callback              Callback                `json:"-"`
	CallbackRaw           json.RawMessage         `json:"callback,omitempty"`
	CronExpression        string                  `json:"cronExpression,omitempty"`
	Status                Status                  `json:"status,omitempty"`
	ErrorMessage          string                  `json:"errorMessage,omitempty"`
	ParentScheduleId      gocql.UUID              `json:"-"`
	ReconciliationHistory []ReconciliationHistory `json:"reconciliationHistory,omitempty"`
	//Deprecated
	Ttl            int            `json:"-"`
	AirbusCallback AirbusCallback `json:"airbusCallback,omitempty"`
	HttpCallback   HTTPCallback   `json:"httpCallback,omitempty"`
}

type AirbusCallback struct {
	EventName string            `json:"eventName,omitempty"`
	AppName   string            `json:"appName,omitempty"`
	Headers   map[string]string `json:"headers,omitempty"`
}

type HTTPCallback struct {
	Url     string            `json:"url"`
	Headers map[string]string `json:"headers"`
}

type ReconciliationHistory struct {
	Status       Status `json:"status,omitempty"`
	ErrorMessage string `json:"errorMessage,omitempty"`
	CallbackOn   string `json:"callbackOn,omitempty"`
}

type ScheduleWrapper struct {
	Schedule         Schedule
	App              App
	IsReconciliation bool
}

type BulkActionTask struct {
	App               App        `json:"app,omitempty"`
	PartitionId       int        `json:"partitionId,omitempty"`
	ScheduleTimeGroup time.Time  `json:"scheduleTimeGroup,omitempty"`
	Status            Status     `json:"status,omitempty"`
	ActionType        ActionType `json:"actionType"`
}

type StatusTask struct {
	Schedules []Schedule
	App       App
}

type CreateScheduleTask struct {
	Cron     Schedule
	From     time.Time
	Duration time.Duration
}

func (s Schedule) GetCallBackType() string {
	return s.Callback.GetType()
}

func (s Schedule) GetCallbackDetails() string {
	details, _ := s.Callback.GetDetails()
	return details
}

func (s *Schedule) CreateScheduleFromCassandraMap(m map[string]interface{}) error {
	glog.V(constants.INFO).Infof("Map: %+v", m)
	if len(m) == 0 {
		return nil
	}

	s.AppId = m["app_id"].(string)
	s.PartitionId = m["partition_id"].(int)

	// Get Callback from map
	callback, err := createCallbackFromMap(m)
	if err != nil {
		return err
	}
	s.Callback = callback

	// Get callbackRaw from map
	raw, err := convertCallbackToRaw(s)
	if err != nil {
		return err
	}
	s.CallbackRaw = raw

	s.Payload = m["payload"].(string)

	if cronExpr, ok := m["cron_expression"]; ok {
		s.CronExpression = cronExpr.(string)
	} else {
		s.ScheduleGroup = m["schedule_time_group"].(time.Time).Unix()
		s.ScheduleTime = m["schedule_time"].(time.Time).Unix()
	}

	if status, ok := m["status"]; ok {
		s.Status = Status(status.(string))
	}

	s.ScheduleId = m["schedule_id"].(gocql.UUID)
	if m["parent_schedule_id"] != nil && !util.IsZeroUUID(m["parent_schedule_id"].(gocql.UUID)) {
		s.ParentScheduleId = m["parent_schedule_id"].(gocql.UUID)
	}

	return nil
}

func createCallbackFromMap(m map[string]interface{}) (Callback, error) {
	callbackType := m["callback_type"].(string)

	callbackFactory, exists := Registry[callbackType]
	if !exists {
		return nil, errors.New("wrong callback type")
	}

	callback := callbackFactory()

	err := callback.Marshal(m)
	if err != nil {
		return nil, err
	}

	return callback, nil
}

func convertCallbackToRaw(s *Schedule) ([]byte, error) {
	if s.Callback != nil {
		raw, err := json.Marshal(s.Callback)
		if err != nil {
			return nil, err
		}
		return raw, nil
	}
	return nil, errors.New("nil callback object")
}

func (s Schedule) IsRecurring() bool {
	return len(s.CronExpression) > 0
}

// CloneAsOneTime Clones a given recurring schedule to one time schedule at a supplied time.:w
func (s Schedule) CloneAsOneTime(at time.Time) Schedule {
	clone := Schedule{}

	clone.ScheduleId = gocql.TimeUUID()
	clone.ScheduleGroup = at.Unix()
	clone.ScheduleTime = at.Unix()
	clone.AppId = s.AppId
	clone.Callback = s.Callback
	clone.Payload = s.Payload
	clone.ParentScheduleId = s.ScheduleId

	return clone
}

// CheckUntriggeredCallback checks if the current time is already past the schedule time group of the schedule
// with gap of more than a minute plus flush period
func (s Schedule) CheckUntriggeredCallback(flushPeriod int) bool {
	scheduleTimeGroup := s.ScheduleGroup
	now := time.Now()
	return time.Unix(scheduleTimeGroup, 0).Before(now) &&
		(now.Sub(time.Unix(scheduleTimeGroup, 0)).Seconds() > float64(_60seconds+flushPeriod))
}

// GetTTL TTL will be set at schedule level
// ttl = scheduleTime - now
func (s Schedule) GetTTL(app App, bufferTTL int) int {
	return int(s.ScheduleTime-time.Now().Unix()) + app.GetBufferTTL(bufferTTL)
}

// Set status, error_msg and reconciliation_history of the schedule from map
func (s *Schedule) SetStatus(m map[string]interface{}) error {
	if len(m) == 0 {
		return nil
	}
	s.Status = Status(m["schedule_status"].(string))
	s.ErrorMessage = m["error_msg"].(string)

	if m["reconciliation_history"].(string) == "" {
		s.ReconciliationHistory = []ReconciliationHistory{}
		return nil
	}

	if err := json.Unmarshal([]byte(m["reconciliation_history"].(string)), &s.ReconciliationHistory); err != nil {
		glog.Infof("Error unmarshalling: %v", err)
		return err
	}

	return nil
}

// Set status as Scheduled if callback time is yet to come
// Set status as Miss if callback time is already expired
func (s *Schedule) SetUnknownStatus(flushPeriod int) {
	switch {
	case s.CheckUntriggeredCallback(flushPeriod):
		s.Status = Miss
		s.ErrorMessage = "Failed to make a callback"
	default:
		s.Status = Scheduled
		s.ErrorMessage = ""
	}
}

// Update schedule reconciliation history
// If the reconciliation history contains more than "HistorySize" reconciliations
// then consider the latest "HistorySize" reconciliations
func (s *Schedule) UpdateReconciliationHistory(status Status, errMsg string) {
	glog.Infof("Found reconciliations: %+v", s.ReconciliationHistory)

	s.ReconciliationHistory = append(s.ReconciliationHistory, ReconciliationHistory{
		Status:       status,
		ErrorMessage: errMsg,
		CallbackOn:   time.Now().Format(DefaultTimeLayout),
	})

	historyLength := len(s.ReconciliationHistory)
	if historyLength > maxHistorySize {
		s.ReconciliationHistory = s.ReconciliationHistory[historyLength-maxHistorySize : historyLength]
	}
}

func (s *Schedule) UnmarshalJSON(data []byte) error {
	type Alias Schedule
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(s),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	var callbackData struct {
		Type string `json:"type"`
	}

	if err := json.Unmarshal(s.CallbackRaw, &callbackData); err != nil {
		return err
	}

	factoryFunc, ok := Registry[callbackData.Type]
	if !ok {
		return fmt.Errorf("unknown callback type: %s", callbackData.Type)
	}

	callback := factoryFunc()
	if err := json.Unmarshal(s.CallbackRaw, callback); err != nil {
		return err
	}

	s.Callback = callback
	return nil
}

func (s *Schedule) ValidateSchedule(app App, conf conf.AppLevelConfiguration) []string {
	glog.Infof("ValidateSchedule: %+v", s)
	var errs []string

	if errStr := validateField(s.AppId, "appId"); errStr != "" {
		errs = append(errs, errStr)
	}

	if errStr := validateField(s.Payload, "payload"); errStr != "" {
		errs = append(errs, errStr)
	}

	if errStr := validatePayloadSize(s.Payload, app, conf.PayloadSize); errStr != "" {
		errs = append(errs, errStr)
	}

	if errStr := validateCallback(s.Callback); errStr != "" {
		errs = append(errs, errStr)
	}

	if s.IsRecurring() {
		if er := validateCronExpression(s.CronExpression); len(er) > 0 {
			errs = append(errs, er...)
		}
	} else {
		if errStr := validateScheduleTime(s.ScheduleTime, app, conf.FutureScheduleCreationPeriod); errStr != "" {
			errs = append(errs, errStr)
		}
	}

	return errs
}

func (s *Schedule) SetFields(app App) {
	s.ScheduleId = gocql.TimeUUID()
	s.PartitionId = int(uuidToPartition(s.ScheduleId, app.Partitions))
	s.ScheduleGroup = 60 * (s.ScheduleTime / 60)
}

func uuidToPartition(uuid gocql.UUID, partitions uint32) uint64 {
	partitionString := gocql.UUID.String(uuid)
	var partitionByte = []byte(partitionString)
	data := binary.BigEndian.Uint64(partitionByte)
	partitionKey := data % uint64(partitions)
	return partitionKey
}

func validateField(fieldData string, field string) string {
	if len(fieldData) == 0 {
		return "Missing '" + field + "' parameter, cannot continue"
	}
	return ""
}

func validateScheduleTime(scheduleTime int64, app App, maxTTL int) string {
	now := time.Now().Unix()
	if scheduleTime < now {
		return fmt.Sprintf("schedule time : %d is less than current time: %d for app: %s. Time cannot be in past.",
			scheduleTime,
			now,
			app.AppId)
	} else if (scheduleTime - now) > int64(app.GetMaxTTL(maxTTL)) {
		return fmt.Sprintf("schedule time : %d cannot be more than %d days from current time : %d for app: %s",
			scheduleTime,
			app.GetMaxTTL(maxTTL)/(24*60*60),
			now,
			app.AppId)
	}
	return ""
}

func validatePayloadSize(payload string, app App, maxPayload int) string {
	var maxPayloadSize int

	if app.Configuration.PayloadSize == 0 {
		maxPayloadSize = maxPayload
	} else {
		maxPayloadSize = app.Configuration.PayloadSize
	}

	if len(payload) > maxPayloadSize {
		glog.Errorf("PayloadSize for app: %s cannot be more than %d bytes, given payloadSize bytes: %d ", app.AppId, maxPayloadSize, len(payload))
		return fmt.Sprintf("PayloadSize for app: %s cannot be more than %d bytes, given payloadSize bytes: %d", app.AppId, maxPayloadSize, len(payload))
	}

	return ""
}

func validateCronExpression(cronExpression string) []string {
	if _, err := cron.Parse(cronExpression); err != nil {
		return err
	}
	return nil
}

func validateCallback(callback Callback) string {
	glog.Infof("Callback Data: %+v", callback)
	if err := callback.Validate(); err != nil {
		return err.Error()
	}
	return ""
}
