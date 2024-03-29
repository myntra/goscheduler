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

package constants

const (
	ErrorConfig                              = "Error while loading config"
	AppName                                  = "goscheduler"
	HttpCallback                             = "HttpCallback"
	Start                                    = "Start"
	Stop                                     = "Stop"
	Running                                  = "Running"
	EmptyString                              = ""
	CreateSchedule                           = "CreateSchedule"
	CreateRecurringSchedule                  = "CreateRecurringSchedule"
	DeleteSchedule                           = "DeleteSchedule"
	GetSchedule                              = "GetSchedule"
	GetScheduleRuns                          = "GetScheduleRuns"
	GetAppSchedule                           = "GetAppSchedule"
	GetCronSchedule                          = "GetCronSchedule"
	RegisterApp                              = "RegisterApp"
	ActivateApp                              = "ActivateApp"
	DeactivateApp                            = "DeactivateApp"
	Success                                  = "Success"
	StatusType                               = "statusType"
	StatusCode                               = "statusCode"
	Fail                                     = "Fail"
	StatusMessage                            = "statusMessage"
	GetApps                                  = "GetApps"
	DOT                                      = "."
	CassandraInsert                          = "CassandraInsert"
	HttpRetry                                = "HttpRetry"
	Panic                                    = "Panic"
	ContentType                              = "Content-Type"
	ApplicationJson                          = "application/json"
	SecondsToMillis                          = 1000
	SuccessCode200                           = 200
	SuccessCode201                           = 201
	ScheduleIdHeader                         = "Schedule-Id"
	ParentScheduleId                         = "Parent-Schedule-Id"
	INFO                                     = 2 // This log level is used for Create and Delete happy flows to avoid excessive latency
	PollerKeySep                             = "."
	BulkAction                               = "BulkAction"
	DefaultCallback                          = "http"
	HttpResponseSuccessStatusCodeLowerBound  = 200
	HttpResponseSuccessStatusCodeHigherBound = 299
)
