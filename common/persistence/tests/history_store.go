package tests

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	historypb "go.temporal.io/api/history/v1"
	persistencespb "go.temporal.io/server/api/persistence/v1"
	"go.temporal.io/server/common"
	"go.temporal.io/server/common/debug"
	"go.temporal.io/server/common/dynamicconfig"
	"go.temporal.io/server/common/log"
	p "go.temporal.io/server/common/persistence"
	"go.temporal.io/server/common/persistence/serialization"
	"go.temporal.io/server/common/testing/protorequire"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// TODO add UT for the following
//  * DeleteHistoryBranch
//  * GetHistoryTreeContainingBranch
//  * GetAllHistoryTreeBranches

type (
	HistoryEventsPacket struct {
		nodeID            int64
		transactionID     int64
		prevTransactionID int64
		events            []*historypb.HistoryEvent
	}

	HistoryEventsSuite struct {
		suite.Suite
		*require.Assertions
		protorequire.ProtoAssertions

		ShardID int32

		store      p.ExecutionManager
		serializer serialization.Serializer
		logger     log.Logger

		Ctx    context.Context
		Cancel context.CancelFunc
	}
)

func NewHistoryEventsSuite(
	t *testing.T,
	store p.ExecutionStore,
	logger log.Logger,
) *HistoryEventsSuite {
	eventSerializer := serialization.NewSerializer()
	return &HistoryEventsSuite{
		Assertions:      require.New(t),
		ProtoAssertions: protorequire.New(t),
		store: p.NewExecutionManager(
			store,
			eventSerializer,
			nil,
			logger,
			dynamicconfig.GetIntPropertyFn(4*1024*1024),
		),
		serializer: eventSerializer,
		logger:     logger,
	}
}

func (s *HistoryEventsSuite) SetupSuite() {

}

func (s *HistoryEventsSuite) TearDownSuite() {

}

func (s *HistoryEventsSuite) SetupTest() {
	s.Assertions = require.New(s.T())
	s.ProtoAssertions = protorequire.New(s.T())
	s.Ctx, s.Cancel = context.WithTimeout(context.Background(), 30*time.Second*debug.TimeoutMultiplier)

	s.ShardID++
}

func (s *HistoryEventsSuite) TearDownTest() {
	s.Cancel()
}

func (s *HistoryEventsSuite) TestAppendSelect_First() {
	treeID := uuid.New()
	branchID := uuid.New()
	branchToken, err := s.store.GetHistoryBranchUtil().NewHistoryBranch(
		uuid.New(),
		uuid.New(),
		uuid.New(),
		treeID,
		&branchID,
		[]*persistencespb.HistoryBranchRange{},
		time.Duration(0),
		time.Duration(0),
		time.Duration(0),
	)
	s.NoError(err)

	eventsPacket := s.newHistoryEvents(
		[]int64{1, 2, 3},
		rand.Int63(),
		0,
	)
	s.appendHistoryEvents(s.ShardID, branchToken, eventsPacket)

	protorequire.ProtoSliceEqual(s.T(), eventsPacket.events, s.listHistoryEvents(s.ShardID, branchToken, common.FirstEventID, 4))
	protorequire.ProtoSliceEqual(s.T(), eventsPacket.events, s.listAllHistoryEvents(s.ShardID, branchToken))
}

func (s *HistoryEventsSuite) TestAppendSelect_NonShadowing() {
	treeID := uuid.New()
	branchID := uuid.New()
	branchToken, err := s.store.GetHistoryBranchUtil().NewHistoryBranch(
		uuid.New(),
		uuid.New(),
		uuid.New(),
		treeID,
		&branchID,
		[]*persistencespb.HistoryBranchRange{},
		time.Duration(0),
		time.Duration(0),
		time.Duration(0),
	)
	s.NoError(err)
	var events []*historypb.HistoryEvent

	eventsPacket0 := s.newHistoryEvents(
		[]int64{1, 2, 3},
		rand.Int63(),
		0,
	)
	s.appendHistoryEvents(s.ShardID, branchToken, eventsPacket0)
	events = append(events, eventsPacket0.events...)

	eventsPacket1 := s.newHistoryEvents(
		[]int64{4, 5},
		eventsPacket0.transactionID+1,
		eventsPacket0.transactionID,
	)
	s.appendHistoryEvents(s.ShardID, branchToken, eventsPacket1)
	events = append(events, eventsPacket1.events...)

	protorequire.ProtoSliceEqual(s.T(), eventsPacket0.events, s.listHistoryEvents(s.ShardID, branchToken, common.FirstEventID, 4))
	protorequire.ProtoSliceEqual(s.T(), eventsPacket1.events, s.listHistoryEvents(s.ShardID, branchToken, 4, 6))
	protorequire.ProtoSliceEqual(s.T(), events, s.listAllHistoryEvents(s.ShardID, branchToken))
}

func (s *HistoryEventsSuite) TestAppendSelect_Shadowing() {
	treeID := uuid.New()
	branchID := uuid.New()
	branchToken, err := s.store.GetHistoryBranchUtil().NewHistoryBranch(
		uuid.New(),
		uuid.New(),
		uuid.New(),
		treeID,
		&branchID,
		[]*persistencespb.HistoryBranchRange{},
		time.Duration(0),
		time.Duration(0),
		time.Duration(0),
	)
	s.NoError(err)
	var events0 []*historypb.HistoryEvent
	var events1 []*historypb.HistoryEvent

	eventsPacket0 := s.newHistoryEvents(
		[]int64{1, 2, 3},
		rand.Int63(),
		0,
	)
	s.appendHistoryEvents(s.ShardID, branchToken, eventsPacket0)
	events0 = append(events0, eventsPacket0.events...)
	events1 = append(events1, eventsPacket0.events...)

	eventsPacket10 := s.newHistoryEvents(
		[]int64{4, 5},
		eventsPacket0.transactionID+1,
		eventsPacket0.transactionID,
	)
	s.appendHistoryEvents(s.ShardID, branchToken, eventsPacket10)
	events0 = append(events0, eventsPacket10.events...)

	protorequire.ProtoSliceEqual(s.T(), events0, s.listAllHistoryEvents(s.ShardID, branchToken))

	eventsPacket11 := s.newHistoryEvents(
		[]int64{4, 5},
		eventsPacket0.transactionID+2,
		eventsPacket0.transactionID,
	)
	s.appendHistoryEvents(s.ShardID, branchToken, eventsPacket11)
	events1 = append(events1, eventsPacket11.events...)

	protorequire.ProtoSliceEqual(s.T(), eventsPacket0.events, s.listHistoryEvents(s.ShardID, branchToken, common.FirstEventID, 4))
	protorequire.ProtoSliceEqual(s.T(), eventsPacket11.events, s.listHistoryEvents(s.ShardID, branchToken, 4, 6))
	protorequire.ProtoSliceEqual(s.T(), events1, s.listAllHistoryEvents(s.ShardID, branchToken))
}

func (s *HistoryEventsSuite) TestAppendForkSelect_NoShadowing() {
	treeID := uuid.New()
	branchID := uuid.New()
	branchToken, err := s.store.GetHistoryBranchUtil().NewHistoryBranch(
		uuid.New(),
		uuid.New(),
		uuid.New(),
		treeID,
		&branchID,
		[]*persistencespb.HistoryBranchRange{},
		time.Duration(0),
		time.Duration(0),
		time.Duration(0),
	)
	s.NoError(err)
	var events0 []*historypb.HistoryEvent
	var events1 []*historypb.HistoryEvent

	eventsPacket0 := s.newHistoryEvents(
		[]int64{1, 2, 3},
		rand.Int63(),
		0,
	)
	s.appendHistoryEvents(s.ShardID, branchToken, eventsPacket0)
	events0 = append(events0, eventsPacket0.events...)
	events1 = append(events1, eventsPacket0.events...)

	eventsPacket10 := s.newHistoryEvents(
		[]int64{4, 5},
		eventsPacket0.transactionID+1,
		eventsPacket0.transactionID,
	)
	s.appendHistoryEvents(s.ShardID, branchToken, eventsPacket10)
	events0 = append(events0, eventsPacket10.events...)

	newBranchToken := s.forkHistoryBranch(s.ShardID, branchToken, 4)
	eventsPacket11 := s.newHistoryEvents(
		[]int64{4, 5},
		eventsPacket0.transactionID+2,
		eventsPacket0.transactionID,
	)
	s.appendHistoryEvents(s.ShardID, newBranchToken, eventsPacket11)
	events1 = append(events1, eventsPacket11.events...)

	protorequire.ProtoSliceEqual(s.T(), eventsPacket0.events, s.listHistoryEvents(s.ShardID, branchToken, common.FirstEventID, 4))
	protorequire.ProtoSliceEqual(s.T(), eventsPacket0.events, s.listHistoryEvents(s.ShardID, newBranchToken, common.FirstEventID, 4))
	protorequire.ProtoSliceEqual(s.T(), eventsPacket10.events, s.listHistoryEvents(s.ShardID, branchToken, 4, 6))
	protorequire.ProtoSliceEqual(s.T(), eventsPacket11.events, s.listHistoryEvents(s.ShardID, newBranchToken, 4, 6))
	protorequire.ProtoSliceEqual(s.T(), events0, s.listAllHistoryEvents(s.ShardID, branchToken))
	protorequire.ProtoSliceEqual(s.T(), events1, s.listAllHistoryEvents(s.ShardID, newBranchToken))
}

func (s *HistoryEventsSuite) TestAppendForkSelect_Shadowing_NonLastBranch() {
	treeID := uuid.New()
	branchID := uuid.New()
	branchToken, err := s.store.GetHistoryBranchUtil().NewHistoryBranch(
		uuid.New(),
		uuid.New(),
		uuid.New(),
		treeID,
		&branchID,
		[]*persistencespb.HistoryBranchRange{},
		time.Duration(0),
		time.Duration(0),
		time.Duration(0),
	)
	s.NoError(err)
	var events0 []*historypb.HistoryEvent
	var events1 []*historypb.HistoryEvent

	eventsPacket0 := s.newHistoryEvents(
		[]int64{1, 2, 3},
		rand.Int63(),
		0,
	)
	s.appendHistoryEvents(s.ShardID, branchToken, eventsPacket0)
	events0 = append(events0, eventsPacket0.events...)
	events1 = append(events1, eventsPacket0.events...)

	s.appendHistoryEvents(s.ShardID, branchToken, s.newHistoryEvents(
		[]int64{4, 5},
		eventsPacket0.transactionID+1,
		eventsPacket0.transactionID,
	))

	eventsPacket1 := s.newHistoryEvents(
		[]int64{4, 5},
		eventsPacket0.transactionID+2,
		eventsPacket0.transactionID,
	)
	s.appendHistoryEvents(s.ShardID, branchToken, eventsPacket1)
	events0 = append(events0, eventsPacket1.events...)
	events1 = append(events1, eventsPacket1.events...)

	eventsPacket20 := s.newHistoryEvents(
		[]int64{6},
		eventsPacket1.transactionID+1,
		eventsPacket1.transactionID,
	)
	s.appendHistoryEvents(s.ShardID, branchToken, eventsPacket20)
	events0 = append(events0, eventsPacket20.events...)

	newBranchToken := s.forkHistoryBranch(s.ShardID, branchToken, 6)
	eventsPacket21 := s.newHistoryEvents(
		[]int64{6},
		eventsPacket1.transactionID+2,
		eventsPacket1.transactionID,
	)
	s.appendHistoryEvents(s.ShardID, newBranchToken, eventsPacket21)
	events1 = append(events1, eventsPacket21.events...)

	protorequire.ProtoSliceEqual(s.T(), eventsPacket0.events, s.listHistoryEvents(s.ShardID, branchToken, common.FirstEventID, 4))
	protorequire.ProtoSliceEqual(s.T(), eventsPacket0.events, s.listHistoryEvents(s.ShardID, newBranchToken, common.FirstEventID, 4))
	protorequire.ProtoSliceEqual(s.T(), eventsPacket1.events, s.listHistoryEvents(s.ShardID, branchToken, 4, 6))
	protorequire.ProtoSliceEqual(s.T(), eventsPacket1.events, s.listHistoryEvents(s.ShardID, newBranchToken, 4, 6))
	protorequire.ProtoSliceEqual(s.T(), eventsPacket20.events, s.listHistoryEvents(s.ShardID, branchToken, 6, 7))
	protorequire.ProtoSliceEqual(s.T(), eventsPacket21.events, s.listHistoryEvents(s.ShardID, newBranchToken, 6, 7))
	protorequire.ProtoSliceEqual(s.T(), events0, s.listAllHistoryEvents(s.ShardID, branchToken))
	protorequire.ProtoSliceEqual(s.T(), events1, s.listAllHistoryEvents(s.ShardID, newBranchToken))
}

func (s *HistoryEventsSuite) TestAppendForkSelect_Shadowing_LastBranch() {
	treeID := uuid.New()
	branchID := uuid.New()
	branchToken, err := s.store.GetHistoryBranchUtil().NewHistoryBranch(
		uuid.New(),
		uuid.New(),
		uuid.New(),
		treeID,
		&branchID,
		[]*persistencespb.HistoryBranchRange{},
		time.Duration(0),
		time.Duration(0),
		time.Duration(0),
	)
	s.NoError(err)
	var events0 []*historypb.HistoryEvent
	var events1 []*historypb.HistoryEvent

	eventsPacket0 := s.newHistoryEvents(
		[]int64{1, 2, 3},
		rand.Int63(),
		0,
	)
	s.appendHistoryEvents(s.ShardID, branchToken, eventsPacket0)
	events0 = append(events0, eventsPacket0.events...)
	events1 = append(events1, eventsPacket0.events...)

	s.appendHistoryEvents(s.ShardID, branchToken, s.newHistoryEvents(
		[]int64{4, 5},
		eventsPacket0.transactionID+1,
		eventsPacket0.transactionID,
	))

	newBranchToken := s.forkHistoryBranch(s.ShardID, branchToken, 4)
	eventsPacket20 := s.newHistoryEvents(
		[]int64{4, 5},
		eventsPacket0.transactionID+2,
		eventsPacket0.transactionID,
	)
	s.appendHistoryEvents(s.ShardID, newBranchToken, eventsPacket20)
	events0 = append(events0, eventsPacket20.events...)

	protorequire.ProtoSliceEqual(s.T(), eventsPacket0.events, s.listHistoryEvents(s.ShardID, newBranchToken, common.FirstEventID, 4))
	protorequire.ProtoSliceEqual(s.T(), eventsPacket20.events, s.listHistoryEvents(s.ShardID, newBranchToken, 4, 6))
	protorequire.ProtoSliceEqual(s.T(), events0, s.listAllHistoryEvents(s.ShardID, newBranchToken))

	eventsPacket21 := s.newHistoryEvents(
		[]int64{4, 5},
		eventsPacket0.transactionID+3,
		eventsPacket0.transactionID,
	)
	s.appendHistoryEvents(s.ShardID, newBranchToken, eventsPacket21)
	events1 = append(events1, eventsPacket21.events...)

	protorequire.ProtoSliceEqual(s.T(), eventsPacket0.events, s.listHistoryEvents(s.ShardID, newBranchToken, common.FirstEventID, 4))
	protorequire.ProtoSliceEqual(s.T(), eventsPacket21.events, s.listHistoryEvents(s.ShardID, newBranchToken, 4, 6))
	protorequire.ProtoSliceEqual(s.T(), events1, s.listAllHistoryEvents(s.ShardID, newBranchToken))
}

func (s *HistoryEventsSuite) TestAppendSelectTrim() {
	treeID := uuid.New()
	branchID := uuid.New()
	branchToken, err := s.store.GetHistoryBranchUtil().NewHistoryBranch(
		uuid.New(),
		uuid.New(),
		uuid.New(),
		treeID,
		&branchID,
		[]*persistencespb.HistoryBranchRange{},
		time.Duration(0),
		time.Duration(0),
		time.Duration(0),
	)
	s.NoError(err)
	var events []*historypb.HistoryEvent

	eventsPacket0 := s.newHistoryEvents(
		[]int64{1, 2, 3},
		rand.Int63(),
		0,
	)
	s.appendHistoryEvents(s.ShardID, branchToken, eventsPacket0)
	events = append(events, eventsPacket0.events...)

	eventsPacket1 := s.newHistoryEvents(
		[]int64{4, 5},
		eventsPacket0.transactionID+1,
		eventsPacket0.transactionID,
	)
	s.appendHistoryEvents(s.ShardID, branchToken, eventsPacket1)
	events = append(events, eventsPacket1.events...)

	s.appendHistoryEvents(s.ShardID, branchToken, s.newHistoryEvents(
		[]int64{4, 5},
		eventsPacket0.transactionID+2,
		eventsPacket0.transactionID,
	))

	s.trimHistoryBranch(s.ShardID, branchToken, eventsPacket1.nodeID, eventsPacket1.transactionID)

	protorequire.ProtoSliceEqual(s.T(), events, s.listAllHistoryEvents(s.ShardID, branchToken))
}

func (s *HistoryEventsSuite) TestAppendForkSelectTrim_NonLastBranch() {
	treeID := uuid.New()
	branchID := uuid.New()
	branchToken, err := s.store.GetHistoryBranchUtil().NewHistoryBranch(
		uuid.New(),
		uuid.New(),
		uuid.New(),
		treeID,
		&branchID,
		[]*persistencespb.HistoryBranchRange{},
		time.Duration(0),
		time.Duration(0),
		time.Duration(0),
	)
	s.NoError(err)
	var events0 []*historypb.HistoryEvent
	var events1 []*historypb.HistoryEvent

	eventsPacket0 := s.newHistoryEvents(
		[]int64{1, 2, 3},
		rand.Int63(),
		0,
	)
	s.appendHistoryEvents(s.ShardID, branchToken, eventsPacket0)
	events0 = append(events0, eventsPacket0.events...)
	events1 = append(events1, eventsPacket0.events...)

	eventsPacket1 := s.newHistoryEvents(
		[]int64{4, 5},
		eventsPacket0.transactionID+1,
		eventsPacket0.transactionID,
	)
	s.appendHistoryEvents(s.ShardID, branchToken, eventsPacket1)
	events0 = append(events0, eventsPacket1.events...)
	events1 = append(events1, eventsPacket1.events...)

	s.appendHistoryEvents(s.ShardID, branchToken, s.newHistoryEvents(
		[]int64{4, 5},
		eventsPacket0.transactionID+2,
		eventsPacket0.transactionID,
	))

	eventsPacket20 := s.newHistoryEvents(
		[]int64{6},
		eventsPacket1.transactionID+2,
		eventsPacket1.transactionID,
	)
	s.appendHistoryEvents(s.ShardID, branchToken, eventsPacket20)
	events0 = append(events0, eventsPacket20.events...)

	newBranchToken := s.forkHistoryBranch(s.ShardID, branchToken, 6)
	eventsPacket21 := s.newHistoryEvents(
		[]int64{6},
		eventsPacket1.transactionID+3,
		eventsPacket1.transactionID,
	)
	s.appendHistoryEvents(s.ShardID, newBranchToken, eventsPacket21)
	events1 = append(events1, eventsPacket21.events...)

	if rand.Intn(2)%2 == 0 {
		s.trimHistoryBranch(s.ShardID, branchToken, eventsPacket20.nodeID, eventsPacket20.transactionID)
	} else {
		s.trimHistoryBranch(s.ShardID, newBranchToken, eventsPacket21.nodeID, eventsPacket21.transactionID)
	}

	protorequire.ProtoSliceEqual(s.T(), events0, s.listAllHistoryEvents(s.ShardID, branchToken))
	protorequire.ProtoSliceEqual(s.T(), events1, s.listAllHistoryEvents(s.ShardID, newBranchToken))
}

func (s *HistoryEventsSuite) TestAppendForkSelectTrim_LastBranch() {
	treeID := uuid.New()
	branchID := uuid.New()
	branchToken, err := s.store.GetHistoryBranchUtil().NewHistoryBranch(
		uuid.New(),
		uuid.New(),
		uuid.New(),
		treeID,
		&branchID,
		[]*persistencespb.HistoryBranchRange{},
		time.Duration(0),
		time.Duration(0),
		time.Duration(0),
	)
	s.NoError(err)
	var events []*historypb.HistoryEvent

	eventsPacket0 := s.newHistoryEvents(
		[]int64{1, 2, 3},
		rand.Int63(),
		0,
	)
	s.appendHistoryEvents(s.ShardID, branchToken, eventsPacket0)
	events = append(events, eventsPacket0.events...)

	s.appendHistoryEvents(s.ShardID, branchToken, s.newHistoryEvents(
		[]int64{4, 5},
		eventsPacket0.transactionID+1,
		eventsPacket0.transactionID,
	))

	newBranchToken := s.forkHistoryBranch(s.ShardID, branchToken, 4)
	eventsPacket1 := s.newHistoryEvents(
		[]int64{4, 5},
		eventsPacket0.transactionID+2,
		eventsPacket0.transactionID,
	)
	s.appendHistoryEvents(s.ShardID, newBranchToken, eventsPacket1)
	events = append(events, eventsPacket1.events...)

	s.appendHistoryEvents(s.ShardID, newBranchToken, s.newHistoryEvents(
		[]int64{4, 5},
		eventsPacket0.transactionID+3,
		eventsPacket0.transactionID,
	))

	s.trimHistoryBranch(s.ShardID, newBranchToken, eventsPacket1.nodeID, eventsPacket1.transactionID)

	protorequire.ProtoSliceEqual(s.T(), events, s.listAllHistoryEvents(s.ShardID, newBranchToken))
}

func (s *HistoryEventsSuite) TestAppendBatches() {
	treeID := uuid.New()
	branchID := uuid.New()
	branchToken, err := s.store.GetHistoryBranchUtil().NewHistoryBranch(
		uuid.New(),
		uuid.New(),
		uuid.New(),
		treeID,
		&branchID,
		[]*persistencespb.HistoryBranchRange{},
		time.Duration(0),
		time.Duration(0),
		time.Duration(0),
	)
	s.NoError(err)

	eventsPacket1 := s.newHistoryEvents(
		[]int64{1, 2, 3},
		rand.Int63(),
		0,
	)
	eventsPacket2 := s.newHistoryEvents(
		[]int64{4, 5},
		eventsPacket1.transactionID+100,
		eventsPacket1.transactionID,
	)
	eventsPacket3 := s.newHistoryEvents(
		[]int64{6},
		eventsPacket2.transactionID+100,
		eventsPacket2.transactionID,
	)

	s.appendRawHistoryBatches(s.ShardID, branchToken, eventsPacket1)
	s.appendRawHistoryBatches(s.ShardID, branchToken, eventsPacket2)
	s.appendRawHistoryBatches(s.ShardID, branchToken, eventsPacket3)
	protorequire.ProtoSliceEqual(s.T(), eventsPacket1.events, s.listHistoryEvents(s.ShardID, branchToken, common.FirstEventID, 4))
	expectedEvents := append(eventsPacket1.events, append(eventsPacket2.events, eventsPacket3.events...)...)
	events := s.listAllHistoryEvents(s.ShardID, branchToken)
	protorequire.ProtoSliceEqual(s.T(), expectedEvents, events)
}

func (s *HistoryEventsSuite) TestForkDeleteBranch_DeleteBaseBranchFirst() {
	treeID := uuid.New()
	branchID := uuid.New()
	br1Token, err := s.store.GetHistoryBranchUtil().NewHistoryBranch(
		uuid.New(),
		uuid.New(),
		uuid.New(),
		treeID,
		&branchID,
		[]*persistencespb.HistoryBranchRange{},
		time.Duration(0),
		time.Duration(0),
		time.Duration(0),
	)
	s.NoError(err)

	eventsPacket0 := s.newHistoryEvents(
		[]int64{1, 2, 3},
		rand.Int63(),
		0,
	)
	s.appendHistoryEvents(s.ShardID, br1Token, eventsPacket0)

	s.appendHistoryEvents(s.ShardID, br1Token, s.newHistoryEvents(
		[]int64{4, 5},
		eventsPacket0.transactionID+1,
		eventsPacket0.transactionID,
	))

	br2Token := s.forkHistoryBranch(s.ShardID, br1Token, 4)
	eventsPacket1 := s.newHistoryEvents(
		[]int64{4, 5},
		eventsPacket0.transactionID+2,
		eventsPacket0.transactionID,
	)
	s.appendHistoryEvents(s.ShardID, br2Token, eventsPacket1)

	// delete branch1, should only delete branch1:[4,5], keep branch1:[1,2,3] as it is used as ancestor by branch2
	s.deleteHistoryBranch(s.ShardID, br1Token)
	// verify branch1:[1,2,3] still remains
	protorequire.ProtoSliceEqual(s.T(), eventsPacket0.events, s.listAllHistoryEvents(s.ShardID, br1Token))
	// verify branch2 is not affected
	protorequire.ProtoSliceEqual(s.T(), append(eventsPacket0.events, eventsPacket1.events...), s.listAllHistoryEvents(s.ShardID, br2Token))

	// delete branch2, should delete branch2:[4,5], and also should delete ancestor branch1:[1,2,3] as it is no longer
	// used by anyone
	s.deleteHistoryBranch(s.ShardID, br2Token)

	// at this point, both branch1 and branch2 are deleted.
	_, err = s.store.ReadHistoryBranch(s.Ctx, &p.ReadHistoryBranchRequest{
		ShardID:     s.ShardID,
		BranchToken: br1Token,
		MinEventID:  common.FirstEventID,
		MaxEventID:  common.LastEventID,
		PageSize:    1,
	})
	s.Error(err, "Workflow execution history not found.")

	_, err = s.store.ReadHistoryBranch(s.Ctx, &p.ReadHistoryBranchRequest{
		ShardID:     s.ShardID,
		BranchToken: br2Token,
		MinEventID:  common.FirstEventID,
		MaxEventID:  common.LastEventID,
		PageSize:    1,
	})
	s.Error(err, "Workflow execution history not found.")
}

func (s *HistoryEventsSuite) TestForkDeleteBranch_DeleteForkedBranchFirst() {
	treeID := uuid.New()
	branchID := uuid.New()
	br1Token, err := s.store.GetHistoryBranchUtil().NewHistoryBranch(
		uuid.New(),
		uuid.New(),
		uuid.New(),
		treeID,
		&branchID,
		[]*persistencespb.HistoryBranchRange{},
		time.Duration(0),
		time.Duration(0),
		time.Duration(0),
	)
	s.NoError(err)

	transactionID := rand.Int63()
	eventsPacket0 := s.newHistoryEvents(
		[]int64{1, 2, 3},
		transactionID,
		0,
	)
	s.appendHistoryEvents(s.ShardID, br1Token, eventsPacket0)
	eventsPacket1 := s.newHistoryEvents(
		[]int64{4, 5},
		transactionID+1,
		transactionID,
	)
	s.appendHistoryEvents(s.ShardID, br1Token, eventsPacket1)

	br2Token := s.forkHistoryBranch(s.ShardID, br1Token, 4)
	s.appendHistoryEvents(s.ShardID, br2Token, s.newHistoryEvents(
		[]int64{4, 5},
		transactionID+2,
		transactionID,
	))

	// delete branch2, should only delete branch2:[4,5], keep branch1:[1,2,3] [4,5] as it is by branch1
	s.deleteHistoryBranch(s.ShardID, br2Token)
	// verify branch1 is not affected
	protorequire.ProtoSliceEqual(s.T(), append(eventsPacket0.events, eventsPacket1.events...), s.listAllHistoryEvents(s.ShardID, br1Token))

	// branch2:[4,5] should be deleted
	_, err = s.store.ReadHistoryBranch(s.Ctx, &p.ReadHistoryBranchRequest{
		ShardID:     s.ShardID,
		BranchToken: br2Token,
		MinEventID:  4,
		MaxEventID:  common.LastEventID,
		PageSize:    1,
	})
	s.Error(err, "Workflow execution history not found.")

	// delete branch1, should delete branch1:[1,2,3] [4,5]
	s.deleteHistoryBranch(s.ShardID, br1Token)

	// branch1 should be deleted
	_, err = s.store.ReadHistoryBranch(s.Ctx, &p.ReadHistoryBranchRequest{
		ShardID:     s.ShardID,
		BranchToken: br1Token,
		MinEventID:  common.FirstEventID,
		MaxEventID:  common.LastEventID,
		PageSize:    1,
	})
	s.Error(err, "Workflow execution history not found.")
}

func (s *HistoryEventsSuite) appendHistoryEvents(
	shardID int32,
	branchToken []byte,
	packet HistoryEventsPacket,
) {
	_, err := s.store.AppendHistoryNodes(s.Ctx, &p.AppendHistoryNodesRequest{
		ShardID:           shardID,
		BranchToken:       branchToken,
		Events:            packet.events,
		TransactionID:     packet.transactionID,
		PrevTransactionID: packet.prevTransactionID,
		IsNewBranch:       packet.nodeID == common.FirstEventID,
		Info:              "",
	})
	s.NoError(err)
}

func (s *HistoryEventsSuite) appendRawHistoryBatches(
	shardID int32,
	branchToken []byte,
	packet HistoryEventsPacket,
) {
	blob, err := s.serializer.SerializeEvents(packet.events)
	s.NoError(err)
	_, err = s.store.AppendRawHistoryNodes(s.Ctx, &p.AppendRawHistoryNodesRequest{
		ShardID:           shardID,
		BranchToken:       branchToken,
		NodeID:            packet.nodeID,
		TransactionID:     packet.transactionID,
		PrevTransactionID: packet.prevTransactionID,
		IsNewBranch:       packet.nodeID == common.FirstEventID,
		Info:              "",
		History:           blob,
	})
	s.NoError(err)
}

func (s *HistoryEventsSuite) forkHistoryBranch(
	shardID int32,
	branchToken []byte,
	newNodeID int64,
) []byte {
	resp, err := s.store.ForkHistoryBranch(s.Ctx, &p.ForkHistoryBranchRequest{
		ShardID:         shardID,
		NamespaceID:     uuid.New(),
		ForkBranchToken: branchToken,
		ForkNodeID:      newNodeID,
		Info:            "",
		NewRunID:        uuid.New(),
	})
	s.NoError(err)
	return resp.NewBranchToken
}

func (s *HistoryEventsSuite) deleteHistoryBranch(
	shardID int32,
	branchToken []byte,
) {
	err := s.store.DeleteHistoryBranch(s.Ctx, &p.DeleteHistoryBranchRequest{
		ShardID:     shardID,
		BranchToken: branchToken,
	})
	s.NoError(err)
}

func (s *HistoryEventsSuite) trimHistoryBranch(
	shardID int32,
	branchToken []byte,
	nodeID int64,
	transactionID int64,
) {
	_, err := s.store.TrimHistoryBranch(s.Ctx, &p.TrimHistoryBranchRequest{
		ShardID:       shardID,
		BranchToken:   branchToken,
		NodeID:        nodeID,
		TransactionID: transactionID,
	})
	s.NoError(err)
}

func (s *HistoryEventsSuite) listHistoryEvents(
	shardID int32,
	branchToken []byte,
	startEventID int64,
	endEventID int64,
) []*historypb.HistoryEvent {
	var token []byte
	var events []*historypb.HistoryEvent
	for doContinue := true; doContinue; doContinue = len(token) > 0 {
		resp, err := s.store.ReadHistoryBranch(s.Ctx, &p.ReadHistoryBranchRequest{
			ShardID:       shardID,
			BranchToken:   branchToken,
			MinEventID:    startEventID,
			MaxEventID:    endEventID,
			PageSize:      1, // use 1 here for better testing exp
			NextPageToken: token,
		})
		s.NoError(err)
		token = resp.NextPageToken
		events = append(events, resp.HistoryEvents...)
	}
	return events
}

func (s *HistoryEventsSuite) listAllHistoryEvents(
	shardID int32,
	branchToken []byte,
) []*historypb.HistoryEvent {
	var token []byte
	var events []*historypb.HistoryEvent
	for doContinue := true; doContinue; doContinue = len(token) > 0 {
		resp, err := s.store.ReadHistoryBranch(s.Ctx, &p.ReadHistoryBranchRequest{
			ShardID:       shardID,
			BranchToken:   branchToken,
			MinEventID:    common.FirstEventID,
			MaxEventID:    common.LastEventID,
			PageSize:      1, // use 1 here for better testing exp
			NextPageToken: token,
		})
		s.NoError(err)
		token = resp.NextPageToken
		events = append(events, resp.HistoryEvents...)
	}
	return events
}

func (s *HistoryEventsSuite) newHistoryEvents(
	eventIDs []int64,
	transactionID int64,
	prevTransactionID int64,
) HistoryEventsPacket {

	events := make([]*historypb.HistoryEvent, len(eventIDs))
	for index, eventID := range eventIDs {
		events[index] = &historypb.HistoryEvent{
			EventId:   eventID,
			EventTime: timestamppb.New(time.Unix(0, rand.Int63()).UTC()),
		}
	}

	return HistoryEventsPacket{
		nodeID:            eventIDs[0],
		transactionID:     transactionID,
		prevTransactionID: prevTransactionID,
		events:            events,
	}
}
