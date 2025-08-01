// Code generated by MockGen. DO NOT EDIT.
// Source: chasm_tree.go
//
// Generated by this command:
//
//	mockgen -package interfaces -source chasm_tree.go -destination chasm_tree_mock.go
//

// Package interfaces is a generated GoMock package.
package interfaces

import (
	context "context"
	reflect "reflect"
	time "time"

	persistence "go.temporal.io/server/api/persistence/v1"
	chasm "go.temporal.io/server/chasm"
	gomock "go.uber.org/mock/gomock"
)

// MockChasmTree is a mock of ChasmTree interface.
type MockChasmTree struct {
	ctrl     *gomock.Controller
	recorder *MockChasmTreeMockRecorder
	isgomock struct{}
}

// MockChasmTreeMockRecorder is the mock recorder for MockChasmTree.
type MockChasmTreeMockRecorder struct {
	mock *MockChasmTree
}

// NewMockChasmTree creates a new mock instance.
func NewMockChasmTree(ctrl *gomock.Controller) *MockChasmTree {
	mock := &MockChasmTree{ctrl: ctrl}
	mock.recorder = &MockChasmTreeMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockChasmTree) EXPECT() *MockChasmTreeMockRecorder {
	return m.recorder
}

// ApplyMutation mocks base method.
func (m *MockChasmTree) ApplyMutation(arg0 chasm.NodesMutation) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ApplyMutation", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// ApplyMutation indicates an expected call of ApplyMutation.
func (mr *MockChasmTreeMockRecorder) ApplyMutation(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ApplyMutation", reflect.TypeOf((*MockChasmTree)(nil).ApplyMutation), arg0)
}

// ApplySnapshot mocks base method.
func (m *MockChasmTree) ApplySnapshot(arg0 chasm.NodesSnapshot) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ApplySnapshot", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// ApplySnapshot indicates an expected call of ApplySnapshot.
func (mr *MockChasmTreeMockRecorder) ApplySnapshot(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ApplySnapshot", reflect.TypeOf((*MockChasmTree)(nil).ApplySnapshot), arg0)
}

// Archetype mocks base method.
func (m *MockChasmTree) Archetype() chasm.Archetype {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Archetype")
	ret0, _ := ret[0].(chasm.Archetype)
	return ret0
}

// Archetype indicates an expected call of Archetype.
func (mr *MockChasmTreeMockRecorder) Archetype() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Archetype", reflect.TypeOf((*MockChasmTree)(nil).Archetype))
}

// CloseTransaction mocks base method.
func (m *MockChasmTree) CloseTransaction() (chasm.NodesMutation, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CloseTransaction")
	ret0, _ := ret[0].(chasm.NodesMutation)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CloseTransaction indicates an expected call of CloseTransaction.
func (mr *MockChasmTreeMockRecorder) CloseTransaction() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CloseTransaction", reflect.TypeOf((*MockChasmTree)(nil).CloseTransaction))
}

// Component mocks base method.
func (m *MockChasmTree) Component(arg0 chasm.Context, arg1 chasm.ComponentRef) (chasm.Component, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Component", arg0, arg1)
	ret0, _ := ret[0].(chasm.Component)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Component indicates an expected call of Component.
func (mr *MockChasmTreeMockRecorder) Component(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Component", reflect.TypeOf((*MockChasmTree)(nil).Component), arg0, arg1)
}

// ComponentByPath mocks base method.
func (m *MockChasmTree) ComponentByPath(arg0 chasm.Context, arg1 string) (chasm.Component, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ComponentByPath", arg0, arg1)
	ret0, _ := ret[0].(chasm.Component)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ComponentByPath indicates an expected call of ComponentByPath.
func (mr *MockChasmTreeMockRecorder) ComponentByPath(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ComponentByPath", reflect.TypeOf((*MockChasmTree)(nil).ComponentByPath), arg0, arg1)
}

// EachPureTask mocks base method.
func (m *MockChasmTree) EachPureTask(deadline time.Time, callback func(chasm.NodePureTask, chasm.TaskAttributes, any) error) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "EachPureTask", deadline, callback)
	ret0, _ := ret[0].(error)
	return ret0
}

// EachPureTask indicates an expected call of EachPureTask.
func (mr *MockChasmTreeMockRecorder) EachPureTask(deadline, callback any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "EachPureTask", reflect.TypeOf((*MockChasmTree)(nil).EachPureTask), deadline, callback)
}

// ExecuteSideEffectTask mocks base method.
func (m *MockChasmTree) ExecuteSideEffectTask(ctx context.Context, registry *chasm.Registry, entityKey chasm.EntityKey, taskAttributes chasm.TaskAttributes, taskInfo *persistence.ChasmTaskInfo, validate func(chasm.NodeBackend, chasm.Context, chasm.Component) error) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ExecuteSideEffectTask", ctx, registry, entityKey, taskAttributes, taskInfo, validate)
	ret0, _ := ret[0].(error)
	return ret0
}

// ExecuteSideEffectTask indicates an expected call of ExecuteSideEffectTask.
func (mr *MockChasmTreeMockRecorder) ExecuteSideEffectTask(ctx, registry, entityKey, taskAttributes, taskInfo, validate any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ExecuteSideEffectTask", reflect.TypeOf((*MockChasmTree)(nil).ExecuteSideEffectTask), ctx, registry, entityKey, taskAttributes, taskInfo, validate)
}

// IsDirty mocks base method.
func (m *MockChasmTree) IsDirty() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsDirty")
	ret0, _ := ret[0].(bool)
	return ret0
}

// IsDirty indicates an expected call of IsDirty.
func (mr *MockChasmTreeMockRecorder) IsDirty() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsDirty", reflect.TypeOf((*MockChasmTree)(nil).IsDirty))
}

// IsStale mocks base method.
func (m *MockChasmTree) IsStale(arg0 chasm.ComponentRef) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsStale", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// IsStale indicates an expected call of IsStale.
func (mr *MockChasmTreeMockRecorder) IsStale(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsStale", reflect.TypeOf((*MockChasmTree)(nil).IsStale), arg0)
}

// IsStateDirty mocks base method.
func (m *MockChasmTree) IsStateDirty() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsStateDirty")
	ret0, _ := ret[0].(bool)
	return ret0
}

// IsStateDirty indicates an expected call of IsStateDirty.
func (mr *MockChasmTreeMockRecorder) IsStateDirty() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsStateDirty", reflect.TypeOf((*MockChasmTree)(nil).IsStateDirty))
}

// RefreshTasks mocks base method.
func (m *MockChasmTree) RefreshTasks() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RefreshTasks")
	ret0, _ := ret[0].(error)
	return ret0
}

// RefreshTasks indicates an expected call of RefreshTasks.
func (mr *MockChasmTreeMockRecorder) RefreshTasks() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RefreshTasks", reflect.TypeOf((*MockChasmTree)(nil).RefreshTasks))
}

// Snapshot mocks base method.
func (m *MockChasmTree) Snapshot(arg0 *persistence.VersionedTransition) chasm.NodesSnapshot {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Snapshot", arg0)
	ret0, _ := ret[0].(chasm.NodesSnapshot)
	return ret0
}

// Snapshot indicates an expected call of Snapshot.
func (mr *MockChasmTreeMockRecorder) Snapshot(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Snapshot", reflect.TypeOf((*MockChasmTree)(nil).Snapshot), arg0)
}

// Terminate mocks base method.
func (m *MockChasmTree) Terminate(arg0 chasm.TerminateComponentRequest) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Terminate", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Terminate indicates an expected call of Terminate.
func (mr *MockChasmTreeMockRecorder) Terminate(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Terminate", reflect.TypeOf((*MockChasmTree)(nil).Terminate), arg0)
}

// ValidateSideEffectTask mocks base method.
func (m *MockChasmTree) ValidateSideEffectTask(ctx context.Context, taskAttributes chasm.TaskAttributes, taskInfo *persistence.ChasmTaskInfo) (any, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ValidateSideEffectTask", ctx, taskAttributes, taskInfo)
	ret0, _ := ret[0].(any)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ValidateSideEffectTask indicates an expected call of ValidateSideEffectTask.
func (mr *MockChasmTreeMockRecorder) ValidateSideEffectTask(ctx, taskAttributes, taskInfo any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ValidateSideEffectTask", reflect.TypeOf((*MockChasmTree)(nil).ValidateSideEffectTask), ctx, taskAttributes, taskInfo)
}
