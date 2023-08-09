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
