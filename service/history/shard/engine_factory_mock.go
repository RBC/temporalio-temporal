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
// Source: engine_factory.go
//
// Generated by this command:
//
//	mockgen -copyright_file ../../../LICENSE -package shard -source engine_factory.go -destination engine_factory_mock.go
//

// Package shard is a generated GoMock package.
package shard

import (
	reflect "reflect"

	interfaces "go.temporal.io/server/service/history/interfaces"
	gomock "go.uber.org/mock/gomock"
)

// MockEngineFactory is a mock of EngineFactory interface.
type MockEngineFactory struct {
	ctrl     *gomock.Controller
	recorder *MockEngineFactoryMockRecorder
	isgomock struct{}
}

// MockEngineFactoryMockRecorder is the mock recorder for MockEngineFactory.
type MockEngineFactoryMockRecorder struct {
	mock *MockEngineFactory
}

// NewMockEngineFactory creates a new mock instance.
func NewMockEngineFactory(ctrl *gomock.Controller) *MockEngineFactory {
	mock := &MockEngineFactory{ctrl: ctrl}
	mock.recorder = &MockEngineFactoryMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockEngineFactory) EXPECT() *MockEngineFactoryMockRecorder {
	return m.recorder
}

// CreateEngine mocks base method.
func (m *MockEngineFactory) CreateEngine(context interfaces.ShardContext) interfaces.Engine {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateEngine", context)
	ret0, _ := ret[0].(interfaces.Engine)
	return ret0
}

// CreateEngine indicates an expected call of CreateEngine.
func (mr *MockEngineFactoryMockRecorder) CreateEngine(context any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateEngine", reflect.TypeOf((*MockEngineFactory)(nil).CreateEngine), context)
}
