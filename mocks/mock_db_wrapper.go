// Code generated by MockGen. DO NOT EDIT.
// Source: ./db_wrapper/db_wrapper.go

// Package mocks is a generated GoMock package.
package mocks

import (
	db_wrapper "github.com/myntra/goscheduler/db_wrapper"
	reflect "reflect"

	gocql "github.com/gocql/gocql"
	gomock "github.com/golang/mock/gomock"
)

// MockSessionInterface is a mock of SessionInterface interface.
type MockSessionInterface struct {
	ctrl     *gomock.Controller
	recorder *MockSessionInterfaceMockRecorder
}

// MockSessionInterfaceMockRecorder is the mock recorder for MockSessionInterface.
type MockSessionInterfaceMockRecorder struct {
	mock *MockSessionInterface
}

// NewMockSessionInterface creates a new mock instance.
func NewMockSessionInterface(ctrl *gomock.Controller) *MockSessionInterface {
	mock := &MockSessionInterface{ctrl: ctrl}
	mock.recorder = &MockSessionInterfaceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockSessionInterface) EXPECT() *MockSessionInterfaceMockRecorder {
	return m.recorder
}

// Close mocks base method.
func (m *MockSessionInterface) Close() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Close")
}

// Close indicates an expected call of Close.
func (mr *MockSessionInterfaceMockRecorder) Close() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Close", reflect.TypeOf((*MockSessionInterface)(nil).Close))
}

// ExecuteBatch mocks base method.
func (m *MockSessionInterface) ExecuteBatch(batch *gocql.Batch) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ExecuteBatch", batch)
	ret0, _ := ret[0].(error)
	return ret0
}

// ExecuteBatch indicates an expected call of ExecuteBatch.
func (mr *MockSessionInterfaceMockRecorder) ExecuteBatch(batch interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ExecuteBatch", reflect.TypeOf((*MockSessionInterface)(nil).ExecuteBatch), batch)
}

// Query mocks base method.
func (m *MockSessionInterface) Query(arg0 string, arg1 ...interface{}) db_wrapper.QueryInterface {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0}
	for _, a := range arg1 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Query", varargs...)
	ret0, _ := ret[0].(db_wrapper.QueryInterface)
	return ret0
}

// Query indicates an expected call of Query.
func (mr *MockSessionInterfaceMockRecorder) Query(arg0 interface{}, arg1 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0}, arg1...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Query", reflect.TypeOf((*MockSessionInterface)(nil).Query), varargs...)
}

// MockQueryInterface is a mock of QueryInterface interface.
type MockQueryInterface struct {
	ctrl     *gomock.Controller
	recorder *MockQueryInterfaceMockRecorder
}

// MockQueryInterfaceMockRecorder is the mock recorder for MockQueryInterface.
type MockQueryInterfaceMockRecorder struct {
	mock *MockQueryInterface
}

// NewMockQueryInterface creates a new mock instance.
func NewMockQueryInterface(ctrl *gomock.Controller) *MockQueryInterface {
	mock := &MockQueryInterface{ctrl: ctrl}
	mock.recorder = &MockQueryInterfaceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockQueryInterface) EXPECT() *MockQueryInterfaceMockRecorder {
	return m.recorder
}

// Bind mocks base method.
func (m *MockQueryInterface) Bind(arg0 ...interface{}) db_wrapper.QueryInterface {
	m.ctrl.T.Helper()
	varargs := []interface{}{}
	for _, a := range arg0 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Bind", varargs...)
	ret0, _ := ret[0].(db_wrapper.QueryInterface)
	return ret0
}

// Bind indicates an expected call of Bind.
func (mr *MockQueryInterfaceMockRecorder) Bind(arg0 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Bind", reflect.TypeOf((*MockQueryInterface)(nil).Bind), arg0...)
}

// Consistency mocks base method.
func (m *MockQueryInterface) Consistency(c gocql.Consistency) db_wrapper.QueryInterface {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Consistency", c)
	ret0, _ := ret[0].(db_wrapper.QueryInterface)
	return ret0
}

// Consistency indicates an expected call of Consistency.
func (mr *MockQueryInterfaceMockRecorder) Consistency(c interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Consistency", reflect.TypeOf((*MockQueryInterface)(nil).Consistency), c)
}

// Exec mocks base method.
func (m *MockQueryInterface) Exec() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Exec")
	ret0, _ := ret[0].(error)
	return ret0
}

// Exec indicates an expected call of Exec.
func (mr *MockQueryInterfaceMockRecorder) Exec() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Exec", reflect.TypeOf((*MockQueryInterface)(nil).Exec))
}

// Iter mocks base method.
func (m *MockQueryInterface) Iter() db_wrapper.IterInterface {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Iter")
	ret0, _ := ret[0].(db_wrapper.IterInterface)
	return ret0
}

// Iter indicates an expected call of Iter.
func (mr *MockQueryInterfaceMockRecorder) Iter() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Iter", reflect.TypeOf((*MockQueryInterface)(nil).Iter))
}

// MapScan mocks base method.
func (m_2 *MockQueryInterface) MapScan(m map[string]interface{}) error {
	m_2.ctrl.T.Helper()
	ret := m_2.ctrl.Call(m_2, "MapScan", m)
	ret0, _ := ret[0].(error)
	return ret0
}

// MapScan indicates an expected call of MapScan.
func (mr *MockQueryInterfaceMockRecorder) MapScan(m interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "MapScan", reflect.TypeOf((*MockQueryInterface)(nil).MapScan), m)
}

// PageSize mocks base method.
func (m *MockQueryInterface) PageSize(n int) db_wrapper.QueryInterface {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PageSize", n)
	ret0, _ := ret[0].(db_wrapper.QueryInterface)
	return ret0
}

// PageSize indicates an expected call of PageSize.
func (mr *MockQueryInterfaceMockRecorder) PageSize(n interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PageSize", reflect.TypeOf((*MockQueryInterface)(nil).PageSize), n)
}

// PageState mocks base method.
func (m *MockQueryInterface) PageState(state []byte) db_wrapper.QueryInterface {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PageState", state)
	ret0, _ := ret[0].(db_wrapper.QueryInterface)
	return ret0
}

// PageState indicates an expected call of PageState.
func (mr *MockQueryInterfaceMockRecorder) PageState(state interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PageState", reflect.TypeOf((*MockQueryInterface)(nil).PageState), state)
}

// RetryPolicy mocks base method.
func (m *MockQueryInterface) RetryPolicy(policy gocql.RetryPolicy) db_wrapper.QueryInterface {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RetryPolicy", policy)
	ret0, _ := ret[0].(db_wrapper.QueryInterface)
	return ret0
}

// RetryPolicy indicates an expected call of RetryPolicy.
func (mr *MockQueryInterfaceMockRecorder) RetryPolicy(policy interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RetryPolicy", reflect.TypeOf((*MockQueryInterface)(nil).RetryPolicy), policy)
}

// Scan mocks base method.
func (m *MockQueryInterface) Scan(arg0 ...interface{}) error {
	m.ctrl.T.Helper()
	varargs := []interface{}{}
	for _, a := range arg0 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Scan", varargs...)
	ret0, _ := ret[0].(error)
	return ret0
}

// Scan indicates an expected call of Scan.
func (mr *MockQueryInterfaceMockRecorder) Scan(arg0 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Scan", reflect.TypeOf((*MockQueryInterface)(nil).Scan), arg0...)
}

// MockIterInterface is a mock of IterInterface interface.
type MockIterInterface struct {
	ctrl     *gomock.Controller
	recorder *MockIterInterfaceMockRecorder
}

// MockIterInterfaceMockRecorder is the mock recorder for MockIterInterface.
type MockIterInterfaceMockRecorder struct {
	mock *MockIterInterface
}

// NewMockIterInterface creates a new mock instance.
func NewMockIterInterface(ctrl *gomock.Controller) *MockIterInterface {
	mock := &MockIterInterface{ctrl: ctrl}
	mock.recorder = &MockIterInterfaceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockIterInterface) EXPECT() *MockIterInterfaceMockRecorder {
	return m.recorder
}

// Close mocks base method.
func (m *MockIterInterface) Close() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Close")
	ret0, _ := ret[0].(error)
	return ret0
}

// Close indicates an expected call of Close.
func (mr *MockIterInterfaceMockRecorder) Close() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Close", reflect.TypeOf((*MockIterInterface)(nil).Close))
}

// MapScan mocks base method.
func (m_2 *MockIterInterface) MapScan(m map[string]interface{}) bool {
	m_2.ctrl.T.Helper()
	ret := m_2.ctrl.Call(m_2, "MapScan", m)
	ret0, _ := ret[0].(bool)
	return ret0
}

// MapScan indicates an expected call of MapScan.
func (mr *MockIterInterfaceMockRecorder) MapScan(m interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "MapScan", reflect.TypeOf((*MockIterInterface)(nil).MapScan), m)
}

// PageState mocks base method.
func (m *MockIterInterface) PageState() []byte {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PageState")
	ret0, _ := ret[0].([]byte)
	return ret0
}

// PageState indicates an expected call of PageState.
func (mr *MockIterInterfaceMockRecorder) PageState() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PageState", reflect.TypeOf((*MockIterInterface)(nil).PageState))
}

// Scan mocks base method.
func (m *MockIterInterface) Scan(arg0 ...interface{}) bool {
	m.ctrl.T.Helper()
	varargs := []interface{}{}
	for _, a := range arg0 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Scan", varargs...)
	ret0, _ := ret[0].(bool)
	return ret0
}

// Scan indicates an expected call of Scan.
func (mr *MockIterInterfaceMockRecorder) Scan(arg0 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Scan", reflect.TypeOf((*MockIterInterface)(nil).Scan), arg0...)
}