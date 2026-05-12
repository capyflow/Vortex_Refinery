package types

import "time"

// Event represents an internal event from external sources
type Event struct {
	EventID   string                 `json:"event_id" bson:"event_id"`
	EventType string                 `json:"event_type" bson:"event_type"`
	Timestamp time.Time              `json:"timestamp" bson:"timestamp"`
	Payload   map[string]interface{} `json:"payload" bson:"payload"`
}

// WorkflowDefinition defines a workflow template
type WorkflowDefinition struct {
	ID        string         `json:"id" bson:"_id"`
	Name      string         `json:"name" bson:"name"`
	Version   int            `json:"version" bson:"version"`
	Trigger   Trigger        `json:"trigger" bson:"trigger"`
	Nodes     []WorkflowNode `json:"nodes" bson:"nodes"`
	OnComplete CallbackAction `json:"on_complete" bson:"on_complete"`
	OnFailure CallbackAction  `json:"on_failure" bson:"on_failure"`
}

// Trigger defines event matching conditions
type Trigger struct {
	EventType  string                 `json:"event_type" bson:"event_type"`
	Conditions map[string]interface{} `json:"conditions" bson:"conditions"`
}

// WorkflowNode defines a single processing node
type WorkflowNode struct {
	ID        string                 `json:"id" bson:"id"`
	Name      string                 `json:"name" bson:"name"`
	Plugin    string                 `json:"plugin" bson:"plugin"`
	DependsOn []string               `json:"depends_on" bson:"depends_on"`
	Config    map[string]interface{} `json:"config" bson:"config"`
}

// CallbackAction defines callback on completion or failure
type CallbackAction struct {
	Action string                 `json:"action" bson:"action"`
	Config map[string]interface{} `json:"config" bson:"config"`
}

// WorkflowInstance represents a running workflow
type WorkflowInstance struct {
	ID             string    `json:"id" bson:"_id"`
	WorkflowID     string    `json:"workflow_id" bson:"workflow_id"`
	Status         string    `json:"status" bson:"status"`
	Event          *Event    `json:"event" bson:"event"`
	CurrentNodes   []string  `json:"current_nodes" bson:"current_nodes"`
	CompletedNodes []string  `json:"completed_nodes" bson:"completed_nodes"`
	FailedNodes    []string  `json:"failed_nodes" bson:"failed_nodes"`
	CreatedAt      time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" bson:"updated_at"`
}

// TaskRecord represents a single task execution
type TaskRecord struct {
	ID          string            `json:"id" bson:"_id"`
	InstanceID  string            `json:"instance_id" bson:"instance_id"`
	NodeID      string            `json:"node_id" bson:"node_id"`
	Plugin      string            `json:"plugin" bson:"plugin"`
	Status      string            `json:"status" bson:"status"`
	Input       map[string]string `json:"input" bson:"input"`
	Output      map[string]string `json:"output" bson:"output"`
	Error       string            `json:"error" bson:"error"`
	WorkerID    string            `json:"worker_id" bson:"worker_id"`
	CreatedAt   time.Time         `json:"created_at" bson:"created_at"`
	StartedAt   *time.Time        `json:"started_at" bson:"started_at"`
	CompletedAt *time.Time        `json:"completed_at" bson:"completed_at"`
	RetryCount  int               `json:"retry_count" bson:"retry_count"`
}

// WorkerRegistry records registered workers
type WorkerRegistry struct {
	WorkerID       string    `json:"worker_id" bson:"worker_id"`
	Plugins        []string  `json:"plugins" bson:"plugins"`
	Status         string    `json:"status" bson:"status"`
	RegisteredAt  time.Time `json:"registered_at" bson:"registered_at"`
	LastHeartbeat  time.Time `json:"last_heartbeat" bson:"last_heartbeat"`
}

// Task represents a task dispatched to a worker
type Task struct {
	TaskID     string                 `json:"task_id"`
	InstanceID string                 `json:"instance_id"`
	NodeID     string                 `json:"node_id"`
	Plugin     string                 `json:"plugin"`
	Config     map[string]interface{} `json:"config"`
	Context    map[string]string      `json:"context"`
}

// TaskResult represents the result of a task execution
type TaskResult struct {
	TaskID  string            `json:"task_id"`
	Success bool              `json:"success"`
	Output  map[string]string `json:"output"`
	Error   string            `json:"error"`
}
