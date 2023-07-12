package store

import (
	"encoding/json"
	"errors"
	"testing"
)

// Test for GetType method
func TestGetType(t *testing.T) {
	callback := HttpCallback{
		Type: "http",
	}

	if callback.GetType() != "http" {
		t.Errorf("GetType() failed, expected %v, got %v", "http", callback.GetType())
	}
}

// Test for GetDetails method
func TestGetDetails(t *testing.T) {
	details := Details{
		Url:    "https://example.com",
		Method: "GET",
		Headers: map[string]string{
			"Authorization": "Bearer token",
		},
	}

	callback := HttpCallback{
		Type:    "http",
		Details: details,
	}

	detailsJson, err := json.Marshal(details)
	if err != nil {
		t.Errorf("Error while marshaling details: %s", err.Error())
	}

	got, err := callback.GetDetails()
	if err != nil {
		t.Errorf("GetDetails() failed with error: %s", err.Error())
	}

	if string(detailsJson) != got {
		t.Errorf("GetDetails() failed, expected %v, got %v", string(detailsJson), got)
	}
}

// Test for Marshal method
func TestMarshal(t *testing.T) {
	details := Details{
		Url:    "https://example.com",
		Method: "GET",
		Headers: map[string]string{
			"Authorization": "Bearer token",
		},
	}

	detailsJson, err := json.Marshal(details)
	if err != nil {
		t.Errorf("Error while marshaling details: %s", err.Error())
	}

	callback := &HttpCallback{}
	err = callback.Marshal(map[string]interface{}{
		"callback_type":    "http",
		"callback_details": string(detailsJson),
	})

	if err != nil {
		t.Errorf("Marshal() failed with error: %s", err.Error())
	}

	if callback.Type != "http" || callback.Details.Url != details.Url || callback.Details.Method != details.Method || callback.Details.Headers["Authorization"] != details.Headers["Authorization"] {
		t.Errorf("Marshal() failed, expected %v, got %v", details, callback.Details)
	}
}

// Test for Validate method
func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		details Details
		want    error
	}{
		{
			name: "Empty URL",
			details: Details{
				Url:    "",
				Method: "GET",
			},
			want: errors.New("url cannot be empty"),
		},
		{
			name: "Invalid URL",
			details: Details{
				Url:    "invalid_url",
				Method: "GET",
			},
			want: errors.New("invalid url"),
		},
		{
			name: "Empty Method",
			details: Details{
				Url:    "https://example.com",
				Method: "",
			},
			want: errors.New("method cannot be empty"),
		},
		{
			name: "Invalid Method",
			details: Details{
				Url:    "https://example.com",
				Method: "INVALID",
			},
			want: errors.New("Invalid http callback method INVALID"),
		},
		{
			name: "Valid HttpCallback",
			details: Details{
				Url:    "https://example.com",
				Method: "GET",
			},
			want: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			callback := &HttpCallback{
				Type:    "http",
				Details: test.details,
			}

			if err := callback.Validate(); err != nil {
				if test.want == nil || err.Error() != test.want.Error() {
					t.Errorf("Validate() failed, expected %v, got %v", test.want, err)
				}
			}
		})
	}
}
