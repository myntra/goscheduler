package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/myntra/goscheduler/constants"
)

func (h *AirbusCallback) GetType() string {
	return constants.AirbusCallback
}

func (h *AirbusCallback) GetDetails() (string, error) {
	details, err := json.Marshal(h)
	return string(details), err
}

func (h *AirbusCallback) Marshal(m map[string]interface{}) error {
	eventName, ok := m["call_back_url"].(string)
	if !ok {
		return fmt.Errorf("wrong call_back_url")
	}

	headers, ok := m["headers"].(map[string]string)
	if !ok {
		return fmt.Errorf("wrong headers")
	}

	h.EventName = eventName
	h.Headers = headers

	return nil
}

// UnmarshalJSON Implement UnmarshalJSON for HttpCallback
func (h *AirbusCallback) UnmarshalJSON(data []byte) error {
	type Alias AirbusCallback
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(h),
	}
	return json.Unmarshal(data, &aux)
}

func (h *AirbusCallback) Invoke(wrapper ScheduleWrapper) error {
	AirbusTaskQueue <- wrapper
	return nil
}

func (h *AirbusCallback) Validate() error {
	// Checking if URL is empty
	if h.EventName == "" {
		return errors.New("eventName cannot be empty")
	}

	return nil
}
