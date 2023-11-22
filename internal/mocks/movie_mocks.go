// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/djudju12/greenlight/internal/data (interfaces: MovieQuerier)
//
// Generated by this command:
//
//	mockgen -package mockdb -destination internal/mocks/movie_mocks.go --build_flags=--mod=mod github.com/djudju12/greenlight/internal/data MovieQuerier
//
// Package mockdb is a generated GoMock package.
package mockdb

import (
	reflect "reflect"

	data "github.com/djudju12/greenlight/internal/data"
	gomock "go.uber.org/mock/gomock"
)

// MockMovieQuerier is a mock of MovieQuerier interface.
type MockMovieQuerier struct {
	ctrl     *gomock.Controller
	recorder *MockMovieQuerierMockRecorder
}

// MockMovieQuerierMockRecorder is the mock recorder for MockMovieQuerier.
type MockMovieQuerierMockRecorder struct {
	mock *MockMovieQuerier
}

// NewMockMovieQuerier creates a new mock instance.
func NewMockMovieQuerier(ctrl *gomock.Controller) *MockMovieQuerier {
	mock := &MockMovieQuerier{ctrl: ctrl}
	mock.recorder = &MockMovieQuerierMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockMovieQuerier) EXPECT() *MockMovieQuerierMockRecorder {
	return m.recorder
}

// Delete mocks base method.
func (m *MockMovieQuerier) Delete(arg0 int64) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Delete indicates an expected call of Delete.
func (mr *MockMovieQuerierMockRecorder) Delete(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockMovieQuerier)(nil).Delete), arg0)
}

// Get mocks base method.
func (m *MockMovieQuerier) Get(arg0 int64) (*data.Movie, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", arg0)
	ret0, _ := ret[0].(*data.Movie)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get.
func (mr *MockMovieQuerierMockRecorder) Get(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockMovieQuerier)(nil).Get), arg0)
}

// GetAll mocks base method.
func (m *MockMovieQuerier) GetAll(arg0 string, arg1 []string, arg2 data.Filters) ([]*data.Movie, data.Metadata, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAll", arg0, arg1, arg2)
	ret0, _ := ret[0].([]*data.Movie)
	ret1, _ := ret[1].(data.Metadata)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// GetAll indicates an expected call of GetAll.
func (mr *MockMovieQuerierMockRecorder) GetAll(arg0, arg1, arg2 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAll", reflect.TypeOf((*MockMovieQuerier)(nil).GetAll), arg0, arg1, arg2)
}

// Insert mocks base method.
func (m *MockMovieQuerier) Insert(arg0 *data.Movie) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Insert", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Insert indicates an expected call of Insert.
func (mr *MockMovieQuerierMockRecorder) Insert(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Insert", reflect.TypeOf((*MockMovieQuerier)(nil).Insert), arg0)
}

// Update mocks base method.
func (m *MockMovieQuerier) Update(arg0 *data.Movie) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Update", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Update indicates an expected call of Update.
func (mr *MockMovieQuerierMockRecorder) Update(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Update", reflect.TypeOf((*MockMovieQuerier)(nil).Update), arg0)
}