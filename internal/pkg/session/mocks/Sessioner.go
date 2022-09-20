// Code generated by mockery v2.14.0. DO NOT EDIT.

package mocks

import (
	context "context"

	mock "github.com/stretchr/testify/mock"

	time "time"
)

// Sessioner is an autogenerated mock type for the Sessioner type
type Sessioner struct {
	mock.Mock
}

// Add provides a mock function with given fields: _a0, _a1, _a2, _a3
func (_m *Sessioner) Add(_a0 context.Context, _a1 string, _a2 string, _a3 time.Duration) error {
	ret := _m.Called(_a0, _a1, _a2, _a3)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string, time.Duration) error); ok {
		r0 = rf(_a0, _a1, _a2, _a3)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Close provides a mock function with given fields:
func (_m *Sessioner) Close() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// IsValid provides a mock function with given fields: _a0, _a1, _a2
func (_m *Sessioner) IsValid(_a0 context.Context, _a1 string, _a2 string) bool {
	ret := _m.Called(_a0, _a1, _a2)

	var r0 bool
	if rf, ok := ret.Get(0).(func(context.Context, string, string) bool); ok {
		r0 = rf(_a0, _a1, _a2)
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

type mockConstructorTestingTNewSessioner interface {
	mock.TestingT
	Cleanup(func())
}

// NewSessioner creates a new instance of Sessioner. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewSessioner(t mockConstructorTestingTNewSessioner) *Sessioner {
	mock := &Sessioner{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}