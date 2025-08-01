package workflow

import (
	"context"
	"fmt"
	"math/rand"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/uber-go/tally/v4"
	commandpb "go.temporal.io/api/command/v1"
	commonpb "go.temporal.io/api/common/v1"
	deploymentpb "go.temporal.io/api/deployment/v1"
	enumspb "go.temporal.io/api/enums/v1"
	failurepb "go.temporal.io/api/failure/v1"
	historypb "go.temporal.io/api/history/v1"
	"go.temporal.io/api/serviceerror"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	updatepb "go.temporal.io/api/update/v1"
	workflowpb "go.temporal.io/api/workflow/v1"
	"go.temporal.io/api/workflowservice/v1"
	clockspb "go.temporal.io/server/api/clock/v1"
	enumsspb "go.temporal.io/server/api/enums/v1"
	historyspb "go.temporal.io/server/api/history/v1"
	"go.temporal.io/server/api/historyservice/v1"
	persistencespb "go.temporal.io/server/api/persistence/v1"
	taskqueuespb "go.temporal.io/server/api/taskqueue/v1"
	"go.temporal.io/server/chasm"
	chasmworkflow "go.temporal.io/server/chasm/lib/workflow"
	"go.temporal.io/server/common"
	"go.temporal.io/server/common/cluster"
	"go.temporal.io/server/common/definition"
	"go.temporal.io/server/common/dynamicconfig"
	"go.temporal.io/server/common/failure"
	"go.temporal.io/server/common/log"
	"go.temporal.io/server/common/namespace"
	"go.temporal.io/server/common/payloads"
	"go.temporal.io/server/common/persistence/transitionhistory"
	"go.temporal.io/server/common/persistence/versionhistory"
	"go.temporal.io/server/common/primitives/timestamp"
	"go.temporal.io/server/common/searchattribute"
	serviceerror2 "go.temporal.io/server/common/serviceerror"
	"go.temporal.io/server/common/testing/protorequire"
	"go.temporal.io/server/common/testing/testvars"
	"go.temporal.io/server/common/tqid"
	"go.temporal.io/server/common/worker_versioning"
	"go.temporal.io/server/components/callbacks"
	"go.temporal.io/server/service/history/configs"
	"go.temporal.io/server/service/history/events"
	"go.temporal.io/server/service/history/historybuilder"
	"go.temporal.io/server/service/history/hsm"
	"go.temporal.io/server/service/history/hsm/hsmtest"
	historyi "go.temporal.io/server/service/history/interfaces"
	"go.temporal.io/server/service/history/shard"
	"go.temporal.io/server/service/history/tasks"
	"go.temporal.io/server/service/history/tests"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type (
	mutableStateSuite struct {
		suite.Suite
		*require.Assertions

		controller      *gomock.Controller
		mockConfig      *configs.Config
		mockShard       *shard.ContextTest
		mockEventsCache *events.MockCache

		namespaceEntry *namespace.Namespace
		mutableState   *MutableStateImpl
		logger         log.Logger
		testScope      tally.TestScope

		replicationMultipleBatches bool
	}
)

var (
	testPayload = &commonpb.Payload{
		Metadata: map[string][]byte{
			"random metadata key": []byte("random metadata value"),
		},
		Data: []byte("random data"),
	}
	testPayloads                 = &commonpb.Payloads{Payloads: []*commonpb.Payload{testPayload}}
	workflowTaskCompletionLimits = historyi.WorkflowTaskCompletionLimits{
		MaxResetPoints:              10,
		MaxSearchAttributeValueSize: 1024,
	}
	deployment1 = &deploymentpb.Deployment{
		SeriesName: "my_app",
		BuildId:    "build_1",
	}
	deployment2 = &deploymentpb.Deployment{
		SeriesName: "my_app",
		BuildId:    "build_2",
	}
	deployment3 = &deploymentpb.Deployment{
		SeriesName: "my_app",
		BuildId:    "build_3",
	}
	versionOverrideMask = &fieldmaskpb.FieldMask{Paths: []string{"versioning_behavior_override"}}
	pinnedOptions1      = &workflowpb.WorkflowExecutionOptions{
		VersioningOverride: &workflowpb.VersioningOverride{
			Behavior:      enumspb.VERSIONING_BEHAVIOR_PINNED,
			PinnedVersion: worker_versioning.WorkerDeploymentVersionToStringV31(worker_versioning.DeploymentVersionFromDeployment(deployment1)),
		},
	}
	pinnedOptions2 = &workflowpb.WorkflowExecutionOptions{
		VersioningOverride: &workflowpb.VersioningOverride{
			Behavior:      enumspb.VERSIONING_BEHAVIOR_PINNED,
			PinnedVersion: worker_versioning.WorkerDeploymentVersionToStringV31(worker_versioning.DeploymentVersionFromDeployment(deployment2)),
		},
	}
	pinnedOptions3 = &workflowpb.WorkflowExecutionOptions{
		VersioningOverride: &workflowpb.VersioningOverride{
			Behavior:      enumspb.VERSIONING_BEHAVIOR_PINNED,
			PinnedVersion: worker_versioning.WorkerDeploymentVersionToStringV31(worker_versioning.DeploymentVersionFromDeployment(deployment3)),
		},
	}
	unpinnedOptions = &workflowpb.WorkflowExecutionOptions{
		VersioningOverride: &workflowpb.VersioningOverride{
			Behavior: enumspb.VERSIONING_BEHAVIOR_AUTO_UPGRADE,
		},
	}
	emptyOptions = &workflowpb.WorkflowExecutionOptions{}
)

func TestMutableStateSuite(t *testing.T) {
	for _, tc := range []struct {
		name                       string
		replicationMultipleBatches bool
	}{
		{
			name:                       "ReplicationMultipleBatchesEnabled",
			replicationMultipleBatches: true,
		},
		{
			name:                       "ReplicationMultipleBatchesDisabled",
			replicationMultipleBatches: false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := &mutableStateSuite{
				replicationMultipleBatches: tc.replicationMultipleBatches,
			}
			suite.Run(t, s)
		})
	}
}

func (s *mutableStateSuite) SetupSuite() {

}

func (s *mutableStateSuite) TearDownSuite() {

}

func (s *mutableStateSuite) Name() string {
	return "MutableStateSuite"
}

func (s *mutableStateSuite) SetupTest() {
	s.Assertions = require.New(s.T())

	s.controller = gomock.NewController(s.T())
	s.mockEventsCache = events.NewMockCache(s.controller)

	s.mockConfig = tests.NewDynamicConfig()
	s.mockConfig.ReplicationMultipleBatches = dynamicconfig.GetBoolPropertyFn(s.replicationMultipleBatches)
	s.mockShard = shard.NewTestContext(
		s.controller,
		&persistencespb.ShardInfo{
			ShardId: 0,
			RangeId: 1,
		},
		s.mockConfig,
	)
	reg := hsm.NewRegistry()
	s.Require().NoError(RegisterStateMachine(reg))
	s.Require().NoError(callbacks.RegisterStateMachine(reg))
	s.mockShard.SetStateMachineRegistry(reg)

	s.mockConfig.MutableStateActivityFailureSizeLimitWarn = func(namespace string) int { return 1 * 1024 }
	s.mockConfig.MutableStateActivityFailureSizeLimitError = func(namespace string) int { return 2 * 1024 }
	s.mockConfig.EnableTransitionHistory = func() bool { return true }
	s.mockShard.SetEventsCacheForTesting(s.mockEventsCache)

	s.namespaceEntry = tests.GlobalNamespaceEntry
	s.mockShard.Resource.NamespaceCache.EXPECT().GetNamespaceByID(tests.NamespaceID).Return(s.namespaceEntry, nil).AnyTimes()
	s.mockShard.Resource.ClusterMetadata.EXPECT().ClusterNameForFailoverVersion(s.namespaceEntry.IsGlobalNamespace(), s.namespaceEntry.FailoverVersion()).Return(cluster.TestCurrentClusterName).AnyTimes()
	s.mockShard.Resource.ClusterMetadata.EXPECT().GetCurrentClusterName().Return(cluster.TestCurrentClusterName).AnyTimes()
	s.mockShard.Resource.ClusterMetadata.EXPECT().GetClusterID().Return(int64(1)).AnyTimes()
	s.testScope = s.mockShard.Resource.MetricsScope.(tally.TestScope)
	s.logger = s.mockShard.GetLogger()

	s.mutableState = NewMutableState(s.mockShard, s.mockEventsCache, s.logger, s.namespaceEntry, tests.WorkflowID, tests.RunID, time.Now().UTC())
}

func (s *mutableStateSuite) TearDownTest() {
	s.controller.Finish()
	s.mockShard.StopForTest()
}

func (s *mutableStateSuite) SetupSubTest() {
	// create a fresh mutable state for each sub test
	// this will be invoked upon s.Run()
	s.mutableState = NewMutableState(s.mockShard, s.mockEventsCache, s.logger, s.mutableState.GetNamespaceEntry(), tests.WorkflowID, tests.RunID, time.Now().UTC())
}

func (s *mutableStateSuite) TestTransientWorkflowTaskCompletionFirstBatchApplied_ApplyWorkflowTaskCompleted() {
	version := int64(12)
	workflowID := "some random workflow ID"
	runID := uuid.New()
	s.mutableState = TestGlobalMutableState(
		s.mockShard,
		s.mockEventsCache,
		s.logger,
		version,
		workflowID,
		runID,
	)

	newWorkflowTaskScheduleEvent, newWorkflowTaskStartedEvent := s.prepareTransientWorkflowTaskCompletionFirstBatchApplied(version, workflowID, runID)

	newWorkflowTaskCompletedEvent := &historypb.HistoryEvent{
		Version:   version,
		EventId:   newWorkflowTaskStartedEvent.GetEventId() + 1,
		EventTime: timestamppb.New(time.Now().UTC()),
		EventType: enumspb.EVENT_TYPE_WORKFLOW_TASK_COMPLETED,
		Attributes: &historypb.HistoryEvent_WorkflowTaskCompletedEventAttributes{WorkflowTaskCompletedEventAttributes: &historypb.WorkflowTaskCompletedEventAttributes{
			ScheduledEventId: newWorkflowTaskScheduleEvent.GetEventId(),
			StartedEventId:   newWorkflowTaskStartedEvent.GetEventId(),
			Identity:         "some random identity",
		}},
	}
	s.mutableState.SetHistoryBuilder(historybuilder.NewImmutable([]*historypb.HistoryEvent{
		newWorkflowTaskCompletedEvent,
	}))
	err := s.mutableState.ApplyWorkflowTaskCompletedEvent(newWorkflowTaskCompletedEvent)
	s.NoError(err)
	s.Equal(0, s.mutableState.hBuilder.NumBufferedEvents())
}

func (s *mutableStateSuite) TestTransientWorkflowTaskCompletionFirstBatchApplied_FailoverWorkflowTaskTimeout() {
	version := int64(12)
	workflowID := "some random workflow ID"
	runID := uuid.New()
	s.mutableState = TestGlobalMutableState(
		s.mockShard,
		s.mockEventsCache,
		s.logger,
		version,
		workflowID,
		runID,
	)

	newWorkflowTaskScheduleEvent, _ := s.prepareTransientWorkflowTaskCompletionFirstBatchApplied(version, workflowID, runID)

	newWorkflowTask := s.mutableState.GetWorkflowTaskByID(newWorkflowTaskScheduleEvent.GetEventId())
	s.NotNil(newWorkflowTask)

	_, err := s.mutableState.AddWorkflowTaskTimedOutEvent(
		newWorkflowTask,
	)
	s.NoError(err)
	s.Equal(0, s.mutableState.hBuilder.NumBufferedEvents())
}

func (s *mutableStateSuite) TestTransientWorkflowTaskCompletionFirstBatchApplied_FailoverWorkflowTaskFailed() {
	version := int64(12)
	workflowID := "some random workflow ID"
	runID := uuid.New()
	s.mutableState = TestGlobalMutableState(
		s.mockShard,
		s.mockEventsCache,
		s.logger,
		version,
		workflowID,
		runID,
	)

	newWorkflowTaskScheduleEvent, _ := s.prepareTransientWorkflowTaskCompletionFirstBatchApplied(version, workflowID, runID)

	newWorkflowTask := s.mutableState.GetWorkflowTaskByID(newWorkflowTaskScheduleEvent.GetEventId())
	s.NotNil(newWorkflowTask)

	_, err := s.mutableState.AddWorkflowTaskFailedEvent(
		newWorkflowTask,
		enumspb.WORKFLOW_TASK_FAILED_CAUSE_WORKFLOW_WORKER_UNHANDLED_FAILURE,
		failure.NewServerFailure("some random workflow task failure details", false),
		"some random workflow task failure identity",
		nil,
		"",
		"",
		"",
		0,
	)
	s.NoError(err)
	s.Equal(0, s.mutableState.hBuilder.NumBufferedEvents())
}

func (s *mutableStateSuite) TestRedirectInfoValidation_Valid() {
	tq := &taskqueuepb.TaskQueue{Name: "tq"}
	s.createVersionedMutableStateWithCompletedWFT(tq)

	wft, err := s.mutableState.AddWorkflowTaskScheduledEvent(true, enumsspb.WORKFLOW_TASK_TYPE_NORMAL)
	s.NoError(err)
	e, wft, err := s.mutableState.AddWorkflowTaskStartedEvent(
		wft.ScheduledEventID,
		"",
		tq,
		"",
		worker_versioning.StampFromBuildId("b2"),
		&taskqueuespb.BuildIdRedirectInfo{AssignedBuildId: "b1"},
		nil,
		false,
	)
	s.NoError(err)
	s.Equal("b2", wft.BuildId)
	s.Equal("b2", e.GetWorkflowTaskStartedEventAttributes().GetWorkerVersion().GetBuildId())
	s.Equal("b2", s.mutableState.GetAssignedBuildId())
	s.Equal(int64(1), wft.BuildIdRedirectCounter)
	s.Equal(int64(1), s.mutableState.GetExecutionInfo().GetBuildIdRedirectCounter())
}

func (s *mutableStateSuite) TestRedirectInfoValidation_Invalid() {
	tq := &taskqueuepb.TaskQueue{Name: "tq"}
	s.createVersionedMutableStateWithCompletedWFT(tq)

	wft, err := s.mutableState.AddWorkflowTaskScheduledEvent(true, enumsspb.WORKFLOW_TASK_TYPE_NORMAL)
	s.NoError(err)
	_, _, err = s.mutableState.AddWorkflowTaskStartedEvent(
		wft.ScheduledEventID,
		"",
		tq,
		"",
		worker_versioning.StampFromBuildId("b2"),
		&taskqueuespb.BuildIdRedirectInfo{AssignedBuildId: "b0"},
		nil,
		false,
	)
	expectedErr := &serviceerror2.ObsoleteDispatchBuildId{}
	s.ErrorAs(err, &expectedErr)
	s.Equal("b1", s.mutableState.GetAssignedBuildId())
	s.Equal(int64(0), s.mutableState.GetExecutionInfo().GetBuildIdRedirectCounter())
}

func (s *mutableStateSuite) TestRedirectInfoValidation_EmptyRedirectInfo() {
	tq := &taskqueuepb.TaskQueue{Name: "tq"}
	s.createVersionedMutableStateWithCompletedWFT(tq)

	wft, err := s.mutableState.AddWorkflowTaskScheduledEvent(true, enumsspb.WORKFLOW_TASK_TYPE_NORMAL)
	s.NoError(err)
	_, _, err = s.mutableState.AddWorkflowTaskStartedEvent(
		wft.ScheduledEventID,
		"",
		tq,
		"",
		worker_versioning.StampFromBuildId("b2"),
		nil,
		nil,
		false,
	)
	expectedErr := &serviceerror2.ObsoleteDispatchBuildId{}
	s.ErrorAs(err, &expectedErr)
	s.Equal("b1", s.mutableState.GetAssignedBuildId())
	s.Equal(int64(0), s.mutableState.GetExecutionInfo().GetBuildIdRedirectCounter())
}

func (s *mutableStateSuite) TestRedirectInfoValidation_EmptyStamp() {
	tq := &taskqueuepb.TaskQueue{Name: "tq"}
	s.createVersionedMutableStateWithCompletedWFT(tq)

	wft, err := s.mutableState.AddWorkflowTaskScheduledEvent(true, enumsspb.WORKFLOW_TASK_TYPE_NORMAL)
	s.NoError(err)
	_, _, err = s.mutableState.AddWorkflowTaskStartedEvent(
		wft.ScheduledEventID,
		"",
		tq,
		"",
		nil,
		&taskqueuespb.BuildIdRedirectInfo{AssignedBuildId: "b1"},
		nil,
		false,
	)
	expectedErr := &serviceerror2.ObsoleteDispatchBuildId{}
	s.ErrorAs(err, &expectedErr)
	s.Equal("b1", s.mutableState.GetAssignedBuildId())
	s.Equal(int64(0), s.mutableState.GetExecutionInfo().GetBuildIdRedirectCounter())
}

func (s *mutableStateSuite) TestRedirectInfoValidation_Sticky() {
	tq := &taskqueuepb.TaskQueue{Name: "tq"}
	sticky := &taskqueuepb.TaskQueue{Name: "sticky-tq", Kind: enumspb.TASK_QUEUE_KIND_STICKY}
	s.createVersionedMutableStateWithCompletedWFT(tq)

	s.mutableState.SetStickyTaskQueue(sticky.Name, durationpb.New(time.Second))
	wft, err := s.mutableState.AddWorkflowTaskScheduledEvent(true, enumsspb.WORKFLOW_TASK_TYPE_NORMAL)
	s.NoError(err)
	e, wft, err := s.mutableState.AddWorkflowTaskStartedEvent(
		wft.ScheduledEventID,
		"",
		sticky,
		"",
		nil,
		nil,
		nil,
		false,
	)
	s.NoError(err)
	s.Equal("", wft.BuildId)
	s.Equal("", e.GetWorkflowTaskStartedEventAttributes().GetWorkerVersion().GetBuildId())
	s.Equal("b1", s.mutableState.GetAssignedBuildId())
	s.Equal(int64(0), wft.BuildIdRedirectCounter)
	s.Equal(int64(0), s.mutableState.GetExecutionInfo().GetBuildIdRedirectCounter())
}

func (s *mutableStateSuite) TestRedirectInfoValidation_StickyInvalid() {
	tq := &taskqueuepb.TaskQueue{Name: "tq"}
	sticky := &taskqueuepb.TaskQueue{Name: "sticky-tq", Kind: enumspb.TASK_QUEUE_KIND_STICKY}
	s.createVersionedMutableStateWithCompletedWFT(tq)

	s.mutableState.SetStickyTaskQueue(sticky.Name, durationpb.New(time.Second))
	wft, err := s.mutableState.AddWorkflowTaskScheduledEvent(true, enumsspb.WORKFLOW_TASK_TYPE_NORMAL)
	s.NoError(err)
	_, _, err = s.mutableState.AddWorkflowTaskStartedEvent(
		wft.ScheduledEventID,
		"",
		&taskqueuepb.TaskQueue{Name: "another-sticky-tq"},
		"",
		nil,
		nil,
		nil,
		false,
	)
	expectedErr := &serviceerror2.ObsoleteDispatchBuildId{}
	s.ErrorAs(err, &expectedErr)
	s.Equal("b1", s.mutableState.GetAssignedBuildId())
	s.Equal(int64(0), s.mutableState.GetExecutionInfo().GetBuildIdRedirectCounter())
}

func (s *mutableStateSuite) TestRedirectInfoValidation_UnexpectedSticky() {
	tq := &taskqueuepb.TaskQueue{Name: "tq"}
	sticky := &taskqueuepb.TaskQueue{Name: "sticky-tq", Kind: enumspb.TASK_QUEUE_KIND_STICKY}
	s.createVersionedMutableStateWithCompletedWFT(tq)

	wft, err := s.mutableState.AddWorkflowTaskScheduledEvent(true, enumsspb.WORKFLOW_TASK_TYPE_NORMAL)
	s.NoError(err)
	_, _, err = s.mutableState.AddWorkflowTaskStartedEvent(
		wft.ScheduledEventID,
		"",
		sticky,
		"",
		nil,
		nil,
		nil,
		false,
	)
	expectedErr := &serviceerror2.ObsoleteDispatchBuildId{}
	s.ErrorAs(err, &expectedErr)
	s.Equal("b1", s.mutableState.GetAssignedBuildId())
	s.Equal(int64(0), s.mutableState.GetExecutionInfo().GetBuildIdRedirectCounter())
}

// creates a mutable state with first WFT completed on Build ID "b1"
func (s *mutableStateSuite) createVersionedMutableStateWithCompletedWFT(tq *taskqueuepb.TaskQueue) {
	version := int64(12)
	workflowID := "some random workflow ID"
	runID := uuid.New()
	s.mutableState = TestGlobalMutableState(
		s.mockShard,
		s.mockEventsCache,
		s.logger,
		version,
		workflowID,
		runID,
	)

	wft, err := s.mutableState.AddWorkflowTaskScheduledEvent(true, enumsspb.WORKFLOW_TASK_TYPE_NORMAL)
	s.NoError(err)
	e, wft, err := s.mutableState.AddWorkflowTaskStartedEvent(
		wft.ScheduledEventID,
		"",
		tq,
		"",
		worker_versioning.StampFromBuildId("b1"),
		nil,
		nil,
		false,
	)
	s.NoError(err)
	s.Equal("b1", wft.BuildId)
	s.Equal("b1", e.GetWorkflowTaskStartedEventAttributes().GetWorkerVersion().GetBuildId())
	s.Equal("b1", s.mutableState.GetAssignedBuildId())
	s.Equal(int64(0), wft.BuildIdRedirectCounter)
	s.Equal(int64(0), s.mutableState.GetExecutionInfo().GetBuildIdRedirectCounter())
	_, err = s.mutableState.AddWorkflowTaskCompletedEvent(
		wft,
		&workflowservice.RespondWorkflowTaskCompletedRequest{},
		workflowTaskCompletionLimits,
	)
	s.NoError(err)
}

func (s *mutableStateSuite) TestEffectiveDeployment() {
	tv := testvars.New(s)
	ms := TestGlobalMutableState(
		s.mockShard,
		s.mockEventsCache,
		s.logger,
		int64(12),
		tv.WorkflowID(),
		tv.RunID(),
	)
	s.Nil(ms.executionInfo.VersioningInfo)
	s.mutableState = ms
	s.verifyEffectiveDeployment(nil, enumspb.VERSIONING_BEHAVIOR_UNSPECIFIED)

	versioningInfo := &workflowpb.WorkflowExecutionVersioningInfo{}
	ms.executionInfo.VersioningInfo = versioningInfo
	s.verifyEffectiveDeployment(nil, enumspb.VERSIONING_BEHAVIOR_UNSPECIFIED)

	dv1 := worker_versioning.WorkerDeploymentVersionToStringV31(worker_versioning.DeploymentVersionFromDeployment(deployment1))
	dv2 := worker_versioning.WorkerDeploymentVersionToStringV31(worker_versioning.DeploymentVersionFromDeployment(deployment2))
	dv3 := worker_versioning.WorkerDeploymentVersionToStringV31(worker_versioning.DeploymentVersionFromDeployment(deployment3))

	deploymentVersion1 := worker_versioning.ExternalWorkerDeploymentVersionFromDeployment(deployment1)
	deploymentVersion2 := worker_versioning.ExternalWorkerDeploymentVersionFromDeployment(deployment2)
	deploymentVersion3 := worker_versioning.ExternalWorkerDeploymentVersionFromDeployment(deployment3)

	for _, useV32 := range []bool{true, false} {
		// ------- Without override, without transition

		// deployment is set but behavior is not -> unversioned
		if useV32 {
			versioningInfo.DeploymentVersion = deploymentVersion1
		} else {
			versioningInfo.Version = dv1 //nolint:staticcheck // SA1019: worker versioning v0.31
		}
		versioningInfo.Behavior = enumspb.VERSIONING_BEHAVIOR_UNSPECIFIED
		s.verifyEffectiveDeployment(nil, enumspb.VERSIONING_BEHAVIOR_UNSPECIFIED)

		if useV32 {
			versioningInfo.DeploymentVersion = deploymentVersion1
		} else {
			versioningInfo.Version = dv1 //nolint:staticcheck // SA1019: worker versioning v0.31
		}
		versioningInfo.Behavior = enumspb.VERSIONING_BEHAVIOR_PINNED
		s.verifyEffectiveDeployment(deployment1, enumspb.VERSIONING_BEHAVIOR_PINNED)

		if useV32 {
			versioningInfo.DeploymentVersion = deploymentVersion1
		} else {
			versioningInfo.Version = dv1 //nolint:staticcheck // SA1019: worker versioning v0.31
		}
		versioningInfo.Behavior = enumspb.VERSIONING_BEHAVIOR_AUTO_UPGRADE
		s.verifyEffectiveDeployment(deployment1, enumspb.VERSIONING_BEHAVIOR_AUTO_UPGRADE)

		// ------- With override, without transition

		// deployment and behavior are not set, but override behavior is AUTO_UPGRADE -> AUTO_UPGRADE
		if useV32 {
			versioningInfo.DeploymentVersion = nil
			versioningInfo.VersioningOverride = &workflowpb.VersioningOverride{
				Override: &workflowpb.VersioningOverride_AutoUpgrade{AutoUpgrade: true},
			}
		} else {
			versioningInfo.Version = "" //nolint:staticcheck // SA1019: worker versioning v0.31
			versioningInfo.VersioningOverride = &workflowpb.VersioningOverride{
				Behavior: enumspb.VERSIONING_BEHAVIOR_AUTO_UPGRADE, //nolint:staticcheck // SA1019: worker versioning v0.31
			}
		}
		versioningInfo.Behavior = enumspb.VERSIONING_BEHAVIOR_UNSPECIFIED
		s.verifyEffectiveDeployment(nil, enumspb.VERSIONING_BEHAVIOR_AUTO_UPGRADE)

		// deployment is set, behavior is not, but override behavior is AUTO_UPGRADE -> AUTO_UPGRADE
		if useV32 {
			versioningInfo.DeploymentVersion = deploymentVersion1
			versioningInfo.VersioningOverride = &workflowpb.VersioningOverride{
				Override: &workflowpb.VersioningOverride_AutoUpgrade{AutoUpgrade: true},
			}
		} else {
			versioningInfo.Version = dv1 //nolint:staticcheck // SA1019: worker versioning v0.31
			versioningInfo.VersioningOverride = &workflowpb.VersioningOverride{
				Behavior: enumspb.VERSIONING_BEHAVIOR_AUTO_UPGRADE, //nolint:staticcheck // SA1019: worker versioning v0.31
			}
		}
		versioningInfo.Behavior = enumspb.VERSIONING_BEHAVIOR_UNSPECIFIED
		s.verifyEffectiveDeployment(deployment1, enumspb.VERSIONING_BEHAVIOR_AUTO_UPGRADE)

		// worker says PINNED, but override behavior is AUTO_UPGRADE -> AUTO_UPGRADE
		if useV32 {
			versioningInfo.DeploymentVersion = deploymentVersion1
			versioningInfo.VersioningOverride = &workflowpb.VersioningOverride{
				Override: &workflowpb.VersioningOverride_AutoUpgrade{AutoUpgrade: true},
			}
		} else {
			versioningInfo.Version = dv1 //nolint:staticcheck // SA1019: worker versioning v0.31
			versioningInfo.VersioningOverride = &workflowpb.VersioningOverride{
				Behavior: enumspb.VERSIONING_BEHAVIOR_AUTO_UPGRADE, //nolint:staticcheck // SA1019: worker versioning v0.31
				// Technically, API should not allow deployment to be set for AUTO_UPGRADE override, but we
				// test it this way to make sure it is ignored.
				PinnedVersion: dv2, //nolint:staticcheck // SA1019: worker versioning v0.31
			}
		}
		versioningInfo.Behavior = enumspb.VERSIONING_BEHAVIOR_PINNED
		s.verifyEffectiveDeployment(deployment1, enumspb.VERSIONING_BEHAVIOR_AUTO_UPGRADE)

		// deployment and behavior are not set, but override behavior is PINNED -> PINNED
		if useV32 {
			versioningInfo.DeploymentVersion = nil
			versioningInfo.VersioningOverride = &workflowpb.VersioningOverride{
				Override: &workflowpb.VersioningOverride_Pinned{Pinned: &workflowpb.VersioningOverride_PinnedOverride{
					Behavior: workflowpb.VersioningOverride_PINNED_OVERRIDE_BEHAVIOR_PINNED,
					Version:  deploymentVersion2,
				}},
			}
		} else {
			versioningInfo.Version = "" //nolint:staticcheck // SA1019: worker versioning v0.31
			versioningInfo.VersioningOverride = &workflowpb.VersioningOverride{
				Behavior:      enumspb.VERSIONING_BEHAVIOR_PINNED, //nolint:staticcheck // SA1019: worker versioning v0.31
				PinnedVersion: dv2,                                //nolint:staticcheck // SA1019: worker versioning v0.31
			}
		}
		versioningInfo.Behavior = enumspb.VERSIONING_BEHAVIOR_UNSPECIFIED
		s.verifyEffectiveDeployment(deployment2, enumspb.VERSIONING_BEHAVIOR_PINNED)

		// deployment is set, behavior is not, but override behavior is PINNED --> PINNED
		if useV32 {
			versioningInfo.DeploymentVersion = deploymentVersion1
			versioningInfo.VersioningOverride = &workflowpb.VersioningOverride{
				Override: &workflowpb.VersioningOverride_Pinned{Pinned: &workflowpb.VersioningOverride_PinnedOverride{
					Behavior: workflowpb.VersioningOverride_PINNED_OVERRIDE_BEHAVIOR_PINNED,
					Version:  deploymentVersion2,
				}},
			}
		} else {
			versioningInfo.Version = dv1 //nolint:staticcheck // SA1019: worker versioning v0.31
			versioningInfo.VersioningOverride = &workflowpb.VersioningOverride{
				Behavior:      enumspb.VERSIONING_BEHAVIOR_PINNED, //nolint:staticcheck // SA1019: worker versioning v0.31
				PinnedVersion: dv2,                                //nolint:staticcheck // SA1019: worker versioning v0.31
			}
		}
		versioningInfo.Behavior = enumspb.VERSIONING_BEHAVIOR_UNSPECIFIED
		s.verifyEffectiveDeployment(deployment2, enumspb.VERSIONING_BEHAVIOR_PINNED)

		// worker says AUTO_UPGRADE, but override behavior is PINNED --> PINNED
		if useV32 {
			versioningInfo.DeploymentVersion = deploymentVersion1
			versioningInfo.VersioningOverride = &workflowpb.VersioningOverride{
				Override: &workflowpb.VersioningOverride_Pinned{Pinned: &workflowpb.VersioningOverride_PinnedOverride{
					Behavior: workflowpb.VersioningOverride_PINNED_OVERRIDE_BEHAVIOR_PINNED,
					Version:  deploymentVersion2,
				}},
			}
		} else {
			versioningInfo.Version = dv1 //nolint:staticcheck // SA1019: worker versioning v0.31
			versioningInfo.VersioningOverride = &workflowpb.VersioningOverride{
				Behavior:      enumspb.VERSIONING_BEHAVIOR_PINNED, //nolint:staticcheck // SA1019: worker versioning v0.31
				PinnedVersion: dv2,                                //nolint:staticcheck // SA1019: worker versioning v0.31
			}
		}
		versioningInfo.Behavior = enumspb.VERSIONING_BEHAVIOR_AUTO_UPGRADE
		s.verifyEffectiveDeployment(deployment2, enumspb.VERSIONING_BEHAVIOR_PINNED)

		// ------- With transition

		if useV32 {
			versioningInfo.DeploymentVersion = deploymentVersion1
			versioningInfo.VersioningOverride = &workflowpb.VersioningOverride{
				Override: &workflowpb.VersioningOverride_AutoUpgrade{AutoUpgrade: true},
			}
			versioningInfo.VersionTransition = &workflowpb.DeploymentVersionTransition{
				DeploymentVersion: deploymentVersion3,
			}
		} else {
			versioningInfo.Version = dv1 //nolint:staticcheck // SA1019: worker versioning v0.31
			versioningInfo.VersioningOverride = &workflowpb.VersioningOverride{
				Behavior:      enumspb.VERSIONING_BEHAVIOR_AUTO_UPGRADE, //nolint:staticcheck // SA1019: worker versioning v0.31
				PinnedVersion: dv2,                                      //nolint:staticcheck // SA1019: worker versioning v0.31
			}
			versioningInfo.VersionTransition = &workflowpb.DeploymentVersionTransition{
				Version: dv3, //nolint:staticcheck // SA1019: worker versioning v0.31
			}
		}
		versioningInfo.Behavior = enumspb.VERSIONING_BEHAVIOR_PINNED
		s.verifyEffectiveDeployment(deployment3, enumspb.VERSIONING_BEHAVIOR_AUTO_UPGRADE)

		if useV32 {
			versioningInfo.DeploymentVersion = deploymentVersion1
			versioningInfo.VersioningOverride = &workflowpb.VersioningOverride{
				Override: &workflowpb.VersioningOverride_AutoUpgrade{AutoUpgrade: true},
			}
			versioningInfo.VersionTransition = &workflowpb.DeploymentVersionTransition{
				DeploymentVersion: nil,
			}
		} else {
			versioningInfo.Version = dv1 //nolint:staticcheck // SA1019: worker versioning v0.31
			versioningInfo.VersioningOverride = &workflowpb.VersioningOverride{
				// Transitioning to unversioned
				Behavior:      enumspb.VERSIONING_BEHAVIOR_AUTO_UPGRADE, //nolint:staticcheck // SA1019: worker versioning v0.31
				PinnedVersion: dv2,                                      //nolint:staticcheck // SA1019: worker versioning v0.31
			}
			versioningInfo.VersionTransition = &workflowpb.DeploymentVersionTransition{
				Version: "", //nolint:staticcheck // SA1019: worker versioning v0.31
			}
		}
		versioningInfo.Behavior = enumspb.VERSIONING_BEHAVIOR_PINNED
		s.verifyEffectiveDeployment(nil, enumspb.VERSIONING_BEHAVIOR_AUTO_UPGRADE)

		// clear for next round
		versioningInfo.VersioningOverride = nil
		versioningInfo.VersionTransition = nil
		versioningInfo.DeploymentVersion = nil
		versioningInfo.Version = "" //nolint:staticcheck // SA1019: worker versioning v0.31
	}
}

func (s *mutableStateSuite) verifyEffectiveDeployment(
	expectedDeployment *deploymentpb.Deployment,
	expectedBehavior enumspb.VersioningBehavior,
) {
	if !s.mutableState.GetEffectiveDeployment().Equal(expectedDeployment) {
		s.Fail(fmt.Sprintf("expected: {%s}, actual: {%s}",
			expectedDeployment.String(),
			s.mutableState.GetEffectiveDeployment().String(),
		),
		)
	}
	s.Equal(expectedBehavior, s.mutableState.GetEffectiveVersioningBehavior())
}

// Creates a mutable state with first WFT completed on the given deployment and behavior set
// to the given behavior, testing expected output after Add, Start, and Complete Workflow Task.
func (s *mutableStateSuite) createMutableStateWithVersioningBehavior(
	behavior enumspb.VersioningBehavior,
	deployment *deploymentpb.Deployment,
	tq *taskqueuepb.TaskQueue,
) {
	version := int64(12)
	workflowID := "some random workflow ID"
	runID := uuid.New()

	s.mutableState = TestGlobalMutableState(
		s.mockShard,
		s.mockEventsCache,
		s.logger,
		version,
		workflowID,
		runID,
	)

	wft, err := s.mutableState.AddWorkflowTaskScheduledEvent(true, enumsspb.WORKFLOW_TASK_TYPE_NORMAL)
	s.NoError(err)
	s.verifyEffectiveDeployment(nil, enumspb.VERSIONING_BEHAVIOR_UNSPECIFIED)

	err = s.mutableState.StartDeploymentTransition(deployment)
	s.NoError(err)
	s.verifyEffectiveDeployment(deployment, enumspb.VERSIONING_BEHAVIOR_AUTO_UPGRADE)

	_, wft, err = s.mutableState.AddWorkflowTaskStartedEvent(
		wft.ScheduledEventID,
		"",
		tq,
		"",
		nil,
		nil,
		nil,
		false,
	)
	s.NoError(err)
	s.verifyEffectiveDeployment(deployment, enumspb.VERSIONING_BEHAVIOR_AUTO_UPGRADE)

	_, err = s.mutableState.AddWorkflowTaskCompletedEvent(
		wft,
		&workflowservice.RespondWorkflowTaskCompletedRequest{
			VersioningBehavior: behavior,
			Deployment:         deployment, //nolint:staticcheck // SA1019: worker versioning v0.30
		},
		workflowTaskCompletionLimits,
	)
	s.verifyEffectiveDeployment(deployment, behavior)
	s.NoError(err)
}

func (s *mutableStateSuite) TestPinnedFirstWorkflowTask() {
	tq := &taskqueuepb.TaskQueue{Name: "tq"}
	s.createMutableStateWithVersioningBehavior(enumspb.VERSIONING_BEHAVIOR_PINNED, deployment1, tq)
	s.verifyEffectiveDeployment(deployment1, enumspb.VERSIONING_BEHAVIOR_PINNED)
}

func (s *mutableStateSuite) TestUnpinnedFirstWorkflowTask() {
	tq := &taskqueuepb.TaskQueue{Name: "tq"}
	s.createMutableStateWithVersioningBehavior(enumspb.VERSIONING_BEHAVIOR_AUTO_UPGRADE, deployment1, tq)
	s.verifyEffectiveDeployment(deployment1, enumspb.VERSIONING_BEHAVIOR_AUTO_UPGRADE)
}

func (s *mutableStateSuite) TestUnpinnedTransition() {
	tq := &taskqueuepb.TaskQueue{Name: "tq"}
	behavior := enumspb.VERSIONING_BEHAVIOR_AUTO_UPGRADE
	s.createMutableStateWithVersioningBehavior(behavior, deployment1, tq)

	wft, err := s.mutableState.AddWorkflowTaskScheduledEvent(true, enumsspb.WORKFLOW_TASK_TYPE_NORMAL)
	s.NoError(err)
	s.verifyEffectiveDeployment(deployment1, behavior)

	err = s.mutableState.StartDeploymentTransition(deployment2)
	s.NoError(err)
	s.verifyEffectiveDeployment(deployment2, behavior)

	_, wft, err = s.mutableState.AddWorkflowTaskStartedEvent(
		wft.ScheduledEventID,
		"",
		tq,
		"",
		nil,
		nil,
		nil,
		false,
	)
	s.NoError(err)
	s.verifyEffectiveDeployment(deployment2, behavior)

	_, err = s.mutableState.AddWorkflowTaskCompletedEvent(
		wft,
		&workflowservice.RespondWorkflowTaskCompletedRequest{
			// wf is pinned in the new build
			VersioningBehavior: enumspb.VERSIONING_BEHAVIOR_PINNED,
			Deployment:         deployment2, //nolint:staticcheck // SA1019: worker versioning v0.30
		},
		workflowTaskCompletionLimits,
	)
	s.verifyEffectiveDeployment(deployment2, enumspb.VERSIONING_BEHAVIOR_PINNED)
	s.NoError(err)
}

func (s *mutableStateSuite) TestUnpinnedTransitionFailed() {
	tq := &taskqueuepb.TaskQueue{Name: "tq"}
	behavior := enumspb.VERSIONING_BEHAVIOR_AUTO_UPGRADE
	s.createMutableStateWithVersioningBehavior(behavior, deployment1, tq)

	wft, err := s.mutableState.AddWorkflowTaskScheduledEvent(true, enumsspb.WORKFLOW_TASK_TYPE_NORMAL)
	s.NoError(err)
	s.verifyEffectiveDeployment(deployment1, behavior)

	err = s.mutableState.StartDeploymentTransition(deployment2)
	s.NoError(err)
	s.verifyEffectiveDeployment(deployment2, behavior)

	_, wft, err = s.mutableState.AddWorkflowTaskStartedEvent(
		wft.ScheduledEventID,
		"",
		tq,
		"",
		nil,
		nil,
		nil,
		false,
	)
	s.NoError(err)
	s.verifyEffectiveDeployment(deployment2, behavior)

	_, err = s.mutableState.AddWorkflowTaskFailedEvent(
		wft,
		enumspb.WORKFLOW_TASK_FAILED_CAUSE_WORKFLOW_WORKER_UNHANDLED_FAILURE,
		failure.NewServerFailure("some random workflow task failure details", false),
		"some random workflow task failure identity",
		nil,
		"",
		"",
		"",
		0,
	)
	s.NoError(err)
	// WFT failure does not fail transition
	s.verifyEffectiveDeployment(deployment2, behavior)
}

func (s *mutableStateSuite) TestUnpinnedTransitionTimeout() {
	tq := &taskqueuepb.TaskQueue{Name: "tq"}
	behavior := enumspb.VERSIONING_BEHAVIOR_AUTO_UPGRADE
	s.createMutableStateWithVersioningBehavior(behavior, deployment1, tq)

	wft, err := s.mutableState.AddWorkflowTaskScheduledEvent(true, enumsspb.WORKFLOW_TASK_TYPE_NORMAL)
	s.NoError(err)
	s.verifyEffectiveDeployment(deployment1, behavior)

	err = s.mutableState.StartDeploymentTransition(deployment2)
	s.NoError(err)
	s.verifyEffectiveDeployment(deployment2, behavior)

	_, wft, err = s.mutableState.AddWorkflowTaskStartedEvent(
		wft.ScheduledEventID,
		"",
		tq,
		"",
		nil,
		nil,
		nil,
		false,
	)
	s.NoError(err)
	s.verifyEffectiveDeployment(deployment2, behavior)

	_, err = s.mutableState.AddWorkflowTaskTimedOutEvent(wft)
	s.NoError(err)
	// WFT timeout does not fail transition
	s.verifyEffectiveDeployment(deployment2, behavior)
}

func (s *mutableStateSuite) verifyWorkflowOptionsUpdatedEventAttr(
	actualAttr *historypb.WorkflowExecutionOptionsUpdatedEventAttributes,
	expectedAttr *historypb.WorkflowExecutionOptionsUpdatedEventAttributes,
) {
	expectedOverride := worker_versioning.ConvertOverrideToV32(expectedAttr.GetVersioningOverride())
	actualOverride := actualAttr.GetVersioningOverride()
	s.Equal(expectedOverride.GetPinned().GetBehavior(), actualOverride.GetPinned().GetBehavior())
	s.Equal(expectedOverride.GetPinned().GetVersion().GetDeploymentName(), actualOverride.GetPinned().GetVersion().GetDeploymentName())
	s.Equal(expectedOverride.GetPinned().GetVersion().GetBuildId(), actualOverride.GetPinned().GetVersion().GetBuildId())
	s.Equal(expectedOverride.GetAutoUpgrade(), actualOverride.GetAutoUpgrade())

	s.Equal(expectedOverride.GetBehavior(), actualOverride.GetBehavior())                                       //nolint:staticcheck // SA1019: worker versioning v0.31
	s.Equal(expectedOverride.GetDeployment().GetSeriesName(), expectedOverride.GetDeployment().GetSeriesName()) //nolint:staticcheck // SA1019: worker versioning v0.30
	s.Equal(expectedOverride.GetDeployment().GetBuildId(), expectedOverride.GetDeployment().GetBuildId())       //nolint:staticcheck // SA1019: worker versioning v0.30
	s.Equal(expectedOverride.GetPinnedVersion(), actualOverride.GetPinnedVersion())                             //nolint:staticcheck // SA1019: worker versioning v0.31

	s.Equal(actualAttr.GetUnsetVersioningOverride(), expectedAttr.GetUnsetVersioningOverride())
}

func (s *mutableStateSuite) verifyOverrides(
	expectedBehavior, expectedBehaviorOverride enumspb.VersioningBehavior,
	expectedDeployment, expectedDeploymentOverride *deploymentpb.Deployment,
) {
	versioningInfo := s.mutableState.GetExecutionInfo().GetVersioningInfo()
	s.Equal(expectedBehavior, versioningInfo.GetBehavior())
	if versioningInfo.GetVersioningOverride().GetAutoUpgrade() {
		s.Equal(expectedBehaviorOverride, enumspb.VERSIONING_BEHAVIOR_AUTO_UPGRADE)
	} else if versioningInfo.GetVersioningOverride().GetPinned().GetBehavior() == workflowpb.VersioningOverride_PINNED_OVERRIDE_BEHAVIOR_PINNED {
		s.Equal(expectedBehaviorOverride, enumspb.VERSIONING_BEHAVIOR_PINNED)
	}
	expectedVersion := ""
	if expectedDeployment != nil {
		expectedVersion = worker_versioning.WorkerDeploymentVersionToStringV31(worker_versioning.DeploymentVersionFromDeployment(expectedDeployment))
	}
	s.Equal(expectedVersion, versioningInfo.GetVersion()) //nolint:staticcheck // SA1019: worker versioning v0.31
	var expectedPinnedDeploymentVersion *deploymentpb.WorkerDeploymentVersion
	if expectedDeploymentOverride != nil {
		expectedPinnedDeploymentVersion = worker_versioning.ExternalWorkerDeploymentVersionFromDeployment(expectedDeploymentOverride)
	}
	s.Equal(expectedPinnedDeploymentVersion.GetDeploymentName(), versioningInfo.GetVersioningOverride().GetPinned().GetVersion().GetDeploymentName())
	s.Equal(expectedPinnedDeploymentVersion.GetBuildId(), versioningInfo.GetVersioningOverride().GetPinned().GetVersion().GetBuildId())
}

func (s *mutableStateSuite) TestOverride_UnpinnedBase_SetPinnedAndUnsetWithEmptyOptions() {
	tq := &taskqueuepb.TaskQueue{Name: "tq"}
	baseBehavior := enumspb.VERSIONING_BEHAVIOR_AUTO_UPGRADE
	overrideBehavior := enumspb.VERSIONING_BEHAVIOR_PINNED
	s.createMutableStateWithVersioningBehavior(baseBehavior, deployment1, tq)

	// set pinned override
	event, err := s.mutableState.AddWorkflowExecutionOptionsUpdatedEvent(pinnedOptions2.GetVersioningOverride(), false, "", nil, nil)
	s.NoError(err)
	s.verifyEffectiveDeployment(deployment2, overrideBehavior)
	s.verifyWorkflowOptionsUpdatedEventAttr(
		event.GetWorkflowExecutionOptionsUpdatedEventAttributes(),
		&historypb.WorkflowExecutionOptionsUpdatedEventAttributes{
			VersioningOverride:      pinnedOptions2.GetVersioningOverride(),
			UnsetVersioningOverride: false,
		},
	)
	s.verifyOverrides(baseBehavior, overrideBehavior, deployment1, deployment2)

	// unset pinned override with boolean
	event, err = s.mutableState.AddWorkflowExecutionOptionsUpdatedEvent(nil, true, "", nil, nil)
	s.NoError(err)
	s.verifyEffectiveDeployment(deployment1, baseBehavior)
	s.verifyWorkflowOptionsUpdatedEventAttr(
		event.GetWorkflowExecutionOptionsUpdatedEventAttributes(),
		&historypb.WorkflowExecutionOptionsUpdatedEventAttributes{
			VersioningOverride:      nil,
			UnsetVersioningOverride: true,
		},
	)
	s.verifyOverrides(baseBehavior, enumspb.VERSIONING_BEHAVIOR_UNSPECIFIED, deployment1, nil)
}

func (s *mutableStateSuite) TestOverride_PinnedBase_SetUnpinnedAndUnsetWithEmptyOptions() {
	tq := &taskqueuepb.TaskQueue{Name: "tq"}
	baseBehavior := enumspb.VERSIONING_BEHAVIOR_PINNED
	overrideBehavior := enumspb.VERSIONING_BEHAVIOR_AUTO_UPGRADE
	s.createMutableStateWithVersioningBehavior(baseBehavior, deployment1, tq)

	// set unpinned override
	event, err := s.mutableState.AddWorkflowExecutionOptionsUpdatedEvent(unpinnedOptions.GetVersioningOverride(), false, "", nil, nil)
	s.NoError(err)
	s.verifyEffectiveDeployment(deployment1, overrideBehavior)
	s.verifyWorkflowOptionsUpdatedEventAttr(
		event.GetWorkflowExecutionOptionsUpdatedEventAttributes(),
		&historypb.WorkflowExecutionOptionsUpdatedEventAttributes{
			VersioningOverride:      unpinnedOptions.GetVersioningOverride(),
			UnsetVersioningOverride: false,
		},
	)
	s.verifyOverrides(baseBehavior, overrideBehavior, deployment1, nil)

	// unset pinned override with empty
	event, err = s.mutableState.AddWorkflowExecutionOptionsUpdatedEvent(nil, true, "", nil, nil)
	s.NoError(err)
	s.verifyEffectiveDeployment(deployment1, baseBehavior)
	s.verifyWorkflowOptionsUpdatedEventAttr(
		event.GetWorkflowExecutionOptionsUpdatedEventAttributes(),
		&historypb.WorkflowExecutionOptionsUpdatedEventAttributes{
			VersioningOverride:      nil,
			UnsetVersioningOverride: true,
		},
	)
	s.verifyOverrides(baseBehavior, enumspb.VERSIONING_BEHAVIOR_UNSPECIFIED, deployment1, nil)
}

func (s *mutableStateSuite) TestOverride_RedirectFails() {
	tq := &taskqueuepb.TaskQueue{Name: "tq"}
	baseBehavior := enumspb.VERSIONING_BEHAVIOR_AUTO_UPGRADE
	overrideBehavior := enumspb.VERSIONING_BEHAVIOR_PINNED
	s.createMutableStateWithVersioningBehavior(baseBehavior, deployment1, tq)

	event, err := s.mutableState.AddWorkflowExecutionOptionsUpdatedEvent(pinnedOptions3.GetVersioningOverride(), false, "", nil, nil)
	s.NoError(err)
	s.verifyEffectiveDeployment(deployment3, overrideBehavior)
	s.verifyWorkflowOptionsUpdatedEventAttr(
		event.GetWorkflowExecutionOptionsUpdatedEventAttributes(),
		&historypb.WorkflowExecutionOptionsUpdatedEventAttributes{
			VersioningOverride:      pinnedOptions3.GetVersioningOverride(),
			UnsetVersioningOverride: false,
		},
	)
	s.verifyOverrides(baseBehavior, overrideBehavior, deployment1, deployment3)

	// assert that transition fails
	err = s.mutableState.StartDeploymentTransition(deployment2)
	s.ErrorIs(err, ErrPinnedWorkflowCannotTransition)
	s.verifyEffectiveDeployment(deployment3, overrideBehavior)
	s.verifyOverrides(baseBehavior, overrideBehavior, deployment1, deployment3)
}

func (s *mutableStateSuite) TestOverride_BaseDeploymentUpdatedOnCompletion() {
	s.T().Skip("TODO (Shahab)")
	tq := &taskqueuepb.TaskQueue{Name: "tq"}
	baseBehavior := enumspb.VERSIONING_BEHAVIOR_AUTO_UPGRADE
	overrideBehavior := enumspb.VERSIONING_BEHAVIOR_PINNED
	s.createMutableStateWithVersioningBehavior(baseBehavior, deployment1, tq)

	event, err := s.mutableState.AddWorkflowExecutionOptionsUpdatedEvent(pinnedOptions3.GetVersioningOverride(), false, "", nil, nil)
	s.NoError(err)
	s.verifyEffectiveDeployment(deployment3, overrideBehavior)
	s.verifyWorkflowOptionsUpdatedEventAttr(
		event.GetWorkflowExecutionOptionsUpdatedEventAttributes(),
		&historypb.WorkflowExecutionOptionsUpdatedEventAttributes{
			VersioningOverride:      pinnedOptions3.GetVersioningOverride(),
			UnsetVersioningOverride: false,
		},
	)
	s.verifyOverrides(baseBehavior, overrideBehavior, deployment1, deployment3)

	// assert that redirect fails - should be its own test
	err = s.mutableState.StartDeploymentTransition(deployment2)
	s.ErrorIs(err, ErrPinnedWorkflowCannotTransition)
	s.verifyEffectiveDeployment(deployment3, overrideBehavior)
	s.verifyOverrides(baseBehavior, overrideBehavior, deployment1, deployment3) // base deployment still deployment1 here -- good

	wft, err := s.mutableState.AddWorkflowTaskScheduledEvent(true, enumsspb.WORKFLOW_TASK_TYPE_NORMAL)
	s.NoError(err)
	s.verifyEffectiveDeployment(deployment3, overrideBehavior)

	_, wft, err = s.mutableState.AddWorkflowTaskStartedEvent(
		wft.ScheduledEventID,
		"",
		tq,
		"",
		nil,
		nil,
		nil,
		false,
	)
	s.NoError(err)
	s.verifyEffectiveDeployment(deployment3, overrideBehavior)
	s.verifyOverrides(baseBehavior, overrideBehavior, deployment1, deployment3)

	_, err = s.mutableState.AddWorkflowTaskCompletedEvent(
		wft,
		&workflowservice.RespondWorkflowTaskCompletedRequest{
			// sdk says wf is unpinned, but that does not take effect due to the override
			VersioningBehavior: baseBehavior,
			Deployment:         deployment2,
		},
		workflowTaskCompletionLimits,
	)
	s.NoError(err)
	s.verifyEffectiveDeployment(deployment3, overrideBehavior)
	s.verifyOverrides(baseBehavior, overrideBehavior, deployment2, deployment3)

	// now we unset the override and check that the base deployment/behavior is in effect
	event, err = s.mutableState.AddWorkflowExecutionOptionsUpdatedEvent(nil, true, "", nil, nil)
	s.NoError(err)
	s.verifyEffectiveDeployment(deployment2, baseBehavior)
	s.verifyWorkflowOptionsUpdatedEventAttr(
		event.GetWorkflowExecutionOptionsUpdatedEventAttributes(),
		&historypb.WorkflowExecutionOptionsUpdatedEventAttributes{
			VersioningOverride:      nil,
			UnsetVersioningOverride: true,
		},
	)
	s.verifyOverrides(baseBehavior, enumspb.VERSIONING_BEHAVIOR_UNSPECIFIED, deployment2, nil)
}

func (s *mutableStateSuite) TestChecksum() {
	// set the checksum probabilities to 100% for exercising during test
	s.mockConfig.MutableStateChecksumGenProbability = func(namespace string) int { return 100 }
	s.mockConfig.MutableStateChecksumVerifyProbability = func(namespace string) int { return 100 }

	testCases := []struct {
		name                 string
		enableBufferedEvents bool
		closeTxFunc          func(ms *MutableStateImpl) (*persistencespb.Checksum, error)
	}{
		{
			name: "closeTransactionAsSnapshot",
			closeTxFunc: func(ms *MutableStateImpl) (*persistencespb.Checksum, error) {
				snapshot, _, err := ms.CloseTransactionAsSnapshot(historyi.TransactionPolicyPassive)
				if err != nil {
					return nil, err
				}
				return snapshot.Checksum, err
			},
		},
		{
			name:                 "closeTransactionAsMutation",
			enableBufferedEvents: true,
			closeTxFunc: func(ms *MutableStateImpl) (*persistencespb.Checksum, error) {
				mutation, _, err := ms.CloseTransactionAsMutation(historyi.TransactionPolicyPassive)
				if err != nil {
					return nil, err
				}
				return mutation.Checksum, err
			},
		},
	}

	loadErrorsFunc := func() int64 {
		counter := s.testScope.Snapshot().Counters()["test.mutable_state_checksum_mismatch+operation=WorkflowContext,service_name=history"]
		if counter != nil {
			return counter.Value()
		}
		return 0
	}

	var loadErrors int64

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			dbState := s.buildWorkflowMutableState()
			if !tc.enableBufferedEvents {
				dbState.BufferedEvents = nil
			}

			// create mutable state and verify checksum is generated on close
			loadErrors = loadErrorsFunc()
			var err error
			s.mutableState, err = NewMutableStateFromDB(s.mockShard, s.mockEventsCache, s.logger, tests.LocalNamespaceEntry, dbState, 123)
			s.NoError(err)
			s.Equal(loadErrors, loadErrorsFunc()) // no errors expected
			s.EqualValues(dbState.Checksum, s.mutableState.checksum)
			s.mutableState.namespaceEntry = s.newNamespaceCacheEntry()
			csum, err := tc.closeTxFunc(s.mutableState)
			s.Nil(err)
			s.NotNil(csum.Value)
			s.Equal(enumsspb.CHECKSUM_FLAVOR_IEEE_CRC32_OVER_PROTO3_BINARY, csum.Flavor)
			s.Equal(mutableStateChecksumPayloadV1, csum.Version)
			s.EqualValues(csum, s.mutableState.checksum)

			// verify checksum is verified on Load
			dbState.Checksum = csum
			s.mutableState, err = NewMutableStateFromDB(s.mockShard, s.mockEventsCache, s.logger, tests.LocalNamespaceEntry, dbState, 123)
			s.NoError(err)
			s.Equal(loadErrors, loadErrorsFunc())

			// generate checksum again and verify its the same
			csum, err = tc.closeTxFunc(s.mutableState)
			s.Nil(err)
			s.NotNil(csum.Value)
			s.Equal(dbState.Checksum.Value, csum.Value)

			// modify checksum and verify Load fails
			dbState.Checksum.Value[0]++
			s.mutableState, err = NewMutableStateFromDB(s.mockShard, s.mockEventsCache, s.logger, tests.LocalNamespaceEntry, dbState, 123)
			s.NoError(err)
			s.Equal(loadErrors+1, loadErrorsFunc())
			s.EqualValues(dbState.Checksum, s.mutableState.checksum)

			// test checksum is invalidated
			loadErrors = loadErrorsFunc()
			s.mockConfig.MutableStateChecksumInvalidateBefore = func() float64 {
				return float64((s.mutableState.executionInfo.LastUpdateTime.AsTime().UnixNano() / int64(time.Second)) + 1)
			}
			s.mutableState, err = NewMutableStateFromDB(s.mockShard, s.mockEventsCache, s.logger, tests.LocalNamespaceEntry, dbState, 123)
			s.NoError(err)
			s.Equal(loadErrors, loadErrorsFunc())
			s.Nil(s.mutableState.checksum)

			// revert the config value for the next test case
			s.mockConfig.MutableStateChecksumInvalidateBefore = func() float64 {
				return float64(0)
			}
		})
	}
}

func (s *mutableStateSuite) TestChecksumProbabilities() {
	for _, prob := range []int{0, 100} {
		s.mockConfig.MutableStateChecksumGenProbability = func(namespace string) int { return prob }
		s.mockConfig.MutableStateChecksumVerifyProbability = func(namespace string) int { return prob }
		for i := 0; i < 100; i++ {
			shouldGenerate := s.mutableState.shouldGenerateChecksum()
			shouldVerify := s.mutableState.shouldVerifyChecksum()
			s.Equal(prob == 100, shouldGenerate)
			s.Equal(prob == 100, shouldVerify)
		}
	}
}

func (s *mutableStateSuite) TestChecksumShouldInvalidate() {
	s.mockConfig.MutableStateChecksumInvalidateBefore = func() float64 { return 0 }
	s.False(s.mutableState.shouldInvalidateCheckum())
	s.mutableState.executionInfo.LastUpdateTime = timestamp.TimeNowPtrUtc()
	s.mockConfig.MutableStateChecksumInvalidateBefore = func() float64 {
		return float64((s.mutableState.executionInfo.LastUpdateTime.AsTime().UnixNano() / int64(time.Second)) + 1)
	}
	s.True(s.mutableState.shouldInvalidateCheckum())
	s.mockConfig.MutableStateChecksumInvalidateBefore = func() float64 {
		return float64((s.mutableState.executionInfo.LastUpdateTime.AsTime().UnixNano() / int64(time.Second)) - 1)
	}
	s.False(s.mutableState.shouldInvalidateCheckum())
}

func (s *mutableStateSuite) TestContinueAsNewMinBackoff() {
	// set ContinueAsNew min interval to 5s
	s.mockConfig.WorkflowIdReuseMinimalInterval = func(namespace string) time.Duration {
		return 5 * time.Second
	}

	// with no backoff, verify min backoff is in [3s, 5s]
	minBackoff := s.mutableState.ContinueAsNewMinBackoff(nil).AsDuration()
	s.NotZero(minBackoff)
	s.True(minBackoff >= 3*time.Second)
	s.True(minBackoff <= 5*time.Second)

	// with 2s backoff, verify min backoff is in [3s, 5s]
	minBackoff = s.mutableState.ContinueAsNewMinBackoff(durationpb.New(time.Second * 2)).AsDuration()
	s.NotZero(minBackoff)
	s.True(minBackoff >= 3*time.Second)
	s.True(minBackoff <= 5*time.Second)

	// with 6s backoff, verify min backoff unchanged
	backoff := time.Second * 6
	minBackoff = s.mutableState.ContinueAsNewMinBackoff(durationpb.New(backoff)).AsDuration()
	s.NotZero(minBackoff)
	s.True(minBackoff == backoff)

	// set start time to be 3s ago
	startTime := timestamppb.New(time.Now().Add(-time.Second * 3))
	s.mutableState.executionInfo.StartTime = startTime
	s.mutableState.executionInfo.ExecutionTime = startTime
	s.mutableState.executionState.StartTime = startTime
	// with no backoff, verify min backoff is in [0, 2s]
	minBackoff = s.mutableState.ContinueAsNewMinBackoff(nil).AsDuration()
	s.NotNil(minBackoff)
	s.True(minBackoff >= 0)
	s.True(minBackoff <= 2*time.Second, "%v\n", minBackoff)

	// with 2s backoff, verify min backoff not changed
	backoff = time.Second * 2
	minBackoff = s.mutableState.ContinueAsNewMinBackoff(durationpb.New(backoff)).AsDuration()
	s.True(minBackoff == backoff)

	// set start time to be 5s ago
	startTime = timestamppb.New(time.Now().Add(-time.Second * 5))
	s.mutableState.executionInfo.StartTime = startTime
	s.mutableState.executionInfo.ExecutionTime = startTime
	s.mutableState.executionState.StartTime = startTime
	// with no backoff, verify backoff unchanged (no backoff needed)
	minBackoff = s.mutableState.ContinueAsNewMinBackoff(nil).AsDuration()
	s.Zero(minBackoff)

	// with 2s backoff, verify backoff unchanged
	backoff = time.Second * 2
	minBackoff = s.mutableState.ContinueAsNewMinBackoff(durationpb.New(backoff)).AsDuration()
	s.True(minBackoff == backoff)
}

func (s *mutableStateSuite) TestEventReapplied() {
	runID := uuid.New()
	eventID := int64(1)
	version := int64(2)
	dedupResource := definition.NewEventReappliedID(runID, eventID, version)
	isReapplied := s.mutableState.IsResourceDuplicated(dedupResource)
	s.False(isReapplied)
	s.mutableState.UpdateDuplicatedResource(dedupResource)
	isReapplied = s.mutableState.IsResourceDuplicated(dedupResource)
	s.True(isReapplied)
}

func (s *mutableStateSuite) TestTransientWorkflowTaskSchedule_CurrentVersionChanged() {
	version := int64(2000)
	workflowID := "some random workflow ID"
	runID := uuid.New()
	s.mutableState = TestGlobalMutableState(
		s.mockShard,
		s.mockEventsCache,
		s.logger,
		version,
		workflowID,
		runID,
	)
	_, _ = s.prepareTransientWorkflowTaskCompletionFirstBatchApplied(version, workflowID, runID)
	err := s.mutableState.ApplyWorkflowTaskFailedEvent()
	s.NoError(err)

	err = s.mutableState.UpdateCurrentVersion(version+1, true)
	s.NoError(err)
	versionHistories := s.mutableState.GetExecutionInfo().GetVersionHistories()
	versionHistory, err := versionhistory.GetCurrentVersionHistory(versionHistories)
	s.NoError(err)
	err = versionhistory.AddOrUpdateVersionHistoryItem(versionHistory, &historyspb.VersionHistoryItem{
		EventId: s.mutableState.GetNextEventID() - 1,
		Version: version,
	})
	s.NoError(err)

	wt, err := s.mutableState.AddWorkflowTaskScheduledEventAsHeartbeat(true, timestamp.TimeNowPtrUtc(), enumsspb.WORKFLOW_TASK_TYPE_NORMAL)
	s.NoError(err)
	s.NotNil(wt)

	s.Equal(int32(1), s.mutableState.GetExecutionInfo().WorkflowTaskAttempt)
	s.Equal(0, s.mutableState.hBuilder.NumBufferedEvents())
}

func (s *mutableStateSuite) TestTransientWorkflowTaskStart_CurrentVersionChanged() {
	version := int64(2000)
	workflowID := "some random workflow ID"
	runID := uuid.New()
	s.mutableState = TestGlobalMutableState(
		s.mockShard,
		s.mockEventsCache,
		s.logger,
		version,
		workflowID,
		runID,
	)
	_, _ = s.prepareTransientWorkflowTaskCompletionFirstBatchApplied(version, workflowID, runID)
	err := s.mutableState.ApplyWorkflowTaskFailedEvent()
	s.NoError(err)

	versionHistories := s.mutableState.GetExecutionInfo().GetVersionHistories()
	versionHistory, err := versionhistory.GetCurrentVersionHistory(versionHistories)
	s.NoError(err)
	err = versionhistory.AddOrUpdateVersionHistoryItem(versionHistory, &historyspb.VersionHistoryItem{
		EventId: s.mutableState.GetNextEventID() - 1,
		Version: version,
	})
	s.NoError(err)

	wt, err := s.mutableState.AddWorkflowTaskScheduledEventAsHeartbeat(true, timestamp.TimeNowPtrUtc(), enumsspb.WORKFLOW_TASK_TYPE_NORMAL)
	s.NoError(err)
	s.NotNil(wt)

	err = s.mutableState.UpdateCurrentVersion(version+1, true)
	s.NoError(err)

	f, err := tqid.NewTaskQueueFamily("", "tq")
	s.NoError(err)

	_, _, err = s.mutableState.AddWorkflowTaskStartedEvent(
		s.mutableState.GetNextEventID(),
		uuid.New(),
		&taskqueuepb.TaskQueue{Name: f.TaskQueue(enumspb.TASK_QUEUE_TYPE_WORKFLOW).NormalPartition(5).RpcName()},
		"random identity",
		nil,
		nil,
		nil,
		false,
	)
	s.NoError(err)
	s.Equal(0, s.mutableState.hBuilder.NumBufferedEvents())

	mutation, err := s.mutableState.hBuilder.Finish(true)
	s.NoError(err)
	s.Equal(1, len(mutation.DBEventsBatches))
	s.Equal(2, len(mutation.DBEventsBatches[0]))
	attrs := mutation.DBEventsBatches[0][0].GetWorkflowTaskScheduledEventAttributes()
	s.NotNil(attrs)
	s.Equal("tq", attrs.TaskQueue.Name)
}

func (s *mutableStateSuite) TestNewMutableStateInChain() {
	executionTimerTaskStatuses := []int32{
		TimerTaskStatusNone,
		TimerTaskStatusCreated,
	}

	for _, taskStatus := range executionTimerTaskStatuses {
		s.T().Run(
			fmt.Sprintf("TimerTaskStatus: %v", taskStatus),
			func(t *testing.T) {
				currentMutableState := TestGlobalMutableState(
					s.mockShard,
					s.mockEventsCache,
					s.logger,
					1000,
					tests.WorkflowID,
					uuid.New(),
				)
				currentMutableState.GetExecutionInfo().WorkflowExecutionTimerTaskStatus = taskStatus

				newMutableState, err := NewMutableStateInChain(
					s.mockShard,
					s.mockEventsCache,
					s.logger,
					tests.GlobalNamespaceEntry,
					tests.WorkflowID,
					uuid.New(),
					s.mockShard.GetTimeSource().Now(),
					currentMutableState,
				)
				s.NoError(err)
				s.Equal(taskStatus, newMutableState.GetExecutionInfo().WorkflowExecutionTimerTaskStatus)
			},
		)
	}
}

func (s *mutableStateSuite) TestSanitizedMutableState() {
	txnID := int64(2000)
	runID := uuid.New()
	mutableState := TestGlobalMutableState(
		s.mockShard,
		s.mockEventsCache,
		s.logger,
		1000,
		tests.WorkflowID,
		runID,
	)

	mutableState.executionInfo.LastFirstEventTxnId = txnID
	mutableState.executionInfo.ParentClock = &clockspb.VectorClock{
		ShardId: 1,
		Clock:   1,
	}
	mutableState.pendingChildExecutionInfoIDs = map[int64]*persistencespb.ChildExecutionInfo{1: {
		Clock: &clockspb.VectorClock{
			ShardId: 1,
			Clock:   1,
		},
	}}
	mutableState.executionInfo.WorkflowExecutionTimerTaskStatus = TimerTaskStatusCreated
	mutableState.executionInfo.TaskGenerationShardClockTimestamp = 1000

	stateMachineDef := hsmtest.NewDefinition("test")
	err := s.mockShard.StateMachineRegistry().RegisterMachine(stateMachineDef)
	s.NoError(err)
	_, err = mutableState.HSM().AddChild(hsm.Key{Type: stateMachineDef.Type(), ID: "child_1"}, hsmtest.NewData(hsmtest.State1))
	s.NoError(err)

	mutableStateProto := mutableState.CloneToProto()
	sanitizedMutableState, err := NewSanitizedMutableState(s.mockShard, s.mockEventsCache, s.logger, tests.LocalNamespaceEntry, mutableStateProto, 0)
	s.NoError(err)
	s.Equal(int64(0), sanitizedMutableState.executionInfo.LastFirstEventTxnId)
	s.Nil(sanitizedMutableState.executionInfo.ParentClock)
	for _, childInfo := range sanitizedMutableState.pendingChildExecutionInfoIDs {
		s.Nil(childInfo.Clock)
	}
	s.Equal(int32(TimerTaskStatusNone), sanitizedMutableState.executionInfo.WorkflowExecutionTimerTaskStatus)
	s.Zero(sanitizedMutableState.executionInfo.TaskGenerationShardClockTimestamp)
	err = sanitizedMutableState.HSM().Walk(func(node *hsm.Node) error {
		if node.Parent != nil {
			s.Equal(int64(1), node.InternalRepr().TransitionCount)
		}
		return nil
	})
	s.NoError(err)
}

func (s *mutableStateSuite) prepareTransientWorkflowTaskCompletionFirstBatchApplied(version int64, workflowID, runID string) (*historypb.HistoryEvent, *historypb.HistoryEvent) {
	namespaceID := tests.NamespaceID
	execution := &commonpb.WorkflowExecution{
		WorkflowId: workflowID,
		RunId:      runID,
	}

	now := time.Now().UTC()
	workflowType := "some random workflow type"
	taskqueue := "some random taskqueue"
	workflowTimeout := 222 * time.Second
	runTimeout := 111 * time.Second
	workflowTaskTimeout := 11 * time.Second
	workflowTaskAttempt := int32(1)

	eventID := int64(1)
	workflowStartEvent := &historypb.HistoryEvent{
		Version:   version,
		EventId:   eventID,
		EventTime: timestamppb.New(now),
		EventType: enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED,
		Attributes: &historypb.HistoryEvent_WorkflowExecutionStartedEventAttributes{WorkflowExecutionStartedEventAttributes: &historypb.WorkflowExecutionStartedEventAttributes{
			WorkflowType:             &commonpb.WorkflowType{Name: workflowType},
			TaskQueue:                &taskqueuepb.TaskQueue{Name: taskqueue},
			Input:                    nil,
			WorkflowExecutionTimeout: durationpb.New(workflowTimeout),
			WorkflowRunTimeout:       durationpb.New(runTimeout),
			WorkflowTaskTimeout:      durationpb.New(workflowTaskTimeout),
		}},
	}
	eventID++

	workflowTaskScheduleEvent := &historypb.HistoryEvent{
		Version:   version,
		EventId:   eventID,
		EventTime: timestamppb.New(now),
		EventType: enumspb.EVENT_TYPE_WORKFLOW_TASK_SCHEDULED,
		Attributes: &historypb.HistoryEvent_WorkflowTaskScheduledEventAttributes{WorkflowTaskScheduledEventAttributes: &historypb.WorkflowTaskScheduledEventAttributes{
			TaskQueue:           &taskqueuepb.TaskQueue{Name: taskqueue},
			StartToCloseTimeout: durationpb.New(workflowTaskTimeout),
			Attempt:             workflowTaskAttempt,
		}},
	}
	eventID++

	workflowTaskStartedEvent := &historypb.HistoryEvent{
		Version:   version,
		EventId:   eventID,
		EventTime: timestamppb.New(now),
		EventType: enumspb.EVENT_TYPE_WORKFLOW_TASK_STARTED,
		Attributes: &historypb.HistoryEvent_WorkflowTaskStartedEventAttributes{WorkflowTaskStartedEventAttributes: &historypb.WorkflowTaskStartedEventAttributes{
			ScheduledEventId: workflowTaskScheduleEvent.GetEventId(),
			RequestId:        uuid.New(),
		}},
	}
	eventID++

	_ = &historypb.HistoryEvent{
		Version:   version,
		EventId:   eventID,
		EventTime: timestamppb.New(now),
		EventType: enumspb.EVENT_TYPE_WORKFLOW_TASK_FAILED,
		Attributes: &historypb.HistoryEvent_WorkflowTaskFailedEventAttributes{WorkflowTaskFailedEventAttributes: &historypb.WorkflowTaskFailedEventAttributes{
			ScheduledEventId: workflowTaskScheduleEvent.GetEventId(),
			StartedEventId:   workflowTaskStartedEvent.GetEventId(),
		}},
	}
	eventID++

	s.mockEventsCache.EXPECT().PutEvent(
		events.EventKey{
			NamespaceID: namespaceID,
			WorkflowID:  execution.GetWorkflowId(),
			RunID:       execution.GetRunId(),
			EventID:     workflowStartEvent.GetEventId(),
			Version:     version,
		},
		workflowStartEvent,
	)
	err := s.mutableState.ApplyWorkflowExecutionStartedEvent(
		nil,
		execution,
		uuid.New(),
		workflowStartEvent,
	)
	s.Nil(err)

	// setup transient workflow task
	wt, err := s.mutableState.ApplyWorkflowTaskScheduledEvent(
		workflowTaskScheduleEvent.GetVersion(),
		workflowTaskScheduleEvent.GetEventId(),
		workflowTaskScheduleEvent.GetWorkflowTaskScheduledEventAttributes().GetTaskQueue(),
		workflowTaskScheduleEvent.GetWorkflowTaskScheduledEventAttributes().GetStartToCloseTimeout(),
		workflowTaskScheduleEvent.GetWorkflowTaskScheduledEventAttributes().GetAttempt(),
		nil,
		nil,
		enumsspb.WORKFLOW_TASK_TYPE_NORMAL,
	)
	s.Nil(err)
	s.NotNil(wt)

	wt, err = s.mutableState.ApplyWorkflowTaskStartedEvent(
		nil,
		workflowTaskStartedEvent.GetVersion(),
		workflowTaskScheduleEvent.GetEventId(),
		workflowTaskStartedEvent.GetEventId(),
		workflowTaskStartedEvent.GetWorkflowTaskStartedEventAttributes().GetRequestId(),
		timestamp.TimeValue(workflowTaskStartedEvent.GetEventTime()),
		false,
		123678,
		nil,
		int64(0),
	)
	s.Nil(err)
	s.NotNil(wt)

	err = s.mutableState.ApplyWorkflowTaskFailedEvent()
	s.Nil(err)

	workflowTaskAttempt = int32(123)
	newWorkflowTaskScheduleEvent := &historypb.HistoryEvent{
		Version:   version,
		EventId:   eventID,
		EventTime: timestamppb.New(now),
		EventType: enumspb.EVENT_TYPE_WORKFLOW_TASK_SCHEDULED,
		Attributes: &historypb.HistoryEvent_WorkflowTaskScheduledEventAttributes{WorkflowTaskScheduledEventAttributes: &historypb.WorkflowTaskScheduledEventAttributes{
			TaskQueue:           &taskqueuepb.TaskQueue{Name: taskqueue},
			StartToCloseTimeout: durationpb.New(workflowTaskTimeout),
			Attempt:             workflowTaskAttempt,
		}},
	}
	eventID++

	newWorkflowTaskStartedEvent := &historypb.HistoryEvent{
		Version:   version,
		EventId:   eventID,
		EventTime: timestamppb.New(now),
		EventType: enumspb.EVENT_TYPE_WORKFLOW_TASK_STARTED,
		Attributes: &historypb.HistoryEvent_WorkflowTaskStartedEventAttributes{WorkflowTaskStartedEventAttributes: &historypb.WorkflowTaskStartedEventAttributes{
			ScheduledEventId: workflowTaskScheduleEvent.GetEventId(),
			RequestId:        uuid.New(),
		}},
	}
	eventID++

	wt, err = s.mutableState.ApplyWorkflowTaskScheduledEvent(
		newWorkflowTaskScheduleEvent.GetVersion(),
		newWorkflowTaskScheduleEvent.GetEventId(),
		newWorkflowTaskScheduleEvent.GetWorkflowTaskScheduledEventAttributes().GetTaskQueue(),
		newWorkflowTaskScheduleEvent.GetWorkflowTaskScheduledEventAttributes().GetStartToCloseTimeout(),
		newWorkflowTaskScheduleEvent.GetWorkflowTaskScheduledEventAttributes().GetAttempt(),
		nil,
		nil,
		enumsspb.WORKFLOW_TASK_TYPE_NORMAL,
	)
	s.Nil(err)
	s.NotNil(wt)

	wt, err = s.mutableState.ApplyWorkflowTaskStartedEvent(
		nil,
		newWorkflowTaskStartedEvent.GetVersion(),
		newWorkflowTaskScheduleEvent.GetEventId(),
		newWorkflowTaskStartedEvent.GetEventId(),
		newWorkflowTaskStartedEvent.GetWorkflowTaskStartedEventAttributes().GetRequestId(),
		timestamp.TimeValue(newWorkflowTaskStartedEvent.GetEventTime()),
		false,
		123678,
		nil,
		int64(0),
	)
	s.Nil(err)
	s.NotNil(wt)

	s.mutableState.SetHistoryBuilder(historybuilder.NewImmutable([]*historypb.HistoryEvent{
		newWorkflowTaskScheduleEvent,
		newWorkflowTaskStartedEvent,
	}))
	_, _, err = s.mutableState.CloseTransactionAsMutation(historyi.TransactionPolicyPassive)
	s.NoError(err)

	return newWorkflowTaskScheduleEvent, newWorkflowTaskStartedEvent
}

func (s *mutableStateSuite) newNamespaceCacheEntry() *namespace.Namespace {
	return namespace.NewNamespaceForTest(
		&persistencespb.NamespaceInfo{Name: "mutableStateTest"},
		&persistencespb.NamespaceConfig{},
		true,
		&persistencespb.NamespaceReplicationConfig{},
		1,
	)
}

func (s *mutableStateSuite) buildWorkflowMutableState() *persistencespb.WorkflowMutableState {

	namespaceID := s.namespaceEntry.ID()
	we := &commonpb.WorkflowExecution{
		WorkflowId: "wId",
		RunId:      tests.RunID,
	}
	tl := "testTaskQueue"
	failoverVersion := s.namespaceEntry.FailoverVersion()

	startTime := timestamppb.New(time.Date(2020, 8, 22, 1, 2, 3, 4, time.UTC))
	info := &persistencespb.WorkflowExecutionInfo{
		NamespaceId:                             namespaceID.String(),
		WorkflowId:                              we.GetWorkflowId(),
		TaskQueue:                               tl,
		WorkflowTypeName:                        "wType",
		WorkflowRunTimeout:                      timestamp.DurationFromSeconds(200),
		DefaultWorkflowTaskTimeout:              timestamp.DurationFromSeconds(100),
		LastCompletedWorkflowTaskStartedEventId: int64(99),
		LastUpdateTime:                          timestamp.TimeNowPtrUtc(),
		ExecutionTime:                           startTime,
		WorkflowTaskVersion:                     failoverVersion,
		WorkflowTaskScheduledEventId:            101,
		WorkflowTaskStartedEventId:              102,
		WorkflowTaskTimeout:                     timestamp.DurationFromSeconds(100),
		WorkflowTaskAttempt:                     1,
		WorkflowTaskType:                        enumsspb.WORKFLOW_TASK_TYPE_NORMAL,
		VersionHistories: &historyspb.VersionHistories{
			Histories: []*historyspb.VersionHistory{
				{
					BranchToken: []byte("token#1"),
					Items: []*historyspb.VersionHistoryItem{
						{EventId: 102, Version: failoverVersion},
					},
				},
			},
		},
		TransitionHistory: []*persistencespb.VersionedTransition{
			{
				NamespaceFailoverVersion: failoverVersion,
				TransitionCount:          1024,
			},
		},
		FirstExecutionRunId:              uuid.New(),
		WorkflowExecutionTimerTaskStatus: TimerTaskStatusCreated,
	}

	state := &persistencespb.WorkflowExecutionState{
		RunId:     we.GetRunId(),
		State:     enumsspb.WORKFLOW_EXECUTION_STATE_RUNNING,
		Status:    enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING,
		StartTime: startTime,
	}

	activityInfos := map[int64]*persistencespb.ActivityInfo{
		90: {
			Version:                failoverVersion,
			ScheduledEventId:       int64(90),
			ScheduledTime:          timestamppb.New(time.Now().UTC()),
			StartedEventId:         common.EmptyEventID,
			StartedTime:            timestamppb.New(time.Now().UTC()),
			ActivityId:             "activityID_5",
			ScheduleToStartTimeout: timestamp.DurationFromSeconds(100),
			ScheduleToCloseTimeout: timestamp.DurationFromSeconds(200),
			StartToCloseTimeout:    timestamp.DurationFromSeconds(300),
			HeartbeatTimeout:       timestamp.DurationFromSeconds(50),
		},
	}

	expiryTime := timestamp.TimeNowPtrUtcAddDuration(time.Hour)
	timerInfos := map[string]*persistencespb.TimerInfo{
		"25": {
			Version:        failoverVersion,
			TimerId:        "25",
			StartedEventId: 85,
			ExpiryTime:     expiryTime,
		},
	}

	childInfos := map[int64]*persistencespb.ChildExecutionInfo{
		80: {
			Version:               failoverVersion,
			InitiatedEventId:      80,
			InitiatedEventBatchId: 20,
			StartedEventId:        common.EmptyEventID,
			CreateRequestId:       uuid.New(),
			Namespace:             tests.Namespace.String(),
			WorkflowTypeName:      "code.uber.internal/test/foobar",
		},
	}

	requestCancelInfo := map[int64]*persistencespb.RequestCancelInfo{
		70: {
			Version:               failoverVersion,
			InitiatedEventBatchId: 20,
			CancelRequestId:       uuid.New(),
			InitiatedEventId:      70,
		},
	}

	signalInfos := map[int64]*persistencespb.SignalInfo{
		75: {
			Version:               failoverVersion,
			InitiatedEventId:      75,
			InitiatedEventBatchId: 17,
			RequestId:             uuid.New(),
		},
	}

	signalRequestIDs := []string{
		"signal_request_id_1",
	}

	chasmNodes := map[string]*persistencespb.ChasmNode{
		"component-path": {
			Metadata: &persistencespb.ChasmNodeMetadata{
				InitialVersionedTransition:    &persistencespb.VersionedTransition{NamespaceFailoverVersion: failoverVersion, TransitionCount: 1},
				LastUpdateVersionedTransition: &persistencespb.VersionedTransition{NamespaceFailoverVersion: failoverVersion, TransitionCount: 90},
				Attributes: &persistencespb.ChasmNodeMetadata_ComponentAttributes{
					ComponentAttributes: &persistencespb.ChasmComponentAttributes{},
				},
			},
			Data: &commonpb.DataBlob{Data: []byte("test-data")},
		},
		"component-path/collection": {
			Metadata: &persistencespb.ChasmNodeMetadata{
				InitialVersionedTransition:    &persistencespb.VersionedTransition{NamespaceFailoverVersion: failoverVersion, TransitionCount: 1},
				LastUpdateVersionedTransition: &persistencespb.VersionedTransition{NamespaceFailoverVersion: failoverVersion, TransitionCount: 90},
				Attributes: &persistencespb.ChasmNodeMetadata_CollectionAttributes{
					CollectionAttributes: &persistencespb.ChasmCollectionAttributes{},
				},
			},
		},
	}

	bufferedEvents := []*historypb.HistoryEvent{
		{
			EventId:   common.BufferedEventID,
			EventType: enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_SIGNALED,
			Version:   failoverVersion,
			Attributes: &historypb.HistoryEvent_WorkflowExecutionSignaledEventAttributes{WorkflowExecutionSignaledEventAttributes: &historypb.WorkflowExecutionSignaledEventAttributes{
				SignalName: "test-signal-buffered",
				Input:      payloads.EncodeString("test-signal-buffered-input"),
			}},
		},
	}

	return &persistencespb.WorkflowMutableState{
		ExecutionInfo:       info,
		ExecutionState:      state,
		NextEventId:         int64(103),
		ActivityInfos:       activityInfos,
		TimerInfos:          timerInfos,
		ChildExecutionInfos: childInfos,
		RequestCancelInfos:  requestCancelInfo,
		SignalInfos:         signalInfos,
		ChasmNodes:          chasmNodes,
		SignalRequestedIds:  signalRequestIDs,
		BufferedEvents:      bufferedEvents,
	}
}

func (s *mutableStateSuite) TestUpdateInfos() {
	ctx := context.Background()
	cacheStore := map[events.EventKey]*historypb.HistoryEvent{}
	dbstate := s.buildWorkflowMutableState()
	var err error

	namespaceEntry := tests.GlobalNamespaceEntry
	s.mutableState, err = NewMutableStateFromDB(
		s.mockShard,
		NewMapEventCache(s.T(), cacheStore),
		s.logger,
		namespaceEntry,
		dbstate,
		123,
	)
	s.NoError(err)
	err = s.mutableState.UpdateCurrentVersion(namespaceEntry.FailoverVersion(), false)
	s.NoError(err)

	// 1st accepted update (without acceptedRequest)
	updateID1 := fmt.Sprintf("%s-1-accepted-update-id", s.T().Name())
	acptEvent1, err := s.mutableState.AddWorkflowExecutionUpdateAcceptedEvent(
		updateID1,
		fmt.Sprintf("%s-1-accepted-msg-id", s.T().Name()),
		1,
		nil) // no acceptedRequest!
	s.NoError(err)
	s.NotNil(acptEvent1)

	// 2nd accepted update (with acceptedRequest)
	updateID2 := fmt.Sprintf("%s-2-accepted-update-id", s.T().Name())
	acptEvent2, err := s.mutableState.AddWorkflowExecutionUpdateAcceptedEvent(
		updateID2,
		fmt.Sprintf("%s-2-accepted-msg-id", s.T().Name()),
		1,
		&updatepb.Request{Meta: &updatepb.Meta{UpdateId: updateID2}})
	s.NoError(err)
	s.NotNil(acptEvent2)

	_, err = s.mutableState.AddWorkflowExecutionUpdateCompletedEvent(
		1234,
		&updatepb.Response{
			Meta: &updatepb.Meta{UpdateId: s.T().Name() + "-completed-update-without-accepted-event"},
			Outcome: &updatepb.Outcome{
				Value: &updatepb.Outcome_Success{Success: testPayloads},
			},
		},
	)
	s.Error(err)

	completedEvent, err := s.mutableState.AddWorkflowExecutionUpdateCompletedEvent(
		acptEvent1.EventId,
		&updatepb.Response{
			Meta: &updatepb.Meta{UpdateId: updateID1},
			Outcome: &updatepb.Outcome{
				Value: &updatepb.Outcome_Success{Success: testPayloads},
			},
		},
	)
	s.NoError(err)
	s.NotNil(completedEvent)

	s.Len(cacheStore, 3, "expected 1 UpdateCompleted event + 2 UpdateAccepted events in cache")

	numCompleted := 0
	numAccepted := 0
	s.mutableState.VisitUpdates(func(updID string, updInfo *persistencespb.UpdateInfo) {
		s.Contains([]string{updateID1, updateID2}, updID)
		switch {
		case updInfo.GetCompletion() != nil:
			numCompleted++
		case updInfo.GetAcceptance() != nil:
			numAccepted++
		}
	})
	s.Equal(numCompleted, 1, "expected 1 completed")
	s.Equal(numAccepted, 1, "expected 1 accepted")

	s.mockShard.Resource.ClusterMetadata.EXPECT().ClusterNameForFailoverVersion(
		namespaceEntry.IsGlobalNamespace(),
		namespaceEntry.FailoverVersion(),
	).Return(cluster.TestCurrentClusterName).AnyTimes()
	s.mockShard.Resource.ClusterMetadata.EXPECT().GetCurrentClusterName().Return(cluster.TestCurrentClusterName).AnyTimes()

	mutation, _, err := s.mutableState.CloseTransactionAsMutation(historyi.TransactionPolicyActive)
	s.NoError(err)
	s.Len(mutation.ExecutionInfo.UpdateInfos, 2,
		"expected 1 completed update + 1 accepted in mutation")

	// this must be done after the transaction is closed
	// as GetUpdateOutcome relies on event version history which is only updated when closing the transaction
	outcome, err := s.mutableState.GetUpdateOutcome(ctx, completedEvent.GetWorkflowExecutionUpdateCompletedEventAttributes().GetMeta().GetUpdateId())
	s.NoError(err)
	s.Equal(completedEvent.GetWorkflowExecutionUpdateCompletedEventAttributes().GetOutcome(), outcome)

	_, err = s.mutableState.GetUpdateOutcome(ctx, "not_an_update_id")
	s.Error(err)
	s.IsType((*serviceerror.NotFound)(nil), err)
}

func (s *mutableStateSuite) TestApplyActivityTaskStartedEvent() {
	state := s.buildWorkflowMutableState()

	var err error
	s.mutableState, err = NewMutableStateFromDB(s.mockShard, s.mockEventsCache, s.logger, tests.LocalNamespaceEntry, state, 123)
	s.NoError(err)

	var scheduledEventID int64
	var ai *persistencespb.ActivityInfo
	for scheduledEventID, ai = range s.mutableState.GetPendingActivityInfos() {
		break
	}
	s.Nil(ai.LastHeartbeatDetails)

	now := time.Now().UTC()
	version := int64(101)
	requestID := "102"
	eventID := int64(104)
	attributes := &historypb.ActivityTaskStartedEventAttributes{
		ScheduledEventId: scheduledEventID,
		RequestId:        requestID,
	}
	err = s.mutableState.ApplyActivityTaskStartedEvent(&historypb.HistoryEvent{
		EventId:   eventID,
		EventTime: timestamppb.New(now),
		Version:   version,
		Attributes: &historypb.HistoryEvent_ActivityTaskStartedEventAttributes{
			ActivityTaskStartedEventAttributes: attributes,
		},
	})
	s.NoError(err)
	s.Assert().Equal(version, ai.Version)
	s.Assert().Equal(eventID, ai.StartedEventId)
	s.NotNil(ai.StartedTime)
	s.Assert().Equal(now, ai.StartedTime.AsTime())
	s.Assert().Equal(requestID, ai.RequestId)
	s.Assert().Nil(ai.LastHeartbeatDetails)
}

func (s *mutableStateSuite) TestAddContinueAsNewEvent_Default() {
	dbState := s.buildWorkflowMutableState()
	dbState.BufferedEvents = nil

	var err error
	s.mutableState, err = NewMutableStateFromDB(s.mockShard, s.mockEventsCache, s.logger, tests.LocalNamespaceEntry, dbState, 123)
	s.NoError(err)

	workflowTaskInfo := s.mutableState.GetStartedWorkflowTask()
	workflowTaskCompletedEvent, err := s.mutableState.AddWorkflowTaskCompletedEvent(
		workflowTaskInfo,
		&workflowservice.RespondWorkflowTaskCompletedRequest{},
		workflowTaskCompletionLimits,
	)
	s.NoError(err)

	coll := callbacks.MachineCollection(s.mutableState.HSM())
	_, err = coll.Add(
		"test-callback-carryover",
		callbacks.NewCallback(
			"random-request-id",
			timestamppb.Now(),
			callbacks.NewWorkflowClosedTrigger(),
			&persistencespb.Callback{
				Variant: &persistencespb.Callback_Nexus_{
					Nexus: &persistencespb.Callback_Nexus{
						Url: "test-callback-carryover-url",
					},
				},
			},
		),
	)
	s.NoError(err)

	s.mockEventsCache.EXPECT().GetEvent(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&historypb.HistoryEvent{}, nil)
	s.mockEventsCache.EXPECT().PutEvent(gomock.Any(), gomock.Any()).Times(2)
	_, newRunMutableState, err := s.mutableState.AddContinueAsNewEvent(
		context.Background(),
		workflowTaskCompletedEvent.GetEventId(),
		workflowTaskCompletedEvent.GetEventId(),
		"",
		&commandpb.ContinueAsNewWorkflowExecutionCommandAttributes{
			// All other fields will default to those in the current run.
			WorkflowRunTimeout: s.mutableState.GetExecutionInfo().WorkflowRunTimeout,
		},
		nil,
	)
	s.NoError(err)

	newColl := callbacks.MachineCollection(newRunMutableState.HSM())
	s.Equal(1, newColl.Size())

	currentRunExecutionInfo := s.mutableState.GetExecutionInfo()
	newRunExecutionInfo := newRunMutableState.GetExecutionInfo()
	s.Equal(currentRunExecutionInfo.TaskQueue, newRunExecutionInfo.TaskQueue)
	s.Equal(currentRunExecutionInfo.WorkflowTypeName, newRunExecutionInfo.WorkflowTypeName)
	protorequire.ProtoEqual(s.T(), currentRunExecutionInfo.DefaultWorkflowTaskTimeout, newRunExecutionInfo.DefaultWorkflowTaskTimeout)
	protorequire.ProtoEqual(s.T(), currentRunExecutionInfo.WorkflowRunTimeout, newRunExecutionInfo.WorkflowRunTimeout)
	protorequire.ProtoEqual(s.T(), currentRunExecutionInfo.WorkflowExecutionExpirationTime, newRunExecutionInfo.WorkflowExecutionExpirationTime)
	s.Equal(currentRunExecutionInfo.WorkflowExecutionTimerTaskStatus, newRunExecutionInfo.WorkflowExecutionTimerTaskStatus)
	s.Equal(currentRunExecutionInfo.FirstExecutionRunId, newRunExecutionInfo.FirstExecutionRunId)

	// Add more checks here if needed.
}

func (s *mutableStateSuite) TestTotalEntitiesCount() {
	s.mockEventsCache.EXPECT().PutEvent(gomock.Any(), gomock.Any()).AnyTimes()

	// scheduling, starting & completing workflow task is omitted here

	workflowTaskCompletedEventID := int64(4)
	_, _, err := s.mutableState.AddActivityTaskScheduledEvent(
		workflowTaskCompletedEventID,
		&commandpb.ScheduleActivityTaskCommandAttributes{},
		false,
	)
	s.NoError(err)

	_, _, err = s.mutableState.AddStartChildWorkflowExecutionInitiatedEvent(
		workflowTaskCompletedEventID,
		&commandpb.StartChildWorkflowExecutionCommandAttributes{},
		namespace.ID(uuid.New()),
	)
	s.NoError(err)

	_, _, err = s.mutableState.AddTimerStartedEvent(
		workflowTaskCompletedEventID,
		&commandpb.StartTimerCommandAttributes{},
	)
	s.NoError(err)

	_, _, err = s.mutableState.AddRequestCancelExternalWorkflowExecutionInitiatedEvent(
		workflowTaskCompletedEventID,
		uuid.New(),
		&commandpb.RequestCancelExternalWorkflowExecutionCommandAttributes{},
		namespace.ID(uuid.New()),
	)
	s.NoError(err)

	_, _, err = s.mutableState.AddSignalExternalWorkflowExecutionInitiatedEvent(
		workflowTaskCompletedEventID,
		uuid.New(),
		&commandpb.SignalExternalWorkflowExecutionCommandAttributes{
			Execution: &commonpb.WorkflowExecution{
				WorkflowId: tests.WorkflowID,
				RunId:      tests.RunID,
			},
		},
		namespace.ID(uuid.New()),
	)
	s.NoError(err)

	updateID := "random-updateId"
	accptEvent, err := s.mutableState.AddWorkflowExecutionUpdateAcceptedEvent(
		updateID, "random", 0, nil)
	s.NoError(err)
	s.NotNil(accptEvent)

	completedEvent, err := s.mutableState.AddWorkflowExecutionUpdateCompletedEvent(
		accptEvent.EventId, &updatepb.Response{Meta: &updatepb.Meta{UpdateId: updateID}})
	s.NoError(err)
	s.NotNil(completedEvent)

	_, err = s.mutableState.AddWorkflowExecutionSignaled(
		"signalName",
		&commonpb.Payloads{},
		"identity",
		&commonpb.Header{},
		nil,
	)
	s.NoError(err)

	mutation, _, err := s.mutableState.CloseTransactionAsMutation(
		historyi.TransactionPolicyActive,
	)
	s.NoError(err)

	s.Equal(int64(1), mutation.ExecutionInfo.ActivityCount)
	s.Equal(int64(1), mutation.ExecutionInfo.ChildExecutionCount)
	s.Equal(int64(1), mutation.ExecutionInfo.UserTimerCount)
	s.Equal(int64(1), mutation.ExecutionInfo.RequestCancelExternalCount)
	s.Equal(int64(1), mutation.ExecutionInfo.SignalExternalCount)
	s.Equal(int64(1), mutation.ExecutionInfo.SignalCount)
	s.Equal(int64(1), mutation.ExecutionInfo.UpdateCount)
}

func (s *mutableStateSuite) TestSpeculativeWorkflowTaskNotPersisted() {
	testCases := []struct {
		name                 string
		enableBufferedEvents bool
		closeTxFunc          func(ms *MutableStateImpl) (*persistencespb.WorkflowExecutionInfo, error)
	}{
		{
			name: "CloseTransactionAsSnapshot",
			closeTxFunc: func(ms *MutableStateImpl) (*persistencespb.WorkflowExecutionInfo, error) {
				snapshot, _, err := ms.CloseTransactionAsSnapshot(historyi.TransactionPolicyActive)
				if err != nil {
					return nil, err
				}
				return snapshot.ExecutionInfo, err
			},
		},
		{
			name:                 "CloseTransactionAsMutation",
			enableBufferedEvents: true,
			closeTxFunc: func(ms *MutableStateImpl) (*persistencespb.WorkflowExecutionInfo, error) {
				mutation, _, err := ms.CloseTransactionAsMutation(historyi.TransactionPolicyActive)
				if err != nil {
					return nil, err
				}
				return mutation.ExecutionInfo, err
			},
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			dbState := s.buildWorkflowMutableState()
			if !tc.enableBufferedEvents {
				dbState.BufferedEvents = nil
			}

			var err error
			namespaceEntry := tests.GlobalNamespaceEntry
			s.mutableState, err = NewMutableStateFromDB(s.mockShard, s.mockEventsCache, s.logger, namespaceEntry, dbState, 123)
			s.NoError(err)
			err = s.mutableState.UpdateCurrentVersion(namespaceEntry.FailoverVersion(), false)
			s.NoError(err)

			s.mutableState.executionInfo.WorkflowTaskScheduledEventId = s.mutableState.GetNextEventID()
			s.mutableState.executionInfo.WorkflowTaskStartedEventId = s.mutableState.GetNextEventID() + 1

			s.mockShard.Resource.ClusterMetadata.EXPECT().ClusterNameForFailoverVersion(
				namespaceEntry.IsGlobalNamespace(),
				namespaceEntry.FailoverVersion(),
			).Return(cluster.TestCurrentClusterName).AnyTimes()
			s.mockShard.Resource.ClusterMetadata.EXPECT().GetCurrentClusterName().Return(cluster.TestCurrentClusterName).AnyTimes()

			// Normal WT is persisted as is.
			execInfo, err := tc.closeTxFunc(s.mutableState)
			s.Nil(err)
			s.Equal(enumsspb.WORKFLOW_TASK_TYPE_NORMAL, execInfo.WorkflowTaskType)
			s.NotEqual(common.EmptyEventID, execInfo.WorkflowTaskScheduledEventId)
			s.NotEqual(common.EmptyEventID, execInfo.WorkflowTaskStartedEventId)

			s.mutableState.executionInfo.WorkflowTaskType = enumsspb.WORKFLOW_TASK_TYPE_SPECULATIVE

			// Speculative WT is converted to normal.
			execInfo, err = tc.closeTxFunc(s.mutableState)
			s.Nil(err)
			s.Equal(enumsspb.WORKFLOW_TASK_TYPE_NORMAL, execInfo.WorkflowTaskType)
			s.NotEqual(common.EmptyEventID, execInfo.WorkflowTaskScheduledEventId)
			s.NotEqual(common.EmptyEventID, execInfo.WorkflowTaskStartedEventId)
		})
	}
}

func (s *mutableStateSuite) TestRetryWorkflowTask_WithNextRetryDelay() {
	expectedDelayDuration := time.Minute
	s.mutableState.executionInfo.HasRetryPolicy = true
	applicationFailure := &failurepb.Failure{
		Message: "application failure with customized next retry delay",
		Source:  "application",
		FailureInfo: &failurepb.Failure_ApplicationFailureInfo{ApplicationFailureInfo: &failurepb.ApplicationFailureInfo{
			Type:           "application-failure-type",
			NonRetryable:   false,
			NextRetryDelay: durationpb.New(expectedDelayDuration),
		}},
	}

	duration, retryState := s.mutableState.GetRetryBackoffDuration(applicationFailure)
	s.Equal(enumspb.RETRY_STATE_IN_PROGRESS, retryState)
	s.Equal(duration, expectedDelayDuration)
}
func (s *mutableStateSuite) TestRetryActivity_TruncateRetryableFailure() {
	s.mockEventsCache.EXPECT().PutEvent(gomock.Any(), gomock.Any()).AnyTimes()

	// scheduling, starting & completing workflow task is omitted here

	workflowTaskCompletedEventID := int64(4)
	_, activityInfo, err := s.mutableState.AddActivityTaskScheduledEvent(
		workflowTaskCompletedEventID,
		&commandpb.ScheduleActivityTaskCommandAttributes{
			ActivityId:   "5",
			ActivityType: &commonpb.ActivityType{Name: "activity-type"},
			TaskQueue:    &taskqueuepb.TaskQueue{Name: "task-queue"},
			RetryPolicy: &commonpb.RetryPolicy{
				InitialInterval: timestamp.DurationFromSeconds(1),
			},
		},
		false,
	)
	s.NoError(err)

	_, err = s.mutableState.AddActivityTaskStartedEvent(
		activityInfo,
		activityInfo.ScheduledEventId,
		uuid.New(),
		"worker-identity",
		nil,
		nil,
		nil,
	)
	s.NoError(err)

	failureSizeErrorLimit := s.mockConfig.MutableStateActivityFailureSizeLimitError(
		s.mutableState.namespaceEntry.Name().String(),
	)

	activityFailure := &failurepb.Failure{
		Message: "activity failure with large details",
		Source:  "application",
		FailureInfo: &failurepb.Failure_ApplicationFailureInfo{ApplicationFailureInfo: &failurepb.ApplicationFailureInfo{
			Type:         "application-failure-type",
			NonRetryable: false,
			Details: &commonpb.Payloads{
				Payloads: []*commonpb.Payload{
					{
						Data: make([]byte, failureSizeErrorLimit*2),
					},
				},
			},
		}},
	}
	s.Greater(activityFailure.Size(), failureSizeErrorLimit)

	retryState, err := s.mutableState.RetryActivity(activityInfo, activityFailure)
	s.NoError(err)
	s.Equal(enumspb.RETRY_STATE_IN_PROGRESS, retryState)

	activityInfo, ok := s.mutableState.GetActivityInfo(activityInfo.ScheduledEventId)
	s.True(ok)
	s.LessOrEqual(activityInfo.RetryLastFailure.Size(), failureSizeErrorLimit)
	s.Equal(activityFailure.GetMessage(), activityInfo.RetryLastFailure.Cause.GetMessage())
}

func (s *mutableStateSuite) TestupdateBuildIdsAndDeploymentSearchAttributes() {
	versioned := func(buildId string) *commonpb.WorkerVersionStamp {
		return &commonpb.WorkerVersionStamp{BuildId: buildId, UseVersioning: true}
	}
	versionedSearchAttribute := func(buildIds ...string) []string {
		attrs := []string{}
		for _, buildId := range buildIds {
			attrs = append(attrs, worker_versioning.VersionedBuildIdSearchAttribute(buildId))
		}
		return attrs
	}
	unversioned := func(buildId string) *commonpb.WorkerVersionStamp {
		return &commonpb.WorkerVersionStamp{BuildId: buildId, UseVersioning: false}
	}
	unversionedSearchAttribute := func(buildIds ...string) []string {
		// assumed limit is 2
		attrs := []string{worker_versioning.UnversionedSearchAttribute, worker_versioning.UnversionedBuildIdSearchAttribute(buildIds[len(buildIds)-1])}
		return attrs
	}

	type testCase struct {
		name            string
		searchAttribute func(buildIds ...string) []string
		stamp           func(buildId string) *commonpb.WorkerVersionStamp
	}
	matrix := []testCase{
		{name: "unversioned", searchAttribute: unversionedSearchAttribute, stamp: unversioned},
		{name: "versioned", searchAttribute: versionedSearchAttribute, stamp: versioned},
	}
	for _, c := range matrix {
		s.T().Run(c.name, func(t *testing.T) {
			dbState := s.buildWorkflowMutableState()
			var err error
			s.mutableState, err = NewMutableStateFromDB(s.mockShard, s.mockEventsCache, s.logger, tests.LocalNamespaceEntry, dbState, 123)
			s.NoError(err)

			// Max 0
			err = s.mutableState.updateBuildIdsAndDeploymentSearchAttributes(c.stamp("0.1"), 0)
			s.NoError(err)
			s.Equal([]string{}, s.getBuildIdsFromMutableState())

			err = s.mutableState.updateBuildIdsAndDeploymentSearchAttributes(c.stamp("0.1"), 40)
			s.NoError(err)
			s.Equal(c.searchAttribute("0.1"), s.getBuildIdsFromMutableState())

			// Add the same build ID
			err = s.mutableState.updateBuildIdsAndDeploymentSearchAttributes(c.stamp("0.1"), 40)
			s.NoError(err)
			s.Equal(c.searchAttribute("0.1"), s.getBuildIdsFromMutableState())

			err = s.mutableState.updateBuildIdsAndDeploymentSearchAttributes(c.stamp("0.2"), 40)
			s.NoError(err)
			s.Equal(c.searchAttribute("0.1", "0.2"), s.getBuildIdsFromMutableState())

			// Limit applies
			err = s.mutableState.updateBuildIdsAndDeploymentSearchAttributes(c.stamp("0.3"), 40)
			s.NoError(err)
			s.Equal(c.searchAttribute("0.2", "0.3"), s.getBuildIdsFromMutableState())
		})
	}
}

func (s *mutableStateSuite) TestAddResetPointFromCompletion() {
	dbState := s.buildWorkflowMutableState()
	var err error
	s.mutableState, err = NewMutableStateFromDB(s.mockShard, s.mockEventsCache, s.logger, tests.LocalNamespaceEntry, dbState, 123)
	s.NoError(err)

	s.Nil(s.cleanedResetPoints().GetPoints())

	s.mutableState.addResetPointFromCompletion("checksum1", "buildid1", 32, 10)
	p1 := &workflowpb.ResetPointInfo{
		BuildId:                      "buildid1",
		BinaryChecksum:               "checksum1",
		RunId:                        s.mutableState.executionState.RunId,
		FirstWorkflowTaskCompletedId: 32,
	}
	s.Equal([]*workflowpb.ResetPointInfo{p1}, s.cleanedResetPoints().GetPoints())

	// new checksum + buildid
	s.mutableState.addResetPointFromCompletion("checksum2", "buildid2", 35, 10)
	p2 := &workflowpb.ResetPointInfo{
		BuildId:                      "buildid2",
		BinaryChecksum:               "checksum2",
		RunId:                        s.mutableState.executionState.RunId,
		FirstWorkflowTaskCompletedId: 35,
	}
	s.Equal([]*workflowpb.ResetPointInfo{p1, p2}, s.cleanedResetPoints().GetPoints())

	// same checksum + buildid, does not add new point
	s.mutableState.addResetPointFromCompletion("checksum2", "buildid2", 42, 10)
	s.Equal([]*workflowpb.ResetPointInfo{p1, p2}, s.cleanedResetPoints().GetPoints())

	// back to 1, does not add new point
	s.mutableState.addResetPointFromCompletion("checksum1", "buildid1", 48, 10)
	s.Equal([]*workflowpb.ResetPointInfo{p1, p2}, s.cleanedResetPoints().GetPoints())

	// buildid changes
	s.mutableState.addResetPointFromCompletion("checksum2", "buildid3", 53, 10)
	p3 := &workflowpb.ResetPointInfo{
		BuildId:                      "buildid3",
		BinaryChecksum:               "checksum2",
		RunId:                        s.mutableState.executionState.RunId,
		FirstWorkflowTaskCompletedId: 53,
	}
	s.Equal([]*workflowpb.ResetPointInfo{p1, p2, p3}, s.cleanedResetPoints().GetPoints())

	// limit to 3, p1 gets dropped
	s.mutableState.addResetPointFromCompletion("checksum2", "buildid4", 55, 3)
	p4 := &workflowpb.ResetPointInfo{
		BuildId:                      "buildid4",
		BinaryChecksum:               "checksum2",
		RunId:                        s.mutableState.executionState.RunId,
		FirstWorkflowTaskCompletedId: 55,
	}
	s.Equal([]*workflowpb.ResetPointInfo{p2, p3, p4}, s.cleanedResetPoints().GetPoints())
}

func (s *mutableStateSuite) TestRolloverAutoResetPointsWithExpiringTime() {
	runId1 := uuid.New()
	runId2 := uuid.New()
	runId3 := uuid.New()

	retention := 3 * time.Hour
	base := time.Now()
	t1 := timestamppb.New(base)
	now := timestamppb.New(base.Add(1 * time.Hour))
	t2 := timestamppb.New(base.Add(2 * time.Hour))
	t3 := timestamppb.New(base.Add(4 * time.Hour))

	points := []*workflowpb.ResetPointInfo{
		{
			BuildId:                      "buildid1",
			RunId:                        runId1,
			FirstWorkflowTaskCompletedId: 32,
			ExpireTime:                   t1,
		},
		{
			BuildId:                      "buildid2",
			RunId:                        runId1,
			FirstWorkflowTaskCompletedId: 63,
			ExpireTime:                   t1,
		},
		{
			BuildId:                      "buildid3",
			RunId:                        runId2,
			FirstWorkflowTaskCompletedId: 94,
			ExpireTime:                   t2,
		},
		{
			BuildId:                      "buildid4",
			RunId:                        runId3,
			FirstWorkflowTaskCompletedId: 125,
		},
	}

	newPoints := rolloverAutoResetPointsWithExpiringTime(&workflowpb.ResetPoints{Points: points}, runId3, now.AsTime(), retention)
	expected := []*workflowpb.ResetPointInfo{
		{
			BuildId:                      "buildid3",
			RunId:                        runId2,
			FirstWorkflowTaskCompletedId: 94,
			ExpireTime:                   t2,
		},
		{
			BuildId:                      "buildid4",
			RunId:                        runId3,
			FirstWorkflowTaskCompletedId: 125,
			ExpireTime:                   t3,
		},
	}
	s.Equal(expected, newPoints.Points)
}

func (s *mutableStateSuite) TestCloseTransactionUpdateTransition() {
	namespaceEntry := tests.GlobalNamespaceEntry

	completWorkflowTaskFn := func(ms historyi.MutableState) {
		workflowTaskInfo := ms.GetStartedWorkflowTask()
		_, err := ms.AddWorkflowTaskCompletedEvent(
			workflowTaskInfo,
			&workflowservice.RespondWorkflowTaskCompletedRequest{},
			workflowTaskCompletionLimits,
		)
		s.NoError(err)
	}

	testCases := []struct {
		name                       string
		dbStateMutationFn          func(dbState *persistencespb.WorkflowMutableState)
		txFunc                     func(ms historyi.MutableState) (*persistencespb.WorkflowExecutionInfo, error)
		versionedTransitionUpdated bool
	}{
		{
			name: "CloseTransactionAsMutation_HistoryEvents",
			dbStateMutationFn: func(dbState *persistencespb.WorkflowMutableState) {
				dbState.BufferedEvents = nil
			},
			txFunc: func(ms historyi.MutableState) (*persistencespb.WorkflowExecutionInfo, error) {
				completWorkflowTaskFn(ms)

				mutation, _, err := ms.CloseTransactionAsMutation(historyi.TransactionPolicyActive)
				if err != nil {
					return nil, err
				}
				return mutation.ExecutionInfo, err
			},
			versionedTransitionUpdated: true,
		},
		{
			name: "CloseTransactionAsMutation_BufferedEvents",
			dbStateMutationFn: func(dbState *persistencespb.WorkflowMutableState) {
				dbState.BufferedEvents = nil
			},
			txFunc: func(ms historyi.MutableState) (*persistencespb.WorkflowExecutionInfo, error) {
				var activityScheduleEventID int64
				for activityScheduleEventID = range s.mutableState.GetPendingActivityInfos() {
					break
				}
				_, err := s.mutableState.AddActivityTaskTimedOutEvent(
					activityScheduleEventID,
					common.EmptyEventID,
					failure.NewTimeoutFailure("test-timeout", enumspb.TIMEOUT_TYPE_SCHEDULE_TO_START),
					enumspb.RETRY_STATE_TIMEOUT,
				)
				s.NoError(err)

				mutation, _, err := ms.CloseTransactionAsMutation(historyi.TransactionPolicyActive)
				if err != nil {
					return nil, err
				}
				return mutation.ExecutionInfo, err
			},
			versionedTransitionUpdated: true,
		},
		{
			name: "CloseTransactionAsMutation_SyncActivity",
			dbStateMutationFn: func(dbState *persistencespb.WorkflowMutableState) {
				dbState.BufferedEvents = nil
			},
			txFunc: func(ms historyi.MutableState) (*persistencespb.WorkflowExecutionInfo, error) {
				for _, ai := range ms.GetPendingActivityInfos() {
					ms.UpdateActivityProgress(ai, &workflowservice.RecordActivityTaskHeartbeatRequest{})
					break
				}

				mutation, _, err := ms.CloseTransactionAsMutation(historyi.TransactionPolicyActive)
				if err != nil {
					return nil, err
				}
				return mutation.ExecutionInfo, err
			},
			versionedTransitionUpdated: true,
		},
		{
			name: "CloseTransactionAsMutation_DirtyStateMachine",
			dbStateMutationFn: func(dbState *persistencespb.WorkflowMutableState) {
				dbState.BufferedEvents = nil
			},
			txFunc: func(ms historyi.MutableState) (*persistencespb.WorkflowExecutionInfo, error) {
				root := ms.HSM()
				err := hsm.MachineTransition(root, func(*MutableStateImpl) (hsm.TransitionOutput, error) {
					return hsm.TransitionOutput{}, nil
				})
				s.NoError(err)
				s.True(root.Dirty())

				mutation, _, err := ms.CloseTransactionAsMutation(historyi.TransactionPolicyActive)
				if err != nil {
					return nil, err
				}
				return mutation.ExecutionInfo, err
			},
			versionedTransitionUpdated: true,
		},
		{
			name: "CloseTransactionAsMutation_SignalWorkflow",
			dbStateMutationFn: func(dbState *persistencespb.WorkflowMutableState) {
				dbState.BufferedEvents = nil
			},
			txFunc: func(ms historyi.MutableState) (*persistencespb.WorkflowExecutionInfo, error) {
				_, err := ms.AddWorkflowExecutionSignaledEvent(
					"signalName",
					&commonpb.Payloads{},
					"identity",
					&commonpb.Header{},
					nil,
					nil,
				)
				if err != nil {
					return nil, err
				}

				mutation, _, err := ms.CloseTransactionAsMutation(historyi.TransactionPolicyActive)
				if err != nil {
					return nil, err
				}
				return mutation.ExecutionInfo, err
			},
			versionedTransitionUpdated: true,
		},
		{
			name: "CloseTransactionAsMutation_ChasmTree",
			dbStateMutationFn: func(dbState *persistencespb.WorkflowMutableState) {
				dbState.BufferedEvents = nil
			},
			txFunc: func(ms historyi.MutableState) (*persistencespb.WorkflowExecutionInfo, error) {
				mockChasmTree := historyi.NewMockChasmTree(s.controller)
				mockChasmTree.EXPECT().Archetype().Return(chasm.Archetype("mock-archetype")).AnyTimes()
				gomock.InOrder(
					mockChasmTree.EXPECT().IsStateDirty().Return(true).AnyTimes(),
					mockChasmTree.EXPECT().CloseTransaction().Return(chasm.NodesMutation{
						UpdatedNodes: map[string]*persistencespb.ChasmNode{
							"node-path": {
								Metadata: &persistencespb.ChasmNodeMetadata{
									Attributes: &persistencespb.ChasmNodeMetadata_DataAttributes{
										DataAttributes: &persistencespb.ChasmDataAttributes{},
									},
								},
								Data: &commonpb.DataBlob{Data: []byte("test-data")},
							},
						},
					}, nil),
				)
				ms.(*MutableStateImpl).chasmTree = mockChasmTree

				mutation, _, err := ms.CloseTransactionAsMutation(historyi.TransactionPolicyActive)
				if err != nil {
					return nil, err
				}
				return mutation.ExecutionInfo, err
			},
			versionedTransitionUpdated: true,
		},
		{
			name: "CloseTransactionAsSnapshot",
			dbStateMutationFn: func(dbState *persistencespb.WorkflowMutableState) {
				dbState.BufferedEvents = nil
			},
			txFunc: func(ms historyi.MutableState) (*persistencespb.WorkflowExecutionInfo, error) {
				completWorkflowTaskFn(ms)

				mutation, _, err := ms.CloseTransactionAsSnapshot(historyi.TransactionPolicyActive)
				if err != nil {
					return nil, err
				}
				return mutation.ExecutionInfo, err
			},
			versionedTransitionUpdated: true,
		},
		{
			name: "CloseTransactionAsSnapshot, from unknown to enable",
			dbStateMutationFn: func(dbState *persistencespb.WorkflowMutableState) {
				dbState.BufferedEvents = nil
			},
			txFunc: func(ms historyi.MutableState) (*persistencespb.WorkflowExecutionInfo, error) {
				ms.GetExecutionInfo().PreviousTransitionHistory = ms.GetExecutionInfo().TransitionHistory
				ms.GetExecutionInfo().TransitionHistory = nil
				completWorkflowTaskFn(ms)

				mutation, _, err := ms.CloseTransactionAsSnapshot(historyi.TransactionPolicyActive)
				if err != nil {
					return nil, err
				}
				return mutation.ExecutionInfo, err
			},
			versionedTransitionUpdated: true,
		},
		// TODO: add a test for flushing buffered events using last event version.
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			dbState := s.buildWorkflowMutableState()
			if tc.dbStateMutationFn != nil {
				tc.dbStateMutationFn(dbState)
			}

			var err error
			s.mutableState, err = NewMutableStateFromDB(s.mockShard, s.mockEventsCache, s.logger, namespaceEntry, dbState, 123)
			s.NoError(err)
			err = s.mutableState.UpdateCurrentVersion(namespaceEntry.FailoverVersion(), false)
			s.NoError(err)

			s.mockShard.Resource.ClusterMetadata.EXPECT().ClusterNameForFailoverVersion(
				namespaceEntry.IsGlobalNamespace(),
				namespaceEntry.FailoverVersion(),
			).Return(cluster.TestCurrentClusterName).AnyTimes()
			s.mockShard.Resource.ClusterMetadata.EXPECT().GetCurrentClusterName().Return(cluster.TestCurrentClusterName).AnyTimes()
			var expectedTransitionHistory []*persistencespb.VersionedTransition
			if s.mutableState.executionInfo.TransitionHistory == nil {
				expectedTransitionHistory = transitionhistory.CopyVersionedTransitions(s.mutableState.executionInfo.PreviousTransitionHistory)
			} else {
				expectedTransitionHistory = transitionhistory.CopyVersionedTransitions(s.mutableState.executionInfo.TransitionHistory)
			}

			if tc.versionedTransitionUpdated {
				expectedTransitionHistory = UpdatedTransitionHistory(expectedTransitionHistory, namespaceEntry.FailoverVersion())
			}

			execInfo, err := tc.txFunc(s.mutableState)
			s.Nil(err)

			protorequire.ProtoSliceEqual(t, expectedTransitionHistory, execInfo.TransitionHistory)
		})
	}
}

func (s *mutableStateSuite) TestCloseTransactionTrackLastUpdateVersionedTransition() {
	namespaceEntry := tests.GlobalNamespaceEntry
	s.mockEventsCache.EXPECT().PutEvent(gomock.Any(), gomock.Any()).AnyTimes()

	stateMachineDef := hsmtest.NewDefinition("test")
	err := s.mockShard.StateMachineRegistry().RegisterMachine(stateMachineDef)
	s.NoError(err)

	completWorkflowTaskFn := func(ms historyi.MutableState) *historypb.HistoryEvent {
		workflowTaskInfo := ms.GetStartedWorkflowTask()
		completedEvent, err := ms.AddWorkflowTaskCompletedEvent(
			workflowTaskInfo,
			&workflowservice.RespondWorkflowTaskCompletedRequest{},
			workflowTaskCompletionLimits,
		)
		s.NoError(err)
		return completedEvent
	}

	buildHSMFn := func(ms historyi.MutableState) {
		hsmRoot := ms.HSM()
		child1, err := hsmRoot.AddChild(hsm.Key{Type: stateMachineDef.Type(), ID: "child_1"}, hsmtest.NewData(hsmtest.State1))
		s.NoError(err)
		_, err = child1.AddChild(hsm.Key{Type: stateMachineDef.Type(), ID: "child_1_1"}, hsmtest.NewData(hsmtest.State2))
		s.NoError(err)
		_, err = hsmRoot.AddChild(hsm.Key{Type: stateMachineDef.Type(), ID: "child_2"}, hsmtest.NewData(hsmtest.State3))
		s.NoError(err)
	}

	testCases := []struct {
		name   string
		testFn func(ms historyi.MutableState)
	}{
		{
			name: "Activity",
			testFn: func(ms historyi.MutableState) {
				completedEvent := completWorkflowTaskFn(ms)
				scheduledEvent, _, err := ms.AddActivityTaskScheduledEvent(
					completedEvent.GetEventId(),
					&commandpb.ScheduleActivityTaskCommandAttributes{},
					false,
				)
				s.NoError(err)

				_, _, err = ms.CloseTransactionAsMutation(historyi.TransactionPolicyActive)
				s.NoError(err)

				currentVersionedTransition := ms.CurrentVersionedTransition()

				s.Len(ms.GetPendingActivityInfos(), 2)
				for _, ai := range ms.GetPendingActivityInfos() {
					if ai.ScheduledEventId == scheduledEvent.EventId {
						protorequire.ProtoEqual(s.T(), currentVersionedTransition, ai.LastUpdateVersionedTransition)
					} else {
						protorequire.NotProtoEqual(s.T(), currentVersionedTransition, ai.LastUpdateVersionedTransition)
					}
				}
			},
		},
		{
			name: "UserTimer",
			testFn: func(ms historyi.MutableState) {
				completedEvent := completWorkflowTaskFn(ms)
				newTimerID := "new-timer-id"
				_, _, err := ms.AddTimerStartedEvent(
					completedEvent.GetEventId(),
					&commandpb.StartTimerCommandAttributes{
						TimerId: newTimerID,
					},
				)
				s.NoError(err)

				_, _, err = ms.CloseTransactionAsMutation(historyi.TransactionPolicyActive)
				s.NoError(err)

				currentVersionedTransition := ms.CurrentVersionedTransition()

				s.Len(ms.GetPendingTimerInfos(), 2)
				for _, ti := range ms.GetPendingTimerInfos() {
					if ti.TimerId == newTimerID {
						protorequire.ProtoEqual(s.T(), currentVersionedTransition, ti.LastUpdateVersionedTransition)
					} else {
						protorequire.NotProtoEqual(s.T(), currentVersionedTransition, ti.LastUpdateVersionedTransition)
					}
				}
			},
		},
		{
			name: "ChildExecution",
			testFn: func(ms historyi.MutableState) {
				completedEvent := completWorkflowTaskFn(ms)
				initiatedEvent, _, err := ms.AddStartChildWorkflowExecutionInitiatedEvent(
					completedEvent.GetEventId(),
					&commandpb.StartChildWorkflowExecutionCommandAttributes{},
					ms.GetNamespaceEntry().ID(),
				)
				s.NoError(err)

				_, _, err = ms.CloseTransactionAsMutation(historyi.TransactionPolicyActive)
				s.NoError(err)

				currentVersionedTransition := ms.CurrentVersionedTransition()

				s.Len(ms.GetPendingChildExecutionInfos(), 2)
				for _, ci := range ms.GetPendingChildExecutionInfos() {
					if ci.InitiatedEventId == initiatedEvent.EventId {
						protorequire.ProtoEqual(s.T(), currentVersionedTransition, ci.LastUpdateVersionedTransition)
					} else {
						protorequire.NotProtoEqual(s.T(), currentVersionedTransition, ci.LastUpdateVersionedTransition)
					}
				}
			},
		},
		{
			name: "RequestCancelExternal",
			testFn: func(ms historyi.MutableState) {
				completedEvent := completWorkflowTaskFn(ms)
				initiatedEvent, _, err := ms.AddRequestCancelExternalWorkflowExecutionInitiatedEvent(
					completedEvent.GetEventId(),
					uuid.New(),
					&commandpb.RequestCancelExternalWorkflowExecutionCommandAttributes{},
					ms.GetNamespaceEntry().ID(),
				)
				s.NoError(err)

				_, _, err = ms.CloseTransactionAsMutation(historyi.TransactionPolicyActive)
				s.NoError(err)

				currentVersionedTransition := ms.CurrentVersionedTransition()

				s.Len(ms.GetPendingRequestCancelExternalInfos(), 2)
				for _, ci := range ms.GetPendingRequestCancelExternalInfos() {
					if ci.InitiatedEventId == initiatedEvent.EventId {
						protorequire.ProtoEqual(s.T(), currentVersionedTransition, ci.LastUpdateVersionedTransition)
					} else {
						protorequire.NotProtoEqual(s.T(), currentVersionedTransition, ci.LastUpdateVersionedTransition)
					}
				}
			},
		},
		{
			name: "SignalExternal",
			testFn: func(ms historyi.MutableState) {
				completedEvent := completWorkflowTaskFn(ms)
				initiatedEvent, _, err := ms.AddSignalExternalWorkflowExecutionInitiatedEvent(
					completedEvent.GetEventId(),
					uuid.New(),
					&commandpb.SignalExternalWorkflowExecutionCommandAttributes{
						Execution: &commonpb.WorkflowExecution{
							WorkflowId: "target-workflow-id",
							RunId:      "target-run-id",
						},
					},
					ms.GetNamespaceEntry().ID(),
				)
				s.NoError(err)

				_, _, err = ms.CloseTransactionAsMutation(historyi.TransactionPolicyActive)
				s.NoError(err)

				currentVersionedTransition := ms.CurrentVersionedTransition()

				s.Len(ms.GetPendingSignalExternalInfos(), 2)
				for _, ci := range ms.GetPendingSignalExternalInfos() {
					if ci.InitiatedEventId == initiatedEvent.EventId {
						protorequire.ProtoEqual(s.T(), currentVersionedTransition, ci.LastUpdateVersionedTransition)
					} else {
						protorequire.NotProtoEqual(s.T(), currentVersionedTransition, ci.LastUpdateVersionedTransition)
					}
				}
			},
		},
		{
			name: "SignalRequestedID",
			testFn: func(ms historyi.MutableState) {
				ms.AddSignalRequested(uuid.New())

				_, _, err := ms.CloseTransactionAsMutation(historyi.TransactionPolicyActive)
				s.NoError(err)

				currentVersionedTransition := ms.CurrentVersionedTransition()
				protorequire.ProtoEqual(s.T(), currentVersionedTransition, ms.GetExecutionInfo().SignalRequestIdsLastUpdateVersionedTransition)
			},
		},
		{
			name: "UpdateInfo",
			testFn: func(ms historyi.MutableState) {
				updateID := "test-updateId"
				_, err := ms.AddWorkflowExecutionUpdateAcceptedEvent(
					updateID,
					"update-message-id",
					65,
					nil, // this is an optional field
				)
				s.NoError(err)

				_, _, err = ms.CloseTransactionAsMutation(historyi.TransactionPolicyActive)
				s.NoError(err)

				currentVersionedTransition := ms.CurrentVersionedTransition()
				s.Len(ms.GetExecutionInfo().UpdateInfos, 1)
				protorequire.ProtoEqual(s.T(), currentVersionedTransition, ms.GetExecutionInfo().UpdateInfos[updateID].LastUpdateVersionedTransition)
			},
		},
		{
			name: "WorkflowTask/Completed",
			testFn: func(ms historyi.MutableState) {
				completWorkflowTaskFn(ms)

				_, _, err := ms.CloseTransactionAsMutation(historyi.TransactionPolicyActive)
				s.NoError(err)

				currentVersionedTransition := ms.CurrentVersionedTransition()
				protorequire.ProtoEqual(s.T(), currentVersionedTransition, ms.GetExecutionInfo().WorkflowTaskLastUpdateVersionedTransition)
			},
		},
		{
			name: "WorkflowTask/Scheduled",
			testFn: func(ms historyi.MutableState) {
				completWorkflowTaskFn(ms)
				_, err := ms.AddWorkflowTaskScheduledEvent(false, enumsspb.WORKFLOW_TASK_TYPE_NORMAL)
				s.NoError(err)

				_, _, err = ms.CloseTransactionAsMutation(historyi.TransactionPolicyActive)
				s.NoError(err)

				currentVersionedTransition := ms.CurrentVersionedTransition()
				protorequire.ProtoEqual(s.T(), currentVersionedTransition, ms.GetExecutionInfo().WorkflowTaskLastUpdateVersionedTransition)
			},
		},
		{
			name: "Visibility",
			testFn: func(ms historyi.MutableState) {
				completedEvent := completWorkflowTaskFn(ms)
				_, err := ms.AddUpsertWorkflowSearchAttributesEvent(
					completedEvent.EventId,
					&commandpb.UpsertWorkflowSearchAttributesCommandAttributes{},
				)
				s.NoError(err)

				_, _, err = ms.CloseTransactionAsMutation(historyi.TransactionPolicyActive)
				s.NoError(err)

				currentVersionedTransition := ms.CurrentVersionedTransition()
				protorequire.ProtoEqual(s.T(), currentVersionedTransition, ms.GetExecutionInfo().VisibilityLastUpdateVersionedTransition)
			},
		},
		{
			name: "ExecutionState",
			testFn: func(ms historyi.MutableState) {
				completedEvent := completWorkflowTaskFn(ms)
				_, err := ms.AddCompletedWorkflowEvent(
					completedEvent.EventId,
					&commandpb.CompleteWorkflowExecutionCommandAttributes{},
					"",
				)
				s.NoError(err)

				_, _, err = ms.CloseTransactionAsMutation(historyi.TransactionPolicyActive)
				s.NoError(err)

				currentVersionedTransition := ms.CurrentVersionedTransition()
				protorequire.ProtoEqual(s.T(), currentVersionedTransition, ms.GetExecutionState().LastUpdateVersionedTransition)
			},
		},
		{
			name: "HSM/CloseAsMutation",
			testFn: func(ms historyi.MutableState) {
				completWorkflowTaskFn(ms)
				buildHSMFn(ms)

				_, _, err := ms.CloseTransactionAsMutation(historyi.TransactionPolicyActive)
				s.NoError(err)

				currentVersionedTransition := ms.CurrentVersionedTransition()
				err = ms.HSM().Walk(func(n *hsm.Node) error {
					if n.Parent == nil {
						// skip root which is entire mutable state
						return nil
					}
					protorequire.ProtoEqual(s.T(), currentVersionedTransition, n.InternalRepr().LastUpdateVersionedTransition)
					return nil
				})
				s.NoError(err)
			},
		},
		{
			name: "HSM/CloseAsSnapshot",
			testFn: func(ms historyi.MutableState) {
				completWorkflowTaskFn(ms)
				buildHSMFn(ms)

				_, _, err := ms.CloseTransactionAsSnapshot(historyi.TransactionPolicyActive)
				s.NoError(err)

				currentVersionedTransition := ms.CurrentVersionedTransition()
				err = ms.HSM().Walk(func(n *hsm.Node) error {
					if n.Parent == nil {
						// skip root which is entire mutable state
						return nil
					}
					protorequire.ProtoEqual(s.T(), currentVersionedTransition, n.InternalRepr().LastUpdateVersionedTransition)
					return nil
				})
				s.NoError(err)
			},
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {

			dbState := s.buildWorkflowMutableState()
			dbState.BufferedEvents = nil

			var err error
			s.mutableState, err = NewMutableStateFromDB(s.mockShard, s.mockEventsCache, s.logger, namespaceEntry, dbState, 123)
			s.NoError(err)
			err = s.mutableState.UpdateCurrentVersion(namespaceEntry.FailoverVersion(), false)
			s.NoError(err)

			s.mockShard.Resource.ClusterMetadata.EXPECT().ClusterNameForFailoverVersion(
				namespaceEntry.IsGlobalNamespace(),
				namespaceEntry.FailoverVersion(),
			).Return(cluster.TestCurrentClusterName).AnyTimes()
			s.mockShard.Resource.ClusterMetadata.EXPECT().GetCurrentClusterName().Return(cluster.TestCurrentClusterName).AnyTimes()

			tc.testFn(s.mutableState)
		})
	}
}

func (s *mutableStateSuite) TestCloseTransactionHandleUnknownVersionedTransition() {
	namespaceEntry := tests.GlobalNamespaceEntry

	completWorkflowTaskFn := func(ms historyi.MutableState) {
		workflowTaskInfo := ms.GetStartedWorkflowTask()
		_, err := ms.AddWorkflowTaskCompletedEvent(
			workflowTaskInfo,
			&workflowservice.RespondWorkflowTaskCompletedRequest{},
			historyi.WorkflowTaskCompletionLimits{
				MaxResetPoints:              10,
				MaxSearchAttributeValueSize: 1024,
			},
		)
		s.NoError(err)
	}

	testCases := []struct {
		name              string
		dbStateMutationFn func(dbState *persistencespb.WorkflowMutableState)
		txFunc            func(ms historyi.MutableState) (*persistencespb.WorkflowExecutionInfo, error)
	}{
		{
			name: "CloseTransactionAsPassive", // this scenario simulate the case to clear the transition history (non state-based transition happened at passive side)
			dbStateMutationFn: func(dbState *persistencespb.WorkflowMutableState) {
				dbState.BufferedEvents = nil
			},
			txFunc: func(ms historyi.MutableState) (*persistencespb.WorkflowExecutionInfo, error) {
				completWorkflowTaskFn(ms)

				mutation, _, err := ms.CloseTransactionAsMutation(historyi.TransactionPolicyPassive)
				if err != nil {
					return nil, err
				}
				return mutation.ExecutionInfo, err
			},
		},
		{
			name: "CloseTransactionAsMutation_HistoryEvents",
			dbStateMutationFn: func(dbState *persistencespb.WorkflowMutableState) {
				dbState.BufferedEvents = nil
			},
			txFunc: func(ms historyi.MutableState) (*persistencespb.WorkflowExecutionInfo, error) {
				completWorkflowTaskFn(ms)

				mutation, _, err := ms.CloseTransactionAsMutation(historyi.TransactionPolicyActive)
				if err != nil {
					return nil, err
				}
				return mutation.ExecutionInfo, err
			},
		},
		{
			name: "CloseTransactionAsMutation_BufferedEvents",
			dbStateMutationFn: func(dbState *persistencespb.WorkflowMutableState) {
				dbState.BufferedEvents = nil
			},
			txFunc: func(ms historyi.MutableState) (*persistencespb.WorkflowExecutionInfo, error) {
				var activityScheduleEventID int64
				for activityScheduleEventID = range s.mutableState.GetPendingActivityInfos() {
					break
				}
				_, err := s.mutableState.AddActivityTaskTimedOutEvent(
					activityScheduleEventID,
					common.EmptyEventID,
					failure.NewTimeoutFailure("test-timeout", enumspb.TIMEOUT_TYPE_SCHEDULE_TO_START),
					enumspb.RETRY_STATE_TIMEOUT,
				)
				s.NoError(err)

				mutation, _, err := ms.CloseTransactionAsMutation(historyi.TransactionPolicyActive)
				if err != nil {
					return nil, err
				}
				return mutation.ExecutionInfo, err
			},
		},
		{
			name: "CloseTransactionAsMutation_SyncActivity",
			dbStateMutationFn: func(dbState *persistencespb.WorkflowMutableState) {
				dbState.BufferedEvents = nil
			},
			txFunc: func(ms historyi.MutableState) (*persistencespb.WorkflowExecutionInfo, error) {
				for _, ai := range ms.GetPendingActivityInfos() {
					ms.UpdateActivityProgress(ai, &workflowservice.RecordActivityTaskHeartbeatRequest{})
					break
				}

				mutation, _, err := ms.CloseTransactionAsMutation(historyi.TransactionPolicyActive)
				if err != nil {
					return nil, err
				}
				return mutation.ExecutionInfo, err
			},
		},
		{
			name: "CloseTransactionAsMutation_DirtyStateMachine",
			dbStateMutationFn: func(dbState *persistencespb.WorkflowMutableState) {
				dbState.BufferedEvents = nil
			},
			txFunc: func(ms historyi.MutableState) (*persistencespb.WorkflowExecutionInfo, error) {
				root := ms.HSM()
				err := hsm.MachineTransition(root, func(*MutableStateImpl) (hsm.TransitionOutput, error) {
					return hsm.TransitionOutput{}, nil
				})
				s.NoError(err)
				s.True(root.Dirty())

				mutation, _, err := ms.CloseTransactionAsMutation(historyi.TransactionPolicyActive)
				if err != nil {
					return nil, err
				}
				return mutation.ExecutionInfo, err
			},
		},
		{
			name: "CloseTransactionAsSnapshot",
			dbStateMutationFn: func(dbState *persistencespb.WorkflowMutableState) {
				dbState.BufferedEvents = nil
			},
			txFunc: func(ms historyi.MutableState) (*persistencespb.WorkflowExecutionInfo, error) {
				completWorkflowTaskFn(ms)

				mutation, _, err := ms.CloseTransactionAsSnapshot(historyi.TransactionPolicyActive)
				if err != nil {
					return nil, err
				}
				return mutation.ExecutionInfo, err
			},
		},
		// TODO: add a test for flushing buffered events using last event version.
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			dbState := s.buildWorkflowMutableState()
			if tc.dbStateMutationFn != nil {
				tc.dbStateMutationFn(dbState)
			}

			var err error
			s.mutableState, err = NewMutableStateFromDB(s.mockShard, s.mockEventsCache, s.logger, namespaceEntry, dbState, 123)
			s.NoError(err)
			err = s.mutableState.UpdateCurrentVersion(namespaceEntry.FailoverVersion(), false)
			s.NoError(err)
			s.mutableState.transitionHistoryEnabled = false

			s.mockShard.Resource.ClusterMetadata.EXPECT().ClusterNameForFailoverVersion(
				namespaceEntry.IsGlobalNamespace(),
				namespaceEntry.FailoverVersion(),
			).Return(cluster.TestCurrentClusterName).AnyTimes()
			s.mockShard.Resource.ClusterMetadata.EXPECT().GetCurrentClusterName().Return(cluster.TestCurrentClusterName).AnyTimes()

			s.NotNil(s.mutableState.executionInfo.TransitionHistory)
			execInfo, err := tc.txFunc(s.mutableState)
			s.NotNil(execInfo.PreviousTransitionHistory)
			s.Nil(execInfo.TransitionHistory)
			s.Nil(err)
		})
	}
}

func (s *mutableStateSuite) getBuildIdsFromMutableState() []string {
	payload, found := s.mutableState.executionInfo.SearchAttributes[searchattribute.BuildIds]
	if !found {
		return []string{}
	}
	decoded, err := searchattribute.DecodeValue(payload, enumspb.INDEXED_VALUE_TYPE_KEYWORD_LIST, true)
	s.NoError(err)
	buildIDs, ok := decoded.([]string)
	s.True(ok)
	return buildIDs
}

// return reset points minus a few fields that are hard to check for equality
func (s *mutableStateSuite) cleanedResetPoints() *workflowpb.ResetPoints {
	out := common.CloneProto(s.mutableState.executionInfo.GetAutoResetPoints())
	for _, point := range out.GetPoints() {
		point.CreateTime = nil // current time
		point.ExpireTime = nil
	}
	return out
}

func (s *mutableStateSuite) TestCollapseVisibilityTasks() {
	testCases := []struct {
		name  string
		tasks []tasks.Task
		res   []enumsspb.TaskType
	}{
		{
			name: "start upsert close delete",
			tasks: []tasks.Task{
				&tasks.StartExecutionVisibilityTask{},
				&tasks.UpsertExecutionVisibilityTask{},
				&tasks.UpsertExecutionVisibilityTask{},
				&tasks.CloseExecutionVisibilityTask{},
				&tasks.DeleteExecutionVisibilityTask{},
			},
			res: []enumsspb.TaskType{
				enumsspb.TASK_TYPE_VISIBILITY_DELETE_EXECUTION,
			},
		},
		{
			name: "upsert close delete",
			tasks: []tasks.Task{
				&tasks.UpsertExecutionVisibilityTask{},
				&tasks.UpsertExecutionVisibilityTask{},
				&tasks.CloseExecutionVisibilityTask{},
				&tasks.DeleteExecutionVisibilityTask{},
			},
			res: []enumsspb.TaskType{
				enumsspb.TASK_TYPE_VISIBILITY_DELETE_EXECUTION,
			},
		},
		{
			name: "close delete",
			tasks: []tasks.Task{
				&tasks.CloseExecutionVisibilityTask{},
				&tasks.DeleteExecutionVisibilityTask{},
			},
			res: []enumsspb.TaskType{
				enumsspb.TASK_TYPE_VISIBILITY_DELETE_EXECUTION,
			},
		},
		{
			name: "delete",
			tasks: []tasks.Task{
				&tasks.DeleteExecutionVisibilityTask{},
			},
			res: []enumsspb.TaskType{
				enumsspb.TASK_TYPE_VISIBILITY_DELETE_EXECUTION,
			},
		},
		{
			name: "start upsert close",
			tasks: []tasks.Task{
				&tasks.StartExecutionVisibilityTask{},
				&tasks.UpsertExecutionVisibilityTask{},
				&tasks.UpsertExecutionVisibilityTask{},
				&tasks.CloseExecutionVisibilityTask{},
			},
			res: []enumsspb.TaskType{
				enumsspb.TASK_TYPE_VISIBILITY_CLOSE_EXECUTION,
			},
		},
		{
			name: "upsert close",
			tasks: []tasks.Task{
				&tasks.UpsertExecutionVisibilityTask{},
				&tasks.UpsertExecutionVisibilityTask{},
				&tasks.CloseExecutionVisibilityTask{},
			},
			res: []enumsspb.TaskType{
				enumsspb.TASK_TYPE_VISIBILITY_CLOSE_EXECUTION,
			},
		},
		{
			name: "close",
			tasks: []tasks.Task{
				&tasks.CloseExecutionVisibilityTask{},
			},
			res: []enumsspb.TaskType{
				enumsspb.TASK_TYPE_VISIBILITY_CLOSE_EXECUTION,
			},
		},
		{
			name: "start upsert",
			tasks: []tasks.Task{
				&tasks.StartExecutionVisibilityTask{},
				&tasks.UpsertExecutionVisibilityTask{},
				&tasks.UpsertExecutionVisibilityTask{},
			},
			res: []enumsspb.TaskType{
				enumsspb.TASK_TYPE_VISIBILITY_UPSERT_EXECUTION,
			},
		},
		{
			name: "upsert",
			tasks: []tasks.Task{
				&tasks.UpsertExecutionVisibilityTask{},
				&tasks.UpsertExecutionVisibilityTask{},
			},
			res: []enumsspb.TaskType{
				enumsspb.TASK_TYPE_VISIBILITY_UPSERT_EXECUTION,
			},
		},
		{
			name: "start",
			tasks: []tasks.Task{
				&tasks.StartExecutionVisibilityTask{},
			},
			res: []enumsspb.TaskType{
				enumsspb.TASK_TYPE_VISIBILITY_START_EXECUTION,
			},
		},
		{
			name: "upsert start delete close",
			tasks: []tasks.Task{
				&tasks.UpsertExecutionVisibilityTask{},
				&tasks.StartExecutionVisibilityTask{},
				&tasks.DeleteExecutionVisibilityTask{},
				&tasks.CloseExecutionVisibilityTask{},
			},
			res: []enumsspb.TaskType{
				enumsspb.TASK_TYPE_VISIBILITY_DELETE_EXECUTION,
			},
		},
		{
			name: "close upsert",
			tasks: []tasks.Task{
				&tasks.CloseExecutionVisibilityTask{},
				&tasks.UpsertExecutionVisibilityTask{},
			},
			res: []enumsspb.TaskType{
				enumsspb.TASK_TYPE_VISIBILITY_CLOSE_EXECUTION,
			},
		},
	}

	ms := s.mutableState

	for _, tc := range testCases {
		s.Run(
			tc.name,
			func() {
				ms.InsertTasks[tasks.CategoryVisibility] = []tasks.Task{}
				ms.AddTasks(tc.tasks...)
				ms.closeTransactionCollapseVisibilityTasks()
				visTasks := ms.InsertTasks[tasks.CategoryVisibility]
				s.Equal(len(tc.res), len(visTasks))
				for i, expectTaskType := range tc.res {
					s.Equal(expectTaskType, visTasks[i].GetType())
				}
			},
		)
	}
}

func (s *mutableStateSuite) TestStartChildWorkflowRequestID() {
	workflowTaskCompletionEventID := rand.Int63()
	attributes := &commandpb.StartChildWorkflowExecutionCommandAttributes{}
	event := s.mutableState.hBuilder.AddStartChildWorkflowExecutionInitiatedEvent(
		workflowTaskCompletionEventID,
		attributes,
		tests.NamespaceID,
	)
	createRequestID := fmt.Sprintf("%s:%d:%d", s.mutableState.executionState.RunId, event.GetEventId(), event.GetVersion())
	s.mockEventsCache.EXPECT().PutEvent(gomock.Any(), gomock.Any()).AnyTimes()

	ci, err := s.mutableState.ApplyStartChildWorkflowExecutionInitiatedEvent(
		workflowTaskCompletionEventID,
		event,
	)
	s.NoError(err)
	s.Equal(createRequestID, ci.CreateRequestId)
}

func (s *mutableStateSuite) TestGetCloseVersion() {
	s.mockEventsCache.EXPECT().PutEvent(gomock.Any(), gomock.Any()).AnyTimes()

	_, err := s.mutableState.AddWorkflowExecutionStartedEvent(
		&commonpb.WorkflowExecution{
			WorkflowId: tests.WorkflowID,
			RunId:      tests.RunID,
		},
		&historyservice.StartWorkflowExecutionRequest{
			StartRequest: &workflowservice.StartWorkflowExecutionRequest{},
		},
	)
	s.NoError(err)
	_, err = s.mutableState.AddWorkflowTaskScheduledEvent(false, enumsspb.WORKFLOW_TASK_TYPE_NORMAL)
	s.NoError(err)
	_, _, err = s.mutableState.CloseTransactionAsMutation(historyi.TransactionPolicyActive)
	s.NoError(err)

	_, err = s.mutableState.GetCloseVersion()
	s.Error(err) // workflow still open

	namespaceEntry, err := s.mockShard.GetNamespaceRegistry().GetNamespaceByID(tests.NamespaceID)
	s.NoError(err)
	expectedVersion := namespaceEntry.FailoverVersion()

	_, err = s.mutableState.AddCompletedWorkflowEvent(
		5,
		&commandpb.CompleteWorkflowExecutionCommandAttributes{},
		"",
	)
	s.NoError(err)
	// get close version in the transaction that closes the workflow
	closeVersion, err := s.mutableState.GetCloseVersion()
	s.NoError(err)
	s.Equal(expectedVersion, closeVersion)

	_, _, err = s.mutableState.CloseTransactionAsMutation(historyi.TransactionPolicyActive)
	s.NoError(err)

	// get close version after workflow is closed
	closeVersion, err = s.mutableState.GetCloseVersion()
	s.NoError(err)
	s.Equal(expectedVersion, closeVersion)

	// verify close version doesn't change after workflow is closed
	err = s.mutableState.UpdateCurrentVersion(12345, true)
	s.NoError(err)
	closeVersion, err = s.mutableState.GetCloseVersion()
	s.NoError(err)
	s.Equal(expectedVersion, closeVersion)
}

func (s *mutableStateSuite) TestCloseTransactionPrepareReplicationTasks_HistoryTask() {
	version := int64(777)
	firstEventID := int64(2)
	lastEventID := int64(3)
	now := time.Now().UTC()
	taskqueue := "taskqueue for test"
	workflowTaskTimeout := 11 * time.Second
	workflowTaskAttempt := int32(1)
	eventBatches := [][]*historypb.HistoryEvent{
		{
			&historypb.HistoryEvent{
				Version:   version,
				EventId:   firstEventID,
				EventTime: timestamppb.New(now),
				EventType: enumspb.EVENT_TYPE_WORKFLOW_TASK_SCHEDULED,
				Attributes: &historypb.HistoryEvent_WorkflowTaskScheduledEventAttributes{WorkflowTaskScheduledEventAttributes: &historypb.WorkflowTaskScheduledEventAttributes{
					TaskQueue:           &taskqueuepb.TaskQueue{Name: taskqueue},
					StartToCloseTimeout: durationpb.New(workflowTaskTimeout),
					Attempt:             workflowTaskAttempt,
				}},
			},
		},
		{
			&historypb.HistoryEvent{
				Version:   version,
				EventId:   lastEventID,
				EventTime: timestamppb.New(now),
				EventType: enumspb.EVENT_TYPE_WORKFLOW_TASK_STARTED,
				Attributes: &historypb.HistoryEvent_WorkflowTaskStartedEventAttributes{WorkflowTaskStartedEventAttributes: &historypb.WorkflowTaskStartedEventAttributes{
					ScheduledEventId: firstEventID,
					RequestId:        uuid.New(),
				}},
			},
		},
	}

	testCases := []struct {
		name                       string
		replicationMultipleBatches bool
		tasks                      []tasks.Task
	}{
		{
			name:                       "multiple event batches disabled",
			replicationMultipleBatches: false,
			tasks: []tasks.Task{
				&tasks.HistoryReplicationTask{
					WorkflowKey:  s.mutableState.GetWorkflowKey(),
					FirstEventID: firstEventID,
					NextEventID:  firstEventID + 1,
					Version:      version,
				},
				&tasks.HistoryReplicationTask{
					WorkflowKey:  s.mutableState.GetWorkflowKey(),
					FirstEventID: lastEventID,
					NextEventID:  lastEventID + 1,
					Version:      version,
				},
			},
		},
		{
			name:                       "multiple event batches enabled",
			replicationMultipleBatches: true,
			tasks: []tasks.Task{
				&tasks.HistoryReplicationTask{
					WorkflowKey:  s.mutableState.GetWorkflowKey(),
					FirstEventID: firstEventID,
					NextEventID:  lastEventID + 1,
					Version:      version,
				},
			},
		},
	}

	ms := s.mutableState
	ms.transitionHistoryEnabled = false
	for _, tc := range testCases {
		s.Run(
			tc.name,
			func() {
				if s.replicationMultipleBatches != tc.replicationMultipleBatches {
					return
				}
				ms.InsertTasks[tasks.CategoryReplication] = []tasks.Task{}
				err := ms.closeTransactionPrepareReplicationTasks(historyi.TransactionPolicyActive, eventBatches, false)
				if err != nil {
					s.Fail("closeTransactionPrepareReplicationTasks failed", err)
				}
				repicationTasks := ms.InsertTasks[tasks.CategoryReplication]
				s.Equal(len(tc.tasks), len(repicationTasks))
				for i, task := range tc.tasks {
					s.Equal(task, repicationTasks[i])
				}
			},
		)
	}
}

func (s *mutableStateSuite) TestCloseTransactionPrepareReplicationTasks_SyncVersionedTransitionTask() {
	if s.replicationMultipleBatches == true {
		return
	}
	version := int64(777)
	firstEventID := int64(2)
	lastEventID := int64(3)
	now := time.Now().UTC()
	taskqueue := "taskqueue for test"
	workflowTaskTimeout := 11 * time.Second
	workflowTaskAttempt := int32(1)
	eventBatches := [][]*historypb.HistoryEvent{
		{
			&historypb.HistoryEvent{
				Version:   version,
				EventId:   firstEventID,
				EventTime: timestamppb.New(now),
				EventType: enumspb.EVENT_TYPE_WORKFLOW_TASK_SCHEDULED,
				Attributes: &historypb.HistoryEvent_WorkflowTaskScheduledEventAttributes{WorkflowTaskScheduledEventAttributes: &historypb.WorkflowTaskScheduledEventAttributes{
					TaskQueue:           &taskqueuepb.TaskQueue{Name: taskqueue},
					StartToCloseTimeout: durationpb.New(workflowTaskTimeout),
					Attempt:             workflowTaskAttempt,
				}},
			},
		},
		{
			&historypb.HistoryEvent{
				Version:   version,
				EventId:   lastEventID,
				EventTime: timestamppb.New(now),
				EventType: enumspb.EVENT_TYPE_WORKFLOW_TASK_STARTED,
				Attributes: &historypb.HistoryEvent_WorkflowTaskStartedEventAttributes{WorkflowTaskStartedEventAttributes: &historypb.WorkflowTaskStartedEventAttributes{
					ScheduledEventId: firstEventID,
					RequestId:        uuid.New(),
				}},
			},
		},
	}

	ms := s.mutableState
	ms.transitionHistoryEnabled = true
	ms.syncActivityTasks[1] = struct{}{}
	ms.pendingActivityInfoIDs[1] = &persistencespb.ActivityInfo{
		Version:          version,
		ScheduledEventId: 1,
	}
	ms.InsertTasks[tasks.CategoryReplication] = []tasks.Task{}
	transitionHistory := []*persistencespb.VersionedTransition{
		{
			NamespaceFailoverVersion: 1,
			TransitionCount:          10,
		},
	}
	ms.executionInfo.TransitionHistory = transitionHistory
	err := ms.closeTransactionPrepareReplicationTasks(historyi.TransactionPolicyActive, eventBatches, false)
	s.NoError(err)
	replicationTasks := ms.InsertTasks[tasks.CategoryReplication]
	s.Equal(1, len(replicationTasks))
	historyTasks := []tasks.Task{
		&tasks.HistoryReplicationTask{
			WorkflowKey:  s.mutableState.GetWorkflowKey(),
			FirstEventID: firstEventID,
			NextEventID:  firstEventID + 1,
			Version:      version,
		},
		&tasks.HistoryReplicationTask{
			WorkflowKey:  s.mutableState.GetWorkflowKey(),
			FirstEventID: lastEventID,
			NextEventID:  lastEventID + 1,
			Version:      version,
		},
	}
	expectedTask := &tasks.SyncVersionedTransitionTask{
		WorkflowKey:         s.mutableState.GetWorkflowKey(),
		VisibilityTimestamp: now,
		Priority:            enumsspb.TASK_PRIORITY_HIGH,
		VersionedTransition: transitionHistory[0],
		FirstEventID:        firstEventID,
		NextEventID:         lastEventID + 1,
	}
	s.Equal(enumsspb.TASK_TYPE_REPLICATION_SYNC_VERSIONED_TRANSITION, replicationTasks[0].GetType())
	actualTask, ok := replicationTasks[0].(*tasks.SyncVersionedTransitionTask)
	s.True(ok)
	s.Equal(expectedTask.WorkflowKey, actualTask.WorkflowKey)
	s.Equal(expectedTask.VersionedTransition, actualTask.VersionedTransition)
	s.Equal(3, len(actualTask.TaskEquivalents))
	s.Equal(historyTasks[0], actualTask.TaskEquivalents[0])
	s.Equal(historyTasks[1], actualTask.TaskEquivalents[1])
	s.Equal(enumsspb.TASK_TYPE_REPLICATION_SYNC_ACTIVITY, actualTask.TaskEquivalents[2].GetType())
}

func (s *mutableStateSuite) TestMaxAllowedTimer() {
	testCases := []struct {
		name                   string
		runTimeout             time.Duration
		runTimeoutTimerDropped bool
	}{
		{
			name:                   "run timeout timer preserved",
			runTimeout:             time.Hour * 24 * 365,
			runTimeoutTimerDropped: false,
		},
		{
			name:                   "run timeout timer dropped",
			runTimeout:             time.Hour * 24 * 365 * 100,
			runTimeoutTimerDropped: true,
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			s.mockEventsCache.EXPECT().PutEvent(gomock.Any(), gomock.Any()).Times(1)
			s.mutableState = NewMutableState(s.mockShard, s.mockEventsCache, s.logger, s.namespaceEntry, tests.WorkflowID, tests.RunID, time.Now().UTC())

			workflowKey := s.mutableState.GetWorkflowKey()
			_, err := s.mutableState.AddWorkflowExecutionStartedEvent(
				&commonpb.WorkflowExecution{
					WorkflowId: workflowKey.WorkflowID,
					RunId:      workflowKey.RunID,
				},
				&historyservice.StartWorkflowExecutionRequest{
					NamespaceId: workflowKey.NamespaceID,
					StartRequest: &workflowservice.StartWorkflowExecutionRequest{
						Namespace:  s.mutableState.GetNamespaceEntry().Name().String(),
						WorkflowId: workflowKey.WorkflowID,
						WorkflowType: &commonpb.WorkflowType{
							Name: "test-workflow-type",
						},
						TaskQueue: &taskqueuepb.TaskQueue{
							Name: "test-task-queue",
						},
						WorkflowRunTimeout: durationpb.New(tc.runTimeout),
					},
				},
			)
			s.NoError(err)

			snapshot, _, err := s.mutableState.CloseTransactionAsSnapshot(historyi.TransactionPolicyActive)
			s.NoError(err)

			timerTasks := snapshot.Tasks[tasks.CategoryTimer]
			if tc.runTimeoutTimerDropped {
				s.Empty(timerTasks)
			} else {
				s.Len(timerTasks, 1)
				s.Equal(enumsspb.TASK_TYPE_WORKFLOW_RUN_TIMEOUT, timerTasks[0].GetType())
			}
		})
	}
}

func (s *mutableStateSuite) TestCloseTransactionPrepareReplicationTasks_SyncHSMTask() {
	version := s.mutableState.GetCurrentVersion()
	stateMachineDef := hsmtest.NewDefinition("test")
	err := s.mockShard.StateMachineRegistry().RegisterMachine(stateMachineDef)
	s.NoError(err)

	testCases := []struct {
		name                    string
		hsmEmpty                bool
		hsmDirty                bool
		eventBatches            [][]*historypb.HistoryEvent
		clearBufferEvents       bool
		expectedReplicationTask tasks.Task
	}{
		{
			name:     "WithEvents",
			hsmEmpty: false,
			hsmDirty: true,
			eventBatches: [][]*historypb.HistoryEvent{
				{
					&historypb.HistoryEvent{
						Version:   version,
						EventId:   5,
						EventTime: timestamppb.New(time.Now()),
						EventType: enumspb.EVENT_TYPE_NEXUS_OPERATION_SCHEDULED,
						Attributes: &historypb.HistoryEvent_NexusOperationScheduledEventAttributes{
							NexusOperationScheduledEventAttributes: &historypb.NexusOperationScheduledEventAttributes{},
						},
					},
				},
			},
			clearBufferEvents: false,
			expectedReplicationTask: &tasks.HistoryReplicationTask{
				WorkflowKey:  s.mutableState.GetWorkflowKey(),
				FirstEventID: 5,
				NextEventID:  6,
				Version:      version,
			},
		},
		{
			name:              "NoEvents",
			hsmEmpty:          false,
			hsmDirty:          true,
			eventBatches:      nil,
			clearBufferEvents: false,
			expectedReplicationTask: &tasks.SyncHSMTask{
				WorkflowKey: s.mutableState.GetWorkflowKey(),
			},
		},
		{
			name:                    "NoChildren/ClearBufferFalse",
			hsmEmpty:                true,
			hsmDirty:                false,
			eventBatches:            nil,
			clearBufferEvents:       false,
			expectedReplicationTask: nil,
		},
		{
			name:                    "NoChildren/ClearBufferTrue",
			hsmEmpty:                true,
			hsmDirty:                false,
			eventBatches:            nil,
			clearBufferEvents:       true,
			expectedReplicationTask: nil,
		},
		{
			name:                    "CleanChildren/ClearBufferFalse",
			hsmEmpty:                false,
			hsmDirty:                false,
			clearBufferEvents:       false,
			expectedReplicationTask: nil,
		},
		{
			name:              "CleanChildren/ClearBufferTrue",
			hsmEmpty:          false,
			hsmDirty:          false,
			clearBufferEvents: true,
			expectedReplicationTask: &tasks.SyncHSMTask{
				WorkflowKey: s.mutableState.GetWorkflowKey(),
			},
		},
	}

	for _, tc := range testCases {
		s.Run(
			tc.name,
			func() {
				if !tc.hsmEmpty {
					_, err = s.mutableState.HSM().AddChild(
						hsm.Key{Type: stateMachineDef.Type(), ID: "child_1"},
						hsmtest.NewData(hsmtest.State1),
					)
					s.NoError(err)

					if !tc.hsmDirty {
						s.mutableState.HSM().ClearTransactionState()
					}
				}
				s.mutableState.transitionHistoryEnabled = false
				err := s.mutableState.closeTransactionPrepareReplicationTasks(historyi.TransactionPolicyActive, tc.eventBatches, tc.clearBufferEvents)
				s.NoError(err)

				repicationTasks := s.mutableState.PopTasks()[tasks.CategoryReplication]

				if tc.expectedReplicationTask != nil {
					s.Len(repicationTasks, 1)
					s.Equal(tc.expectedReplicationTask, repicationTasks[0])
				} else {
					s.Empty(repicationTasks)
				}
			},
		)
	}
}

func (s *mutableStateSuite) setDisablingTransitionHistory(ms *MutableStateImpl) {
	ms.versionedTransitionInDB = &persistencespb.VersionedTransition{
		NamespaceFailoverVersion: s.namespaceEntry.FailoverVersion(),
		TransitionCount:          1025,
	}
	ms.executionInfo.TransitionHistory = nil
}

func (s *mutableStateSuite) TestCloseTransactionPrepareReplicationTasks_SyncActivityTask() {
	testCases := []struct {
		name                       string
		disablingTransitionHistory bool
		expectedReplicationTask    []tasks.SyncActivityTask
	}{
		{
			name:                       "NoDisablingTransitionHistory",
			disablingTransitionHistory: false,
			expectedReplicationTask: []tasks.SyncActivityTask{
				{
					ScheduledEventID: 100,
				},
			},
		},
		{
			name:                       "DisablingTransitionHistory",
			disablingTransitionHistory: true,
			expectedReplicationTask: []tasks.SyncActivityTask{
				{
					ScheduledEventID: 90,
				},
				{
					ScheduledEventID: 100,
				},
			},
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			dbState := s.buildWorkflowMutableState()
			dbState.ActivityInfos[100] = &persistencespb.ActivityInfo{
				ScheduledEventId: 100,
			}
			ms, err := NewMutableStateFromDB(s.mockShard, s.mockEventsCache, s.logger, s.namespaceEntry, dbState, 123)
			s.NoError(err)

			if tc.disablingTransitionHistory {
				s.setDisablingTransitionHistory(ms)
			}

			ms.UpdateActivityProgress(ms.pendingActivityInfoIDs[100], &workflowservice.RecordActivityTaskHeartbeatRequest{})

			repicationTasks := ms.syncActivityToReplicationTask(historyi.TransactionPolicyActive)
			s.Len(repicationTasks, len(tc.expectedReplicationTask))
			sort.Slice(repicationTasks, func(i, j int) bool {
				return repicationTasks[i].(*tasks.SyncActivityTask).ScheduledEventID < repicationTasks[j].(*tasks.SyncActivityTask).ScheduledEventID
			})
			for i, task := range tc.expectedReplicationTask {
				s.Equal(task.ScheduledEventID, repicationTasks[i].(*tasks.SyncActivityTask).ScheduledEventID)
			}
		})
	}
}

func (s *mutableStateSuite) TestVersionedTransitionInDB() {
	// case 1: versionedTransitionInDB is not nil
	dbState := s.buildWorkflowMutableState()
	ms, err := NewMutableStateFromDB(s.mockShard, s.mockEventsCache, s.logger, s.namespaceEntry, dbState, 123)
	s.NoError(err)

	s.True(proto.Equal(ms.CurrentVersionedTransition(), ms.versionedTransitionInDB))

	s.NoError(ms.cleanupTransaction())
	s.True(proto.Equal(ms.CurrentVersionedTransition(), ms.versionedTransitionInDB))

	ms.executionInfo.TransitionHistory = nil
	s.NoError(ms.cleanupTransaction())
	s.Nil(ms.versionedTransitionInDB)

	// case 2: versionedTransitionInDB is nil
	dbState = s.buildWorkflowMutableState()
	dbState.ExecutionInfo.TransitionHistory = nil
	ms, err = NewMutableStateFromDB(s.mockShard, s.mockEventsCache, s.logger, s.namespaceEntry, dbState, 123)
	s.NoError(err)

	s.Nil(ms.versionedTransitionInDB)

	ms.executionInfo.TransitionHistory = UpdatedTransitionHistory(ms.executionInfo.TransitionHistory, s.namespaceEntry.FailoverVersion())
	s.NoError(ms.cleanupTransaction())
	s.True(proto.Equal(ms.CurrentVersionedTransition(), ms.versionedTransitionInDB))
}

func (s *mutableStateSuite) TestCloseTransactionTrackTombstones() {
	testCases := []struct {
		name        string
		tombstoneFn func(ms historyi.MutableState) (*persistencespb.StateMachineTombstone, error)
	}{
		{
			name: "Activity",
			tombstoneFn: func(mutableState historyi.MutableState) (*persistencespb.StateMachineTombstone, error) {
				var activityScheduleEventID int64
				for activityScheduleEventID = range mutableState.GetPendingActivityInfos() {
					break
				}
				_, err := mutableState.AddActivityTaskTimedOutEvent(
					activityScheduleEventID,
					common.EmptyEventID,
					failure.NewTimeoutFailure("test-timeout", enumspb.TIMEOUT_TYPE_SCHEDULE_TO_START),
					enumspb.RETRY_STATE_TIMEOUT,
				)
				return &persistencespb.StateMachineTombstone{
					StateMachineKey: &persistencespb.StateMachineTombstone_ActivityScheduledEventId{
						ActivityScheduledEventId: activityScheduleEventID,
					},
				}, err
			},
		},
		{
			name: "UserTimer",
			tombstoneFn: func(mutableState historyi.MutableState) (*persistencespb.StateMachineTombstone, error) {
				var timerID string
				for timerID = range mutableState.GetPendingTimerInfos() {
					break
				}
				_, err := mutableState.AddTimerFiredEvent(timerID)
				return &persistencespb.StateMachineTombstone{
					StateMachineKey: &persistencespb.StateMachineTombstone_TimerId{
						TimerId: timerID,
					},
				}, err
			},
		},
		{
			name: "ChildWorkflow",
			tombstoneFn: func(mutableState historyi.MutableState) (*persistencespb.StateMachineTombstone, error) {
				var initiatedEventId int64
				var ci *persistencespb.ChildExecutionInfo
				for initiatedEventId, ci = range mutableState.GetPendingChildExecutionInfos() {
					break
				}
				childExecution := &commonpb.WorkflowExecution{
					WorkflowId: uuid.New(),
					RunId:      uuid.New(),
				}
				_, err := mutableState.AddChildWorkflowExecutionStartedEvent(
					childExecution,
					&commonpb.WorkflowType{Name: ci.WorkflowTypeName},
					initiatedEventId,
					nil,
					nil,
				)
				if err != nil {
					return nil, err
				}
				_, err = mutableState.AddChildWorkflowExecutionTerminatedEvent(
					initiatedEventId,
					childExecution,
				)
				return &persistencespb.StateMachineTombstone{
					StateMachineKey: &persistencespb.StateMachineTombstone_ChildExecutionInitiatedEventId{
						ChildExecutionInitiatedEventId: initiatedEventId,
					},
				}, err
			},
		},
		{
			name: "RequestCancelExternal",
			tombstoneFn: func(mutableState historyi.MutableState) (*persistencespb.StateMachineTombstone, error) {
				var initiatedEventId int64
				for initiatedEventId = range mutableState.GetPendingRequestCancelExternalInfos() {
					break
				}
				_, err := mutableState.AddRequestCancelExternalWorkflowExecutionFailedEvent(
					initiatedEventId,
					s.namespaceEntry.Name(),
					s.namespaceEntry.ID(),
					uuid.New(),
					uuid.New(),
					enumspb.CANCEL_EXTERNAL_WORKFLOW_EXECUTION_FAILED_CAUSE_EXTERNAL_WORKFLOW_EXECUTION_NOT_FOUND,
				)
				return &persistencespb.StateMachineTombstone{
					StateMachineKey: &persistencespb.StateMachineTombstone_RequestCancelInitiatedEventId{
						RequestCancelInitiatedEventId: initiatedEventId,
					},
				}, err
			},
		},
		{
			name: "SignalExternal",
			tombstoneFn: func(mutableState historyi.MutableState) (*persistencespb.StateMachineTombstone, error) {
				var initiatedEventId int64
				for initiatedEventId = range mutableState.GetPendingSignalExternalInfos() {
					break
				}
				_, err := mutableState.AddSignalExternalWorkflowExecutionFailedEvent(
					initiatedEventId,
					s.namespaceEntry.Name(),
					s.namespaceEntry.ID(),
					uuid.New(),
					uuid.New(),
					"",
					enumspb.SIGNAL_EXTERNAL_WORKFLOW_EXECUTION_FAILED_CAUSE_EXTERNAL_WORKFLOW_EXECUTION_NOT_FOUND,
				)
				return &persistencespb.StateMachineTombstone{
					StateMachineKey: &persistencespb.StateMachineTombstone_SignalExternalInitiatedEventId{
						SignalExternalInitiatedEventId: initiatedEventId,
					},
				}, err
			},
		},
		{
			name: "CHASM",
			tombstoneFn: func(mutableState historyi.MutableState) (*persistencespb.StateMachineTombstone, error) {
				deletedNodePath := "deleted-node-path"
				tombstone := &persistencespb.StateMachineTombstone{
					StateMachineKey: &persistencespb.StateMachineTombstone_ChasmNodePath{
						ChasmNodePath: deletedNodePath,
					},
				}

				mockChasmTree := historyi.NewMockChasmTree(s.controller)
				mockChasmTree.EXPECT().Archetype().Return(chasm.Archetype("mock-archetype")).AnyTimes()
				gomock.InOrder(
					mockChasmTree.EXPECT().IsStateDirty().Return(true).AnyTimes(),
					mockChasmTree.EXPECT().CloseTransaction().Return(chasm.NodesMutation{
						DeletedNodes: map[string]struct{}{deletedNodePath: {}},
					}, nil),
				)
				mutableState.(*MutableStateImpl).chasmTree = mockChasmTree

				return tombstone, nil
			},
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			dbState := s.buildWorkflowMutableState()

			mutableState, err := NewMutableStateFromDB(s.mockShard, s.mockEventsCache, s.logger, s.namespaceEntry, dbState, 123)
			s.NoError(err)

			currentVersionedTransition := mutableState.CurrentVersionedTransition()
			newVersionedTranstion := common.CloneProto(currentVersionedTransition)
			newVersionedTranstion.TransitionCount += 1

			_, err = mutableState.StartTransaction(s.namespaceEntry)
			s.NoError(err)

			expectedTombstone, err := tc.tombstoneFn(mutableState)
			s.NoError(err)

			_, _, err = mutableState.CloseTransactionAsMutation(historyi.TransactionPolicyActive)
			s.NoError(err)

			tombstoneBatches := mutableState.GetExecutionInfo().SubStateMachineTombstoneBatches
			s.Len(tombstoneBatches, 1)
			tombstoneBatch := tombstoneBatches[0]
			protorequire.ProtoEqual(s.T(), newVersionedTranstion, tombstoneBatch.VersionedTransition)
			s.True(tombstoneExists(tombstoneBatch.StateMachineTombstones, expectedTombstone))
		})
	}
}

func tombstoneExists(
	tombstones []*persistencespb.StateMachineTombstone,
	expectedTombstone *persistencespb.StateMachineTombstone,
) bool {
	for _, tombstone := range tombstones {
		if tombstone.Equal(expectedTombstone) {
			return true
		}
	}
	return false
}

func (s *mutableStateSuite) TestCloseTransactionTrackTombstones_CapIfLargerThanLimit() {
	dbState := s.buildWorkflowMutableState()

	mutableState, err := NewMutableStateFromDB(s.mockShard, s.mockEventsCache, s.logger, s.namespaceEntry, dbState, 123)
	s.NoError(err)
	mutableState.executionInfo.SubStateMachineTombstoneBatches = []*persistencespb.StateMachineTombstoneBatch{
		{
			VersionedTransition: &persistencespb.VersionedTransition{
				NamespaceFailoverVersion: s.namespaceEntry.FailoverVersion(),
				TransitionCount:          1,
			},
		},
	}

	currentVersionedTransition := mutableState.CurrentVersionedTransition()
	newVersionedTranstion := common.CloneProto(currentVersionedTransition)
	newVersionedTranstion.TransitionCount += 1
	signalMap := mutableState.GetPendingSignalExternalInfos()
	for i := 0; i < s.mockConfig.MutableStateTombstoneCountLimit(); i++ {
		signalMap[int64(76+i)] = &persistencespb.SignalInfo{

			Version:               s.namespaceEntry.FailoverVersion(),
			InitiatedEventId:      int64(76 + i),
			InitiatedEventBatchId: 17,
			RequestId:             uuid.New(),
		}
	}

	_, err = mutableState.StartTransaction(s.namespaceEntry)
	s.NoError(err)
	var initiatedEventId int64
	for initiatedEventId = range signalMap {
		_, err := mutableState.AddSignalExternalWorkflowExecutionFailedEvent(
			initiatedEventId,
			s.namespaceEntry.Name(),
			s.namespaceEntry.ID(),
			uuid.New(),
			uuid.New(),
			"",
			enumspb.SIGNAL_EXTERNAL_WORKFLOW_EXECUTION_FAILED_CAUSE_EXTERNAL_WORKFLOW_EXECUTION_NOT_FOUND,
		)
		s.NoError(err)
	}

	_, _, err = mutableState.CloseTransactionAsMutation(historyi.TransactionPolicyActive)
	s.NoError(err)

	tombstoneBatches := mutableState.GetExecutionInfo().SubStateMachineTombstoneBatches
	s.Len(tombstoneBatches, 0)
}

func (s *mutableStateSuite) TestCloseTransactionTrackTombstones_OnlyTrackFirstEmpty() {
	dbState := s.buildWorkflowMutableState()

	mutableState, err := NewMutableStateFromDB(s.mockShard, s.mockEventsCache, s.logger, s.namespaceEntry, dbState, 123)
	s.NoError(err)
	mutableState.executionInfo.SubStateMachineTombstoneBatches = []*persistencespb.StateMachineTombstoneBatch{
		{
			VersionedTransition: &persistencespb.VersionedTransition{
				NamespaceFailoverVersion: s.namespaceEntry.FailoverVersion(),
				TransitionCount:          1,
			},
		},
	}

	currentVersionedTransition := mutableState.CurrentVersionedTransition()
	newVersionedTranstion := common.CloneProto(currentVersionedTransition)
	newVersionedTranstion.TransitionCount += 1

	_, err = mutableState.StartTransaction(s.namespaceEntry)
	s.NoError(err)

	_, _, err = mutableState.CloseTransactionAsMutation(historyi.TransactionPolicyActive)
	s.NoError(err)

	tombstoneBatches := mutableState.GetExecutionInfo().SubStateMachineTombstoneBatches
	s.Len(tombstoneBatches, 1)
	s.Equal(int64(1), tombstoneBatches[0].VersionedTransition.TransitionCount)
}

func (s *mutableStateSuite) TestCloseTransactionGenerateCHASMRetentionTask() {
	dbState := s.buildWorkflowMutableState()

	mutableState, err := NewMutableStateFromDB(s.mockShard, s.mockEventsCache, s.logger, s.namespaceEntry, dbState, 123)
	s.NoError(err)

	// First close transaction once to get rid of unrelated tasks like UserTimer and ActivityTimeout
	_, err = mutableState.StartTransaction(s.namespaceEntry)
	s.NoError(err)
	_, _, err = mutableState.CloseTransactionAsMutation(historyi.TransactionPolicyActive)
	s.NoError(err)

	// Switch to a mock CHASM tree
	mockChasmTree := historyi.NewMockChasmTree(s.controller)
	mutableState.chasmTree = mockChasmTree

	// Is workflow, should not generate retention task
	mockChasmTree.EXPECT().IsStateDirty().Return(true).AnyTimes()
	mockChasmTree.EXPECT().Archetype().Return(chasmworkflow.Archetype).Times(1)
	mockChasmTree.EXPECT().CloseTransaction().Return(chasm.NodesMutation{}, nil).AnyTimes()
	mutation, _, err := mutableState.CloseTransactionAsMutation(historyi.TransactionPolicyActive)
	s.NoError(err)
	s.Empty(mutation.Tasks[tasks.CategoryTimer])

	// Now make the mutable state non-workflow.
	mockChasmTree.EXPECT().Archetype().Return(chasm.Archetype("test-archetype")).Times(2) // One time for each CloseTransactionAsMutation call
	_, err = mutableState.UpdateWorkflowStateStatus(
		enumsspb.WORKFLOW_EXECUTION_STATE_COMPLETED,
		enumspb.WORKFLOW_EXECUTION_STATUS_COMPLETED,
	)
	s.NoError(err)
	mutation, _, err = mutableState.CloseTransactionAsMutation(historyi.TransactionPolicyActive)
	s.NoError(err)
	s.Len(mutation.Tasks[tasks.CategoryTimer], 1)
	s.Equal(enumsspb.TASK_TYPE_DELETE_HISTORY_EVENT, mutation.Tasks[tasks.CategoryTimer][0].GetType())

	// Already closed before, should not generate retention task again.
	mutation, _, err = mutableState.CloseTransactionAsMutation(historyi.TransactionPolicyActive)
	s.NoError(err)
	s.Empty(mutation.Tasks[tasks.CategoryTimer])
}

func (s *mutableStateSuite) TestExecutionInfoClone() {
	newInstance := reflect.New(reflect.TypeOf(s.mutableState.executionInfo).Elem()).Interface()
	clone, ok := newInstance.(*persistencespb.WorkflowExecutionInfo)
	if !ok {
		s.T().Fatal("type assertion to *persistencespb.WorkflowExecutionInfo failed")
	}
	clone.NamespaceId = "namespace-id"
	clone.WorkflowId = "workflow-id"
	err := common.MergeProtoExcludingFields(s.mutableState.executionInfo, clone, func(v any) []interface{} {
		info, ok := v.(*persistencespb.WorkflowExecutionInfo)
		if !ok || info == nil {
			return nil
		}
		return []interface{}{
			&info.NamespaceId,
		}
	})
	s.Nil(err)
}

func (s *mutableStateSuite) addChangesForStateReplication(state *persistencespb.WorkflowMutableState) {
	// These fields will be updated during ApplySnapshot
	proto.Merge(state.ExecutionInfo, &persistencespb.WorkflowExecutionInfo{
		LastUpdateTime: timestamp.TimeNowPtrUtc(),
	})
	proto.Merge(state.ExecutionState, &persistencespb.WorkflowExecutionState{
		State: enumsspb.WORKFLOW_EXECUTION_STATE_CREATED,
	})

	state.ActivityInfos[90].TimerTaskStatus = TimerTaskStatusCreated
	state.TimerInfos["25"].ExpiryTime = timestamp.TimeNowPtrUtcAddDuration(time.Hour)
	state.ChildExecutionInfos[80].StartedEventId = 84
	state.RequestCancelInfos[70].CancelRequestId = uuid.New()
	state.SignalInfos[75].RequestId = uuid.New()

	// These infos will be deleted during ApplySnapshot
	state.ActivityInfos[89] = &persistencespb.ActivityInfo{}
	state.TimerInfos["to-be-deleted"] = &persistencespb.TimerInfo{}
	state.ChildExecutionInfos[79] = &persistencespb.ChildExecutionInfo{}
	state.RequestCancelInfos[69] = &persistencespb.RequestCancelInfo{}
	state.SignalInfos[74] = &persistencespb.SignalInfo{}
	state.SignalRequestedIds = []string{"to-be-deleted"}
}

func compareMapOfProto[K comparable, V proto.Message](s *mutableStateSuite, expected, actual map[K]V) {
	s.Equal(len(expected), len(actual))
	for k, v := range expected {
		s.True(proto.Equal(v, actual[k]))
	}
}

func (s *mutableStateSuite) verifyChildExecutionInfos(expectedMap, actualMap, originMap map[int64]*persistencespb.ChildExecutionInfo) {
	s.Equal(len(expectedMap), len(actualMap))
	for k, expected := range expectedMap {
		actual, ok := actualMap[k]
		s.True(ok)
		origin := originMap[k]

		s.Equal(expected.Version, actual.Version, "Version mismatch")
		s.Equal(expected.InitiatedEventBatchId, actual.InitiatedEventBatchId, "InitiatedEventBatchId mismatch")
		s.Equal(expected.StartedEventId, actual.StartedEventId, "StartedEventId mismatch")
		s.Equal(expected.StartedWorkflowId, actual.StartedWorkflowId, "StartedWorkflowId mismatch")
		s.Equal(expected.StartedRunId, actual.StartedRunId, "StartedRunId mismatch")
		s.Equal(expected.CreateRequestId, actual.CreateRequestId, "CreateRequestId mismatch")
		s.Equal(expected.Namespace, actual.Namespace, "Namespace mismatch")
		s.Equal(expected.WorkflowTypeName, actual.WorkflowTypeName, "WorkflowTypeName mismatch")
		s.Equal(expected.ParentClosePolicy, actual.ParentClosePolicy, "ParentClosePolicy mismatch")
		s.Equal(expected.InitiatedEventId, actual.InitiatedEventId, "InitiatedEventId mismatch")
		s.Equal(expected.NamespaceId, actual.NamespaceId, "NamespaceId mismatch")
		s.True(proto.Equal(expected.LastUpdateVersionedTransition, actual.LastUpdateVersionedTransition), "LastUpdateVersionedTransition mismatch")

		// special handled fields
		if origin != nil {
			s.Equal(origin.Clock, actual.Clock, "Clock mismatch")
		}
	}
}

func (s *mutableStateSuite) verifyActivityInfos(expectedMap, actualMap map[int64]*persistencespb.ActivityInfo) {
	s.Equal(len(expectedMap), len(actualMap))
	for k, expected := range expectedMap {
		actual, ok := actualMap[k]
		s.True(ok)

		s.Equal(expected.Version, actual.Version, "Version mismatch")
		s.Equal(expected.ScheduledEventBatchId, actual.ScheduledEventBatchId, "ScheduledEventBatchId mismatch")
		s.True(proto.Equal(expected.ScheduledTime, actual.ScheduledTime), "ScheduledTime mismatch")
		s.Equal(expected.StartedEventId, actual.StartedEventId, "StartedEventId mismatch")
		s.True(proto.Equal(expected.StartedTime, actual.StartedTime), "StartedTime mismatch")
		s.Equal(expected.ActivityId, actual.ActivityId, "ActivityId mismatch")
		s.Equal(expected.RequestId, actual.RequestId, "RequestId mismatch")
		s.True(proto.Equal(expected.ScheduleToStartTimeout, actual.ScheduleToStartTimeout), "ScheduleToStartTimeout mismatch")
		s.True(proto.Equal(expected.ScheduleToCloseTimeout, actual.ScheduleToCloseTimeout), "ScheduleToCloseTimeout mismatch")
		s.True(proto.Equal(expected.StartToCloseTimeout, actual.StartToCloseTimeout), "StartToCloseTimeout mismatch")
		s.True(proto.Equal(expected.HeartbeatTimeout, actual.HeartbeatTimeout), "HeartbeatTimeout mismatch")
		s.Equal(expected.CancelRequested, actual.CancelRequested, "CancelRequested mismatch")
		s.Equal(expected.CancelRequestId, actual.CancelRequestId, "CancelRequestId mismatch")
		s.Equal(expected.Attempt, actual.Attempt, "Attempt mismatch")
		s.Equal(expected.TaskQueue, actual.TaskQueue, "TaskQueue mismatch")
		s.Equal(expected.StartedIdentity, actual.StartedIdentity, "StartedIdentity mismatch")
		s.Equal(expected.HasRetryPolicy, actual.HasRetryPolicy, "HasRetryPolicy mismatch")
		s.True(proto.Equal(expected.RetryInitialInterval, actual.RetryInitialInterval), "RetryInitialInterval mismatch")
		s.True(proto.Equal(expected.RetryMaximumInterval, actual.RetryMaximumInterval), "RetryMaximumInterval mismatch")
		s.Equal(expected.RetryMaximumAttempts, actual.RetryMaximumAttempts, "RetryMaximumAttempts mismatch")
		s.True(proto.Equal(expected.RetryExpirationTime, actual.RetryExpirationTime), "RetryExpirationTime mismatch")
		s.Equal(expected.RetryBackoffCoefficient, actual.RetryBackoffCoefficient, "RetryBackoffCoefficient mismatch")
		s.Equal(expected.RetryNonRetryableErrorTypes, actual.RetryNonRetryableErrorTypes, "RetryNonRetryableErrorTypes mismatch")
		s.True(proto.Equal(expected.RetryLastFailure, actual.RetryLastFailure), "RetryLastFailure mismatch")
		s.Equal(expected.RetryLastWorkerIdentity, actual.RetryLastWorkerIdentity, "RetryLastWorkerIdentity mismatch")
		s.Equal(expected.ScheduledEventId, actual.ScheduledEventId, "ScheduledEventId mismatch")
		s.True(proto.Equal(expected.LastHeartbeatDetails, actual.LastHeartbeatDetails), "LastHeartbeatDetails mismatch")
		s.True(proto.Equal(expected.LastHeartbeatUpdateTime, actual.LastHeartbeatUpdateTime), "LastHeartbeatUpdateTime mismatch")
		s.Equal(expected.UseCompatibleVersion, actual.UseCompatibleVersion, "UseCompatibleVersion mismatch")
		s.True(proto.Equal(expected.ActivityType, actual.ActivityType), "ActivityType mismatch")
		s.True(proto.Equal(expected.LastWorkerVersionStamp, actual.LastWorkerVersionStamp), "LastWorkerVersionStamp mismatch")
		s.True(proto.Equal(expected.LastStartedDeployment, actual.LastStartedDeployment), "LastStartedDeployment mismatch")
		s.True(proto.Equal(expected.LastUpdateVersionedTransition, actual.LastUpdateVersionedTransition), "LastUpdateVersionedTransition mismatch")

		// special handled fields
		s.Equal(int32(TimerTaskStatusNone), actual.TimerTaskStatus, "TimerTaskStatus mismatch")
	}
}

func (s *mutableStateSuite) verifyExecutionInfo(current, target, origin *persistencespb.WorkflowExecutionInfo) {
	// These fields should not change.
	s.Equal(origin.WorkflowTaskVersion, current.WorkflowTaskVersion, "WorkflowTaskVersion mismatch")
	s.Equal(origin.WorkflowTaskScheduledEventId, current.WorkflowTaskScheduledEventId, "WorkflowTaskScheduledEventId mismatch")
	s.Equal(origin.WorkflowTaskStartedEventId, current.WorkflowTaskStartedEventId, "WorkflowTaskStartedEventId mismatch")
	s.Equal(origin.WorkflowTaskRequestId, current.WorkflowTaskRequestId, "WorkflowTaskRequestId mismatch")
	s.Equal(origin.WorkflowTaskTimeout, current.WorkflowTaskTimeout, "WorkflowTaskTimeout mismatch")
	s.Equal(origin.WorkflowTaskAttempt, current.WorkflowTaskAttempt, "WorkflowTaskAttempt mismatch")
	s.Equal(origin.WorkflowTaskStartedTime, current.WorkflowTaskStartedTime, "WorkflowTaskStartedTime mismatch")
	s.Equal(origin.WorkflowTaskScheduledTime, current.WorkflowTaskScheduledTime, "WorkflowTaskScheduledTime mismatch")
	s.Equal(origin.WorkflowTaskOriginalScheduledTime, current.WorkflowTaskOriginalScheduledTime, "WorkflowTaskOriginalScheduledTime mismatch")
	s.Equal(origin.WorkflowTaskType, current.WorkflowTaskType, "WorkflowTaskType mismatch")
	s.Equal(origin.WorkflowTaskSuggestContinueAsNew, current.WorkflowTaskSuggestContinueAsNew, "WorkflowTaskSuggestContinueAsNew mismatch")
	s.Equal(origin.WorkflowTaskHistorySizeBytes, current.WorkflowTaskHistorySizeBytes, "WorkflowTaskHistorySizeBytes mismatch")
	s.Equal(origin.WorkflowTaskBuildId, current.WorkflowTaskBuildId, "WorkflowTaskBuildId mismatch")
	s.Equal(origin.WorkflowTaskBuildIdRedirectCounter, current.WorkflowTaskBuildIdRedirectCounter, "WorkflowTaskBuildIdRedirectCounter mismatch")
	s.True(proto.Equal(origin.VersioningInfo, current.VersioningInfo), "VersioningInfo mismatch")
	s.True(proto.Equal(origin.VersionHistories, current.VersionHistories), "VersionHistories mismatch")
	s.True(proto.Equal(origin.ExecutionStats, current.ExecutionStats), "ExecutionStats mismatch")
	s.Equal(origin.LastFirstEventTxnId, current.LastFirstEventTxnId, "LastFirstEventTxnId mismatch")
	s.True(proto.Equal(origin.ParentClock, current.ParentClock), "ParentClock mismatch")
	s.Equal(origin.CloseTransferTaskId, current.CloseTransferTaskId, "CloseTransferTaskId mismatch")
	s.Equal(origin.CloseVisibilityTaskId, current.CloseVisibilityTaskId, "CloseVisibilityTaskId mismatch")
	s.Equal(origin.RelocatableAttributesRemoved, current.RelocatableAttributesRemoved, "RelocatableAttributesRemoved mismatch")
	s.Equal(origin.WorkflowExecutionTimerTaskStatus, current.WorkflowExecutionTimerTaskStatus, "WorkflowExecutionTimerTaskStatus mismatch")
	s.Equal(origin.SubStateMachinesByType, current.SubStateMachinesByType, "SubStateMachinesByType mismatch")
	s.Equal(origin.StateMachineTimers, current.StateMachineTimers, "StateMachineTimers mismatch")
	s.Equal(origin.TaskGenerationShardClockTimestamp, current.TaskGenerationShardClockTimestamp, "TaskGenerationShardClockTimestamp mismatch")
	s.Equal(origin.UpdateInfos, current.UpdateInfos, "UpdateInfos mismatch")

	// These fields should be updated.
	s.Equal(target.NamespaceId, current.NamespaceId, "NamespaceId mismatch")
	s.Equal(target.WorkflowId, current.WorkflowId, "WorkflowId mismatch")
	s.Equal(target.ParentNamespaceId, current.ParentNamespaceId, "ParentNamespaceId mismatch")
	s.Equal(target.ParentWorkflowId, current.ParentWorkflowId, "ParentWorkflowId mismatch")
	s.Equal(target.ParentRunId, current.ParentRunId, "ParentRunId mismatch")
	s.Equal(target.ParentInitiatedId, current.ParentInitiatedId, "ParentInitiatedId mismatch")
	s.Equal(target.CompletionEventBatchId, current.CompletionEventBatchId, "CompletionEventBatchId mismatch")
	s.Equal(target.TaskQueue, current.TaskQueue, "TaskQueue mismatch")
	s.Equal(target.WorkflowTypeName, current.WorkflowTypeName, "WorkflowTypeName mismatch")
	s.True(proto.Equal(target.WorkflowExecutionTimeout, current.WorkflowExecutionTimeout), "WorkflowExecutionTimeout mismatch")
	s.True(proto.Equal(target.WorkflowRunTimeout, current.WorkflowRunTimeout), "WorkflowRunTimeout mismatch")
	s.True(proto.Equal(target.DefaultWorkflowTaskTimeout, current.DefaultWorkflowTaskTimeout), "DefaultWorkflowTaskTimeout mismatch")
	s.Equal(target.LastRunningClock, current.LastRunningClock, "LastRunningClock mismatch")
	s.Equal(target.LastFirstEventId, current.LastFirstEventId, "LastFirstEventId mismatch")
	s.Equal(target.LastCompletedWorkflowTaskStartedEventId, current.LastCompletedWorkflowTaskStartedEventId, "LastCompletedWorkflowTaskStartedEventId mismatch")
	s.True(proto.Equal(target.StartTime, current.StartTime), "StartTime mismatch")
	s.True(proto.Equal(target.LastUpdateTime, current.LastUpdateTime), "LastUpdateTime mismatch")
	s.Equal(target.CancelRequested, current.CancelRequested, "CancelRequested mismatch")
	s.Equal(target.CancelRequestId, current.CancelRequestId, "CancelRequestId mismatch")
	s.Equal(target.StickyTaskQueue, current.StickyTaskQueue, "StickyTaskQueue mismatch")
	s.True(proto.Equal(target.StickyScheduleToStartTimeout, current.StickyScheduleToStartTimeout), "StickyScheduleToStartTimeout mismatch")
	s.Equal(target.Attempt, current.Attempt, "Attempt mismatch")
	s.True(proto.Equal(target.RetryInitialInterval, current.RetryInitialInterval), "RetryInitialInterval mismatch")
	s.True(proto.Equal(target.RetryMaximumInterval, current.RetryMaximumInterval), "RetryMaximumInterval mismatch")
	s.Equal(target.RetryMaximumAttempts, current.RetryMaximumAttempts, "RetryMaximumAttempts mismatch")
	s.Equal(target.RetryBackoffCoefficient, current.RetryBackoffCoefficient, "RetryBackoffCoefficient mismatch")
	s.True(proto.Equal(target.WorkflowExecutionExpirationTime, current.WorkflowExecutionExpirationTime), "WorkflowExecutionExpirationTime mismatch")
	s.Equal(target.RetryNonRetryableErrorTypes, current.RetryNonRetryableErrorTypes, "RetryNonRetryableErrorTypes mismatch")
	s.Equal(target.HasRetryPolicy, current.HasRetryPolicy, "HasRetryPolicy mismatch")
	s.Equal(target.CronSchedule, current.CronSchedule, "CronSchedule mismatch")
	s.Equal(target.SignalCount, current.SignalCount, "SignalCount mismatch")
	s.Equal(target.ActivityCount, current.ActivityCount, "ActivityCount mismatch")
	s.Equal(target.ChildExecutionCount, current.ChildExecutionCount, "ChildExecutionCount mismatch")
	s.Equal(target.UserTimerCount, current.UserTimerCount, "UserTimerCount mismatch")
	s.Equal(target.RequestCancelExternalCount, current.RequestCancelExternalCount, "RequestCancelExternalCount mismatch")
	s.Equal(target.SignalExternalCount, current.SignalExternalCount, "SignalExternalCount mismatch")
	s.Equal(target.UpdateCount, current.UpdateCount, "UpdateCount mismatch")
	s.True(proto.Equal(target.AutoResetPoints, current.AutoResetPoints), "AutoResetPoints mismatch")
	s.Equal(target.SearchAttributes, current.SearchAttributes, "SearchAttributes mismatch")
	s.Equal(target.Memo, current.Memo, "Memo mismatch")
	s.Equal(target.FirstExecutionRunId, current.FirstExecutionRunId, "FirstExecutionRunId mismatch")
	s.True(proto.Equal(target.WorkflowRunExpirationTime, current.WorkflowRunExpirationTime), "WorkflowRunExpirationTime mismatch")
	s.Equal(target.StateTransitionCount, current.StateTransitionCount, "StateTransitionCount mismatch")
	s.True(proto.Equal(target.ExecutionTime, current.ExecutionTime), "ExecutionTime mismatch")
	s.Equal(target.NewExecutionRunId, current.NewExecutionRunId, "NewExecutionRunId mismatch")
	s.Equal(target.ParentInitiatedVersion, current.ParentInitiatedVersion, "ParentInitiatedVersion mismatch")
	s.True(proto.Equal(target.CloseTime, current.CloseTime), "CloseTime mismatch")
	s.True(proto.Equal(target.BaseExecutionInfo, current.BaseExecutionInfo), "BaseExecutionInfo mismatch")
	s.True(proto.Equal(target.MostRecentWorkerVersionStamp, current.MostRecentWorkerVersionStamp), "MostRecentWorkerVersionStamp mismatch")
	s.Equal(target.AssignedBuildId, current.AssignedBuildId, "AssignedBuildId mismatch")
	s.Equal(target.InheritedBuildId, current.InheritedBuildId, "InheritedBuildId mismatch")
	s.Equal(target.BuildIdRedirectCounter, current.BuildIdRedirectCounter, "BuildIdRedirectCounter mismatch")
	s.Equal(target.SubStateMachinesByType, current.SubStateMachinesByType, "SubStateMachinesByType mismatch")
	s.Equal(target.RootWorkflowId, current.RootWorkflowId, "RootWorkflowId mismatch")
	s.Equal(target.RootRunId, current.RootRunId, "RootRunId mismatch")
	s.Equal(target.StateMachineTimers, current.StateMachineTimers, "StateMachineTimers mismatch")
	s.True(proto.Equal(target.WorkflowTaskLastUpdateVersionedTransition, current.WorkflowTaskLastUpdateVersionedTransition), "WorkflowTaskLastUpdateVersionedTransition mismatch")
	s.True(proto.Equal(target.VisibilityLastUpdateVersionedTransition, current.VisibilityLastUpdateVersionedTransition), "VisibilityLastUpdateVersionedTransition mismatch")
	s.True(proto.Equal(target.SignalRequestIdsLastUpdateVersionedTransition, current.SignalRequestIdsLastUpdateVersionedTransition), "SignalRequestIdsLastUpdateVersionedTransition mismatch")
	s.Equal(target.SubStateMachineTombstoneBatches, current.SubStateMachineTombstoneBatches, "SubStateMachineTombstoneBatches mismatch")
}

func (s *mutableStateSuite) verifyMutableState(current, target, origin *MutableStateImpl) {
	s.verifyExecutionInfo(current.executionInfo, target.executionInfo, origin.executionInfo)
	s.True(proto.Equal(target.executionState, current.executionState), "executionState mismatch")

	s.Equal(target.pendingActivityTimerHeartbeats, current.pendingActivityTimerHeartbeats, "pendingActivityTimerHeartbeats mismatch")
	s.verifyActivityInfos(target.pendingActivityInfoIDs, current.pendingActivityInfoIDs)
	s.Equal(target.pendingActivityIDToEventID, current.pendingActivityIDToEventID, "pendingActivityIDToEventID mismatch")
	compareMapOfProto(s, current.pendingActivityInfoIDs, current.updateActivityInfos)
	s.Equal(map[int64]struct{}{89: {}}, current.deleteActivityInfos, "deleteActivityInfos mismatch")

	compareMapOfProto(s, target.pendingTimerInfoIDs, current.pendingTimerInfoIDs)
	s.Equal(target.pendingTimerEventIDToID, current.pendingTimerEventIDToID, "pendingTimerEventIDToID mismatch")
	compareMapOfProto(s, target.pendingTimerInfoIDs, current.updateTimerInfos)
	s.Equal(map[string]struct{}{"to-be-deleted": {}}, current.deleteTimerInfos, "deleteTimerInfos mismatch")

	s.verifyChildExecutionInfos(target.pendingChildExecutionInfoIDs, current.pendingChildExecutionInfoIDs, origin.pendingChildExecutionInfoIDs)
	s.verifyChildExecutionInfos(target.pendingChildExecutionInfoIDs, current.updateChildExecutionInfos, origin.pendingChildExecutionInfoIDs)
	s.Equal(map[int64]struct{}{79: {}}, current.deleteChildExecutionInfos, "deleteChildExecutionInfos mismatch")

	compareMapOfProto(s, target.pendingRequestCancelInfoIDs, current.pendingRequestCancelInfoIDs)
	compareMapOfProto(s, target.pendingRequestCancelInfoIDs, current.updateRequestCancelInfos)
	s.Equal(map[int64]struct{}{69: {}}, current.deleteRequestCancelInfos, "deleteRequestCancelInfos mismatch")

	compareMapOfProto(s, target.pendingSignalInfoIDs, current.pendingSignalInfoIDs)
	compareMapOfProto(s, target.pendingSignalInfoIDs, current.updateSignalInfos)
	s.Equal(map[int64]struct{}{74: {}}, current.deleteSignalInfos, "deleteSignalInfos mismatch")

	s.Equal(target.pendingSignalRequestedIDs, current.pendingSignalRequestedIDs, "pendingSignalRequestedIDs mismatch")
	s.Equal(target.pendingSignalRequestedIDs, current.updateSignalRequestedIDs, "updateSignalRequestedIDs mismatch")
	s.Equal(map[string]struct{}{"to-be-deleted": {}}, current.deleteSignalRequestedIDs, "deleteSignalRequestedIDs mismatch")

	s.Equal(target.currentVersion, current.currentVersion, "currentVersion mismatch")
	s.Equal(target.totalTombstones, current.totalTombstones, "totalTombstones mismatch")
	s.Equal(target.dbRecordVersion, current.dbRecordVersion, "dbRecordVersion mismatch")
	s.True(proto.Equal(target.checksum, current.checksum), "checksum mismatch")
}

func (s *mutableStateSuite) buildSnapshot(state *MutableStateImpl) *persistencespb.WorkflowMutableState {
	snapshot := &persistencespb.WorkflowMutableState{
		ActivityInfos: map[int64]*persistencespb.ActivityInfo{
			90: {
				Version:                       1234,
				ScheduledTime:                 state.pendingActivityInfoIDs[90].ScheduledTime,
				StartedTime:                   state.pendingActivityInfoIDs[90].StartedTime,
				ActivityId:                    "activityID_5",
				ScheduleToStartTimeout:        timestamp.DurationPtr(time.Second * 100),
				ScheduleToCloseTimeout:        timestamp.DurationPtr(time.Second * 200),
				StartToCloseTimeout:           timestamp.DurationPtr(time.Second * 300),
				HeartbeatTimeout:              timestamp.DurationPtr(time.Second * 50),
				ScheduledEventId:              90,
				LastUpdateVersionedTransition: &persistencespb.VersionedTransition{TransitionCount: 1025},
			},
			91: {
				ActivityId:                    "activity_id_91",
				LastUpdateVersionedTransition: &persistencespb.VersionedTransition{TransitionCount: 1025},
			},
		},
		TimerInfos: map[string]*persistencespb.TimerInfo{
			"25": {
				Version:                       1234,
				StartedEventId:                85,
				ExpiryTime:                    state.pendingTimerInfoIDs["25"].ExpiryTime,
				TimerId:                       "25",
				LastUpdateVersionedTransition: &persistencespb.VersionedTransition{TransitionCount: 1025},
			},
			"26": {
				TimerId:                       "26",
				LastUpdateVersionedTransition: &persistencespb.VersionedTransition{TransitionCount: 1025},
			},
		},
		ChildExecutionInfos: map[int64]*persistencespb.ChildExecutionInfo{
			80: {
				Version:                       1234,
				InitiatedEventBatchId:         20,
				CreateRequestId:               state.pendingChildExecutionInfoIDs[80].CreateRequestId,
				Namespace:                     "mock namespace name",
				WorkflowTypeName:              "code.uber.internal/test/foobar",
				InitiatedEventId:              80,
				LastUpdateVersionedTransition: &persistencespb.VersionedTransition{TransitionCount: 1025},
			},
			81: {
				InitiatedEventBatchId:         81,
				LastUpdateVersionedTransition: &persistencespb.VersionedTransition{TransitionCount: 1025},
			},
		},
		RequestCancelInfos: map[int64]*persistencespb.RequestCancelInfo{
			70: {
				Version:                       1234,
				InitiatedEventBatchId:         20,
				CancelRequestId:               state.pendingRequestCancelInfoIDs[70].CancelRequestId,
				InitiatedEventId:              70,
				LastUpdateVersionedTransition: &persistencespb.VersionedTransition{TransitionCount: 1025},
			},
			71: {
				InitiatedEventBatchId:         71,
				LastUpdateVersionedTransition: &persistencespb.VersionedTransition{TransitionCount: 1025},
			},
		},
		SignalInfos: map[int64]*persistencespb.SignalInfo{
			75: {
				Version:                       1234,
				InitiatedEventBatchId:         17,
				RequestId:                     state.pendingSignalInfoIDs[75].RequestId,
				InitiatedEventId:              75,
				LastUpdateVersionedTransition: &persistencespb.VersionedTransition{TransitionCount: 1025},
			},
			76: {
				InitiatedEventBatchId:         76,
				LastUpdateVersionedTransition: &persistencespb.VersionedTransition{TransitionCount: 1025},
			},
		},
		ChasmNodes:         state.chasmTree.Snapshot(nil).Nodes,
		SignalRequestedIds: []string{"signal_request_id_1", "signal_requested_id_2"},
		ExecutionInfo: &persistencespb.WorkflowExecutionInfo{
			NamespaceId:                             "deadbeef-0123-4567-890a-bcdef0123456",
			WorkflowId:                              "wId",
			TaskQueue:                               "testTaskQueue",
			WorkflowTypeName:                        "wType",
			WorkflowRunTimeout:                      timestamp.DurationPtr(time.Second * 200),
			DefaultWorkflowTaskTimeout:              timestamp.DurationPtr(time.Second * 100),
			LastCompletedWorkflowTaskStartedEventId: 99,
			LastUpdateTime:                          state.executionInfo.LastUpdateTime,
			WorkflowTaskVersion:                     1234,
			WorkflowTaskScheduledEventId:            101,
			WorkflowTaskStartedEventId:              102,
			WorkflowTaskTimeout:                     timestamp.DurationPtr(time.Second * 100),
			WorkflowTaskAttempt:                     1,
			WorkflowTaskType:                        enumsspb.WORKFLOW_TASK_TYPE_NORMAL,
			VersionHistories: &historyspb.VersionHistories{
				Histories: []*historyspb.VersionHistory{
					{
						BranchToken: []byte("token#1"),
						Items: []*historyspb.VersionHistoryItem{
							{EventId: 102, Version: 1234},
						},
					},
				},
			},
			FirstExecutionRunId: state.executionInfo.FirstExecutionRunId,
			ExecutionTime:       state.executionInfo.ExecutionTime,
			TransitionHistory: []*persistencespb.VersionedTransition{
				{NamespaceFailoverVersion: 1234, TransitionCount: 1024},
				{TransitionCount: 1025},
			},
			SignalRequestIdsLastUpdateVersionedTransition: &persistencespb.VersionedTransition{TransitionCount: 1025},
			WorkflowTaskLastUpdateVersionedTransition:     state.executionInfo.WorkflowTaskLastUpdateVersionedTransition,
		},
		ExecutionState: &persistencespb.WorkflowExecutionState{
			RunId:     state.executionState.RunId,
			State:     enumsspb.WORKFLOW_EXECUTION_STATE_RUNNING,
			Status:    enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING,
			StartTime: state.executionState.StartTime,
		},
		NextEventId: 103,
	}
	return snapshot
}

func (s *mutableStateSuite) TestApplySnapshot() {
	testCases := []struct {
		name                        string
		updateWorkflowTask          bool
		speculativeTask             bool
		expectedWorkflowTaskUpdated bool
	}{
		{
			name:                        "update workflow task",
			updateWorkflowTask:          true,
			speculativeTask:             false,
			expectedWorkflowTaskUpdated: true,
		},
		{
			name:                        "not update workflow task",
			updateWorkflowTask:          false,
			speculativeTask:             false,
			expectedWorkflowTaskUpdated: false,
		},
		{
			name:                        "update speculative workflow task",
			updateWorkflowTask:          true,
			speculativeTask:             true,
			expectedWorkflowTaskUpdated: true,
		},
		{
			name:                        "not update speculative workflow task",
			updateWorkflowTask:          false,
			speculativeTask:             true,
			expectedWorkflowTaskUpdated: false,
		},
	}
	for _, tc := range testCases {
		s.Run(tc.name, func() {
			state := s.buildWorkflowMutableState()
			s.addChangesForStateReplication(state)

			chasmNodesSnapshot := chasm.NodesSnapshot{
				Nodes: state.ChasmNodes,
			}

			originMS, err := NewMutableStateFromDB(s.mockShard, s.mockEventsCache, s.logger, tests.LocalNamespaceEntry, state, 123)
			s.NoError(err)

			currentMS, err := NewMutableStateFromDB(s.mockShard, s.mockEventsCache, s.logger, tests.LocalNamespaceEntry, state, 123)
			s.NoError(err)
			currentMockChasmTree := historyi.NewMockChasmTree(s.controller)
			currentMockChasmTree.EXPECT().ApplySnapshot(chasmNodesSnapshot).Return(nil).Times(1)
			currentMS.chasmTree = currentMockChasmTree

			state = s.buildWorkflowMutableState()
			state.ActivityInfos[91] = &persistencespb.ActivityInfo{
				ActivityId: "activity_id_91",
			}
			state.TimerInfos["26"] = &persistencespb.TimerInfo{
				TimerId: "26",
			}
			state.ChildExecutionInfos[81] = &persistencespb.ChildExecutionInfo{
				InitiatedEventBatchId: 81,
			}
			state.RequestCancelInfos[71] = &persistencespb.RequestCancelInfo{
				InitiatedEventBatchId: 71,
			}
			state.SignalInfos[76] = &persistencespb.SignalInfo{
				InitiatedEventBatchId: 76,
			}
			state.SignalRequestedIds = append(state.SignalRequestedIds, "signal_requested_id_2")

			targetMS, err := NewMutableStateFromDB(s.mockShard, s.mockEventsCache, s.logger, tests.LocalNamespaceEntry, state, 123)
			s.NoError(err)
			targetMockChasmTree := historyi.NewMockChasmTree(s.controller)
			targetMockChasmTree.EXPECT().Snapshot(nil).Return(chasmNodesSnapshot).Times(1)
			targetMS.chasmTree = targetMockChasmTree

			targetMS.GetExecutionInfo().TransitionHistory = UpdatedTransitionHistory(targetMS.GetExecutionInfo().TransitionHistory, targetMS.GetCurrentVersion())

			// set updateXXX so LastUpdateVersionedTransition will be updated
			targetMS.updateActivityInfos = targetMS.pendingActivityInfoIDs
			for key := range targetMS.updateActivityInfos {
				targetMS.activityInfosUserDataUpdated[key] = struct{}{}
			}
			targetMS.updateTimerInfos = targetMS.pendingTimerInfoIDs
			for key := range targetMS.updateTimerInfos {
				targetMS.timerInfosUserDataUpdated[key] = struct{}{}
			}
			targetMS.updateChildExecutionInfos = targetMS.pendingChildExecutionInfoIDs
			targetMS.updateRequestCancelInfos = targetMS.pendingRequestCancelInfoIDs
			targetMS.updateSignalInfos = targetMS.pendingSignalInfoIDs
			targetMS.updateSignalRequestedIDs = targetMS.pendingSignalRequestedIDs

			if tc.updateWorkflowTask {
				// test mutation with workflow task update
				workflowTask := &historyi.WorkflowTaskInfo{
					ScheduledEventID: 1234,
				}
				targetMS.workflowTaskManager.UpdateWorkflowTask(workflowTask)
			}

			if tc.speculativeTask {
				targetMS.executionInfo.WorkflowTaskType = enumsspb.WORKFLOW_TASK_TYPE_SPECULATIVE
			}

			targetMS.closeTransactionTrackLastUpdateVersionedTransition(historyi.TransactionPolicyActive)

			snapshot := s.buildSnapshot(targetMS)
			s.Nil(snapshot.ExecutionInfo.SubStateMachinesByType)
			err = currentMS.ApplySnapshot(snapshot)
			s.NoError(err)
			s.NotNil(currentMS.GetExecutionInfo().SubStateMachinesByType)

			s.verifyMutableState(currentMS, targetMS, originMS)
			s.Equal(tc.expectedWorkflowTaskUpdated, currentMS.workflowTaskUpdated)
		})
	}
}

func (s *mutableStateSuite) buildMutation(
	state *MutableStateImpl,
	tombstones []*persistencespb.StateMachineTombstoneBatch,
) *persistencespb.WorkflowMutableStateMutation {
	executionInfoClone := common.CloneProto(state.executionInfo)
	executionInfoClone.SubStateMachineTombstoneBatches = nil
	mutation := &persistencespb.WorkflowMutableStateMutation{
		UpdatedActivityInfos:            state.pendingActivityInfoIDs,
		UpdatedTimerInfos:               state.pendingTimerInfoIDs,
		UpdatedChildExecutionInfos:      state.pendingChildExecutionInfoIDs,
		UpdatedRequestCancelInfos:       state.pendingRequestCancelInfoIDs,
		UpdatedSignalInfos:              state.pendingSignalInfoIDs,
		UpdatedChasmNodes:               state.chasmTree.Snapshot(nil).Nodes,
		SignalRequestedIds:              state.GetPendingSignalRequestedIds(),
		SubStateMachineTombstoneBatches: tombstones,
		ExecutionInfo:                   executionInfoClone,
		ExecutionState:                  state.executionState,
	}
	return mutation
}

func (s *mutableStateSuite) TestApplyMutation() {
	testCases := []struct {
		name                        string
		updateWorkflowTask          bool
		speculativeTask             bool
		expectedWorkflowTaskUpdated bool
	}{
		{
			name:                        "update workflow task",
			updateWorkflowTask:          true,
			speculativeTask:             false,
			expectedWorkflowTaskUpdated: true,
		},
		{
			name:                        "not update workflow task",
			updateWorkflowTask:          false,
			speculativeTask:             false,
			expectedWorkflowTaskUpdated: false,
		},
		{
			name:                        "update speculative workflow task",
			updateWorkflowTask:          true,
			speculativeTask:             true,
			expectedWorkflowTaskUpdated: true,
		},
		{
			name:                        "not update speculative workflow task",
			updateWorkflowTask:          false,
			speculativeTask:             true,
			expectedWorkflowTaskUpdated: false,
		},
	}
	for _, tc := range testCases {
		s.Run(tc.name, func() {
			state := s.buildWorkflowMutableState()
			s.addChangesForStateReplication(state)

			originMS, err := NewMutableStateFromDB(s.mockShard, s.mockEventsCache, s.logger, tests.LocalNamespaceEntry, state, 123)
			s.NoError(err)
			tombstones := []*persistencespb.StateMachineTombstoneBatch{
				{
					VersionedTransition: &persistencespb.VersionedTransition{NamespaceFailoverVersion: 1, TransitionCount: 1},
					StateMachineTombstones: []*persistencespb.StateMachineTombstone{
						{
							StateMachineKey: &persistencespb.StateMachineTombstone_ActivityScheduledEventId{
								ActivityScheduledEventId: 10,
							},
						},
					},
				},
			}
			originMS.GetExecutionInfo().SubStateMachineTombstoneBatches = tombstones

			currentMS, err := NewMutableStateFromDB(s.mockShard, s.mockEventsCache, s.logger, tests.LocalNamespaceEntry, state, 123)
			s.NoError(err)
			currentMockChasmTree := historyi.NewMockChasmTree(s.controller)
			currentMS.chasmTree = currentMockChasmTree

			currentMS.GetExecutionInfo().SubStateMachineTombstoneBatches = tombstones

			state = s.buildWorkflowMutableState()

			targetMS, err := NewMutableStateFromDB(s.mockShard, s.mockEventsCache, s.logger, tests.LocalNamespaceEntry, state, 123)
			s.NoError(err)

			targetMockChasmTree := historyi.NewMockChasmTree(s.controller)
			updateChasmNodes := map[string]*persistencespb.ChasmNode{
				"node-path": {
					Metadata: &persistencespb.ChasmNodeMetadata{
						InitialVersionedTransition:    &persistencespb.VersionedTransition{NamespaceFailoverVersion: 1, TransitionCount: 1},
						LastUpdateVersionedTransition: &persistencespb.VersionedTransition{NamespaceFailoverVersion: 1, TransitionCount: 1},
						Attributes:                    &persistencespb.ChasmNodeMetadata_DataAttributes{},
					},
					Data: &commonpb.DataBlob{Data: []byte("test-data")},
				},
				"node-path/collection-node": {
					Metadata: &persistencespb.ChasmNodeMetadata{
						InitialVersionedTransition:    &persistencespb.VersionedTransition{NamespaceFailoverVersion: 1, TransitionCount: 1},
						LastUpdateVersionedTransition: &persistencespb.VersionedTransition{NamespaceFailoverVersion: 1, TransitionCount: 1},
						Attributes:                    &persistencespb.ChasmNodeMetadata_CollectionAttributes{},
					},
				},
			}
			targetMockChasmTree.EXPECT().Snapshot(nil).Return(chasm.NodesSnapshot{
				Nodes: updateChasmNodes,
			})
			targetMS.chasmTree = targetMockChasmTree

			transitionHistory := targetMS.executionInfo.TransitionHistory
			failoverVersion := transitionhistory.LastVersionedTransition(transitionHistory).NamespaceFailoverVersion
			targetMS.executionInfo.TransitionHistory = UpdatedTransitionHistory(transitionHistory, failoverVersion)

			// set updateXXX so LastUpdateVersionedTransition will be updated
			targetMS.updateActivityInfos = targetMS.pendingActivityInfoIDs
			for key := range targetMS.updateActivityInfos {
				targetMS.activityInfosUserDataUpdated[key] = struct{}{}
			}
			targetMS.updateTimerInfos = targetMS.pendingTimerInfoIDs
			for key := range targetMS.updateTimerInfos {
				targetMS.timerInfosUserDataUpdated[key] = struct{}{}
			}
			targetMS.updateChildExecutionInfos = targetMS.pendingChildExecutionInfoIDs
			targetMS.updateRequestCancelInfos = targetMS.pendingRequestCancelInfoIDs
			targetMS.updateSignalInfos = targetMS.pendingSignalInfoIDs
			targetMS.updateSignalRequestedIDs = targetMS.pendingSignalRequestedIDs

			if tc.updateWorkflowTask {
				// test mutation with workflow task update
				workflowTask := &historyi.WorkflowTaskInfo{
					ScheduledEventID: 1234,
				}
				targetMS.workflowTaskManager.UpdateWorkflowTask(workflowTask)
			}

			if tc.speculativeTask {
				targetMS.executionInfo.WorkflowTaskType = enumsspb.WORKFLOW_TASK_TYPE_SPECULATIVE
			}

			targetMS.closeTransactionTrackLastUpdateVersionedTransition(historyi.TransactionPolicyActive)

			tombstonesToAdd := []*persistencespb.StateMachineTombstoneBatch{
				{
					VersionedTransition: targetMS.CurrentVersionedTransition(),
					StateMachineTombstones: []*persistencespb.StateMachineTombstone{
						{
							StateMachineKey: &persistencespb.StateMachineTombstone_ActivityScheduledEventId{
								ActivityScheduledEventId: 89,
							},
						},
						{
							StateMachineKey: &persistencespb.StateMachineTombstone_ActivityScheduledEventId{
								ActivityScheduledEventId: 9999, // not exist
							},
						},
						{
							StateMachineKey: &persistencespb.StateMachineTombstone_TimerId{
								TimerId: "to-be-deleted",
							},
						},
						{

							StateMachineKey: &persistencespb.StateMachineTombstone_TimerId{
								TimerId: "not-exist",
							},
						},
						{
							StateMachineKey: &persistencespb.StateMachineTombstone_ChildExecutionInitiatedEventId{
								ChildExecutionInitiatedEventId: 79,
							},
						},
						{
							StateMachineKey: &persistencespb.StateMachineTombstone_ChildExecutionInitiatedEventId{
								ChildExecutionInitiatedEventId: 9998, // not exist
							},
						},
						{
							StateMachineKey: &persistencespb.StateMachineTombstone_RequestCancelInitiatedEventId{
								RequestCancelInitiatedEventId: 69,
							},
						},
						{
							StateMachineKey: &persistencespb.StateMachineTombstone_RequestCancelInitiatedEventId{
								RequestCancelInitiatedEventId: 9997, // not exist
							},
						},
						{
							StateMachineKey: &persistencespb.StateMachineTombstone_SignalExternalInitiatedEventId{
								SignalExternalInitiatedEventId: 74,
							},
						},
						{
							StateMachineKey: &persistencespb.StateMachineTombstone_SignalExternalInitiatedEventId{
								SignalExternalInitiatedEventId: 9996, // not exist
							},
						},
						{
							StateMachineKey: &persistencespb.StateMachineTombstone_ChasmNodePath{
								ChasmNodePath: "deleted-node-path",
							},
						},
					},
				},
			}
			targetMS.GetExecutionInfo().SubStateMachineTombstoneBatches = append(tombstones, tombstonesToAdd...)
			targetMS.totalTombstones = len(tombstones[0].StateMachineTombstones) + len(tombstonesToAdd[0].StateMachineTombstones)
			mutation := s.buildMutation(targetMS, tombstonesToAdd)

			currentMockChasmTree.EXPECT().ApplyMutation(chasm.NodesMutation{
				DeletedNodes: map[string]struct{}{"deleted-node-path": {}},
			}).Return(nil).Times(1)
			currentMockChasmTree.EXPECT().ApplyMutation(chasm.NodesMutation{
				UpdatedNodes: updateChasmNodes,
			}).Return(nil).Times(1)

			err = currentMS.ApplyMutation(mutation)
			s.NoError(err)
			s.verifyMutableState(currentMS, targetMS, originMS)
			s.Equal(tc.expectedWorkflowTaskUpdated, currentMS.workflowTaskUpdated)
		})
	}
}

func (s *mutableStateSuite) TestRefreshTask_DiffCluster() {
	version := int64(99)
	attempt := int32(1)
	incomingActivityInfo := &persistencespb.ActivityInfo{
		Version: version,
		Attempt: attempt,
	}
	localActivityInfo := &persistencespb.ActivityInfo{
		Version: int64(100),
		Attempt: incomingActivityInfo.Attempt,
	}

	s.mockShard.Resource.ClusterMetadata.EXPECT().IsVersionFromSameCluster(localActivityInfo.Version, version).Return(false)

	shouldReset := s.mutableState.ShouldResetActivityTimerTaskMask(
		localActivityInfo,
		incomingActivityInfo,
	)
	s.True(shouldReset)
}

func (s *mutableStateSuite) TestRefreshTask_SameCluster_DiffAttempt() {
	version := int64(99)
	attempt := int32(1)
	incomingActivityInfo := &persistencespb.ActivityInfo{
		Version: version,
		Attempt: attempt,
	}
	localActivityInfo := &persistencespb.ActivityInfo{
		Version: version,
		Attempt: attempt + 1,
	}

	s.mockShard.Resource.ClusterMetadata.EXPECT().IsVersionFromSameCluster(version, version).Return(true)

	shouldReset := s.mutableState.ShouldResetActivityTimerTaskMask(
		localActivityInfo,
		incomingActivityInfo,
	)
	s.True(shouldReset)
}

func (s *mutableStateSuite) TestRefreshTask_SameCluster_SameAttempt() {
	version := int64(99)
	attempt := int32(1)
	incomingActivityInfo := &persistencespb.ActivityInfo{
		Version: version,
		Attempt: attempt,
	}
	localActivityInfo := &persistencespb.ActivityInfo{
		Version: version,
		Attempt: attempt,
	}

	s.mockShard.Resource.ClusterMetadata.EXPECT().IsVersionFromSameCluster(version, version).Return(true)

	shouldReset := s.mutableState.ShouldResetActivityTimerTaskMask(
		localActivityInfo,
		incomingActivityInfo,
	)
	s.False(shouldReset)
}

func (s *mutableStateSuite) TestUpdateActivityTaskStatusWithTimerHeartbeat() {
	dbState := s.buildWorkflowMutableState()
	scheduleEventId := int64(781)
	dbState.ActivityInfos[scheduleEventId] = &persistencespb.ActivityInfo{
		Version:                5,
		ScheduledEventId:       int64(90),
		ScheduledTime:          timestamppb.New(time.Now().UTC()),
		StartedEventId:         common.EmptyEventID,
		StartedTime:            timestamppb.New(time.Now().UTC()),
		ActivityId:             "activityID_5",
		TimerTaskStatus:        0,
		ScheduleToStartTimeout: timestamp.DurationFromSeconds(100),
		ScheduleToCloseTimeout: timestamp.DurationFromSeconds(200),
		StartToCloseTimeout:    timestamp.DurationFromSeconds(300),
		HeartbeatTimeout:       timestamp.DurationFromSeconds(50),
	}
	mutableState, err := NewMutableStateFromDB(s.mockShard, s.mockEventsCache, s.logger, s.namespaceEntry, dbState, 123)
	s.NoError(err)
	originalTime := time.Now().UTC().Add(time.Second * 60)
	mutableState.pendingActivityTimerHeartbeats[scheduleEventId] = originalTime
	status := int32(1)
	err = mutableState.UpdateActivityTaskStatusWithTimerHeartbeat(scheduleEventId, status, nil)
	s.NoError(err)
	s.Equal(status, dbState.ActivityInfos[scheduleEventId].TimerTaskStatus)
	s.Equal(originalTime, mutableState.pendingActivityTimerHeartbeats[scheduleEventId])
}

func (s *mutableStateSuite) TestHasRequestID() {
	testCases := []struct {
		name          string
		requestID     string
		setupFunc     func(ms *MutableStateImpl) // Setup function to prepare the mutable state
		expectedFound bool
	}{
		{
			name:      "empty_request_id",
			requestID: "",
			setupFunc: func(ms *MutableStateImpl) {
				// No setup needed
			},
			expectedFound: false,
		},
		{
			name:      "request_id_not_found",
			requestID: "non-existent-request-id",
			setupFunc: func(ms *MutableStateImpl) {
				// No setup needed
			},
			expectedFound: false,
		},
		{
			name:      "request_id_found",
			requestID: "existing-request-id",
			setupFunc: func(ms *MutableStateImpl) {
				// Add request ID to execution state
				ms.AttachRequestID("existing-request-id", enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_OPTIONS_UPDATED, 100)
			},
			expectedFound: true,
		},
		{
			name:      "multiple_request_ids_found_target",
			requestID: "target-request-id",
			setupFunc: func(ms *MutableStateImpl) {
				// Add multiple request IDs
				ms.AttachRequestID("request-id-1", enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_OPTIONS_UPDATED, 101)
				ms.AttachRequestID("target-request-id", enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_OPTIONS_UPDATED, 102)
				ms.AttachRequestID("request-id-3", enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_OPTIONS_UPDATED, 103)
			},
			expectedFound: true,
		},
		{
			name:      "multiple_request_ids_not_found_target",
			requestID: "missing-request-id",
			setupFunc: func(ms *MutableStateImpl) {
				// Add multiple request IDs, but not the target one
				ms.AttachRequestID("request-id-1", enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_OPTIONS_UPDATED, 104)
				ms.AttachRequestID("request-id-2", enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_OPTIONS_UPDATED, 105)
				ms.AttachRequestID("request-id-3", enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_OPTIONS_UPDATED, 106)
			},
			expectedFound: false,
		},
		{
			name:      "request_id_case_sensitive",
			requestID: "Case-Sensitive-ID",
			setupFunc: func(ms *MutableStateImpl) {
				// Add request ID with different case
				ms.AttachRequestID("case-sensitive-id", enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_OPTIONS_UPDATED, 107)
			},
			expectedFound: false,
		},
		{
			name:      "request_id_exact_match",
			requestID: "exact-match-id",
			setupFunc: func(ms *MutableStateImpl) {
				// Add exact match
				ms.AttachRequestID("exact-match-id", enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_OPTIONS_UPDATED, 108)
			},
			expectedFound: true,
		},
		{
			name:      "request_id_with_special_characters",
			requestID: "request-id-with-$pecial-ch@racters_123",
			setupFunc: func(ms *MutableStateImpl) {
				ms.AttachRequestID("request-id-with-$pecial-ch@racters_123", enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_OPTIONS_UPDATED, 109)
			},
			expectedFound: true,
		},
		{
			name:      "uuid_style_request_id",
			requestID: "12345678-1234-1234-1234-123456789abc",
			setupFunc: func(ms *MutableStateImpl) {
				ms.AttachRequestID("12345678-1234-1234-1234-123456789abc", enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_OPTIONS_UPDATED, 110)
			},
			expectedFound: true,
		},
		{
			name:      "very_long_request_id",
			requestID: "very-long-request-id-" + strings.Repeat("x", 100),
			setupFunc: func(ms *MutableStateImpl) {
				ms.AttachRequestID("very-long-request-id-"+strings.Repeat("x", 100), enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_OPTIONS_UPDATED, 111)
			},
			expectedFound: true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupSubTest()

			// Setup the mutable state
			tc.setupFunc(s.mutableState)

			// Test HasRequestID
			found := s.mutableState.HasRequestID(tc.requestID)
			s.Equal(tc.expectedFound, found, "HasRequestID result should match expected")

			// Verify the request ID existence directly in the execution state if expected to be found
			if tc.expectedFound && tc.requestID != "" {
				_, exists := s.mutableState.executionState.RequestIds[tc.requestID]
				s.True(exists, "Request ID should exist in execution state RequestIds map")
			}
		})
	}
}

func (s *mutableStateSuite) TestHasRequestID_StateConsistency() {
	s.SetupTest()

	// Test that HasRequestID is consistent with AttachRequestID
	requestID := "consistency-test-request-id"

	// Initially should not exist
	s.False(s.mutableState.HasRequestID(requestID))

	// After attaching, should exist
	s.mutableState.AttachRequestID(requestID, enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_OPTIONS_UPDATED, 200)
	s.True(s.mutableState.HasRequestID(requestID))

	// Should still exist after multiple calls
	s.True(s.mutableState.HasRequestID(requestID))
	s.True(s.mutableState.HasRequestID(requestID))
}

func (s *mutableStateSuite) TestHasRequestID_EmptyExecutionState() {
	s.SetupTest()

	// Ensure execution state has no request IDs initially
	if s.mutableState.executionState.RequestIds == nil {
		s.mutableState.executionState.RequestIds = make(map[string]*persistencespb.RequestIDInfo)
	}

	// Clear any existing request IDs
	for k := range s.mutableState.executionState.RequestIds {
		delete(s.mutableState.executionState.RequestIds, k)
	}

	// Test various request IDs on empty state
	testRequestIDs := []string{
		"",
		"test-id",
		"another-id",
	}

	for _, requestID := range testRequestIDs {
		s.False(s.mutableState.HasRequestID(requestID), "Should return false for request ID: %s", requestID)
	}
}
