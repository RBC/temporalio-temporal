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

package manager

// -aux_files is required here due to Closeable interface being in another file.
//go:generate mockgen -copyright_file ../../../../LICENSE -package $GOPACKAGE -source $GOFILE -destination visibility_manager_mock.go -aux_files go.temporal.io/server/common/persistence=../../data_interfaces.go

import (
	"context"
	"time"

	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	workflowpb "go.temporal.io/api/workflow/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/server/common/namespace"
	"go.temporal.io/server/common/persistence"
)

type (
	// VisibilityManager is used to manage the visibility store
	VisibilityManager interface {
		persistence.Closeable
		GetReadStoreName(nsName namespace.Name) string
		GetStoreNames() []string
		HasStoreName(stName string) bool
		GetIndexName() string
		ValidateCustomSearchAttributes(searchAttributes map[string]any) (map[string]any, error)

		// Write APIs.
		RecordWorkflowExecutionStarted(ctx context.Context, request *RecordWorkflowExecutionStartedRequest) error
		RecordWorkflowExecutionClosed(ctx context.Context, request *RecordWorkflowExecutionClosedRequest) error
		UpsertWorkflowExecution(ctx context.Context, request *UpsertWorkflowExecutionRequest) error
		DeleteWorkflowExecution(ctx context.Context, request *VisibilityDeleteWorkflowExecutionRequest) error

		// Read APIs.
		ListWorkflowExecutions(ctx context.Context, request *ListWorkflowExecutionsRequestV2) (*ListWorkflowExecutionsResponse, error)
		ScanWorkflowExecutions(ctx context.Context, request *ListWorkflowExecutionsRequestV2) (*ListWorkflowExecutionsResponse, error)
		CountWorkflowExecutions(ctx context.Context, request *CountWorkflowExecutionsRequest) (*CountWorkflowExecutionsResponse, error)
		GetWorkflowExecution(ctx context.Context, request *GetWorkflowExecutionRequest) (*GetWorkflowExecutionResponse, error)
	}

	VisibilityRequestBase struct {
		NamespaceID      namespace.ID
		Namespace        namespace.Name // namespace.Name is not persisted.
		Execution        *commonpb.WorkflowExecution
		WorkflowTypeName string
		StartTime        time.Time
		Status           enumspb.WorkflowExecutionStatus
		ExecutionTime    time.Time
		TaskID           int64 // used as condition update version for ES and _version for SQL stores
		ShardID          int32 // not persisted
		Memo             *commonpb.Memo
		TaskQueue        string
		SearchAttributes *commonpb.SearchAttributes
		ParentExecution  *commonpb.WorkflowExecution
		RootExecution    *commonpb.WorkflowExecution
	}

	// RecordWorkflowExecutionStartedRequest is used to add a record of a newly started execution
	RecordWorkflowExecutionStartedRequest struct {
		*VisibilityRequestBase
	}

	// RecordWorkflowExecutionClosedRequest is used to add a record of a closed execution
	RecordWorkflowExecutionClosedRequest struct {
		*VisibilityRequestBase
		CloseTime            time.Time
		ExecutionDuration    time.Duration
		HistoryLength        int64
		HistorySizeBytes     int64
		StateTransitionCount int64
	}

	// UpsertWorkflowExecutionRequest is used to upsert workflow execution
	UpsertWorkflowExecutionRequest struct {
		*VisibilityRequestBase
	}

	// ListWorkflowExecutionsRequest is used to list executions in a namespace
	ListWorkflowExecutionsRequest struct {
		NamespaceID       namespace.ID
		Namespace         namespace.Name // namespace.Name is not persisted.
		NamespaceDivision string
		EarliestStartTime time.Time
		LatestStartTime   time.Time
		// Maximum number of workflow executions per page
		PageSize int
		// Token to continue reading next page of workflow executions.
		// Pass in empty slice for first page.
		NextPageToken []byte
	}

	// ListWorkflowExecutionsRequestV2 is used to list executions in a namespace
	ListWorkflowExecutionsRequestV2 struct {
		NamespaceID namespace.ID
		Namespace   namespace.Name // namespace.Name is not persisted.
		PageSize    int            // Maximum number of workflow executions per page
		// Token to continue reading next page of workflow executions.
		// Pass in empty slice for first page.
		NextPageToken []byte
		Query         string
	}

	// ListWorkflowExecutionsResponse is the response to ListWorkflowExecutionsRequest
	ListWorkflowExecutionsResponse struct {
		Executions []*workflowpb.WorkflowExecutionInfo
		// Token to read next page if there are more workflow executions beyond page size.
		// Use this to set NextPageToken on ListWorkflowExecutionsRequest to read the next page.
		NextPageToken []byte
	}

	// CountWorkflowExecutionsRequest is request from CountWorkflowExecutions
	CountWorkflowExecutionsRequest struct {
		NamespaceID namespace.ID
		Namespace   namespace.Name // namespace.Name is not persisted.
		Query       string
	}

	// CountWorkflowExecutionsResponse is response to CountWorkflowExecutions
	CountWorkflowExecutionsResponse struct {
		Count  int64 // sum of counts in Groups
		Groups []*workflowservice.CountWorkflowExecutionsResponse_AggregationGroup
	}

	// VisibilityDeleteWorkflowExecutionRequest contains the request params for DeleteWorkflowExecution call
	VisibilityDeleteWorkflowExecutionRequest struct {
		NamespaceID namespace.ID
		RunID       string
		WorkflowID  string
		TaskID      int64
		CloseTime   *time.Time
	}

	// GetWorkflowExecutionRequest is request from GetWorkflowExecution
	GetWorkflowExecutionRequest struct {
		NamespaceID namespace.ID
		Namespace   namespace.Name // namespace.Name is not persisted
		RunID       string
		WorkflowID  string
	}

	// GetWorkflowExecutionResponse is response to GetWorkflowExecution
	GetWorkflowExecutionResponse struct {
		Execution *workflowpb.WorkflowExecutionInfo
	}
)

func (r *ListWorkflowExecutionsRequest) OverrideToken(token []byte) {
	r.NextPageToken = token
}

func (r *ListWorkflowExecutionsRequest) GetToken() []byte {
	return r.NextPageToken
}
func (r *ListWorkflowExecutionsRequest) OverridePageSize(pageSize int) {
	r.PageSize = pageSize
}
func (r *ListWorkflowExecutionsRequest) GetPageSize() int {
	return r.PageSize
}
