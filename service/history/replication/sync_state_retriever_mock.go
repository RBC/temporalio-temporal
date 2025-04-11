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
// Source: sync_state_retriever.go
//
// Generated by this command:
//
//	mockgen -copyright_file ../../../LICENSE -package replication -source sync_state_retriever.go -destination sync_state_retriever_mock.go
//

// Package replication is a generated GoMock package.
package replication

import (
	context "context"
	reflect "reflect"

	common "go.temporal.io/api/common/v1"
	history "go.temporal.io/server/api/history/v1"
	persistence "go.temporal.io/server/api/persistence/v1"
	interfaces "go.temporal.io/server/service/history/interfaces"
	gomock "go.uber.org/mock/gomock"
)

// MockSyncStateRetriever is a mock of SyncStateRetriever interface.
type MockSyncStateRetriever struct {
	ctrl     *gomock.Controller
	recorder *MockSyncStateRetrieverMockRecorder
	isgomock struct{}
}

// MockSyncStateRetrieverMockRecorder is the mock recorder for MockSyncStateRetriever.
type MockSyncStateRetrieverMockRecorder struct {
	mock *MockSyncStateRetriever
}

// NewMockSyncStateRetriever creates a new mock instance.
func NewMockSyncStateRetriever(ctrl *gomock.Controller) *MockSyncStateRetriever {
	mock := &MockSyncStateRetriever{ctrl: ctrl}
	mock.recorder = &MockSyncStateRetrieverMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockSyncStateRetriever) EXPECT() *MockSyncStateRetrieverMockRecorder {
	return m.recorder
}

// GetSyncWorkflowStateArtifact mocks base method.
func (m *MockSyncStateRetriever) GetSyncWorkflowStateArtifact(ctx context.Context, namespaceID string, execution *common.WorkflowExecution, targetVersionedTransition *persistence.VersionedTransition, targetVersionHistories *history.VersionHistories) (*SyncStateResult, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetSyncWorkflowStateArtifact", ctx, namespaceID, execution, targetVersionedTransition, targetVersionHistories)
	ret0, _ := ret[0].(*SyncStateResult)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetSyncWorkflowStateArtifact indicates an expected call of GetSyncWorkflowStateArtifact.
func (mr *MockSyncStateRetrieverMockRecorder) GetSyncWorkflowStateArtifact(ctx, namespaceID, execution, targetVersionedTransition, targetVersionHistories any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetSyncWorkflowStateArtifact", reflect.TypeOf((*MockSyncStateRetriever)(nil).GetSyncWorkflowStateArtifact), ctx, namespaceID, execution, targetVersionedTransition, targetVersionHistories)
}

// GetSyncWorkflowStateArtifactFromMutableState mocks base method.
func (m *MockSyncStateRetriever) GetSyncWorkflowStateArtifactFromMutableState(ctx context.Context, namespaceID string, execution *common.WorkflowExecution, mutableState interfaces.MutableState, targetVersionedTransition *persistence.VersionedTransition, targetVersionHistories [][]*history.VersionHistoryItem, releaseFunc interfaces.ReleaseWorkflowContextFunc) (*SyncStateResult, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetSyncWorkflowStateArtifactFromMutableState", ctx, namespaceID, execution, mutableState, targetVersionedTransition, targetVersionHistories, releaseFunc)
	ret0, _ := ret[0].(*SyncStateResult)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetSyncWorkflowStateArtifactFromMutableState indicates an expected call of GetSyncWorkflowStateArtifactFromMutableState.
func (mr *MockSyncStateRetrieverMockRecorder) GetSyncWorkflowStateArtifactFromMutableState(ctx, namespaceID, execution, mutableState, targetVersionedTransition, targetVersionHistories, releaseFunc any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetSyncWorkflowStateArtifactFromMutableState", reflect.TypeOf((*MockSyncStateRetriever)(nil).GetSyncWorkflowStateArtifactFromMutableState), ctx, namespaceID, execution, mutableState, targetVersionedTransition, targetVersionHistories, releaseFunc)
}

// GetSyncWorkflowStateArtifactFromMutableStateForNewWorkflow mocks base method.
func (m *MockSyncStateRetriever) GetSyncWorkflowStateArtifactFromMutableStateForNewWorkflow(ctx context.Context, namespaceID string, execution *common.WorkflowExecution, mutableState interfaces.MutableState, releaseFunc interfaces.ReleaseWorkflowContextFunc, taskVersionedTransition *persistence.VersionedTransition) (*SyncStateResult, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetSyncWorkflowStateArtifactFromMutableStateForNewWorkflow", ctx, namespaceID, execution, mutableState, releaseFunc, taskVersionedTransition)
	ret0, _ := ret[0].(*SyncStateResult)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetSyncWorkflowStateArtifactFromMutableStateForNewWorkflow indicates an expected call of GetSyncWorkflowStateArtifactFromMutableStateForNewWorkflow.
func (mr *MockSyncStateRetrieverMockRecorder) GetSyncWorkflowStateArtifactFromMutableStateForNewWorkflow(ctx, namespaceID, execution, mutableState, releaseFunc, taskVersionedTransition any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetSyncWorkflowStateArtifactFromMutableStateForNewWorkflow", reflect.TypeOf((*MockSyncStateRetriever)(nil).GetSyncWorkflowStateArtifactFromMutableStateForNewWorkflow), ctx, namespaceID, execution, mutableState, releaseFunc, taskVersionedTransition)
}

// MocklastUpdatedStateTransitionGetter is a mock of lastUpdatedStateTransitionGetter interface.
type MocklastUpdatedStateTransitionGetter struct {
	ctrl     *gomock.Controller
	recorder *MocklastUpdatedStateTransitionGetterMockRecorder
	isgomock struct{}
}

// MocklastUpdatedStateTransitionGetterMockRecorder is the mock recorder for MocklastUpdatedStateTransitionGetter.
type MocklastUpdatedStateTransitionGetterMockRecorder struct {
	mock *MocklastUpdatedStateTransitionGetter
}

// NewMocklastUpdatedStateTransitionGetter creates a new mock instance.
func NewMocklastUpdatedStateTransitionGetter(ctrl *gomock.Controller) *MocklastUpdatedStateTransitionGetter {
	mock := &MocklastUpdatedStateTransitionGetter{ctrl: ctrl}
	mock.recorder = &MocklastUpdatedStateTransitionGetterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MocklastUpdatedStateTransitionGetter) EXPECT() *MocklastUpdatedStateTransitionGetterMockRecorder {
	return m.recorder
}

// GetLastUpdateVersionedTransition mocks base method.
func (m *MocklastUpdatedStateTransitionGetter) GetLastUpdateVersionedTransition() *persistence.VersionedTransition {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetLastUpdateVersionedTransition")
	ret0, _ := ret[0].(*persistence.VersionedTransition)
	return ret0
}

// GetLastUpdateVersionedTransition indicates an expected call of GetLastUpdateVersionedTransition.
func (mr *MocklastUpdatedStateTransitionGetterMockRecorder) GetLastUpdateVersionedTransition() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetLastUpdateVersionedTransition", reflect.TypeOf((*MocklastUpdatedStateTransitionGetter)(nil).GetLastUpdateVersionedTransition))
}
