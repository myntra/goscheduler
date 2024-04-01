package store

type Configuration struct {
	FutureScheduleCreationPeriod int `json:"futureScheduleCreationPeriod,omitempty"`
	FiredScheduleRetentionPeriod int `json:"firedScheduleRetentionPeriod,omitempty"`
	PayloadSize                  int `json:"payloadSize,omitempty"`
	HttpRetries                  int `json:"httpRetries,omitempty"`
	HttpTimeout                  int `json:"httpTimeout,omitempty"`
}
