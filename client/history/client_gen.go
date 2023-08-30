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

// Code generated by cmd/tools/rpcwrappers. DO NOT EDIT.

package history

import (
	"context"

	"go.temporal.io/api/serviceerror"
	"go.temporal.io/server/api/historyservice/v1"
	"google.golang.org/grpc"
)

func (c *clientImpl) CloseShard(
	ctx context.Context,
	request *historyservice.CloseShardRequest,
	opts ...grpc.CallOption,
) (*historyservice.CloseShardResponse, error) {
	shardID := request.GetShardId()
	var response *historyservice.CloseShardResponse
	op := func(ctx context.Context, client historyservice.HistoryServiceClient) error {
		var err error
		ctx, cancel := c.createContext(ctx)
		defer cancel()
		response, err = client.CloseShard(ctx, request, opts...)
		return err
	}
	if err := c.executeWithRedirect(ctx, shardID, op); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *clientImpl) DeleteWorkflowExecution(
	ctx context.Context,
	request *historyservice.DeleteWorkflowExecutionRequest,
	opts ...grpc.CallOption,
) (*historyservice.DeleteWorkflowExecutionResponse, error) {
	shardID := c.shardIDFromWorkflowID(request.NamespaceId, request.GetWorkflowExecution().GetWorkflowId())
	var response *historyservice.DeleteWorkflowExecutionResponse
	op := func(ctx context.Context, client historyservice.HistoryServiceClient) error {
		var err error
		ctx, cancel := c.createContext(ctx)
		defer cancel()
		response, err = client.DeleteWorkflowExecution(ctx, request, opts...)
		return err
	}
	if err := c.executeWithRedirect(ctx, shardID, op); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *clientImpl) DeleteWorkflowVisibilityRecord(
	ctx context.Context,
	request *historyservice.DeleteWorkflowVisibilityRecordRequest,
	opts ...grpc.CallOption,
) (*historyservice.DeleteWorkflowVisibilityRecordResponse, error) {
	shardID := c.shardIDFromWorkflowID(request.NamespaceId, request.GetExecution().GetWorkflowId())
	var response *historyservice.DeleteWorkflowVisibilityRecordResponse
	op := func(ctx context.Context, client historyservice.HistoryServiceClient) error {
		var err error
		ctx, cancel := c.createContext(ctx)
		defer cancel()
		response, err = client.DeleteWorkflowVisibilityRecord(ctx, request, opts...)
		return err
	}
	if err := c.executeWithRedirect(ctx, shardID, op); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *clientImpl) DescribeMutableState(
	ctx context.Context,
	request *historyservice.DescribeMutableStateRequest,
	opts ...grpc.CallOption,
) (*historyservice.DescribeMutableStateResponse, error) {
	shardID := c.shardIDFromWorkflowID(request.NamespaceId, request.GetExecution().GetWorkflowId())
	var response *historyservice.DescribeMutableStateResponse
	op := func(ctx context.Context, client historyservice.HistoryServiceClient) error {
		var err error
		ctx, cancel := c.createContext(ctx)
		defer cancel()
		response, err = client.DescribeMutableState(ctx, request, opts...)
		return err
	}
	if err := c.executeWithRedirect(ctx, shardID, op); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *clientImpl) DescribeWorkflowExecution(
	ctx context.Context,
	request *historyservice.DescribeWorkflowExecutionRequest,
	opts ...grpc.CallOption,
) (*historyservice.DescribeWorkflowExecutionResponse, error) {
	shardID := c.shardIDFromWorkflowID(request.NamespaceId, request.GetRequest().GetExecution().GetWorkflowId())
	var response *historyservice.DescribeWorkflowExecutionResponse
	op := func(ctx context.Context, client historyservice.HistoryServiceClient) error {
		var err error
		ctx, cancel := c.createContext(ctx)
		defer cancel()
		response, err = client.DescribeWorkflowExecution(ctx, request, opts...)
		return err
	}
	if err := c.executeWithRedirect(ctx, shardID, op); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *clientImpl) ForceDeleteWorkflowExecution(
	ctx context.Context,
	request *historyservice.ForceDeleteWorkflowExecutionRequest,
	opts ...grpc.CallOption,
) (*historyservice.ForceDeleteWorkflowExecutionResponse, error) {
	shardID := c.shardIDFromWorkflowID(request.NamespaceId, request.GetRequest().GetExecution().GetWorkflowId())
	var response *historyservice.ForceDeleteWorkflowExecutionResponse
	op := func(ctx context.Context, client historyservice.HistoryServiceClient) error {
		var err error
		ctx, cancel := c.createContext(ctx)
		defer cancel()
		response, err = client.ForceDeleteWorkflowExecution(ctx, request, opts...)
		return err
	}
	if err := c.executeWithRedirect(ctx, shardID, op); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *clientImpl) GenerateLastHistoryReplicationTasks(
	ctx context.Context,
	request *historyservice.GenerateLastHistoryReplicationTasksRequest,
	opts ...grpc.CallOption,
) (*historyservice.GenerateLastHistoryReplicationTasksResponse, error) {
	shardID := c.shardIDFromWorkflowID(request.NamespaceId, request.GetExecution().GetWorkflowId())
	var response *historyservice.GenerateLastHistoryReplicationTasksResponse
	op := func(ctx context.Context, client historyservice.HistoryServiceClient) error {
		var err error
		ctx, cancel := c.createContext(ctx)
		defer cancel()
		response, err = client.GenerateLastHistoryReplicationTasks(ctx, request, opts...)
		return err
	}
	if err := c.executeWithRedirect(ctx, shardID, op); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *clientImpl) GetDLQMessages(
	ctx context.Context,
	request *historyservice.GetDLQMessagesRequest,
	opts ...grpc.CallOption,
) (*historyservice.GetDLQMessagesResponse, error) {
	shardID := request.GetShardId()
	var response *historyservice.GetDLQMessagesResponse
	op := func(ctx context.Context, client historyservice.HistoryServiceClient) error {
		var err error
		ctx, cancel := c.createContext(ctx)
		defer cancel()
		response, err = client.GetDLQMessages(ctx, request, opts...)
		return err
	}
	if err := c.executeWithRedirect(ctx, shardID, op); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *clientImpl) GetDLQReplicationMessages(
	ctx context.Context,
	request *historyservice.GetDLQReplicationMessagesRequest,
	opts ...grpc.CallOption,
) (*historyservice.GetDLQReplicationMessagesResponse, error) {
	// All workflow IDs are in the same shard per request
	if len(request.GetTaskInfos()) == 0 {
		return nil, serviceerror.NewInvalidArgument("missing TaskInfos")
	}
	shardID := c.shardIDFromWorkflowID(request.GetTaskInfos()[0].NamespaceId, request.GetTaskInfos()[0].WorkflowId)
	var response *historyservice.GetDLQReplicationMessagesResponse
	op := func(ctx context.Context, client historyservice.HistoryServiceClient) error {
		var err error
		ctx, cancel := c.createContext(ctx)
		defer cancel()
		response, err = client.GetDLQReplicationMessages(ctx, request, opts...)
		return err
	}
	if err := c.executeWithRedirect(ctx, shardID, op); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *clientImpl) GetMutableState(
	ctx context.Context,
	request *historyservice.GetMutableStateRequest,
	opts ...grpc.CallOption,
) (*historyservice.GetMutableStateResponse, error) {
	shardID := c.shardIDFromWorkflowID(request.NamespaceId, request.GetExecution().GetWorkflowId())
	var response *historyservice.GetMutableStateResponse
	op := func(ctx context.Context, client historyservice.HistoryServiceClient) error {
		var err error
		ctx, cancel := c.createContext(ctx)
		defer cancel()
		response, err = client.GetMutableState(ctx, request, opts...)
		return err
	}
	if err := c.executeWithRedirect(ctx, shardID, op); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *clientImpl) GetShard(
	ctx context.Context,
	request *historyservice.GetShardRequest,
	opts ...grpc.CallOption,
) (*historyservice.GetShardResponse, error) {
	shardID := request.GetShardId()
	var response *historyservice.GetShardResponse
	op := func(ctx context.Context, client historyservice.HistoryServiceClient) error {
		var err error
		ctx, cancel := c.createContext(ctx)
		defer cancel()
		response, err = client.GetShard(ctx, request, opts...)
		return err
	}
	if err := c.executeWithRedirect(ctx, shardID, op); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *clientImpl) GetWorkflowExecutionHistory(
	ctx context.Context,
	request *historyservice.GetWorkflowExecutionHistoryRequest,
	opts ...grpc.CallOption,
) (*historyservice.GetWorkflowExecutionHistoryResponse, error) {
	shardID := c.shardIDFromWorkflowID(request.NamespaceId, request.GetRequest().GetExecution().GetWorkflowId())
	var response *historyservice.GetWorkflowExecutionHistoryResponse
	op := func(ctx context.Context, client historyservice.HistoryServiceClient) error {
		var err error
		ctx, cancel := c.createContext(ctx)
		defer cancel()
		response, err = client.GetWorkflowExecutionHistory(ctx, request, opts...)
		return err
	}
	if err := c.executeWithRedirect(ctx, shardID, op); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *clientImpl) GetWorkflowExecutionHistoryReverse(
	ctx context.Context,
	request *historyservice.GetWorkflowExecutionHistoryReverseRequest,
	opts ...grpc.CallOption,
) (*historyservice.GetWorkflowExecutionHistoryReverseResponse, error) {
	shardID := c.shardIDFromWorkflowID(request.NamespaceId, request.GetRequest().GetExecution().GetWorkflowId())
	var response *historyservice.GetWorkflowExecutionHistoryReverseResponse
	op := func(ctx context.Context, client historyservice.HistoryServiceClient) error {
		var err error
		ctx, cancel := c.createContext(ctx)
		defer cancel()
		response, err = client.GetWorkflowExecutionHistoryReverse(ctx, request, opts...)
		return err
	}
	if err := c.executeWithRedirect(ctx, shardID, op); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *clientImpl) GetWorkflowExecutionRawHistoryV2(
	ctx context.Context,
	request *historyservice.GetWorkflowExecutionRawHistoryV2Request,
	opts ...grpc.CallOption,
) (*historyservice.GetWorkflowExecutionRawHistoryV2Response, error) {
	shardID := c.shardIDFromWorkflowID(request.NamespaceId, request.GetRequest().GetExecution().GetWorkflowId())
	var response *historyservice.GetWorkflowExecutionRawHistoryV2Response
	op := func(ctx context.Context, client historyservice.HistoryServiceClient) error {
		var err error
		ctx, cancel := c.createContext(ctx)
		defer cancel()
		response, err = client.GetWorkflowExecutionRawHistoryV2(ctx, request, opts...)
		return err
	}
	if err := c.executeWithRedirect(ctx, shardID, op); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *clientImpl) IsActivityTaskValid(
	ctx context.Context,
	request *historyservice.IsActivityTaskValidRequest,
	opts ...grpc.CallOption,
) (*historyservice.IsActivityTaskValidResponse, error) {
	shardID := c.shardIDFromWorkflowID(request.NamespaceId, request.GetExecution().GetWorkflowId())
	var response *historyservice.IsActivityTaskValidResponse
	op := func(ctx context.Context, client historyservice.HistoryServiceClient) error {
		var err error
		ctx, cancel := c.createContext(ctx)
		defer cancel()
		response, err = client.IsActivityTaskValid(ctx, request, opts...)
		return err
	}
	if err := c.executeWithRedirect(ctx, shardID, op); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *clientImpl) IsWorkflowTaskValid(
	ctx context.Context,
	request *historyservice.IsWorkflowTaskValidRequest,
	opts ...grpc.CallOption,
) (*historyservice.IsWorkflowTaskValidResponse, error) {
	shardID := c.shardIDFromWorkflowID(request.NamespaceId, request.GetExecution().GetWorkflowId())
	var response *historyservice.IsWorkflowTaskValidResponse
	op := func(ctx context.Context, client historyservice.HistoryServiceClient) error {
		var err error
		ctx, cancel := c.createContext(ctx)
		defer cancel()
		response, err = client.IsWorkflowTaskValid(ctx, request, opts...)
		return err
	}
	if err := c.executeWithRedirect(ctx, shardID, op); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *clientImpl) MergeDLQMessages(
	ctx context.Context,
	request *historyservice.MergeDLQMessagesRequest,
	opts ...grpc.CallOption,
) (*historyservice.MergeDLQMessagesResponse, error) {
	shardID := request.GetShardId()
	var response *historyservice.MergeDLQMessagesResponse
	op := func(ctx context.Context, client historyservice.HistoryServiceClient) error {
		var err error
		ctx, cancel := c.createContext(ctx)
		defer cancel()
		response, err = client.MergeDLQMessages(ctx, request, opts...)
		return err
	}
	if err := c.executeWithRedirect(ctx, shardID, op); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *clientImpl) PollMutableState(
	ctx context.Context,
	request *historyservice.PollMutableStateRequest,
	opts ...grpc.CallOption,
) (*historyservice.PollMutableStateResponse, error) {
	shardID := c.shardIDFromWorkflowID(request.NamespaceId, request.GetExecution().GetWorkflowId())
	var response *historyservice.PollMutableStateResponse
	op := func(ctx context.Context, client historyservice.HistoryServiceClient) error {
		var err error
		ctx, cancel := c.createContext(ctx)
		defer cancel()
		response, err = client.PollMutableState(ctx, request, opts...)
		return err
	}
	if err := c.executeWithRedirect(ctx, shardID, op); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *clientImpl) PollWorkflowExecutionUpdate(
	ctx context.Context,
	request *historyservice.PollWorkflowExecutionUpdateRequest,
	opts ...grpc.CallOption,
) (*historyservice.PollWorkflowExecutionUpdateResponse, error) {
	shardID := c.shardIDFromWorkflowID(request.NamespaceId, request.GetRequest().GetUpdateRef().GetWorkflowExecution().GetWorkflowId())
	var response *historyservice.PollWorkflowExecutionUpdateResponse
	op := func(ctx context.Context, client historyservice.HistoryServiceClient) error {
		var err error
		ctx, cancel := c.createContext(ctx)
		defer cancel()
		response, err = client.PollWorkflowExecutionUpdate(ctx, request, opts...)
		return err
	}
	if err := c.executeWithRedirect(ctx, shardID, op); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *clientImpl) PurgeDLQMessages(
	ctx context.Context,
	request *historyservice.PurgeDLQMessagesRequest,
	opts ...grpc.CallOption,
) (*historyservice.PurgeDLQMessagesResponse, error) {
	shardID := request.GetShardId()
	var response *historyservice.PurgeDLQMessagesResponse
	op := func(ctx context.Context, client historyservice.HistoryServiceClient) error {
		var err error
		ctx, cancel := c.createContext(ctx)
		defer cancel()
		response, err = client.PurgeDLQMessages(ctx, request, opts...)
		return err
	}
	if err := c.executeWithRedirect(ctx, shardID, op); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *clientImpl) QueryWorkflow(
	ctx context.Context,
	request *historyservice.QueryWorkflowRequest,
	opts ...grpc.CallOption,
) (*historyservice.QueryWorkflowResponse, error) {
	shardID := c.shardIDFromWorkflowID(request.NamespaceId, request.GetRequest().GetExecution().GetWorkflowId())
	var response *historyservice.QueryWorkflowResponse
	op := func(ctx context.Context, client historyservice.HistoryServiceClient) error {
		var err error
		ctx, cancel := c.createContext(ctx)
		defer cancel()
		response, err = client.QueryWorkflow(ctx, request, opts...)
		return err
	}
	if err := c.executeWithRedirect(ctx, shardID, op); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *clientImpl) ReapplyEvents(
	ctx context.Context,
	request *historyservice.ReapplyEventsRequest,
	opts ...grpc.CallOption,
) (*historyservice.ReapplyEventsResponse, error) {
	shardID := c.shardIDFromWorkflowID(request.NamespaceId, request.GetRequest().GetWorkflowExecution().GetWorkflowId())
	var response *historyservice.ReapplyEventsResponse
	op := func(ctx context.Context, client historyservice.HistoryServiceClient) error {
		var err error
		ctx, cancel := c.createContext(ctx)
		defer cancel()
		response, err = client.ReapplyEvents(ctx, request, opts...)
		return err
	}
	if err := c.executeWithRedirect(ctx, shardID, op); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *clientImpl) RebuildMutableState(
	ctx context.Context,
	request *historyservice.RebuildMutableStateRequest,
	opts ...grpc.CallOption,
) (*historyservice.RebuildMutableStateResponse, error) {
	shardID := c.shardIDFromWorkflowID(request.NamespaceId, request.GetExecution().GetWorkflowId())
	var response *historyservice.RebuildMutableStateResponse
	op := func(ctx context.Context, client historyservice.HistoryServiceClient) error {
		var err error
		ctx, cancel := c.createContext(ctx)
		defer cancel()
		response, err = client.RebuildMutableState(ctx, request, opts...)
		return err
	}
	if err := c.executeWithRedirect(ctx, shardID, op); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *clientImpl) RecordActivityTaskHeartbeat(
	ctx context.Context,
	request *historyservice.RecordActivityTaskHeartbeatRequest,
	opts ...grpc.CallOption,
) (*historyservice.RecordActivityTaskHeartbeatResponse, error) {
	taskToken, err := c.tokenSerializer.Deserialize(request.GetHeartbeatRequest().GetTaskToken())
	if err != nil {
		return nil, err
	}
	shardID := c.shardIDFromWorkflowID(request.NamespaceId, taskToken.GetWorkflowId())

	var response *historyservice.RecordActivityTaskHeartbeatResponse
	op := func(ctx context.Context, client historyservice.HistoryServiceClient) error {
		var err error
		ctx, cancel := c.createContext(ctx)
		defer cancel()
		response, err = client.RecordActivityTaskHeartbeat(ctx, request, opts...)
		return err
	}
	if err := c.executeWithRedirect(ctx, shardID, op); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *clientImpl) RecordActivityTaskStarted(
	ctx context.Context,
	request *historyservice.RecordActivityTaskStartedRequest,
	opts ...grpc.CallOption,
) (*historyservice.RecordActivityTaskStartedResponse, error) {
	shardID := c.shardIDFromWorkflowID(request.NamespaceId, request.GetWorkflowExecution().GetWorkflowId())
	var response *historyservice.RecordActivityTaskStartedResponse
	op := func(ctx context.Context, client historyservice.HistoryServiceClient) error {
		var err error
		ctx, cancel := c.createContext(ctx)
		defer cancel()
		response, err = client.RecordActivityTaskStarted(ctx, request, opts...)
		return err
	}
	if err := c.executeWithRedirect(ctx, shardID, op); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *clientImpl) RecordChildExecutionCompleted(
	ctx context.Context,
	request *historyservice.RecordChildExecutionCompletedRequest,
	opts ...grpc.CallOption,
) (*historyservice.RecordChildExecutionCompletedResponse, error) {
	shardID := c.shardIDFromWorkflowID(request.NamespaceId, request.GetParentExecution().GetWorkflowId())
	var response *historyservice.RecordChildExecutionCompletedResponse
	op := func(ctx context.Context, client historyservice.HistoryServiceClient) error {
		var err error
		ctx, cancel := c.createContext(ctx)
		defer cancel()
		response, err = client.RecordChildExecutionCompleted(ctx, request, opts...)
		return err
	}
	if err := c.executeWithRedirect(ctx, shardID, op); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *clientImpl) RecordWorkflowTaskStarted(
	ctx context.Context,
	request *historyservice.RecordWorkflowTaskStartedRequest,
	opts ...grpc.CallOption,
) (*historyservice.RecordWorkflowTaskStartedResponse, error) {
	shardID := c.shardIDFromWorkflowID(request.NamespaceId, request.GetWorkflowExecution().GetWorkflowId())
	var response *historyservice.RecordWorkflowTaskStartedResponse
	op := func(ctx context.Context, client historyservice.HistoryServiceClient) error {
		var err error
		ctx, cancel := c.createContext(ctx)
		defer cancel()
		response, err = client.RecordWorkflowTaskStarted(ctx, request, opts...)
		return err
	}
	if err := c.executeWithRedirect(ctx, shardID, op); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *clientImpl) RefreshWorkflowTasks(
	ctx context.Context,
	request *historyservice.RefreshWorkflowTasksRequest,
	opts ...grpc.CallOption,
) (*historyservice.RefreshWorkflowTasksResponse, error) {
	shardID := c.shardIDFromWorkflowID(request.NamespaceId, request.GetRequest().GetExecution().GetWorkflowId())
	var response *historyservice.RefreshWorkflowTasksResponse
	op := func(ctx context.Context, client historyservice.HistoryServiceClient) error {
		var err error
		ctx, cancel := c.createContext(ctx)
		defer cancel()
		response, err = client.RefreshWorkflowTasks(ctx, request, opts...)
		return err
	}
	if err := c.executeWithRedirect(ctx, shardID, op); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *clientImpl) RemoveSignalMutableState(
	ctx context.Context,
	request *historyservice.RemoveSignalMutableStateRequest,
	opts ...grpc.CallOption,
) (*historyservice.RemoveSignalMutableStateResponse, error) {
	shardID := c.shardIDFromWorkflowID(request.NamespaceId, request.GetWorkflowExecution().GetWorkflowId())
	var response *historyservice.RemoveSignalMutableStateResponse
	op := func(ctx context.Context, client historyservice.HistoryServiceClient) error {
		var err error
		ctx, cancel := c.createContext(ctx)
		defer cancel()
		response, err = client.RemoveSignalMutableState(ctx, request, opts...)
		return err
	}
	if err := c.executeWithRedirect(ctx, shardID, op); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *clientImpl) RemoveTask(
	ctx context.Context,
	request *historyservice.RemoveTaskRequest,
	opts ...grpc.CallOption,
) (*historyservice.RemoveTaskResponse, error) {
	shardID := request.GetShardId()
	var response *historyservice.RemoveTaskResponse
	op := func(ctx context.Context, client historyservice.HistoryServiceClient) error {
		var err error
		ctx, cancel := c.createContext(ctx)
		defer cancel()
		response, err = client.RemoveTask(ctx, request, opts...)
		return err
	}
	if err := c.executeWithRedirect(ctx, shardID, op); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *clientImpl) ReplicateEventsV2(
	ctx context.Context,
	request *historyservice.ReplicateEventsV2Request,
	opts ...grpc.CallOption,
) (*historyservice.ReplicateEventsV2Response, error) {
	shardID := c.shardIDFromWorkflowID(request.NamespaceId, request.GetWorkflowExecution().GetWorkflowId())
	var response *historyservice.ReplicateEventsV2Response
	op := func(ctx context.Context, client historyservice.HistoryServiceClient) error {
		var err error
		ctx, cancel := c.createContext(ctx)
		defer cancel()
		response, err = client.ReplicateEventsV2(ctx, request, opts...)
		return err
	}
	if err := c.executeWithRedirect(ctx, shardID, op); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *clientImpl) ReplicateWorkflowState(
	ctx context.Context,
	request *historyservice.ReplicateWorkflowStateRequest,
	opts ...grpc.CallOption,
) (*historyservice.ReplicateWorkflowStateResponse, error) {
	shardID := c.shardIDFromWorkflowID(request.NamespaceId, request.GetWorkflowState().GetExecutionInfo().GetWorkflowId())
	var response *historyservice.ReplicateWorkflowStateResponse
	op := func(ctx context.Context, client historyservice.HistoryServiceClient) error {
		var err error
		ctx, cancel := c.createContext(ctx)
		defer cancel()
		response, err = client.ReplicateWorkflowState(ctx, request, opts...)
		return err
	}
	if err := c.executeWithRedirect(ctx, shardID, op); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *clientImpl) RequestCancelWorkflowExecution(
	ctx context.Context,
	request *historyservice.RequestCancelWorkflowExecutionRequest,
	opts ...grpc.CallOption,
) (*historyservice.RequestCancelWorkflowExecutionResponse, error) {
	shardID := c.shardIDFromWorkflowID(request.NamespaceId, request.GetCancelRequest().GetWorkflowExecution().GetWorkflowId())
	var response *historyservice.RequestCancelWorkflowExecutionResponse
	op := func(ctx context.Context, client historyservice.HistoryServiceClient) error {
		var err error
		ctx, cancel := c.createContext(ctx)
		defer cancel()
		response, err = client.RequestCancelWorkflowExecution(ctx, request, opts...)
		return err
	}
	if err := c.executeWithRedirect(ctx, shardID, op); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *clientImpl) ResetStickyTaskQueue(
	ctx context.Context,
	request *historyservice.ResetStickyTaskQueueRequest,
	opts ...grpc.CallOption,
) (*historyservice.ResetStickyTaskQueueResponse, error) {
	shardID := c.shardIDFromWorkflowID(request.NamespaceId, request.GetExecution().GetWorkflowId())
	var response *historyservice.ResetStickyTaskQueueResponse
	op := func(ctx context.Context, client historyservice.HistoryServiceClient) error {
		var err error
		ctx, cancel := c.createContext(ctx)
		defer cancel()
		response, err = client.ResetStickyTaskQueue(ctx, request, opts...)
		return err
	}
	if err := c.executeWithRedirect(ctx, shardID, op); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *clientImpl) ResetWorkflowExecution(
	ctx context.Context,
	request *historyservice.ResetWorkflowExecutionRequest,
	opts ...grpc.CallOption,
) (*historyservice.ResetWorkflowExecutionResponse, error) {
	shardID := c.shardIDFromWorkflowID(request.NamespaceId, request.GetResetRequest().GetWorkflowExecution().GetWorkflowId())
	var response *historyservice.ResetWorkflowExecutionResponse
	op := func(ctx context.Context, client historyservice.HistoryServiceClient) error {
		var err error
		ctx, cancel := c.createContext(ctx)
		defer cancel()
		response, err = client.ResetWorkflowExecution(ctx, request, opts...)
		return err
	}
	if err := c.executeWithRedirect(ctx, shardID, op); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *clientImpl) RespondActivityTaskCanceled(
	ctx context.Context,
	request *historyservice.RespondActivityTaskCanceledRequest,
	opts ...grpc.CallOption,
) (*historyservice.RespondActivityTaskCanceledResponse, error) {
	taskToken, err := c.tokenSerializer.Deserialize(request.GetCancelRequest().GetTaskToken())
	if err != nil {
		return nil, err
	}
	shardID := c.shardIDFromWorkflowID(request.NamespaceId, taskToken.GetWorkflowId())

	var response *historyservice.RespondActivityTaskCanceledResponse
	op := func(ctx context.Context, client historyservice.HistoryServiceClient) error {
		var err error
		ctx, cancel := c.createContext(ctx)
		defer cancel()
		response, err = client.RespondActivityTaskCanceled(ctx, request, opts...)
		return err
	}
	if err := c.executeWithRedirect(ctx, shardID, op); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *clientImpl) RespondActivityTaskCompleted(
	ctx context.Context,
	request *historyservice.RespondActivityTaskCompletedRequest,
	opts ...grpc.CallOption,
) (*historyservice.RespondActivityTaskCompletedResponse, error) {
	taskToken, err := c.tokenSerializer.Deserialize(request.GetCompleteRequest().GetTaskToken())
	if err != nil {
		return nil, err
	}
	shardID := c.shardIDFromWorkflowID(request.NamespaceId, taskToken.GetWorkflowId())

	var response *historyservice.RespondActivityTaskCompletedResponse
	op := func(ctx context.Context, client historyservice.HistoryServiceClient) error {
		var err error
		ctx, cancel := c.createContext(ctx)
		defer cancel()
		response, err = client.RespondActivityTaskCompleted(ctx, request, opts...)
		return err
	}
	if err := c.executeWithRedirect(ctx, shardID, op); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *clientImpl) RespondActivityTaskFailed(
	ctx context.Context,
	request *historyservice.RespondActivityTaskFailedRequest,
	opts ...grpc.CallOption,
) (*historyservice.RespondActivityTaskFailedResponse, error) {
	taskToken, err := c.tokenSerializer.Deserialize(request.GetFailedRequest().GetTaskToken())
	if err != nil {
		return nil, err
	}
	shardID := c.shardIDFromWorkflowID(request.NamespaceId, taskToken.GetWorkflowId())

	var response *historyservice.RespondActivityTaskFailedResponse
	op := func(ctx context.Context, client historyservice.HistoryServiceClient) error {
		var err error
		ctx, cancel := c.createContext(ctx)
		defer cancel()
		response, err = client.RespondActivityTaskFailed(ctx, request, opts...)
		return err
	}
	if err := c.executeWithRedirect(ctx, shardID, op); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *clientImpl) RespondWorkflowTaskCompleted(
	ctx context.Context,
	request *historyservice.RespondWorkflowTaskCompletedRequest,
	opts ...grpc.CallOption,
) (*historyservice.RespondWorkflowTaskCompletedResponse, error) {
	taskToken, err := c.tokenSerializer.Deserialize(request.GetCompleteRequest().GetTaskToken())
	if err != nil {
		return nil, err
	}
	shardID := c.shardIDFromWorkflowID(request.NamespaceId, taskToken.GetWorkflowId())

	var response *historyservice.RespondWorkflowTaskCompletedResponse
	op := func(ctx context.Context, client historyservice.HistoryServiceClient) error {
		var err error
		ctx, cancel := c.createContext(ctx)
		defer cancel()
		response, err = client.RespondWorkflowTaskCompleted(ctx, request, opts...)
		return err
	}
	if err := c.executeWithRedirect(ctx, shardID, op); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *clientImpl) RespondWorkflowTaskFailed(
	ctx context.Context,
	request *historyservice.RespondWorkflowTaskFailedRequest,
	opts ...grpc.CallOption,
) (*historyservice.RespondWorkflowTaskFailedResponse, error) {
	taskToken, err := c.tokenSerializer.Deserialize(request.GetFailedRequest().GetTaskToken())
	if err != nil {
		return nil, err
	}
	shardID := c.shardIDFromWorkflowID(request.NamespaceId, taskToken.GetWorkflowId())

	var response *historyservice.RespondWorkflowTaskFailedResponse
	op := func(ctx context.Context, client historyservice.HistoryServiceClient) error {
		var err error
		ctx, cancel := c.createContext(ctx)
		defer cancel()
		response, err = client.RespondWorkflowTaskFailed(ctx, request, opts...)
		return err
	}
	if err := c.executeWithRedirect(ctx, shardID, op); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *clientImpl) ScheduleWorkflowTask(
	ctx context.Context,
	request *historyservice.ScheduleWorkflowTaskRequest,
	opts ...grpc.CallOption,
) (*historyservice.ScheduleWorkflowTaskResponse, error) {
	shardID := c.shardIDFromWorkflowID(request.NamespaceId, request.GetWorkflowExecution().GetWorkflowId())
	var response *historyservice.ScheduleWorkflowTaskResponse
	op := func(ctx context.Context, client historyservice.HistoryServiceClient) error {
		var err error
		ctx, cancel := c.createContext(ctx)
		defer cancel()
		response, err = client.ScheduleWorkflowTask(ctx, request, opts...)
		return err
	}
	if err := c.executeWithRedirect(ctx, shardID, op); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *clientImpl) SignalWithStartWorkflowExecution(
	ctx context.Context,
	request *historyservice.SignalWithStartWorkflowExecutionRequest,
	opts ...grpc.CallOption,
) (*historyservice.SignalWithStartWorkflowExecutionResponse, error) {
	shardID := c.shardIDFromWorkflowID(request.NamespaceId, request.GetSignalWithStartRequest().GetWorkflowId())
	var response *historyservice.SignalWithStartWorkflowExecutionResponse
	op := func(ctx context.Context, client historyservice.HistoryServiceClient) error {
		var err error
		ctx, cancel := c.createContext(ctx)
		defer cancel()
		response, err = client.SignalWithStartWorkflowExecution(ctx, request, opts...)
		return err
	}
	if err := c.executeWithRedirect(ctx, shardID, op); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *clientImpl) SignalWorkflowExecution(
	ctx context.Context,
	request *historyservice.SignalWorkflowExecutionRequest,
	opts ...grpc.CallOption,
) (*historyservice.SignalWorkflowExecutionResponse, error) {
	shardID := c.shardIDFromWorkflowID(request.NamespaceId, request.GetSignalRequest().GetWorkflowExecution().GetWorkflowId())
	var response *historyservice.SignalWorkflowExecutionResponse
	op := func(ctx context.Context, client historyservice.HistoryServiceClient) error {
		var err error
		ctx, cancel := c.createContext(ctx)
		defer cancel()
		response, err = client.SignalWorkflowExecution(ctx, request, opts...)
		return err
	}
	if err := c.executeWithRedirect(ctx, shardID, op); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *clientImpl) StartWorkflowExecution(
	ctx context.Context,
	request *historyservice.StartWorkflowExecutionRequest,
	opts ...grpc.CallOption,
) (*historyservice.StartWorkflowExecutionResponse, error) {
	shardID := c.shardIDFromWorkflowID(request.NamespaceId, request.GetStartRequest().GetWorkflowId())
	var response *historyservice.StartWorkflowExecutionResponse
	op := func(ctx context.Context, client historyservice.HistoryServiceClient) error {
		var err error
		ctx, cancel := c.createContext(ctx)
		defer cancel()
		response, err = client.StartWorkflowExecution(ctx, request, opts...)
		return err
	}
	if err := c.executeWithRedirect(ctx, shardID, op); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *clientImpl) SyncActivity(
	ctx context.Context,
	request *historyservice.SyncActivityRequest,
	opts ...grpc.CallOption,
) (*historyservice.SyncActivityResponse, error) {
	shardID := c.shardIDFromWorkflowID(request.NamespaceId, request.GetWorkflowId())
	var response *historyservice.SyncActivityResponse
	op := func(ctx context.Context, client historyservice.HistoryServiceClient) error {
		var err error
		ctx, cancel := c.createContext(ctx)
		defer cancel()
		response, err = client.SyncActivity(ctx, request, opts...)
		return err
	}
	if err := c.executeWithRedirect(ctx, shardID, op); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *clientImpl) SyncShardStatus(
	ctx context.Context,
	request *historyservice.SyncShardStatusRequest,
	opts ...grpc.CallOption,
) (*historyservice.SyncShardStatusResponse, error) {
	shardID := request.GetShardId()
	var response *historyservice.SyncShardStatusResponse
	op := func(ctx context.Context, client historyservice.HistoryServiceClient) error {
		var err error
		ctx, cancel := c.createContext(ctx)
		defer cancel()
		response, err = client.SyncShardStatus(ctx, request, opts...)
		return err
	}
	if err := c.executeWithRedirect(ctx, shardID, op); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *clientImpl) TerminateWorkflowExecution(
	ctx context.Context,
	request *historyservice.TerminateWorkflowExecutionRequest,
	opts ...grpc.CallOption,
) (*historyservice.TerminateWorkflowExecutionResponse, error) {
	shardID := c.shardIDFromWorkflowID(request.NamespaceId, request.GetTerminateRequest().GetWorkflowExecution().GetWorkflowId())
	var response *historyservice.TerminateWorkflowExecutionResponse
	op := func(ctx context.Context, client historyservice.HistoryServiceClient) error {
		var err error
		ctx, cancel := c.createContext(ctx)
		defer cancel()
		response, err = client.TerminateWorkflowExecution(ctx, request, opts...)
		return err
	}
	if err := c.executeWithRedirect(ctx, shardID, op); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *clientImpl) UpdateWorkflowExecution(
	ctx context.Context,
	request *historyservice.UpdateWorkflowExecutionRequest,
	opts ...grpc.CallOption,
) (*historyservice.UpdateWorkflowExecutionResponse, error) {
	shardID := c.shardIDFromWorkflowID(request.NamespaceId, request.GetRequest().GetWorkflowExecution().GetWorkflowId())
	var response *historyservice.UpdateWorkflowExecutionResponse
	op := func(ctx context.Context, client historyservice.HistoryServiceClient) error {
		var err error
		ctx, cancel := c.createContext(ctx)
		defer cancel()
		response, err = client.UpdateWorkflowExecution(ctx, request, opts...)
		return err
	}
	if err := c.executeWithRedirect(ctx, shardID, op); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *clientImpl) VerifyChildExecutionCompletionRecorded(
	ctx context.Context,
	request *historyservice.VerifyChildExecutionCompletionRecordedRequest,
	opts ...grpc.CallOption,
) (*historyservice.VerifyChildExecutionCompletionRecordedResponse, error) {
	shardID := c.shardIDFromWorkflowID(request.NamespaceId, request.GetParentExecution().GetWorkflowId())
	var response *historyservice.VerifyChildExecutionCompletionRecordedResponse
	op := func(ctx context.Context, client historyservice.HistoryServiceClient) error {
		var err error
		ctx, cancel := c.createContext(ctx)
		defer cancel()
		response, err = client.VerifyChildExecutionCompletionRecorded(ctx, request, opts...)
		return err
	}
	if err := c.executeWithRedirect(ctx, shardID, op); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *clientImpl) VerifyFirstWorkflowTaskScheduled(
	ctx context.Context,
	request *historyservice.VerifyFirstWorkflowTaskScheduledRequest,
	opts ...grpc.CallOption,
) (*historyservice.VerifyFirstWorkflowTaskScheduledResponse, error) {
	shardID := c.shardIDFromWorkflowID(request.NamespaceId, request.GetWorkflowExecution().GetWorkflowId())
	var response *historyservice.VerifyFirstWorkflowTaskScheduledResponse
	op := func(ctx context.Context, client historyservice.HistoryServiceClient) error {
		var err error
		ctx, cancel := c.createContext(ctx)
		defer cancel()
		response, err = client.VerifyFirstWorkflowTaskScheduled(ctx, request, opts...)
		return err
	}
	if err := c.executeWithRedirect(ctx, shardID, op); err != nil {
		return nil, err
	}
	return response, nil
}
