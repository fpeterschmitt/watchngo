// Code generated by MockGen. DO NOT EDIT.
// Source: filter.go

// Package pkg_test is a generated GoMock package.
package pkg_test

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
)

// MockFilter is a mock of Filter interface.
type MockFilter struct {
	ctrl     *gomock.Controller
	recorder *MockFilterMockRecorder
}

// MockFilterMockRecorder is the mock recorder for MockFilter.
type MockFilterMockRecorder struct {
	mock *MockFilter
}

// NewMockFilter creates a new mock instance.
func NewMockFilter(ctrl *gomock.Controller) *MockFilter {
	mock := &MockFilter{ctrl: ctrl}
	mock.recorder = &MockFilterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockFilter) EXPECT() *MockFilterMockRecorder {
	return m.recorder
}

// Match mocks base method.
func (m *MockFilter) Match(file string) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Match", file)
	ret0, _ := ret[0].(bool)
	return ret0
}

// Match indicates an expected call of Match.
func (mr *MockFilterMockRecorder) Match(file interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Match", reflect.TypeOf((*MockFilter)(nil).Match), file)
}