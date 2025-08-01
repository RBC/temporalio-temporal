package workerdeployment

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/dgryski/go-farm"
	"github.com/pborman/uuid"
	commonpb "go.temporal.io/api/common/v1"
	deploymentpb "go.temporal.io/api/deployment/v1"
	enumspb "go.temporal.io/api/enums/v1"
	querypb "go.temporal.io/api/query/v1"
	"go.temporal.io/api/serviceerror"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	updatepb "go.temporal.io/api/update/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/temporal"
	deploymentspb "go.temporal.io/server/api/deployment/v1"
	"go.temporal.io/server/api/historyservice/v1"
	"go.temporal.io/server/api/matchingservice/v1"
	"go.temporal.io/server/common/backoff"
	"go.temporal.io/server/common/dynamicconfig"
	"go.temporal.io/server/common/log"
	"go.temporal.io/server/common/log/tag"
	"go.temporal.io/server/common/namespace"
	"go.temporal.io/server/common/payload"
	"go.temporal.io/server/common/persistence/visibility/manager"
	"go.temporal.io/server/common/primitives"
	"go.temporal.io/server/common/resource"
	"go.temporal.io/server/common/sdk"
	"go.temporal.io/server/common/searchattribute"
	"go.temporal.io/server/common/testing/testhooks"
	"go.temporal.io/server/common/worker_versioning"
	"go.temporal.io/server/service/history/consts"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Client interface {
	RegisterTaskQueueWorker(
		ctx context.Context,
		namespaceEntry *namespace.Namespace,
		deploymentName, buildId string,
		taskQueueName string,
		taskQueueType enumspb.TaskQueueType,
		identity string,
	) error

	DescribeVersion(
		ctx context.Context,
		namespaceEntry *namespace.Namespace,
		version string,
		reportTaskQueueStats bool,
	) (*deploymentpb.WorkerDeploymentVersionInfo, []*workflowservice.DescribeWorkerDeploymentVersionResponse_VersionTaskQueue, error)

	DescribeWorkerDeployment(
		ctx context.Context,
		namespaceEntry *namespace.Namespace,
		deploymentName string,
	) (*deploymentpb.WorkerDeploymentInfo, []byte, error)

	SetCurrentVersion(
		ctx context.Context,
		namespaceEntry *namespace.Namespace,
		deploymentName string,
		version string,
		identity string,
		ignoreMissingTaskQueues bool,
		conflictToken []byte,
	) (*deploymentspb.SetCurrentVersionResponse, error)

	ListWorkerDeployments(
		ctx context.Context,
		namespaceEntry *namespace.Namespace,
		pageSize int,
		nextPageToken []byte,
	) ([]*deploymentspb.WorkerDeploymentSummary, []byte, error)

	DeleteWorkerDeploymentVersion(
		ctx context.Context,
		namespaceEntry *namespace.Namespace,
		version string,
		skipDrainage bool,
		identity string,
	) error

	DeleteWorkerDeployment(
		ctx context.Context,
		namespaceEntry *namespace.Namespace,
		deploymentName string,
		identity string,
	) error

	SetRampingVersion(
		ctx context.Context,
		namespaceEntry *namespace.Namespace,
		deploymentName string,
		version string,
		percentage float32,
		identity string,
		ignoreMissingTaskQueues bool,
		conflictToken []byte,
	) (*deploymentspb.SetRampingVersionResponse, error)

	UpdateVersionMetadata(
		ctx context.Context,
		namespaceEntry *namespace.Namespace,
		version string,
		upsertEntries map[string]*commonpb.Payload,
		removeEntries []string,
		identity string,
	) (*deploymentpb.VersionMetadata, error)

	// Used internally by the Worker Deployment workflow in its StartWorkerDeployment Activity
	StartWorkerDeployment(
		ctx context.Context,
		namespaceEntry *namespace.Namespace,
		deploymentName string,
		identity string,
		requestID string,
	) error

	// Used internally by the Worker Deployment workflow in its SyncWorkerDeploymentVersion Activity
	SyncVersionWorkflowFromWorkerDeployment(
		ctx context.Context,
		namespaceEntry *namespace.Namespace,
		deploymentName, version string,
		args *deploymentspb.SyncVersionStateUpdateArgs,
		identity string,
		requestID string,
	) (*deploymentspb.SyncVersionStateResponse, error)

	// Used internally by the Worker Deployment workflow in its DeleteVersion Activity
	DeleteVersionFromWorkerDeployment(
		ctx context.Context,
		namespaceEntry *namespace.Namespace,
		deploymentName, version string,
		identity string,
		requestID string,
		skipDrainage bool,
	) error

	// Used internally by the Worker Deployment Version workflow in its AddVersionToWorkerDeployment Activity
	// to-be-deprecated
	AddVersionToWorkerDeployment(
		ctx context.Context,
		namespaceEntry *namespace.Namespace,
		deploymentName string,
		args *deploymentspb.AddVersionUpdateArgs,
		identity string,
		requestID string,
	) (*deploymentspb.AddVersionToWorkerDeploymentResponse, error)

	// Used internally by the Drainage workflow (child of Worker Deployment Version workflow)
	// in its GetVersionDrainageStatus Activity
	GetVersionDrainageStatus(
		ctx context.Context,
		namespaceEntry *namespace.Namespace,
		version string) (enumspb.VersionDrainageStatus, error)

	// Used internally by the Worker Deployment workflow in its IsVersionMissingTaskQueues Activity
	// to verify if there are missing task queues in the new current/ramping version.
	IsVersionMissingTaskQueues(
		ctx context.Context,
		namespaceEntry *namespace.Namespace,
		prevCurrentVersion, newVersion string,
	) (bool, error)

	// Used internally by the Worker Deployment workflow in its RegisterWorkerInVersion Activity
	// to register a task-queue worker in a version.
	RegisterWorkerInVersion(
		ctx context.Context,
		namespaceEntry *namespace.Namespace,
		args *deploymentspb.RegisterWorkerInVersionArgs,
		identity string,
	) error
}

type ErrMaxTaskQueuesInVersion struct{ error }
type ErrMaxVersionsInDeployment struct{ error }
type ErrMaxDeploymentsInNamespace struct{ error }
type ErrRegister struct{ error }

// ClientImpl implements Client
type ClientImpl struct {
	logger                           log.Logger
	historyClient                    historyservice.HistoryServiceClient
	visibilityManager                manager.VisibilityManager
	matchingClient                   resource.MatchingClient
	maxIDLengthLimit                 dynamicconfig.IntPropertyFn
	visibilityMaxPageSize            dynamicconfig.IntPropertyFnWithNamespaceFilter
	maxTaskQueuesInDeploymentVersion dynamicconfig.IntPropertyFnWithNamespaceFilter
	maxDeployments                   dynamicconfig.IntPropertyFnWithNamespaceFilter
	testHooks                        testhooks.TestHooks
}

var _ Client = (*ClientImpl)(nil)

var errRetry = errors.New("retry update")

func (d *ClientImpl) RegisterTaskQueueWorker(
	ctx context.Context,
	namespaceEntry *namespace.Namespace,
	deploymentName, buildId string,
	taskQueueName string,
	taskQueueType enumspb.TaskQueueType,
	identity string,
) (retErr error) {
	//revive:disable-next-line:defer
	defer d.record("RegisterTaskQueueWorker", &retErr, taskQueueName, taskQueueType, identity)()

	// Creating request ID out of build ID + TQ name + TQ type. Many updates may come from multiple
	// matching partitions, we do not want them to create new update requests.
	requestID := fmt.Sprintf("reg-ver-%v-%v-%d", farm.Fingerprint64([]byte(buildId)), farm.Fingerprint64([]byte(taskQueueName)), taskQueueType)

	updatePayload, err := sdk.PreferProtoDataConverter.ToPayloads(&deploymentspb.RegisterWorkerInWorkerDeploymentArgs{
		TaskQueueName: taskQueueName,
		TaskQueueType: taskQueueType,
		MaxTaskQueues: int32(d.maxTaskQueuesInDeploymentVersion(namespaceEntry.Name().String())),
		Version: &deploymentspb.WorkerDeploymentVersion{
			DeploymentName: deploymentName,
			BuildId:        buildId,
		},
	})
	if err != nil {
		return err
	}

	// starting and updating the deployment version workflow, which in turn starts a deployment workflow.
	outcome, err := d.updateWithStartWorkerDeployment(ctx, namespaceEntry, deploymentName, buildId, &updatepb.Request{
		Input: &updatepb.Input{Name: RegisterWorkerInWorkerDeployment, Args: updatePayload},
		Meta:  &updatepb.Meta{UpdateId: requestID, Identity: identity},
	}, identity, requestID, d.getSyncBatchSize())
	if err != nil {
		return err
	}

	if failure := outcome.GetFailure(); failure.GetApplicationFailureInfo().GetType() == errMaxTaskQueuesInVersionType {
		// translate to client-side error type
		return ErrMaxTaskQueuesInVersion{error: errors.New(failure.Message)}
	} else if failure.GetApplicationFailureInfo().GetType() == errTooManyVersions {
		return ErrMaxVersionsInDeployment{error: errors.New(failure.Message)}
	} else if failure.GetApplicationFailureInfo().GetType() == errNoChangeType {
		return nil
	} else if failure != nil {
		return ErrRegister{error: errors.New(failure.Message)}
	}

	return nil
}

func (d *ClientImpl) DescribeVersion(
	ctx context.Context,
	namespaceEntry *namespace.Namespace,
	version string,
	reportTaskQueueStats bool,
) (
	_ *deploymentpb.WorkerDeploymentVersionInfo,
	_ []*workflowservice.DescribeWorkerDeploymentVersionResponse_VersionTaskQueue,
	retErr error,
) {
	v, err := worker_versioning.WorkerDeploymentVersionFromStringV31(version)
	if err != nil {
		return nil, nil, serviceerror.NewInvalidArgumentf("invalid version string %q, expected format is \"<deployment_name>.<build_id>\"", version)
	}
	deploymentName := v.GetDeploymentName()
	buildID := v.GetBuildId()

	//revive:disable-next-line:defer
	defer d.record("DescribeVersion", &retErr, deploymentName, buildID)()

	// validate deployment name
	if err = validateVersionWfParams(WorkerDeploymentNameFieldName, deploymentName, d.maxIDLengthLimit()); err != nil {
		return nil, nil, err
	}

	// validate buildID
	if err = validateVersionWfParams(WorkerDeploymentBuildIDFieldName, buildID, d.maxIDLengthLimit()); err != nil {
		return nil, nil, err
	}

	workflowID := worker_versioning.GenerateVersionWorkflowID(deploymentName, buildID)

	req := &historyservice.QueryWorkflowRequest{
		NamespaceId: namespaceEntry.ID().String(),
		Request: &workflowservice.QueryWorkflowRequest{
			Namespace: namespaceEntry.Name().String(),
			Execution: &commonpb.WorkflowExecution{
				WorkflowId: workflowID,
			},
			Query:                &querypb.WorkflowQuery{QueryType: QueryDescribeVersion},
			QueryRejectCondition: enumspb.QUERY_REJECT_CONDITION_NOT_OPEN,
		},
	}

	res, err := d.historyClient.QueryWorkflow(ctx, req)
	if err != nil {
		var notFound *serviceerror.NotFound
		if errors.As(err, &notFound) {
			return nil, nil, serviceerror.NewNotFound("Worker Deployment Version not found")
		}
		return nil, nil, err
	}

	// on closed workflows, the response is empty.
	if res.GetResponse().GetQueryResult() == nil {
		return nil, nil, serviceerror.NewNotFound("Worker Deployment Version not found")
	}

	var queryResponse deploymentspb.QueryDescribeVersionResponse
	err = sdk.PreferProtoDataConverter.FromPayloads(res.GetResponse().GetQueryResult(), &queryResponse)
	if err != nil {
		return nil, nil, err
	}

	tqInfos, err := d.getTaskQueueDetails(ctx, namespaceEntry.ID(), queryResponse.VersionState, reportTaskQueueStats)
	if err != nil {
		return nil, nil, err
	}

	versionInfo := versionStateToVersionInfo(queryResponse.VersionState, tqInfos)
	return versionInfo, tqInfos, nil
}

func (d *ClientImpl) UpdateVersionMetadata(
	ctx context.Context,
	namespaceEntry *namespace.Namespace,
	version string,
	upsertEntries map[string]*commonpb.Payload,
	removeEntries []string,
	identity string,
) (_ *deploymentpb.VersionMetadata, retErr error) {
	//revive:disable-next-line:defer
	defer d.record("UpdateVersionMetadata", &retErr, namespaceEntry.Name(), version, upsertEntries, removeEntries, identity)()
	requestID := uuid.New()

	versionObj, err := worker_versioning.WorkerDeploymentVersionFromStringV31(version)
	if err != nil {
		return nil, serviceerror.NewInvalidArgument("invalid version string: " + err.Error())
	}

	updatePayload, err := sdk.PreferProtoDataConverter.ToPayloads(&deploymentspb.UpdateVersionMetadataArgs{
		UpsertEntries: upsertEntries,
		RemoveEntries: removeEntries,
		Identity:      identity,
	})
	if err != nil {
		return nil, err
	}

	workflowID := worker_versioning.GenerateVersionWorkflowID(versionObj.GetDeploymentName(), versionObj.GetBuildId())
	outcome, err := d.update(ctx, namespaceEntry, workflowID, &updatepb.Request{
		Input: &updatepb.Input{Name: UpdateVersionMetadata, Args: updatePayload},
		Meta:  &updatepb.Meta{UpdateId: requestID, Identity: identity},
	})
	if err != nil {
		return nil, err
	}

	if failure := outcome.GetFailure(); failure != nil {
		return nil, errors.New(failure.Message)
	}
	success := outcome.GetSuccess()
	if success == nil {
		return nil, serviceerror.NewInternal("outcome missing success and failure")
	}

	var res deploymentspb.UpdateVersionMetadataResponse
	if err := sdk.PreferProtoDataConverter.FromPayloads(outcome.GetSuccess(), &res); err != nil {
		return nil, err
	}

	return res.Metadata, nil
}

func (d *ClientImpl) DescribeWorkerDeployment(
	ctx context.Context,
	namespaceEntry *namespace.Namespace,
	deploymentName string,
) (_ *deploymentpb.WorkerDeploymentInfo, conflictToken []byte, retErr error) {
	//revive:disable-next-line:defer
	defer d.record("DescribeWorkerDeployment", &retErr, deploymentName)()

	// validating params
	err := validateVersionWfParams(WorkerDeploymentNameFieldName, deploymentName, d.maxIDLengthLimit())
	if err != nil {
		return nil, nil, err
	}

	deploymentWorkflowID := worker_versioning.GenerateDeploymentWorkflowID(deploymentName)

	req := &historyservice.QueryWorkflowRequest{
		NamespaceId: namespaceEntry.ID().String(),
		Request: &workflowservice.QueryWorkflowRequest{
			Namespace: namespaceEntry.Name().String(),
			Execution: &commonpb.WorkflowExecution{
				WorkflowId: deploymentWorkflowID,
			},
			Query:                &querypb.WorkflowQuery{QueryType: QueryDescribeDeployment},
			QueryRejectCondition: enumspb.QUERY_REJECT_CONDITION_NOT_OPEN,
		},
	}

	res, err := d.historyClient.QueryWorkflow(ctx, req)
	if err != nil {
		var notFound *serviceerror.NotFound
		if errors.As(err, &notFound) {
			return nil, nil, serviceerror.NewNotFound("Worker Deployment not found")
		}
		return nil, nil, err
	}

	if res.GetResponse().GetQueryResult() == nil {
		return nil, nil, serviceerror.NewNotFound("Worker Deployment not found")
	}

	var queryResponse deploymentspb.QueryDescribeWorkerDeploymentResponse
	err = sdk.PreferProtoDataConverter.FromPayloads(res.GetResponse().GetQueryResult(), &queryResponse)
	if err != nil {
		var notFound *serviceerror.NotFound
		if errors.As(err, &notFound) {
			return nil, nil, serviceerror.NewNotFound("Worker Deployment not found")
		}
		return nil, nil, err
	}

	dInfo, err := d.deploymentStateToDeploymentInfo(deploymentName, queryResponse.State)
	if err != nil {
		return nil, nil, err
	}
	return dInfo, queryResponse.GetState().GetConflictToken(), nil
}

func (d *ClientImpl) workerDeploymentExists(
	ctx context.Context,
	namespaceEntry *namespace.Namespace,
	deploymentName string,
) (bool, error) {
	deploymentWorkflowID := worker_versioning.GenerateDeploymentWorkflowID(deploymentName)

	res, err := d.historyClient.DescribeWorkflowExecution(ctx, &historyservice.DescribeWorkflowExecutionRequest{
		NamespaceId: namespaceEntry.ID().String(),
		Request: &workflowservice.DescribeWorkflowExecutionRequest{
			Namespace: namespaceEntry.Name().String(),
			Execution: &commonpb.WorkflowExecution{
				WorkflowId: deploymentWorkflowID,
			},
		},
	})
	if err != nil {
		var notFound *serviceerror.NotFound
		if errors.As(err, &notFound) {
			return false, nil
		}
		return false, err
	}

	// Deployment exists only if the entity wf is running
	return res.GetWorkflowExecutionInfo().GetStatus() == enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING, nil
}

func (d *ClientImpl) ListWorkerDeployments(
	ctx context.Context,
	namespaceEntry *namespace.Namespace,
	pageSize int,
	nextPageToken []byte,
) (_ []*deploymentspb.WorkerDeploymentSummary, _ []byte, retError error) {
	//revive:disable-next-line:defer
	defer d.record("ListWorkerDeployments", &retError)()

	query := WorkerDeploymentVisibilityBaseListQuery

	if pageSize == 0 {
		pageSize = d.visibilityMaxPageSize(namespaceEntry.Name().String())
	}

	persistenceResp, err := d.visibilityManager.ListWorkflowExecutions(
		ctx,
		&manager.ListWorkflowExecutionsRequestV2{
			NamespaceID:   namespaceEntry.ID(),
			Namespace:     namespaceEntry.Name(),
			PageSize:      pageSize,
			NextPageToken: nextPageToken,
			Query:         query,
		},
	)
	if err != nil {
		return nil, nil, err
	}

	workerDeploymentSummaries := make([]*deploymentspb.WorkerDeploymentSummary, len(persistenceResp.Executions))
	for i, ex := range persistenceResp.Executions {
		var workerDeploymentInfo *deploymentspb.WorkerDeploymentWorkflowMemo
		if ex.GetMemo() != nil {
			workerDeploymentInfo = DecodeWorkerDeploymentMemo(ex.GetMemo())
		} else {
			// There is a race condition where the Deployment workflow exists, but has not yet
			// upserted the memo. If that is the case, we handle it here.
			workerDeploymentInfo = &deploymentspb.WorkerDeploymentWorkflowMemo{
				DeploymentName: worker_versioning.GetDeploymentNameFromWorkflowID(ex.GetExecution().GetWorkflowId()),
				CreateTime:     ex.GetStartTime(),
				RoutingConfig:  &deploymentpb.RoutingConfig{CurrentVersion: worker_versioning.UnversionedVersionId},
			}
		}

		workerDeploymentSummaries[i] = &deploymentspb.WorkerDeploymentSummary{
			Name:                  workerDeploymentInfo.DeploymentName,
			CreateTime:            workerDeploymentInfo.CreateTime,
			RoutingConfig:         workerDeploymentInfo.RoutingConfig,
			LatestVersionSummary:  workerDeploymentInfo.LatestVersionSummary,
			RampingVersionSummary: workerDeploymentInfo.RampingVersionSummary,
			CurrentVersionSummary: workerDeploymentInfo.CurrentVersionSummary,
		}
	}

	return workerDeploymentSummaries, persistenceResp.NextPageToken, nil
}

func (d *ClientImpl) SetCurrentVersion(
	ctx context.Context,
	namespaceEntry *namespace.Namespace,
	deploymentName string,
	version string,
	identity string,
	ignoreMissingTaskQueues bool,
	conflictToken []byte,
) (_ *deploymentspb.SetCurrentVersionResponse, retErr error) {
	//revive:disable-next-line:defer
	defer d.record("SetCurrentVersion", &retErr, namespaceEntry.Name(), version, identity)()

	versionObj, err := worker_versioning.WorkerDeploymentVersionFromStringV31(version)
	if err != nil {
		return nil, serviceerror.NewInvalidArgument("invalid version string: " + err.Error())
	}
	if versionObj.GetDeploymentName() != "" && versionObj.GetDeploymentName() != deploymentName {
		return nil, serviceerror.NewInvalidArgumentf("invalid version string '%s' does not match deployment name '%s'", version, deploymentName)
	}

	err = validateVersionWfParams(WorkerDeploymentNameFieldName, deploymentName, d.maxIDLengthLimit())
	if err != nil {
		return nil, err
	}

	workflowID := worker_versioning.GenerateDeploymentWorkflowID(deploymentName)

	updatePayload, err := sdk.PreferProtoDataConverter.ToPayloads(&deploymentspb.SetCurrentVersionArgs{
		Identity:                identity,
		Version:                 version,
		IgnoreMissingTaskQueues: ignoreMissingTaskQueues,
		ConflictToken:           conflictToken,
	})
	if err != nil {
		return nil, err
	}

	// Generating a new updateID for each request. No-ops are handled by the worker-deployment workflow.
	updateID := uuid.New()

	outcome, err := d.update(
		ctx,
		namespaceEntry,
		workflowID,
		&updatepb.Request{
			Input: &updatepb.Input{Name: SetCurrentVersion, Args: updatePayload},
			Meta:  &updatepb.Meta{UpdateId: updateID, Identity: identity},
		},
	)
	if err != nil {
		return nil, err
	}

	var res deploymentspb.SetCurrentVersionResponse
	if failure := outcome.GetFailure(); failure.GetApplicationFailureInfo().GetType() == errNoChangeType {
		res.PreviousVersion = version
		// Returning the latest conflict token
		details := failure.GetApplicationFailureInfo().GetDetails().GetPayloads()
		if len(details) > 0 {
			res.ConflictToken = details[0].GetData()
		}
		return &res, nil
	} else if failure := outcome.GetFailure(); failure.GetApplicationFailureInfo().GetType() == errVersionNotFound {
		return nil, serviceerror.NewNotFound(errVersionNotFound)
	} else if failure.GetApplicationFailureInfo().GetType() == errFailedPrecondition {
		return nil, serviceerror.NewFailedPrecondition(failure.Message)
	} else if failure != nil {
		return nil, serviceerror.NewInternal(failure.Message)
	}

	success := outcome.GetSuccess()
	if success == nil {
		return nil, serviceerror.NewInternal("outcome missing success and failure")
	}

	if err := sdk.PreferProtoDataConverter.FromPayloads(success, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

func (d *ClientImpl) SetRampingVersion(
	ctx context.Context,
	namespaceEntry *namespace.Namespace,
	deploymentName string,
	version string,
	percentage float32,
	identity string,
	ignoreMissingTaskQueues bool,
	conflictToken []byte,
) (_ *deploymentspb.SetRampingVersionResponse, retErr error) {
	//revive:disable-next-line:defer
	defer d.record("SetRampingVersion", &retErr, namespaceEntry.Name(), version, percentage, identity)()

	var err error
	if version != "" {
		var versionObj *deploymentspb.WorkerDeploymentVersion
		versionObj, err = worker_versioning.WorkerDeploymentVersionFromStringV31(version)
		if err != nil {
			return nil, serviceerror.NewInvalidArgument("invalid version string: " + err.Error())
		}
		if versionObj.GetDeploymentName() != "" && versionObj.GetDeploymentName() != deploymentName {
			return nil, serviceerror.NewInvalidArgumentf("invalid version string '%s' does not match deployment name '%s'", version, deploymentName)
		}
	}

	err = validateVersionWfParams(WorkerDeploymentNameFieldName, deploymentName, d.maxIDLengthLimit())
	if err != nil {
		return nil, err
	}

	workflowID := worker_versioning.GenerateDeploymentWorkflowID(deploymentName)

	updatePayload, err := sdk.PreferProtoDataConverter.ToPayloads(&deploymentspb.SetRampingVersionArgs{
		Identity:                identity,
		Version:                 version,
		Percentage:              percentage,
		IgnoreMissingTaskQueues: ignoreMissingTaskQueues,
		ConflictToken:           conflictToken,
	})
	if err != nil {
		return nil, err
	}

	// Generating a new updateID for each request. No-ops are handled by the worker-deployment workflow.
	updateID := uuid.New()

	outcome, err := d.update(
		ctx,
		namespaceEntry,
		workflowID,
		&updatepb.Request{
			Input: &updatepb.Input{Name: SetRampingVersion, Args: updatePayload},
			Meta:  &updatepb.Meta{UpdateId: updateID, Identity: identity},
		},
	)
	if err != nil {
		return nil, err
	}

	var res deploymentspb.SetRampingVersionResponse
	if failure := outcome.GetFailure(); failure.GetApplicationFailureInfo().GetType() == errNoChangeType {
		res.PreviousVersion = version
		res.PreviousPercentage = percentage

		// Returning the latest conflict token
		details := failure.GetApplicationFailureInfo().GetDetails().GetPayloads()
		if len(details) > 0 {
			res.ConflictToken = details[0].GetData()
		}

		return &res, nil
	} else if failure := outcome.GetFailure(); failure.GetApplicationFailureInfo().GetType() == errVersionNotFound {
		return nil, serviceerror.NewNotFound(errVersionNotFound)
	} else if failure.GetApplicationFailureInfo().GetType() == errFailedPrecondition {
		return nil, serviceerror.NewFailedPrecondition(failure.Message)
	} else if failure != nil {
		return nil, serviceerror.NewInternal(failure.Message)
	}

	success := outcome.GetSuccess()
	if success == nil {
		return nil, serviceerror.NewInternal("outcome missing success and failure")
	}

	if err := sdk.PreferProtoDataConverter.FromPayloads(success, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

func (d *ClientImpl) DeleteWorkerDeploymentVersion(
	ctx context.Context,
	namespaceEntry *namespace.Namespace,
	version string,
	skipDrainage bool,
	identity string,
) (retErr error) {
	v, err := worker_versioning.WorkerDeploymentVersionFromStringV31(version)
	if err != nil {
		return serviceerror.NewInvalidArgumentf("invalid version string %q, expected format is \"<deployment_name>.<build_id>\"", version)
	}
	deploymentName := v.GetDeploymentName()
	buildId := v.GetBuildId()

	//revive:disable-next-line:defer
	defer d.record("DeleteWorkerDeploymentVersion", &retErr, namespaceEntry.Name(), deploymentName, buildId)()
	requestID := uuid.New()

	if identity == "" {
		identity = requestID
	}

	updatePayload, err := sdk.PreferProtoDataConverter.ToPayloads(&deploymentspb.DeleteVersionArgs{
		Identity: identity,
		Version: worker_versioning.WorkerDeploymentVersionToStringV31(&deploymentspb.WorkerDeploymentVersion{
			DeploymentName: deploymentName,
			BuildId:        buildId,
		}),
		SkipDrainage: skipDrainage,
	})
	if err != nil {
		return err
	}

	err = validateVersionWfParams(WorkerDeploymentNameFieldName, deploymentName, d.maxIDLengthLimit())
	if err != nil {
		return err
	}

	workflowID := worker_versioning.GenerateDeploymentWorkflowID(deploymentName)

	outcome, err := d.update(
		ctx,
		namespaceEntry,
		workflowID,
		&updatepb.Request{
			Input: &updatepb.Input{Name: DeleteVersion, Args: updatePayload},
			Meta:  &updatepb.Meta{UpdateId: requestID, Identity: identity},
		},
	)
	if err != nil {
		return err
	}

	if failure := outcome.GetFailure(); failure != nil {
		if failure.GetApplicationFailureInfo().GetType() == errVersionNotFound {
			return nil
		} else if failure.GetApplicationFailureInfo().GetType() == errFailedPrecondition {
			return serviceerror.NewFailedPrecondition(failure.GetMessage()) // non-retryable error to stop multiple activity attempts
		} else if failure.GetCause().GetApplicationFailureInfo().GetType() == errFailedPrecondition {
			return serviceerror.NewFailedPrecondition(failure.GetCause().GetMessage())
		}
		return serviceerror.NewInternal(failure.Message)
	}

	success := outcome.GetSuccess()
	if success == nil {
		return serviceerror.NewInternal("outcome missing success and failure")
	}
	return nil
}

func (d *ClientImpl) DeleteWorkerDeployment(
	ctx context.Context,
	namespaceEntry *namespace.Namespace,
	deploymentName string,
	identity string,
) (retErr error) {
	//revive:disable-next-line:defer
	defer d.record("DeleteWorkerDeployment", &retErr, namespaceEntry.Name(), deploymentName, identity)()

	// validating params
	err := validateVersionWfParams(WorkerDeploymentNameFieldName, deploymentName, d.maxIDLengthLimit())
	if err != nil {
		return err
	}

	requestID := uuid.New()
	updatePayload, err := sdk.PreferProtoDataConverter.ToPayloads(&deploymentspb.DeleteDeploymentArgs{
		Identity: identity,
	})
	if err != nil {
		return err
	}

	err = validateVersionWfParams(WorkerDeploymentNameFieldName, deploymentName, d.maxIDLengthLimit())
	if err != nil {
		return err
	}
	workflowID := worker_versioning.GenerateDeploymentWorkflowID(deploymentName)

	outcome, err := d.update(
		ctx,
		namespaceEntry,
		workflowID,
		&updatepb.Request{
			Input: &updatepb.Input{Name: DeleteDeployment, Args: updatePayload},
			Meta:  &updatepb.Meta{UpdateId: requestID, Identity: identity},
		},
	)
	if err != nil {
		var notFound *serviceerror.NotFound
		if errors.As(err, &notFound) {
			return nil
		}
		return err
	}

	if failure := outcome.GetFailure(); failure != nil {
		return serviceerror.NewInternal(failure.Message)
	}

	success := outcome.GetSuccess()
	if success == nil {
		return serviceerror.NewInternal("outcome missing success and failure")
	}

	return nil
}

func (d *ClientImpl) StartWorkerDeployment(
	ctx context.Context,
	namespaceEntry *namespace.Namespace,
	deploymentName string,
	identity string,
	requestID string,
) (retErr error) {
	//revive:disable-next-line:defer
	defer d.record("StartWorkerDeployment", &retErr, namespaceEntry.Name(), deploymentName, identity)()

	workflowID := worker_versioning.GenerateDeploymentWorkflowID(deploymentName)

	input, err := sdk.PreferProtoDataConverter.ToPayloads(&deploymentspb.WorkerDeploymentWorkflowArgs{
		NamespaceName:  namespaceEntry.Name().String(),
		NamespaceId:    namespaceEntry.ID().String(),
		DeploymentName: deploymentName,
	})
	if err != nil {
		return err
	}

	startReq := &workflowservice.StartWorkflowExecutionRequest{
		RequestId:                requestID,
		Namespace:                namespaceEntry.Name().String(),
		WorkflowId:               workflowID,
		WorkflowType:             &commonpb.WorkflowType{Name: WorkerDeploymentWorkflowType},
		TaskQueue:                &taskqueuepb.TaskQueue{Name: primitives.PerNSWorkerTaskQueue},
		Input:                    input,
		WorkflowIdReusePolicy:    enumspb.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
		WorkflowIdConflictPolicy: enumspb.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING,
		SearchAttributes:         d.buildSearchAttributes(),
	}

	historyStartReq := &historyservice.StartWorkflowExecutionRequest{
		NamespaceId:  namespaceEntry.ID().String(),
		StartRequest: startReq,
	}

	_, err = d.historyClient.StartWorkflowExecution(ctx, historyStartReq)
	return err
}

func (d *ClientImpl) SyncVersionWorkflowFromWorkerDeployment(
	ctx context.Context,
	namespaceEntry *namespace.Namespace,
	deploymentName, version string,
	args *deploymentspb.SyncVersionStateUpdateArgs,
	identity string,
	requestID string,
) (_ *deploymentspb.SyncVersionStateResponse, retErr error) {
	//revive:disable-next-line:defer
	defer d.record("SyncVersionWorkflowFromWorkerDeployment", &retErr, namespaceEntry.Name(), deploymentName, version, args, identity)()

	versionObj, err := worker_versioning.WorkerDeploymentVersionFromStringV31(version)
	if err != nil {
		return nil, serviceerror.NewInvalidArgument("invalid version string: " + err.Error())
	}

	updatePayload, err := sdk.PreferProtoDataConverter.ToPayloads(args)
	if err != nil {
		return nil, err
	}

	workflowID := worker_versioning.GenerateVersionWorkflowID(deploymentName, versionObj.GetBuildId())

	// updates an already existing deployment version workflow.
	outcome, err := d.update(
		ctx,
		namespaceEntry,
		workflowID,
		&updatepb.Request{
			Input: &updatepb.Input{Name: SyncVersionState, Args: updatePayload},
			Meta:  &updatepb.Meta{UpdateId: requestID, Identity: identity},
		},
	)
	if err != nil {
		return nil, err
	}

	if failure := outcome.GetFailure(); failure.GetApplicationFailureInfo().GetType() == errNoChangeType {
		// pretend this is a success
		outcome = &updatepb.Outcome{
			Value: &updatepb.Outcome_Success{
				Success: failure.GetApplicationFailureInfo().GetDetails(),
			},
		}
	} else if failure != nil {
		// TODO: is there an easy way to recover the original type here?
		return nil, serviceerror.NewInternal(failure.Message)
	}

	success := outcome.GetSuccess()
	if success == nil {
		return nil, serviceerror.NewInternal("outcome missing success and failure")
	}

	var res deploymentspb.SyncVersionStateResponse
	if err := sdk.PreferProtoDataConverter.FromPayloads(success, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

func (d *ClientImpl) DeleteVersionFromWorkerDeployment(
	ctx context.Context,
	namespaceEntry *namespace.Namespace,
	deploymentName, version string,
	identity string,
	requestID string,
	skipDrainage bool,
) (retErr error) {
	//revive:disable-next-line:defer
	defer d.record("DeleteVersionFromWorkerDeployment", &retErr, namespaceEntry.Name(), deploymentName, version, identity, skipDrainage)()

	versionObj, err := worker_versioning.WorkerDeploymentVersionFromStringV31(version)
	if err != nil {
		return err
	}

	workflowID := worker_versioning.GenerateVersionWorkflowID(deploymentName, versionObj.GetBuildId())
	updatePayload, err := sdk.PreferProtoDataConverter.ToPayloads(&deploymentspb.DeleteVersionArgs{
		Identity:     identity,
		Version:      version,
		SkipDrainage: skipDrainage,
	})
	if err != nil {
		return err
	}

	outcome, err := d.update(
		ctx,
		namespaceEntry,
		workflowID,
		&updatepb.Request{
			Input: &updatepb.Input{Name: DeleteVersion, Args: updatePayload},
			Meta:  &updatepb.Meta{UpdateId: requestID, Identity: identity},
		},
	)
	if err != nil {
		return err
	}

	if failure := outcome.GetFailure(); failure != nil {
		if failure.Message == ErrVersionIsDraining {
			return temporal.NewNonRetryableApplicationError(ErrVersionIsDraining, errFailedPrecondition, nil) // non-retryable error to stop multiple activity attempts
		} else if failure.Message == ErrVersionHasPollers {
			return temporal.NewNonRetryableApplicationError(ErrVersionHasPollers, errFailedPrecondition, nil) // non-retryable error to stop multiple activity attempts
		}
		return serviceerror.NewInternal(failure.Message)
	}

	success := outcome.GetSuccess()
	if success == nil {
		return serviceerror.NewInternal("outcome missing success and failure")
	}
	return nil
}

// update updates an already existing deployment version/deployment workflow.
func (d *ClientImpl) update(
	ctx context.Context,
	namespaceEntry *namespace.Namespace,
	workflowID string,
	updateRequest *updatepb.Request,
) (*updatepb.Outcome, error) {

	updateReq := &historyservice.UpdateWorkflowExecutionRequest{
		NamespaceId: namespaceEntry.ID().String(),
		Request: &workflowservice.UpdateWorkflowExecutionRequest{
			Namespace: namespaceEntry.Name().String(),
			WorkflowExecution: &commonpb.WorkflowExecution{
				WorkflowId: workflowID,
			},
			Request:    updateRequest,
			WaitPolicy: &updatepb.WaitPolicy{LifecycleStage: enumspb.UPDATE_WORKFLOW_EXECUTION_LIFECYCLE_STAGE_COMPLETED},
		},
	}

	policy := backoff.NewExponentialRetryPolicy(100 * time.Millisecond)
	isRetryable := func(err error) bool {
		// All updates that are admitted as the workflow is closing are considered retryable.
		return errors.Is(err, errRetry) || err.Error() == consts.ErrWorkflowClosing.Error()
	}

	var outcome *updatepb.Outcome
	err := backoff.ThrottleRetryContext(ctx, func(ctx context.Context) error {
		// historyClient retries internally on retryable rpc errors, we just have to retry on
		// successful but un-completed responses.
		res, err := d.historyClient.UpdateWorkflowExecution(ctx, updateReq)
		if err != nil {
			return err
		}

		if res.GetResponse() == nil {
			return serviceerror.NewInternal("failed to update workflow with workflowID: " + workflowID)
		}

		stage := res.GetResponse().GetStage()
		if stage != enumspb.UPDATE_WORKFLOW_EXECUTION_LIFECYCLE_STAGE_COMPLETED {
			// update not completed, try again
			return errRetry
		}

		outcome = res.GetResponse().GetOutcome()
		return nil
	}, policy, isRetryable)

	return outcome, err
}

func (d *ClientImpl) updateWithStartWorkerDeployment(
	ctx context.Context,
	namespaceEntry *namespace.Namespace,
	deploymentName, buildID string,
	updateRequest *updatepb.Request,
	identity string,
	requestID string,
	syncBatchSize int32,
) (*updatepb.Outcome, error) {
	err := validateVersionWfParams(WorkerDeploymentNameFieldName, deploymentName, d.maxIDLengthLimit())
	if err != nil {
		return nil, err
	}
	err = validateVersionWfParams(WorkerDeploymentBuildIDFieldName, buildID, d.maxIDLengthLimit())
	if err != nil {
		return nil, err
	}

	workflowID := worker_versioning.GenerateDeploymentWorkflowID(deploymentName)

	exists, err := d.workerDeploymentExists(ctx, namespaceEntry, deploymentName)
	if err != nil {
		return nil, err
	}
	if !exists {
		// New deployment, make sure we're not exceeding the limit
		count, err := d.countWorkerDeployments(ctx, namespaceEntry)
		if err != nil {
			return nil, err
		}
		limit := d.maxDeployments(namespaceEntry.Name().String())
		if count >= int64(limit) {
			return nil, ErrMaxDeploymentsInNamespace{error: errors.New(fmt.Sprintf("reached maximum deployments in namespace (%d)", limit))}
		}
	}

	input, err := sdk.PreferProtoDataConverter.ToPayloads(&deploymentspb.WorkerDeploymentWorkflowArgs{
		NamespaceName:  namespaceEntry.Name().String(),
		NamespaceId:    namespaceEntry.ID().String(),
		DeploymentName: deploymentName,
		State: &deploymentspb.WorkerDeploymentLocalState{
			SyncBatchSize: syncBatchSize,
		},
	})
	if err != nil {
		return nil, err
	}

	return d.updateWithStart(
		ctx,
		namespaceEntry,
		WorkerDeploymentWorkflowType,
		workflowID,
		nil,
		input,
		updateRequest,
		identity,
		requestID,
	)
}

func (d *ClientImpl) countWorkerDeployments(
	ctx context.Context,
	namespaceEntry *namespace.Namespace,
) (count int64, retError error) {
	query := WorkerDeploymentVisibilityBaseListQuery

	persistenceResp, err := d.visibilityManager.CountWorkflowExecutions(
		ctx,
		&manager.CountWorkflowExecutionsRequest{
			NamespaceID: namespaceEntry.ID(),
			Namespace:   namespaceEntry.Name(),
			Query:       query,
		},
	)
	if err != nil {
		return 0, err
	}
	return persistenceResp.Count, nil
}

func (d *ClientImpl) updateWithStartWorkerDeploymentVersion(
	ctx context.Context,
	namespaceEntry *namespace.Namespace,
	deploymentName, buildID string,
	updateRequest *updatepb.Request,
	identity string,
	requestID string,
) (*updatepb.Outcome, error) {
	err := validateVersionWfParams(WorkerDeploymentNameFieldName, deploymentName, d.maxIDLengthLimit())
	if err != nil {
		return nil, err
	}
	err = validateVersionWfParams(WorkerDeploymentBuildIDFieldName, buildID, d.maxIDLengthLimit())
	if err != nil {
		return nil, err
	}

	workflowID := worker_versioning.GenerateVersionWorkflowID(deploymentName, buildID)

	now := timestamppb.Now()
	input, err := sdk.PreferProtoDataConverter.ToPayloads(&deploymentspb.WorkerDeploymentVersionWorkflowArgs{
		NamespaceName: namespaceEntry.Name().String(),
		NamespaceId:   namespaceEntry.ID().String(),
		VersionState: &deploymentspb.VersionLocalState{
			Version: &deploymentspb.WorkerDeploymentVersion{
				DeploymentName: deploymentName,
				BuildId:        buildID,
			},
			CreateTime:        now,
			RoutingUpdateTime: nil,
			CurrentSinceTime:  nil,                                 // not current
			RampingSinceTime:  nil,                                 // not ramping
			RampPercentage:    0,                                   // not ramping
			DrainageInfo:      &deploymentpb.VersionDrainageInfo{}, // not draining or drained
			Metadata:          nil,                                 // todo
			SyncBatchSize:     d.getSyncBatchSize(),
		},
	})
	if err != nil {
		return nil, err
	}

	return d.updateWithStart(
		ctx,
		namespaceEntry,
		WorkerDeploymentVersionWorkflowType,
		workflowID,
		nil,
		input,
		updateRequest,
		identity,
		requestID,
	)
}

func (d *ClientImpl) AddVersionToWorkerDeployment(
	ctx context.Context,
	namespaceEntry *namespace.Namespace,
	deploymentName string,
	args *deploymentspb.AddVersionUpdateArgs,
	identity string,
	requestID string,
) (*deploymentspb.AddVersionToWorkerDeploymentResponse, error) {
	updatePayload, err := sdk.PreferProtoDataConverter.ToPayloads(args)
	if err != nil {
		return nil, err
	}

	updateRequest := &updatepb.Request{
		Input: &updatepb.Input{Name: AddVersionToWorkerDeployment, Args: updatePayload},
		Meta:  &updatepb.Meta{UpdateId: requestID, Identity: identity},
	}

	workflowID := worker_versioning.GenerateDeploymentWorkflowID(deploymentName)

	outcome, err := d.updateWithStart(
		ctx,
		namespaceEntry,
		WorkerDeploymentWorkflowType,
		workflowID,
		nil,
		nil,
		updateRequest,
		identity,
		requestID,
	)
	if err != nil {
		return nil, err
	}

	if failure := outcome.GetFailure(); failure.GetApplicationFailureInfo().GetType() == errVersionAlreadyExistsType {
		// pretend this is a success
		return &deploymentspb.AddVersionToWorkerDeploymentResponse{}, nil
	} else if failure := outcome.GetFailure(); failure.GetApplicationFailureInfo().GetType() == errTooManyVersions {
		return nil, serviceerror.NewFailedPrecondition(failure.Message)
	} else if failure != nil {
		return nil, serviceerror.NewInternalf("failed to add version %v to worker deployment %v with error %v", args.Version, deploymentName, failure.Message)
	}

	success := outcome.GetSuccess()
	if success == nil {
		return nil, serviceerror.NewInternalf("outcome missing success and failure while adding version %v to worker deployment %v", args.Version, deploymentName)
	}

	return &deploymentspb.AddVersionToWorkerDeploymentResponse{}, nil
}

func (d *ClientImpl) updateWithStart(
	ctx context.Context,
	namespaceEntry *namespace.Namespace,
	workflowType string,
	workflowID string,
	memo *commonpb.Memo,
	input *commonpb.Payloads,
	updateRequest *updatepb.Request,
	identity string,
	requestID string,
) (*updatepb.Outcome, error) {
	// Start workflow execution, if it hasn't already
	startReq := &workflowservice.StartWorkflowExecutionRequest{
		RequestId:                requestID,
		Namespace:                namespaceEntry.Name().String(),
		WorkflowId:               workflowID,
		WorkflowType:             &commonpb.WorkflowType{Name: workflowType},
		TaskQueue:                &taskqueuepb.TaskQueue{Name: primitives.PerNSWorkerTaskQueue},
		Input:                    input,
		WorkflowIdReusePolicy:    enumspb.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
		WorkflowIdConflictPolicy: enumspb.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING,
		SearchAttributes:         d.buildSearchAttributes(),
		Memo:                     memo,
		Identity:                 identity,
	}

	updateReq := &workflowservice.UpdateWorkflowExecutionRequest{
		Namespace: namespaceEntry.Name().String(),
		WorkflowExecution: &commonpb.WorkflowExecution{
			WorkflowId: workflowID,
		},
		Request:    updateRequest,
		WaitPolicy: &updatepb.WaitPolicy{LifecycleStage: enumspb.UPDATE_WORKFLOW_EXECUTION_LIFECYCLE_STAGE_COMPLETED},
	}

	// This is an atomic operation; if one operation fails, both will.
	multiOpReq := &historyservice.ExecuteMultiOperationRequest{
		NamespaceId: namespaceEntry.ID().String(),
		WorkflowId:  workflowID,
		Operations: []*historyservice.ExecuteMultiOperationRequest_Operation{
			{
				Operation: &historyservice.ExecuteMultiOperationRequest_Operation_StartWorkflow{
					StartWorkflow: &historyservice.StartWorkflowExecutionRequest{
						NamespaceId:  namespaceEntry.ID().String(),
						StartRequest: startReq,
					},
				},
			},
			{
				Operation: &historyservice.ExecuteMultiOperationRequest_Operation_UpdateWorkflow{
					UpdateWorkflow: &historyservice.UpdateWorkflowExecutionRequest{
						NamespaceId: namespaceEntry.ID().String(),
						Request:     updateReq,
					},
				},
			},
		},
	}

	policy := backoff.NewExponentialRetryPolicy(100 * time.Millisecond)
	isRetryable := func(err error) bool {
		// All updates that are admitted as the workflow is closing are considered retryable.
		return errors.Is(err, errRetry) || err.Error() == consts.ErrWorkflowClosing.Error()
	}
	var outcome *updatepb.Outcome

	err := backoff.ThrottleRetryContext(ctx, func(ctx context.Context) error {
		// historyClient retries internally on retryable rpc errors, we just have to retry on
		// successful but un-completed responses.
		res, err := d.historyClient.ExecuteMultiOperation(ctx, multiOpReq)
		if err != nil {
			return err
		}

		// we should get exactly one of each of these
		var startRes *historyservice.StartWorkflowExecutionResponse
		var updateRes *workflowservice.UpdateWorkflowExecutionResponse
		for _, response := range res.Responses {
			if sr := response.GetStartWorkflow(); sr != nil {
				startRes = sr
			} else if ur := response.GetUpdateWorkflow().GetResponse(); ur != nil {
				updateRes = ur
			}
		}
		if startRes == nil {
			return serviceerror.NewInternal("failed to start deployment workflow")
		} else if updateRes == nil {
			return serviceerror.NewInternal("failed to update deployment workflow")
		}

		if updateRes.Stage != enumspb.UPDATE_WORKFLOW_EXECUTION_LIFECYCLE_STAGE_COMPLETED {
			// update not completed, try again
			return errRetry
		}

		outcome = updateRes.GetOutcome()
		return nil
	}, policy, isRetryable)

	return outcome, err
}

func (d *ClientImpl) buildSearchAttributes() *commonpb.SearchAttributes {
	sa := &commonpb.SearchAttributes{}
	searchattribute.AddSearchAttribute(&sa, searchattribute.TemporalNamespaceDivision, payload.EncodeString(WorkerDeploymentNamespaceDivision))
	return sa
}

func (d *ClientImpl) record(operation string, retErr *error, args ...any) func() {
	start := time.Now()
	return func() {
		elapsed := time.Since(start)

		// TODO: add metrics recording here

		if *retErr != nil {
			if isFailedPrecondition(*retErr) {
				d.logger.Debug("deployment client failure due to a failed precondition",
					tag.Error(*retErr),
					tag.Operation(operation),
					tag.NewDurationTag("elapsed", elapsed),
					tag.NewAnyTag("args", args),
				)
			} else {
				d.logger.Error("deployment client error",
					tag.Error(*retErr),
					tag.Operation(operation),
					tag.NewDurationTag("elapsed", elapsed),
					tag.NewAnyTag("args", args),
				)
			}
		} else {
			d.logger.Debug("deployment client success",
				tag.Operation(operation),
				tag.NewDurationTag("elapsed", elapsed),
				tag.NewAnyTag("args", args),
			)
		}
	}
}

//nolint:staticcheck
func versionStateToVersionInfo(
	state *deploymentspb.VersionLocalState,
	taskQueueInfos []*workflowservice.DescribeWorkerDeploymentVersionResponse_VersionTaskQueue,
) *deploymentpb.WorkerDeploymentVersionInfo {
	if state == nil {
		return nil
	}

	infos := make([]*deploymentpb.WorkerDeploymentVersionInfo_VersionTaskQueueInfo, 0, len(taskQueueInfos))
	for _, taskQueueInfo := range taskQueueInfos {
		infos = append(infos, &deploymentpb.WorkerDeploymentVersionInfo_VersionTaskQueueInfo{
			Name: taskQueueInfo.Name,
			Type: taskQueueInfo.Type,
		})
	}

	// never return empty drainage info
	drainageInfo := state.GetDrainageInfo()
	if drainageInfo.GetStatus() == enumspb.VERSION_DRAINAGE_STATUS_UNSPECIFIED {
		drainageInfo = nil
	}

	return &deploymentpb.WorkerDeploymentVersionInfo{
		Version:            worker_versioning.WorkerDeploymentVersionToStringV31(state.Version),
		DeploymentVersion:  worker_versioning.ExternalWorkerDeploymentVersionFromVersion(state.Version),
		Status:             state.Status,
		CreateTime:         state.CreateTime,
		RoutingChangedTime: state.RoutingUpdateTime,
		CurrentSinceTime:   state.CurrentSinceTime,
		RampingSinceTime:   state.RampingSinceTime,
		RampPercentage:     state.RampPercentage,
		TaskQueueInfos:     infos,
		DrainageInfo:       drainageInfo,
		Metadata:           state.Metadata,
	}
}

func (d *ClientImpl) getTaskQueueDetails(
	ctx context.Context,
	namespaceID namespace.ID,
	state *deploymentspb.VersionLocalState,
	reportTaskQueueStats bool,
) ([]*workflowservice.DescribeWorkerDeploymentVersionResponse_VersionTaskQueue, error) {
	if state == nil {
		return nil, nil
	}

	tqOutputs := []*workflowservice.DescribeWorkerDeploymentVersionResponse_VersionTaskQueue{}
	for tqName, taskQueueFamilyInfo := range state.TaskQueueFamilies {
		for tqType := range taskQueueFamilyInfo.TaskQueues {
			tqOutputs = append(tqOutputs, &workflowservice.DescribeWorkerDeploymentVersionResponse_VersionTaskQueue{
				Name: tqName,
				Type: enumspb.TaskQueueType(tqType),
			})
		}
	}
	if len(tqOutputs) == 0 {
		return nil, nil
	}

	// Only query the matching service for task queue stats if requested since it's an expensive operation.
	if reportTaskQueueStats {
		tqInputs := []*matchingservice.DescribeVersionedTaskQueuesRequest_VersionTaskQueue{}
		for _, tq := range tqOutputs {
			vtq := &matchingservice.DescribeVersionedTaskQueuesRequest_VersionTaskQueue{
				Name: tq.Name,
				Type: tq.Type,
			}
			tqInputs = append(tqInputs, vtq)
		}

		// Sort the task queues by name and type to ensure that the task queue we query is deterministic.
		// This ensures we'll hit the cache on the same task queue partition.
		sort.Slice(tqInputs, func(i, j int) bool {
			if tqInputs[i].Name != tqInputs[j].Name {
				return tqInputs[i].Name < tqInputs[j].Name
			}
			return tqInputs[i].Type < tqInputs[j].Type
		})
		routeTQ := tqInputs[0]

		tqResp, err := d.matchingClient.DescribeVersionedTaskQueues(ctx,
			&matchingservice.DescribeVersionedTaskQueuesRequest{
				NamespaceId:       namespaceID.String(),
				TaskQueue:         &taskqueuepb.TaskQueue{Name: routeTQ.Name, Kind: enumspb.TASK_QUEUE_KIND_NORMAL},
				TaskQueueType:     routeTQ.Type,
				Version:           state.Version,
				VersionTaskQueues: tqInputs,
			})
		if err != nil {
			return nil, err
		}

		tqKey := func(tqName string, tqType enumspb.TaskQueueType) string { return fmt.Sprintf("%s-%d", tqName, tqType) }
		tqRespMap := make(map[string]*matchingservice.DescribeVersionedTaskQueuesResponse_VersionTaskQueue)
		for _, tq := range tqResp.GetVersionTaskQueues() {
			tqRespMap[tqKey(tq.Name, tq.Type)] = tq
		}

		// Update stats for existing entries
		for i, tq := range tqOutputs {
			if tqRespTQ, ok := tqRespMap[tqKey(tq.Name, tq.Type)]; ok {
				tqOutputs[i].Stats = tqRespTQ.Stats
				tqOutputs[i].StatsByPriorityKey = tqRespTQ.StatsByPriorityKey
				continue
			}
			// This *should* never happen, but in case it does, we should error instead of returning partial results.
			return nil, serviceerror.NewNotFoundf("task queue %s of type %s not found in this version", tq.Name, tq.Type)
		}
	}

	return tqOutputs, nil
}

func (d *ClientImpl) deploymentStateToDeploymentInfo(deploymentName string, state *deploymentspb.WorkerDeploymentLocalState) (*deploymentpb.WorkerDeploymentInfo, error) {
	if state == nil {
		return nil, nil
	}

	var workerDeploymentInfo deploymentpb.WorkerDeploymentInfo

	workerDeploymentInfo.Name = deploymentName
	workerDeploymentInfo.CreateTime = state.CreateTime
	workerDeploymentInfo.RoutingConfig = state.RoutingConfig
	workerDeploymentInfo.LastModifierIdentity = state.LastModifierIdentity

	for _, v := range state.Versions {
		workerDeploymentInfo.VersionSummaries = append(workerDeploymentInfo.VersionSummaries, &deploymentpb.WorkerDeploymentInfo_WorkerDeploymentVersionSummary{
			Version:              v.GetVersion(),
			DeploymentVersion:    worker_versioning.ExternalWorkerDeploymentVersionFromStringV31(v.Version),
			CreateTime:           v.GetCreateTime(),
			DrainageStatus:       v.GetDrainageInfo().GetStatus(), // deprecated.
			DrainageInfo:         v.GetDrainageInfo(),
			RoutingUpdateTime:    v.GetRoutingUpdateTime(),
			CurrentSinceTime:     v.GetCurrentSinceTime(),
			RampingSinceTime:     v.GetRampingSinceTime(),
			FirstActivationTime:  v.GetFirstActivationTime(),
			LastDeactivationTime: v.GetLastDeactivationTime(),
			Status:               v.GetStatus(),
		})
	}

	// Sort by create time, with the latest version first.
	sort.Slice(workerDeploymentInfo.VersionSummaries, func(i, j int) bool {
		return workerDeploymentInfo.VersionSummaries[i].CreateTime.AsTime().After(workerDeploymentInfo.VersionSummaries[j].CreateTime.AsTime())
	})

	return &workerDeploymentInfo, nil
}

func (d *ClientImpl) GetVersionDrainageStatus(
	ctx context.Context,
	namespaceEntry *namespace.Namespace,
	version string) (enumspb.VersionDrainageStatus, error) {
	countRequest := manager.CountWorkflowExecutionsRequest{
		NamespaceID: namespaceEntry.ID(),
		Namespace:   namespaceEntry.Name(),
		Query:       makeDeploymentQuery(worker_versioning.ExternalWorkerDeploymentVersionToString(worker_versioning.ExternalWorkerDeploymentVersionFromStringV31(version))),
	}
	countResponse, err := d.visibilityManager.CountWorkflowExecutions(ctx, &countRequest)
	if err != nil {
		return enumspb.VERSION_DRAINAGE_STATUS_UNSPECIFIED, err
	}
	if countResponse.Count == 0 {
		return enumspb.VERSION_DRAINAGE_STATUS_DRAINED, nil
	}
	return enumspb.VERSION_DRAINAGE_STATUS_DRAINING, nil
}

func makeDeploymentQuery(version string) string {
	var statusFilter string
	deploymentFilter := fmt.Sprintf("= '%s'", worker_versioning.PinnedBuildIdSearchAttribute(version))
	statusFilter = "= 'Running'"
	return fmt.Sprintf("%s %s AND %s %s", searchattribute.BuildIds, deploymentFilter, searchattribute.ExecutionStatus, statusFilter)
}

func (d *ClientImpl) IsVersionMissingTaskQueues(ctx context.Context, namespaceEntry *namespace.Namespace, prevCurrentVersion, newVersion string) (bool, error) {
	// Check if all the task-queues in the prevCurrentVersion are present in the newCurrentVersion (newVersion is either the new ramping version or the new current version)
	prevCurrentVersionInfo, _, err := d.DescribeVersion(ctx, namespaceEntry, prevCurrentVersion, false)
	if err != nil {
		return false, serviceerror.NewFailedPreconditionf("Version %s not found in deployment with error: %v", prevCurrentVersion, err)
	}

	newVersionInfo, _, err := d.DescribeVersion(ctx, namespaceEntry, newVersion, false)
	if err != nil {
		return false, serviceerror.NewFailedPreconditionf("Version %s not found in deployment with error: %v", newVersion, err)
	}

	missingTaskQueues, err := d.checkForMissingTaskQueues(prevCurrentVersionInfo, newVersionInfo)
	if err != nil {
		return false, err
	}

	if len(missingTaskQueues) == 0 {
		return false, nil
	}

	// Verify that all the missing task-queues have been added to another deployment or do not have backlogged tasks/add-rate > 0
	for _, missingTaskQueue := range missingTaskQueues {
		isExpectedInNewVersion, err := d.isTaskQueueExpectedInNewVersion(ctx, namespaceEntry, missingTaskQueue, prevCurrentVersionInfo)
		if err != nil {
			return false, err
		}
		if isExpectedInNewVersion {
			// one of the missing task queues is expected in the new version
			return true, nil
		}
	}

	// all expected task queues are present in the new version
	return false, nil
}

// isTaskQueueExpectedInNewVersion checks if a task queue is expected in the new version. A task queue is expected in the new version if:
// 1. It is not assigned to a deployment different from the deployment's current version.
// 2. It has backlogged tasks or add-rate > 0.
func (d *ClientImpl) isTaskQueueExpectedInNewVersion(
	ctx context.Context,
	namespaceEntry *namespace.Namespace,
	taskQueue *deploymentpb.WorkerDeploymentVersionInfo_VersionTaskQueueInfo,
	prevCurrentVersionInfo *deploymentpb.WorkerDeploymentVersionInfo,
) (bool, error) {
	// First check if task queue is assigned to another deployment
	response, err := d.matchingClient.DescribeTaskQueue(ctx, &matchingservice.DescribeTaskQueueRequest{
		NamespaceId: namespaceEntry.ID().String(),
		DescRequest: &workflowservice.DescribeTaskQueueRequest{
			TaskQueue: &taskqueuepb.TaskQueue{
				Name: taskQueue.Name,
				Kind: enumspb.TASK_QUEUE_KIND_NORMAL,
			},
			TaskQueueType: taskQueue.Type,
		},
	})
	if err != nil {
		return false, err
	}

	// Task Queue has been moved to another Worker Deployment
	if response.DescResponse.VersioningInfo != nil &&
		response.DescResponse.VersioningInfo.GetCurrentVersion() != prevCurrentVersionInfo.GetVersion() {
		return false, nil
	}

	versionStr := worker_versioning.ExternalWorkerDeploymentVersionToString(prevCurrentVersionInfo.GetDeploymentVersion())

	// Check if task queue has backlogged tasks or add-rate > 0
	req := &matchingservice.DescribeTaskQueueRequest{
		NamespaceId: namespaceEntry.ID().String(),
		DescRequest: &workflowservice.DescribeTaskQueueRequest{
			ApiMode: enumspb.DESCRIBE_TASK_QUEUE_MODE_ENHANCED,
			TaskQueue: &taskqueuepb.TaskQueue{
				Name: taskQueue.Name,
				Kind: enumspb.TASK_QUEUE_KIND_NORMAL,
			},
			TaskQueueTypes: []enumspb.TaskQueueType{taskQueue.Type},
			Versions: &taskqueuepb.TaskQueueVersionSelection{
				BuildIds: []string{versionStr}, // pretending the version string is a build id
			},
			// Since request doesn't pass through frontend, this field is not automatically populated.
			// Moreover, DescribeTaskQueueEnhanced requires this field to be set to WORKFLOW type.
			TaskQueueType: enumspb.TASK_QUEUE_TYPE_WORKFLOW,
			ReportStats:   true,
		},
	}
	response, err = d.matchingClient.DescribeTaskQueue(ctx, req)
	if err != nil {
		d.logger.Error("error fetching AddRate for task-queue", tag.Error(err))
		return false, err
	}

	typesInfo := response.GetDescResponse().GetVersionsInfo()[versionStr].GetTypesInfo()
	if typesInfo != nil {
		typeStats := typesInfo[int32(taskQueue.Type)]
		if typeStats != nil && typeStats.GetStats() != nil &&
			(typeStats.GetStats().GetTasksAddRate() != 0 || typeStats.GetStats().GetApproximateBacklogCount() != 0) {
			return true, nil
		}
	}

	return false, nil
}

// checkForMissingTaskQueues checks if all the task-queues in the previous version are present in the new version
func (d *ClientImpl) checkForMissingTaskQueues(prevCurrentVersionInfo, newCurrentVersionInfo *deploymentpb.WorkerDeploymentVersionInfo) ([]*deploymentpb.WorkerDeploymentVersionInfo_VersionTaskQueueInfo, error) {
	prevCurrentVersionTaskQueues := prevCurrentVersionInfo.GetTaskQueueInfos()
	newCurrentVersionTaskQueues := newCurrentVersionInfo.GetTaskQueueInfos()

	missingTaskQueues := []*deploymentpb.WorkerDeploymentVersionInfo_VersionTaskQueueInfo{}
	for _, prevTaskQueue := range prevCurrentVersionTaskQueues {
		found := false
		for _, newTaskQueue := range newCurrentVersionTaskQueues {
			if prevTaskQueue.GetName() == newTaskQueue.GetName() && prevTaskQueue.GetType() == newTaskQueue.GetType() {
				found = true
				break
			}
		}
		if !found {
			missingTaskQueues = append(missingTaskQueues, prevTaskQueue)
		}
	}

	return missingTaskQueues, nil
}

func (d *ClientImpl) RegisterWorkerInVersion(
	ctx context.Context,
	namespaceEntry *namespace.Namespace,
	args *deploymentspb.RegisterWorkerInVersionArgs,
	identity string,
) error {
	versionObj, err := worker_versioning.WorkerDeploymentVersionFromStringV31(args.Version)
	if err != nil {
		return serviceerror.NewInvalidArgument("invalid version string: " + err.Error())
	}

	updatePayload, err := sdk.PreferProtoDataConverter.ToPayloads(&deploymentspb.RegisterWorkerInVersionArgs{
		TaskQueueName: args.TaskQueueName,
		TaskQueueType: args.TaskQueueType,
		MaxTaskQueues: args.MaxTaskQueues,
	})
	if err != nil {
		return err
	}

	requestID := uuid.New()
	outcome, err := d.updateWithStartWorkerDeploymentVersion(ctx, namespaceEntry, versionObj.DeploymentName, versionObj.BuildId, &updatepb.Request{
		Input: &updatepb.Input{Name: RegisterWorkerInDeploymentVersion, Args: updatePayload},
		Meta:  &updatepb.Meta{UpdateId: requestID, Identity: identity},
	}, identity, requestID)
	if err != nil {
		return err
	}

	if failure := outcome.GetFailure(); failure.GetApplicationFailureInfo().GetType() == errMaxTaskQueuesInVersionType {
		// translate to a non-retryable error
		return temporal.NewNonRetryableApplicationError(failure.Message, errMaxTaskQueuesInVersionType, serviceerror.NewFailedPrecondition(failure.Message))
	} else if failure.GetApplicationFailureInfo().GetType() == errNoChangeType {
		return nil
	} else if failure != nil {
		return ErrRegister{error: errors.New(failure.Message)}
	}

	return nil
}

func (d *ClientImpl) getSyncBatchSize() int32 {
	syncBatchSize := int32(25)
	if n, ok := testhooks.Get[int](d.testHooks, testhooks.TaskQueuesInDeploymentSyncBatchSize); ok && n > 0 {
		// In production, the testhook would be set to 0 and never reach here!
		syncBatchSize = int32(n)
	}
	return syncBatchSize
}
