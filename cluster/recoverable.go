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

package cluster

import (
	"errors"
	"sync"

	"github.com/myntra/goscheduler/cluster_entity"
)

var errRecoverableEntityStopped = errors.New("recoverable entity is stopped")

type recoverableEntityState struct {
	mu      sync.Mutex
	stopped bool
}

// RecoverableEntity is a wrapper for a recoverable entity.
type RecoverableEntity struct {
	Obj       cluster_entity.Entity // The underlying entity.
	Recovered int                   // The number of times the entity was recovered.
	state     *recoverableEntityState
}

func newRecoverableEntity(entity cluster_entity.Entity) RecoverableEntity {
	return RecoverableEntity{
		Obj:   entity,
		state: &recoverableEntityState{},
	}
}

// Stop stops the recoverable entity.
func (r RecoverableEntity) Stop() {
	if r.state == nil {
		r.Obj.Stop()
		return
	}
	r.state.mu.Lock()
	defer r.state.mu.Unlock()
	r.state.stopped = true
	r.Obj.Stop()
}

// Init initializes the recoverable entity before its asynchronous start.
func (r RecoverableEntity) Init() error {
	if r.state == nil {
		return r.Obj.Init()
	}
	r.state.mu.Lock()
	defer r.state.mu.Unlock()
	if r.state.stopped {
		return errRecoverableEntityStopped
	}
	return r.Obj.Init()
}

// Start runs an initialized recoverable entity.
func (r RecoverableEntity) Start() {
	defer func() {
		// Recover from a panic and restart the entity if needed.
		if err := recover(); err != nil {
			r.Recovered++
			if initErr := r.Init(); initErr == nil {
				r.Obj.Start()
			}
		}
	}()

	r.Obj.Start()
}
