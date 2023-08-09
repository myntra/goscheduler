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

package db_wrapper

import (
	"github.com/gocql/gocql"
)

// gocql does not use interfaces, which makes mocking impossible.
// So pass our own wrapper implementation around.
// This allows us to use gomock in tests.

// For variadic arguments - https://github.com/golang/mock/pull/595

// SessionInterface allows gomock mock of gocql.Session
type SessionInterface interface {
	Query(string, ...interface{}) QueryInterface
	ExecuteBatch(batch *gocql.Batch) error
	Close()
}

// QueryInterface allows gomock mock of gocql.Query
type QueryInterface interface {
	Bind(...interface{}) QueryInterface
	Exec() error
	Iter() IterInterface
	Scan(...interface{}) error
	MapScan(m map[string]interface{}) error
	Consistency(c gocql.Consistency) QueryInterface
	PageState(state []byte) QueryInterface
	PageSize(n int) QueryInterface
	RetryPolicy(policy gocql.RetryPolicy) QueryInterface
}

// IterInterface allows gomock mock of gocql.Iter
type IterInterface interface {
	Close() error
	Scan(...interface{}) bool
	MapScan(m map[string]interface{}) bool
	PageState() []byte
}

// Session is a wrapper for a session for mockability.
type Session struct {
	session *gocql.Session
}

// Query is a wrapper for a query for mockability.
type Query struct {
	query *gocql.Query
}

// Iter is a wrapper for an iter for mockability.
type Iter struct {
	iter *gocql.Iter
}

// NewSession instantiates a new Session
func NewSession(session *gocql.Session) SessionInterface {
	return &Session{
		session,
	}
}

// NewQuery instantiates a new Query
func NewQuery(query *gocql.Query) QueryInterface {
	return &Query{
		query,
	}
}

// NewIter instantiates a new Iter
func NewIter(iter *gocql.Iter) IterInterface {
	return &Iter{
		iter,
	}
}

// ExecuteBatch wraps the session's ExecuteBatch method
func (s *Session) ExecuteBatch(batch *gocql.Batch) error {
	return s.session.ExecuteBatch(batch)
}

// Query wraps the session's Query method
func (s *Session) Query(stmt string, values ...interface{}) QueryInterface {
	return NewQuery(s.session.Query(stmt, values...))
}

// Close wraps the session's Close method
func (s *Session) Close() {
	s.session.Close()
}

// Bind wraps the query's Bind method
func (q *Query) Bind(v ...interface{}) QueryInterface {
	return NewQuery(q.query.Bind(v...))
}

// Exec wraps the query's Exec method
func (q *Query) Exec() error {
	return q.query.Exec()
}

// Iter wraps the query's Iter method
func (q *Query) Iter() IterInterface {
	return NewIter(q.query.Iter())
}

// Scan wraps the query's Scan method
func (q *Query) Scan(dest ...interface{}) error {
	return q.query.Scan(dest...)
}

// MapScan wraps the query's MapScan method
func (q *Query) MapScan(m map[string]interface{}) error {
	return q.query.MapScan(m)
}

// Consistency wraps the query's Consistency method
func (q *Query) Consistency(c gocql.Consistency) QueryInterface {
	return NewQuery(q.query.Consistency(c))
}

// PageState wraps the query's PageState method
func (q *Query) PageState(state []byte) QueryInterface {
	return NewQuery(q.query.PageState(state))
}

// PageSize wraps the query's PageSize method
func (q *Query) PageSize(n int) QueryInterface {
	return NewQuery(q.query.PageSize(n))
}

// RetryPolicy wraps the query's RetryPolicy method
func (q *Query) RetryPolicy(policy gocql.RetryPolicy) QueryInterface {
	return NewQuery(q.query.RetryPolicy(policy))
}

// Scan wraps iter's Scan method
func (i *Iter) Scan(dest ...interface{}) bool {
	return i.iter.Scan(dest...)
}

// MapScan wraps iter's MapScan method
func (i *Iter) MapScan(m map[string]interface{}) bool {
	return i.iter.MapScan(m)
}

// PageState wraps iter's PageState method
func (i *Iter) PageState() []byte {
	return i.iter.PageState()
}

// Close wraps iter's Close method
func (i *Iter) Close() error {
	return i.iter.Close()
}