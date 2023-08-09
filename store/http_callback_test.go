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
