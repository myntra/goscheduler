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
	"github.com/gocql/gocql"
	"github.com/gorilla/mux"
	"github.com/myntra/goscheduler/constants"
	er "github.com/myntra/goscheduler/error"
	sch "github.com/myntra/goscheduler/store"
	"net/http"
)

func (s *Service) CancelSchedule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	uuid := vars["scheduleId"]

	// Extract actor from request header, default to "system" if not provided
	actor := r.Header.Get("X-User-ID")
	if actor == "" {
		actor = "system"
	}

	schedule, err := s.DeleteSchedule(uuid, actor)
	if err != nil {
		s.recordRequestStatus(constants.DeleteSchedule, constants.Fail)
		er.Handle(w, r, err.(er.AppError))
		return
	}

	s.recordRequestAppStatus(constants.DeleteSchedule, schedule.AppId, constants.Success)

	status := Status{
		StatusCode:    constants.SuccessCode200,
		StatusMessage: constants.Success,
		StatusType:    constants.Success,
		TotalCount:    1,
	}
	data := DeleteScheduleData{
		Schedule: schedule,
	}
	_ = json.NewEncoder(w).Encode(
		DeleteScheduleResponse{
			Status: status,
			Data:   data,
		})
}

func (s *Service) DeleteSchedule(uuid string, actor ...string) (sch.Schedule, error) {
	scheduleId, err := gocql.ParseUUID(uuid)
	if err != nil {
		return sch.Schedule{}, er.NewError(er.InvalidDataCode, err)
	}

	// Get schedule details before deletion for audit log
	schedule, err := s.ScheduleDao.GetSchedule(scheduleId)
	if err != nil {
		return sch.Schedule{}, er.NewError(er.DataNotFound, err)
	}

	switch deletedSchedule, err := s.ScheduleDao.DeleteSchedule(scheduleId); err {
	case gocql.ErrNotFound:
		return sch.Schedule{}, er.NewError(er.DataNotFound, err)
	case nil:
		// Create audit log for schedule deletion
		auditActor := "system"
		if len(actor) > 0 && actor[0] != "" {
			auditActor = actor[0]
		}
		s.createAuditLog(schedule, sch.AuditActionDelete, auditActor)
		return deletedSchedule, nil
	default:
		return sch.Schedule{}, er.NewError(er.DataFetchFailure, err)
	}
}
