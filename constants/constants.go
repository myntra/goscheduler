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
