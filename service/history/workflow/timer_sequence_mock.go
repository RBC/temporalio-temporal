// The MIT License
//
// Copyright (c) 2020 Temporal Technologies Inc.  All rights reserved.
//
// Copyright (c) 2020 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

// Code generated by MockGen. DO NOT EDIT.
// Source: timer_sequence.go
//
// Generated by this command:
//
//	mockgen -copyright_file ../../../LICENSE -package workflow -source timer_sequence.go -destination timer_sequence_mock.go
//

// Package workflow is a generated GoMock package.
package workflow

import (
	reflect "reflect"

	gomock "go.uber.org/mock/gomock"
)

// MockTimerSequence is a mock of TimerSequence interface.
type MockTimerSequence struct {
	ctrl     *gomock.Controller
	recorder *MockTimerSequenceMockRecorder
	isgomock struct{}
}

// MockTimerSequenceMockRecorder is the mock recorder for MockTimerSequence.
type MockTimerSequenceMockRecorder struct {
	mock *MockTimerSequence
}

// NewMockTimerSequence creates a new mock instance.
func NewMockTimerSequence(ctrl *gomock.Controller) *MockTimerSequence {
	mock := &MockTimerSequence{ctrl: ctrl}
	mock.recorder = &MockTimerSequenceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockTimerSequence) EXPECT() *MockTimerSequenceMockRecorder {
	return m.recorder
}

// CreateNextActivityTimer mocks base method.
func (m *MockTimerSequence) CreateNextActivityTimer() (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateNextActivityTimer")
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateNextActivityTimer indicates an expected call of CreateNextActivityTimer.
func (mr *MockTimerSequenceMockRecorder) CreateNextActivityTimer() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateNextActivityTimer", reflect.TypeOf((*MockTimerSequence)(nil).CreateNextActivityTimer))
}

// CreateNextUserTimer mocks base method.
func (m *MockTimerSequence) CreateNextUserTimer() (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateNextUserTimer")
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateNextUserTimer indicates an expected call of CreateNextUserTimer.
func (mr *MockTimerSequenceMockRecorder) CreateNextUserTimer() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateNextUserTimer", reflect.TypeOf((*MockTimerSequence)(nil).CreateNextUserTimer))
}

// LoadAndSortActivityTimers mocks base method.
func (m *MockTimerSequence) LoadAndSortActivityTimers() []TimerSequenceID {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "LoadAndSortActivityTimers")
	ret0, _ := ret[0].([]TimerSequenceID)
	return ret0
}

// LoadAndSortActivityTimers indicates an expected call of LoadAndSortActivityTimers.
func (mr *MockTimerSequenceMockRecorder) LoadAndSortActivityTimers() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "LoadAndSortActivityTimers", reflect.TypeOf((*MockTimerSequence)(nil).LoadAndSortActivityTimers))
}

// LoadAndSortUserTimers mocks base method.
func (m *MockTimerSequence) LoadAndSortUserTimers() []TimerSequenceID {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "LoadAndSortUserTimers")
	ret0, _ := ret[0].([]TimerSequenceID)
	return ret0
}

// LoadAndSortUserTimers indicates an expected call of LoadAndSortUserTimers.
func (mr *MockTimerSequenceMockRecorder) LoadAndSortUserTimers() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "LoadAndSortUserTimers", reflect.TypeOf((*MockTimerSequence)(nil).LoadAndSortUserTimers))
}
