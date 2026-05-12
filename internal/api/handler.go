package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"Vortex_Refinery/internal/store"
	"Vortex_Refinery/pkg/types"
)

// Handler handles REST API requests
type Handler struct {
	mongoStore *store.MongoStore
	workflowFS string // workflow JSON file storage directory
}

// NewHandler creates a new API handler
func NewHandler(s *store.MongoStore, workflowFS string) *Handler {
	return &Handler{
		mongoStore: s,
		workflowFS: workflowFS,
	}
}

// RegisterRoutes registers all API routes
func (h *Handler) RegisterRoutes(r *mux.Router) {
	// CORS middleware
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	})

	api := r.PathPrefix("/api").Subrouter()

	// Workflows
	api.HandleFunc("/workflows", h.listWorkflows).Methods(http.MethodGet)
	api.HandleFunc("/workflows", h.createWorkflow).Methods(http.MethodPost)
	api.HandleFunc("/workflows/{id}", h.getWorkflow).Methods(http.MethodGet)
	api.HandleFunc("/workflows/{id}", h.updateWorkflow).Methods(http.MethodPut)
	api.HandleFunc("/workflows/{id}", h.deleteWorkflow).Methods(http.MethodDelete)
	api.HandleFunc("/workflows/{id}/publish", h.publishWorkflow).Methods(http.MethodPost)

	// Plugins (reads from worker registry in MongoDB)
	api.HandleFunc("/plugins", h.listPlugins).Methods(http.MethodGet)
	api.HandleFunc("/plugins/{name}", h.getPlugin).Methods(http.MethodGet)

	// Instances
	api.HandleFunc("/instances", h.listInstances).Methods(http.MethodGet)
	api.HandleFunc("/instances/{id}", h.getInstance).Methods(http.MethodGet)
	api.HandleFunc("/instances/{id}/retry", h.retryInstance).Methods(http.MethodPost)
	api.HandleFunc("/instances/{id}/tasks", h.getInstanceTasks).Methods(http.MethodGet)

	// Workers
	api.HandleFunc("/workers", h.listWorkers).Methods(http.MethodGet)

	// Dashboard stats
	api.HandleFunc("/stats", h.getStats).Methods(http.MethodGet)

	// Health
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}).Methods(http.MethodGet)
}

// ─── Workflow Handlers ────────────────────────────────────────────────────────

func (h *Handler) listWorkflows(w http.ResponseWriter, r *http.Request) {
	// Try MongoDB first
	ctx := r.Context()
	wfStore := store.NewWorkflowStore(h.mongoStore)
	list, err := wfStore.List(ctx)
	if err == nil && len(list) > 0 {
		// Enrich with published status from JSON
		result := make([]*workflowResponse, 0, len(list))
		for _, wf := range list {
			result = append(result, toWorkflowResponse(wf))
		}
		// Sort by updatedAt desc
	sort.Slice(result, func(i, j int) bool {
		a, _ := time.Parse(time.RFC3339, result[i].UpdatedAt)
		b, _ := time.Parse(time.RFC3339, result[j].UpdatedAt)
		return a.After(b)
	})
		writeJSON(w, http.StatusOK, result)
		return
	}

	// Fall back to JSON files
	files, err := os.ReadDir(h.workflowFS)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to read workflows")
		return
	}

	workflows := make([]*workflowResponse, 0, len(files))
	for _, f := range files {
		if f.IsDir() || filepath.Ext(f.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(h.workflowFS, f.Name()))
		if err != nil {
			continue
		}
		var wf types.WorkflowDefinition
		if err := json.Unmarshal(data, &wf); err != nil {
			continue
		}
		workflows = append(workflows, toWorkflowResponse(&wf))
	}
	sort.Slice(workflows, func(i, j int) bool {
		a, _ := time.Parse(time.RFC3339, workflows[i].UpdatedAt)
		b, _ := time.Parse(time.RFC3339, workflows[j].UpdatedAt)
		return a.After(b)
	})
	writeJSON(w, http.StatusOK, workflows)
}

func (h *Handler) createWorkflow(w http.ResponseWriter, r *http.Request) {
	var req createWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	wf := &types.WorkflowDefinition{
		ID:        "wf-" + uuid.New().String(),
		Name:      req.Name,
		Version:   1,
		Trigger:   req.Trigger,
		Nodes:     req.Nodes,
		OnComplete: req.OnComplete,
		OnFailure:  req.OnFailure,
		Published:  false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Save to JSON file
	if err := h.saveWorkflowJSON(wf); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save workflow: "+err.Error())
		return
	}

	// Also save to MongoDB
	ctx := r.Context()
	wfStore := store.NewWorkflowStore(h.mongoStore)
	if err := wfStore.Create(ctx, wf); err != nil {
		// Log but don't fail — file storage is primary
	}

	writeJSON(w, http.StatusCreated, map[string]string{"id": wf.ID})
}

func (h *Handler) getWorkflow(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	// Try JSON first
	wf, err := h.loadWorkflowJSON(id)
	if err == nil && wf != nil {
		writeJSON(w, http.StatusOK, toWorkflowResponse(wf))
		return
	}

	// Fall back to MongoDB
	ctx := r.Context()
	wfStore := store.NewWorkflowStore(h.mongoStore)
	wf, err = wfStore.GetByID(ctx, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "workflow not found")
		return
	}
	writeJSON(w, http.StatusOK, toWorkflowResponse(wf))
}

func (h *Handler) updateWorkflow(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var req updateWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	// Load existing
	wf, err := h.loadWorkflowJSON(id)
	if err != nil || wf == nil {
		// Try MongoDB
		ctx := r.Context()
		wfStore := store.NewWorkflowStore(h.mongoStore)
		wf, err = wfStore.GetByID(ctx, id)
		if err != nil {
			writeError(w, http.StatusNotFound, "workflow not found")
			return
		}
	}

	// Apply updates
	if req.Name != nil && *req.Name != "" {
		wf.Name = *req.Name
	}
	if req.Nodes != nil {
		wf.Nodes = req.Nodes
	}
	if req.Trigger != nil {
		wf.Trigger = *req.Trigger
	}
	if req.OnComplete != nil {
		wf.OnComplete = req.OnComplete
	}
	if req.OnFailure != nil {
		wf.OnFailure = req.OnFailure
	}
	wf.UpdatedAt = time.Now()

	// Save to JSON
	if err := h.saveWorkflowJSON(wf); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save workflow: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, toWorkflowResponse(wf))
}

func (h *Handler) deleteWorkflow(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	// Delete JSON file
	filePath := filepath.Join(h.workflowFS, id+".json")
	os.Remove(filePath)

	// Delete from MongoDB
	ctx := r.Context()
	wfStore := store.NewWorkflowStore(h.mongoStore)
	wfStore.Delete(ctx, id)

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) publishWorkflow(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	wf, err := h.loadWorkflowJSON(id)
	if err != nil || wf == nil {
		ctx := r.Context()
		wfStore := store.NewWorkflowStore(h.mongoStore)
		wf, err = wfStore.GetByID(ctx, id)
		if err != nil {
			writeError(w, http.StatusNotFound, "workflow not found")
			return
		}
	}

	wf.Published = true
	wf.UpdatedAt = time.Now()

	if err := h.saveWorkflowJSON(wf); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to publish: "+err.Error())
		return
	}

	ctx := r.Context()
	wfStore := store.NewWorkflowStore(h.mongoStore)
	wfStore.Update(ctx, wf)

	writeJSON(w, http.StatusOK, toWorkflowResponse(wf))
}

// ─── Plugin Handlers ─────────────────────────────────────────────────────────

func (h *Handler) listPlugins(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	coll := h.mongoStore.Collection("worker_registry")

	cursor, err := coll.Find(ctx, map[string]interface{}{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list plugins")
		return
	}
	defer cursor.Close(ctx)

	var workers []types.WorkerRegistry
	if err := cursor.All(ctx, &workers); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to decode workers")
		return
	}

	// Deduplicate plugins from all workers
	pluginMap := make(map[string]*pluginResponse)
	for _, worker := range workers {
		status := "unknown"
		if worker.Status == "online" {
			status = "healthy"
		}
		for _, p := range worker.Plugins {
			if _, exists := pluginMap[p]; !exists {
				pluginMap[p] = &pluginResponse{
					Name:         p,
					Version:      "1.0.0",
					Status:       status,
					RegisteredAt: worker.RegisteredAt.Format(time.RFC3339),
					NodeTypes:    []string{p},
				}
			}
		}
	}

	plugins := make([]*pluginResponse, 0, len(pluginMap))
	for _, p := range pluginMap {
		plugins = append(plugins, p)
	}
	writeJSON(w, http.StatusOK, plugins)
}

func (h *Handler) getPlugin(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	ctx := r.Context()
	coll := h.mongoStore.Collection("worker_registry")

	cursor, err := coll.Find(ctx, map[string]interface{}{})
	if err != nil {
		writeError(w, http.StatusNotFound, "plugin not found")
		return
	}
	defer cursor.Close(ctx)

	var workers []types.WorkerRegistry
	if err := cursor.All(ctx, &workers); err != nil {
		writeError(w, http.StatusNotFound, "plugin not found")
		return
	}

	for _, worker := range workers {
		for _, p := range worker.Plugins {
			if p == name {
				status := "unknown"
				if worker.Status == "online" {
					status = "healthy"
				}
				writeJSON(w, http.StatusOK, &pluginResponse{
					Name:         p,
					Version:      "1.0.0",
					Status:       status,
					RegisteredAt: worker.RegisteredAt.Format(time.RFC3339),
					NodeTypes:    []string{p},
				})
				return
			}
		}
	}
	writeError(w, http.StatusNotFound, "plugin not found")
}

// ─── Instance Handlers ────────────────────────────────────────────────────────

func (h *Handler) listInstances(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	status := r.URL.Query().Get("status")
	instStore := store.NewInstanceStore(h.mongoStore)

	list, err := instStore.ListByStatus(ctx, status)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list instances")
		return
	}

	// Enrich with workflow names
	wfStore := store.NewWorkflowStore(h.mongoStore)
	result := make([]*instanceResponse, 0, len(list))
	for _, inst := range list {
		wf, _ := wfStore.GetByID(ctx, inst.WorkflowID)
		name := inst.WorkflowID
		if wf != nil {
			name = wf.Name
		}
		result = append(result, toInstanceResponse(inst, name))
	}

	// Sort by createdAt desc
	sort.Slice(result, func(i, j int) bool {
		a, _ := time.Parse(time.RFC3339, result[i].CreatedAt)
		b, _ := time.Parse(time.RFC3339, result[j].CreatedAt)
		return a.After(b)
	})

	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) getInstance(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	ctx := r.Context()
	instStore := store.NewInstanceStore(h.mongoStore)

	inst, err := instStore.GetByID(ctx, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "instance not found")
		return
	}

	wfStore := store.NewWorkflowStore(h.mongoStore)
	wf, _ := wfStore.GetByID(ctx, inst.WorkflowID)
	name := inst.WorkflowID
	if wf != nil {
		name = wf.Name
	}
	writeJSON(w, http.StatusOK, toInstanceResponse(inst, name))
}

func (h *Handler) retryInstance(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	ctx := r.Context()
	instStore := store.NewInstanceStore(h.mongoStore)

	inst, err := instStore.GetByID(ctx, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "instance not found")
		return
	}

	// Reset status and retry
	inst.Status = "pending"
	inst.FailedNodes = []string{}
	inst.UpdatedAt = time.Now()
	instStore.Update(ctx, inst)

	writeJSON(w, http.StatusOK, map[string]string{"status": "retrying"})
}

func (h *Handler) getInstanceTasks(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	ctx := r.Context()
	coll := h.mongoStore.Collection("task_records")

	cursor, err := coll.Find(ctx, map[string]interface{}{"instance_id": id})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list tasks")
		return
	}
	defer cursor.Close(ctx)

	var tasks []types.TaskRecord
	if err := cursor.All(ctx, &tasks); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to decode tasks")
		return
	}
	writeJSON(w, http.StatusOK, tasks)
}

// ─── Worker Handlers ──────────────────────────────────────────────────────────

func (h *Handler) listWorkers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	coll := h.mongoStore.Collection("worker_registry")

	cursor, err := coll.Find(ctx, map[string]interface{}{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list workers")
		return
	}
	defer cursor.Close(ctx)

	var workers []types.WorkerRegistry
	if err := cursor.All(ctx, &workers); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to decode workers")
		return
	}
	writeJSON(w, http.StatusOK, workers)
}

// ─── Stats Handler ────────────────────────────────────────────────────────────

func (h *Handler) getStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Count plugins
	wfStore := store.NewWorkflowStore(h.mongoStore)
	instStore := store.NewInstanceStore(h.mongoStore)
	workerColl := h.mongoStore.Collection("worker_registry")

	plugins := 0
	{
		cursor, _ := workerColl.Find(ctx, map[string]interface{}{})
		var workers []types.WorkerRegistry
		if cursor != nil {
			cursor.All(ctx, &workers)
			cursor.Close(ctx)
		}
		seen := make(map[string]bool)
		for _, worker := range workers {
			for _, p := range worker.Plugins {
				if !seen[p] {
					seen[p] = true
					plugins++
				}
			}
		}
	}

	wfList, _ := wfStore.List(ctx)
	instances, _ := instStore.ListByStatus(ctx, "running")

	// Count today's triggered
	today := time.Now().Truncate(24 * time.Hour)
	running := 0
	for _, i := range instances {
		if !i.CreatedAt.Before(today) {
			running++
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"totalPlugins":     plugins,
		"totalWorkflows":   len(wfList),
		"runningInstances": len(instances),
		"triggeredToday":   running,
	})
}

// ─── File Storage Helpers ────────────────────────────────────────────────────

func (h *Handler) saveWorkflowJSON(wf *types.WorkflowDefinition) error {
	if err := os.MkdirAll(h.workflowFS, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(wf, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(h.workflowFS, wf.ID+".json"), data, 0644)
}

func (h *Handler) loadWorkflowJSON(id string) (*types.WorkflowDefinition, error) {
	filePath := filepath.Join(h.workflowFS, id+".json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	var wf types.WorkflowDefinition
	if err := json.Unmarshal(data, &wf); err != nil {
		return nil, err
	}
	return &wf, nil
}

// ─── Response Helpers ────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// ─── Response Types ───────────────────────────────────────────────────────────

type workflowResponse struct {
	ID         string                  `json:"id"`
	Name       string                  `json:"name"`
	Version    int                     `json:"version"`
	Trigger    types.Trigger           `json:"trigger"`
	Nodes      []types.WorkflowNode    `json:"nodes"`
	OnComplete *types.CallbackAction   `json:"onComplete,omitempty"`
	OnFailure  *types.CallbackAction   `json:"onFailure,omitempty"`
	Published  bool                    `json:"published"`
	CreatedAt  string                  `json:"createdAt"`
	UpdatedAt  string                  `json:"updatedAt"`
}

type pluginResponse struct {
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	Status       string   `json:"status"`
	RegisteredAt string   `json:"registeredAt"`
	NodeTypes    []string `json:"nodeTypes"`
}

type instanceResponse struct {
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

type createWorkflowRequest struct {
	Name       string                `json:"name"`
	Trigger    types.Trigger        `json:"trigger"`
	Nodes      []types.WorkflowNode  `json:"nodes"`
	OnComplete *types.CallbackAction `json:"on_complete"`
	OnFailure  *types.CallbackAction `json:"on_failure"`
}

type updateWorkflowRequest struct {
	Name       *string               `json:"name"`
	Nodes      []types.WorkflowNode  `json:"nodes"`
	Trigger    *types.Trigger        `json:"trigger"`
	OnComplete *types.CallbackAction `json:"on_complete"`
	OnFailure  *types.CallbackAction `json:"on_failure"`
}

func toWorkflowResponse(wf *types.WorkflowDefinition) *workflowResponse {
	return &workflowResponse{
		ID:         wf.ID,
		Name:       wf.Name,
		Version:    wf.Version,
		Trigger:    wf.Trigger,
		Nodes:      wf.Nodes,
		OnComplete: wf.OnComplete,
		OnFailure:  wf.OnFailure,
		Published:  wf.Published,
		CreatedAt:  wf.CreatedAt.Format(time.RFC3339),
		UpdatedAt:  wf.UpdatedAt.Format(time.RFC3339),
	}
}

func toInstanceResponse(inst *types.WorkflowInstance, wfName string) *instanceResponse {
	return &instanceResponse{
		ID:             inst.ID,
		WorkflowID:     inst.WorkflowID,
		WorkflowName:   wfName,
		Status:         inst.Status,
		Event:          inst.Event,
		CurrentNodes:   inst.CurrentNodes,
		CompletedNodes: inst.CompletedNodes,
		FailedNodes:    inst.FailedNodes,
		CreatedAt:      inst.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      inst.UpdatedAt.Format(time.RFC3339),
	}
}
