package service

import (
	"github.com/gocql/gocql"
	s "github.com/myntra/goscheduler/store"
)

type CreateAppResponse struct {
	Status Status        `json:"status"`
	Data   CreateAppData `json:"data"`
}

type CreateAppData struct {
	AppId      string `json:"appId"`
	Partitions uint32 `json:"partitions"`
	Active     bool   `json:"active"`
	//Configuration s.Configuration `json:"configuration"`
}

type UpdateAppActiveStatusResponse struct {
	Status Status                    `json:"status"`
	Data   UpdateAppActiveStatusData `json:"data"`
}

type UpdateAppActiveStatusData struct {
	AppId  string `json:"appId"`
	Active bool   `json:"Active"`
}

type UpdateConfigurationData struct {
	AppId string `json:"appId"`
	//Configuration s.Configuration `json:"configuration"`
}

type UpdateConfigurationResponse struct {
	Status Status                  `json:"status"`
	Data   UpdateConfigurationData `json:"data"`
}

type DeleteScheduleResponse struct {
	Status Status             `json:"status"`
	Data   DeleteScheduleData `json:"data"`
}

type DeleteScheduleData struct {
	Schedule s.Schedule `json:"schedule"`
}

type DeleteConfigurationData struct {
	AppId string `json:"appId"`
	//Configuration s.Configuration `json:"configuration"`
}

type DeleteConfigurationResponse struct {
	Status Status                  `json:"status"`
	Data   DeleteConfigurationData `json:"data"`
}

type GetScheduleResponse struct {
	Status Status          `json:"status"`
	Data   GetScheduleData `json:"data"`
}

type GetScheduleData struct {
	Schedule s.Schedule `json:"schedule"`
}

type GetScheduleRunsResponse struct {
	Status Status              `json:"status"`
	Data   GetScheduleRunsData `json:"data"`
}
type GetPaginatedRunSchedulesData struct {
	Schedules         []s.Schedule `json:"schedules"`
	ContinuationToken string       `json:"continuationToken"`
}

type GetPaginatedRunSchedulesResponse struct {
	Status Status                       `json:"status"`
	Data   GetPaginatedRunSchedulesData `json:"data"`
}

type GetScheduleRunsData struct {
	Schedules []s.Schedule `json:"schedules"`
}

type GetAppSchedulesResponse struct {
	Status Status              `json:"status"`
	Data   GetAppSchedulesData `json:"data"`
}

type GetCronSchedulesResponse struct {
	Status Status       `json:"status"`
	Data   []s.Schedule `json:"data"`
}

type GetPaginatedAppSchedulesResponse struct {
	Status Status                       `json:"status"`
	Data   GetPaginatedAppSchedulesData `json:"data"`
}

type GetAppSchedulesData struct {
	Schedules []s.Schedule `json:"schedules"`
}

type GetPaginatedAppSchedulesData struct {
	Schedules             []s.Schedule `json:"schedules"`
	ContinuationToken     string       `json:"continuationToken"`
	ContinuationStartTime int64        `json:"continuationStartTime"`
}

//type GetConfigurationData struct {
//	AppId         string          `json:"appId"`
//	Configuration s.Configuration `json:"configuration"`
//}

type GetAppsData struct {
	Apps []s.App `json:"apps"`
}

//type GetConfigurationResponse struct {
//	Status Status               `json:"status"`
//	Data   GetConfigurationData `json:"data"`
//}

type CreateScheduleResponse struct {
	Status Status             `json:"status"`
	Data   CreateScheduleData `json:"data"`
}

type CreateScheduleData struct {
	Schedule s.Schedule `json:"schedule"`
}

type CreateConfigurationData struct {
	AppId string `json:"appId"`
	//Configuration s.Configuration `json:"configuration"`
}

type CreateConfigurationResponse struct {
	Status Status                  `json:"status"`
	Data   CreateConfigurationData `json:"data"`
}

type Status struct {
	StatusCode    int    `json:"statusCode"`
	StatusMessage string `json:"statusMessage"`
	StatusType    string `json:"statusType"`
	TotalCount    int    `json:"totalCount"`
}

type AllScheduleResponse struct {
	Schedule []s.Schedule `json:"schdeules"`
}

type NewScheduleResponse struct {
	ScheduleId gocql.UUID `json:"scheduleid"`
}

type ErrorResponse struct {
	Errors []string `json:"errors"`
}

type BulkActionResponse struct {
	Status  Status `json:"status"`
	Remarks string `json:"remarks"`
}

type GetAppsResponse struct {
	Status Status      `json:"status"`
	Data   GetAppsData `json:"data"`
}
