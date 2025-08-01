//go:generate mockgen -package $GOPACKAGE -source $GOFILE -destination dlq_handler_mock.go

package replication

import (
	"context"
	"fmt"
	"sync"

	"go.temporal.io/server/api/adminservice/v1"
	enumsspb "go.temporal.io/server/api/enums/v1"
	replicationspb "go.temporal.io/server/api/replication/v1"
	"go.temporal.io/server/client"
	"go.temporal.io/server/common/log"
	"go.temporal.io/server/common/log/tag"
	"go.temporal.io/server/common/namespace"
	"go.temporal.io/server/common/persistence"
	"go.temporal.io/server/service/history/deletemanager"
	historyi "go.temporal.io/server/service/history/interfaces"
	"go.temporal.io/server/service/history/replication/eventhandler"
	"go.temporal.io/server/service/history/tasks"
	wcache "go.temporal.io/server/service/history/workflow/cache"
)

type (
	// DLQHandler is the interface handles replication DLQ messages
	DLQHandler interface {
		GetMessages(
			ctx context.Context,
			sourceCluster string,
			lastMessageID int64,
			pageSize int,
			pageToken []byte,
		) ([]*replicationspb.ReplicationTask, []*replicationspb.ReplicationTaskInfo, []byte, error)
		PurgeMessages(
			ctx context.Context,
			sourceCluster string,
			lastMessageID int64,
		) error
		MergeMessages(
			ctx context.Context,
			sourceCluster string,
			lastMessageID int64,
			pageSize int,
			pageToken []byte,
		) ([]byte, error)
	}

	dlqHandlerImpl struct {
		taskExecutorsLock    sync.Mutex
		taskExecutors        map[string]TaskExecutor
		shard                historyi.ShardContext
		deleteManager        deletemanager.DeleteManager
		workflowCache        wcache.Cache
		resender             eventhandler.ResendHandler
		taskExecutorProvider TaskExecutorProvider
		logger               log.Logger
	}
)

func NewLazyDLQHandler(
	shard historyi.ShardContext,
	deleteManager deletemanager.DeleteManager,
	workflowCache wcache.Cache,
	clientBean client.Bean,
	taskExecutorProvider TaskExecutorProvider,
) DLQHandler {
	return newDLQHandler(
		shard,
		deleteManager,
		workflowCache,
		clientBean,
		make(map[string]TaskExecutor),
		taskExecutorProvider,
	)
}

func newDLQHandler(
	shard historyi.ShardContext,
	deleteManager deletemanager.DeleteManager,
	workflowCache wcache.Cache,
	clientBean client.Bean,
	taskExecutors map[string]TaskExecutor,
	taskExecutorProvider TaskExecutorProvider,
) *dlqHandlerImpl {

	if taskExecutors == nil {
		panic("Failed to initialize replication DLQ handler due to nil task executors")
	}
	historyFetcher := eventhandler.NewHistoryPaginatedFetcher(shard.GetNamespaceRegistry(), clientBean, shard.GetPayloadSerializer(), shard.GetLogger())
	engineProvider := func(ctx context.Context, namespaceId namespace.ID, workflowId string) (historyi.Engine, error) {
		return shard.GetEngine(ctx)
	}
	return &dlqHandlerImpl{
		shard:         shard,
		deleteManager: deleteManager,
		workflowCache: workflowCache,
		resender: eventhandler.NewResendHandler(
			shard.GetNamespaceRegistry(),
			clientBean,
			shard.GetPayloadSerializer(),
			shard.GetClusterMetadata(),
			engineProvider,
			historyFetcher,
			eventhandler.NewEventImporter(historyFetcher, engineProvider, shard.GetPayloadSerializer(), shard.GetLogger()),
			shard.GetLogger(),
			shard.GetConfig(),
		),
		taskExecutors:        taskExecutors,
		taskExecutorProvider: taskExecutorProvider,
		logger:               shard.GetLogger(),
	}
}

func (r *dlqHandlerImpl) GetMessages(
	ctx context.Context,
	sourceCluster string,
	lastMessageID int64,
	pageSize int,
	pageToken []byte,
) ([]*replicationspb.ReplicationTask, []*replicationspb.ReplicationTaskInfo, []byte, error) {

	taskList, taskInfoList, _, token, err := r.readMessagesWithAckLevel(
		ctx,
		sourceCluster,
		lastMessageID,
		pageSize,
		pageToken,
	)
	return taskList, taskInfoList, token, err
}

func (r *dlqHandlerImpl) PurgeMessages(
	ctx context.Context,
	sourceCluster string,
	lastMessageID int64,
) error {

	ackLevel := r.shard.GetReplicatorDLQAckLevel(sourceCluster)
	err := r.shard.GetExecutionManager().RangeDeleteReplicationTaskFromDLQ(
		ctx,
		&persistence.RangeDeleteReplicationTaskFromDLQRequest{
			RangeCompleteHistoryTasksRequest: persistence.RangeCompleteHistoryTasksRequest{
				ShardID:             r.shard.GetShardID(),
				TaskCategory:        tasks.CategoryReplication,
				InclusiveMinTaskKey: tasks.NewImmediateKey(ackLevel + 1),
				ExclusiveMaxTaskKey: tasks.NewImmediateKey(lastMessageID + 1),
			},
			SourceClusterName: sourceCluster,
		},
	)
	if err != nil {
		return err
	}

	if err = r.shard.UpdateReplicatorDLQAckLevel(
		sourceCluster,
		lastMessageID,
	); err != nil {
		r.logger.Error("Failed to purge history replication message", tag.Error(err))
		// The update ack level should not block the call. Ignore the error.
	}
	return nil
}

func (r *dlqHandlerImpl) MergeMessages(
	ctx context.Context,
	sourceCluster string,
	lastMessageID int64,
	pageSize int,
	pageToken []byte,
) ([]byte, error) {

	replicationTasks, _, ackLevel, token, err := r.readMessagesWithAckLevel(
		ctx,
		sourceCluster,
		lastMessageID,
		pageSize,
		pageToken,
	)
	if err != nil {
		return nil, err
	}

	taskExecutor, err := r.getOrCreateTaskExecutor(sourceCluster)
	if err != nil {
		return nil, err
	}

	for _, task := range replicationTasks {
		if err := taskExecutor.Execute(
			ctx,
			task,
			true,
		); err != nil {
			return nil, err
		}
	}

	err = r.shard.GetExecutionManager().RangeDeleteReplicationTaskFromDLQ(
		ctx,
		&persistence.RangeDeleteReplicationTaskFromDLQRequest{
			RangeCompleteHistoryTasksRequest: persistence.RangeCompleteHistoryTasksRequest{
				ShardID:             r.shard.GetShardID(),
				TaskCategory:        tasks.CategoryReplication,
				InclusiveMinTaskKey: tasks.NewImmediateKey(ackLevel + 1),
				ExclusiveMaxTaskKey: tasks.NewImmediateKey(lastMessageID + 1),
			},
			SourceClusterName: sourceCluster,
		},
	)
	if err != nil {
		return nil, err
	}

	if err = r.shard.UpdateReplicatorDLQAckLevel(
		sourceCluster,
		lastMessageID,
	); err != nil {
		r.logger.Error("Failed to purge history replication message", tag.Error(err))
		// The update ack level should not block the call. Ignore the error.
	}
	return token, nil
}

func (r *dlqHandlerImpl) readMessagesWithAckLevel(
	ctx context.Context,
	sourceCluster string,
	lastMessageID int64,
	pageSize int,
	pageToken []byte,
) ([]*replicationspb.ReplicationTask, []*replicationspb.ReplicationTaskInfo, int64, []byte, error) {

	ackLevel := r.shard.GetReplicatorDLQAckLevel(sourceCluster)
	resp, err := r.shard.GetExecutionManager().GetReplicationTasksFromDLQ(ctx, &persistence.GetReplicationTasksFromDLQRequest{
		GetHistoryTasksRequest: persistence.GetHistoryTasksRequest{
			ShardID:             r.shard.GetShardID(),
			TaskCategory:        tasks.CategoryReplication,
			InclusiveMinTaskKey: tasks.NewImmediateKey(ackLevel + 1),
			ExclusiveMaxTaskKey: tasks.NewImmediateKey(lastMessageID + 1),
			BatchSize:           pageSize,
			NextPageToken:       pageToken,
		},
		SourceClusterName: sourceCluster,
	})
	if err != nil {
		return nil, nil, ackLevel, nil, err
	}
	pageToken = resp.NextPageToken

	remoteAdminClient, err := r.shard.GetRemoteAdminClient(sourceCluster)
	if err != nil {
		return nil, nil, ackLevel, nil, err
	}
	taskInfo := make([]*replicationspb.ReplicationTaskInfo, 0, len(resp.Tasks))
	for _, task := range resp.Tasks {
		switch task := task.(type) {
		case *tasks.SyncActivityTask:
			taskInfo = append(taskInfo, &replicationspb.ReplicationTaskInfo{
				NamespaceId:      task.NamespaceID,
				WorkflowId:       task.WorkflowID,
				RunId:            task.RunID,
				TaskType:         enumsspb.TASK_TYPE_REPLICATION_SYNC_ACTIVITY,
				TaskId:           task.TaskID,
				Version:          task.GetVersion(),
				FirstEventId:     0,
				NextEventId:      0,
				ScheduledEventId: task.ScheduledEventID,
			})
		case *tasks.HistoryReplicationTask:
			taskInfo = append(taskInfo, &replicationspb.ReplicationTaskInfo{
				NamespaceId:      task.NamespaceID,
				WorkflowId:       task.WorkflowID,
				RunId:            task.RunID,
				TaskType:         enumsspb.TASK_TYPE_REPLICATION_HISTORY,
				TaskId:           task.TaskID,
				Version:          task.Version,
				FirstEventId:     task.FirstEventID,
				NextEventId:      task.NextEventID,
				ScheduledEventId: 0,
			})
		case *tasks.SyncWorkflowStateTask:
			taskInfo = append(taskInfo, &replicationspb.ReplicationTaskInfo{
				NamespaceId:      task.NamespaceID,
				WorkflowId:       task.WorkflowID,
				RunId:            task.RunID,
				TaskType:         enumsspb.TASK_TYPE_REPLICATION_SYNC_WORKFLOW_STATE,
				TaskId:           task.TaskID,
				Version:          task.Version,
				FirstEventId:     0,
				NextEventId:      0,
				ScheduledEventId: 0,
			})
		case *tasks.SyncHSMTask:
			taskInfo = append(taskInfo, &replicationspb.ReplicationTaskInfo{
				NamespaceId: task.NamespaceID,
				WorkflowId:  task.WorkflowID,
				RunId:       task.RunID,
				TaskType:    enumsspb.TASK_TYPE_REPLICATION_SYNC_HSM,
				TaskId:      task.TaskID,
			})
		default:
			panic(fmt.Sprintf("Unknown repication task type: %v", task))
		}
	}

	if len(taskInfo) == 0 {
		return nil, nil, ackLevel, pageToken, nil
	}

	dlqResponse, err := remoteAdminClient.GetDLQReplicationMessages(
		ctx,
		&adminservice.GetDLQReplicationMessagesRequest{
			TaskInfos: taskInfo,
		},
	)
	if err != nil {
		return nil, nil, ackLevel, nil, err
	}

	return dlqResponse.ReplicationTasks, taskInfo, ackLevel, pageToken, nil
}

func (r *dlqHandlerImpl) getOrCreateTaskExecutor(clusterName string) (TaskExecutor, error) {
	r.taskExecutorsLock.Lock()
	defer r.taskExecutorsLock.Unlock()
	if executor, ok := r.taskExecutors[clusterName]; ok {
		return executor, nil
	}
	taskExecutor := r.taskExecutorProvider(TaskExecutorParams{
		RemoteCluster:   clusterName,
		Shard:           r.shard,
		HistoryResender: r.resender,
		DeleteManager:   r.deleteManager,
		WorkflowCache:   r.workflowCache,
	})
	r.taskExecutors[clusterName] = taskExecutor
	return taskExecutor, nil
}
