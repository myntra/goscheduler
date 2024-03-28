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
	"github.com/myntra/goscheduler/constants"
	"net/url"
)

func (h *HTTPCallback) GetType() string {
	return constants.HttpCallback
}

func (h *HTTPCallback) GetDetails() (string, error) {
	details, err := json.Marshal(h)
	return string(details), err
}

func (h *HTTPCallback) Marshal(m map[string]interface{}) error {
	callbackUrl, ok := m["call_back_url"].(string)
	if !ok {
		return fmt.Errorf("wrong call_back_url")
	}

	headers, ok := m["headers"].(map[string]string)
	if !ok {
		return fmt.Errorf("wrong headers")
	}

	h.Url = callbackUrl
	h.Headers = headers

	return nil
}

// UnmarshalJSON Implement UnmarshalJSON for HttpCallback
func (h *HTTPCallback) UnmarshalJSON(data []byte) error {
	type Alias HTTPCallback
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(h),
	}
	return json.Unmarshal(data, &aux)
}

func (h *HTTPCallback) Invoke(wrapper ScheduleWrapper) error {
	OldHttpTaskQueue <- wrapper
	return nil
}

func (h *HTTPCallback) Validate() error {
	// Checking if URL is empty
	if h.Url == "" {
		return errors.New("url cannot be empty")
	}

	// Checking if the URL is valid
	_, err := url.ParseRequestURI(h.Url)
	if err != nil {
		return errors.New("invalid url")
	}

	return nil
}
