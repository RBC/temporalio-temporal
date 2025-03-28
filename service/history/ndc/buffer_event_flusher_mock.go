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
// Source: buffer_event_flusher.go
//
// Generated by this command:
//
//	mockgen -copyright_file ../../../LICENSE -package ndc -source buffer_event_flusher.go -destination buffer_event_flusher_mock.go
//

// Package ndc is a generated GoMock package.
package ndc

import (
	context "context"
	reflect "reflect"

	interfaces "go.temporal.io/server/service/history/interfaces"
	gomock "go.uber.org/mock/gomock"
)

// MockBufferEventFlusher is a mock of BufferEventFlusher interface.
type MockBufferEventFlusher struct {
	ctrl     *gomock.Controller
	recorder *MockBufferEventFlusherMockRecorder
	isgomock struct{}
}

// MockBufferEventFlusherMockRecorder is the mock recorder for MockBufferEventFlusher.
type MockBufferEventFlusherMockRecorder struct {
	mock *MockBufferEventFlusher
}

// NewMockBufferEventFlusher creates a new mock instance.
func NewMockBufferEventFlusher(ctrl *gomock.Controller) *MockBufferEventFlusher {
	mock := &MockBufferEventFlusher{ctrl: ctrl}
	mock.recorder = &MockBufferEventFlusherMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockBufferEventFlusher) EXPECT() *MockBufferEventFlusherMockRecorder {
	return m.recorder
}

// flush mocks base method.
func (m *MockBufferEventFlusher) flush(ctx context.Context) (interfaces.WorkflowContext, interfaces.MutableState, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "flush", ctx)
	ret0, _ := ret[0].(interfaces.WorkflowContext)
	ret1, _ := ret[1].(interfaces.MutableState)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// flush indicates an expected call of flush.
func (mr *MockBufferEventFlusherMockRecorder) flush(ctx any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "flush", reflect.TypeOf((*MockBufferEventFlusher)(nil).flush), ctx)
}
