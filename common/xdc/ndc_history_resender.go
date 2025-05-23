//go:generate mockgen -package $GOPACKAGE -source $GOFILE -destination ndc_history_resender_mock.go

package xdc

import (
	"context"
	"time"

	commonpb "go.temporal.io/api/common/v1"
	historypb "go.temporal.io/api/history/v1"
	"go.temporal.io/server/api/adminservice/v1"
	historyspb "go.temporal.io/server/api/history/v1"
	"go.temporal.io/server/client"
	"go.temporal.io/server/common/collection"
	"go.temporal.io/server/common/dynamicconfig"
	"go.temporal.io/server/common/log"
	"go.temporal.io/server/common/log/tag"
	"go.temporal.io/server/common/namespace"
	"go.temporal.io/server/common/persistence/serialization"
	"go.temporal.io/server/common/persistence/versionhistory"
	"go.temporal.io/server/common/rpc"
	"go.temporal.io/server/service/history/configs"
)

const (
	resendContextTimeout = 30 * time.Second
)

type (
	// nDCHistoryReplicationFn provides the functionality to deliver replication raw history request to history
	// the provided func should be thread safe
	nDCHistoryReplicationFn func(
		ctx context.Context,
		sourceClusterName string,
		namespaceId namespace.ID,
		workflowId string,
		runId string,
		events [][]*historypb.HistoryEvent,
		versionHistory []*historyspb.VersionHistoryItem,
	) error
	// NDCHistoryResender is the interface for resending history events to remote
	NDCHistoryResender interface {
		// SendSingleWorkflowHistory sends multiple run IDs's history events to remote
		SendSingleWorkflowHistory(
			ctx context.Context,
			remoteClusterName string,
			namespaceID namespace.ID,
			workflowID string,
			runID string,
			startEventID int64,
			startEventVersion int64,
			endEventID int64,
			endEventVersion int64,
		) error
	}

	// NDCHistoryResenderImpl is the implementation of NDCHistoryResender
	NDCHistoryResenderImpl struct {
		namespaceRegistry    namespace.Registry
		clientBean           client.Bean
		historyReplicationFn nDCHistoryReplicationFn
		serializer           serialization.Serializer
		rereplicationTimeout dynamicconfig.DurationPropertyFnWithNamespaceIDFilter
		logger               log.Logger
		config               *configs.Config
	}

	historyBatch struct {
		versionHistory *historyspb.VersionHistory
		rawEventBatch  *commonpb.DataBlob
	}
)

const (
	defaultPageSize = int32(100)
)

// NewNDCHistoryResender create a new NDCHistoryResenderImpl
func NewNDCHistoryResender(
	namespaceRegistry namespace.Registry,
	clientBean client.Bean,
	historyReplicationFn nDCHistoryReplicationFn,
	serializer serialization.Serializer,
	rereplicationTimeout dynamicconfig.DurationPropertyFnWithNamespaceIDFilter,
	logger log.Logger,
	config *configs.Config,
) *NDCHistoryResenderImpl {

	return &NDCHistoryResenderImpl{
		namespaceRegistry:    namespaceRegistry,
		clientBean:           clientBean,
		historyReplicationFn: historyReplicationFn,
		serializer:           serializer,
		rereplicationTimeout: rereplicationTimeout,
		logger:               logger,
		config:               config,
	}
}

// SendSingleWorkflowHistory sends one run IDs's history events to remote
func (n *NDCHistoryResenderImpl) SendSingleWorkflowHistory(
	ctx context.Context,
	remoteClusterName string,
	namespaceID namespace.ID,
	workflowID string,
	runID string,
	startEventID int64,
	startEventVersion int64,
	endEventID int64,
	endEventVersion int64,
) error {

	resendCtx := context.Background()
	var cancel context.CancelFunc
	if n.rereplicationTimeout != nil {
		resendContextTimeout := n.rereplicationTimeout(namespaceID)
		if resendContextTimeout > 0 {
			resendCtx, cancel = context.WithTimeout(resendCtx, resendContextTimeout)
			defer cancel()
		}
	}
	resendCtx = rpc.CopyContextValues(resendCtx, ctx)

	historyIterator := collection.NewPagingIterator(n.getPaginationFn(
		resendCtx,
		remoteClusterName,
		namespaceID,
		workflowID,
		runID,
		startEventID,
		startEventVersion,
		endEventID,
		endEventVersion,
	))

	getMaxBatchCount := func() int {
		if n.config == nil {
			return 1
		}
		return n.config.ReplicationResendMaxBatchCount()
	}
	var eventsBatch [][]*historypb.HistoryEvent
	var versionHistory []*historyspb.VersionHistoryItem
	const EmptyVersion = int64(-1) // 0 is a valid event version when namespace is local
	eventsVersion := EmptyVersion
	applyFn := func() error {
		err := n.ApplyReplicateFn(
			ctx,
			remoteClusterName,
			namespaceID,
			workflowID,
			runID,
			eventsBatch,
			versionHistory,
		)
		if err != nil {
			n.logger.Error("failed to replicate events",
				tag.WorkflowNamespaceID(namespaceID.String()),
				tag.WorkflowID(workflowID),
				tag.WorkflowRunID(runID),
				tag.Error(err))
			return err
		}
		eventsBatch = nil
		versionHistory = nil
		eventsVersion = EmptyVersion
		return nil
	}
	for historyIterator.HasNext() {
		batch, err := historyIterator.Next()
		if err != nil {
			n.logger.Error("failed to get history events",
				tag.WorkflowNamespaceID(namespaceID.String()),
				tag.WorkflowID(workflowID),
				tag.WorkflowRunID(runID),
				tag.Error(err))
			return err
		}
		events, err := n.serializer.DeserializeEvents(batch.rawEventBatch)
		if err != nil {
			return err
		}
		if len(events) == 0 {
			continue
		}
		// check if version history changed during the batching process
		if len(eventsBatch) != 0 && len(versionHistory) != 0 {
			if !versionhistory.IsEqualVersionHistoryItems(versionHistory, batch.versionHistory.Items) ||
				(eventsVersion != EmptyVersion && eventsVersion != events[0].Version) {
				err := applyFn()
				if err != nil {
					return err
				}
			}
		}
		eventsBatch = append(eventsBatch, events)
		versionHistory = batch.versionHistory.Items
		eventsVersion = events[0].Version
		if len(eventsBatch) >= getMaxBatchCount() {
			err := applyFn()
			if err != nil {
				return err
			}
		}
	}
	if len(eventsBatch) > 0 {
		err := applyFn()
		if err != nil {
			return err
		}
	}
	return nil
}

func (n *NDCHistoryResenderImpl) getPaginationFn(
	ctx context.Context,
	remoteClusterName string,
	namespaceID namespace.ID,
	workflowID string,
	runID string,
	startEventID int64,
	startEventVersion int64,
	endEventID int64,
	endEventVersion int64,
) collection.PaginationFn[historyBatch] {

	return func(paginationToken []byte) ([]historyBatch, []byte, error) {

		response, err := n.getHistory(
			ctx,
			remoteClusterName,
			namespaceID,
			workflowID,
			runID,
			startEventID,
			startEventVersion,
			endEventID,
			endEventVersion,
			paginationToken,
			defaultPageSize,
		)
		if err != nil {
			return nil, nil, err
		}

		batches := make([]historyBatch, 0, len(response.GetHistoryBatches()))
		versionHistory := response.GetVersionHistory()
		for _, history := range response.GetHistoryBatches() {
			batch := historyBatch{
				versionHistory: versionHistory,
				rawEventBatch:  history,
			}
			batches = append(batches, batch)
		}
		return batches, response.NextPageToken, nil
	}
}

func (n *NDCHistoryResenderImpl) ApplyReplicateFn(
	ctx context.Context,
	sourceClusterName string,
	namespaceId namespace.ID,
	workflowId string,
	runId string,
	events [][]*historypb.HistoryEvent,
	versionHistory []*historyspb.VersionHistoryItem,
) error {
	ctx, cancel := context.WithTimeout(ctx, resendContextTimeout)
	defer cancel()

	return n.historyReplicationFn(ctx, sourceClusterName, namespaceId, workflowId, runId, events, versionHistory)
}

func (n *NDCHistoryResenderImpl) getHistory(
	ctx context.Context,
	remoteClusterName string,
	namespaceID namespace.ID,
	workflowID string,
	runID string,
	startEventID int64,
	startEventVersion int64,
	endEventID int64,
	endEventVersion int64,
	token []byte,
	pageSize int32,
) (*adminservice.GetWorkflowExecutionRawHistoryV2Response, error) {

	logger := log.With(n.logger, tag.WorkflowRunID(runID))

	ctx, cancel := rpc.NewContextFromParentWithTimeoutAndVersionHeaders(ctx, resendContextTimeout)
	defer cancel()

	adminClient, err := n.clientBean.GetRemoteAdminClient(remoteClusterName)
	if err != nil {
		return nil, err
	}

	response, err := adminClient.GetWorkflowExecutionRawHistoryV2(ctx, &adminservice.GetWorkflowExecutionRawHistoryV2Request{
		NamespaceId: namespaceID.String(),
		Execution: &commonpb.WorkflowExecution{
			WorkflowId: workflowID,
			RunId:      runID,
		},
		StartEventId:      startEventID,
		StartEventVersion: startEventVersion,
		EndEventId:        endEventID,
		EndEventVersion:   endEventVersion,
		MaximumPageSize:   pageSize,
		NextPageToken:     token,
	})
	if err != nil {
		logger.Error("error getting history", tag.Error(err))
		return nil, err
	}

	return response, nil
}
