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
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/gocql/gocql"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/myntra/goscheduler/constants"
	er "github.com/myntra/goscheduler/error"
	"github.com/myntra/goscheduler/store"
	"github.com/myntra/goscheduler/util"
)

// validateImmutableFields ensures that immutable fields (appId, scheduleId, partitionId)
// are not being modified in the update request
func (s *Service) validateImmutableFields(inputSchedule, existing store.Schedule) error {
	var errs []string

	// Verify that appId is not being modified (only if provided in inputSchedule)
	if inputSchedule.AppId != "" && inputSchedule.AppId != existing.AppId {
		glog.Infof("Cannot modify appId for schedule with id %s", existing.ScheduleId)
		errs = append(errs, "Cannot modify appId for an existing schedule")
	}

	// Verify that scheduleId is not being modified (only if provided in inputSchedule)
	if !util.IsZeroUUID(inputSchedule.ScheduleId) && inputSchedule.ScheduleId != existing.ScheduleId {
		glog.Infof("Cannot modify scheduleId for schedule with id %s", existing.ScheduleId)
		errs = append(errs, "Cannot modify scheduleId for an existing schedule")
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, ","))
	}

	return nil
}

// validateScheduleID validates and parses the schedule ID from the request
func validateScheduleID(scheduleID string) (gocql.UUID, error) {
	uuid, err := gocql.ParseUUID(scheduleID)
	if err != nil {
		return gocql.UUID{}, fmt.Errorf("cannot parse UUID from %s: %w", scheduleID, err)
	}
	return uuid, nil
}

// validateExistingScheduleAndExtractApp checks if the existing schedule is valid and recurring
func (s *Service) validateExistingScheduleAndExtractApp(uuid gocql.UUID) (*store.Schedule, store.App, error) {
	// Get the existing schedule
	existingSchedule, err := s.ScheduleDao.GetSchedule(uuid)
	if err != nil {
		if err == gocql.ErrNotFound {
			return nil, store.App{}, er.NewError(er.DataNotFound, fmt.Errorf("schedule with id: %s not found", uuid))
		}
		return nil, store.App{}, er.NewError(er.DataPersistenceFailure, fmt.Errorf("error fetching existing schedule with id %s: %w", uuid, err))
	}

	// Check if the existingSchedule is recurring
	if !existingSchedule.IsRecurring() {
		return nil, store.App{}, er.NewError(er.UnprocessableEntity, fmt.Errorf("schedule with id: %s is not a recurring schedule", uuid))
	}

	if existingSchedule.Status == store.Deleted {
		return nil, store.App{}, er.NewError(er.UnprocessableEntity, fmt.Errorf("schedule with id: %s is deleted", uuid))
	}

	// Get the app for validation
	app, err := s.getApp(existingSchedule.AppId)
	if err != nil {
		return nil, store.App{}, err
	}

	return &existingSchedule, app, nil
}

// updateScheduleFields updates the allowed fields in the existing schedule
func updateScheduleFields(existingSchedule *store.Schedule, inputSchedule store.Schedule) error {
	// Update allowed fields
	if inputSchedule.CronExpression != "" {
		existingSchedule.CronExpression = inputSchedule.CronExpression
	}
	if inputSchedule.Payload != "" {
		existingSchedule.Payload = inputSchedule.Payload
	}
	if inputSchedule.CallbackRaw != nil {
		existingSchedule.CallbackRaw = inputSchedule.CallbackRaw
		// Create Callback from CallbackRaw
		callback, err := store.CreateCallbackFromRawMessage(inputSchedule.CallbackRaw)
		if err != nil {
			return fmt.Errorf("error creating callback from raw message: %w", err)
		}
		existingSchedule.Callback = callback
	}
	return nil
}

// validateUpdatedSchedule validates the schedule after updates
func (s *Service) validateUpdatedSchedule(schedule *store.Schedule, app store.App) error {
	validationErrs := schedule.ValidateSchedule(app, s.Config.AppLevelConfiguration)
	if len(validationErrs) > 0 {
		return fmt.Errorf("validation errors: %s", strings.Join(validationErrs, ","))
	}
	return nil
}

// UpdateRecurringSchedule updates the existing recurring schedule with new values
// It supports updating cron expression, payload, headers, callback_type, call_back_url
func (s *Service) UpdateRecurringSchedule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scheduleID := vars["scheduleId"]

	// Step 1: Validate and parse schedule ID
	uuid, err := validateScheduleID(scheduleID)
	if err != nil {
		glog.Errorf("UpdateRecurringSchedule: %v", err)
		s.recordRequestStatus(constants.UpdateRecurringSchedule, constants.Fail)
		er.Handle(w, r, er.NewError(er.InvalidDataCode, err))
		return
	}

	// Step 2: Read and parse request body
	var inputSchedule store.Schedule
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		glog.Errorf("UpdateRecurringSchedule: Error reading request body: %v", err)
		s.recordRequestStatus(constants.UpdateRecurringSchedule, constants.Fail)
		er.Handle(w, r, er.NewError(er.UnmarshalErrorCode, err))
		return
	}

	err = json.Unmarshal(b, &inputSchedule)
	if err != nil {
		s.recordRequestStatus(constants.UpdateRecurringSchedule, constants.Fail)
		er.Handle(w, r, er.NewError(er.UnmarshalErrorCode, err))
		return
	}

	// Step 3: Validate existing schedule and get app
	existingSchedule, app, err := s.validateExistingScheduleAndExtractApp(uuid)
	if err != nil {
		s.recordRequestStatus(constants.UpdateRecurringSchedule, constants.Fail)
		// Check if it's already an AppError, otherwise wrap it
		if appErr, ok := err.(er.AppError); ok {
			er.Handle(w, r, appErr)
		} else {
			er.Handle(w, r, er.NewError(er.DataPersistenceFailure, err))
		}
		return
	}

	// Step 4: Validate immutable fields
	if err := s.validateImmutableFields(inputSchedule, *existingSchedule); err != nil {
		s.recordRequestStatus(constants.UpdateRecurringSchedule, constants.Fail)
		er.Handle(w, r, er.NewError(er.InvalidDataCode, err))
		return
	}

	// Step 5: Update allowed fields
	if err := updateScheduleFields(existingSchedule, inputSchedule); err != nil {
		glog.Errorf("UpdateRecurringSchedule: %v", err)
		s.recordRequestStatus(constants.UpdateRecurringSchedule, constants.Fail)
		er.Handle(w, r, er.NewError(er.InvalidDataCode, err))
		return
	}

	// Step 6: Validate updated schedule
	if err := s.validateUpdatedSchedule(existingSchedule, app); err != nil {
		glog.Errorf("UpdateRecurringSchedule: %v", err)
		s.recordRequestStatus(constants.UpdateRecurringSchedule, constants.Fail)
		er.Handle(w, r, er.NewError(er.UnprocessableEntity, err))
		return
	}

	// Step 7: Persist the update
	updatedSchedule, err := s.ScheduleDao.UpdateRecurringSchedule(*existingSchedule)
	if err != nil {
		glog.Errorf("UpdateRecurringSchedule: %v", err)
		s.recordRequestStatus(constants.UpdateRecurringSchedule, constants.Fail)
		er.Handle(w, r, er.NewError(er.DataPersistenceFailure, err))
		return
	}

	// Step 8: Send success response
	glog.V(constants.INFO).Infof("Recurring schedule with id %s updated", uuid.String())
	s.recordRequestStatus(constants.UpdateRecurringSchedule, constants.Success)
	status := Status{
		StatusCode:    constants.SuccessCode200,
		StatusMessage: "Recurring schedule updated successfully",
		StatusType:    constants.Success,
		TotalCount:    1,
	}

	data := UpdatedScheduleData{
		Schedule: updatedSchedule,
	}
	_ = json.NewEncoder(w).Encode(
		UpdatedScheduleResponse{
			Status: status,
			Data:   data,
		})
}
