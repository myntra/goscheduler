package error

import (
	"encoding/json"
	"net/http"
	"github.com/golang/glog"
	"github.com/myntra/goscheduler/constants"
)

type AppError struct {
	Code int
	Err  error
}

func (err AppError) Error() string {
	return err.Err.Error()
}

func NewError(code int, err error) AppError {
	return AppError{Code: code, Err: err}
}

const (
	InvalidDataCode        = 400
	DataNotFound           = 404
	TooManyRequests        = 429
	InvalidAppId           = 4001
	DeactivatedApp         = 4002
	ActivatedApp           = 4003
	BulkActionPushFailure  = 4004
	InvalidBulkActionType  = 4005
	UnmarshalErrorCode     = 5001
	ValidationFailCode     = 5003
	DataPersistenceFailure = 5004
	InvalidCallbackType    = 5005
	DataFetchFailure       = 5006
	EntityBootFailed       = 5007
)

func Handle(w http.ResponseWriter, r *http.Request, err AppError) {
	glog.Errorf(err.Error())
	responseStatus := make(map[string]interface{})
	responseStatus[constants.StatusType] = constants.Fail
	responseStatus[constants.StatusMessage] = err.Error()
	responseStatus[constants.StatusCode] = err.Code
	response := make(map[string]interface{})
	response["status"] = responseStatus
	switch err.Code {
	case DataNotFound:
		w.WriteHeader(http.StatusNotFound)
	case InvalidDataCode:
		w.WriteHeader(http.StatusBadRequest)
	case ValidationFailCode:
		w.WriteHeader(http.StatusBadRequest)
	case InvalidAppId:
		w.WriteHeader(http.StatusBadRequest)
	case DeactivatedApp:
		w.WriteHeader(http.StatusBadRequest)
	case ActivatedApp:
		w.WriteHeader(http.StatusBadRequest)
	case DataPersistenceFailure:
		w.WriteHeader(http.StatusInternalServerError)
	case InvalidCallbackType:
		w.WriteHeader(http.StatusInternalServerError)
	case UnmarshalErrorCode:
		w.WriteHeader(http.StatusBadRequest)
	case TooManyRequests:
		w.WriteHeader(http.StatusTooManyRequests)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
	jsonStr, _ := json.Marshal(response)
	_, _ = w.Write(jsonStr)
	w.Header().Set(constants.ContentType, constants.ApplicationJson)
}
