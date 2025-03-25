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
// Source: consistency_checker.go
//
// Generated by this command:
//
//	mockgen -copyright_file ../../../LICENSE -package api -source consistency_checker.go -destination consistency_checker_mock.go
//

// Package api is a generated GoMock package.
package api

import (
	context "context"
	reflect "reflect"

	clock "go.temporal.io/server/api/clock/v1"
	definition "go.temporal.io/server/common/definition"
	locks "go.temporal.io/server/common/locks"
	cache "go.temporal.io/server/service/history/workflow/cache"
	gomock "go.uber.org/mock/gomock"
)

// MockWorkflowConsistencyChecker is a mock of WorkflowConsistencyChecker interface.
type MockWorkflowConsistencyChecker struct {
	ctrl     *gomock.Controller
	recorder *MockWorkflowConsistencyCheckerMockRecorder
	isgomock struct{}
}

// MockWorkflowConsistencyCheckerMockRecorder is the mock recorder for MockWorkflowConsistencyChecker.
type MockWorkflowConsistencyCheckerMockRecorder struct {
	mock *MockWorkflowConsistencyChecker
}

// NewMockWorkflowConsistencyChecker creates a new mock instance.
func NewMockWorkflowConsistencyChecker(ctrl *gomock.Controller) *MockWorkflowConsistencyChecker {
	mock := &MockWorkflowConsistencyChecker{ctrl: ctrl}
	mock.recorder = &MockWorkflowConsistencyCheckerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockWorkflowConsistencyChecker) EXPECT() *MockWorkflowConsistencyCheckerMockRecorder {
	return m.recorder
}

// GetCurrentRunID mocks base method.
func (m *MockWorkflowConsistencyChecker) GetCurrentRunID(ctx context.Context, namespaceID, workflowID string, lockPriority locks.Priority) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCurrentRunID", ctx, namespaceID, workflowID, lockPriority)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetCurrentRunID indicates an expected call of GetCurrentRunID.
func (mr *MockWorkflowConsistencyCheckerMockRecorder) GetCurrentRunID(ctx, namespaceID, workflowID, lockPriority any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCurrentRunID", reflect.TypeOf((*MockWorkflowConsistencyChecker)(nil).GetCurrentRunID), ctx, namespaceID, workflowID, lockPriority)
}

// GetWorkflowCache mocks base method.
func (m *MockWorkflowConsistencyChecker) GetWorkflowCache() cache.Cache {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetWorkflowCache")
	ret0, _ := ret[0].(cache.Cache)
	return ret0
}

// GetWorkflowCache indicates an expected call of GetWorkflowCache.
func (mr *MockWorkflowConsistencyCheckerMockRecorder) GetWorkflowCache() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetWorkflowCache", reflect.TypeOf((*MockWorkflowConsistencyChecker)(nil).GetWorkflowCache))
}

// GetWorkflowLease mocks base method.
func (m *MockWorkflowConsistencyChecker) GetWorkflowLease(ctx context.Context, reqClock *clock.VectorClock, workflowKey definition.WorkflowKey, lockPriority locks.Priority) (WorkflowLease, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetWorkflowLease", ctx, reqClock, workflowKey, lockPriority)
	ret0, _ := ret[0].(WorkflowLease)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetWorkflowLease indicates an expected call of GetWorkflowLease.
func (mr *MockWorkflowConsistencyCheckerMockRecorder) GetWorkflowLease(ctx, reqClock, workflowKey, lockPriority any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetWorkflowLease", reflect.TypeOf((*MockWorkflowConsistencyChecker)(nil).GetWorkflowLease), ctx, reqClock, workflowKey, lockPriority)
}

// GetWorkflowLeaseWithConsistencyCheck mocks base method.
func (m *MockWorkflowConsistencyChecker) GetWorkflowLeaseWithConsistencyCheck(ctx context.Context, reqClock *clock.VectorClock, consistencyPredicate MutableStateConsistencyPredicate, workflowKey definition.WorkflowKey, lockPriority locks.Priority) (WorkflowLease, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetWorkflowLeaseWithConsistencyCheck", ctx, reqClock, consistencyPredicate, workflowKey, lockPriority)
	ret0, _ := ret[0].(WorkflowLease)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetWorkflowLeaseWithConsistencyCheck indicates an expected call of GetWorkflowLeaseWithConsistencyCheck.
func (mr *MockWorkflowConsistencyCheckerMockRecorder) GetWorkflowLeaseWithConsistencyCheck(ctx, reqClock, consistencyPredicate, workflowKey, lockPriority any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetWorkflowLeaseWithConsistencyCheck", reflect.TypeOf((*MockWorkflowConsistencyChecker)(nil).GetWorkflowLeaseWithConsistencyCheck), ctx, reqClock, consistencyPredicate, workflowKey, lockPriority)
}
