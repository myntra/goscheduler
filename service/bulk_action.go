package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/myntra/goscheduler/constants"
	"github.com/myntra/goscheduler/dao"
	er "github.com/myntra/goscheduler/error"
	"github.com/myntra/goscheduler/store"
	"net/http"
	"time"
)

const MaxBulkActionPeriodInDays = 7

func (s *Service) recordPushBulkActionSuccess(appId string, actionType store.ActionType) {
	if s.Monitoring != nil && s.Monitoring.StatsDClient != nil {
		bucket := appId + constants.DOT + string(actionType) + constants.DOT + constants.Success
		s.Monitoring.StatsDClient.Increment(bucket)
	}
}

func (s *Service) recordPushBulkActionFail(appId string, actionType store.ActionType) {
	if s.Monitoring != nil && s.Monitoring.StatsDClient != nil {
		bucket := appId + constants.DOT + string(actionType) + constants.DOT + constants.Fail
		s.Monitoring.StatsDClient.Increment(bucket)
	}
}

func pushBulkActionQueries(app store.App, timeRange dao.Range, status store.Status, actionType store.ActionType) error {
	for partition := 0; partition < int(app.Partitions); partition++ {
		for _time := timeRange.StartTime; _time.Before(timeRange.EndTime); _time = _time.Add(time.Minute * 1) {
			//Push to BulkActionQueue
			t := store.BulkActionTask{
				App:               app,
				PartitionId:       partition,
				ScheduleTimeGroup: _time,
				Status:            status,
				ActionType:        actionType,
			}
			store.BulkActionQueue <- t
			glog.Infof("Pushed %+v", t)
		}
	}
	return nil
}

func (s *Service) BulkAction(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	appId := vars["appId"]
	action := store.ActionType(vars["action"])

	_, status, timeRange, _, _, err := parse(r)
	if err != nil {
		s.recordPushBulkActionFail(appId, action)
		er.Handle(w, r, er.NewError(er.InvalidDataCode, err))
		return
	}

	err = s.ExecuteBulkAction(appId, action, status, timeRange)
	if err != nil {
		s.recordPushBulkActionFail(appId, action)
		er.Handle(w, r, err.(er.AppError))
		return
	}

	s.recordPushBulkActionSuccess(appId, action)
	_ = json.NewEncoder(w).Encode(
		BulkActionResponse{
			Status: Status{
				StatusCode:    constants.SuccessCode200,
				StatusMessage: constants.Success,
				StatusType:    constants.Success,
			},
			Remarks: fmt.Sprintf("%s initiated successfully for app: %s, timeRange: %+v, status: %+v", action, appId, timeRange, status),
		})
}

func (s *Service) ExecuteBulkAction(appId string, action store.ActionType, status store.Status, timeRange dao.Range) error {
	if action != store.Reconcile && action != store.Delete {
		return er.NewError(er.InvalidBulkActionType, errors.New(fmt.Sprintf("action type %s is invalid", action)))
	}

	app, err := s.getActiveOrInactiveApp(appId)
	if err != nil {
		return err
	}

	if err := validateTimeRange(timeRange); err != nil {
		return err
	}

	if err := pushBulkActionQueries(app, timeRange, status, action); err != nil {
		return er.NewError(er.BulkActionPushFailure, err)
	}

	return nil
}

func validateTimeRange(timeRange dao.Range) error {
	if timeRange.EndTime.Before(timeRange.StartTime) {
		return er.NewError(er.InvalidDataCode, errors.New(fmt.Sprintf("End time: %s cannot be before start time: %s", timeRange.EndTime, timeRange.StartTime)))
	} else if timeRange.EndTime.Sub(timeRange.StartTime).Seconds() > float64(MaxBulkActionPeriodInDays*24*60*60) {
		return er.NewError(er.InvalidDataCode, errors.New(fmt.Sprintf("Time range of more than %d days is not allowed", defaultDays)))
	}

	return nil
}