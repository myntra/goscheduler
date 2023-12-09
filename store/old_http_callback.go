package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/golang/glog"
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
	glog.Infof("Pushing to OldHttpTaskQueue!!!!!!!!!!!!!!!!!!!!")
	OldHttpTaskQueue <- wrapper
	glog.Infof("Pushed to OldHttpTaskQueue!!!!!!!!!!!!!!!!!!!!")
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
