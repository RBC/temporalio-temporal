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
// Source: transaction.go
//
// Generated by this command:
//
//	mockgen -copyright_file ../../../LICENSE -package workflow -source transaction.go -destination transaction_mock.go
//

// Package workflow is a generated GoMock package.
package workflow

import (
	context "context"
	reflect "reflect"

	persistence "go.temporal.io/server/common/persistence"
	gomock "go.uber.org/mock/gomock"
)

// MockTransaction is a mock of Transaction interface.
type MockTransaction struct {
	ctrl     *gomock.Controller
	recorder *MockTransactionMockRecorder
	isgomock struct{}
}

// MockTransactionMockRecorder is the mock recorder for MockTransaction.
type MockTransactionMockRecorder struct {
	mock *MockTransaction
}

// NewMockTransaction creates a new mock instance.
func NewMockTransaction(ctrl *gomock.Controller) *MockTransaction {
	mock := &MockTransaction{ctrl: ctrl}
	mock.recorder = &MockTransactionMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockTransaction) EXPECT() *MockTransactionMockRecorder {
	return m.recorder
}

// ConflictResolveWorkflowExecution mocks base method.
func (m *MockTransaction) ConflictResolveWorkflowExecution(ctx context.Context, conflictResolveMode persistence.ConflictResolveWorkflowMode, resetWorkflowFailoverVersion int64, resetWorkflowSnapshot *persistence.WorkflowSnapshot, resetWorkflowEventsSeq []*persistence.WorkflowEvents, newWorkflowFailoverVersion *int64, newWorkflowSnapshot *persistence.WorkflowSnapshot, newWorkflowEventsSeq []*persistence.WorkflowEvents, currentWorkflowFailoverVersion *int64, currentWorkflowMutation *persistence.WorkflowMutation, currentWorkflowEventsSeq []*persistence.WorkflowEvents) (int64, int64, int64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ConflictResolveWorkflowExecution", ctx, conflictResolveMode, resetWorkflowFailoverVersion, resetWorkflowSnapshot, resetWorkflowEventsSeq, newWorkflowFailoverVersion, newWorkflowSnapshot, newWorkflowEventsSeq, currentWorkflowFailoverVersion, currentWorkflowMutation, currentWorkflowEventsSeq)
	ret0, _ := ret[0].(int64)
	ret1, _ := ret[1].(int64)
	ret2, _ := ret[2].(int64)
	ret3, _ := ret[3].(error)
	return ret0, ret1, ret2, ret3
}

// ConflictResolveWorkflowExecution indicates an expected call of ConflictResolveWorkflowExecution.
func (mr *MockTransactionMockRecorder) ConflictResolveWorkflowExecution(ctx, conflictResolveMode, resetWorkflowFailoverVersion, resetWorkflowSnapshot, resetWorkflowEventsSeq, newWorkflowFailoverVersion, newWorkflowSnapshot, newWorkflowEventsSeq, currentWorkflowFailoverVersion, currentWorkflowMutation, currentWorkflowEventsSeq any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ConflictResolveWorkflowExecution", reflect.TypeOf((*MockTransaction)(nil).ConflictResolveWorkflowExecution), ctx, conflictResolveMode, resetWorkflowFailoverVersion, resetWorkflowSnapshot, resetWorkflowEventsSeq, newWorkflowFailoverVersion, newWorkflowSnapshot, newWorkflowEventsSeq, currentWorkflowFailoverVersion, currentWorkflowMutation, currentWorkflowEventsSeq)
}

// CreateWorkflowExecution mocks base method.
func (m *MockTransaction) CreateWorkflowExecution(ctx context.Context, createMode persistence.CreateWorkflowMode, newWorkflowFailoverVersion int64, newWorkflowSnapshot *persistence.WorkflowSnapshot, newWorkflowEventsSeq []*persistence.WorkflowEvents) (int64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateWorkflowExecution", ctx, createMode, newWorkflowFailoverVersion, newWorkflowSnapshot, newWorkflowEventsSeq)
	ret0, _ := ret[0].(int64)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateWorkflowExecution indicates an expected call of CreateWorkflowExecution.
func (mr *MockTransactionMockRecorder) CreateWorkflowExecution(ctx, createMode, newWorkflowFailoverVersion, newWorkflowSnapshot, newWorkflowEventsSeq any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateWorkflowExecution", reflect.TypeOf((*MockTransaction)(nil).CreateWorkflowExecution), ctx, createMode, newWorkflowFailoverVersion, newWorkflowSnapshot, newWorkflowEventsSeq)
}

// SetWorkflowExecution mocks base method.
func (m *MockTransaction) SetWorkflowExecution(ctx context.Context, workflowSnapshot *persistence.WorkflowSnapshot) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetWorkflowExecution", ctx, workflowSnapshot)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetWorkflowExecution indicates an expected call of SetWorkflowExecution.
func (mr *MockTransactionMockRecorder) SetWorkflowExecution(ctx, workflowSnapshot any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetWorkflowExecution", reflect.TypeOf((*MockTransaction)(nil).SetWorkflowExecution), ctx, workflowSnapshot)
}

// UpdateWorkflowExecution mocks base method.
func (m *MockTransaction) UpdateWorkflowExecution(ctx context.Context, updateMode persistence.UpdateWorkflowMode, currentWorkflowFailoverVersion int64, currentWorkflowMutation *persistence.WorkflowMutation, currentWorkflowEventsSeq []*persistence.WorkflowEvents, newWorkflowFailoverVersion *int64, newWorkflowSnapshot *persistence.WorkflowSnapshot, newWorkflowEventsSeq []*persistence.WorkflowEvents) (int64, int64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateWorkflowExecution", ctx, updateMode, currentWorkflowFailoverVersion, currentWorkflowMutation, currentWorkflowEventsSeq, newWorkflowFailoverVersion, newWorkflowSnapshot, newWorkflowEventsSeq)
	ret0, _ := ret[0].(int64)
	ret1, _ := ret[1].(int64)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// UpdateWorkflowExecution indicates an expected call of UpdateWorkflowExecution.
func (mr *MockTransactionMockRecorder) UpdateWorkflowExecution(ctx, updateMode, currentWorkflowFailoverVersion, currentWorkflowMutation, currentWorkflowEventsSeq, newWorkflowFailoverVersion, newWorkflowSnapshot, newWorkflowEventsSeq any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateWorkflowExecution", reflect.TypeOf((*MockTransaction)(nil).UpdateWorkflowExecution), ctx, updateMode, currentWorkflowFailoverVersion, currentWorkflowMutation, currentWorkflowEventsSeq, newWorkflowFailoverVersion, newWorkflowSnapshot, newWorkflowEventsSeq)
}
