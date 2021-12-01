// Code generated by MockGen. DO NOT EDIT.
// Source: executor.go

// Package pkg_test is a generated GoMock package.
package pkg_test

import (
	reflect "reflect"

	pkg "github.com/Leryan/watchngo/pkg"
	gomock "github.com/golang/mock/gomock"
)

// MockExecutor is a mock of Executor interface.
type MockExecutor struct {
	ctrl     *gomock.Controller
	recorder *MockExecutorMockRecorder
}

// MockExecutorMockRecorder is the mock recorder for MockExecutor.
type MockExecutorMockRecorder struct {
	mock *MockExecutor
}

// NewMockExecutor creates a new mock instance.
func NewMockExecutor(ctrl *gomock.Controller) *MockExecutor {
	mock := &MockExecutor{ctrl: ctrl}
	mock.recorder = &MockExecutorMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockExecutor) EXPECT() *MockExecutorMockRecorder {
	return m.recorder
}

// Exec mocks base method.
func (m *MockExecutor) Exec(event pkg.NotificationEvent, eventFile string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Exec", event, eventFile)
	ret0, _ := ret[0].(error)
	return ret0
}

// Exec indicates an expected call of Exec.
func (mr *MockExecutorMockRecorder) Exec(event, eventFile interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Exec", reflect.TypeOf((*MockExecutor)(nil).Exec), event, eventFile)
}

// Running mocks base method.
func (m *MockExecutor) Running() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Running")
	ret0, _ := ret[0].(bool)
	return ret0
}

// Running indicates an expected call of Running.
func (mr *MockExecutorMockRecorder) Running() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Running", reflect.TypeOf((*MockExecutor)(nil).Running))
}