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

type App struct {
	AppId         string        `json:"appId"`
	Partitions    uint32        `json:"partitions"`
	Active        bool          `json:"active"`
	Configuration Configuration `json:"configuration"`
}

type AppErrorResponse struct {
	Errors []string `json:"errors"`
}

// GetMaxTTL gets maxCassandraTTL in seconds
func (a App) GetMaxTTL(maxTTL int) int {
	if a.Configuration.FutureScheduleCreationPeriod == 0 {
		return 60 * 60 * 24 * maxTTL
	}

	return 60 * 60 * 24 * a.Configuration.FutureScheduleCreationPeriod
}

// GetBufferTTL gets bufferCassandraTTL in seconds
func (a App) GetBufferTTL(bufferTTL int) int {
	if a.Configuration.FiredScheduleRetentionPeriod == 0 {
		return 60 * 60 * 24 * bufferTTL
	}

	return 60 * 60 * 24 * a.Configuration.FiredScheduleRetentionPeriod
}
