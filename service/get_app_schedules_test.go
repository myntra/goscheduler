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

package service

import (
	s "github.com/myntra/goscheduler/store"
	"net/http"
	"net/url"
	"testing"
)

func TestService_parse(t *testing.T) {
	request := http.Request{
		URL: &url.URL{
			RawQuery: "",
		},
	}

	type Params struct {
		Size      int64
		Status    s.Status
		StartTime int64
		EndTime   int64
	}

	//check valid cases
	for _, test := range []struct {
		Input    string
		Expected Params
	}{
		{"size=10&status=ERROR&start_time=2020-12-01 00:00:00&end_time=2020-12-02 00:00:00",
			Params{10, s.Status("ERROR"), 1606761000, 1606847400}},
		{"size=2000&status=SUCCESS&start_time=2020-12-01 00:00:00&end_time=2020-12-02 00:00:00",
			Params{2000, s.Status("SUCCESS"), 1606761000, 1606847400}},
		{"size=820&status=FAILURE&start_time=2020-12-01 00:00:00&end_time=2020-12-02 00:00:00",
			Params{820, s.Status("FAILURE"), 1606761000, 1606847400}},
		{"size=1999999920&status=SCHEDULED&start_time=2020-12-01 00:00:00&end_time=2020-12-02 00:00:00",
			Params{1999999920, s.Status("SCHEDULED"), 1606761000, 1606847400}},
	} {
		request.URL.RawQuery = test.Input
		if size, status, timeRange, _, _, err := parse(&request); err != nil {
			t.Errorf("Got error %s for input %s", err, test.Input)
		} else if size != test.Expected.Size {
			t.Errorf("Got size: %d for input %s, expected: %d", size, test.Input, test.Expected.Size)
		} else if status != test.Expected.Status {
			t.Errorf("Got status: %s for input %s, expected: %s", status, test.Input, test.Expected.Status)
		} else if timeRange.StartTime.Unix() != test.Expected.StartTime {
			t.Errorf("Got startTime: %v for input %v, expected: %d", timeRange.StartTime.Unix(), test.Input, test.Expected.StartTime)
		} else if timeRange.EndTime.Unix() != test.Expected.EndTime {
			t.Errorf("Got endTime: %v for input %s, expected: %d", timeRange.EndTime.Unix(), test.Input, test.Expected.EndTime)
		}
	}

	//check invalid cases
	for _, test := range []struct {
		Input    string
		Expected string
	}{
		{"size=10&status=ERROR&start_time=2020-12-01 00:00:00&end_time=2020-12-02",
			"Incorrect end_time query parameter format parsing time \"2020-12-02\" as \"2006-01-02 15:04:05\": cannot parse \"\" as \"15\" (expected format 2006-01-02 15:04:05)"},
		{"size=ABC&status=SUCCESS", "strconv.ParseInt: parsing \"ABC\": invalid syntax"},
		{"size=!@#$&status=FAILURE", "strconv.ParseInt: parsing \"!@#$\": invalid syntax"},
		{"size=()&status=SCHEDULED", "strconv.ParseInt: parsing \"()\": invalid syntax"},
	} {
		request.URL.RawQuery = test.Input
		if _, _, _, _, _, err := parse(&request); err != nil && err.Error() != test.Expected {
			t.Errorf("Got error %s for input %s", err, test.Input)
		}
	}
}

func TestService_parseDate(t *testing.T) {
	//check valid cases
	for _, test := range []struct {
		Input    string
		Expected int64
	}{
		{"2021-01-25 14:00:11", 1611563400},
		{"2021-02-28 13:50:51", 1614500400},
		{"2021-03-05 07:50:51", 1614910800},
		{"2021-01-25 23:50:00.0000", 1611598800},
	} {
		if timestamp, err := parseDate(test.Input); err != nil {
			t.Errorf("Got error %s for input %s", err, test.Input)
		} else if timestamp.Unix() != test.Expected {
			t.Errorf("Got timestamp: %v for input %s, expected: %d", timestamp.Unix(), test.Input, test.Expected)
		}
	}

	//check invalid cases
	for _, test := range []struct {
		Input    string
		Expected string
	}{
		{"2021-01-25", "parsing time \"2021-01-25\" as \"2006-01-02 15:04:05\": cannot parse \"\" as \"15\""},
		{"2021-02-28 07", "parsing time \"2021-02-28 07\" as \"2006-01-02 15:04:05\": cannot parse \"\" as \":\""},
		{"2021-03-05 07:50", "parsing time \"2021-03-05 07:50\" as \"2006-01-02 15:04:05\": cannot parse \"\" as \":\""},
		{"2006-01-02T15:04:05.000Z", "parsing time \"2006-01-02T15:04:05.000Z\" as \"2006-01-02 15:04:05\": cannot parse \"T15:04:05.000Z\" as \" \""},
		{"Mon Jan 2 15:04:05 MST 2006", "parsing time \"Mon Jan 2 15:04:05 MST 2006\" as \"2006-01-02 15:04:05\": cannot parse \"Mon Jan 2 15:04:05 MST 2006\" as \"2006\""},
	} {
		if _, err := parseDate(test.Input); err.Error() != test.Expected {
			t.Errorf("Got error %s for input %s", err, test.Input)
		}
	}
}
