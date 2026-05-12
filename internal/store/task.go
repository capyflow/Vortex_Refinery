package store

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"Vortex_Refinery/pkg/types"
)

const taskCollection = "task_records"

// TaskStore handles task record CRUD operations
type TaskStore struct {
	mongo       *MongoStore
	collection  *mongo.Collection
}

// NewTaskStore creates a new task store
func NewTaskStore(m *MongoStore) *TaskStore {
	return &TaskStore{
		mongo:       m,
		collection: m.Collection(taskCollection),
	}
}

// Create creates a new task record
func (s *TaskStore) Create(ctx context.Context, task *types.TaskRecord) error {
	task.CreatedAt = time.Now()
	_, err := s.collection.InsertOne(ctx, task)
	return err
}

// GetByID retrieves a task record by ID
func (s *TaskStore) GetByID(ctx context.Context, id string) (*types.TaskRecord, error) {
	var task types.TaskRecord
	err := s.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&task)
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// Update updates a task record
func (s *TaskStore) Update(ctx context.Context, task *types.TaskRecord) error {
	_, err := s.collection.UpdateOne(
		ctx,
		bson.M{"_id": task.ID},
		bson.M{"$set": task},
	)
	return err
}

// UpdateStatus updates the status of a task record
func (s *TaskStore) UpdateStatus(ctx context.Context, id, status string) error {
	update := bson.M{"$set": bson.M{"status": status}}

	switch status {
	case "running":
		now := time.Now()
		update["$set"].(bson.M)["started_at"] = &now
	case "completed", "failed":
		now := time.Now()
		update["$set"].(bson.M)["completed_at"] = &now
	}

	_, err := s.collection.UpdateOne(ctx, bson.M{"_id": id}, update)
	return err
}

// SetOutput sets the output of a task record
func (s *TaskStore) SetOutput(ctx context.Context, id string, output map[string]string) error {
	update := bson.M{
		"$set": bson.M{
			"output": output,
		},
	}
	_, err := s.collection.UpdateOne(ctx, bson.M{"_id": id}, update)
	return err
}

// SetError sets the error of a task record
func (s *TaskStore) SetError(ctx context.Context, id, errMsg string) error {
	update := bson.M{
		"$set": bson.M{
			"error": errMsg,
		},
	}
	_, err := s.collection.UpdateOne(ctx, bson.M{"_id": id}, update)
	return err
}

// IncrementRetryCount increments the retry count
func (s *TaskStore) IncrementRetryCount(ctx context.Context, id string) error {
	update := bson.M{
		"$inc": bson.M{"retry_count": 1},
	}
	_, err := s.collection.UpdateOne(ctx, bson.M{"_id": id}, update)
	return err
}

// AssignToWorker assigns a task to a worker
func (s *TaskStore) AssignToWorker(ctx context.Context, id, workerID string) error {
	update := bson.M{
		"$set": bson.M{
			"worker_id": workerID,
		},
	}
	_, err := s.collection.UpdateOne(ctx, bson.M{"_id": id}, update)
	return err
}

// ListByInstanceID lists task records by instance ID
func (s *TaskStore) ListByInstanceID(ctx context.Context, instanceID string) ([]*types.TaskRecord, error) {
	cursor, err := s.collection.Find(ctx, bson.M{"instance_id": instanceID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var tasks []*types.TaskRecord
	if err := cursor.All(ctx, &tasks); err != nil {
		return nil, err
	}
	return tasks, nil
}

// ListByWorkerID lists task records by worker ID
func (s *TaskStore) ListByWorkerID(ctx context.Context, workerID string) ([]*types.TaskRecord, error) {
	cursor, err := s.collection.Find(ctx, bson.M{"worker_id": workerID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var tasks []*types.TaskRecord
	if err := cursor.All(ctx, &tasks); err != nil {
		return nil, err
	}
	return tasks, nil
}

// ListByStatus lists task records by status
func (s *TaskStore) ListByStatus(ctx context.Context, status string) ([]*types.TaskRecord, error) {
	cursor, err := s.collection.Find(ctx, bson.M{"status": status})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var tasks []*types.TaskRecord
	if err := cursor.All(ctx, &tasks); err != nil {
		return nil, err
	}
	return tasks, nil
}

// EnsureIndexes ensures required indexes exist
func (s *TaskStore) EnsureIndexes(ctx context.Context) error {
	_, err := s.collection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "instance_id", Value: 1}}},
		{Keys: bson.D{{Key: "worker_id", Value: 1}}},
		{Keys: bson.D{{Key: "status", Value: 1}}},
	})
	return err
}

// TaskStoreInterface defines the task store interface
type TaskStoreInterface interface {
	Create(ctx context.Context, task *types.TaskRecord) error
	GetByID(ctx context.Context, id string) (*types.TaskRecord, error)
	Update(ctx context.Context, task *types.TaskRecord) error
	UpdateStatus(ctx context.Context, id, status string) error
	SetOutput(ctx context.Context, id string, output map[string]string) error
	SetError(ctx context.Context, id, errMsg string) error
	IncrementRetryCount(ctx context.Context, id string) error
	AssignToWorker(ctx context.Context, id, workerID string) error
	ListByInstanceID(ctx context.Context, instanceID string) ([]*types.TaskRecord, error)
	ListByWorkerID(ctx context.Context, workerID string) ([]*types.TaskRecord, error)
	ListByStatus(ctx context.Context, status string) ([]*types.TaskRecord, error)
	EnsureIndexes(ctx context.Context) error
}

var _ TaskStoreInterface = (*TaskStore)(nil)
