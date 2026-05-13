package server

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/capyflow/allspark-go/logx"
	"github.com/google/uuid"
	vhttp "github.com/capyflow/vortexv3/server/http"

	"Vortex_Refinery/internal/store"
	"Vortex_Refinery/model"
	"Vortex_Refinery/pkg/types"
)

// ListWorkflows 获取处理流列表
func (s *RefineryServer) ListWorkflows(ctx *vhttp.Context) error {
	gctx := ctx.Context()
	wfStore := store.NewWorkflowStore(s.store)

	// Try MongoDB first
	list, err := wfStore.List(gctx)
	if err == nil && len(list) > 0 {
		result := make([]*model.WorkflowResp, 0, len(list))
		for _, wf := range list {
			result = append(result, toWorkflowResp(wf))
		}
		sortByUpdatedAtDesc(result)
		return ctx.JsonResponse(vhttp.Codes.Success, result)
	}

	// Fall back to JSON files
	files, err := os.ReadDir(s.cfg.Workflows.Dir)
	if err != nil {
		logx.Errorf("RefineryServer|ListWorkflows|ReadDir|Error|%v", err)
		return ctx.JsonResponse(vhttp.Codes.InternalError, nil)
	}

	workflows := make([]*model.WorkflowResp, 0)
	for _, f := range files {
		if f.IsDir() || filepath.Ext(f.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.cfg.Workflows.Dir, f.Name()))
		if err != nil {
			continue
		}
		var wf types.WorkflowDefinition
		if err := json.Unmarshal(data, &wf); err != nil {
			continue
		}
		workflows = append(workflows, toWorkflowResp(&wf))
	}
	sortByUpdatedAtDesc(workflows)
	return ctx.JsonResponse(vhttp.Codes.Success, workflows)
}

// CreateWorkflow 创建处理流
func (s *RefineryServer) CreateWorkflow(ctx *vhttp.Context) error {
	var req model.CreateWorkflowReq
	if err := ctx.UnmarshalBody(&req); err != nil {
		logx.Errorf("RefineryServer|CreateWorkflow|UnmarshalBody|Error|%v", err)
		return ctx.JsonResponse(vhttp.Codes.BadRequest, nil)
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

	// Save to JSON file (primary storage)
	if err := s.saveWorkflowJSON(wf); err != nil {
		logx.Errorf("RefineryServer|CreateWorkflow|saveWorkflowJSON|Error|%v", err)
		return ctx.JsonResponse(vhttp.Codes.InternalError, nil)
	}

	// Also save to MongoDB
	gctx := ctx.Context()
	wfStore := store.NewWorkflowStore(s.store)
	if err := wfStore.Create(gctx, wf); err != nil {
		logx.Errorf("RefineryServer|CreateWorkflow|wfStore.Create|Error|%v", err)
	}

	return ctx.JsonResponse(vhttp.Codes.Success, model.IDResp{ID: wf.ID})
}

// GetWorkflow 获取单个处理流
func (s *RefineryServer) GetWorkflow(ctx *vhttp.Context) error {
	id := ctx.GinContext().Param("id")
	gctx := ctx.Context()

	// Try JSON first
	wf, err := s.loadWorkflowJSON(id)
	if err == nil && wf != nil {
		return ctx.JsonResponse(vhttp.Codes.Success, toWorkflowResp(wf))
	}

	// Fall back to MongoDB
	wfStore := store.NewWorkflowStore(s.store)
	wf, err = wfStore.GetByID(gctx, id)
	if err != nil {
		logx.Errorf("RefineryServer|GetWorkflow|GetByID|Error|%v|id:%s", err, id)
		return ctx.JsonResponse(vhttp.Codes.BadRequest, nil)
	}
	return ctx.JsonResponse(vhttp.Codes.Success, toWorkflowResp(wf))
}

// UpdateWorkflow 更新处理流
func (s *RefineryServer) UpdateWorkflow(ctx *vhttp.Context) error {
	id := ctx.GinContext().Param("id")
	gctx := ctx.Context()

	var req model.UpdateWorkflowReq
	if err := ctx.UnmarshalBody(&req); err != nil {
		logx.Errorf("RefineryServer|UpdateWorkflow|UnmarshalBody|Error|%v", err)
		return ctx.JsonResponse(vhttp.Codes.BadRequest, nil)
	}

	// Load existing
	wf, err := s.loadWorkflowJSON(id)
	if err != nil || wf == nil {
		wfStore := store.NewWorkflowStore(s.store)
		wf, err = wfStore.GetByID(gctx, id)
		if err != nil {
			logx.Errorf("RefineryServer|UpdateWorkflow|GetByID|Error|%v|id:%s", err, id)
			return ctx.JsonResponse(vhttp.Codes.BadRequest, nil)
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
	if err := s.saveWorkflowJSON(wf); err != nil {
		logx.Errorf("RefineryServer|UpdateWorkflow|saveWorkflowJSON|Error|%v", err)
		return ctx.JsonResponse(vhttp.Codes.InternalError, nil)
	}

	return ctx.JsonResponse(vhttp.Codes.Success, toWorkflowResp(wf))
}

// DeleteWorkflow 删除处理流
func (s *RefineryServer) DeleteWorkflow(ctx *vhttp.Context) error {
	id := ctx.GinContext().Param("id")
	gctx := ctx.Context()

	// Delete JSON file
	filePath := filepath.Join(s.cfg.Workflows.Dir, id+".json")
	os.Remove(filePath)

	// Delete from MongoDB
	wfStore := store.NewWorkflowStore(s.store)
	wfStore.Delete(gctx, id)

	return ctx.JsonResponse(vhttp.Codes.Success, model.StatusResp{Status: "deleted"})
}

// PublishWorkflow 发布处理流
func (s *RefineryServer) PublishWorkflow(ctx *vhttp.Context) error {
	id := ctx.GinContext().Param("id")
	gctx := ctx.Context()

	wf, err := s.loadWorkflowJSON(id)
	if err != nil || wf == nil {
		wfStore := store.NewWorkflowStore(s.store)
		wf, err = wfStore.GetByID(gctx, id)
		if err != nil {
			logx.Errorf("RefineryServer|PublishWorkflow|GetByID|Error|%v|id:%s", err, id)
			return ctx.JsonResponse(vhttp.Codes.BadRequest, nil)
		}
	}

	wf.Published = true
	wf.UpdatedAt = time.Now()

	if err := s.saveWorkflowJSON(wf); err != nil {
		logx.Errorf("RefineryServer|PublishWorkflow|saveWorkflowJSON|Error|%v", err)
		return ctx.JsonResponse(vhttp.Codes.InternalError, nil)
	}

	wfStore := store.NewWorkflowStore(s.store)
	wfStore.Update(gctx, wf)

	return ctx.JsonResponse(vhttp.Codes.Success, toWorkflowResp(wf))
}

// ─── Helpers ───────────────────────────────────────────────────────────────────

func (s *RefineryServer) saveWorkflowJSON(wf *types.WorkflowDefinition) error {
	if err := os.MkdirAll(s.cfg.Workflows.Dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(wf, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(s.cfg.Workflows.Dir, wf.ID+".json"), data, 0644)
}

func (s *RefineryServer) loadWorkflowJSON(id string) (*types.WorkflowDefinition, error) {
	filePath := filepath.Join(s.cfg.Workflows.Dir, id+".json")
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

func toWorkflowResp(wf *types.WorkflowDefinition) *model.WorkflowResp {
	return &model.WorkflowResp{
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

func sortByUpdatedAtDesc(list []*model.WorkflowResp) {
	sort.Slice(list, func(i, j int) bool {
		a, _ := time.Parse(time.RFC3339, list[i].UpdatedAt)
		b, _ := time.Parse(time.RFC3339, list[j].UpdatedAt)
		return a.After(b)
	})
}
