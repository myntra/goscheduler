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
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

type Details struct {
	Url     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
}

type HttpCallback struct {
	Type    string  `json:"type"`
	Details Details `json:"details"`
}

func (h *HttpCallback) GetType() string {
	return h.Type
}

func (h *HttpCallback) GetDetails() (string, error) {
	details, err := json.Marshal(h.Details)
	return string(details), err
}

func (h *HttpCallback) Marshal(m map[string]interface{}) error {
	callbackType, ok := m["callback_type"].(string)
	if !ok {
		return fmt.Errorf("wrong type for callback_type")
	}

	callbackDetailsJSON, ok := m["callback_details"].(string)
	if !ok {
		return fmt.Errorf("wrong type for callback_details")
	}

	var details Details
	err := json.Unmarshal([]byte(callbackDetailsJSON), &details)
	if err != nil {
		return err
	}

	h.Type = callbackType
	h.Details = details

	return nil
}

// UnmarshalJSON Implement UnmarshalJSON for HttpCallback
func (h *HttpCallback) UnmarshalJSON(data []byte) error {
	type Alias HttpCallback
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(h),
	}
	return json.Unmarshal(data, &aux)
}

func (h HttpCallback) Invoke(wrapper ScheduleWrapper) error {
	HttpTaskQueue <- wrapper
	return nil
}

func (h *HttpCallback) Validate() error {
	// Checking if URL is empty
	if h.Details.Url == "" {
		return errors.New("url cannot be empty")
	}

	// Checking if the URL is valid
	_, err := url.ParseRequestURI(h.Details.Url)
	if err != nil {
		return errors.New("invalid url")
	}

	// Checking if method is empty
	if h.Details.Method == "" {
		return errors.New("method cannot be empty")
	}

	// Checking if method is valid
	if ok := isValidRequestMethod(h.Details.Method); !ok {
		return errors.New(fmt.Sprintf("Invalid http callback method %s", h.Details.Method))
	}

	return nil
}

// Check if a given HTTP request method string is valid
func isValidRequestMethod(method string) bool {
	_, ok := validMethods[method]
	return ok
}

var validMethods = map[string]bool{
	http.MethodGet:     true,
	http.MethodHead:    true,
	http.MethodPost:    true,
	http.MethodPut:     true,
	http.MethodPatch:   true,
	http.MethodDelete:  true,
	http.MethodConnect: true,
	http.MethodOptions: true,
	http.MethodTrace:   true,
}
