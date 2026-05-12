package master

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"Vortex_Refinery/internal/bus"
	"Vortex_Refinery/internal/store"
	"Vortex_Refinery/pkg/types"
)

// Matcher finds matching workflow definitions for an event
type Matcher struct {
	store *store.MongoStore
}

// NewMatcher creates a new Matcher
func NewMatcher(s *store.MongoStore) *Matcher {
	return &Matcher{store: s}
}

// Match finds a workflow definition that matches the given event
func (m *Matcher) Match(event *types.Event) (*types.WorkflowDefinition, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := m.store.Collection("workflow_definitions")

	filter := bson.M{"trigger.event_type": event.EventType}
	opts := options.FindOne().SetSort(bson.M{"version": -1})

	var wf types.WorkflowDefinition
	err := collection.FindOne(ctx, filter, opts).Decode(&wf)
	if err != nil {
		return nil, err
	}

	// Check additional conditions
	if wf.Trigger.Conditions != nil {
		if !m.checkConditions(wf.Trigger.Conditions, event.Payload) {
			return nil, nil // Conditions not met
		}
	}

	return &wf, nil
}

// checkConditions evaluates trigger conditions against event payload
func (m *Matcher) checkConditions(conditions map[string]interface{}, payload map[string]interface{}) bool {
	for key, condition := range conditions {
		value, exists := payload[key]
		if !exists {
			return false
		}

		// Handle $in operator
		if condMap, ok := condition.(map[string]interface{}); ok {
			if inList, ok := condMap["$in"]; ok {
				if !m.checkIn(key, value, inList) {
					return false
				}
			}
			// Handle $eq operator
			if eqVal, ok := condMap["$eq"]; ok {
				if !m.checkEq(value, eqVal) {
					return false
				}
			}
			// Handle $contains operator
			if contains, ok := condMap["$contains"]; ok {
				if str, ok := value.(string); ok {
					if !strings.Contains(str, contains.(string)) {
						return false
					}
				} else {
					return false
				}
			}
		} else {
			// Direct equality check
			if value != condition {
				return false
			}
		}
	}
	return true
}

func (m *Matcher) checkIn(key string, value interface{}, list interface{}) bool {
	switch v := value.(type) {
	case string:
		if listStr, ok := list.([]interface{}); ok {
			for _, item := range listStr {
				if item.(string) == v {
					return true
				}
			}
		}
	case float64:
		if listNum, ok := list.([]interface{}); ok {
			for _, item := range listNum {
				if num, ok := item.(float64); ok {
					if num == v {
						return true
					}
				}
			}
		}
	}
	return false
}

func (m *Matcher) checkEq(value, expected interface{}) bool {
	// Handle type conversion for numeric types
	switch v := value.(type) {
	case float64:
		if exp, ok := expected.(float64); ok {
			return v == exp
		}
	case string:
		if exp, ok := expected.(string); ok {
			return v == exp
		}
	}
	return value == expected
}

// Scheduler handles workflow node scheduling with topological sort
type Scheduler struct{}

// NewScheduler creates a new Scheduler
func NewScheduler() *Scheduler {
	return &Scheduler{}
}

// ScheduleLayer represents a batch of nodes that can be executed in parallel
type ScheduleLayer []string

// Schedule computes the execution layers for workflow nodes based on dependencies
func (s *Scheduler) Schedule(nodes []types.WorkflowNode) ([]ScheduleLayer, error) {
	// Build dependency graph
	nodeMap := make(map[string]types.WorkflowNode)
	inDegree := make(map[string]int)
	dependents := make(map[string][]string) // node -> nodes that depend on it

	for _, node := range nodes {
		nodeMap[node.ID] = node
		inDegree[node.ID] = len(node.DependsOn)
		for _, dep := range node.DependsOn {
			dependents[dep] = append(dependents[dep], node.ID)
		}
	}

	var layers []ScheduleLayer
	visited := make(map[string]bool)

	for len(visited) < len(nodes) {
		// Find all nodes with zero in-degree (no unmet dependencies)
		var layer ScheduleLayer
		for _, node := range nodes {
			if !visited[node.ID] && inDegree[node.ID] == 0 {
				layer = append(layer, node.ID)
			}
		}

		if len(layer) == 0 {
			// Circular dependency or unmet dependency
			return nil, ErrCircularDependency
		}

		layers = append(layers, layer)

		// Mark nodes in this layer as visited and update in-degrees
		for _, nodeID := range layer {
			visited[nodeID] = true
			for _, dependent := range dependents[nodeID] {
				inDegree[dependent]--
			}
		}
	}

	return layers, nil
}

// ErrCircularDependency is returned when a circular dependency is detected
var ErrCircularDependency = &SchedulingError{"circular dependency detected"}

type SchedulingError struct {
	msg string
}

func (e *SchedulingError) Error() string {
	return e.msg
}

// CreateInstance creates a new workflow instance
func (m *Master) CreateInstance(wf *types.WorkflowDefinition, event *types.Event) (*types.WorkflowInstance, error) {
	instance := &types.WorkflowInstance{
		ID:             "wfi-" + uuid.New().String(),
		WorkflowID:     wf.ID,
		Status:         "pending",
		Event:          event,
		CurrentNodes:   []string{},
		CompletedNodes: []string{},
		FailedNodes:    []string{},
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := m.store.Collection("workflow_instances").InsertOne(ctx, instance)
	if err != nil {
		return nil, err
	}

	return instance, nil
}

// Dispatcher handles task dispatching to workers
type Dispatcher struct {
	taskBus *bus.TaskBus
	store   *store.MongoStore
}

// NewDispatcher creates a new Dispatcher
func NewDispatcher(tb *bus.TaskBus, s *store.MongoStore) *Dispatcher {
	return &Dispatcher{
		taskBus: tb,
		store:   s,
	}
}

// DispatchTask dispatches a task to a specific worker
func (d *Dispatcher) DispatchTask(ctx context.Context, instance *types.WorkflowInstance, node *types.WorkflowNode, contextData map[string]string) (*types.TaskRecord, error) {
	taskRecord := &types.TaskRecord{
		ID:         "task-" + uuid.New().String(),
		InstanceID: instance.ID,
		NodeID:     node.ID,
		Plugin:     node.Plugin,
		Status:     "pending",
		Input:      contextData,
		CreatedAt:  time.Now(),
		RetryCount: 0,
	}

	// Save task record to MongoDB
	_, err := d.store.Collection("task_records").InsertOne(ctx, taskRecord)
	if err != nil {
		return nil, err
	}

	// Create task for Redis
	task := &types.Task{
		TaskID:     taskRecord.ID,
		InstanceID: instance.ID,
		NodeID:     node.ID,
		Plugin:     node.Plugin,
		Config:     node.Config,
		Context:    contextData,
	}

	// Find a worker that has this plugin
	worker, err := d.findAvailableWorker(node.Plugin)
	if err != nil {
		return nil, err
	}

	// Dispatch to worker's queue
	err = d.taskBus.DispatchTask(ctx, worker.WorkerID, task)
	if err != nil {
		return nil, err
	}

	// Update task record with worker
	taskRecord.WorkerID = worker.WorkerID

	return taskRecord, nil
}

func (d *Dispatcher) findAvailableWorker(plugin string) (*types.WorkerRegistry, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{
		"plugins": plugin,
		"status":  "online",
	}

	var worker types.WorkerRegistry
	err := d.store.Collection("worker_registry").FindOne(ctx, filter).Decode(&worker)
	if err != nil {
		return nil, ErrNoAvailableWorker
	}

	return &worker, nil
}

// ErrNoAvailableWorker is returned when no worker is available for a plugin
var ErrNoAvailableWorker = &DispatcherError{"no available worker found for plugin"}

type DispatcherError struct {
	msg string
}

func (e *DispatcherError) Error() string {
	return e.msg
}

// Reporter handles task result reports from workers
type Reporter struct {
	store *store.MongoStore
}

// NewReporter creates a new Reporter
func NewReporter(s *store.MongoStore) *Reporter {
	return &Reporter{store: s}
}

// HandleResult processes a task result
func (r *Reporter) HandleResult(ctx context.Context, result *types.TaskResult) error {
	// Find and update task record
	collection := r.store.Collection("task_records")
	taskFilter := bson.M{"_id": result.TaskID}

	var task types.TaskRecord
	err := collection.FindOne(ctx, taskFilter).Decode(&task)
	if err != nil {
		return err
	}

	// Update task record
	update := bson.M{
		"$set": bson.M{
			"status": func() string {
				if result.Success {
					return "completed"
				}
				return "failed"
			}(),
			"output":       result.Output,
			"error":        result.Error,
			"completed_at": time.Now(),
		},
	}

	_, err = collection.UpdateOne(ctx, taskFilter, update)
	if err != nil {
		return err
	}

	// Update workflow instance
	instanceCollection := r.store.Collection("workflow_instances")
	instanceFilter := bson.M{"_id": task.InstanceID}

	if result.Success {
		// Move node from current to completed
		_, err = instanceCollection.UpdateOne(ctx, instanceFilter, bson.M{
			"$pull": bson.M{"current_nodes": task.NodeID},
			"$push": bson.M{"completed_nodes": task.NodeID},
			"$set":  bson.M{"updated_at": time.Now()},
		})
	} else {
		// Move node from current to failed
		_, err = instanceCollection.UpdateOne(ctx, instanceFilter, bson.M{
			"$pull": bson.M{"current_nodes": task.NodeID},
			"$push": bson.M{"failed_nodes": task.NodeID},
			"$set":  bson.M{"updated_at": time.Now()},
		})
	}

	return err
}

// GetPendingTasksForInstance returns pending nodes for a workflow instance
func (r *Reporter) GetPendingTasksForInstance(ctx context.Context, instanceID string) ([]string, error) {
	coll := r.store.Collection("workflow_instances")
	var instance types.WorkflowInstance
	err := coll.FindOne(ctx, bson.M{"_id": instanceID}).Decode(&instance)
	if err != nil {
		return nil, err
	}

	// Get all nodes for this workflow
	wf, err := r.store.Collection("workflow_definitions").Find(ctx, bson.M{"_id": instance.WorkflowID})
	if err != nil {
		return nil, err
	}

	var wfDef types.WorkflowDefinition
	if wf.Next(ctx) {
		bson.Unmarshal(wf.Current, &wfDef)
	}

	pending := []string{}
	completed := make(map[string]bool)
	for _, c := range instance.CompletedNodes {
		completed[c] = true
	}
	for _, c := range instance.FailedNodes {
		completed[c] = true
	}

	for _, node := range wfDef.Nodes {
		if !completed[node.ID] {
			// Check if all dependencies are met
			allMet := true
			for _, dep := range node.DependsOn {
				if !completed[dep] {
					allMet = false
					break
				}
			}
			if allMet {
				pending = append(pending, node.ID)
			}
		}
	}

	return pending, nil
}

// MarshalJSON safely marshals a WorkflowDefinition to JSON
func MarshalWorkflow(wf *types.WorkflowDefinition) ([]byte, error) {
	return json.Marshal(wf)
}
