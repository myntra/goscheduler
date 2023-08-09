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

// EntityIDs contains a slice of entity IDs.
type EntityIDs struct {
	Ids []string
}

// AppNames contains a slice of application names.
type AppNames struct {
	Names []string
}

// Request represents a request to be sent to a remote node.
type Request struct {
	entity   interface{} // The entity to be sent.
	method   string      // The method to be called.
	destNode string      // The destination node address.
}

// Response represents a response received from a remote node.
type Response struct {
	ServerAddress string // The server address that sent the response.
	Error         string // The error message, if any.
	Status        int    // The status of the response.
}

const (
	SUCCESS = iota // The request was successful.
	FAILED         // The request failed.
)

