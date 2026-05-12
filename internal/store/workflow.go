package store

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"Vortex_Refinery/pkg/types"
)

const workflowCollection = "workflow_definitions"

// WorkflowStore handles workflow definition CRUD operations
type WorkflowStore struct {
	mongo   *MongoStore
	collection *mongo.Collection
}

// NewWorkflowStore creates a new workflow store
func NewWorkflowStore(m *MongoStore) *WorkflowStore {
	return &WorkflowStore{
		mongo:       m,
		collection: m.Collection(workflowCollection),
	}
}

// Create creates a new workflow definition
func (s *WorkflowStore) Create(ctx context.Context, wf *types.WorkflowDefinition) error {
	wf.Version = 1
	_, err := s.collection.InsertOne(ctx, wf)
	return err
}

// GetByID retrieves a workflow by ID
func (s *WorkflowStore) GetByID(ctx context.Context, id string) (*types.WorkflowDefinition, error) {
	var wf types.WorkflowDefinition
	err := s.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&wf)
	if err != nil {
		return nil, err
	}
	return &wf, nil
}

// List lists all workflow definitions
func (s *WorkflowStore) List(ctx context.Context) ([]*types.WorkflowDefinition, error) {
	cursor, err := s.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var workflows []*types.WorkflowDefinition
	if err := cursor.All(ctx, &workflows); err != nil {
		return nil, err
	}
	return workflows, nil
}

// Update updates a workflow definition
func (s *WorkflowStore) Update(ctx context.Context, wf *types.WorkflowDefinition) error {
	wf.Version++
	_, err := s.collection.UpdateOne(
		ctx,
		bson.M{"_id": wf.ID},
		bson.M{"$set": wf},
	)
	return err
}

// Delete deletes a workflow by ID
func (s *WorkflowStore) Delete(ctx context.Context, id string) error {
	_, err := s.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

// FindByEventType finds workflows that match a given event type
func (s *WorkflowStore) FindByEventType(ctx context.Context, eventType string) ([]*types.WorkflowDefinition, error) {
	cursor, err := s.collection.Find(ctx, bson.M{"trigger.event_type": eventType})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var workflows []*types.WorkflowDefinition
	if err := cursor.All(ctx, &workflows); err != nil {
		return nil, err
	}
	return workflows, nil
}

// EnsureIndexes ensures required indexes exist
func (s *WorkflowStore) EnsureIndexes(ctx context.Context) error {
	_, err := s.collection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "trigger.event_type", Value: 1}},
		},
	})
	return err
}

// WorkflowStoreInterface defines the workflow store interface
type WorkflowStoreInterface interface {
	Create(ctx context.Context, wf *types.WorkflowDefinition) error
	GetByID(ctx context.Context, id string) (*types.WorkflowDefinition, error)
	List(ctx context.Context) ([]*types.WorkflowDefinition, error)
	Update(ctx context.Context, wf *types.WorkflowDefinition) error
	Delete(ctx context.Context, id string) error
	FindByEventType(ctx context.Context, eventType string) ([]*types.WorkflowDefinition, error)
	EnsureIndexes(ctx context.Context) error
}

var _ WorkflowStoreInterface = (*WorkflowStore)(nil)

// Upsert creates or updates a workflow definition
func (s *WorkflowStore) Upsert(ctx context.Context, wf *types.WorkflowDefinition) error {
	opts := options.Update().SetUpsert(true)
	_, err := s.collection.UpdateOne(
		ctx,
		bson.M{"_id": wf.ID},
		bson.M{"$set": wf},
		opts,
	)
	return err
}
