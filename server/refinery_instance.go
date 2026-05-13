package server

import (
	"sort"
	"time"

	"github.com/capyflow/allspark-go/logx"
	vhttp "github.com/capyflow/vortexv3/server/http"

	"Vortex_Refinery/internal/store"
	"Vortex_Refinery/model"
	"Vortex_Refinery/pkg/types"
)

// ListInstances 获取处理流实例列表
func (s *RefineryServer) ListInstances(ctx *vhttp.Context) error {
	gctx := ctx.Context()
	status := ctx.GinContext().Query("status")
	instStore := store.NewInstanceStore(s.store)

	list, err := instStore.ListByStatus(gctx, status)
	if err != nil {
		logx.Errorf("RefineryServer|ListInstances|ListByStatus|Error|%v", err)
		return ctx.JsonResponse(vhttp.Codes.InternalError, nil)
	}

	// Enrich with workflow names
	wfStore := store.NewWorkflowStore(s.store)
	result := make([]*model.InstanceResp, 0, len(list))
	for _, inst := range list {
		wf, _ := wfStore.GetByID(gctx, inst.WorkflowID)
		name := inst.WorkflowID
		if wf != nil {
			name = wf.Name
		}
		result = append(result, toInstanceResp(inst, name))
	}

	// Sort by createdAt desc
	sort.Slice(result, func(i, j int) bool {
		a, _ := time.Parse(time.RFC3339, result[i].CreatedAt)
		b, _ := time.Parse(time.RFC3339, result[j].CreatedAt)
		return a.After(b)
	})

	return ctx.JsonResponse(vhttp.Codes.Success, result)
}

// GetInstance 获取单个实例详情
func (s *RefineryServer) GetInstance(ctx *vhttp.Context) error {
	id := ctx.GinContext().Param("id")
	gctx := ctx.Context()
	instStore := store.NewInstanceStore(s.store)

	inst, err := instStore.GetByID(gctx, id)
	if err != nil {
		logx.Errorf("RefineryServer|GetInstance|GetByID|Error|%v|id:%s", err, id)
		return ctx.JsonResponse(vhttp.Codes.BadRequest, nil)
	}

	wfStore := store.NewWorkflowStore(s.store)
	wf, _ := wfStore.GetByID(gctx, inst.WorkflowID)
	name := inst.WorkflowID
	if wf != nil {
		name = wf.Name
	}
	return ctx.JsonResponse(vhttp.Codes.Success, toInstanceResp(inst, name))
}

// RetryInstance 重试失败的实例
func (s *RefineryServer) RetryInstance(ctx *vhttp.Context) error {
	id := ctx.GinContext().Param("id")
	gctx := ctx.Context()
	instStore := store.NewInstanceStore(s.store)

	inst, err := instStore.GetByID(gctx, id)
	if err != nil {
		logx.Errorf("RefineryServer|RetryInstance|GetByID|Error|%v|id:%s", err, id)
		return ctx.JsonResponse(vhttp.Codes.BadRequest, nil)
	}

	// Reset status and retry
	inst.Status = "pending"
	inst.FailedNodes = []string{}
	inst.UpdatedAt = time.Now()
	instStore.Update(gctx, inst)

	return ctx.JsonResponse(vhttp.Codes.Success, model.StatusResp{Status: "retrying"})
}

// GetInstanceTasks 获取实例的所有任务记录
func (s *RefineryServer) GetInstanceTasks(ctx *vhttp.Context) error {
	id := ctx.GinContext().Param("id")
	gctx := ctx.Context()
	coll := s.store.Collection("task_records")

	cursor, err := coll.Find(gctx, map[string]interface{}{"instance_id": id})
	if err != nil {
		logx.Errorf("RefineryServer|GetInstanceTasks|Find|Error|%v", err)
		return ctx.JsonResponse(vhttp.Codes.InternalError, nil)
	}
	defer cursor.Close(gctx)

	var tasks []types.TaskRecord
	if err := cursor.All(gctx, &tasks); err != nil {
		logx.Errorf("RefineryServer|GetInstanceTasks|All|Error|%v", err)
		return ctx.JsonResponse(vhttp.Codes.InternalError, nil)
	}
	return ctx.JsonResponse(vhttp.Codes.Success, tasks)
}

// ─── Helpers ───────────────────────────────────────────────────────────────────

func toInstanceResp(inst *types.WorkflowInstance, wfName string) *model.InstanceResp {
	return &model.InstanceResp{
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
