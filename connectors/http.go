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

package connectors

import (
	"bytes"
	"fmt"
	"github.com/gocql/gocql"
	"github.com/golang/glog"
	"github.com/myntra/goscheduler/constants"
	"github.com/myntra/goscheduler/store"
	"github.com/myntra/goscheduler/util"
	"gopkg.in/alexcesaro/statsd.v2"
	"net/http"
	"net/http/httputil"
	"runtime/debug"
	"strconv"
)

// trim trims message to max number of characters
func trim(message string) string {
	if len(message) < 200 {
		return message
	} else {
		return message[:200]
	}
}

// getKey generates a key string based on the appId and partitionId
func getKey(appId string, partitionId int) string {
	return constants.HttpCallback + constants.DOT + appId + constants.DOT + strconv.Itoa(partitionId)
}

// shouldRetry checks if the request should be retried based on maxAttempts, attempts, and response
func shouldRetry(maxAttempts int, attempts int, response *http.Response) bool {
	if attempts >= maxAttempts {
		return false
	}
	if isSuccess(response) {
		return false
	}
	return true
}

// isSuccess checks if the response is considered successful
func isSuccess(response *http.Response) bool {
	return response != nil && (response.StatusCode >= constants.HttpResponseSuccessStatusCodeLowerBound &&
		response.StatusCode <= constants.HttpResponseSuccessStatusCodeHigherBound)
}

// createRequest creates a new HTTP request from a given input schedule
func createRequest(input store.Schedule) (*http.Request, error) {
	glog.Infof("Method: %s, URL: %s, Headers: %+v", input.Callback.(*store.HttpCallback).Details.Method, input.Callback.(*store.HttpCallback).Details.Url, input.Callback.(*store.HttpCallback).Details.Headers)
	jsonStr := []byte(input.Payload)
	req, err := http.NewRequest(input.Callback.(*store.HttpCallback).Details.Method, input.Callback.(*store.HttpCallback).Details.Url, bytes.NewBuffer(jsonStr))
	if err != nil {
		return nil, err
	}

	setRequestHeaders(req, input)
	handleRequestDump(req, input.ScheduleId)

	return req, nil
}

// handleRequestDump logs the request dump or error if it occurs
func handleRequestDump(req *http.Request, scheduleID gocql.UUID) {
	requestDump, err := httputil.DumpRequest(req, true)
	if err != nil {
		glog.Error("Request dump failed with error for schedule id : " + scheduleID.String() + " ==> " + err.Error())
	} else {
		glog.Info("Request fired for schedule id: " + scheduleID.String() + " ==> " + string(requestDump))
	}
}

// setRequestHeaders sets the required headers for the request
func setRequestHeaders(req *http.Request, input store.Schedule) {
	req.Header.Set("Content-Type", "application/json")
	for header, value := range input.Callback.(*store.HttpCallback).Details.Headers {
		req.Header.Set(header, value)
	}

	// Add scheduleId in header
	req.Header.Set(constants.ScheduleIdHeader, input.ScheduleId.String())

	// ParentScheduleId needed for cron-schedule callbacks
	if !util.IsZeroUUID(input.ParentScheduleId) {
		glog.Infof("Adding ParentScheduleId: %s in http callback header for scheduleId: %s", input.ParentScheduleId.String(), input.ScheduleId.String())
		req.Header.Set(constants.ParentScheduleId, input.ParentScheduleId.String())
	}

	glog.Infof("http callback headers: %v for scheduleId: %s", req.Header, input.ScheduleId.String())
}

// handleResponseDump logs the response dump or error if it occurs, and logs the callback failure if an error exists
func handleResponseDump(input store.Schedule, response *http.Response, attempts int, err error) {
	if err != nil {
		glog.Errorf("Callback failed schedule id: %s during attempt: %d with error %s", input.ScheduleId.String(), attempts, err.Error())
	} else {
		defer response.Body.Close()
		body, er := httputil.DumpResponse(response, true)
		if er != nil {
			glog.Errorf("Response dump failed with error for schedule id: %s ==> %s", input.ScheduleId.String(), er.Error())
		} else {
			glog.Infof("Response received for schedule id: %s ==> %s", input.ScheduleId.String(), string(body))
		}
	}
}

func (c *Connector) recordHttpCallbackRetry(appId string, partitionId int) {
	if c.Monitoring != nil && c.Monitoring.StatsDClient != nil {
		key := constants.HttpRetry + constants.DOT + appId + constants.DOT + strconv.Itoa(partitionId)
		c.Monitoring.StatsDClient.Increment(key)
	}
}

func (c *Connector) recordHTTPCallbackSuccess(appId string, partitionId int) {
	if c.Monitoring != nil && c.Monitoring.StatsDClient != nil {
		key := getKey(appId, partitionId) + constants.DOT + constants.Success
		c.Monitoring.StatsDClient.Increment(key)
	}
}

func (c *Connector) recordHTTPCallbackFailure(appId string, partitionId int) {
	if c.Monitoring != nil && c.Monitoring.StatsDClient != nil {
		key := getKey(appId, partitionId) + constants.DOT + constants.Fail
		c.Monitoring.StatsDClient.Increment(key)
	}
}

func (c *Connector) recordTiming(do func() (*http.Response, error), bucket string) (*http.Response, error) {
	var timing statsd.Timing
	if c.Monitoring != nil && c.Monitoring.StatsDClient != nil {
		timing = c.Monitoring.StatsDClient.NewTiming()
	}

	response, err := do()

	if c.Monitoring != nil && c.Monitoring.StatsDClient != nil {
		timing.Send(bucket)
		c.Monitoring.StatsDClient.Increment(bucket)
	}

	return response, err
}

// processSchedule processes a single ScheduleWrapper, executing the retryPost function and handling the callback result
func (c *Connector) processSchedule(scheduleWrapper store.ScheduleWrapper) {
	result := scheduleWrapper.Schedule
	app := scheduleWrapper.App
	isReconciliation := scheduleWrapper.IsReconciliation

	glog.Infof("Callback fired for schedule with schedule id %s and schedule entity %+v", result.ScheduleId.String(), result)
	key := constants.HttpCallback + constants.DOT + result.AppId + constants.DOT + strconv.Itoa(result.PartitionId)
	response, err := c.recordTiming(func() (response *http.Response, err error) {
		return c.retryPost(result, app)
	}, key)

	c.handleCallbackResult(response, err, result, app, isReconciliation)
}

// handleCallbackResult processes the result of a callback, updating the schedule status and sending the updated ScheduleWrapper to the AggregationTaskQueue
func (c *Connector) handleCallbackResult(response *http.Response, err error, result store.Schedule, app store.App, isReconciliation bool) {
	if err != nil {
		c.recordHTTPCallbackFailure(result.AppId, result.PartitionId)
		glog.Errorf("Callback failed for schedule id %s with error %s", result.ScheduleId.String(), err.Error())

		result.Status = store.Failure
		result.ErrorMessage = trim(err.Error())
	} else if !isSuccess(response) {
		c.recordHTTPCallbackFailure(result.AppId, result.PartitionId)
		glog.Errorf("Callback failed for schedule id %s with response %+v", result.ScheduleId.String(), response)

		result.Status = store.Failure
		result.ErrorMessage = trim(response.Status)
	} else {
		c.recordHTTPCallbackSuccess(result.AppId, result.PartitionId)
		glog.Infof("Callback success for schedule id %s with response %+v", result.ScheduleId.String(), response)

		result.Status = store.Success
		result.ErrorMessage = ""
	}

	if isReconciliation {
		result.UpdateReconciliationHistory(result.Status, result.ErrorMessage)
	}

	store.AggregationTaskQueue <- store.ScheduleWrapper{
		Schedule: result,
		App:      app,
	}
}

// listen processes ScheduleWrapper items from the provided channel
func (c *Connector) listen(buf chan store.ScheduleWrapper) {
	for sw := range buf {
		c.processSchedule(sw)
	}
}

// retryPost attempts to execute an HTTP request according to the schedule and app provided, retrying up to the specified maximum number of attempts
func (c *Connector) retryPost(input store.Schedule, app store.App) (*http.Response, error) {
	defer func() {
		if r := recover(); r != nil {
			c.Monitoring.StatsDClient.Increment(constants.Panic + constants.DOT + "RetryPost")
			glog.Errorf("Recovered in RetryPost from error %s with stacktrace %s", r, string(debug.Stack()))
		}
	}()

	attempts := 0
	maxAttempts := 3

	for {
		attempts++
		glog.Infof("\nPOSTING SCHEDULE %s , ATTEMPT %d ", input.ScheduleId, attempts)
		url := input.Callback.(*store.HttpCallback).Details.Url
		fmt.Println("\nURL:>", url)

		req, err := createRequest(input)
		if err != nil {
			return nil, err
		}

		response, err := c.HttpClient.Do(req)
		handleResponseDump(input, response, attempts, err)

		retry := shouldRetry(maxAttempts, attempts, response)
		if retry {
			c.recordHttpCallbackRetry(input.AppId, input.PartitionId)
		} else {
			return response, err
		}
	}
}

func (c *Connector) createWorkerPool(buf chan store.ScheduleWrapper) {
	noOfWorkers := c.Config.HttpConnector.Routines
	for i := 0; i < noOfWorkers; i++ {
		fmt.Printf("\nInitializing worker for *HTTP* connector %d", i)
		go c.listen(buf)
	}
}

func (c *Connector) initHttpWorkers() {
	go c.createWorkerPool(store.HttpTaskQueue)
}
