package model

import "Vortex_Refinery/pkg/types"

// ─── API Request/Response types ───────────────────────────────────────────────

type CreateWorkflowReq struct {
	Name       string                `json:"name"`
	Trigger    types.Trigger        `json:"trigger"`
	Nodes      []types.WorkflowNode `json:"nodes"`
	OnComplete *types.CallbackAction `json:"onComplete,omitempty"`
	OnFailure  *types.CallbackAction `json:"onFailure,omitempty"`
}

type UpdateWorkflowReq struct {
	Name       *string               `json:"name,omitempty"`
	Nodes      []types.WorkflowNode  `json:"nodes,omitempty"`
	Trigger    *types.Trigger        `json:"trigger,omitempty"`
	OnComplete *types.CallbackAction `json:"onComplete,omitempty"`
	OnFailure  *types.CallbackAction `json:"onFailure,omitempty"`
}

type WorkflowResp struct {
	ID         string                `json:"id"`
	Name       string                `json:"name"`
	Version    int                   `json:"version"`
	Trigger    types.Trigger        `json:"trigger"`
	Nodes      []types.WorkflowNode  `json:"nodes"`
	OnComplete *types.CallbackAction `json:"onComplete,omitempty"`
	OnFailure  *types.CallbackAction `json:"onFailure,omitempty"`
	Published  bool                  `json:"published"`
	CreatedAt  string                `json:"createdAt"`
	UpdatedAt  string                `json:"updatedAt"`
}

type PluginResp struct {
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	Status       string   `json:"status"`
	RegisteredAt string   `json:"registeredAt"`
	NodeTypes    []string `json:"nodeTypes"`
}

type InstanceResp struct {
	ID             string   `json:"id"`
	WorkflowID     string   `json:"workflowId"`
	WorkflowName   string   `json:"workflowName"`
	Status         string   `json:"status"`
	Event          any      `json:"event"`
	CurrentNodes   []string `json:"currentNodes"`
	CompletedNodes []string `json:"completedNodes"`
	FailedNodes    []string `json:"failedNodes"`
	CreatedAt      string   `json:"createdAt"`
	UpdatedAt      string   `json:"updatedAt"`
}

type StatsResp struct {
	TotalPlugins     int `json:"totalPlugins"`
	TotalWorkflows  int `json:"totalWorkflows"`
	RunningInstances int `json:"runningInstances"`
	TriggeredToday   int `json:"triggeredToday"`
}

type IDResp struct {
	ID string `json:"id"`
}

type StatusResp struct {
	Status string `json:"status"`
}
