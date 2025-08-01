package tests

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	enumspb "go.temporal.io/api/enums/v1"
	clockspb "go.temporal.io/server/api/clock/v1"
	persistencespb "go.temporal.io/server/api/persistence/v1"
	"go.temporal.io/server/common/debug"
	"go.temporal.io/server/common/log"
	p "go.temporal.io/server/common/persistence"
	"go.temporal.io/server/common/persistence/serialization"
	"go.temporal.io/server/common/testing/protorequire"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type (
	TaskQueueTaskSuite struct {
		suite.Suite
		*require.Assertions

		stickyTTL     time.Duration
		taskTTL       time.Duration
		namespaceID   string
		taskQueueName string
		taskQueueType enumspb.TaskQueueType

		taskManager p.TaskManager
		logger      log.Logger

		ctx    context.Context
		cancel context.CancelFunc
	}
)

func NewTaskQueueTaskSuite(
	t *testing.T,
	taskStore p.TaskStore,
	logger log.Logger,
) *TaskQueueTaskSuite {
	return &TaskQueueTaskSuite{
		Assertions: require.New(t),
		taskManager: p.NewTaskManager(
			taskStore,
			serialization.NewSerializer(),
		),
		logger: logger,
	}
}

func (s *TaskQueueTaskSuite) SetupSuite() {
}

func (s *TaskQueueTaskSuite) TearDownSuite() {
}

func (s *TaskQueueTaskSuite) SetupTest() {
	s.Assertions = require.New(s.T())
	s.ctx, s.cancel = context.WithTimeout(context.Background(), 30*time.Second*debug.TimeoutMultiplier)

	s.stickyTTL = time.Second * 10
	s.taskTTL = time.Second * 16
	s.namespaceID = uuid.New().String()
	s.taskQueueName = uuid.New().String()
	s.taskQueueType = enumspb.TaskQueueType(rand.Int31n(
		int32(len(enumspb.TaskQueueType_name)) + 1),
	)
}

func (s *TaskQueueTaskSuite) TearDownTest() {
	s.cancel()
}

func (s *TaskQueueTaskSuite) TestCreateGet_Conflict() {
	rangeID := rand.Int63()
	taskQueue := s.createTaskQueue(rangeID)

	taskID := rand.Int63()
	task := s.randomTask(taskID)
	_, err := s.taskManager.CreateTasks(s.ctx, &p.CreateTasksRequest{
		TaskQueueInfo: &p.PersistedTaskQueueInfo{
			RangeID: rand.Int63(),
			Data:    taskQueue,
		},
		Tasks: []*persistencespb.AllocatedTaskInfo{task},
	})
	s.IsType(&p.ConditionFailedError{}, err)

	resp, err := s.taskManager.GetTasks(s.ctx, &p.GetTasksRequest{
		NamespaceID:        s.namespaceID,
		TaskQueue:          s.taskQueueName,
		TaskType:           s.taskQueueType,
		InclusiveMinTaskID: taskID,
		ExclusiveMaxTaskID: taskID + 1,
		PageSize:           100,
		NextPageToken:      nil,
	})
	s.NoError(err)
	protorequire.ProtoSliceEqual(s.T(), []*persistencespb.AllocatedTaskInfo{}, resp.Tasks)
	s.Nil(resp.NextPageToken)
}

func (s *TaskQueueTaskSuite) TestCreateGet_One() {
	rangeID := rand.Int63()
	taskQueue := s.createTaskQueue(rangeID)

	taskID := rand.Int63()
	task := s.randomTask(taskID)
	_, err := s.taskManager.CreateTasks(s.ctx, &p.CreateTasksRequest{
		TaskQueueInfo: &p.PersistedTaskQueueInfo{
			RangeID: rangeID,
			Data:    taskQueue,
		},
		Tasks: []*persistencespb.AllocatedTaskInfo{task},
	})
	s.NoError(err)

	resp, err := s.taskManager.GetTasks(s.ctx, &p.GetTasksRequest{
		NamespaceID:        s.namespaceID,
		TaskQueue:          s.taskQueueName,
		TaskType:           s.taskQueueType,
		InclusiveMinTaskID: taskID,
		ExclusiveMaxTaskID: taskID + 1,
		PageSize:           100,
		NextPageToken:      nil,
	})
	s.NoError(err)
	protorequire.ProtoSliceEqual(s.T(), []*persistencespb.AllocatedTaskInfo{task}, resp.Tasks)
	s.Nil(resp.NextPageToken)
}

func (s *TaskQueueTaskSuite) TestCreateGet_Multiple() {
	numCreateBatch := 32
	createBatchSize := 32
	numTasks := int64(createBatchSize * numCreateBatch)
	minTaskID := rand.Int63()
	maxTaskID := minTaskID + numTasks

	rangeID := rand.Int63()
	taskQueue := s.createTaskQueue(rangeID)

	var expectedTasks []*persistencespb.AllocatedTaskInfo
	for i := 0; i < numCreateBatch; i++ {
		var tasks []*persistencespb.AllocatedTaskInfo
		for j := 0; j < createBatchSize; j++ {
			taskID := minTaskID + int64(i*numCreateBatch+j)
			task := s.randomTask(taskID)
			tasks = append(tasks, task)
			expectedTasks = append(expectedTasks, task)
		}
		_, err := s.taskManager.CreateTasks(s.ctx, &p.CreateTasksRequest{
			TaskQueueInfo: &p.PersistedTaskQueueInfo{
				RangeID: rangeID,
				Data:    taskQueue,
			},
			Tasks: tasks,
		})
		s.NoError(err)
	}

	var token []byte
	var actualTasks []*persistencespb.AllocatedTaskInfo
	for doContinue := true; doContinue; doContinue = len(token) > 0 {
		resp, err := s.taskManager.GetTasks(s.ctx, &p.GetTasksRequest{
			NamespaceID:        s.namespaceID,
			TaskQueue:          s.taskQueueName,
			TaskType:           s.taskQueueType,
			InclusiveMinTaskID: minTaskID,
			ExclusiveMaxTaskID: maxTaskID + 1,
			PageSize:           1,
			NextPageToken:      token,
		})
		s.NoError(err)
		token = resp.NextPageToken
		actualTasks = append(actualTasks, resp.Tasks...)
	}
	protorequire.ProtoSliceEqual(s.T(), expectedTasks, actualTasks)
}

func (s *TaskQueueTaskSuite) TestCreateDelete_Multiple() {
	numCreateBatch := 32
	createBatchSize := 32
	numTasks := int64(createBatchSize * numCreateBatch)
	minTaskID := rand.Int63()
	maxTaskID := minTaskID + numTasks

	rangeID := rand.Int63()
	taskQueue := s.createTaskQueue(rangeID)

	for i := 0; i < numCreateBatch; i++ {
		var tasks []*persistencespb.AllocatedTaskInfo
		for j := 0; j < createBatchSize; j++ {
			taskID := minTaskID + int64(i*numCreateBatch+j)
			task := s.randomTask(taskID)
			tasks = append(tasks, task)
		}
		_, err := s.taskManager.CreateTasks(s.ctx, &p.CreateTasksRequest{
			TaskQueueInfo: &p.PersistedTaskQueueInfo{
				RangeID: rangeID,
				Data:    taskQueue,
			},
			Tasks: tasks,
		})
		s.NoError(err)
	}

	_, err := s.taskManager.CompleteTasksLessThan(s.ctx, &p.CompleteTasksLessThanRequest{
		NamespaceID:        s.namespaceID,
		TaskQueueName:      s.taskQueueName,
		TaskType:           s.taskQueueType,
		ExclusiveMaxTaskID: maxTaskID + 1,
		Limit:              int(numTasks),
	})
	s.NoError(err)

	resp, err := s.taskManager.GetTasks(s.ctx, &p.GetTasksRequest{
		NamespaceID:        s.namespaceID,
		TaskQueue:          s.taskQueueName,
		TaskType:           s.taskQueueType,
		InclusiveMinTaskID: minTaskID,
		ExclusiveMaxTaskID: maxTaskID + 1,
		PageSize:           100,
		NextPageToken:      nil,
	})
	s.NoError(err)
	protorequire.ProtoSliceEqual(s.T(), []*persistencespb.AllocatedTaskInfo{}, resp.Tasks)
	s.Nil(resp.NextPageToken)
}

func (s *TaskQueueTaskSuite) createTaskQueue(
	rangeID int64,
) *persistencespb.TaskQueueInfo {
	taskQueueKind := enumspb.TaskQueueKind(rand.Int31n(
		int32(len(enumspb.TaskQueueKind_name)) + 1),
	)
	taskQueue := s.randomTaskQueueInfo(taskQueueKind)
	_, err := s.taskManager.CreateTaskQueue(s.ctx, &p.CreateTaskQueueRequest{
		RangeID:       rangeID,
		TaskQueueInfo: taskQueue,
	})
	s.NoError(err)
	return taskQueue
}

func (s *TaskQueueTaskSuite) randomTaskQueueInfo(
	taskQueueKind enumspb.TaskQueueKind,
) *persistencespb.TaskQueueInfo {
	now := time.Now().UTC()
	var expiryTime *timestamppb.Timestamp
	if taskQueueKind == enumspb.TASK_QUEUE_KIND_STICKY {
		expiryTime = timestamppb.New(now.Add(s.stickyTTL))
	}

	return &persistencespb.TaskQueueInfo{
		NamespaceId:    s.namespaceID,
		Name:           s.taskQueueName,
		TaskType:       s.taskQueueType,
		Kind:           taskQueueKind,
		AckLevel:       rand.Int63(),
		ExpiryTime:     expiryTime,
		LastUpdateTime: timestamppb.New(now),
	}
}

func (s *TaskQueueTaskSuite) randomTask(
	taskID int64,
) *persistencespb.AllocatedTaskInfo {
	now := time.Now().UTC()
	return &persistencespb.AllocatedTaskInfo{
		TaskId: taskID,
		Data: &persistencespb.TaskInfo{
			NamespaceId:      s.namespaceID,
			WorkflowId:       uuid.New().String(),
			RunId:            uuid.New().String(),
			ScheduledEventId: rand.Int63(),
			CreateTime:       timestamppb.New(now),
			ExpiryTime:       timestamppb.New(now.Add(s.taskTTL)),
			Clock: &clockspb.VectorClock{
				ClusterId: rand.Int63(),
				ShardId:   rand.Int31(),
				Clock:     rand.Int63(),
			},
		},
	}
}
