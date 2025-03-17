// Code generated by mockery v2.53.2. DO NOT EDIT.

package toolstesting

import (
	mock "github.com/stretchr/testify/mock"
	log "github.com/symflower/eval-dev-quality/log"
)

// MockTool is an autogenerated mock type for the Tool type
type MockTool struct {
	mock.Mock
}

// BinaryName provides a mock function with no fields
func (_m *MockTool) BinaryName() string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for BinaryName")
	}

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// BinaryPath provides a mock function with no fields
func (_m *MockTool) BinaryPath() string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for BinaryPath")
	}

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// CheckVersion provides a mock function with given fields: logger, binaryPath
func (_m *MockTool) CheckVersion(logger *log.Logger, binaryPath string) error {
	ret := _m.Called(logger, binaryPath)

	if len(ret) == 0 {
		panic("no return value specified for CheckVersion")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(*log.Logger, string) error); ok {
		r0 = rf(logger, binaryPath)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ID provides a mock function with no fields
func (_m *MockTool) ID() string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for ID")
	}

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// Install provides a mock function with given fields: logger, installPath
func (_m *MockTool) Install(logger *log.Logger, installPath string) error {
	ret := _m.Called(logger, installPath)

	if len(ret) == 0 {
		panic("no return value specified for Install")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(*log.Logger, string) error); ok {
		r0 = rf(logger, installPath)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// RequiredVersion provides a mock function with no fields
func (_m *MockTool) RequiredVersion() string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for RequiredVersion")
	}

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// NewMockTool creates a new instance of MockTool. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockTool(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockTool {
	mock := &MockTool{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
