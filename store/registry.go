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

import "github.com/myntra/goscheduler/constants"

type Factory func() Callback

var Registry = map[string]Factory{}

func InitializeCallbackRegistry(clientCallbacks map[string]Factory) {
	// default implementations
	defaultCallbacks := map[string]Factory{
		constants.DefaultCallback: func() Callback { return &HttpCallback{} },
	}

	// First, register all client-provided callbacks
	for callbackType, factory := range clientCallbacks {
		Registry[callbackType] = factory
	}

	// Then, register default callbacks only if they haven't been provided by the client
	for callbackType, factory := range defaultCallbacks {
		if _, ok := Registry[callbackType]; !ok {
			Registry[callbackType] = factory
		}
	}
}
