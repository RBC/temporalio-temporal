//go:generate stringer -type loadCause -trimprefix loadCause -output loadcause_string_gen.go
//go:generate stringer -type unloadCause -trimprefix unloadCause -output unloadcause_string_gen.go

package matching

import (
	"time"

	"go.temporal.io/server/common/backoff"
	"go.temporal.io/server/common/dynamicconfig"
	"go.temporal.io/server/common/namespace"
	"go.temporal.io/server/common/tqid"
	"go.temporal.io/server/service/matching/counter"
)

type (
	// Config represents configuration for matching service
	Config struct {
		PersistenceMaxQPS                    dynamicconfig.IntPropertyFn
		PersistenceGlobalMaxQPS              dynamicconfig.IntPropertyFn
		PersistenceNamespaceMaxQPS           dynamicconfig.IntPropertyFnWithNamespaceFilter
		PersistenceGlobalNamespaceMaxQPS     dynamicconfig.IntPropertyFnWithNamespaceFilter
		PersistencePerShardNamespaceMaxQPS   dynamicconfig.IntPropertyFnWithNamespaceFilter
		PersistenceDynamicRateLimitingParams dynamicconfig.TypedPropertyFn[dynamicconfig.DynamicRateLimitingParams]
		PersistenceQPSBurstRatio             dynamicconfig.FloatPropertyFn
		SyncMatchWaitDuration                dynamicconfig.DurationPropertyFnWithTaskQueueFilter
		RPS                                  dynamicconfig.IntPropertyFn
		OperatorRPSRatio                     dynamicconfig.FloatPropertyFn
		AlignMembershipChange                dynamicconfig.DurationPropertyFn
		ShutdownDrainDuration                dynamicconfig.DurationPropertyFn
		HistoryMaxPageSize                   dynamicconfig.IntPropertyFnWithNamespaceFilter
		EnableDeployments                    dynamicconfig.BoolPropertyFnWithNamespaceFilter // [cleanup-wv-pre-release]
		EnableDeploymentVersions             dynamicconfig.BoolPropertyFnWithNamespaceFilter
		MaxTaskQueuesInDeployment            dynamicconfig.IntPropertyFnWithNamespaceFilter
		MaxIDLengthLimit                     dynamicconfig.IntPropertyFn

		// task queue configuration

		RangeSize                                int64
		NewMatcher                               dynamicconfig.TypedSubscribableWithTaskQueueFilter[bool]
		EnableFairness                           dynamicconfig.TypedSubscribableWithTaskQueueFilter[bool]
		GetTasksBatchSize                        dynamicconfig.IntPropertyFnWithTaskQueueFilter
		GetTasksReloadAt                         dynamicconfig.IntPropertyFnWithTaskQueueFilter
		UpdateAckInterval                        dynamicconfig.DurationPropertyFnWithTaskQueueFilter
		MaxTaskQueueIdleTime                     dynamicconfig.DurationPropertyFnWithTaskQueueFilter
		NumTaskqueueWritePartitions              dynamicconfig.IntPropertyFnWithTaskQueueFilter
		NumTaskqueueReadPartitions               dynamicconfig.IntPropertyFnWithTaskQueueFilter
		BreakdownMetricsByTaskQueue              dynamicconfig.BoolPropertyFnWithTaskQueueFilter
		BreakdownMetricsByPartition              dynamicconfig.BoolPropertyFnWithTaskQueueFilter
		BreakdownMetricsByBuildID                dynamicconfig.BoolPropertyFnWithTaskQueueFilter
		ForwarderMaxOutstandingPolls             dynamicconfig.IntPropertyFnWithTaskQueueFilter
		ForwarderMaxOutstandingTasks             dynamicconfig.IntPropertyFnWithTaskQueueFilter
		ForwarderMaxRatePerSecond                dynamicconfig.FloatPropertyFnWithTaskQueueFilter
		ForwarderMaxChildrenPerNode              dynamicconfig.IntPropertyFnWithTaskQueueFilter
		VersionCompatibleSetLimitPerQueue        dynamicconfig.IntPropertyFnWithNamespaceFilter
		VersionBuildIdLimitPerQueue              dynamicconfig.IntPropertyFnWithNamespaceFilter
		AssignmentRuleLimitPerQueue              dynamicconfig.IntPropertyFnWithNamespaceFilter
		RedirectRuleLimitPerQueue                dynamicconfig.IntPropertyFnWithNamespaceFilter
		RedirectRuleMaxUpstreamBuildIDsPerQueue  dynamicconfig.IntPropertyFnWithNamespaceFilter
		DeletedRuleRetentionTime                 dynamicconfig.DurationPropertyFnWithNamespaceFilter
		PollerHistoryTTL                         dynamicconfig.DurationPropertyFnWithNamespaceFilter
		ReachabilityBuildIdVisibilityGracePeriod dynamicconfig.DurationPropertyFnWithNamespaceFilter
		ReachabilityCacheOpenWFsTTL              dynamicconfig.DurationPropertyFn
		ReachabilityCacheClosedWFsTTL            dynamicconfig.DurationPropertyFn
		TaskQueueLimitPerBuildId                 dynamicconfig.IntPropertyFnWithNamespaceFilter
		GetUserDataLongPollTimeout               dynamicconfig.DurationPropertyFn
		GetUserDataRefresh                       dynamicconfig.DurationPropertyFn
		BacklogNegligibleAge                     dynamicconfig.DurationPropertyFnWithTaskQueueFilter
		MaxWaitForPollerBeforeFwd                dynamicconfig.DurationPropertyFnWithTaskQueueFilter
		QueryPollerUnavailableWindow             dynamicconfig.DurationPropertyFn
		QueryWorkflowTaskTimeoutLogRate          dynamicconfig.FloatPropertyFnWithTaskQueueFilter
		MembershipUnloadDelay                    dynamicconfig.DurationPropertyFn
		TaskQueueInfoByBuildIdTTL                dynamicconfig.DurationPropertyFnWithTaskQueueFilter
		PriorityLevels                           dynamicconfig.IntPropertyFnWithTaskQueueFilter

		RateLimiterRefreshInterval    time.Duration
		FairnessKeyRateLimitCacheSize dynamicconfig.IntPropertyFnWithTaskQueueFilter

		// Time to hold a poll request before returning an empty response if there are no tasks
		LongPollExpirationInterval dynamicconfig.DurationPropertyFnWithTaskQueueFilter
		BacklogTaskForwardTimeout  dynamicconfig.DurationPropertyFnWithTaskQueueFilter
		MinTaskThrottlingBurstSize dynamicconfig.IntPropertyFnWithTaskQueueFilter
		MaxTaskDeleteBatchSize     dynamicconfig.IntPropertyFnWithTaskQueueFilter
		TaskDeleteInterval         dynamicconfig.DurationPropertyFnWithTaskQueueFilter

		// taskWriter configuration
		OutstandingTaskAppendsThreshold dynamicconfig.IntPropertyFnWithTaskQueueFilter
		MaxTaskBatchSize                dynamicconfig.IntPropertyFnWithTaskQueueFilter

		ThrottledLogRPS dynamicconfig.IntPropertyFn

		AdminNamespaceToPartitionDispatchRate          dynamicconfig.FloatPropertyFnWithNamespaceFilter
		AdminNamespaceToPartitionRateSub               dynamicconfig.TypedSubscribableWithNamespaceFilter[float64]
		AdminNamespaceTaskqueueToPartitionDispatchRate dynamicconfig.FloatPropertyFnWithTaskQueueFilter
		AdminNamespaceTaskqueueToPartitionRateSub      dynamicconfig.TypedSubscribableWithTaskQueueFilter[float64]

		VisibilityPersistenceMaxReadQPS         dynamicconfig.IntPropertyFn
		VisibilityPersistenceMaxWriteQPS        dynamicconfig.IntPropertyFn
		VisibilityPersistenceSlowQueryThreshold dynamicconfig.DurationPropertyFn
		EnableReadFromSecondaryVisibility       dynamicconfig.BoolPropertyFnWithNamespaceFilter
		VisibilityEnableShadowReadMode          dynamicconfig.BoolPropertyFn
		VisibilityDisableOrderByClause          dynamicconfig.BoolPropertyFnWithNamespaceFilter
		VisibilityEnableManualPagination        dynamicconfig.BoolPropertyFnWithNamespaceFilter

		ListNexusEndpointsLongPollTimeout dynamicconfig.DurationPropertyFn
		NexusEndpointsRefreshInterval     dynamicconfig.DurationPropertyFn

		PollerScalingBacklogAgeScaleUp  dynamicconfig.DurationPropertyFnWithTaskQueueFilter
		PollerScalingWaitTime           dynamicconfig.DurationPropertyFnWithTaskQueueFilter
		PollerScalingDecisionsPerSecond dynamicconfig.FloatPropertyFnWithTaskQueueFilter

		FairnessCounter dynamicconfig.TypedPropertyFnWithTaskQueueFilter[counter.CounterParams]

		LogAllReqErrors dynamicconfig.BoolPropertyFnWithNamespaceFilter
	}

	forwarderConfig struct {
		ForwarderMaxOutstandingPolls func() int
		ForwarderMaxOutstandingTasks func() int
		ForwarderMaxRatePerSecond    func() float64
		ForwarderMaxChildrenPerNode  func() int
	}

	taskQueueConfig struct {
		forwarderConfig
		SyncMatchWaitDuration        func() time.Duration
		BacklogNegligibleAge         func() time.Duration
		MaxWaitForPollerBeforeFwd    func() time.Duration
		QueryPollerUnavailableWindow func() time.Duration
		// Time to hold a poll request before returning an empty response if there are no tasks
		LongPollExpirationInterval func() time.Duration
		BacklogTaskForwardTimeout  func() time.Duration
		RangeSize                  int64
		NewMatcher                 func(func(bool)) (bool, func())
		EnableFairness             func(func(bool)) (bool, func())
		GetTasksBatchSize          func() int
		GetTasksReloadAt           func() int
		UpdateAckInterval          func() time.Duration
		MaxTaskQueueIdleTime       func() time.Duration
		MinTaskThrottlingBurstSize func() int
		MaxTaskDeleteBatchSize     func() int
		TaskDeleteInterval         func() time.Duration
		PriorityLevels             func() int32

		GetUserDataLongPollTimeout dynamicconfig.DurationPropertyFn
		GetUserDataMinWaitTime     time.Duration
		GetUserDataReturnBudget    time.Duration
		GetUserDataInitialRefresh  time.Duration
		GetUserDataRefresh         dynamicconfig.DurationPropertyFn

		// taskWriter configuration
		OutstandingTaskAppendsThreshold func() int
		MaxTaskBatchSize                func() int
		NumWritePartitions              func() int
		NumReadPartitions               func() int

		// partition qps = AdminNamespaceToPartitionDispatchRate(namespace)
		AdminNamespaceToPartitionDispatchRate func() float64
		AdminNamespaceToPartitionRateSub      func(func(float64)) (float64, func())
		// partition qps = AdminNamespaceTaskQueueToPartitionDispatchRate(namespace, task_queue)
		AdminNamespaceTaskQueueToPartitionDispatchRate func() float64
		AdminNamespaceTaskQueueToPartitionRateSub      func(func(float64)) (float64, func())

		// Retry policy for fetching user data from root partition. Should retry forever.
		GetUserDataRetryPolicy backoff.RetryPolicy

		// TTL for cache holding TaskQueueInfoByBuildID
		TaskQueueInfoByBuildIdTTL func() time.Duration

		// Rate limiting
		RateLimiterRefreshInterval    time.Duration
		FairnessKeyRateLimitCacheSize func() int

		BreakdownMetricsByTaskQueue func() bool
		BreakdownMetricsByPartition func() bool
		BreakdownMetricsByBuildID   func() bool

		PollerHistoryTTL func() time.Duration

		// Poller scaling decisions configuration
		PollerScalingBacklogAgeScaleUp  func() time.Duration
		PollerScalingWaitTime           func() time.Duration
		PollerScalingDecisionsPerSecond func() float64

		FairnessCounter func() counter.CounterParams

		loadCause loadCause
	}

	loadCause   int
	unloadCause int
)

const (
	loadCauseUnspecified loadCause = iota
	loadCauseTask
	loadCauseQuery
	loadCauseDescribe
	loadCauseUserData
	loadCauseNexusTask
	loadCausePoll
	loadCauseOtherRead  // any other read-only rpc
	loadCauseOtherWrite // any other mutating rpc
	loadCauseForce      // root partition loaded, force load to ensure matching with back logged partitions
)

const (
	unloadCauseUnspecified unloadCause = iota
	unloadCauseInitError
	unloadCauseIdle
	unloadCauseMembership // proactive unload due to ownership change
	unloadCauseConflict   // reactive unload due to other node stealing ownership
	unloadCauseShuttingDown
	unloadCauseForce
	unloadCauseConfigChange
	unloadCauseOtherError
)

// NewConfig returns new service config with default values
func NewConfig(
	dc *dynamicconfig.Collection,
) *Config {
	return &Config{
		PersistenceMaxQPS:                        dynamicconfig.MatchingPersistenceMaxQPS.Get(dc),
		PersistenceGlobalMaxQPS:                  dynamicconfig.MatchingPersistenceGlobalMaxQPS.Get(dc),
		PersistenceNamespaceMaxQPS:               dynamicconfig.MatchingPersistenceNamespaceMaxQPS.Get(dc),
		PersistenceGlobalNamespaceMaxQPS:         dynamicconfig.MatchingPersistenceGlobalNamespaceMaxQPS.Get(dc),
		PersistencePerShardNamespaceMaxQPS:       dynamicconfig.DefaultPerShardNamespaceRPSMax,
		PersistenceDynamicRateLimitingParams:     dynamicconfig.MatchingPersistenceDynamicRateLimitingParams.Get(dc),
		PersistenceQPSBurstRatio:                 dynamicconfig.PersistenceQPSBurstRatio.Get(dc),
		SyncMatchWaitDuration:                    dynamicconfig.MatchingSyncMatchWaitDuration.Get(dc),
		HistoryMaxPageSize:                       dynamicconfig.MatchingHistoryMaxPageSize.Get(dc),
		EnableDeployments:                        dynamicconfig.EnableDeployments.Get(dc), // [cleanup-wv-pre-release]
		EnableDeploymentVersions:                 dynamicconfig.EnableDeploymentVersions.Get(dc),
		MaxTaskQueuesInDeployment:                dynamicconfig.MatchingMaxTaskQueuesInDeployment.Get(dc),
		RPS:                                      dynamicconfig.MatchingRPS.Get(dc),
		OperatorRPSRatio:                         dynamicconfig.OperatorRPSRatio.Get(dc),
		RangeSize:                                100000,
		NewMatcher:                               dynamicconfig.MatchingUseNewMatcher.Subscribe(dc),
		EnableFairness:                           dynamicconfig.MatchingEnableFairness.Subscribe(dc),
		GetTasksBatchSize:                        dynamicconfig.MatchingGetTasksBatchSize.Get(dc),
		GetTasksReloadAt:                         dynamicconfig.MatchingGetTasksReloadAt.Get(dc),
		UpdateAckInterval:                        dynamicconfig.MatchingUpdateAckInterval.Get(dc),
		MaxTaskQueueIdleTime:                     dynamicconfig.MatchingMaxTaskQueueIdleTime.Get(dc),
		LongPollExpirationInterval:               dynamicconfig.MatchingLongPollExpirationInterval.Get(dc),
		BacklogTaskForwardTimeout:                dynamicconfig.MatchingBacklogTaskForwardTimeout.Get(dc),
		MinTaskThrottlingBurstSize:               dynamicconfig.MatchingMinTaskThrottlingBurstSize.Get(dc),
		MaxTaskDeleteBatchSize:                   dynamicconfig.MatchingMaxTaskDeleteBatchSize.Get(dc),
		TaskDeleteInterval:                       dynamicconfig.MatchingTaskDeleteInterval.Get(dc),
		OutstandingTaskAppendsThreshold:          dynamicconfig.MatchingOutstandingTaskAppendsThreshold.Get(dc),
		MaxTaskBatchSize:                         dynamicconfig.MatchingMaxTaskBatchSize.Get(dc),
		ThrottledLogRPS:                          dynamicconfig.MatchingThrottledLogRPS.Get(dc),
		NumTaskqueueWritePartitions:              dynamicconfig.MatchingNumTaskqueueWritePartitions.Get(dc),
		NumTaskqueueReadPartitions:               dynamicconfig.MatchingNumTaskqueueReadPartitions.Get(dc),
		BreakdownMetricsByTaskQueue:              dynamicconfig.MetricsBreakdownByTaskQueue.Get(dc),
		BreakdownMetricsByPartition:              dynamicconfig.MetricsBreakdownByPartition.Get(dc),
		BreakdownMetricsByBuildID:                dynamicconfig.MetricsBreakdownByBuildID.Get(dc),
		ForwarderMaxOutstandingPolls:             dynamicconfig.MatchingForwarderMaxOutstandingPolls.Get(dc),
		ForwarderMaxOutstandingTasks:             dynamicconfig.MatchingForwarderMaxOutstandingTasks.Get(dc),
		ForwarderMaxRatePerSecond:                dynamicconfig.MatchingForwarderMaxRatePerSecond.Get(dc),
		ForwarderMaxChildrenPerNode:              dynamicconfig.MatchingForwarderMaxChildrenPerNode.Get(dc),
		AlignMembershipChange:                    dynamicconfig.MatchingAlignMembershipChange.Get(dc),
		ShutdownDrainDuration:                    dynamicconfig.MatchingShutdownDrainDuration.Get(dc),
		VersionCompatibleSetLimitPerQueue:        dynamicconfig.VersionCompatibleSetLimitPerQueue.Get(dc),
		VersionBuildIdLimitPerQueue:              dynamicconfig.VersionBuildIdLimitPerQueue.Get(dc),
		AssignmentRuleLimitPerQueue:              dynamicconfig.AssignmentRuleLimitPerQueue.Get(dc),
		RedirectRuleLimitPerQueue:                dynamicconfig.RedirectRuleLimitPerQueue.Get(dc),
		RedirectRuleMaxUpstreamBuildIDsPerQueue:  dynamicconfig.RedirectRuleMaxUpstreamBuildIDsPerQueue.Get(dc),
		DeletedRuleRetentionTime:                 dynamicconfig.MatchingDeletedRuleRetentionTime.Get(dc),
		PollerHistoryTTL:                         dynamicconfig.PollerHistoryTTL.Get(dc),
		ReachabilityBuildIdVisibilityGracePeriod: dynamicconfig.ReachabilityBuildIdVisibilityGracePeriod.Get(dc),
		ReachabilityCacheOpenWFsTTL:              dynamicconfig.ReachabilityCacheOpenWFsTTL.Get(dc),
		ReachabilityCacheClosedWFsTTL:            dynamicconfig.ReachabilityCacheClosedWFsTTL.Get(dc),
		TaskQueueLimitPerBuildId:                 dynamicconfig.TaskQueuesPerBuildIdLimit.Get(dc),
		GetUserDataLongPollTimeout:               dynamicconfig.MatchingGetUserDataLongPollTimeout.Get(dc), // Use -10 seconds so that we send back empty response instead of timeout
		GetUserDataRefresh:                       dynamicconfig.MatchingGetUserDataRefresh.Get(dc),
		BacklogNegligibleAge:                     dynamicconfig.MatchingBacklogNegligibleAge.Get(dc),
		MaxWaitForPollerBeforeFwd:                dynamicconfig.MatchingMaxWaitForPollerBeforeFwd.Get(dc),
		QueryPollerUnavailableWindow:             dynamicconfig.QueryPollerUnavailableWindow.Get(dc),
		QueryWorkflowTaskTimeoutLogRate:          dynamicconfig.MatchingQueryWorkflowTaskTimeoutLogRate.Get(dc),
		MembershipUnloadDelay:                    dynamicconfig.MatchingMembershipUnloadDelay.Get(dc),
		TaskQueueInfoByBuildIdTTL:                dynamicconfig.TaskQueueInfoByBuildIdTTL.Get(dc),
		PriorityLevels:                           dynamicconfig.MatchingPriorityLevels.Get(dc),
		RateLimiterRefreshInterval:               time.Minute,
		FairnessKeyRateLimitCacheSize:            dynamicconfig.MatchingFairnessKeyRateLimitCacheSize.Get(dc),
		MaxIDLengthLimit:                         dynamicconfig.MaxIDLengthLimit.Get(dc),

		AdminNamespaceToPartitionDispatchRate:          dynamicconfig.AdminMatchingNamespaceToPartitionDispatchRate.Get(dc),
		AdminNamespaceToPartitionRateSub:               dynamicconfig.AdminMatchingNamespaceToPartitionDispatchRate.Subscribe(dc),
		AdminNamespaceTaskqueueToPartitionDispatchRate: dynamicconfig.AdminMatchingNamespaceTaskqueueToPartitionDispatchRate.Get(dc),
		AdminNamespaceTaskqueueToPartitionRateSub:      dynamicconfig.AdminMatchingNamespaceTaskqueueToPartitionDispatchRate.Subscribe(dc),

		VisibilityPersistenceMaxReadQPS:         dynamicconfig.VisibilityPersistenceMaxReadQPS.Get(dc),
		VisibilityPersistenceMaxWriteQPS:        dynamicconfig.VisibilityPersistenceMaxWriteQPS.Get(dc),
		VisibilityPersistenceSlowQueryThreshold: dynamicconfig.VisibilityPersistenceSlowQueryThreshold.Get(dc),
		EnableReadFromSecondaryVisibility:       dynamicconfig.EnableReadFromSecondaryVisibility.Get(dc),
		VisibilityEnableShadowReadMode:          dynamicconfig.VisibilityEnableShadowReadMode.Get(dc),
		VisibilityDisableOrderByClause:          dynamicconfig.VisibilityDisableOrderByClause.Get(dc),
		VisibilityEnableManualPagination:        dynamicconfig.VisibilityEnableManualPagination.Get(dc),

		ListNexusEndpointsLongPollTimeout: dynamicconfig.MatchingListNexusEndpointsLongPollTimeout.Get(dc),
		NexusEndpointsRefreshInterval:     dynamicconfig.MatchingNexusEndpointsRefreshInterval.Get(dc),

		PollerScalingBacklogAgeScaleUp:  dynamicconfig.MatchingPollerScalingBacklogAgeScaleUp.Get(dc),
		PollerScalingWaitTime:           dynamicconfig.MatchingPollerScalingWaitTime.Get(dc),
		PollerScalingDecisionsPerSecond: dynamicconfig.MatchingPollerScalingDecisionsPerSecond.Get(dc),

		FairnessCounter: dynamicconfig.MatchingFairnessCounter.Get(dc),

		LogAllReqErrors: dynamicconfig.LogAllReqErrors.Get(dc),
	}
}

func newTaskQueueConfig(tq *tqid.TaskQueue, config *Config, ns namespace.Name) *taskQueueConfig {
	taskQueueName := tq.Name()
	taskType := tq.TaskType()

	return &taskQueueConfig{
		RangeSize: config.RangeSize,
		NewMatcher: func(cb func(bool)) (bool, func()) {
			return config.NewMatcher(ns.String(), taskQueueName, taskType, cb)
		},
		EnableFairness: func(cb func(bool)) (bool, func()) {
			return config.EnableFairness(ns.String(), taskQueueName, taskType, cb)
		},
		GetTasksBatchSize: func() int {
			return config.GetTasksBatchSize(ns.String(), taskQueueName, taskType)
		},
		GetTasksReloadAt: func() int {
			return config.GetTasksReloadAt(ns.String(), taskQueueName, taskType)
		},
		UpdateAckInterval: func() time.Duration {
			return config.UpdateAckInterval(ns.String(), taskQueueName, taskType)
		},
		MaxTaskQueueIdleTime: func() time.Duration {
			return config.MaxTaskQueueIdleTime(ns.String(), taskQueueName, taskType)
		},
		MinTaskThrottlingBurstSize: func() int {
			return config.MinTaskThrottlingBurstSize(ns.String(), taskQueueName, taskType)
		},
		SyncMatchWaitDuration: func() time.Duration {
			return config.SyncMatchWaitDuration(ns.String(), taskQueueName, taskType)
		},
		BacklogNegligibleAge: func() time.Duration {
			return config.BacklogNegligibleAge(ns.String(), taskQueueName, taskType)
		},
		MaxWaitForPollerBeforeFwd: func() time.Duration {
			return config.MaxWaitForPollerBeforeFwd(ns.String(), taskQueueName, taskType)
		},
		QueryPollerUnavailableWindow: config.QueryPollerUnavailableWindow,
		LongPollExpirationInterval: func() time.Duration {
			return config.LongPollExpirationInterval(ns.String(), taskQueueName, taskType)
		},
		BacklogTaskForwardTimeout: func() time.Duration {
			return config.BacklogTaskForwardTimeout(ns.String(), taskQueueName, taskType)
		},
		MaxTaskDeleteBatchSize: func() int {
			return config.MaxTaskDeleteBatchSize(ns.String(), taskQueueName, taskType)
		},
		TaskDeleteInterval: func() time.Duration {
			return config.TaskDeleteInterval(ns.String(), taskQueueName, taskType)
		},
		PriorityLevels: func() int32 {
			return int32(config.PriorityLevels(ns.String(), taskQueueName, taskType))
		},
		GetUserDataLongPollTimeout: config.GetUserDataLongPollTimeout,
		GetUserDataMinWaitTime:     1 * time.Second,
		GetUserDataReturnBudget:    returnEmptyTaskTimeBudget,
		GetUserDataInitialRefresh:  ioTimeout,
		GetUserDataRefresh:         config.GetUserDataRefresh,
		OutstandingTaskAppendsThreshold: func() int {
			return config.OutstandingTaskAppendsThreshold(ns.String(), taskQueueName, taskType)
		},
		MaxTaskBatchSize: func() int {
			return config.MaxTaskBatchSize(ns.String(), taskQueueName, taskType)
		},
		NumWritePartitions: func() int {
			return max(1, config.NumTaskqueueWritePartitions(ns.String(), taskQueueName, taskType))
		},
		NumReadPartitions: func() int {
			return max(1, config.NumTaskqueueReadPartitions(ns.String(), taskQueueName, taskType))
		},
		BreakdownMetricsByTaskQueue: func() bool {
			return config.BreakdownMetricsByTaskQueue(ns.String(), taskQueueName, taskType)
		},
		BreakdownMetricsByPartition: func() bool {
			return config.BreakdownMetricsByPartition(ns.String(), taskQueueName, taskType)
		},
		BreakdownMetricsByBuildID: func() bool {
			return config.BreakdownMetricsByBuildID(ns.String(), taskQueueName, taskType)
		},
		AdminNamespaceToPartitionDispatchRate: func() float64 {
			return config.AdminNamespaceToPartitionDispatchRate(ns.String())
		},
		AdminNamespaceToPartitionRateSub: func(cb func(float64)) (float64, func()) {
			return config.AdminNamespaceToPartitionRateSub(ns.String(), cb)
		},
		AdminNamespaceTaskQueueToPartitionDispatchRate: func() float64 {
			return config.AdminNamespaceTaskqueueToPartitionDispatchRate(ns.String(), taskQueueName, taskType)
		},
		AdminNamespaceTaskQueueToPartitionRateSub: func(cb func(float64)) (float64, func()) {
			return config.AdminNamespaceTaskqueueToPartitionRateSub(ns.String(), taskQueueName, taskType, cb)
		},
		forwarderConfig: forwarderConfig{
			ForwarderMaxOutstandingPolls: func() int {
				return config.ForwarderMaxOutstandingPolls(ns.String(), taskQueueName, taskType)
			},
			ForwarderMaxOutstandingTasks: func() int {
				return config.ForwarderMaxOutstandingTasks(ns.String(), taskQueueName, taskType)
			},
			ForwarderMaxRatePerSecond: func() float64 {
				return config.ForwarderMaxRatePerSecond(ns.String(), taskQueueName, taskType)
			},
			ForwarderMaxChildrenPerNode: func() int {
				return max(1, config.ForwarderMaxChildrenPerNode(ns.String(), taskQueueName, taskType))
			},
		},
		GetUserDataRetryPolicy: backoff.NewExponentialRetryPolicy(1 * time.Second).WithMaximumInterval(5 * time.Minute).WithExpirationInterval(backoff.NoInterval),
		TaskQueueInfoByBuildIdTTL: func() time.Duration {
			return config.TaskQueueInfoByBuildIdTTL(ns.String(), taskQueueName, taskType)
		},
		RateLimiterRefreshInterval: config.RateLimiterRefreshInterval,
		FairnessKeyRateLimitCacheSize: func() int {
			return config.FairnessKeyRateLimitCacheSize(ns.String(), taskQueueName, taskType)
		},
		PollerHistoryTTL: func() time.Duration {
			return config.PollerHistoryTTL(ns.String())
		},
		PollerScalingBacklogAgeScaleUp: func() time.Duration {
			return config.PollerScalingBacklogAgeScaleUp(ns.String(), taskQueueName, taskType)
		},
		PollerScalingWaitTime: func() time.Duration {
			return config.PollerScalingWaitTime(ns.String(), taskQueueName, taskType)
		},
		PollerScalingDecisionsPerSecond: func() float64 {
			return config.PollerScalingDecisionsPerSecond(ns.String(), taskQueueName, taskType)
		},
		FairnessCounter: func() counter.CounterParams {
			return config.FairnessCounter(ns.String(), taskQueueName, taskType)
		},
	}
}

func defaultPriorityLevel(priorityLevels int32) int32 {
	return (priorityLevels + 1) / 2
}
