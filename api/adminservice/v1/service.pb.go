// Code generated by protoc-gen-go. DO NOT EDIT.
// plugins:
// 	protoc-gen-go
// 	protoc
// source: temporal/server/api/adminservice/v1/service.proto

package adminservice

import (
	reflect "reflect"
	unsafe "unsafe"

	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

var File_temporal_server_api_adminservice_v1_service_proto protoreflect.FileDescriptor

const file_temporal_server_api_adminservice_v1_service_proto_rawDesc = "" +
	"\n" +
	"1temporal/server/api/adminservice/v1/service.proto\x12#temporal.server.api.adminservice.v1\x1a:temporal/server/api/adminservice/v1/request_response.proto2\xbf4\n" +
	"\fAdminService\x12\x9a\x01\n" +
	"\x13RebuildMutableState\x12?.temporal.server.api.adminservice.v1.RebuildMutableStateRequest\x1a@.temporal.server.api.adminservice.v1.RebuildMutableStateResponse\"\x00\x12\xa6\x01\n" +
	"\x17ImportWorkflowExecution\x12C.temporal.server.api.adminservice.v1.ImportWorkflowExecutionRequest\x1aD.temporal.server.api.adminservice.v1.ImportWorkflowExecutionResponse\"\x00\x12\x9d\x01\n" +
	"\x14DescribeMutableState\x12@.temporal.server.api.adminservice.v1.DescribeMutableStateRequest\x1aA.temporal.server.api.adminservice.v1.DescribeMutableStateResponse\"\x00\x12\x9a\x01\n" +
	"\x13DescribeHistoryHost\x12?.temporal.server.api.adminservice.v1.DescribeHistoryHostRequest\x1a@.temporal.server.api.adminservice.v1.DescribeHistoryHostResponse\"\x00\x12y\n" +
	"\bGetShard\x124.temporal.server.api.adminservice.v1.GetShardRequest\x1a5.temporal.server.api.adminservice.v1.GetShardResponse\"\x00\x12\x7f\n" +
	"\n" +
	"CloseShard\x126.temporal.server.api.adminservice.v1.CloseShardRequest\x1a7.temporal.server.api.adminservice.v1.CloseShardResponse\"\x00\x12\x91\x01\n" +
	"\x10ListHistoryTasks\x12<.temporal.server.api.adminservice.v1.ListHistoryTasksRequest\x1a=.temporal.server.api.adminservice.v1.ListHistoryTasksResponse\"\x00\x12\x7f\n" +
	"\n" +
	"RemoveTask\x126.temporal.server.api.adminservice.v1.RemoveTaskRequest\x1a7.temporal.server.api.adminservice.v1.RemoveTaskResponse\"\x00\x12\xc1\x01\n" +
	" GetWorkflowExecutionRawHistoryV2\x12L.temporal.server.api.adminservice.v1.GetWorkflowExecutionRawHistoryV2Request\x1aM.temporal.server.api.adminservice.v1.GetWorkflowExecutionRawHistoryV2Response\"\x00\x12\xbb\x01\n" +
	"\x1eGetWorkflowExecutionRawHistory\x12J.temporal.server.api.adminservice.v1.GetWorkflowExecutionRawHistoryRequest\x1aK.temporal.server.api.adminservice.v1.GetWorkflowExecutionRawHistoryResponse\"\x00\x12\xa3\x01\n" +
	"\x16GetReplicationMessages\x12B.temporal.server.api.adminservice.v1.GetReplicationMessagesRequest\x1aC.temporal.server.api.adminservice.v1.GetReplicationMessagesResponse\"\x00\x12\xbe\x01\n" +
	"\x1fGetNamespaceReplicationMessages\x12K.temporal.server.api.adminservice.v1.GetNamespaceReplicationMessagesRequest\x1aL.temporal.server.api.adminservice.v1.GetNamespaceReplicationMessagesResponse\"\x00\x12\xac\x01\n" +
	"\x19GetDLQReplicationMessages\x12E.temporal.server.api.adminservice.v1.GetDLQReplicationMessagesRequest\x1aF.temporal.server.api.adminservice.v1.GetDLQReplicationMessagesResponse\"\x00\x12\x88\x01\n" +
	"\rReapplyEvents\x129.temporal.server.api.adminservice.v1.ReapplyEventsRequest\x1a:.temporal.server.api.adminservice.v1.ReapplyEventsResponse\"\x00\x12\x9a\x01\n" +
	"\x13AddSearchAttributes\x12?.temporal.server.api.adminservice.v1.AddSearchAttributesRequest\x1a@.temporal.server.api.adminservice.v1.AddSearchAttributesResponse\"\x00\x12\xa3\x01\n" +
	"\x16RemoveSearchAttributes\x12B.temporal.server.api.adminservice.v1.RemoveSearchAttributesRequest\x1aC.temporal.server.api.adminservice.v1.RemoveSearchAttributesResponse\"\x00\x12\x9a\x01\n" +
	"\x13GetSearchAttributes\x12?.temporal.server.api.adminservice.v1.GetSearchAttributesRequest\x1a@.temporal.server.api.adminservice.v1.GetSearchAttributesResponse\"\x00\x12\x8e\x01\n" +
	"\x0fDescribeCluster\x12;.temporal.server.api.adminservice.v1.DescribeClusterRequest\x1a<.temporal.server.api.adminservice.v1.DescribeClusterResponse\"\x00\x12\x85\x01\n" +
	"\fListClusters\x128.temporal.server.api.adminservice.v1.ListClustersRequest\x1a9.temporal.server.api.adminservice.v1.ListClustersResponse\"\x00\x12\x97\x01\n" +
	"\x12ListClusterMembers\x12>.temporal.server.api.adminservice.v1.ListClusterMembersRequest\x1a?.temporal.server.api.adminservice.v1.ListClusterMembersResponse\"\x00\x12\xa9\x01\n" +
	"\x18AddOrUpdateRemoteCluster\x12D.temporal.server.api.adminservice.v1.AddOrUpdateRemoteClusterRequest\x1aE.temporal.server.api.adminservice.v1.AddOrUpdateRemoteClusterResponse\"\x00\x12\x9a\x01\n" +
	"\x13RemoveRemoteCluster\x12?.temporal.server.api.adminservice.v1.RemoveRemoteClusterRequest\x1a@.temporal.server.api.adminservice.v1.RemoveRemoteClusterResponse\"\x00\x12\x8b\x01\n" +
	"\x0eGetDLQMessages\x12:.temporal.server.api.adminservice.v1.GetDLQMessagesRequest\x1a;.temporal.server.api.adminservice.v1.GetDLQMessagesResponse\"\x00\x12\x91\x01\n" +
	"\x10PurgeDLQMessages\x12<.temporal.server.api.adminservice.v1.PurgeDLQMessagesRequest\x1a=.temporal.server.api.adminservice.v1.PurgeDLQMessagesResponse\"\x00\x12\x91\x01\n" +
	"\x10MergeDLQMessages\x12<.temporal.server.api.adminservice.v1.MergeDLQMessagesRequest\x1a=.temporal.server.api.adminservice.v1.MergeDLQMessagesResponse\"\x00\x12\x9d\x01\n" +
	"\x14RefreshWorkflowTasks\x12@.temporal.server.api.adminservice.v1.RefreshWorkflowTasksRequest\x1aA.temporal.server.api.adminservice.v1.RefreshWorkflowTasksResponse\"\x00\x12\xa3\x01\n" +
	"\x16ResendReplicationTasks\x12B.temporal.server.api.adminservice.v1.ResendReplicationTasksRequest\x1aC.temporal.server.api.adminservice.v1.ResendReplicationTasksResponse\"\x00\x12\x94\x01\n" +
	"\x11GetTaskQueueTasks\x12=.temporal.server.api.adminservice.v1.GetTaskQueueTasksRequest\x1a>.temporal.server.api.adminservice.v1.GetTaskQueueTasksResponse\"\x00\x12\xa6\x01\n" +
	"\x17DeleteWorkflowExecution\x12C.temporal.server.api.adminservice.v1.DeleteWorkflowExecutionRequest\x1aD.temporal.server.api.adminservice.v1.DeleteWorkflowExecutionResponse\"\x00\x12\xc8\x01\n" +
	"!StreamWorkflowReplicationMessages\x12M.temporal.server.api.adminservice.v1.StreamWorkflowReplicationMessagesRequest\x1aN.temporal.server.api.adminservice.v1.StreamWorkflowReplicationMessagesResponse\"\x00(\x010\x01\x12\x85\x01\n" +
	"\fGetNamespace\x128.temporal.server.api.adminservice.v1.GetNamespaceRequest\x1a9.temporal.server.api.adminservice.v1.GetNamespaceResponse\"\x00\x12\x82\x01\n" +
	"\vGetDLQTasks\x127.temporal.server.api.adminservice.v1.GetDLQTasksRequest\x1a8.temporal.server.api.adminservice.v1.GetDLQTasksResponse\"\x00\x12\x88\x01\n" +
	"\rPurgeDLQTasks\x129.temporal.server.api.adminservice.v1.PurgeDLQTasksRequest\x1a:.temporal.server.api.adminservice.v1.PurgeDLQTasksResponse\"\x00\x12\x88\x01\n" +
	"\rMergeDLQTasks\x129.temporal.server.api.adminservice.v1.MergeDLQTasksRequest\x1a:.temporal.server.api.adminservice.v1.MergeDLQTasksResponse\"\x00\x12\x8b\x01\n" +
	"\x0eDescribeDLQJob\x12:.temporal.server.api.adminservice.v1.DescribeDLQJobRequest\x1a;.temporal.server.api.adminservice.v1.DescribeDLQJobResponse\"\x00\x12\x85\x01\n" +
	"\fCancelDLQJob\x128.temporal.server.api.adminservice.v1.CancelDLQJobRequest\x1a9.temporal.server.api.adminservice.v1.CancelDLQJobResponse\"\x00\x12y\n" +
	"\bAddTasks\x124.temporal.server.api.adminservice.v1.AddTasksRequest\x1a5.temporal.server.api.adminservice.v1.AddTasksResponse\"\x00\x12\x7f\n" +
	"\n" +
	"ListQueues\x126.temporal.server.api.adminservice.v1.ListQueuesRequest\x1a7.temporal.server.api.adminservice.v1.ListQueuesResponse\"\x00\x12\x8e\x01\n" +
	"\x0fDeepHealthCheck\x12;.temporal.server.api.adminservice.v1.DeepHealthCheckRequest\x1a<.temporal.server.api.adminservice.v1.DeepHealthCheckResponse\"\x00\x12\x94\x01\n" +
	"\x11SyncWorkflowState\x12=.temporal.server.api.adminservice.v1.SyncWorkflowStateRequest\x1a>.temporal.server.api.adminservice.v1.SyncWorkflowStateResponse\"\x00\x12\xca\x01\n" +
	"#GenerateLastHistoryReplicationTasks\x12O.temporal.server.api.adminservice.v1.GenerateLastHistoryReplicationTasksRequest\x1aP.temporal.server.api.adminservice.v1.GenerateLastHistoryReplicationTasksResponse\"\x00\x12\xaf\x01\n" +
	"\x1aDescribeTaskQueuePartition\x12F.temporal.server.api.adminservice.v1.DescribeTaskQueuePartitionRequest\x1aG.temporal.server.api.adminservice.v1.DescribeTaskQueuePartitionResponse\"\x00\x12\xb8\x01\n" +
	"\x1dForceUnloadTaskQueuePartition\x12I.temporal.server.api.adminservice.v1.ForceUnloadTaskQueuePartitionRequest\x1aJ.temporal.server.api.adminservice.v1.ForceUnloadTaskQueuePartitionResponse\"\x00B8Z6go.temporal.io/server/api/adminservice/v1;adminserviceb\x06proto3"

var file_temporal_server_api_adminservice_v1_service_proto_goTypes = []any{
	(*RebuildMutableStateRequest)(nil),                  // 0: temporal.server.api.adminservice.v1.RebuildMutableStateRequest
	(*ImportWorkflowExecutionRequest)(nil),              // 1: temporal.server.api.adminservice.v1.ImportWorkflowExecutionRequest
	(*DescribeMutableStateRequest)(nil),                 // 2: temporal.server.api.adminservice.v1.DescribeMutableStateRequest
	(*DescribeHistoryHostRequest)(nil),                  // 3: temporal.server.api.adminservice.v1.DescribeHistoryHostRequest
	(*GetShardRequest)(nil),                             // 4: temporal.server.api.adminservice.v1.GetShardRequest
	(*CloseShardRequest)(nil),                           // 5: temporal.server.api.adminservice.v1.CloseShardRequest
	(*ListHistoryTasksRequest)(nil),                     // 6: temporal.server.api.adminservice.v1.ListHistoryTasksRequest
	(*RemoveTaskRequest)(nil),                           // 7: temporal.server.api.adminservice.v1.RemoveTaskRequest
	(*GetWorkflowExecutionRawHistoryV2Request)(nil),     // 8: temporal.server.api.adminservice.v1.GetWorkflowExecutionRawHistoryV2Request
	(*GetWorkflowExecutionRawHistoryRequest)(nil),       // 9: temporal.server.api.adminservice.v1.GetWorkflowExecutionRawHistoryRequest
	(*GetReplicationMessagesRequest)(nil),               // 10: temporal.server.api.adminservice.v1.GetReplicationMessagesRequest
	(*GetNamespaceReplicationMessagesRequest)(nil),      // 11: temporal.server.api.adminservice.v1.GetNamespaceReplicationMessagesRequest
	(*GetDLQReplicationMessagesRequest)(nil),            // 12: temporal.server.api.adminservice.v1.GetDLQReplicationMessagesRequest
	(*ReapplyEventsRequest)(nil),                        // 13: temporal.server.api.adminservice.v1.ReapplyEventsRequest
	(*AddSearchAttributesRequest)(nil),                  // 14: temporal.server.api.adminservice.v1.AddSearchAttributesRequest
	(*RemoveSearchAttributesRequest)(nil),               // 15: temporal.server.api.adminservice.v1.RemoveSearchAttributesRequest
	(*GetSearchAttributesRequest)(nil),                  // 16: temporal.server.api.adminservice.v1.GetSearchAttributesRequest
	(*DescribeClusterRequest)(nil),                      // 17: temporal.server.api.adminservice.v1.DescribeClusterRequest
	(*ListClustersRequest)(nil),                         // 18: temporal.server.api.adminservice.v1.ListClustersRequest
	(*ListClusterMembersRequest)(nil),                   // 19: temporal.server.api.adminservice.v1.ListClusterMembersRequest
	(*AddOrUpdateRemoteClusterRequest)(nil),             // 20: temporal.server.api.adminservice.v1.AddOrUpdateRemoteClusterRequest
	(*RemoveRemoteClusterRequest)(nil),                  // 21: temporal.server.api.adminservice.v1.RemoveRemoteClusterRequest
	(*GetDLQMessagesRequest)(nil),                       // 22: temporal.server.api.adminservice.v1.GetDLQMessagesRequest
	(*PurgeDLQMessagesRequest)(nil),                     // 23: temporal.server.api.adminservice.v1.PurgeDLQMessagesRequest
	(*MergeDLQMessagesRequest)(nil),                     // 24: temporal.server.api.adminservice.v1.MergeDLQMessagesRequest
	(*RefreshWorkflowTasksRequest)(nil),                 // 25: temporal.server.api.adminservice.v1.RefreshWorkflowTasksRequest
	(*ResendReplicationTasksRequest)(nil),               // 26: temporal.server.api.adminservice.v1.ResendReplicationTasksRequest
	(*GetTaskQueueTasksRequest)(nil),                    // 27: temporal.server.api.adminservice.v1.GetTaskQueueTasksRequest
	(*DeleteWorkflowExecutionRequest)(nil),              // 28: temporal.server.api.adminservice.v1.DeleteWorkflowExecutionRequest
	(*StreamWorkflowReplicationMessagesRequest)(nil),    // 29: temporal.server.api.adminservice.v1.StreamWorkflowReplicationMessagesRequest
	(*GetNamespaceRequest)(nil),                         // 30: temporal.server.api.adminservice.v1.GetNamespaceRequest
	(*GetDLQTasksRequest)(nil),                          // 31: temporal.server.api.adminservice.v1.GetDLQTasksRequest
	(*PurgeDLQTasksRequest)(nil),                        // 32: temporal.server.api.adminservice.v1.PurgeDLQTasksRequest
	(*MergeDLQTasksRequest)(nil),                        // 33: temporal.server.api.adminservice.v1.MergeDLQTasksRequest
	(*DescribeDLQJobRequest)(nil),                       // 34: temporal.server.api.adminservice.v1.DescribeDLQJobRequest
	(*CancelDLQJobRequest)(nil),                         // 35: temporal.server.api.adminservice.v1.CancelDLQJobRequest
	(*AddTasksRequest)(nil),                             // 36: temporal.server.api.adminservice.v1.AddTasksRequest
	(*ListQueuesRequest)(nil),                           // 37: temporal.server.api.adminservice.v1.ListQueuesRequest
	(*DeepHealthCheckRequest)(nil),                      // 38: temporal.server.api.adminservice.v1.DeepHealthCheckRequest
	(*SyncWorkflowStateRequest)(nil),                    // 39: temporal.server.api.adminservice.v1.SyncWorkflowStateRequest
	(*GenerateLastHistoryReplicationTasksRequest)(nil),  // 40: temporal.server.api.adminservice.v1.GenerateLastHistoryReplicationTasksRequest
	(*DescribeTaskQueuePartitionRequest)(nil),           // 41: temporal.server.api.adminservice.v1.DescribeTaskQueuePartitionRequest
	(*ForceUnloadTaskQueuePartitionRequest)(nil),        // 42: temporal.server.api.adminservice.v1.ForceUnloadTaskQueuePartitionRequest
	(*RebuildMutableStateResponse)(nil),                 // 43: temporal.server.api.adminservice.v1.RebuildMutableStateResponse
	(*ImportWorkflowExecutionResponse)(nil),             // 44: temporal.server.api.adminservice.v1.ImportWorkflowExecutionResponse
	(*DescribeMutableStateResponse)(nil),                // 45: temporal.server.api.adminservice.v1.DescribeMutableStateResponse
	(*DescribeHistoryHostResponse)(nil),                 // 46: temporal.server.api.adminservice.v1.DescribeHistoryHostResponse
	(*GetShardResponse)(nil),                            // 47: temporal.server.api.adminservice.v1.GetShardResponse
	(*CloseShardResponse)(nil),                          // 48: temporal.server.api.adminservice.v1.CloseShardResponse
	(*ListHistoryTasksResponse)(nil),                    // 49: temporal.server.api.adminservice.v1.ListHistoryTasksResponse
	(*RemoveTaskResponse)(nil),                          // 50: temporal.server.api.adminservice.v1.RemoveTaskResponse
	(*GetWorkflowExecutionRawHistoryV2Response)(nil),    // 51: temporal.server.api.adminservice.v1.GetWorkflowExecutionRawHistoryV2Response
	(*GetWorkflowExecutionRawHistoryResponse)(nil),      // 52: temporal.server.api.adminservice.v1.GetWorkflowExecutionRawHistoryResponse
	(*GetReplicationMessagesResponse)(nil),              // 53: temporal.server.api.adminservice.v1.GetReplicationMessagesResponse
	(*GetNamespaceReplicationMessagesResponse)(nil),     // 54: temporal.server.api.adminservice.v1.GetNamespaceReplicationMessagesResponse
	(*GetDLQReplicationMessagesResponse)(nil),           // 55: temporal.server.api.adminservice.v1.GetDLQReplicationMessagesResponse
	(*ReapplyEventsResponse)(nil),                       // 56: temporal.server.api.adminservice.v1.ReapplyEventsResponse
	(*AddSearchAttributesResponse)(nil),                 // 57: temporal.server.api.adminservice.v1.AddSearchAttributesResponse
	(*RemoveSearchAttributesResponse)(nil),              // 58: temporal.server.api.adminservice.v1.RemoveSearchAttributesResponse
	(*GetSearchAttributesResponse)(nil),                 // 59: temporal.server.api.adminservice.v1.GetSearchAttributesResponse
	(*DescribeClusterResponse)(nil),                     // 60: temporal.server.api.adminservice.v1.DescribeClusterResponse
	(*ListClustersResponse)(nil),                        // 61: temporal.server.api.adminservice.v1.ListClustersResponse
	(*ListClusterMembersResponse)(nil),                  // 62: temporal.server.api.adminservice.v1.ListClusterMembersResponse
	(*AddOrUpdateRemoteClusterResponse)(nil),            // 63: temporal.server.api.adminservice.v1.AddOrUpdateRemoteClusterResponse
	(*RemoveRemoteClusterResponse)(nil),                 // 64: temporal.server.api.adminservice.v1.RemoveRemoteClusterResponse
	(*GetDLQMessagesResponse)(nil),                      // 65: temporal.server.api.adminservice.v1.GetDLQMessagesResponse
	(*PurgeDLQMessagesResponse)(nil),                    // 66: temporal.server.api.adminservice.v1.PurgeDLQMessagesResponse
	(*MergeDLQMessagesResponse)(nil),                    // 67: temporal.server.api.adminservice.v1.MergeDLQMessagesResponse
	(*RefreshWorkflowTasksResponse)(nil),                // 68: temporal.server.api.adminservice.v1.RefreshWorkflowTasksResponse
	(*ResendReplicationTasksResponse)(nil),              // 69: temporal.server.api.adminservice.v1.ResendReplicationTasksResponse
	(*GetTaskQueueTasksResponse)(nil),                   // 70: temporal.server.api.adminservice.v1.GetTaskQueueTasksResponse
	(*DeleteWorkflowExecutionResponse)(nil),             // 71: temporal.server.api.adminservice.v1.DeleteWorkflowExecutionResponse
	(*StreamWorkflowReplicationMessagesResponse)(nil),   // 72: temporal.server.api.adminservice.v1.StreamWorkflowReplicationMessagesResponse
	(*GetNamespaceResponse)(nil),                        // 73: temporal.server.api.adminservice.v1.GetNamespaceResponse
	(*GetDLQTasksResponse)(nil),                         // 74: temporal.server.api.adminservice.v1.GetDLQTasksResponse
	(*PurgeDLQTasksResponse)(nil),                       // 75: temporal.server.api.adminservice.v1.PurgeDLQTasksResponse
	(*MergeDLQTasksResponse)(nil),                       // 76: temporal.server.api.adminservice.v1.MergeDLQTasksResponse
	(*DescribeDLQJobResponse)(nil),                      // 77: temporal.server.api.adminservice.v1.DescribeDLQJobResponse
	(*CancelDLQJobResponse)(nil),                        // 78: temporal.server.api.adminservice.v1.CancelDLQJobResponse
	(*AddTasksResponse)(nil),                            // 79: temporal.server.api.adminservice.v1.AddTasksResponse
	(*ListQueuesResponse)(nil),                          // 80: temporal.server.api.adminservice.v1.ListQueuesResponse
	(*DeepHealthCheckResponse)(nil),                     // 81: temporal.server.api.adminservice.v1.DeepHealthCheckResponse
	(*SyncWorkflowStateResponse)(nil),                   // 82: temporal.server.api.adminservice.v1.SyncWorkflowStateResponse
	(*GenerateLastHistoryReplicationTasksResponse)(nil), // 83: temporal.server.api.adminservice.v1.GenerateLastHistoryReplicationTasksResponse
	(*DescribeTaskQueuePartitionResponse)(nil),          // 84: temporal.server.api.adminservice.v1.DescribeTaskQueuePartitionResponse
	(*ForceUnloadTaskQueuePartitionResponse)(nil),       // 85: temporal.server.api.adminservice.v1.ForceUnloadTaskQueuePartitionResponse
}
var file_temporal_server_api_adminservice_v1_service_proto_depIdxs = []int32{
	0,  // 0: temporal.server.api.adminservice.v1.AdminService.RebuildMutableState:input_type -> temporal.server.api.adminservice.v1.RebuildMutableStateRequest
	1,  // 1: temporal.server.api.adminservice.v1.AdminService.ImportWorkflowExecution:input_type -> temporal.server.api.adminservice.v1.ImportWorkflowExecutionRequest
	2,  // 2: temporal.server.api.adminservice.v1.AdminService.DescribeMutableState:input_type -> temporal.server.api.adminservice.v1.DescribeMutableStateRequest
	3,  // 3: temporal.server.api.adminservice.v1.AdminService.DescribeHistoryHost:input_type -> temporal.server.api.adminservice.v1.DescribeHistoryHostRequest
	4,  // 4: temporal.server.api.adminservice.v1.AdminService.GetShard:input_type -> temporal.server.api.adminservice.v1.GetShardRequest
	5,  // 5: temporal.server.api.adminservice.v1.AdminService.CloseShard:input_type -> temporal.server.api.adminservice.v1.CloseShardRequest
	6,  // 6: temporal.server.api.adminservice.v1.AdminService.ListHistoryTasks:input_type -> temporal.server.api.adminservice.v1.ListHistoryTasksRequest
	7,  // 7: temporal.server.api.adminservice.v1.AdminService.RemoveTask:input_type -> temporal.server.api.adminservice.v1.RemoveTaskRequest
	8,  // 8: temporal.server.api.adminservice.v1.AdminService.GetWorkflowExecutionRawHistoryV2:input_type -> temporal.server.api.adminservice.v1.GetWorkflowExecutionRawHistoryV2Request
	9,  // 9: temporal.server.api.adminservice.v1.AdminService.GetWorkflowExecutionRawHistory:input_type -> temporal.server.api.adminservice.v1.GetWorkflowExecutionRawHistoryRequest
	10, // 10: temporal.server.api.adminservice.v1.AdminService.GetReplicationMessages:input_type -> temporal.server.api.adminservice.v1.GetReplicationMessagesRequest
	11, // 11: temporal.server.api.adminservice.v1.AdminService.GetNamespaceReplicationMessages:input_type -> temporal.server.api.adminservice.v1.GetNamespaceReplicationMessagesRequest
	12, // 12: temporal.server.api.adminservice.v1.AdminService.GetDLQReplicationMessages:input_type -> temporal.server.api.adminservice.v1.GetDLQReplicationMessagesRequest
	13, // 13: temporal.server.api.adminservice.v1.AdminService.ReapplyEvents:input_type -> temporal.server.api.adminservice.v1.ReapplyEventsRequest
	14, // 14: temporal.server.api.adminservice.v1.AdminService.AddSearchAttributes:input_type -> temporal.server.api.adminservice.v1.AddSearchAttributesRequest
	15, // 15: temporal.server.api.adminservice.v1.AdminService.RemoveSearchAttributes:input_type -> temporal.server.api.adminservice.v1.RemoveSearchAttributesRequest
	16, // 16: temporal.server.api.adminservice.v1.AdminService.GetSearchAttributes:input_type -> temporal.server.api.adminservice.v1.GetSearchAttributesRequest
	17, // 17: temporal.server.api.adminservice.v1.AdminService.DescribeCluster:input_type -> temporal.server.api.adminservice.v1.DescribeClusterRequest
	18, // 18: temporal.server.api.adminservice.v1.AdminService.ListClusters:input_type -> temporal.server.api.adminservice.v1.ListClustersRequest
	19, // 19: temporal.server.api.adminservice.v1.AdminService.ListClusterMembers:input_type -> temporal.server.api.adminservice.v1.ListClusterMembersRequest
	20, // 20: temporal.server.api.adminservice.v1.AdminService.AddOrUpdateRemoteCluster:input_type -> temporal.server.api.adminservice.v1.AddOrUpdateRemoteClusterRequest
	21, // 21: temporal.server.api.adminservice.v1.AdminService.RemoveRemoteCluster:input_type -> temporal.server.api.adminservice.v1.RemoveRemoteClusterRequest
	22, // 22: temporal.server.api.adminservice.v1.AdminService.GetDLQMessages:input_type -> temporal.server.api.adminservice.v1.GetDLQMessagesRequest
	23, // 23: temporal.server.api.adminservice.v1.AdminService.PurgeDLQMessages:input_type -> temporal.server.api.adminservice.v1.PurgeDLQMessagesRequest
	24, // 24: temporal.server.api.adminservice.v1.AdminService.MergeDLQMessages:input_type -> temporal.server.api.adminservice.v1.MergeDLQMessagesRequest
	25, // 25: temporal.server.api.adminservice.v1.AdminService.RefreshWorkflowTasks:input_type -> temporal.server.api.adminservice.v1.RefreshWorkflowTasksRequest
	26, // 26: temporal.server.api.adminservice.v1.AdminService.ResendReplicationTasks:input_type -> temporal.server.api.adminservice.v1.ResendReplicationTasksRequest
	27, // 27: temporal.server.api.adminservice.v1.AdminService.GetTaskQueueTasks:input_type -> temporal.server.api.adminservice.v1.GetTaskQueueTasksRequest
	28, // 28: temporal.server.api.adminservice.v1.AdminService.DeleteWorkflowExecution:input_type -> temporal.server.api.adminservice.v1.DeleteWorkflowExecutionRequest
	29, // 29: temporal.server.api.adminservice.v1.AdminService.StreamWorkflowReplicationMessages:input_type -> temporal.server.api.adminservice.v1.StreamWorkflowReplicationMessagesRequest
	30, // 30: temporal.server.api.adminservice.v1.AdminService.GetNamespace:input_type -> temporal.server.api.adminservice.v1.GetNamespaceRequest
	31, // 31: temporal.server.api.adminservice.v1.AdminService.GetDLQTasks:input_type -> temporal.server.api.adminservice.v1.GetDLQTasksRequest
	32, // 32: temporal.server.api.adminservice.v1.AdminService.PurgeDLQTasks:input_type -> temporal.server.api.adminservice.v1.PurgeDLQTasksRequest
	33, // 33: temporal.server.api.adminservice.v1.AdminService.MergeDLQTasks:input_type -> temporal.server.api.adminservice.v1.MergeDLQTasksRequest
	34, // 34: temporal.server.api.adminservice.v1.AdminService.DescribeDLQJob:input_type -> temporal.server.api.adminservice.v1.DescribeDLQJobRequest
	35, // 35: temporal.server.api.adminservice.v1.AdminService.CancelDLQJob:input_type -> temporal.server.api.adminservice.v1.CancelDLQJobRequest
	36, // 36: temporal.server.api.adminservice.v1.AdminService.AddTasks:input_type -> temporal.server.api.adminservice.v1.AddTasksRequest
	37, // 37: temporal.server.api.adminservice.v1.AdminService.ListQueues:input_type -> temporal.server.api.adminservice.v1.ListQueuesRequest
	38, // 38: temporal.server.api.adminservice.v1.AdminService.DeepHealthCheck:input_type -> temporal.server.api.adminservice.v1.DeepHealthCheckRequest
	39, // 39: temporal.server.api.adminservice.v1.AdminService.SyncWorkflowState:input_type -> temporal.server.api.adminservice.v1.SyncWorkflowStateRequest
	40, // 40: temporal.server.api.adminservice.v1.AdminService.GenerateLastHistoryReplicationTasks:input_type -> temporal.server.api.adminservice.v1.GenerateLastHistoryReplicationTasksRequest
	41, // 41: temporal.server.api.adminservice.v1.AdminService.DescribeTaskQueuePartition:input_type -> temporal.server.api.adminservice.v1.DescribeTaskQueuePartitionRequest
	42, // 42: temporal.server.api.adminservice.v1.AdminService.ForceUnloadTaskQueuePartition:input_type -> temporal.server.api.adminservice.v1.ForceUnloadTaskQueuePartitionRequest
	43, // 43: temporal.server.api.adminservice.v1.AdminService.RebuildMutableState:output_type -> temporal.server.api.adminservice.v1.RebuildMutableStateResponse
	44, // 44: temporal.server.api.adminservice.v1.AdminService.ImportWorkflowExecution:output_type -> temporal.server.api.adminservice.v1.ImportWorkflowExecutionResponse
	45, // 45: temporal.server.api.adminservice.v1.AdminService.DescribeMutableState:output_type -> temporal.server.api.adminservice.v1.DescribeMutableStateResponse
	46, // 46: temporal.server.api.adminservice.v1.AdminService.DescribeHistoryHost:output_type -> temporal.server.api.adminservice.v1.DescribeHistoryHostResponse
	47, // 47: temporal.server.api.adminservice.v1.AdminService.GetShard:output_type -> temporal.server.api.adminservice.v1.GetShardResponse
	48, // 48: temporal.server.api.adminservice.v1.AdminService.CloseShard:output_type -> temporal.server.api.adminservice.v1.CloseShardResponse
	49, // 49: temporal.server.api.adminservice.v1.AdminService.ListHistoryTasks:output_type -> temporal.server.api.adminservice.v1.ListHistoryTasksResponse
	50, // 50: temporal.server.api.adminservice.v1.AdminService.RemoveTask:output_type -> temporal.server.api.adminservice.v1.RemoveTaskResponse
	51, // 51: temporal.server.api.adminservice.v1.AdminService.GetWorkflowExecutionRawHistoryV2:output_type -> temporal.server.api.adminservice.v1.GetWorkflowExecutionRawHistoryV2Response
	52, // 52: temporal.server.api.adminservice.v1.AdminService.GetWorkflowExecutionRawHistory:output_type -> temporal.server.api.adminservice.v1.GetWorkflowExecutionRawHistoryResponse
	53, // 53: temporal.server.api.adminservice.v1.AdminService.GetReplicationMessages:output_type -> temporal.server.api.adminservice.v1.GetReplicationMessagesResponse
	54, // 54: temporal.server.api.adminservice.v1.AdminService.GetNamespaceReplicationMessages:output_type -> temporal.server.api.adminservice.v1.GetNamespaceReplicationMessagesResponse
	55, // 55: temporal.server.api.adminservice.v1.AdminService.GetDLQReplicationMessages:output_type -> temporal.server.api.adminservice.v1.GetDLQReplicationMessagesResponse
	56, // 56: temporal.server.api.adminservice.v1.AdminService.ReapplyEvents:output_type -> temporal.server.api.adminservice.v1.ReapplyEventsResponse
	57, // 57: temporal.server.api.adminservice.v1.AdminService.AddSearchAttributes:output_type -> temporal.server.api.adminservice.v1.AddSearchAttributesResponse
	58, // 58: temporal.server.api.adminservice.v1.AdminService.RemoveSearchAttributes:output_type -> temporal.server.api.adminservice.v1.RemoveSearchAttributesResponse
	59, // 59: temporal.server.api.adminservice.v1.AdminService.GetSearchAttributes:output_type -> temporal.server.api.adminservice.v1.GetSearchAttributesResponse
	60, // 60: temporal.server.api.adminservice.v1.AdminService.DescribeCluster:output_type -> temporal.server.api.adminservice.v1.DescribeClusterResponse
	61, // 61: temporal.server.api.adminservice.v1.AdminService.ListClusters:output_type -> temporal.server.api.adminservice.v1.ListClustersResponse
	62, // 62: temporal.server.api.adminservice.v1.AdminService.ListClusterMembers:output_type -> temporal.server.api.adminservice.v1.ListClusterMembersResponse
	63, // 63: temporal.server.api.adminservice.v1.AdminService.AddOrUpdateRemoteCluster:output_type -> temporal.server.api.adminservice.v1.AddOrUpdateRemoteClusterResponse
	64, // 64: temporal.server.api.adminservice.v1.AdminService.RemoveRemoteCluster:output_type -> temporal.server.api.adminservice.v1.RemoveRemoteClusterResponse
	65, // 65: temporal.server.api.adminservice.v1.AdminService.GetDLQMessages:output_type -> temporal.server.api.adminservice.v1.GetDLQMessagesResponse
	66, // 66: temporal.server.api.adminservice.v1.AdminService.PurgeDLQMessages:output_type -> temporal.server.api.adminservice.v1.PurgeDLQMessagesResponse
	67, // 67: temporal.server.api.adminservice.v1.AdminService.MergeDLQMessages:output_type -> temporal.server.api.adminservice.v1.MergeDLQMessagesResponse
	68, // 68: temporal.server.api.adminservice.v1.AdminService.RefreshWorkflowTasks:output_type -> temporal.server.api.adminservice.v1.RefreshWorkflowTasksResponse
	69, // 69: temporal.server.api.adminservice.v1.AdminService.ResendReplicationTasks:output_type -> temporal.server.api.adminservice.v1.ResendReplicationTasksResponse
	70, // 70: temporal.server.api.adminservice.v1.AdminService.GetTaskQueueTasks:output_type -> temporal.server.api.adminservice.v1.GetTaskQueueTasksResponse
	71, // 71: temporal.server.api.adminservice.v1.AdminService.DeleteWorkflowExecution:output_type -> temporal.server.api.adminservice.v1.DeleteWorkflowExecutionResponse
	72, // 72: temporal.server.api.adminservice.v1.AdminService.StreamWorkflowReplicationMessages:output_type -> temporal.server.api.adminservice.v1.StreamWorkflowReplicationMessagesResponse
	73, // 73: temporal.server.api.adminservice.v1.AdminService.GetNamespace:output_type -> temporal.server.api.adminservice.v1.GetNamespaceResponse
	74, // 74: temporal.server.api.adminservice.v1.AdminService.GetDLQTasks:output_type -> temporal.server.api.adminservice.v1.GetDLQTasksResponse
	75, // 75: temporal.server.api.adminservice.v1.AdminService.PurgeDLQTasks:output_type -> temporal.server.api.adminservice.v1.PurgeDLQTasksResponse
	76, // 76: temporal.server.api.adminservice.v1.AdminService.MergeDLQTasks:output_type -> temporal.server.api.adminservice.v1.MergeDLQTasksResponse
	77, // 77: temporal.server.api.adminservice.v1.AdminService.DescribeDLQJob:output_type -> temporal.server.api.adminservice.v1.DescribeDLQJobResponse
	78, // 78: temporal.server.api.adminservice.v1.AdminService.CancelDLQJob:output_type -> temporal.server.api.adminservice.v1.CancelDLQJobResponse
	79, // 79: temporal.server.api.adminservice.v1.AdminService.AddTasks:output_type -> temporal.server.api.adminservice.v1.AddTasksResponse
	80, // 80: temporal.server.api.adminservice.v1.AdminService.ListQueues:output_type -> temporal.server.api.adminservice.v1.ListQueuesResponse
	81, // 81: temporal.server.api.adminservice.v1.AdminService.DeepHealthCheck:output_type -> temporal.server.api.adminservice.v1.DeepHealthCheckResponse
	82, // 82: temporal.server.api.adminservice.v1.AdminService.SyncWorkflowState:output_type -> temporal.server.api.adminservice.v1.SyncWorkflowStateResponse
	83, // 83: temporal.server.api.adminservice.v1.AdminService.GenerateLastHistoryReplicationTasks:output_type -> temporal.server.api.adminservice.v1.GenerateLastHistoryReplicationTasksResponse
	84, // 84: temporal.server.api.adminservice.v1.AdminService.DescribeTaskQueuePartition:output_type -> temporal.server.api.adminservice.v1.DescribeTaskQueuePartitionResponse
	85, // 85: temporal.server.api.adminservice.v1.AdminService.ForceUnloadTaskQueuePartition:output_type -> temporal.server.api.adminservice.v1.ForceUnloadTaskQueuePartitionResponse
	43, // [43:86] is the sub-list for method output_type
	0,  // [0:43] is the sub-list for method input_type
	0,  // [0:0] is the sub-list for extension type_name
	0,  // [0:0] is the sub-list for extension extendee
	0,  // [0:0] is the sub-list for field type_name
}

func init() { file_temporal_server_api_adminservice_v1_service_proto_init() }
func file_temporal_server_api_adminservice_v1_service_proto_init() {
	if File_temporal_server_api_adminservice_v1_service_proto != nil {
		return
	}
	file_temporal_server_api_adminservice_v1_request_response_proto_init()
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_temporal_server_api_adminservice_v1_service_proto_rawDesc), len(file_temporal_server_api_adminservice_v1_service_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   0,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_temporal_server_api_adminservice_v1_service_proto_goTypes,
		DependencyIndexes: file_temporal_server_api_adminservice_v1_service_proto_depIdxs,
	}.Build()
	File_temporal_server_api_adminservice_v1_service_proto = out.File
	file_temporal_server_api_adminservice_v1_service_proto_goTypes = nil
	file_temporal_server_api_adminservice_v1_service_proto_depIdxs = nil
}
