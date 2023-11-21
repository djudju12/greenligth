// Code generated by MockGen. DO NOT EDIT.
// Source: internal/data/tokens.go
//
// Generated by this command:
//
//	mockgen -package mockdb -destination internal/mocks/tokens_mocks.go -source internal/data/tokens.go TokensQuerier
//
// Package mockdb is a generated GoMock package.
package mockdb

import (
	reflect "reflect"
	time "time"

	data "github.com/djudju12/greenlight/internal/data"
	gomock "go.uber.org/mock/gomock"
)

// MockTokenQuerier is a mock of TokenQuerier interface.
type MockTokenQuerier struct {
	ctrl     *gomock.Controller
	recorder *MockTokenQuerierMockRecorder
}

// MockTokenQuerierMockRecorder is the mock recorder for MockTokenQuerier.
type MockTokenQuerierMockRecorder struct {
	mock *MockTokenQuerier
}

// NewMockTokenQuerier creates a new mock instance.
func NewMockTokenQuerier(ctrl *gomock.Controller) *MockTokenQuerier {
	mock := &MockTokenQuerier{ctrl: ctrl}
	mock.recorder = &MockTokenQuerierMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockTokenQuerier) EXPECT() *MockTokenQuerierMockRecorder {
	return m.recorder
}

// DeleteAllForUser mocks base method.
func (m *MockTokenQuerier) DeleteAllForUser(scope string, userID int64) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteAllForUser", scope, userID)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteAllForUser indicates an expected call of DeleteAllForUser.
func (mr *MockTokenQuerierMockRecorder) DeleteAllForUser(scope, userID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteAllForUser", reflect.TypeOf((*MockTokenQuerier)(nil).DeleteAllForUser), scope, userID)
}

// New mocks base method.
func (m *MockTokenQuerier) New(userID int64, ttl time.Duration, scope string) (*data.Token, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "New", userID, ttl, scope)
	ret0, _ := ret[0].(*data.Token)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// New indicates an expected call of New.
func (mr *MockTokenQuerierMockRecorder) New(userID, ttl, scope any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "New", reflect.TypeOf((*MockTokenQuerier)(nil).New), userID, ttl, scope)
}
