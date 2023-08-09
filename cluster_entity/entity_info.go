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

package cluster_entity

import (
	"strconv"
	"strings"
)

// EntityInfo represents a cluster entity and its properties
type EntityInfo struct {
	Id      string // ID of the entity
	Node    string // Node that the entity is running on
	Status  int    // Status code for the entity
	History string // History of the entity
}

// GetAppName returns the name of the application that the entity belongs to
func (entity EntityInfo) GetAppName() string {
	appName := entity.Id[:strings.LastIndex(entity.Id, ".")]
	return appName
}

// GetPartitionId returns the partition ID of the entity
func (entity EntityInfo) GetPartitionId() int {
	partitionId, _ := strconv.Atoi(entity.Id[strings.LastIndex(entity.Id, ".")+1:])
	return partitionId
}
