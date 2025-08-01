package sql

import (
	"fmt"
	"sync"

	"go.temporal.io/server/common/config"
	"go.temporal.io/server/common/log"
	"go.temporal.io/server/common/metrics"
	p "go.temporal.io/server/common/persistence"
	"go.temporal.io/server/common/persistence/sql/sqlplugin"
	"go.temporal.io/server/common/resolver"
)

type (
	// Factory vends store objects backed by MySQL
	Factory struct {
		cfg         config.SQL
		mainDBConn  DbConn
		clusterName string
		logger      log.Logger
	}

	// DbConn represents a logical mysql connection - its a
	// wrapper around the standard sql connection pool with
	// additional reference counting
	DbConn struct {
		dbKind   sqlplugin.DbKind
		cfg      *config.SQL
		resolver resolver.ServiceResolver
		logger   log.Logger
		metrics  metrics.Handler

		sqlplugin.DB

		sync.Mutex
		refCnt int
	}
)

// NewFactory returns an instance of a factory object which can be used to create
// datastores backed by any kind of SQL store
func NewFactory(
	cfg config.SQL,
	r resolver.ServiceResolver,
	clusterName string,
	logger log.Logger,
	metricsHandler metrics.Handler,
) *Factory {
	return &Factory{
		cfg:         cfg,
		clusterName: clusterName,
		logger:      logger,
		mainDBConn:  NewRefCountedDBConn(sqlplugin.DbKindMain, &cfg, r, logger, metricsHandler),
	}
}

// GetDB return a new SQL DB connection
func (f *Factory) GetDB() (sqlplugin.DB, error) {
	conn, err := f.mainDBConn.Get()
	if err != nil {
		return nil, err
	}
	return conn, err
}

// NewTaskStore returns a new task store
func (f *Factory) NewTaskStore() (p.TaskStore, error) {
	conn, err := f.mainDBConn.Get()
	if err != nil {
		return nil, err
	}
	return newTaskPersistence(conn, f.cfg.TaskScanPartitions, f.logger, false)
}

// NewFairTaskStore returns a new task store
func (f *Factory) NewFairTaskStore() (p.TaskStore, error) {
	conn, err := f.mainDBConn.Get()
	if err != nil {
		return nil, err
	}
	return newTaskPersistence(conn, f.cfg.TaskScanPartitions, f.logger, true)
}

// NewShardStore returns a new shard store
func (f *Factory) NewShardStore() (p.ShardStore, error) {
	conn, err := f.mainDBConn.Get()
	if err != nil {
		return nil, err
	}
	return newShardPersistence(conn, f.clusterName, f.logger)
}

// NewMetadataStore returns a new metadata store
func (f *Factory) NewMetadataStore() (p.MetadataStore, error) {
	conn, err := f.mainDBConn.Get()
	if err != nil {
		return nil, err
	}
	return newMetadataPersistenceV2(conn, f.clusterName, f.logger)
}

// NewClusterMetadataStore returns a new ClusterMetadata store
func (f *Factory) NewClusterMetadataStore() (p.ClusterMetadataStore, error) {
	conn, err := f.mainDBConn.Get()
	if err != nil {
		return nil, err
	}
	return newClusterMetadataPersistence(conn, f.logger)
}

// NewExecutionStore returns a new ExecutionStore
func (f *Factory) NewExecutionStore() (p.ExecutionStore, error) {
	conn, err := f.mainDBConn.Get()
	if err != nil {
		return nil, err
	}
	return NewSQLExecutionStore(conn, f.logger)
}

// NewQueue returns a new queue backed by sql
func (f *Factory) NewQueue(queueType p.QueueType) (p.Queue, error) {
	conn, err := f.mainDBConn.Get()
	if err != nil {
		return nil, err
	}

	return newQueue(conn, f.logger, queueType)
}

// NewQueueV2 returns a new data-access object for queues and messages.
func (f *Factory) NewQueueV2() (p.QueueV2, error) {
	conn, err := f.mainDBConn.Get()
	if err != nil {
		return nil, err
	}
	return NewQueueV2(conn, f.logger), nil
}

// NewNexusEndpointStore returns a new NexusEndpointStore
func (f *Factory) NewNexusEndpointStore() (p.NexusEndpointStore, error) {
	conn, err := f.mainDBConn.Get()
	if err != nil {
		return nil, err
	}
	return NewSqlNexusEndpointStore(conn, f.logger)
}

// Close closes the factory
func (f *Factory) Close() {
	f.mainDBConn.ForceClose()
}

// NewRefCountedDBConn returns a  logical mysql connection that
// uses reference counting to decide when to close the
// underlying connection object. The reference count gets incremented
// everytime get() is called and decremented everytime Close() is called
func NewRefCountedDBConn(
	dbKind sqlplugin.DbKind,
	cfg *config.SQL,
	r resolver.ServiceResolver,
	logger log.Logger,
	metricsHandler metrics.Handler,
) DbConn {
	return DbConn{
		dbKind:   dbKind,
		cfg:      cfg,
		resolver: r,
		metrics:  metricsHandler,
		logger:   logger,
	}
}

// Get returns a mysql db connection and increments a reference count
// this method will create a new connection, if an existing connection
// does not exist
func (c *DbConn) Get() (sqlplugin.DB, error) {
	c.Lock()
	defer c.Unlock()
	if c.refCnt == 0 {
		conn, err := NewSQLDB(c.dbKind, c.cfg, c.resolver, c.logger, c.metrics)
		if err != nil {
			return nil, err
		}
		c.DB = conn
	}
	c.refCnt++
	return c, nil
}

// ForceClose ignores reference counts and shutsdown the underlying connection pool
func (c *DbConn) ForceClose() {
	c.Lock()
	defer c.Unlock()
	if c.DB != nil {
		err := c.DB.Close()
		if err != nil {
			fmt.Println("failed to close database connection, may leak some connection", err)
		}
	}
	c.refCnt = 0
}

// Close closes the underlying connection if the reference count becomes zero
func (c *DbConn) Close() error {
	c.Lock()
	defer c.Unlock()
	c.refCnt--
	if c.refCnt == 0 {
		return c.DB.Close()
	}
	return nil
}
