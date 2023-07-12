package store

import (
	"encoding/json"
)

type Callback interface {
	GetType() string
	GetDetails() (string, error)
	Marshal(map[string]interface{}) error
	Invoke(wrapper ScheduleWrapper) error
	Validate() error // Added
	json.Unmarshaler // Add this
}
