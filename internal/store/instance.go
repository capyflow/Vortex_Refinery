package store

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"Vortex_Refinery/pkg/types"
)

const instanceCollection = "workflow_instances"

// InstanceStore handles workflow instance CRUD operations
type InstanceStore struct {
	mongo       *MongoStore
	collection  *mongo.Collection
}

// NewInstanceStore creates a new instance store
func NewInstanceStore(m *MongoStore) *InstanceStore {
	return &InstanceStore{
		mongo:       m,
		collection: m.Collection(instanceCollection),
	}
}

// Create creates a new workflow instance
func (s *InstanceStore) Create(ctx context.Context, inst *types.WorkflowInstance) error {
	inst.CreatedAt = time.Now()
	inst.UpdatedAt = time.Now()
	_, err := s.collection.InsertOne(ctx, inst)
	return err
}

// GetByID retrieves a workflow instance by ID
func (s *InstanceStore) GetByID(ctx context.Context, id string) (*types.WorkflowInstance, error) {
	var inst types.WorkflowInstance
	err := s.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&inst)
	if err != nil {
		return nil, err
	}
	return &inst, nil
}

// Update updates a workflow instance
func (s *InstanceStore) Update(ctx context.Context, inst *types.WorkflowInstance) error {
	inst.UpdatedAt = time.Now()
	_, err := s.collection.UpdateOne(
		ctx,
		bson.M{"_id": inst.ID},
		bson.M{"$set": inst},
	)
	return err
}

// UpdateStatus updates the status of a workflow instance
func (s *InstanceStore) UpdateStatus(ctx context.Context, id, status string) error {
	update := bson.M{
		"$set": bson.M{
			"status":     status,
			"updated_at": time.Now(),
		},
	}
	_, err := s.collection.UpdateOne(ctx, bson.M{"_id": id}, update)
	return err
}

// AddCompletedNode adds a completed node to the instance
func (s *InstanceStore) AddCompletedNode(ctx context.Context, id, nodeID string) error {
	update := bson.M{
		"$push": bson.M{"completed_nodes": nodeID},
		"$set":  bson.M{"updated_at": time.Now()},
	}
	_, err := s.collection.UpdateOne(ctx, bson.M{"_id": id}, update)
	return err
}

// AddFailedNode adds a failed node to the instance
func (s *InstanceStore) AddFailedNode(ctx context.Context, id, nodeID string) error {
	update := bson.M{
		"$push": bson.M{"failed_nodes": nodeID},
		"$set":  bson.M{"updated_at": time.Now()},
	}
	_, err := s.collection.UpdateOne(ctx, bson.M{"_id": id}, update)
	return err
}

// SetCurrentNodes sets the current nodes for an instance
func (s *InstanceStore) SetCurrentNodes(ctx context.Context, id string, nodes []string) error {
	update := bson.M{
		"$set": bson.M{
			"current_nodes": nodes,
			"updated_at":    time.Now(),
		},
	}
	_, err := s.collection.UpdateOne(ctx, bson.M{"_id": id}, update)
	return err
}

// ListByStatus lists workflow instances by status
func (s *InstanceStore) ListByStatus(ctx context.Context, status string) ([]*types.WorkflowInstance, error) {
	filter := bson.M{}
	if status != "" {
		filter["status"] = status
	}

	cursor, err := s.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var instances []*types.WorkflowInstance
	if err := cursor.All(ctx, &instances); err != nil {
		return nil, err
	}
	return instances, nil
}

// EnsureIndexes ensures required indexes exist
func (s *InstanceStore) EnsureIndexes(ctx context.Context) error {
	_, err := s.collection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "workflow_id", Value: 1}}},
		{Keys: bson.D{{Key: "status", Value: 1}}},
		{Keys: bson.D{{Key: "created_at", Value: -1}}},
	})
	return err
}

// InstanceStoreInterface defines the instance store interface
type InstanceStoreInterface interface {
	Create(ctx context.Context, inst *types.WorkflowInstance) error
	GetByID(ctx context.Context, id string) (*types.WorkflowInstance, error)
	Update(ctx context.Context, inst *types.WorkflowInstance) error
	UpdateStatus(ctx context.Context, id, status string) error
	AddCompletedNode(ctx context.Context, id, nodeID string) error
	AddFailedNode(ctx context.Context, id, nodeID string) error
	SetCurrentNodes(ctx context.Context, id string, nodes []string) error
	ListByStatus(ctx context.Context, status string) ([]*types.WorkflowInstance, error)
	EnsureIndexes(ctx context.Context) error
}

var _ InstanceStoreInterface = (*InstanceStore)(nil)
