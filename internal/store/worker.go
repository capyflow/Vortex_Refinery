package store

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"Vortex_Refinery/pkg/types"
)

const workerRegistryCollection = "worker_registry"

// WorkerStore handles worker registry CRUD operations
type WorkerStore struct {
	mongo       *MongoStore
	collection  *mongo.Collection
}

// NewWorkerStore creates a new worker store
func NewWorkerStore(m *MongoStore) *WorkerStore {
	return &WorkerStore{
		mongo:       m,
		collection: m.Collection(workerRegistryCollection),
	}
}

// Register registers a new worker or updates existing one
func (s *WorkerStore) Register(ctx context.Context, worker *types.WorkerRegistry) error {
	worker.RegisteredAt = time.Now()
	worker.LastHeartbeat = time.Now()

	opts := options.Update().SetUpsert(true)
	filter := bson.M{"_id": worker.WorkerID}
	update := bson.M{
		"$set": bson.M{
			"plugins":        worker.Plugins,
			"status":        worker.Status,
			"registered_at": worker.RegisteredAt,
			"last_heartbeat": worker.LastHeartbeat,
		},
		"$setOnInsert": bson.M{
			"_id": worker.WorkerID,
		},
	}

	_, err := s.collection.UpdateOne(ctx, filter, update, opts)
	return err
}

// GetByID retrieves a worker by ID
func (s *WorkerStore) GetByID(ctx context.Context, id string) (*types.WorkerRegistry, error) {
	var worker types.WorkerRegistry
	err := s.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&worker)
	if err != nil {
		return nil, err
	}
	return &worker, nil
}

// UpdateHeartbeat updates the heartbeat of a worker
func (s *WorkerStore) UpdateHeartbeat(ctx context.Context, id string) error {
	update := bson.M{
		"$set": bson.M{
			"last_heartbeat": time.Now(),
		},
	}
	_, err := s.collection.UpdateOne(ctx, bson.M{"_id": id}, update)
	return err
}

// UpdateStatus updates the status of a worker
func (s *WorkerStore) UpdateStatus(ctx context.Context, id, status string) error {
	update := bson.M{
		"$set": bson.M{
			"status": status,
		},
	}
	_, err := s.collection.UpdateOne(ctx, bson.M{"_id": id}, update)
	return err
}

// UpdatePlugins updates the plugins of a worker
func (s *WorkerStore) UpdatePlugins(ctx context.Context, id string, plugins []string) error {
	update := bson.M{
		"$set": bson.M{
			"plugins": plugins,
		},
	}
	_, err := s.collection.UpdateOne(ctx, bson.M{"_id": id}, update)
	return err
}

// ListOnline lists all online workers
func (s *WorkerStore) ListOnline(ctx context.Context) ([]*types.WorkerRegistry, error) {
	return s.ListByStatus(ctx, "online")
}

// ListByStatus lists workers by status
func (s *WorkerStore) ListByStatus(ctx context.Context, status string) ([]*types.WorkerRegistry, error) {
	filter := bson.M{}
	if status != "" {
		filter["status"] = status
	}

	cursor, err := s.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var workers []*types.WorkerRegistry
	if err := cursor.All(ctx, &workers); err != nil {
		return nil, err
	}
	return workers, nil
}

// FindWorkersByPlugin finds workers that have a specific plugin
func (s *WorkerStore) FindWorkersByPlugin(ctx context.Context, pluginName string) ([]*types.WorkerRegistry, error) {
	filter := bson.M{
		"plugins": pluginName,
		"status":  "online",
	}

	cursor, err := s.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var workers []*types.WorkerRegistry
	if err := cursor.All(ctx, &workers); err != nil {
		return nil, err
	}
	return workers, nil
}

// MarkStaleWorkersOffline marks workers with stale heartbeats as offline
func (s *WorkerStore) MarkStaleWorkersOffline(ctx context.Context, threshold time.Duration) error {
	thresholdTime := time.Now().Add(-threshold)
	filter := bson.M{
		"last_heartbeat": bson.M{"$lt": thresholdTime},
		"status":         "online",
	}
	update := bson.M{
		"$set": bson.M{
			"status": "offline",
		},
	}
	_, err := s.collection.UpdateMany(ctx, filter, update)
	return err
}

// EnsureIndexes ensures required indexes exist
func (s *WorkerStore) EnsureIndexes(ctx context.Context) error {
	_, err := s.collection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "status", Value: 1}}},
		{Keys: bson.D{{Key: "plugins", Value: 1}}},
		{Keys: bson.D{{Key: "last_heartbeat", Value: -1}}},
	})
	return err
}

// WorkerStoreInterface defines the worker store interface
type WorkerStoreInterface interface {
	Register(ctx context.Context, worker *types.WorkerRegistry) error
	GetByID(ctx context.Context, id string) (*types.WorkerRegistry, error)
	UpdateHeartbeat(ctx context.Context, id string) error
	UpdateStatus(ctx context.Context, id, status string) error
	UpdatePlugins(ctx context.Context, id string, plugins []string) error
	ListOnline(ctx context.Context) ([]*types.WorkerRegistry, error)
	ListByStatus(ctx context.Context, status string) ([]*types.WorkerRegistry, error)
	FindWorkersByPlugin(ctx context.Context, pluginName string) ([]*types.WorkerRegistry, error)
	MarkStaleWorkersOffline(ctx context.Context, threshold time.Duration) error
	EnsureIndexes(ctx context.Context) error
}

var _ WorkerStoreInterface = (*WorkerStore)(nil)
